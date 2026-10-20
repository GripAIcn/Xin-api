package inbound

import (
	"encoding/json"
	"strings"
	"testing"
)

// Anthropic Messages 归一化断言结构
type messagesNormOut struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	MaxTokens     int      `json:"max_tokens"`
	Stream        bool     `json:"stream"`
	Stop          []string `json:"stop"`
	StreamOptions struct {
		IncludeUsage bool `json:"include_usage"`
	} `json:"stream_options"`
}

func TestMessagesNormalizeRequest(t *testing.T) {
	in := []byte(`{
		"model": "claude-3-5-sonnet",
		"system": [{"type":"text","text":"be nice"}],
		"messages": [{"role":"user","content":[{"type":"text","text":"hi"}]}],
		"stop_sequences": ["END"],
		"stream": true
	}`)
	c := &messagesCodec{}
	out, err := c.NormalizeRequest(in, NormalizeOptions{})
	if err != nil {
		t.Fatalf("NormalizeRequest error: %v", err)
	}
	var got messagesNormOut
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if got.Model != "claude-3-5-sonnet" {
		t.Errorf("model = %q, want %q", got.Model, "claude-3-5-sonnet")
	}
	if len(got.Messages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(got.Messages))
	}
	if got.Messages[0].Role != "system" || got.Messages[0].Content != "be nice" {
		t.Errorf("messages[0] = %+v, want system/be nice", got.Messages[0])
	}
	if got.Messages[1].Role != "user" || got.Messages[1].Content != "hi" {
		t.Errorf("messages[1] = %+v, want user/hi", got.Messages[1])
	}
	if got.MaxTokens != 4096 {
		t.Errorf("max_tokens = %d, want 4096", got.MaxTokens)
	}
	if !got.Stream {
		t.Errorf("stream = false, want true")
	}
	if len(got.Stop) != 1 || got.Stop[0] != "END" {
		t.Errorf("stop = %v, want [END]", got.Stop)
	}
	if !got.StreamOptions.IncludeUsage {
		t.Errorf("stream_options.include_usage = false, want true")
	}
}

func TestMessagesNormalizeMissingModel(t *testing.T) {
	in := []byte(`{"messages":[{"role":"user","content":"hi"}],"stream":false}`)
	c := &messagesCodec{}
	if _, err := c.NormalizeRequest(in, NormalizeOptions{}); err == nil {
		t.Fatalf("expected error for missing model, got nil")
	}
}

func TestMessagesNormalizeSystemString(t *testing.T) {
	in := []byte(`{"model":"claude","system":"be nice","messages":[{"role":"user","content":"hi"}]}`)
	c := &messagesCodec{}
	out, err := c.NormalizeRequest(in, NormalizeOptions{})
	if err != nil {
		t.Fatalf("NormalizeRequest error: %v", err)
	}
	var got messagesNormOut
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("len(messages) = %d, want 2", len(got.Messages))
	}
	if got.Messages[0].Role != "system" || got.Messages[0].Content != "be nice" {
		t.Errorf("messages[0] = %+v, want system/be nice", got.Messages[0])
	}
}

func TestMessagesNormalizeImageInput(t *testing.T) {
	in := []byte(`{
		"model":"claude-3-5-sonnet",
		"messages":[{"role":"user","content":[
			{"type":"text","text":"what is <this>?"},
			{"type":"image","source":{"type":"base64","media_type":"image/png","data":"aGVsbG8="}},
			{"type":"image","source":{"type":"url","url":"https://example.com/b.jpg"}}
		]}]
	}`)
	c := &messagesCodec{}
	out, err := c.NormalizeRequest(in, NormalizeOptions{})
	if err != nil {
		t.Fatalf("NormalizeRequest error: %v", err)
	}
	if strings.Contains(string(out), `\u003c`) {
		t.Errorf("output lost raw HTML: %s", string(out))
	}

	var got struct {
		Messages []struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if len(got.Messages) != 1 {
		t.Fatalf("len(messages) = %d, want 1", len(got.Messages))
	}

	var parts []struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		ImageURL struct {
			URL string `json:"url"`
		} `json:"image_url"`
	}
	if err := json.Unmarshal(got.Messages[0].Content, &parts); err != nil {
		t.Fatalf("content is not an array: %v", err)
	}
	if len(parts) != 3 {
		t.Fatalf("len(content) = %d, want 3", len(parts))
	}
	if parts[0].Type != "text" || parts[0].Text != "what is <this>?" {
		t.Errorf("parts[0] = %+v, want text", parts[0])
	}
	if parts[1].Type != "image_url" || parts[1].ImageURL.URL != "data:image/png;base64,aGVsbG8=" {
		t.Errorf("parts[1] = %+v, want base64 data URI", parts[1])
	}
	if parts[2].Type != "image_url" || parts[2].ImageURL.URL != "https://example.com/b.jpg" {
		t.Errorf("parts[2] = %+v, want image url", parts[2])
	}

	// 纯文本内容仍保持字符串形式
	in = []byte(`{"model":"claude","messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`)
	out, err = c.NormalizeRequest(in, NormalizeOptions{})
	if err != nil {
		t.Fatalf("NormalizeRequest error: %v", err)
	}
	var textOnly messagesNormOut
	if err := json.Unmarshal(out, &textOnly); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if textOnly.Messages[0].Content != "hi" {
		t.Errorf("content = %q, want string hi", textOnly.Messages[0].Content)
	}
}

