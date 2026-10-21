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
// 流式需跨 chunk 累积 message id / 开启状态 / stop_reason / usage / content block 序号，故每请求实例化
type messagesCodec struct {
	id         string
	model      string
	started    bool   // 是否已下发 message_start
	stopReason string // 累积的 stop_reason（来自 finish_reason）
	usage      *chatUsage

	nextIndex   int                        // 下一个 content block 序号
	openIndex   int                        // 当前已 start 且未 stop 的块序号，-1 表示无
	textStarted bool                       // 文本块是否已开启
	textIndex   int                        // 文本块序号
	tools       map[int]*messagesToolState // OpenAI tool_calls index → Anthropic 块状态
}

// messagesToolState 单个工具调用在流式过程中的累积状态
type messagesToolState struct {
	index   int    // Anthropic content block 序号
	id      string // tool_use id
	name    string // 工具名
	started bool   // 是否已下发 content_block_start
}

func (c *messagesCodec) Name() string { return ProtocolMessages }

// NormalizeRequest Anthropic Messages 请求 → 规范 OpenAI Chat 请求
// system → system 消息；messages 保留 user/assistant（含 tool_use / tool_result）；max_tokens 缺省 4096
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
		Tools         json.RawMessage   `json:"tools"`
		ToolChoice    json.RawMessage   `json:"tool_choice"`
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
		switch m.Role {
		case "user":
			// tool_result 块展开为独立 tool 消息，文本/图片块作为一条 user 消息
			messages = append(messages, anthropicUserMessages(m.Content)...)
		case "assistant":
			messages = append(messages, anthropicAssistantMessage(m.Content))
		}
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
	if tools := anthropicChatTools(req.Tools); len(tools) > 0 {
		out["tools"] = tools
	}
	if tc, disableParallel := anthropicChatToolChoice(req.ToolChoice); tc != nil {
		out["tool_choice"] = tc
		if disableParallel {
			out["parallel_tool_calls"] = false
		}
	}
	if req.Stream {
		// 注入 include_usage，保证上游流式响应携带 usage（否则计费回落为估算）
		out["stream_options"] = map[string]any{"include_usage": true}
	}
	return jsonx.Marshal(out)
}

// anthropicContentBlock Anthropic 内容块（text / image / tool_use / tool_result）
type anthropicContentBlock struct {
	Type      string                `json:"type"`
	Text      string                `json:"text"`
	Source    *anthropicImageSource `json:"source"`
	ID        string                `json:"id"`          // tool_use：工具调用 id
	Name      string                `json:"name"`        // tool_use：工具名
	Input     json.RawMessage       `json:"input"`       // tool_use：工具入参对象
	ToolUseID string                `json:"tool_use_id"` // tool_result：对应的 tool_use id
	Content   json.RawMessage       `json:"content"`     // tool_result：工具输出
}

// anthropicImageSource Anthropic image 块的 source 字段
type anthropicImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
	URL       string `json:"url"`
}

// anthropicChatContent 解析 Anthropic 内容：字符串或块数组
// 仅含文本时返回字符串；含 image 块时返回 OpenAI content part 数组（tool_use/tool_result 块不参与）
func anthropicChatContent(raw json.RawMessage) any {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var parts []anthropicContentBlock
	if err := json.Unmarshal(raw, &parts); err != nil {
		return ""
	}
	return anthropicBlocksToContent(parts)
}

// anthropicBlocksToContent 将 Anthropic 内容块数组转为 OpenAI content
// 仅含文本时返回字符串；含 image 块时返回 content part 数组
func anthropicBlocksToContent(parts []anthropicContentBlock) any {
	hasImage := false
	for _, p := range parts {
		if p.Type == "image" && anthropicImageURL(p.Source) != "" {
			hasImage = true
			break
		}
	}
	if !hasImage {
		var b strings.Builder
		for _, p := range parts {
			if p.Type == "text" {
				b.WriteString(p.Text)
			}
		}
		return b.String()
	}
	out := make([]map[string]any, 0, len(parts))
	for _, p := range parts {
		switch p.Type {
		case "text":
			out = append(out, map[string]any{"type": "text", "text": p.Text})
		case "image":
			url := anthropicImageURL(p.Source)
			if url == "" {
				continue
			}
			out = append(out, map[string]any{
				"type":      "image_url",
				"image_url": map[string]any{"url": url},
			})
		}
	}
	return out
}

