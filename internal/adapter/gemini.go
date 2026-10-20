package adapter

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"Xin-api/pkg/jsonx"
)

// geminiAdapter Google Gemini(generateContent) 协议适配器
// 流式转换需跨事件累积函数调用状态，故由 Registry 每请求实例化一份
type geminiAdapter struct {
	chunkID     string         // 流式响应 id（同一流内保持一致）
	toolIndex   map[int]int    // candidate parts 序号 → OpenAI tool_calls index
	toolArgs    map[int]string // 已下发的 args 快照，用于增量补差
	toolCount   int            // 已下发的 tool_calls 数量
	hasToolCall bool           // 本流是否出现过函数调用（决定 finish_reason）
}

// NewGeminiAdapter 创建 Google Gemini 协议适配器
func NewGeminiAdapter() ProviderAdapter {
	return &geminiAdapter{}
}

func (a *geminiAdapter) Name() string {
	return "gemini"
}

// ExtractModel 从入站 OpenAI Chat 请求体中提取顶层 model 字段
func (a *geminiAdapter) ExtractModel(body []byte) (string, error) {
	var req struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return "", fmt.Errorf("gemini: parse request body failed: %w", err)
	}
	if req.Model == "" {
		return "", fmt.Errorf("gemini: model field is required")
	}
	return req.Model, nil
}

