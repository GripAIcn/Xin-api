package inbound

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"Xin-api/pkg/jsonx"
)

// responsesCodec OpenAI Responses API 入站协议 ⇄ OpenAI Chat
// 流式需跨 chunk 累积 id / 内容 / 工具调用 / usage / sequence_number，故每请求实例化
type responsesCodec struct {
	id          string
	model       string
	created     int64
	seq         int
	createdSent bool

	usage        *chatUsage
	finishReason string

	outputCount int

	text        strings.Builder
	textStarted bool
	textIndex   int

	reasoning        strings.Builder
	reasoningStarted bool
	reasoningIndex   int

	tools map[int]*responsesToolState
}

// responsesToolState 单个工具调用在流式过程中的累积状态
type responsesToolState struct {
	outputIndex int
	itemID      string
	callID      string
	name        string
	args        strings.Builder
	started     bool
}

func (c *responsesCodec) Name() string { return ProtocolResponses }

// ==========================================
// 请求归一化
// ==========================================

// responsesRequest Responses 请求体中与本网关相关的字段
type responsesRequest struct {
	Model             string          `json:"model"`
	Instructions      json.RawMessage `json:"instructions"`
	Input             json.RawMessage `json:"input"`
	MaxOutputTokens   *int            `json:"max_output_tokens"`
	Temperature       *float64        `json:"temperature"`
	TopP              *float64        `json:"top_p"`
	User              string          `json:"user"`
	Stream            bool            `json:"stream"`
	Tools             json.RawMessage `json:"tools"`
	ToolChoice        json.RawMessage `json:"tool_choice"`
	ParallelToolCalls *bool           `json:"parallel_tool_calls"`
	Reasoning         *struct {
		Effort string `json:"effort"`
	} `json:"reasoning"`
	Text *struct {
		Format json.RawMessage `json:"format"`
	} `json:"text"`
	Metadata json.RawMessage `json:"metadata"`
}

// responsesInputItem Responses input 数组中的单个条目
type responsesInputItem struct {
	Type      string              `json:"type"`
	Role      string              `json:"role"`
	Content   json.RawMessage     `json:"content"`
	CallID    string              `json:"call_id"`
	Name      string              `json:"name"`
	Arguments string              `json:"arguments"`
	Output    json.RawMessage     `json:"output"`
	Summary   []responsesTextPart `json:"summary"`
}

// responsesTextPart Responses 文本片段（summary / content 通用）
type responsesTextPart struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Refusal string `json:"refusal"`
}

// responsesTool Responses 工具定义
type responsesTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
	Strict      *bool           `json:"strict"`
}

// NormalizeRequest Responses 请求 → 规范 OpenAI Chat 请求
func (c *responsesCodec) NormalizeRequest(body []byte, _ NormalizeOptions) ([]byte, error) {
	var req responsesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("responses: parse request body failed: %w", err)
	}
	if req.Model == "" {
		return nil, fmt.Errorf("responses: model field is required")
	}

	messages := make([]map[string]any, 0, 4)
	if sys := responsesText(req.Instructions); sys != "" {
		messages = append(messages, map[string]any{"role": "system", "content": sys})
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
	if tools := responsesTools(req.Tools); len(tools) > 0 {
		out["tools"] = tools
	}
	if tc := responsesToolChoice(req.ToolChoice); tc != nil {
		out["tool_choice"] = tc
	}
	if req.ParallelToolCalls != nil {
		out["parallel_tool_calls"] = *req.ParallelToolCalls
	}
	if req.Reasoning != nil && req.Reasoning.Effort != "" {
		out["reasoning_effort"] = req.Reasoning.Effort
	}
	if rf := responsesResponseFormat(req.Text); rf != nil {
		out["response_format"] = rf
	}
	if len(req.Metadata) > 0 && string(req.Metadata) != "null" {
		out["metadata"] = req.Metadata
	}
	if req.Stream {
		// 注入 include_usage，保证上游流式响应携带 usage（否则计费回落为估算）
		out["stream_options"] = map[string]any{"include_usage": true}
	}
	return jsonx.Marshal(out)
}