// anthropicImageURL 将 Anthropic image source 转为 OpenAI 可用的图片 URL
// base64 → data URI；url → 原始 URL
func anthropicImageURL(src *anthropicImageSource) string {
	if src == nil {
		return ""
	}
	switch src.Type {
	case "base64":
		if src.MediaType == "" || src.Data == "" {
			return ""
		}
		return "data:" + src.MediaType + ";base64," + src.Data
	case "url":
		return src.URL
	default:
		return ""
	}
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
	var parts []anthropicContentBlock
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

// anthropicUserMessages 将 Anthropic user 消息拆为规范消息：
// tool_result 块 → 独立的 OpenAI tool 消息；其余（text/image）→ 一条 user 消息
func anthropicUserMessages(raw json.RawMessage) []map[string]any {
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
	var parts []anthropicContentBlock
	if err := json.Unmarshal(raw, &parts); err != nil {
		return nil
	}

	out := make([]map[string]any, 0, len(parts))
	rest := make([]anthropicContentBlock, 0, len(parts))
	for _, p := range parts {
		if p.Type == "tool_result" {
			out = append(out, map[string]any{
				"role":         "tool",
				"tool_call_id": p.ToolUseID,
				"content":      anthropicToolResultText(p.Content),
			})
			continue
		}
		rest = append(rest, p)
	}
	// 其余部分内容为空时不产出空 user 消息，避免上游拒收
	if len(rest) > 0 {
		content := anthropicBlocksToContent(rest)
		if text, ok := content.(string); !ok || text != "" {
			out = append(out, map[string]any{"role": "user", "content": content})
		}
	}
	return out
}

// anthropicToolResultText 提取 tool_result 的文本输出（content 为字符串或内容块数组）
func anthropicToolResultText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var parts []anthropicContentBlock
	if err := json.Unmarshal(raw, &parts); err != nil {
		return ""
	}
	// 仅保留文本块：OpenAI tool 消息的 content 不支持图片
	var b strings.Builder
	for _, p := range parts {
		if p.Type == "text" {
			b.WriteString(p.Text)
		}
	}
	return b.String()
}

// anthropicAssistantMessage Anthropic assistant 内容 → 规范 assistant 消息（文本/图片 + tool_calls）
func anthropicAssistantMessage(raw json.RawMessage) map[string]any {
	msg := map[string]any{"role": "assistant", "content": anthropicChatContent(raw)}
	if len(raw) == 0 {
		return msg
	}
	var parts []anthropicContentBlock
	if err := json.Unmarshal(raw, &parts); err != nil {
		return msg
	}
	calls := make([]map[string]any, 0, len(parts))
	for _, p := range parts {
		if p.Type != "tool_use" || p.ID == "" {
			continue
		}
		calls = append(calls, map[string]any{
			"id":   p.ID,
			"type": "function",
			"function": map[string]any{
				"name":      p.Name,
				"arguments": anthropicToolArguments(p.Input),
			},
		})
	}
	if len(calls) > 0 {
		msg["tool_calls"] = calls
	}
	return msg
}

// anthropicToolArguments Anthropic tool_use.input（对象）→ OpenAI function.arguments（JSON 字符串）
func anthropicToolArguments(raw json.RawMessage) string {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return "{}"
	}
	// 已是字符串形态时直接沿用，避免二次编码
	if trimmed[0] == '"' {
		var s string
		if json.Unmarshal(trimmed, &s) == nil {
			return s
		}
	}
	return string(trimmed)
}

// anthropicChatTools Anthropic tools → OpenAI tools
// {name, description, input_schema} → {type:"function", function:{name, description, parameters}}
func anthropicChatTools(raw json.RawMessage) []map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var tools []struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		InputSchema json.RawMessage `json:"input_schema"`
	}
	if err := json.Unmarshal(raw, &tools); err != nil {
		return nil
	}
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		if t.Name == "" {
			continue
		}
		fn := map[string]any{"name": t.Name}
		if t.Description != "" {
			fn["description"] = t.Description
		}
		if len(t.InputSchema) > 0 && string(t.InputSchema) != "null" {
			fn["parameters"] = t.InputSchema
		}
		out = append(out, map[string]any{"type": "function", "function": fn})
	}
	return out
}