type messagesEncOut struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
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

func TestMessagesEncodeResponse(t *testing.T) {
	in := []byte(`{
		"id": "chatcmpl-abc",
		"created": 1,
		"model": "claude-3-5-sonnet",
		"choices": [{"index":0,"message":{"role":"assistant","content":"hello"},"finish_reason":"length"}],
		"usage": {"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}
	}`)
	c := &messagesCodec{}
	out, err := c.EncodeResponse(in)
	if err != nil {
		t.Fatalf("EncodeResponse error: %v", err)
	}
	var got messagesEncOut
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if got.ID != "msg_abc" {
		t.Errorf("id = %q, want %q", got.ID, "msg_abc")
	}
	if got.Type != "message" {
		t.Errorf("type = %q, want %q", got.Type, "message")
	}
	if got.Role != "assistant" {
		t.Errorf("role = %q, want %q", got.Role, "assistant")
	}
	if len(got.Content) != 1 || got.Content[0].Type != "text" || got.Content[0].Text != "hello" {
		t.Errorf("content = %+v, want [{text hello}]", got.Content)
	}
	if got.StopReason != "max_tokens" {
		t.Errorf("stop_reason = %q, want %q", got.StopReason, "max_tokens")
	}
	if got.Usage.InputTokens != 10 {
		t.Errorf("usage.input_tokens = %d, want 10", got.Usage.InputTokens)
	}
	if got.Usage.OutputTokens != 5 {
		t.Errorf("usage.output_tokens = %d, want 5", got.Usage.OutputTokens)
	}
}

func TestMessagesEncodeStream(t *testing.T) {
	blocks := []string{
		"data: {\"id\":\"chatcmpl-abc\",\"model\":\"claude-3-5-sonnet\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"}}]}\n\n",
		"data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"He\"}}]}\n\n",
		"data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"llo\"}}]}\n\n",
		"data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n",
		"data: {\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":2,\"total_tokens\":12}}\n\n",
		"data: [DONE]\n\n",
	}
	c := &messagesCodec{}
	var sb strings.Builder
	for _, blk := range blocks {
		out, err := c.EncodeStreamChunk([]byte(blk))
		if err != nil {
			t.Fatalf("EncodeStreamChunk error: %v", err)
		}
		sb.Write(out)
	}
	stream := sb.String()

	events := []string{
		"event: message_start",
		"event: content_block_start",
		"event: content_block_delta",
		"event: content_block_delta",
		"event: content_block_stop",
		"event: message_delta",
		"event: message_stop",
	}
	last := 0
	for _, ev := range events {
		idx := strings.Index(stream[last:], ev)
		if idx < 0 {
			t.Fatalf("event %q not found in order after position %d; stream=\n%s", ev, last, stream)
		}
		last += idx + len(ev)
	}
	if !strings.Contains(stream, `"stop_reason":"end_turn"`) {
		t.Errorf("missing stop_reason end_turn; stream=\n%s", stream)
	}
	if !strings.Contains(stream, `"output_tokens":2`) {
		t.Errorf("missing output_tokens 2; stream=\n%s", stream)
	}
}

// sseEventPart 解析后的 Anthropic SSE 事件（事件名 + data JSON）
type sseEventPart struct {
	Event string
	Data  map[string]any
}

// parseMessagesSSE 按出现顺序解析 Anthropic SSE 事件流
func parseMessagesSSE(t *testing.T, stream string) []sseEventPart {
	t.Helper()
	var events []sseEventPart
	for _, block := range strings.Split(stream, "\n\n") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		part := sseEventPart{}
		for _, line := range strings.Split(block, "\n") {
			switch {
			case strings.HasPrefix(line, "event: "):
				part.Event = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				data := strings.TrimPrefix(line, "data: ")
				if err := json.Unmarshal([]byte(data), &part.Data); err != nil {
					t.Fatalf("unmarshal SSE data %q: %v", data, err)
				}
			}
		}
		events = append(events, part)
	}
	return events
}

