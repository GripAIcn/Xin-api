package inbound

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// responsesCodec OpenAI Responses API 入站协议 ⇄ OpenAI Chat
// 流式需跨 chunk 累积 id / 文本 / usage / sequence_number，故每请求实例化
type responsesCodec struct {
	id          string
	model       string
	created     int64
	seq         int
	createdSent bool
	text        strings.Builder
	usage       *chatUsage
}

func (c *responsesCodec) Name() string { return ProtocolResponses }

// NormalizeRequest Responses 请求 → 规范 OpenAI Chat 请求
// instructions → system 消息；input → user/assistant 消息；max_output_tokens → max_tokens
func (c *responsesCodec) NormalizeRequest(body []byte, _ NormalizeOptions) ([]byte, error) {
	var req struct {
		Model           string          `json:"model"`
		Instructions    string          `json:"instructions"`
		Input           json.RawMessage `json:"input"`
		MaxOutputTokens *int            `json:"max_output_tokens"`
		Temperature     *float64        `json:"temperature"`
		TopP            *float64        `json:"top_p"`
		User            string          `json:"user"`
		Stream          bool            `json:"stream"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("responses: parse request body failed: %w", err)
	}
	if req.Model == "" {
		return nil, fmt.Errorf("responses: model field is required")
	}

	messages := make([]map[string]any, 0, 4)
	if req.Instructions != "" {
		messages = append(messages, map[string]any{"role": "system", "content": req.Instructions})
	}
	messages = append(messages, responsesInputMessages(req.Input)...)

	out := map[string]any{
		"model":    req.Model,
		"messages": messages,
		"stream":   req.Stream,
	}
	if req.MaxOutputTokens != nil {
		out["max_tokens"] = *req.MaxOutputTokens
	}
	if req.Temperature != nil {
		out["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		out["top_p"] = *req.TopP
	}
	if req.User != "" {
		out["user"] = req.User
	}
	if req.Stream {
		// 注入 include_usage，保证上游流式响应携带 usage（否则计费回落为估算）
		out["stream_options"] = map[string]any{"include_usage": true}
	}
	return json.Marshal(out)
}

// responsesInputMessages 解析 Responses 的 input 字段：字符串或消息数组
func responsesInputMessages(raw json.RawMessage) []map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if s == "" {
			return nil
		}
		return []map[string]any{{"role": "user", "content": s}}
	}
	var items []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		role := it.Role
		if role == "" {
			continue // 非 message 类条目（如 function_call）暂不映射
		}
		if role == "developer" {
			role = "system"
		}
		out = append(out, map[string]any{"role": role, "content": responsesContentText(it.Content)})
	}
	return out
}

// responsesContentText 解析 content：字符串或 [{type,text}] 数组，拼接纯文本
func responsesContentText(raw json.RawMessage) string {
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
		b.WriteString(p.Text)
	}
	return b.String()
}

// EncodeResponse OpenAI Chat 非流式响应 → Responses 响应
func (c *responsesCodec) EncodeResponse(openAIResponse []byte) ([]byte, error) {
	var cc chatCompletion
	if err := json.Unmarshal(openAIResponse, &cc); err != nil {
		return openAIResponse, nil
	}
	content := ""
	if len(cc.Choices) > 0 {
		content = cc.Choices[0].Message.Content
	}
	created := cc.Created
	if created == 0 {
		created = time.Now().Unix()
	}
	id := responseID(cc.ID)
	resp := map[string]any{
		"id":         id,
		"object":     "response",
		"created_at": created,
		"model":      cc.Model,
		"status":     "completed",
		"output": []map[string]any{{
			"id":      "msg_" + id,
			"type":    "message",
			"role":    "assistant",
			"status":  "completed",
			"content": []map[string]any{{"type": "output_text", "text": content, "annotations": []any{}}},
		}},
	}
	if cc.Usage != nil {
		resp["usage"] = responsesUsage(cc.Usage)
	}
	return json.Marshal(resp)
}

// EncodeStreamChunk OpenAI SSE chunk → Responses 事件流（最小可用子集：created / 文本增量 / completed）
func (c *responsesCodec) EncodeStreamChunk(chunk []byte) ([]byte, error) {
	payload := extractSSEPayload(chunk)
	if len(payload) == 0 {
		return nil, nil
	}
	if string(payload) == "[DONE]" {
		if !c.createdSent {
			return nil, nil
		}
		return sseEvent("response.completed", c.completedPayload()), nil
	}
	var ch chatChunk
	if err := json.Unmarshal(payload, &ch); err != nil {
		return nil, nil
	}

	var buf bytes.Buffer
	if !c.createdSent {
		if ch.ID != "" {
			c.id = responseID(ch.ID)
		}
		if c.id == "" {
			c.id = responseID("")
		}
		if ch.Model != "" {
			c.model = ch.Model
		}
		if ch.Created > 0 {
			c.created = ch.Created
		}
		if c.created == 0 {
			c.created = time.Now().Unix()
		}
		c.createdSent = true
		buf.Write(sseEvent("response.created", c.createdPayload()))
	}
	if len(ch.Choices) > 0 {
		if delta := ch.Choices[0].Delta.Content; delta != "" {
			c.text.WriteString(delta)
			c.seq++
			buf.Write(sseEvent("response.output_text.delta", c.deltaPayload(delta)))
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

// createdPayload response.created 事件负载
func (c *responsesCodec) createdPayload() []byte {
	b, _ := json.Marshal(map[string]any{
		"type": "response.created",
		"response": map[string]any{
			"id":         c.id,
			"object":     "response",
			"created_at": c.created,
			"model":      c.model,
			"status":     "in_progress",
			"output":     []any{},
		},
	})
	return b
}

// deltaPayload response.output_text.delta 事件负载
func (c *responsesCodec) deltaPayload(delta string) []byte {
	b, _ := json.Marshal(map[string]any{
		"type":            "response.output_text.delta",
		"item_id":         "msg_" + c.id,
		"output_index":    0,
		"content_index":   0,
		"delta":           delta,
		"sequence_number": c.seq,
	})
	return b
}

// completedPayload response.completed 事件负载（含累积文本与 usage）
func (c *responsesCodec) completedPayload() []byte {
	resp := map[string]any{
		"id":         c.id,
		"object":     "response",
		"created_at": c.created,
		"model":      c.model,
		"status":     "completed",
		"output": []map[string]any{{
			"id":      "msg_" + c.id,
			"type":    "message",
			"role":    "assistant",
			"status":  "completed",
			"content": []map[string]any{{"type": "output_text", "text": c.text.String(), "annotations": []any{}}},
		}},
	}
	if c.usage != nil {
		resp["usage"] = responsesUsage(c.usage)
	}
	b, _ := json.Marshal(map[string]any{"type": "response.completed", "response": resp})
	return b
}

func (c *responsesCodec) LocalError(bizCode int) (int, []byte) {
	return openAILocalError(bizCode)
}

func (c *responsesCodec) UpstreamError(upstreamStatus int, upstreamBody []byte) (int, []byte) {
	return upstreamStatus, upstreamBody
}

// responsesUsage OpenAI usage → Responses usage 字段名
func responsesUsage(u *chatUsage) map[string]any {
	return map[string]any{
		"input_tokens":  u.PromptTokens,
		"output_tokens": u.CompletionTokens,
		"total_tokens":  u.TotalTokens,
	}
}

// responseID 由 OpenAI 完成 ID 派生 Responses ID；无源 ID 时随机生成
func responseID(chatID string) string {
	if strings.HasPrefix(chatID, "chatcmpl-") {
		return "resp_" + strings.TrimPrefix(chatID, "chatcmpl-")
	}
	if chatID != "" {
		return "resp_" + chatID
	}
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("resp_%d", time.Now().UnixNano())
	}
	return "resp_" + hex.EncodeToString(b)
}