// anthropicChatToolChoice Anthropic tool_choice → (OpenAI tool_choice, disable_parallel_tool_use)
// auto → auto；any → required；none → none；tool → 具名 function
func anthropicChatToolChoice(raw json.RawMessage) (any, bool) {
	if len(raw) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return nil, false
	}
	var tc struct {
		Type                   string `json:"type"`
		Name                   string `json:"name"`
		DisableParallelToolUse bool   `json:"disable_parallel_tool_use"`
	}
	if err := json.Unmarshal(raw, &tc); err != nil {
		return nil, false
	}
	switch tc.Type {
	case "auto":
		return "auto", tc.DisableParallelToolUse
	case "any":
		return "required", tc.DisableParallelToolUse
	case "none":
		return "none", tc.DisableParallelToolUse
	case "tool":
		if tc.Name == "" {
			return nil, tc.DisableParallelToolUse
		}
		return map[string]any{"type": "function", "function": map[string]any{"name": tc.Name}}, tc.DisableParallelToolUse
	default:
		return nil, tc.DisableParallelToolUse
	}
}

// EncodeResponse OpenAI Chat 非流式响应 → Anthropic Messages 响应
// 文本块与 tool_use 块按序输出：文本为空且存在工具调用时省略空文本块
func (c *messagesCodec) EncodeResponse(openAIResponse []byte) ([]byte, error) {
	var cc chatCompletion
	if err := json.Unmarshal(openAIResponse, &cc); err != nil {
		return openAIResponse, nil
	}
	text := ""
	stopReason := "end_turn"
	var toolCalls []chatToolCall
	if len(cc.Choices) > 0 {
		text = cc.Choices[0].Message.Content
		toolCalls = cc.Choices[0].Message.ToolCalls
		if fr := cc.Choices[0].FinishReason; fr != nil && *fr != "" {
			stopReason = openAIFinishReasonToAnthropic(*fr)
		}
	}
	content := make([]map[string]any, 0, 1+len(toolCalls))
	if text != "" || len(toolCalls) == 0 {
		content = append(content, map[string]any{"type": "text", "text": text})
	}
	for _, tc := range toolCalls {
		content = append(content, map[string]any{
			"type":  "tool_use",
			"id":    toolUseID(tc.ID),
			"name":  tc.Function.Name,
			"input": toolArgumentsToInput(tc.Function.Arguments),
		})
	}
	out := map[string]any{
		"id":            messageID(cc.ID),
		"type":          "message",
		"role":          "assistant",
		"model":         cc.Model,
		"content":       content,
		"stop_reason":   stopReason,
		"stop_sequence": nil,
		"usage":         anthropicUsage(cc.Usage),
	}
	return jsonx.Marshal(out)
}

// toolArgumentsToInput OpenAI function.arguments（JSON 字符串）→ Anthropic tool_use.input（对象）
func toolArgumentsToInput(args string) any {
	trimmed := strings.TrimSpace(args)
	if trimmed == "" {
		return map[string]any{}
	}
	var v any
	if err := json.Unmarshal([]byte(trimmed), &v); err != nil || v == nil {
		return map[string]any{}
	}
	return v
}

// toolUseID 返回 tool_use id；缺省时随机生成，保证 Anthropic 侧 id 非空
func toolUseID(id string) string {
	if id != "" {
		return id
	}
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("toolu_%d", time.Now().UnixNano())
	}
	return "toolu_" + hex.EncodeToString(b)
}

// EncodeStreamChunk OpenAI SSE chunk → Anthropic SSE 事件流
// 事件顺序：message_start →（text 块 / 逐个 tool_use 块）→ message_delta → message_stop
// content block 序号按出现顺序递增，文本块与每个工具调用各自成块
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
		c.closeContentBlock(&b)
		b.Write(c.messageDelta())
		b.Write(sseEvent("message_stop", []byte(`{"type":"message_stop"}`)))
		return b.Bytes(), nil
	}
	var ch chatChunk
	if err := json.Unmarshal(payload, &ch); err != nil {
		return nil, nil
	}

	var buf bytes.Buffer
	// usage 先于 message_start 合并：首块可能已携带输入侧用量（含缓存读写）
	c.usage = mergeChatUsage(c.usage, ch.Usage)
	if !c.started {
		if ch.Model != "" {
			c.model = ch.Model
		}
		c.id = messageID(ch.ID)
		c.nextIndex = 0
		c.openIndex = -1
		c.tools = make(map[int]*messagesToolState)
		buf.Write(c.messageStart())
		c.started = true
	}
	if len(ch.Choices) > 0 {
		choice := ch.Choices[0]
		if delta := choice.Delta.Content; delta != "" {
			c.writeTextDelta(&buf, delta)
		}
		for _, tc := range choice.Delta.ToolCalls {
			c.writeToolCall(&buf, tc)
		}
		if fr := choice.FinishReason; fr != nil && *fr != "" {
			c.stopReason = openAIFinishReasonToAnthropic(*fr)
		}
	}
	if buf.Len() == 0 {
		return nil, nil
	}
	return buf.Bytes(), nil
}