// responsesInputMessages 解析 Responses 的 input 字段，按条目类型转换为规范消息
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
	var items []responsesInputItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil
	}

	out := make([]map[string]any, 0, len(items))
	var pendingReasoning strings.Builder
	var lastAssistant map[string]any

	flushPendingReasoning := func() {
		if pendingReasoning.Len() == 0 {
			return
		}
		text := pendingReasoning.String()
		pendingReasoning.Reset()
		if lastAssistant == nil {
			return
		}
		if existing, ok := lastAssistant["reasoning_content"].(string); ok {
			text = existing + text
		}
		lastAssistant["reasoning_content"] = text
	}

	for _, it := range items {
		typ := it.Type
		if typ == "" {
			typ = "message"
		}
		switch typ {
		case "message":
			role := it.Role
			if role == "" {
				continue
			}
			if role == "developer" {
				role = "system"
			}
			msg := map[string]any{"role": role, "content": responsesContent(it.Content)}
			if role == "assistant" && pendingReasoning.Len() > 0 {
				msg["reasoning_content"] = pendingReasoning.String()
				pendingReasoning.Reset()
			}
			out = append(out, msg)
			if role == "assistant" {
				lastAssistant = msg
			} else {
				lastAssistant = nil
			}
		case "function_call":
			tc := map[string]any{
				"id":   it.CallID,
				"type": "function",
				"function": map[string]any{
					"name":      it.Name,
					"arguments": it.Arguments,
				},
			}
			if lastAssistant != nil {
				calls, _ := lastAssistant["tool_calls"].([]any)
				lastAssistant["tool_calls"] = append(calls, tc)
				if _, ok := lastAssistant["content"]; !ok {
					lastAssistant["content"] = ""
				}
				continue
			}
			msg := map[string]any{
				"role":       "assistant",
				"content":    "",
				"tool_calls": []any{tc},
			}
			if pendingReasoning.Len() > 0 {
				msg["reasoning_content"] = pendingReasoning.String()
				pendingReasoning.Reset()
			}
			out = append(out, msg)
			lastAssistant = msg
		case "function_call_output":
			out = append(out, map[string]any{
				"role":         "tool",
				"tool_call_id": it.CallID,
				"content":      responsesText(it.Output),
			})
			lastAssistant = nil
		case "reasoning":
			if text := responsesReasoningText(it); text != "" {
				pendingReasoning.WriteString(text)
			}
		default:
			// item_reference 等有状态条目不在本次支持范围
		}
	}
	flushPendingReasoning()
	return out
}

// responsesReasoningText 提取 reasoning item 的 summary 与 content 文本
func responsesReasoningText(it responsesInputItem) string {
	var b strings.Builder
	for _, p := range it.Summary {
		if p.Text != "" {
			b.WriteString(p.Text)
		}
	}
	if text := responsesText(it.Content); text != "" {
		b.WriteString(text)
	}
	return b.String()
}

// responsesText 解析字符串或文本片段数组，拼接为纯文本
func responsesText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var parts []responsesTextPart
	if err := json.Unmarshal(raw, &parts); err != nil {
		return ""
	}
	var b strings.Builder
	for _, p := range parts {
		if p.Text != "" {
			b.WriteString(p.Text)
		} else if p.Refusal != "" {
			b.WriteString(p.Refusal)
		}
	}
	return b.String()
}

// responsesContent 解析消息 content：纯文本合并为字符串，含图片时输出 OpenAI content 数组
func responsesContent(raw json.RawMessage) any {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var parts []struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		Refusal  string `json:"refusal"`
		ImageURL string `json:"image_url"`
		Detail   string `json:"detail"`
	}
	if err := json.Unmarshal(raw, &parts); err != nil {
		return ""
	}

	var text strings.Builder
	out := make([]any, 0, len(parts))
	hasImage := false
	for _, p := range parts {
		if p.Type == "input_image" {
			if p.ImageURL == "" {
				continue // file_id / file_url 暂不支持
			}
			img := map[string]any{"url": p.ImageURL}
			if p.Detail != "" {
				img["detail"] = p.Detail
			}
			out = append(out, map[string]any{"type": "image_url", "image_url": img})
			hasImage = true
			continue
		}
		t := p.Text
		if t == "" {
			t = p.Refusal
		}
		if t == "" {
			continue
		}
		text.WriteString(t)
		out = append(out, map[string]any{"type": "text", "text": t})
	}
	if !hasImage {
		return text.String()
	}
	return out
}