// BuildUpstreamRequest 将入站 OpenAI Chat 请求转换为 Gemini generateContent 请求
func (a *geminiAdapter) BuildUpstreamRequest(body []byte, upstreamModel, apiKey string) (*UpstreamRequest, error) {
	var req struct {
		Model       string          `json:"model"`
		Stream      bool            `json:"stream"`
		MaxTokens   *int            `json:"max_tokens"`
		Temperature *float64        `json:"temperature"`
		TopP        *float64        `json:"top_p"`
		Tools       json.RawMessage `json:"tools"`
		ToolChoice  json.RawMessage `json:"tool_choice"`
		Messages    []geminiMessage `json:"messages"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("gemini: parse request body failed: %w", err)
	}

	// 模型名：优先使用渠道登记的上游模型名
	model := req.Model
	if upstreamModel != "" {
		model = upstreamModel
	}
	if model == "" {
		return nil, fmt.Errorf("gemini: model field is required")
	}

	// messages → contents / systemInstruction（user/assistant 支持图片 part）
	var contents []map[string]any
	var systemTexts []string
	// Gemini 的 functionResponse 只带函数名（无调用 id），需回查此前 assistant tool_calls 的 id → name
	callNames := make(map[string]string)
	// Gemini 要求函数响应紧跟在发起调用的 model 消息之后，故合并连续的 tool 消息为一条 user 消息
	var functionResponses []any
	flushFunctionResponses := func() {
		if len(functionResponses) == 0 {
			return
		}
		contents = append(contents, map[string]any{"role": "user", "parts": functionResponses})
		functionResponses = nil
	}

	for _, msg := range req.Messages {
		if msg.Role != "tool" {
			flushFunctionResponses()
		}
		switch msg.Role {
		case "system":
			if text := chatContentText(msg.Content); text != "" {
				systemTexts = append(systemTexts, text)
			}
		case "assistant":
			recordGeminiToolCallNames(msg.ToolCalls, callNames)
			contents = append(contents, map[string]any{
				"role":  "model",
				"parts": geminiAssistantParts(msg.Content, msg.ToolCalls),
			})
		case "tool":
			// 函数名解析顺序对齐 new-api：优先取 OpenAI tool 消息自带的 name，
			// 其次按 tool_call_id 回查此前 assistant tool_calls 记录的函数名
			name := msg.Name
			if name == "" {
				name = callNames[msg.ToolCallID]
			}
			functionResponses = append(functionResponses, map[string]any{
				"functionResponse": map[string]any{
					"name":     name,
					"response": geminiFunctionResponseBody(chatContentText(msg.Content)),
				},
			})
		default: // user 及其它角色统一按用户消息处理
			contents = append(contents, map[string]any{
				"role":  "user",
				"parts": geminiParts(msg.Content),
			})
		}
	}
	flushFunctionResponses()

	out := map[string]any{
		"contents": contents,
	}
	if len(systemTexts) > 0 {
		out["systemInstruction"] = map[string]any{
			"parts": []map[string]any{{"text": strings.Join(systemTexts, "\n")}},
		}
	}
	// tools / toolConfig：函数声明与调用模式（toolConfig 仅在存在 tools 时下发）
	if declarations := geminiUpstreamTools(req.Tools); len(declarations) > 0 {
		out["tools"] = []map[string]any{{"functionDeclarations": declarations}}
		if config := geminiToolConfig(req.ToolChoice); config != nil {
			out["toolConfig"] = config
		}
	}

	// generationConfig：仅在入参存在时写入对应字段
	genConfig := map[string]any{}
	if req.MaxTokens != nil {
		genConfig["maxOutputTokens"] = *req.MaxTokens
	}
	if req.Temperature != nil {
		genConfig["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		genConfig["topP"] = *req.TopP
	}
	if len(genConfig) > 0 {
		out["generationConfig"] = genConfig
	}

	newBody, err := jsonx.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal upstream body failed: %w", err)
	}

	endpoint := "/v1beta/models/" + model + ":generateContent"
	if req.Stream {
		endpoint += "?alt=sse"
	}
	return &UpstreamRequest{
		Method:   http.MethodPost,
		Endpoint: endpoint,
		Body:     newBody,
		Headers:  map[string]string{"x-goog-api-key": apiKey},
	}, nil
}

// ConvertResponse 将 Gemini 非流式响应转换为 OpenAI Chat Completion
func (a *geminiAdapter) ConvertResponse(body []byte, externalModel string) ([]byte, error) {
	var resp geminiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, nil
	}

	content := ""
	finishReason := ""
	var toolCalls []map[string]any
	if len(resp.Candidates) > 0 {
		candidate := resp.Candidates[0]
		content = candidate.Text()
		toolCalls = geminiToolCalls(candidate.Content.Parts)
		if len(toolCalls) > 0 {
			// 出现函数调用时 OpenAI 语义为 tool_calls
			finishReason = "tool_calls"
		} else {
			finishReason = mapGeminiFinishReason(candidate.FinishReason)
		}
	}

	message := map[string]any{"role": "assistant", "content": content}
	if len(toolCalls) > 0 {
		message["tool_calls"] = toolCalls
	}

	out := map[string]any{
		"id":      newChatCompletionID(),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   externalModel,
		"choices": []map[string]any{
			{
				"index":         0,
				"message":       message,
				"finish_reason": finishReason,
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     resp.UsageMetadata.PromptTokenCount,
			"completion_tokens": resp.UsageMetadata.CandidatesTokenCount,
			"total_tokens":      resp.UsageMetadata.TotalTokenCount,
		},
	}

	return jsonx.Marshal(out)
}

// ConvertStreamChunk 将 Gemini alt=sse 的单个事件块转换为 OpenAI SSE 事件块
func (a *geminiAdapter) ConvertStreamChunk(chunk []byte, externalModel string) ([]byte, error) {
	payload := extractSSEData(chunk)
	if len(payload) == 0 {
		return nil, nil
	}

	var resp geminiResponse
	if err := json.Unmarshal(payload, &resp); err != nil {
		return nil, nil
	}
	if len(resp.Candidates) == 0 {
		return nil, nil
	}

	candidate := resp.Candidates[0]
	if a.chunkID == "" {
		a.chunkID = newChatCompletionID()
	}

	delta := map[string]any{"content": candidate.Text()}
	if calls := a.streamToolCalls(candidate.Content.Parts); len(calls) > 0 {
		delta["tool_calls"] = calls
		a.hasToolCall = true
	}

	// finish_reason 缺省时为 null；出现过函数调用时按 OpenAI 语义返回 tool_calls
	var finishReason any
	if candidate.FinishReason != "" {
		if a.hasToolCall {
			finishReason = "tool_calls"
		} else {
			finishReason = mapGeminiFinishReason(candidate.FinishReason)
		}
	}

	out := map[string]any{
		"id":      a.chunkID,
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   externalModel,
		"choices": []map[string]any{
			{
				"index":         0,
				"delta":         delta,
				"finish_reason": finishReason,
			},
		},
	}

	data, err := jsonx.Marshal(out)
	if err != nil {
		return nil, nil
	}
	result := openaiSSE(data)
	// 含 finishReason 时追加结束标记
	if candidate.FinishReason != "" {
		result = append(result, "data: [DONE]\n\n"...)
	}
	return result, nil
}

// UpstreamModelsRequest 构造 GET {base}/v1beta/models 请求
func (a *geminiAdapter) UpstreamModelsRequest(apiKey string) *UpstreamRequest {
	return &UpstreamRequest{
		Method:   http.MethodGet,
		Endpoint: "/v1beta/models",
		Body:     nil,
		Headers:  map[string]string{"x-goog-api-key": apiKey},
	}
}

// geminiMessage 入站 OpenAI 消息，content 保留原始 JSON 以兼容字符串与数组两种形态
type geminiMessage struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content"`
	ToolCalls  json.RawMessage `json:"tool_calls"`
	ToolCallID string          `json:"tool_call_id"`
	Name       string          `json:"name"`
}

// geminiPart Gemini 内容 part：文本 / 函数调用 / 函数响应
type geminiPart struct {
	Text             string                  `json:"text"`
	FunctionCall     *geminiFunctionCall     `json:"functionCall"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse"`
}

// geminiFunctionCall 模型发起的函数调用，args 为参数对象
type geminiFunctionCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

// geminiFunctionResponse 回传给模型的函数执行结果，response 必须为对象
type geminiFunctionResponse struct {
	Name     string          `json:"name"`
	Response json.RawMessage `json:"response"`
}

// geminiCandidate Gemini 回复候选，可能包含多个文本/函数调用部分
type geminiCandidate struct {
	Content struct {
		Parts []geminiPart `json:"parts"`
	} `json:"content"`
	FinishReason string `json:"finishReason"`
}

// Text 拼接候选内容中所有 text 部分
func (c *geminiCandidate) Text() string {
	var b strings.Builder
	for _, p := range c.Content.Parts {
		b.WriteString(p.Text)
	}
	return b.String()
}

// geminiResponse Gemini generateContent 响应（流式事件块复用同一结构）
type geminiResponse struct {
	Candidates    []geminiCandidate `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
		TotalTokenCount      int `json:"totalTokenCount"`
	} `json:"usageMetadata"`
	ModelVersion string `json:"modelVersion"`
}

// geminiParts 将 OpenAI message.content 转为 Gemini parts：
// 文本 → text，data URI 图片 → inlineData，http(s) 图片 → fileData
func geminiParts(raw json.RawMessage) []map[string]any {
	parts := make([]map[string]any, 0, 1)
	for _, p := range parseChatContentParts(raw) {
		switch p.Type {
		case "text":
			parts = append(parts, map[string]any{"text": p.Text})
		case "image_url":
			if mimeType, data, ok := splitDataURI(p.ImageURL); ok {
				parts = append(parts, map[string]any{
					"inlineData": map[string]any{"mimeType": mimeType, "data": data},
				})
			} else if p.ImageURL != "" {
				parts = append(parts, map[string]any{
					"fileData": map[string]any{"fileUri": p.ImageURL},
				})
			}
		}
	}
	if len(parts) == 0 {
		// Gemini 要求 parts 非空，空内容补一个空文本 part
		parts = append(parts, map[string]any{"text": ""})
	}
	return parts
}

// recordGeminiToolCallNames 记录 assistant tool_calls 的 id → 函数名映射
// Gemini 的 functionResponse 只接受函数名，需要据此把 OpenAI 的 tool_call_id 还原为名称
func recordGeminiToolCallNames(raw json.RawMessage, callNames map[string]string) {
	var calls []struct {
		ID       string `json:"id"`
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &calls) != nil {
		return
	}
	for _, c := range calls {
		if c.ID != "" && c.Function.Name != "" {
			callNames[c.ID] = c.Function.Name
		}
	}
}

// geminiAssistantParts 将 assistant 消息转为 Gemini parts：文本/图片 part + functionCall part
func geminiAssistantParts(content, toolCalls json.RawMessage) []map[string]any {
	parts := geminiParts(content)
	// 纯工具调用场景去掉占位空文本 part
	if len(parts) == 1 {
		if text, ok := parts[0]["text"].(string); ok && text == "" {
			parts = nil
		}
	}

	var calls []struct {
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	}
	if len(toolCalls) > 0 && json.Unmarshal(toolCalls, &calls) == nil {
		for _, c := range calls {
			if c.Function.Name == "" {
				continue
			}
			args, _ := toolObject([]byte(c.Function.Arguments))
			if args == nil {
				args = map[string]any{}
			}
			parts = append(parts, map[string]any{
				"functionCall": map[string]any{"name": c.Function.Name, "args": args},
			})
		}
	}
	if len(parts) == 0 {
		// Gemini 要求 parts 非空
		parts = []map[string]any{{"text": ""}}
	}
	return parts
}

// geminiFunctionResponseBody 构造 functionResponse.response（必须为对象）
// 工具输出为 JSON 对象时原样透传，否则包装为 {"result": ...}
func geminiFunctionResponseBody(content string) any {
	if obj, ok := toolObject([]byte(content)); ok {
		return obj
	}
	return map[string]any{"result": content}
}

// geminiUpstreamTools 将 OpenAI tools 转换为 Gemini functionDeclarations
func geminiUpstreamTools(raw json.RawMessage) []map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var tools []struct {
		Type     string `json:"type"`
		Function struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			Parameters  json.RawMessage `json:"parameters"`
		} `json:"function"`
	}
	if json.Unmarshal(raw, &tools) != nil {
		return nil
	}
	declarations := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		if t.Type != "function" || t.Function.Name == "" {
			continue
		}
		declaration := map[string]any{"name": t.Function.Name}
		if t.Function.Description != "" {
			declaration["description"] = t.Function.Description
		}
		params, _ := toolObject(t.Function.Parameters)
		if params == nil {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		declaration["parameters"] = params
		declarations = append(declarations, declaration)
	}
	if len(declarations) == 0 {
		return nil
	}
	return declarations
}

// geminiToolConfig 将 OpenAI tool_choice 映射为 Gemini toolConfig.functionCallingConfig
func geminiToolConfig(raw json.RawMessage) map[string]any {
	trimmed := trimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil
	}
	mode := ""
	var allowed []string
	if trimmed[0] == '"' {
		var kind string
		if json.Unmarshal(trimmed, &kind) != nil {
			return nil
		}
		mode = geminiFunctionCallingMode(kind)
	} else {
		var obj struct {
			Type     string `json:"type"`
			Function struct {
				Name string `json:"name"`
			} `json:"function"`
		}
		if json.Unmarshal(trimmed, &obj) != nil {
			return nil
		}
		if obj.Type == "function" {
			if obj.Function.Name == "" {
				return nil
			}
			mode = "ANY"
			allowed = []string{obj.Function.Name}
		} else {
			mode = geminiFunctionCallingMode(obj.Type)
		}
	}
	if mode == "" {
		return nil
	}
	config := map[string]any{"mode": mode}
	if len(allowed) > 0 {
		config["allowedFunctionNames"] = allowed
	}
	return map[string]any{"functionCallingConfig": config}
}

// geminiFunctionCallingMode OpenAI tool_choice 类型名 → Gemini functionCallingConfig.mode
func geminiFunctionCallingMode(kind string) string {
	switch kind {
	case "auto":
		return "AUTO"
	case "required", "any":
		return "ANY"
	case "none":
		return "NONE"
	default:
		return ""
	}
}

// geminiToolCalls 将 Gemini 响应中的 functionCall part 转为 OpenAI tool_calls
// Gemini 不返回调用 id，此处生成并在后续 tool 消息中按 id 回查函数名
func geminiToolCalls(parts []geminiPart) []map[string]any {
	calls := make([]map[string]any, 0, len(parts))
	for _, p := range parts {
		if p.FunctionCall == nil || p.FunctionCall.Name == "" {
			continue
		}
		calls = append(calls, map[string]any{
			"id":   toolUseID(""),
			"type": "function",
			"function": map[string]any{
				"name":      p.FunctionCall.Name,
				"arguments": compactToolArguments(p.FunctionCall.Args),
			},
		})
	}
	return calls
}

// streamToolCalls 将流式块中的 functionCall part 转为 OpenAI tool_calls 增量
// 以 part 序号为键：首次出现下发 id/name/arguments，后续出现按已下发前缀补差
func (a *geminiAdapter) streamToolCalls(parts []geminiPart) []map[string]any {
	var out []map[string]any
	for i, p := range parts {
		if p.FunctionCall == nil || p.FunctionCall.Name == "" {
			continue
		}
		if a.toolIndex == nil {
			a.toolIndex = make(map[int]int)
			a.toolArgs = make(map[int]string)
		}
		index, seen := a.toolIndex[i]
		if !seen {
			index = a.toolCount
			a.toolCount++
			a.toolIndex[i] = index
			a.toolArgs[index] = string(p.FunctionCall.Args)
			out = append(out, map[string]any{
				"index": index,
				"id":    toolUseID(""),
				"type":  "function",
				"function": map[string]any{
					"name":      p.FunctionCall.Name,
					"arguments": compactToolArguments(p.FunctionCall.Args),
				},
			})
			continue
		}
		// 同一 part 的后续增量：Gemini 可能下发增量片段，也可能重发累计参数，统一按前缀补差
		current := string(p.FunctionCall.Args)
		previous := a.toolArgs[index]
		if current == "" || current == previous {
			continue
		}
		if strings.HasPrefix(current, previous) {
			current = current[len(previous):]
		}
		a.toolArgs[index] = string(p.FunctionCall.Args)
		out = append(out, map[string]any{
			"index":    index,
			"function": map[string]any{"arguments": current},
		})
	}
	return out
}

// compactToolArguments 压缩函数参数 JSON；参数为增量片段等非法 JSON 时原样返回
func compactToolArguments(raw []byte) string {
	trimmed := trimSpace(raw)
	if len(trimmed) == 0 {
		return ""
	}
	var buf bytes.Buffer
	if json.Compact(&buf, trimmed) != nil {
		return string(trimmed)
	}
	return buf.String()
}

// mapGeminiFinishReason 将 Gemini finishReason 映射为 OpenAI finish_reason
func mapGeminiFinishReason(reason string) string {
	switch reason {
	case "STOP":
		return "stop"
	case "MAX_TOKENS":
		return "length"
	case "SAFETY", "RECITATION":
		return "content_filter"
	default:
		return "stop"
	}
}

// newChatCompletionID 生成 OpenAI 风格的随机完成 ID
func newChatCompletionID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
	}
	return "chatcmpl-" + hex.EncodeToString(b)
}