func TestMessagesNormalizeTools(t *testing.T) {
	in := []byte(`{
		"model":"claude-3-5-sonnet",
		"messages":[
			{"role":"user","content":"weather?"},
			{"role":"assistant","content":[
				{"type":"text","text":"let me check"},
				{"type":"tool_use","id":"toolu_1","name":"get_weather","input":{"location":"Paris"}}
			]},
			{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":[{"type":"text","text":"sunny"}]}]}
		],
		"tools":[{"name":"get_weather","description":"查天气","input_schema":{"type":"object","properties":{"location":{"type":"string"}}}}],
		"tool_choice":{"type":"auto","disable_parallel_tool_use":true}
	}`)

	c := &messagesCodec{}
	out, err := c.NormalizeRequest(in, NormalizeOptions{})
	if err != nil {
		t.Fatalf("NormalizeRequest error: %v", err)
	}

	var got struct {
		Messages []struct {
			Role    string `json:"role"`
			Content any    `json:"content"`
			ToolCalls
		} `json:"messages"`
		Tools []struct {
			Type     string `json:"type"`
			Function struct {
				Name       string          `json:"name"`
				Parameters json.RawMessage `json:"parameters"`
			} `json:"function"`
		} `json:"tools"`
		ToolChoice        string `json:"tool_choice"`
		ParallelToolCalls *bool  `json:"parallel_tool_calls"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	// user → assistant(text+tool_calls) → tool
	if len(got.Messages) != 3 {
		t.Fatalf("len(messages) = %d, want 3; out=%s", len(got.Messages), string(out))
	}
	if got.Messages[0].Role != "user" || got.Messages[0].Content != "weather?" {
		t.Errorf("messages[0] = %+v, want user/weather?", got.Messages[0])
	}
	if got.Messages[1].Role != "assistant" || got.Messages[1].Content != "let me check" {
		t.Errorf("messages[1] = %+v, want assistant/let me check", got.Messages[1])
	}
	calls := got.Messages[1].ToolCalls.List
	if len(calls) != 1 {
		t.Fatalf("assistant tool_calls = %+v, want 1", calls)
	}
	if calls[0].ID != "toolu_1" || calls[0].Type != "function" || calls[0].Function.Name != "get_weather" {
		t.Errorf("tool_call = %+v, want toolu_1/function/get_weather", calls[0])
	}
	if calls[0].Function.Arguments != `{"location":"Paris"}` {
		t.Errorf("arguments = %q, want %q", calls[0].Function.Arguments, `{"location":"Paris"}`)
	}
	if got.Messages[2].Role != "tool" || got.Messages[2].ToolCallID != "toolu_1" || got.Messages[2].Content != "sunny" {
		t.Errorf("messages[2] = %+v, want tool/toolu_1/sunny", got.Messages[2])
	}

	if len(got.Tools) != 1 || got.Tools[0].Type != "function" || got.Tools[0].Function.Name != "get_weather" {
		t.Fatalf("tools = %+v, want 1 function tool get_weather", got.Tools)
	}
	if !strings.Contains(string(got.Tools[0].Function.Parameters), `"location"`) {
		t.Errorf("parameters = %s, want location schema", got.Tools[0].Function.Parameters)
	}
	if got.ToolChoice != "auto" {
		t.Errorf("tool_choice = %q, want auto", got.ToolChoice)
	}
	if got.ParallelToolCalls == nil || *got.ParallelToolCalls {
		t.Errorf("parallel_tool_calls = %v, want false", got.ParallelToolCalls)
	}

	// tool_choice=any → required
	out, err = c.NormalizeRequest([]byte(`{"model":"claude","messages":[{"role":"user","content":"hi"}],"tool_choice":{"type":"any"}}`), NormalizeOptions{})
	if err != nil {
		t.Fatalf("NormalizeRequest error: %v", err)
	}
	var anyChoice struct {
		ToolChoice string `json:"tool_choice"`
	}
	if err := json.Unmarshal(out, &anyChoice); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if anyChoice.ToolChoice != "required" {
		t.Errorf("tool_choice = %q, want required", anyChoice.ToolChoice)
	}
}

func TestMessagesEncodeResponseWithToolCalls(t *testing.T) {
	in := []byte(`{
		"id":"chatcmpl-abc","model":"claude-3-5-sonnet",
		"choices":[{"index":0,"message":{"role":"assistant","content":"","tool_calls":[
			{"id":"call_1","type":"function","function":{"name":"get_weather","arguments":"{\"location\":\"Paris\"}"}}
		]},"finish_reason":"tool_calls"}],
		"usage":{"prompt_tokens":1,"completion_tokens":2,"total_tokens":3}
	}`)

	c := &messagesCodec{}
	out, err := c.EncodeResponse(in)
	if err != nil {
		t.Fatalf("EncodeResponse error: %v", err)
	}
	var got struct {
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
		StopReason string `json:"stop_reason"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	// 文本为空时省略空文本块，仅保留 tool_use 块
	if len(got.Content) != 1 {
		t.Fatalf("content = %+v, want 1 tool_use block", got.Content)
	}
	if got.Content[0].Type != "tool_use" || got.Content[0].ID != "call_1" || got.Content[0].Name != "get_weather" {
		t.Errorf("content[0] = %+v, want tool_use call_1/get_weather", got.Content[0])
	}
	var input map[string]string
	if err := json.Unmarshal(got.Content[0].Input, &input); err != nil {
		t.Fatalf("input is not an object: %v", err)
	}
	if input["location"] != "Paris" {
		t.Errorf("input = %+v, want location Paris", input)
	}
	if got.StopReason != "tool_use" {
		t.Errorf("stop_reason = %q, want tool_use", got.StopReason)
	}
}

func TestMessagesEncodeStreamWithToolCalls(t *testing.T) {
	blocks := []string{
		"data: {\"id\":\"chatcmpl-abc\",\"model\":\"claude-3-5-sonnet\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"}}]}\n\n",
		"data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"checking\"}}]}\n\n",
		"data: {\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"get_weather\",\"arguments\":\"\"}}]}}]}\n\n",
		"data: {\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"{\\\"location\\\":\"}}]}}]}\n\n",
		"data: {\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"\\\"Paris\\\"}\"}}]}}]}\n\n",
		"data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n",
		"data: [DONE]\n\n",
	}

	c := &messagesCodec{}
	var sb strings.Builder
	for _, blk := range blocks {
		out, err := c.EncodeStreamChunk([]byte(blk))
		if err != nil {
			t.Fatalf("EncodeStreamChunk error: %v", err)
		}
		sb.Write(out)
	}

	events := parseMessagesSSE(t, sb.String())
	var names []string
	for _, ev := range events {
		names = append(names, ev.Event)
	}
	want := []string{
		"message_start",
		"content_block_start", // index 0：文本块
		"content_block_delta",
		"content_block_stop",
		"content_block_start", // index 1：工具块
		"content_block_delta",
		"content_block_delta",
		"content_block_stop",
		"message_delta",
		"message_stop",
	}
	if len(names) != len(want) {
		t.Fatalf("events = %v, want %v; stream=\n%s", names, want, sb.String())
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("event[%d] = %q, want %q; stream=\n%s", i, names[i], want[i], sb.String())
		}
	}

	// 文本块 index 0，工具块 index 1
	if events[1].Data["index"] != float64(0) {
		t.Errorf("text block index = %v, want 0", events[1].Data["index"])
	}
	toolStart := events[4].Data
	if toolStart["index"] != float64(1) {
		t.Errorf("tool block index = %v, want 1", toolStart["index"])
	}
	block, _ := toolStart["content_block"].(map[string]any)
	if block["type"] != "tool_use" || block["id"] != "call_1" || block["name"] != "get_weather" {
		t.Errorf("tool content_block = %+v, want tool_use call_1/get_weather", block)
	}

	// 参数增量以 input_json_delta 下发，拼接后为完整 arguments
	var partials []string
	for _, ev := range events {
		if ev.Event != "content_block_delta" {
			continue
		}
		delta, _ := ev.Data["delta"].(map[string]any)
		if delta["type"] == "input_json_delta" {
			partials = append(partials, delta["partial_json"].(string))
		}
	}
	if joined := strings.Join(partials, ""); joined != `{"location":"Paris"}` {
		t.Errorf("input_json_delta joined = %q, want %q", joined, `{"location":"Paris"}`)
	}

	delta, _ := events[8].Data["delta"].(map[string]any)
	if delta["stop_reason"] != "tool_use" {
		t.Errorf("stop_reason = %v, want tool_use", delta["stop_reason"])
	}
}

// ToolCalls 规范化输出中的 tool_calls 断言结构（内嵌以避免重复声明）
type ToolCalls struct {
	List []struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	} `json:"tool_calls"`
	ToolCallID string `json:"tool_call_id"`
}