// responsesTools 将 Responses 函数工具映射为 OpenAI tools
func responsesTools(raw json.RawMessage) []map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var tools []responsesTool
	if err := json.Unmarshal(raw, &tools); err != nil {
		return nil
	}
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		if t.Type != "function" || t.Name == "" {
			continue // 内置工具不在本次支持范围
		}
		fn := map[string]any{"name": t.Name}
		if t.Description != "" {
			fn["description"] = t.Description
		}
		if len(t.Parameters) > 0 {
			fn["parameters"] = t.Parameters
		}
		if t.Strict != nil {
			fn["strict"] = *t.Strict
		}
		out = append(out, map[string]any{"type": "function", "function": fn})
	}
	return out
}

// responsesToolChoice 将 Responses tool_choice 映射为 OpenAI tool_choice
func responsesToolChoice(raw json.RawMessage) any {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var obj struct {
		Type     string `json:"type"`
		Name     string `json:"name"`
		Function *struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	if json.Unmarshal(raw, &obj) != nil {
		return nil
	}
	switch obj.Type {
	case "auto", "none", "required":
		return obj.Type
	case "function":
		name := obj.Name
		if name == "" && obj.Function != nil {
			name = obj.Function.Name
		}
		if name == "" {
			return nil
		}
		return map[string]any{"type": "function", "function": map[string]any{"name": name}}
	default:
		return nil
	}
}

// responsesResponseFormat 将 Responses text.format 映射为 OpenAI response_format
func responsesResponseFormat(text *struct {
	Format json.RawMessage `json:"format"`
}) any {
	if text == nil || len(text.Format) == 0 {
		return nil
	}
	var f struct {
		Type   string          `json:"type"`
		Name   string          `json:"name"`
		Schema json.RawMessage `json:"schema"`
		Strict *bool           `json:"strict"`
	}
	if json.Unmarshal(text.Format, &f) != nil {
		return nil
	}
	switch f.Type {
	case "json_object":
		return map[string]any{"type": "json_object"}
	case "json_schema":
		js := map[string]any{"name": f.Name}
		if len(f.Schema) > 0 {
			js["schema"] = f.Schema
		}
		if f.Strict != nil {
			js["strict"] = *f.Strict
		}
		return map[string]any{"type": "json_schema", "json_schema": js}
	default:
		return nil
	}
}

// ==========================================
// 非流式响应编码
// ==========================================

// EncodeResponse OpenAI Chat 非流式响应 → Responses 响应
func (c *responsesCodec) EncodeResponse(openAIResponse []byte) ([]byte, error) {
	var cc chatCompletion
	if err := json.Unmarshal(openAIResponse, &cc); err != nil {
		return openAIResponse, nil
	}
	created := cc.Created
	if created == 0 {
		created = time.Now().Unix()
	}
	id := responseID(cc.ID)

	var msg chatMessage
	var finishReason string
	if len(cc.Choices) > 0 {
		msg = cc.Choices[0].Message
		if cc.Choices[0].FinishReason != nil {
			finishReason = *cc.Choices[0].FinishReason
		}
	}

	output := make([]any, 0, 2+len(msg.ToolCalls))
	if msg.ReasoningContent != "" {
		output = append(output, responsesReasoningItem(id, msg.ReasoningContent))
	}
	if msg.Content != "" || msg.Refusal != "" || len(msg.ToolCalls) == 0 {
		parts := make([]any, 0, 2)
		if msg.Content != "" {
			parts = append(parts, map[string]any{
				"type": "output_text", "text": msg.Content, "annotations": []any{},
			})
		}
		if msg.Refusal != "" {
			parts = append(parts, map[string]any{"type": "refusal", "refusal": msg.Refusal})
		}
		output = append(output, map[string]any{
			"id":      "msg_" + id,
			"type":    "message",
			"role":    "assistant",
			"status":  "completed",
			"content": parts,
		})
	}
	for i, tc := range msg.ToolCalls {
		output = append(output, map[string]any{
			"id":        fmt.Sprintf("fc_%s_%d", id, i),
			"type":      "function_call",
			"status":    "completed",
			"call_id":   tc.ID,
			"name":      tc.Function.Name,
			"arguments": tc.Function.Arguments,
		})
	}

	resp := map[string]any{
		"id":         id,
		"object":     "response",
		"created_at": created,
		"model":      cc.Model,
		"status":     "completed",
		"output":     output,
	}
	if finishReason == "length" {
		resp["status"] = "incomplete"
		resp["incomplete_details"] = map[string]any{"reason": "max_output_tokens"}
	}
	if cc.Usage != nil {
		resp["usage"] = responsesUsage(cc.Usage)
	}
	return jsonx.Marshal(resp)
}

// responsesReasoningItem 构造非流式 reasoning output item
func responsesReasoningItem(id, text string) map[string]any {
	return map[string]any{
		"id":   "rs_" + id,
		"type": "reasoning",
		"summary": []any{map[string]any{
			"type": "summary_text", "text": text,
		}},
	}
}

// ==========================================
// 流式响应编码
// ==========================================

// EncodeStreamChunk OpenAI SSE chunk → Responses 事件流
func (c *responsesCodec) EncodeStreamChunk(chunk []byte) ([]byte, error) {
	payload := extractSSEPayload(chunk)
	if len(payload) == 0 {
		return nil, nil
	}

	var buf bytes.Buffer
	if string(payload) == "[DONE]" {
		if !c.createdSent {
			return nil, nil
		}
		c.finishStream(&buf)
		return buf.Bytes(), nil
	}

	var ch chatChunk
	if err := json.Unmarshal(payload, &ch); err != nil {
		return nil, nil
	}

	if !c.createdSent {
		c.initStream(&ch, &buf)
	}
	if len(ch.Choices) > 0 {
		choice := ch.Choices[0]
		if choice.Delta.ReasoningContent != "" {
			c.appendReasoning(&buf, choice.Delta.ReasoningContent)
		}
		if choice.Delta.Content != "" {
			c.appendText(&buf, choice.Delta.Content)
		}
		if choice.Delta.Refusal != "" {
			c.appendText(&buf, choice.Delta.Refusal)
		}
		for _, tc := range choice.Delta.ToolCalls {
			c.appendToolCall(&buf, tc)
		}
		if choice.FinishReason != nil && *choice.FinishReason != "" {
			c.finishReason = *choice.FinishReason
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

// initStream 初始化流式状态并下发 response.created / response.in_progress
func (c *responsesCodec) initStream(ch *chatChunk, buf *bytes.Buffer) {
	c.seq = -1
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
	c.tools = make(map[int]*responsesToolState)
	c.createdSent = true

	c.emit(buf, "response.created", map[string]any{"response": c.snapshot("in_progress", nil, "")})
	c.emit(buf, "response.in_progress", map[string]any{"response": c.snapshot("in_progress", nil, "")})
}

// emit 写入一个带 sequence_number 的 Responses 事件
func (c *responsesCodec) emit(buf *bytes.Buffer, event string, payload map[string]any) {
	c.seq++
	payload["type"] = event
	payload["sequence_number"] = c.seq
	data, err := jsonx.Marshal(payload)
	if err != nil {
		return
	}
	buf.Write(sseEvent(event, data))
}

// appendText 追加文本增量，首次出现时补发 output_item.added / content_part.added
func (c *responsesCodec) appendText(buf *bytes.Buffer, delta string) {
	if !c.textStarted {
		c.textStarted = true
		c.textIndex = c.outputCount
		c.outputCount++
		c.emit(buf, "response.output_item.added", map[string]any{
			"output_index": c.textIndex,
			"item":         c.messageItem("in_progress", ""),
		})
		c.emit(buf, "response.content_part.added", map[string]any{
			"item_id":       "msg_" + c.id,
			"output_index":  c.textIndex,
			"content_index": 0,
			"part":          map[string]any{"type": "output_text", "text": "", "annotations": []any{}},
		})
	}
	c.text.WriteString(delta)
	c.emit(buf, "response.output_text.delta", map[string]any{
		"item_id":       "msg_" + c.id,
		"output_index":  c.textIndex,
		"content_index": 0,
		"delta":         delta,
		"logprobs":      []any{},
	})
}

// appendReasoning 追加推理增量，首次出现时补发 reasoning output item
func (c *responsesCodec) appendReasoning(buf *bytes.Buffer, delta string) {
	if !c.reasoningStarted {
		c.reasoningStarted = true
		c.reasoningIndex = c.outputCount
		c.outputCount++
		c.emit(buf, "response.output_item.added", map[string]any{
			"output_index": c.reasoningIndex,
			"item":         c.reasoningItem("in_progress", ""),
		})
	}
	c.reasoning.WriteString(delta)
	c.emit(buf, "response.reasoning_summary_text.delta", map[string]any{
		"item_id":       "rs_" + c.id,
		"output_index":  c.reasoningIndex,
		"summary_index": 0,
		"delta":         delta,
	})
}

// appendToolCall 按 tool_calls[].index 累积工具调用并下发增量事件
func (c *responsesCodec) appendToolCall(buf *bytes.Buffer, tc chatToolCall) {
	st, ok := c.tools[tc.Index]
	if !ok {
		st = &responsesToolState{
			outputIndex: c.outputCount,
			itemID:      fmt.Sprintf("fc_%s_%d", c.id, tc.Index),
		}
		c.outputCount++
		c.tools[tc.Index] = st
	}
	if tc.ID != "" {
		st.callID = tc.ID
	}
	if tc.Function.Name != "" {
		st.name = tc.Function.Name
	}
	if !st.started {
		st.started = true
		c.emit(buf, "response.output_item.added", map[string]any{
			"output_index": st.outputIndex,
			"item": map[string]any{
				"id":        st.itemID,
				"type":      "function_call",
				"status":    "in_progress",
				"call_id":   st.callID,
				"name":      st.name,
				"arguments": "",
			},
		})
	}
	if args := tc.Function.Arguments; args != "" {
		st.args.WriteString(args)
		c.emit(buf, "response.function_call_arguments.delta", map[string]any{
			"item_id":      st.itemID,
			"output_index": st.outputIndex,
			"delta":        args,
		})
	}
}

// finishStream 关闭所有打开的 output item 并下发 completed / incomplete
func (c *responsesCodec) finishStream(buf *bytes.Buffer) {
	if c.reasoningStarted {
		text := c.reasoning.String()
		c.emit(buf, "response.reasoning_summary_text.done", map[string]any{
			"item_id":       "rs_" + c.id,
			"output_index":  c.reasoningIndex,
			"summary_index": 0,
			"text":          text,
		})
		c.emit(buf, "response.output_item.done", map[string]any{
			"output_index": c.reasoningIndex,
			"item":         c.reasoningItem("completed", text),
		})
	}
	if c.textStarted {
		text := c.text.String()
		part := map[string]any{"type": "output_text", "text": text, "annotations": []any{}}
		c.emit(buf, "response.output_text.done", map[string]any{
			"item_id":       "msg_" + c.id,
			"output_index":  c.textIndex,
			"content_index": 0,
			"text":          text,
			"logprobs":      []any{},
		})
		c.emit(buf, "response.content_part.done", map[string]any{
			"item_id":       "msg_" + c.id,
			"output_index":  c.textIndex,
			"content_index": 0,
			"part":          part,
		})
		c.emit(buf, "response.output_item.done", map[string]any{
			"output_index": c.textIndex,
			"item":         map[string]any{"id": "msg_" + c.id, "type": "message", "role": "assistant", "status": "completed", "content": []any{part}},
		})
	}

	keys := make([]int, 0, len(c.tools))
	for k := range c.tools {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		st := c.tools[k]
		args := st.args.String()
		c.emit(buf, "response.function_call_arguments.done", map[string]any{
			"item_id":      st.itemID,
			"output_index": st.outputIndex,
			"arguments":    args,
		})
		c.emit(buf, "response.output_item.done", map[string]any{
			"output_index": st.outputIndex,
			"item": map[string]any{
				"id":        st.itemID,
				"type":      "function_call",
				"status":    "completed",
				"call_id":   st.callID,
				"name":      st.name,
				"arguments": args,
			},
		})
	}

	status := "completed"
	reason := ""
	if c.finishReason == "length" {
		status = "incomplete"
		reason = "max_output_tokens"
	}
	if status == "incomplete" {
		c.emit(buf, "response.incomplete", map[string]any{"response": c.snapshot(status, c.finalOutput(), reason)})
		return
	}
	c.emit(buf, "response.completed", map[string]any{"response": c.snapshot(status, c.finalOutput(), "")})
}

// snapshot 构造 response 对象（created / in_progress / completed / incomplete 共用）
func (c *responsesCodec) snapshot(status string, output []any, incompleteReason string) map[string]any {
	if output == nil {
		output = []any{}
	}
	resp := map[string]any{
		"id":         c.id,
		"object":     "response",
		"created_at": c.created,
		"model":      c.model,
		"status":     status,
		"output":     output,
	}
	if incompleteReason != "" {
		resp["incomplete_details"] = map[string]any{"reason": incompleteReason}
	}
	if c.usage != nil && status != "in_progress" {
		resp["usage"] = responsesUsage(c.usage)
	}
	return resp
}

// finalOutput 按 output_index 顺序汇总最终 output items
func (c *responsesCodec) finalOutput() []any {
	type indexedItem struct {
		index int
		item  map[string]any
	}
	items := make([]indexedItem, 0, 2+len(c.tools))
	if c.reasoningStarted {
		items = append(items, indexedItem{c.reasoningIndex, c.reasoningItem("completed", c.reasoning.String())})
	}
	if c.textStarted {
		items = append(items, indexedItem{c.textIndex, c.messageItem("completed", c.text.String())})
	}
	for _, st := range c.tools {
		items = append(items, indexedItem{st.outputIndex, map[string]any{
			"id":        st.itemID,
			"type":      "function_call",
			"status":    "completed",
			"call_id":   st.callID,
			"name":      st.name,
			"arguments": st.args.String(),
		}})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].index < items[j].index })
	out := make([]any, 0, len(items))
	for _, it := range items {
		out = append(out, it.item)
	}
	return out
}

// messageItem 构造 assistant message output item
func (c *responsesCodec) messageItem(status, text string) map[string]any {
	content := []any{}
	if status == "completed" {
		content = append(content, map[string]any{"type": "output_text", "text": text, "annotations": []any{}})
	}
	return map[string]any{
		"id":      "msg_" + c.id,
		"type":    "message",
		"role":    "assistant",
		"status":  status,
		"content": content,
	}
}

// reasoningItem 构造 reasoning output item
func (c *responsesCodec) reasoningItem(status, text string) map[string]any {
	summary := []any{}
	if status == "completed" {
		summary = append(summary, map[string]any{"type": "summary_text", "text": text})
	}
	return map[string]any{
		"id":      "rs_" + c.id,
		"type":    "reasoning",
		"summary": summary,
		"status":  status,
	}
}

func (c *responsesCodec) LocalError(bizCode int) (int, []byte) {
	return openAILocalError(bizCode)
}

func (c *responsesCodec) UpstreamError(upstreamStatus int, upstreamBody []byte) (int, []byte) {
	return upstreamStatus, upstreamBody
}

// responsesUsage OpenAI usage → Responses usage 字段名
func responsesUsage(u *chatUsage) map[string]any {
	var cached, reasoning int64
	if u.PromptTokensDetails != nil {
		cached = u.PromptTokensDetails.CachedTokens
	}
	if u.CompletionTokensDetails != nil {
		reasoning = u.CompletionTokensDetails.ReasoningTokens
	}
	return map[string]any{
		"input_tokens":  u.PromptTokens,
		"output_tokens": u.CompletionTokens,
		"total_tokens":  u.TotalTokens,
		"input_tokens_details": map[string]any{
			"cached_tokens": cached,
		},
		"output_tokens_details": map[string]any{
			"reasoning_tokens": reasoning,
		},
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
