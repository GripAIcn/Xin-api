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

// anthropicAdapter Anthropic(Claude) 协议适配器：把入站 OpenAI Chat 请求
// 转换为 Anthropic Messages 请求，并把 Anthropic 响应/流事件转回 OpenAI 形态
// 流式转换需跨事件累积工具调用的块序号映射，故由 Registry 每请求实例化一份
type anthropicAdapter struct {
	chunkID   string      // 流式响应 id（取自 message_start）
	toolIndex map[int]int // Anthropic content_block 序号 → OpenAI tool_calls index
	toolCount int         // 已下发的 tool_calls 数量
}

// NewAnthropicAdapter 创建 Anthropic 适配器
func NewAnthropicAdapter() ProviderAdapter {
	return &anthropicAdapter{}
}

func (a *anthropicAdapter) Name() string {
	return "anthropic"
}

// ExtractModel 从入站 OpenAI Chat 请求体中提取顶层 model 字段
func (a *anthropicAdapter) ExtractModel(body []byte) (string, error) {
	var req struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return "", fmt.Errorf("anthropic: parse request body failed: %w", err)
	}
	if req.Model == "" {
		return "", fmt.Errorf("anthropic: model field is required")
	}
	return req.Model, nil
}

// anthropicMessage Anthropic Messages 请求中的单条消息
// content 为字符串（纯文本）或 Anthropic content block 数组（含图片）
type anthropicMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// BuildUpstreamRequest 将 OpenAI Chat 请求体转换为 Anthropic Messages 请求体
func (a *anthropicAdapter) BuildUpstreamRequest(body []byte, upstreamModel, apiKey string) (*UpstreamRequest, error) {
	var in struct {
		Model             string            `json:"model"`
		Messages          []json.RawMessage `json:"messages"`
		MaxTokens         *int              `json:"max_tokens"`
		Stream            bool              `json:"stream"`
		Temperature       *float64          `json:"temperature"`
		TopP              *float64          `json:"top_p"`
		Tools             json.RawMessage   `json:"tools"`
		ToolChoice        json.RawMessage   `json:"tool_choice"`
		ParallelToolCalls *bool             `json:"parallel_tool_calls"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return nil, fmt.Errorf("anthropic: parse request body failed: %w", err)
	}

	model := upstreamModel
	if model == "" {
		model = in.Model
	}

	// system 消息合并到顶层 system（仅文本），user/assistant 保留进 messages（支持图片）
	var systemParts []string
	messages := make([]anthropicMessage, 0, len(in.Messages))
	// Anthropic 要求 tool_result 位于 user 消息且角色交替，故合并连续的 tool 消息为一条 user 消息
	var toolResults []any
	flushToolResults := func() {
		if len(toolResults) == 0 {
			return
		}
		messages = append(messages, anthropicMessage{Role: "user", Content: toolResults})
		toolResults = nil
	}

	for _, rawMsg := range in.Messages {
		var m struct {
			Role       string          `json:"role"`
			Content    json.RawMessage `json:"content"`
			ToolCalls  json.RawMessage `json:"tool_calls"`
			ToolCallID string          `json:"tool_call_id"`
		}
		if err := json.Unmarshal(rawMsg, &m); err != nil {
			continue
		}
		if m.Role != "tool" {
			flushToolResults()
		}
		switch m.Role {
		case "system":
			if text := chatContentText(m.Content); text != "" {
				systemParts = append(systemParts, text)
			}
		case "assistant":
			messages = append(messages, anthropicMessage{
				Role:    "assistant",
				Content: anthropicAssistantContent(m.Content, m.ToolCalls),
			})
		case "tool":
			toolResults = append(toolResults, map[string]any{
				"type":        "tool_result",
				"tool_use_id": m.ToolCallID,
				"content":     chatContentText(m.Content),
			})
		default:
			// user 及未知角色按 user 处理
			messages = append(messages, anthropicMessage{Role: "user", Content: anthropicChatContent(m.Content)})
		}
	}
	flushToolResults()

	// Anthropic 的 max_tokens 为必填，缺失时默认 4096
	maxTokens := 4096
	if in.MaxTokens != nil {
		maxTokens = *in.MaxTokens
	}

	out := map[string]any{
		"model":      model,
		"messages":   messages,
		"max_tokens": maxTokens,
		"stream":     in.Stream,
	}
	if len(systemParts) > 0 {
		out["system"] = strings.Join(systemParts, "\n\n")
	}
	if in.Temperature != nil {
		out["temperature"] = *in.Temperature
	}
	if in.TopP != nil {
		out["top_p"] = *in.TopP
	}
	// tool_choice 仅在存在 tools 时下发（Anthropic 要求二者配套）
	if tools := anthropicUpstreamTools(in.Tools); len(tools) > 0 {
		out["tools"] = tools
		if choice := anthropicUpstreamToolChoice(in.ToolChoice, in.ParallelToolCalls); choice != nil {
			out["tool_choice"] = choice
		}
	}

	newBody, err := jsonx.Marshal(out)
	if err != nil {
		return nil, err
	}

	return &UpstreamRequest{
		Method:   http.MethodPost,
		Endpoint: "/v1/messages",
		Body:     newBody,
		Headers: map[string]string{
			"x-api-key":         apiKey,
			"anthropic-version": "2023-06-01",
		},
	}, nil
}

// ConvertResponse 将 Anthropic 非流式响应转换为 OpenAI Chat Completion 响应
func (a *anthropicAdapter) ConvertResponse(body []byte, externalModel string) ([]byte, error) {
	var resp struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
		Usage      struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		// 解析失败时尽力降级：原样返回
		return body, nil
	}

	model := externalModel
	if model == "" {
		model = resp.Model
	}

	// 文本块拼接为 content，tool_use 块转为 tool_calls
	var sb strings.Builder
	toolCalls := make([]map[string]any, 0, len(resp.Content))
	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			sb.WriteString(block.Text)
		case "tool_use":
			if block.Name == "" {
				continue
			}
			toolCalls = append(toolCalls, map[string]any{
				"id":   toolUseID(block.ID),
				"type": "function",
				"function": map[string]any{
					"name":      block.Name,
					"arguments": string(compactJSON(block.Input)),
				},
			})
		}
	}

	message := map[string]any{
		"role":    "assistant",
		"content": sb.String(),
	}
	if len(toolCalls) > 0 {
		message["tool_calls"] = toolCalls
	}

	out := map[string]any{
		"id":      resp.ID,
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []map[string]any{
			{
				"index":         0,
				"message":       message,
				"finish_reason": anthropicStopReasonToOpenAI(resp.StopReason),
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     resp.Usage.InputTokens,
			"completion_tokens": resp.Usage.OutputTokens,
			"total_tokens":      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}

	newBody, err := jsonx.Marshal(out)
	if err != nil {
		return body, nil
	}
	return newBody, nil
}

// anthropicDefaultChunkID 无 message.id 可用时流的兜底 id
const anthropicDefaultChunkID = "chatcmpl-anthropic"

// ConvertStreamChunk 将单个 Anthropic SSE 事件块转换为 OpenAI SSE 事件块
// 工具调用：content_block_start(tool_use) 开启 tool_calls 增量，input_json_delta 追加参数
func (a *anthropicAdapter) ConvertStreamChunk(chunk []byte, externalModel string) ([]byte, error) {
	data := extractSSEData(chunk)
	if len(data) == 0 {
		return nil, nil
	}

	var ev struct {
		Type    string `json:"type"`
		Index   int    `json:"index"`
		Message struct {
			ID string `json:"id"`
		} `json:"message"`
		ContentBlock struct {
			Type string `json:"type"`
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"content_block"`
		Delta struct {
			Type        string `json:"type"`
			Text        string `json:"text"`
			PartialJSON string `json:"partial_json"`
			StopReason  string `json:"stop_reason"`
		} `json:"delta"`
	}
	if err := json.Unmarshal(data, &ev); err != nil {
		return nil, nil
	}

	// 流 id 取自 message_start，后续事件复用（无则使用兜底 id）
	if ev.Type == "message_start" && ev.Message.ID != "" {
		a.chunkID = ev.Message.ID
	}
	id := a.chunkID
	if id == "" {
		id = anthropicDefaultChunkID
	}

	buildChunk := func(delta map[string]any, finishReason *string) []byte {
		out := map[string]any{
			"id":      id,
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
		payload, err := jsonx.Marshal(out)
		if err != nil {
			return nil
		}
		return openaiSSE(payload)
	}

	switch ev.Type {
	case "message_start":
		// 首个 chunk：仅带 role
		return buildChunk(map[string]any{"role": "assistant", "content": ""}, nil), nil
	case "content_block_start":
		// 仅 tool_use 块需要下发 tool_calls 起始增量（text 块由 text_delta 承载）
		if ev.ContentBlock.Type != "tool_use" {
			return nil, nil
		}
		if a.toolIndex == nil {
			a.toolIndex = make(map[int]int)
		}
		index := a.toolCount
		a.toolCount++
		a.toolIndex[ev.Index] = index
		return buildChunk(map[string]any{
			"tool_calls": []map[string]any{
				{
					"index": index,
					"id":    toolUseID(ev.ContentBlock.ID),
					"type":  "function",
					"function": map[string]any{
						"name":      ev.ContentBlock.Name,
						"arguments": "",
					},
				},
			},
		}, nil), nil
	case "content_block_delta":
		switch ev.Delta.Type {
		case "text_delta":
			return buildChunk(map[string]any{"content": ev.Delta.Text}, nil), nil
		case "input_json_delta":
			if ev.Delta.PartialJSON == "" {
				return nil, nil
			}
			return buildChunk(map[string]any{
				"tool_calls": []map[string]any{
					{
						"index":    a.toolIndex[ev.Index],
						"function": map[string]any{"arguments": ev.Delta.PartialJSON},
					},
				},
			}, nil), nil
		default:
			return nil, nil
		}
	case "message_delta":
		reason := anthropicStopReasonToOpenAI(ev.Delta.StopReason)
		return buildChunk(map[string]any{}, &reason), nil
	case "message_stop":
		return openaiSSE([]byte("[DONE]")), nil
	default:
		// ping / content_block_stop / message_start 以外的其它事件无需下发
		return nil, nil
	}
}

// UpstreamModelsRequest 构造 GET {base}/v1/models 请求（Anthropic 鉴权头）
func (a *anthropicAdapter) UpstreamModelsRequest(apiKey string) *UpstreamRequest {
	return &UpstreamRequest{
		Method:   http.MethodGet,
		Endpoint: "/v1/models",
		Body:     nil,
		Headers: map[string]string{
			"x-api-key":         apiKey,
			"anthropic-version": "2023-06-01",
		},
	}
}

// anthropicChatContent 将 OpenAI message.content 转换为 Anthropic content：
// 纯文本时返回字符串，含图片时返回 content block 数组
func anthropicChatContent(raw json.RawMessage) any {
	parts := parseChatContentParts(raw)
	if len(parts) == 0 {
		return ""
	}

	hasImage := false
	for _, p := range parts {
		if p.Type == "image_url" {
			hasImage = true
			break
		}
	}
	if !hasImage {
		var sb strings.Builder
		for _, p := range parts {
			sb.WriteString(p.Text)
		}
		return sb.String()
	}

	blocks := make([]map[string]any, 0, len(parts))
	for _, p := range parts {
		switch p.Type {
		case "text":
			if p.Text != "" {
				blocks = append(blocks, map[string]any{"type": "text", "text": p.Text})
			}
		case "image_url":
			if source := anthropicImageSource(p.ImageURL); source != nil {
				blocks = append(blocks, map[string]any{"type": "image", "source": source})
			}
		}
	}
	return blocks
}

// anthropicImageSource 将 OpenAI image_url 转为 Anthropic image source：
// data URI → base64，http(s) 链接 → url
func anthropicImageSource(url string) map[string]any {
	if mediaType, data, ok := splitDataURI(url); ok {
		return map[string]any{
			"type":       "base64",
			"media_type": mediaType,
			"data":       data,
		}
	}
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return map[string]any{"type": "url", "url": url}
	}
	return nil
}

