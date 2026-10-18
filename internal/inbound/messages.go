package inbound

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
	"Xin-api/pkg/response"
)

// messagesCodec Anthropic Messages API 入站协议 ⇄ OpenAI Chat
// 流式需跨 chunk 累积 message id / 开启状态 / stop_reason / usage，故每请求实例化
type messagesCodec struct {
	id         string
	model      string
	started    bool   // 是否已下发 message_start 与 content_block_start
	stopReason string // 累积的 stop_reason（来自 finish_reason）
	usage      *chatUsage
}

func (c *messagesCodec) Name() string { return ProtocolMessages }

// NormalizeRequest Anthropic Messages 请求 → 规范 OpenAI Chat 请求
// system → system 消息；messages 仅保留 user/assistant；max_tokens 缺省 4096
func (c *messagesCodec) NormalizeRequest(body []byte, _ NormalizeOptions) ([]byte, error) {
	var req struct {
		Model         string            `json:"model"`
		System        json.RawMessage   `json:"system"`
		Messages      []json.RawMessage `json:"messages"`
		MaxTokens     *int              `json:"max_tokens"`
		Temperature   *float64          `json:"temperature"`
		TopP          *float64          `json:"top_p"`
		StopSequences []string          `json:"stop_sequences"`
		Stream        bool              `json:"stream"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("messages: parse request body failed: %w", err)
	}
	if req.Model == "" {
		return nil, fmt.Errorf("messages: model field is required")
	}
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("messages: messages field is required")
	}

	messages := make([]map[string]any, 0, len(req.Messages)+1)
	if sys := anthropicTextContent(req.System); sys != "" {
		messages = append(messages, map[string]any{"role": "system", "content": sys})
	}
	for _, raw := range req.Messages {
		var m struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		if m.Role != "user" && m.Role != "assistant" {
			continue
		}
		messages = append(messages, map[string]any{"role": m.Role, "content": anthropicTextContent(m.Content)})
	}

	out := map[string]any{
		"model":    req.Model,
		"messages": messages,
		"stream":   req.Stream,
	}
	if req.MaxTokens != nil {
		out["max_tokens"] = *req.MaxTokens
	} else {
		out["max_tokens"] = 4096
	}
	if req.Temperature != nil {
		out["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		out["top_p"] = *req.TopP
	}
	if len(req.StopSequences) > 0 {
		out["stop"] = req.StopSequences
	}
	if req.Stream {
		// 注入 include_usage，保证上游流式响应携带 usage（否则计费回落为估算）
		out["stream_options"] = map[string]any{"include_usage": true}
	}
	return jsonx.Marshal(out)
}

// anthropicTextContent 解析 Anthropic 内容：字符串或 [{type,text}] 块数组，仅拼接 text 块
func anthropicTextContent(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return ""
	}
	var b strings.Builder
	for _, p := range parts {
		if p.Type == "text" {
			b.WriteString(p.Text)
		}
	}
	return b.String()
}

// EncodeResponse OpenAI Chat 非流式响应 → Anthropic Messages 响应
func (c *messagesCodec) EncodeResponse(openAIResponse []byte) ([]byte, error) {
	var cc chatCompletion
	if err := json.Unmarshal(openAIResponse, &cc); err != nil {
		return openAIResponse, nil
	}
	text := ""
	stopReason := "end_turn"
	if len(cc.Choices) > 0 {
		text = cc.Choices[0].Message.Content
		if fr := cc.Choices[0].FinishReason; fr != nil && *fr != "" {
			stopReason = openAIFinishReasonToAnthropic(*fr)
		}
	}
	out := map[string]any{
		"id":            messageID(cc.ID),
		"type":          "message",
		"role":          "assistant",
		"model":         cc.Model,
		"content":       []map[string]any{{"type": "text", "text": text}},
		"stop_reason":   stopReason,
		"stop_sequence": nil,
		"usage":         anthropicUsage(cc.Usage),
	}
	return jsonx.Marshal(out)
}

// EncodeStreamChunk OpenAI SSE chunk → Anthropic SSE 事件流
// 事件顺序：message_start → content_block_start → content_block_delta* → content_block_stop → message_delta → message_stop
func (c *messagesCodec) EncodeStreamChunk(chunk []byte) ([]byte, error) {
	payload := extractSSEPayload(chunk)
	if len(payload) == 0 {
		return nil, nil
	}
	if string(payload) == "[DONE]" {
		if !c.started {
			return nil, nil
		}
		var b bytes.Buffer
		b.Write(c.contentBlockStop())
		b.Write(c.messageDelta())
		b.Write(sseEvent("message_stop", []byte(`{"type":"message_stop"}`)))
		return b.Bytes(), nil
	}
	var ch chatChunk
	if err := json.Unmarshal(payload, &ch); err != nil {
		return nil, nil
	}

	var buf bytes.Buffer
	if !c.started {
		if ch.Model != "" {
			c.model = ch.Model
		}
		c.id = messageID(ch.ID)
		buf.Write(c.messageStart())
		buf.Write(c.contentBlockStart())
		c.started = true
	}
	if len(ch.Choices) > 0 {
		if delta := ch.Choices[0].Delta.Content; delta != "" {
			buf.Write(c.contentBlockDelta(delta))
		}
		if fr := ch.Choices[0].FinishReason; fr != nil && *fr != "" {
			c.stopReason = openAIFinishReasonToAnthropic(*fr)
		}
	}
	if ch.Usage != nil {
		c.usage = ch.Usage
	}
	if buf.Len() == 0 {
		return nil, nil
	}
	return buf.Bytes(), nil
}

// LocalError Anthropic 原生错误体
func (c *messagesCodec) LocalError(bizCode int) (int, []byte) {
	status := response.MapToHTTPStatus(bizCode)
	return status, response.AnthropicErrorBody(status, response.Msg(bizCode))
}

// UpstreamError 上游错误 → Anthropic 原生错误体（保留上游 message）
func (c *messagesCodec) UpstreamError(upstreamStatus int, upstreamBody []byte) (int, []byte) {
	status := upstreamStatus
	if status < http.StatusBadRequest {
		status = http.StatusBadGateway
	}
	msg := extractErrorMessage(upstreamBody)
	if msg == "" {
		msg = http.StatusText(status)
	}
	return status, response.AnthropicErrorBody(status, msg)
}

// messageStart 事件负载
func (c *messagesCodec) messageStart() []byte {
	b, _ := jsonx.Marshal(map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":            c.id,
			"type":          "message",
			"role":          "assistant",
			"model":         c.model,
			"content":       []any{},
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage":         map[string]any{"input_tokens": 0, "output_tokens": 0},
		},
	})
	return sseEvent("message_start", b)
}

// contentBlockStart 事件负载
func (c *messagesCodec) contentBlockStart() []byte {
	b, _ := jsonx.Marshal(map[string]any{
		"type":          "content_block_start",
		"index":         0,
		"content_block": map[string]any{"type": "text", "text": ""},
	})
	return sseEvent("content_block_start", b)
}

// contentBlockDelta 事件负载
func (c *messagesCodec) contentBlockDelta(text string) []byte {
	b, _ := jsonx.Marshal(map[string]any{
		"type":  "content_block_delta",
		"index": 0,
		"delta": map[string]any{"type": "text_delta", "text": text},
	})
	return sseEvent("content_block_delta", b)
}

// contentBlockStop 事件负载
func (c *messagesCodec) contentBlockStop() []byte {
	b, _ := jsonx.Marshal(map[string]any{
		"type":  "content_block_stop",
		"index": 0,
	})
	return sseEvent("content_block_stop", b)
}

// messageDelta 事件负载（含累积 usage）
func (c *messagesCodec) messageDelta() []byte {
	stopReason := c.stopReason
	if stopReason == "" {
		stopReason = "end_turn"
	}
	b, _ := jsonx.Marshal(map[string]any{
		"type":  "message_delta",
		"delta": map[string]any{"stop_reason": stopReason, "stop_sequence": nil},
		"usage": anthropicUsage(c.usage),
	})
	return sseEvent("message_delta", b)
}

// anthropicUsage OpenAI usage → Anthropic usage 字段名
func anthropicUsage(u *chatUsage) map[string]any {
	if u == nil {
		return map[string]any{"input_tokens": 0, "output_tokens": 0}
	}
	return map[string]any{
		"input_tokens":  u.PromptTokens,
		"output_tokens": u.CompletionTokens,
	}
}

// openAIFinishReasonToAnthropic OpenAI finish_reason → Anthropic stop_reason
func openAIFinishReasonToAnthropic(reason string) string {
	switch reason {
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	default:
		return "end_turn"
	}
}

// messageID 由 OpenAI 完成 ID 派生 Anthropic message ID；无源 ID 时随机生成
func messageID(chatID string) string {
	if strings.HasPrefix(chatID, "chatcmpl-") {
		return "msg_" + strings.TrimPrefix(chatID, "chatcmpl-")
	}
	if chatID != "" {
		return "msg_" + chatID
	}
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("msg_%d", time.Now().UnixNano())
	}
	return "msg_" + hex.EncodeToString(b)
}
