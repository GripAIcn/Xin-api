package adapter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"Xin-api/pkg/jsonx"
)

// anthropicAdapter Anthropic(Claude) 协议适配器：把入站 OpenAI Chat 请求
// 转换为 Anthropic Messages 请求，并把 Anthropic 响应/流事件转回 OpenAI 形态
type anthropicAdapter struct{}

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
		Model       string            `json:"model"`
		Messages    []json.RawMessage `json:"messages"`
		MaxTokens   *int              `json:"max_tokens"`
		Stream      bool              `json:"stream"`
		Temperature *float64          `json:"temperature"`
		TopP        *float64          `json:"top_p"`
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
	for _, rawMsg := range in.Messages {
		var m struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(rawMsg, &m); err != nil {
			continue
		}
		switch m.Role {
		case "system":
			if text := chatContentText(m.Content); text != "" {
				systemParts = append(systemParts, text)
			}
		case "user", "assistant":
			messages = append(messages, anthropicMessage{Role: m.Role, Content: anthropicChatContent(m.Content)})
		}
	}

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
			Type string `json:"type"`
			Text string `json:"text"`
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

	// 仅拼接 type==text 的文本块，忽略 tool_use 等其它类型
	var sb strings.Builder
	for _, block := range resp.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}

	out := map[string]any{
		"id":      resp.ID,
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": sb.String(),
				},
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
func (a *anthropicAdapter) ConvertStreamChunk(chunk []byte, externalModel string) ([]byte, error) {
	data := extractSSEData(chunk)
	if len(data) == 0 {
		return nil, nil
	}

	var ev struct {
		Type    string `json:"type"`
		Message struct {
			ID string `json:"id"`
		} `json:"message"`
		Delta struct {
			Type       string `json:"type"`
			Text       string `json:"text"`
			StopReason string `json:"stop_reason"`
		} `json:"delta"`
	}
	if err := json.Unmarshal(data, &ev); err != nil {
		return nil, nil
	}

	id := ev.Message.ID
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
	case "content_block_delta":
		// 仅处理文本增量，其它 delta（如 input_json_delta）忽略
		if ev.Delta.Type != "text_delta" {
			return nil, nil
		}
		return buildChunk(map[string]any{"content": ev.Delta.Text}, nil), nil
	case "message_delta":
		reason := anthropicStopReasonToOpenAI(ev.Delta.StopReason)
		return buildChunk(map[string]any{}, &reason), nil
	case "message_stop":
		return openaiSSE([]byte("[DONE]")), nil
	default:
		// ping / content_block_start / content_block_stop 等无需下发
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