// anthropicAssistantContent 将 OpenAI assistant 消息的 content 与 tool_calls
// 转换为 Anthropic content：文本/图片块 + tool_use 块
func anthropicAssistantContent(content, toolCalls json.RawMessage) any {
	blocks := make([]any, 0, 4)
	switch c := anthropicChatContent(content).(type) {
	case string:
		if c != "" {
			blocks = append(blocks, map[string]any{"type": "text", "text": c})
		}
	case []map[string]any:
		for _, block := range c {
			blocks = append(blocks, block)
		}
	}

	var calls []struct {
		ID       string `json:"id"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	}
	if len(toolCalls) > 0 && json.Unmarshal(toolCalls, &calls) == nil {
		for _, tc := range calls {
			if tc.Function.Name == "" {
				continue
			}
			blocks = append(blocks, map[string]any{
				"type":  "tool_use",
				"id":    toolUseID(tc.ID),
				"name":  tc.Function.Name,
				"input": toolArgumentsObject(tc.Function.Arguments),
			})
		}
	}
	if len(blocks) == 0 {
		return ""
	}
	return blocks
}

// anthropicUpstreamTools 将 OpenAI tools 转换为 Anthropic tools
func anthropicUpstreamTools(raw json.RawMessage) []map[string]any {
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
	if err := json.Unmarshal(raw, &tools); err != nil {
		return nil
	}
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		if t.Type != "function" || t.Function.Name == "" {
			continue
		}
		tool := map[string]any{"name": t.Function.Name}
		if t.Function.Description != "" {
			tool["description"] = t.Function.Description
		}
		// Anthropic 的 input_schema 必填，参数缺失或非对象时回落为空对象 schema
		params, _ := toolObject(t.Function.Parameters)
		if params == nil {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		tool["input_schema"] = params
		out = append(out, tool)
	}
	return out
}

// anthropicUpstreamToolChoice 将 OpenAI tool_choice 映射为 Anthropic tool_choice
// parallel_tool_calls=false 时补充 disable_parallel_tool_use；无 tool_choice 时回落 auto
func anthropicUpstreamToolChoice(raw json.RawMessage, parallel *bool) map[string]any {
	disableParallel := parallel != nil && !*parallel
	choice := parseOpenAIToolChoice(raw)
	if choice == nil {
		if !disableParallel {
			return nil
		}
		choice = map[string]any{"type": "auto"}
	}
	if disableParallel {
		choice["disable_parallel_tool_use"] = true
	}
	return choice
}

// parseOpenAIToolChoice 解析 OpenAI tool_choice（字符串或对象形态）为 Anthropic tool_choice
func parseOpenAIToolChoice(raw json.RawMessage) map[string]any {
	trimmed := trimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil
	}
	if trimmed[0] == '"' {
		var kind string
		if json.Unmarshal(trimmed, &kind) != nil {
			return nil
		}
		return anthropicToolChoiceType(kind)
	}
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
		return map[string]any{"type": "tool", "name": obj.Function.Name}
	}
	return anthropicToolChoiceType(obj.Type)
}

// anthropicToolChoiceType OpenAI tool_choice 类型名 → Anthropic tool_choice
func anthropicToolChoiceType(kind string) map[string]any {
	switch kind {
	case "auto":
		return map[string]any{"type": "auto"}
	case "none":
		return map[string]any{"type": "none"}
	case "required", "any":
		return map[string]any{"type": "any"}
	default:
		return nil
	}
}

// toolArgumentsObject 将 OpenAI function.arguments（JSON 字符串）解析为对象
func toolArgumentsObject(args string) map[string]any {
	obj, _ := toolObject([]byte(args))
	if obj == nil {
		return map[string]any{}
	}
	return obj
}

// toolObject 解析 JSON 对象；非对象或解析失败时返回 nil
func toolObject(raw json.RawMessage) (map[string]any, bool) {
	trimmed := trimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, false
	}
	var obj map[string]any
	if json.Unmarshal(trimmed, &obj) != nil || obj == nil {
		return nil, false
	}
	return obj, true
}

// compactJSON 去除 JSON 中的多余空白，用于回写 function.arguments
func compactJSON(raw json.RawMessage) []byte {
	trimmed := trimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return []byte("{}")
	}
	var buf bytes.Buffer
	if json.Compact(&buf, trimmed) != nil {
		return []byte("{}")
	}
	return buf.Bytes()
}

// toolUseID 返回工具调用 id；缺省时随机生成，保证上游要求的 id 非空
func toolUseID(id string) string {
	if id != "" {
		return id
	}
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("call_%d", time.Now().UnixNano())
	}
	return "call_" + hex.EncodeToString(b)
}

// anthropicStopReasonToOpenAI 将 Anthropic stop_reason 映射为 OpenAI finish_reason
func anthropicStopReasonToOpenAI(reason string) string {
	switch reason {
	case "max_tokens":
		return "length"
	case "tool_use":
		return "tool_calls"
	default:
		// end_turn / stop_sequence / 其它均映射为 stop
		return "stop"
	}
}