// writeTextDelta 追加文本增量；首次出现时先开启 text 块
func (c *messagesCodec) writeTextDelta(buf *bytes.Buffer, delta string) {
	if !c.textStarted {
		c.textStarted = true
		c.textIndex = c.openContentBlock(buf, map[string]any{"type": "text", "text": ""})
	}
	buf.Write(c.contentBlockDelta(c.textIndex, map[string]any{"type": "text_delta", "text": delta}))
}

// writeToolCall 累积并下发工具调用增量；首个增量开启对应 tool_use 块，后续增量写 input_json_delta
func (c *messagesCodec) writeToolCall(buf *bytes.Buffer, tc chatToolCall) {
	st, ok := c.tools[tc.Index]
	if !ok {
		st = &messagesToolState{}
		c.tools[tc.Index] = st
	}
	if tc.ID != "" {
		st.id = tc.ID
	}
	if tc.Function.Name != "" {
		st.name = tc.Function.Name
	}
	if !st.started {
		st.started = true
		st.index = c.openContentBlock(buf, map[string]any{
			"type":  "tool_use",
			"id":    toolUseID(st.id),
			"name":  st.name,
			"input": map[string]any{},
		})
	}
	if args := tc.Function.Arguments; args != "" {
		buf.Write(c.contentBlockDelta(st.index, map[string]any{"type": "input_json_delta", "partial_json": args}))
	}
}

// openContentBlock 关闭当前已开启的块并开启新块，返回新块序号
func (c *messagesCodec) openContentBlock(buf *bytes.Buffer, block map[string]any) int {
	c.closeContentBlock(buf)
	index := c.nextIndex
	c.nextIndex++
	c.openIndex = index
	buf.Write(c.contentBlockStart(index, block))
	return index
}

// closeContentBlock 关闭当前已开启的块；无开启块时不做任何事
func (c *messagesCodec) closeContentBlock(buf *bytes.Buffer) {
	if c.openIndex < 0 {
		return
	}
	buf.Write(c.contentBlockStop(c.openIndex))
	c.openIndex = -1
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

// KeepaliveEvent 空闲保活：Anthropic 原生 ping 事件（SDK 识别且忽略）
func (c *messagesCodec) KeepaliveEvent() []byte {
	return []byte("event: ping\ndata: {\"type\": \"ping\"}\n\n")
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
			"usage":         anthropicUsage(c.usage),
		},
	})
	return sseEvent("message_start", b)
}

// contentBlockStart 事件负载（index 为 content block 序号，block 为块初始内容）
func (c *messagesCodec) contentBlockStart(index int, block map[string]any) []byte {
	b, _ := jsonx.Marshal(map[string]any{
		"type":          "content_block_start",
		"index":         index,
		"content_block": block,
	})
	return sseEvent("content_block_start", b)
}

// contentBlockDelta 事件负载（delta 为 text_delta 或 input_json_delta）
func (c *messagesCodec) contentBlockDelta(index int, delta map[string]any) []byte {
	b, _ := jsonx.Marshal(map[string]any{
		"type":  "content_block_delta",
		"index": index,
		"delta": delta,
	})
	return sseEvent("content_block_delta", b)
}

// contentBlockStop 事件负载
func (c *messagesCodec) contentBlockStop(index int) []byte {
	b, _ := jsonx.Marshal(map[string]any{
		"type":  "content_block_stop",
		"index": index,
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
// Anthropic 的 input_tokens 不含缓存读写，故需从总输入中扣除；缓存读写单独上报
func anthropicUsage(u *chatUsage) map[string]any {
	if u == nil {
		return map[string]any{"input_tokens": 0, "output_tokens": 0}
	}
	var cached, creation int64
	if u.PromptTokensDetails != nil {
		cached = u.PromptTokensDetails.CachedTokens
		creation = u.PromptTokensDetails.CacheCreationTokens
	}
	input := u.PromptTokens - cached - creation
	if input < 0 {
		input = 0
	}
	out := map[string]any{
		"input_tokens":  input,
		"output_tokens": u.CompletionTokens,
	}
	if cached > 0 {
		out["cache_read_input_tokens"] = cached
	}
	if creation > 0 {
		out["cache_creation_input_tokens"] = creation
	}
	return out
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
