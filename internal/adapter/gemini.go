package adapter

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"Xin-api/pkg/jsonx"
)

type geminiAdapter struct{}

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

	// messages → contents / systemInstruction
	var contents []map[string]any
	var systemTexts []string
	for _, msg := range req.Messages {
		text, err := parseGeminiContent(msg.Content)
		if err != nil {
			return nil, fmt.Errorf("gemini: parse message content failed: %w", err)
		}
		switch msg.Role {
		case "system":
			if text != "" {
				systemTexts = append(systemTexts, text)
			}
		case "assistant":
			contents = append(contents, map[string]any{
				"role":  "model",
				"parts": []map[string]any{{"text": text}},
			})
		default: // user 及其它角色统一按用户消息处理
			contents = append(contents, map[string]any{
				"role":  "user",
				"parts": []map[string]any{{"text": text}},
			})
		}
	}

	out := map[string]any{
		"contents": contents,
	}
	if len(systemTexts) > 0 {
		out["systemInstruction"] = map[string]any{
			"parts": []map[string]any{{"text": strings.Join(systemTexts, "\n")}},
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
	if len(resp.Candidates) > 0 {
		content = resp.Candidates[0].Text()
		finishReason = resp.Candidates[0].FinishReason
	}

	out := map[string]any{
		"id":      newChatCompletionID(),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   externalModel,
		"choices": []map[string]any{
			{
				"index":         0,
				"message":       map[string]any{"role": "assistant", "content": content},
				"finish_reason": mapGeminiFinishReason(finishReason),
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
	// finish_reason 缺省时为 null
	var finishReason any
	if candidate.FinishReason != "" {
		finishReason = mapGeminiFinishReason(candidate.FinishReason)
	}

	out := map[string]any{
		"id":      newChatCompletionID(),
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   externalModel,
		"choices": []map[string]any{
			{
				"index":         0,
				"delta":         map[string]any{"content": candidate.Text()},
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
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// geminiCandidate Gemini 回复候选，可能包含多个文本部分
type geminiCandidate struct {
	Content struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
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

// parseGeminiContent 解析 OpenAI 消息 content：字符串或 [{type:"text",text:"..."}] 数组
func parseGeminiContent(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return "", err
	}
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(p.Text)
	}
	return b.String(), nil
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
