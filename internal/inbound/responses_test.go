package inbound

import (
	"encoding/json"
	"strings"
	"testing"
)

// responsesNormOut Responses 归一化断言结构
type responsesNormOut struct {
	Model    string `json:"model"`
	Messages []struct {
		Role             string          `json:"role"`
		Content          json.RawMessage `json:"content"`
		ReasoningContent string          `json:"reasoning_content"`
		ToolCallID       string          `json:"tool_call_id"`
		ToolCalls        []struct {
			ID       string `json:"id"`
			Type     string `json:"type"`
			Function struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
		} `json:"tool_calls"`
	} `json:"messages"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	TopP        float64 `json:"top_p"`
	Stream      bool    `json:"stream"`
	Tools       []struct {
		Type     string `json:"type"`
		Function struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			Parameters  json.RawMessage `json:"parameters"`
		} `json:"function"`
	} `json:"tools"`
	ToolChoice struct {
		Type     string `json:"type"`
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	} `json:"tool_choice"`
	ReasoningEffort string `json:"reasoning_effort"`
	ResponseFormat  struct {
		Type       string `json:"type"`
		JSONSchema struct {
			Name   string `json:"name"`
			Strict bool   `json:"strict"`
		} `json:"json_schema"`
	} `json:"response_format"`
	Metadata      map[string]string `json:"metadata"`
	StreamOptions struct {
		IncludeUsage bool `json:"include_usage"`
	} `json:"stream_options"`
}

func TestResponsesNormalizeRequest(t *testing.T) {
	in := []byte(`{
		"model": "gpt-4o",
		"instructions": [{"type":"input_text","text":"be <b>nice</b> & kind"}],
		"input": [
			{"type":"message","role":"developer","content":[{"type":"input_text","text":"rules <x>"}]},
			{"type":"message","role":"user","content":[
				{"type":"input_text","text":"see this"},
				{"type":"input_image","image_url":"https://example.com/a.png","detail":"high"}
			]},
			{"type":"reasoning","summary":[{"type":"summary_text","text":"we need code"}]},
			{"type":"function_call","call_id":"call_1","name":"run","arguments":"{\"a\":1}"},
			{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}]},
			{"type":"function_call_output","call_id":"call_1","output":"<result>&"}
		],
		"max_output_tokens": 256,
		"temperature": 0.2,
		"top_p": 0.9,
		"stream": true,
		"tools": [{"type":"function","name":"run","description":"run cmd","parameters":{"type":"object"}}],
		"tool_choice": {"type":"function","name":"run"},
		"parallel_tool_calls": false,
		"reasoning": {"effort": "high"},
		"text": {"format": {"type":"json_schema","name":"out","schema":{"type":"object"},"strict":true}},
		"metadata": {"k":"v"}
	}`)

	c := &responsesCodec{}
	out, err := c.NormalizeRequest(in, NormalizeOptions{})
	if err != nil {
		t.Fatalf("NormalizeRequest error: %v", err)
	}
	var got responsesNormOut
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	if got.Model != "gpt-4o" {
		t.Errorf("model = %q, want gpt-4o", got.Model)
	}
	if len(got.Messages) != 6 {
		t.Fatalf("len(messages) = %d, want 6", len(got.Messages))
	}

	var sys0 string
	if err := json.Unmarshal(got.Messages[0].Content, &sys0); err != nil {
		t.Fatalf("messages[0].content not string: %v", err)
	}
	if got.Messages[0].Role != "system" || sys0 != "be <b>nice</b> & kind" {
		t.Errorf("messages[0] = %s/%q, want system/instructions", got.Messages[0].Role, sys0)
	}

	var sys1 string
	if err := json.Unmarshal(got.Messages[1].Content, &sys1); err != nil {
		t.Fatalf("messages[1].content not string: %v", err)
	}
	if got.Messages[1].Role != "system" || sys1 != "rules <x>" {
		t.Errorf("messages[1] = %s/%q, want system/developer", got.Messages[1].Role, sys1)
	}

	var parts []struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		ImageURL struct {
			URL    string `json:"url"`
			Detail string `json:"detail"`
		} `json:"image_url"`
	}
	if err := json.Unmarshal(got.Messages[2].Content, &parts); err != nil {
		t.Fatalf("messages[2].content not array: %v", err)
	}
	if got.Messages[2].Role != "user" || len(parts) != 2 {
		t.Fatalf("messages[2] = %s with %d parts, want user with 2 parts", got.Messages[2].Role, len(parts))
	}
	if parts[0].Type != "text" || parts[0].Text != "see this" {
		t.Errorf("parts[0] = %+v, want text/see this", parts[0])
	}
	if parts[1].Type != "image_url" || parts[1].ImageURL.URL != "https://example.com/a.png" || parts[1].ImageURL.Detail != "high" {
		t.Errorf("parts[1] = %+v, want image_url with detail high", parts[1])
	}

	if got.Messages[3].Role != "assistant" || got.Messages[3].ReasoningContent != "we need code" {
		t.Errorf("messages[3] = %+v, want assistant with reasoning", got.Messages[3])
	}
	if len(got.Messages[3].ToolCalls) != 1 || got.Messages[3].ToolCalls[0].ID != "call_1" ||
		got.Messages[3].ToolCalls[0].Function.Name != "run" || got.Messages[3].ToolCalls[0].Function.Arguments != `{"a":1}` {
		t.Errorf("messages[3].tool_calls = %+v, want call_1/run", got.Messages[3].ToolCalls)
	}

	var asst string
	if err := json.Unmarshal(got.Messages[4].Content, &asst); err != nil {
		t.Fatalf("messages[4].content not string: %v", err)
	}
	if got.Messages[4].Role != "assistant" || asst != "ok" {
		t.Errorf("messages[4] = %s/%q, want assistant/ok", got.Messages[4].Role, asst)
	}

	var toolOut string
	if err := json.Unmarshal(got.Messages[5].Content, &toolOut); err != nil {
		t.Fatalf("messages[5].content not string: %v", err)
	}
	if got.Messages[5].Role != "tool" || got.Messages[5].ToolCallID != "call_1" || toolOut != "<result>&" {
		t.Errorf("messages[5] = %s/%s/%q, want tool/call_1/<result>&", got.Messages[5].Role, got.Messages[5].ToolCallID, toolOut)
	}

	if got.MaxTokens != 256 || got.Temperature != 0.2 || got.TopP != 0.9 || !got.Stream {
		t.Errorf("sampling = max %d temp %v top_p %v stream %v", got.MaxTokens, got.Temperature, got.TopP, got.Stream)
	}
	if len(got.Tools) != 1 || got.Tools[0].Function.Name != "run" || got.Tools[0].Function.Description != "run cmd" {
		t.Errorf("tools = %+v, want run", got.Tools)
	}
	if got.ToolChoice.Type != "function" || got.ToolChoice.Function.Name != "run" {
		t.Errorf("tool_choice = %+v, want function/run", got.ToolChoice)
	}
	if got.ReasoningEffort != "high" {
		t.Errorf("reasoning_effort = %q, want high", got.ReasoningEffort)
	}
	if got.ResponseFormat.Type != "json_schema" || got.ResponseFormat.JSONSchema.Name != "out" || !got.ResponseFormat.JSONSchema.Strict {
		t.Errorf("response_format = %+v, want json_schema/out/strict", got.ResponseFormat)
	}
	if got.Metadata["k"] != "v" {
		t.Errorf("metadata = %v, want k=v", got.Metadata)
	}
	if !got.StreamOptions.IncludeUsage {
		t.Errorf("stream_options.include_usage = false, want true")
	}

	// HTML 字面量必须原样保留
	if !strings.Contains(string(out), `"<result>&"`) {
		t.Errorf("output lost raw HTML: %s", string(out))
	}
	if strings.Contains(string(out), `\u003c`) || strings.Contains(string(out), `\u0026`) {
		t.Errorf("output contains HTML-escaped sequences: %s", string(out))
	}
}

// responsesOut Responses 非流式响应断言结构
type responsesOut struct {
	ID                string `json:"id"`
	Object            string `json:"object"`
	CreatedAt         int64  `json:"created_at"`
	Model             string `json:"model"`
	Status            string `json:"status"`
	IncompleteDetails struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
	Output []struct {
		ID        string `json:"id"`
		Type      string `json:"type"`
		Status    string `json:"status"`
		Role      string `json:"role"`
		CallID    string `json:"call_id"`
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
		Content   []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Summary []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"summary"`
	} `json:"output"`
	Usage struct {
		InputTokens        int64 `json:"input_tokens"`
		OutputTokens       int64 `json:"output_tokens"`
		TotalTokens        int64 `json:"total_tokens"`
		InputTokensDetails struct {
			CachedTokens int64 `json:"cached_tokens"`
		} `json:"input_tokens_details"`
		OutputTokensDetails struct {
			ReasoningTokens int64 `json:"reasoning_tokens"`
		} `json:"output_tokens_details"`
	} `json:"usage"`
}

func TestResponsesEncodeResponse(t *testing.T) {
	in := []byte(`{
		"id":"chatcmpl-abc",
		"created":1700000000,
		"model":"gpt-4o",
		"choices":[{
			"index":0,
			"message":{
				"role":"assistant",
				"content":"<b>hi</b> & bye",
				"reasoning_content":"think",
				"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"run","arguments":"{\"x\":1}"}}]
			},
			"finish_reason":"length"
		}],
		"usage":{
			"prompt_tokens":10,
			"completion_tokens":5,
			"total_tokens":15,
			"prompt_tokens_details":{"cached_tokens":4},
			"completion_tokens_details":{"reasoning_tokens":2}
		}
	}`)

	c := &responsesCodec{}
	out, err := c.EncodeResponse(in)
	if err != nil {
		t.Fatalf("EncodeResponse error: %v", err)
	}
	var got responsesOut
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}

	if got.ID != "resp_abc" || got.Object != "response" || got.Model != "gpt-4o" {
		t.Errorf("response head = %q/%q/%q", got.ID, got.Object, got.Model)
	}
	if got.CreatedAt != 1700000000 {
		t.Errorf("created_at = %d, want 1700000000", got.CreatedAt)
	}
	if got.Status != "incomplete" || got.IncompleteDetails.Reason != "max_output_tokens" {
		t.Errorf("status = %q reason = %q, want incomplete/max_output_tokens", got.Status, got.IncompleteDetails.Reason)
	}
	if len(got.Output) != 3 {
		t.Fatalf("len(output) = %d, want 3", len(got.Output))
	}
	if got.Output[0].Type != "reasoning" || len(got.Output[0].Summary) != 1 || got.Output[0].Summary[0].Text != "think" {
		t.Errorf("output[0] = %+v, want reasoning/think", got.Output[0])
	}
	if got.Output[1].Type != "message" || got.Output[1].Role != "assistant" ||
		len(got.Output[1].Content) != 1 || got.Output[1].Content[0].Text != "<b>hi</b> & bye" {
		t.Errorf("output[1] = %+v, want message with raw HTML", got.Output[1])
	}
	if got.Output[2].Type != "function_call" || got.Output[2].CallID != "call_1" ||
		got.Output[2].Name != "run" || got.Output[2].Arguments != `{"x":1}` {
		t.Errorf("output[2] = %+v, want function_call/call_1/run", got.Output[2])
	}
	if got.Usage.InputTokens != 10 || got.Usage.OutputTokens != 5 || got.Usage.TotalTokens != 15 {
		t.Errorf("usage = %+v, want 10/5/15", got.Usage)
	}
	if got.Usage.InputTokensDetails.CachedTokens != 4 || got.Usage.OutputTokensDetails.ReasoningTokens != 2 {
		t.Errorf("usage details = %+v, want cached 4/reasoning 2", got.Usage)
	}

	if !strings.Contains(string(out), "<b>hi</b> & bye") {
		t.Errorf("output lost raw HTML: %s", string(out))
	}
	if strings.Contains(string(out), `\u003c`) || strings.Contains(string(out), `\u0026`) {
		t.Errorf("output contains HTML-escaped sequences: %s", string(out))
	}
}

// parsedSSE 解析后的单个 SSE 事件
type parsedSSE struct {
	Event string
	Data  map[string]any
}

// parseSSEEvents 解析 SSE 输出为事件列表；忽略无法解析的块
func parseSSEEvents(t *testing.T, raw []byte) []parsedSSE {
	t.Helper()
	var events []parsedSSE
	for _, block := range strings.Split(string(raw), "\n\n") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		var ev parsedSSE
		for _, line := range strings.Split(block, "\n") {
			switch {
			case strings.HasPrefix(line, "event: "):
				ev.Event = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev.Data); err != nil {
					t.Fatalf("unmarshal SSE data failed: %v (%s)", err, line)
				}
			}
		}
		events = append(events, ev)
	}
	return events
}

func TestResponsesStreamEvents(t *testing.T) {
	chunks := []string{
		`data: {"id":"chatcmpl-abc","created":1700000000,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}` + "\n\n",
		`data: {"choices":[{"index":0,"delta":{"reasoning_content":"think"},"finish_reason":null}]}` + "\n\n",
		`data: {"choices":[{"index":0,"delta":{"content":"<b>&"},"finish_reason":null}]}` + "\n\n",
		`data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"run","arguments":"{\"a\""}}]},"finish_reason":null}]}` + "\n\n",
		`data: {"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":":1}"}}]},"finish_reason":null}]}` + "\n\n",
		`data: {"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}` + "\n\n",
		`data: {"choices":[],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15,"prompt_tokens_details":{"cached_tokens":4},"completion_tokens_details":{"reasoning_tokens":2}}}` + "\n\n",
		`data: [DONE]` + "\n\n",
	}

	c := &responsesCodec{}
	var raw []byte
	for _, chunk := range chunks {
		out, err := c.EncodeStreamChunk([]byte(chunk))
		if err != nil {
			t.Fatalf("EncodeStreamChunk error: %v", err)
		}
		raw = append(raw, out...)
	}

	events := parseSSEEvents(t, raw)
	if len(events) == 0 {
		t.Fatal("no events produced")
	}
	if events[0].Event != "response.created" || events[1].Event != "response.in_progress" {
		t.Fatalf("first events = %q,%q, want response.created,response.in_progress", events[0].Event, events[1].Event)
	}

	// sequence_number 必须从 0 开始单调递增
	for i, ev := range events {
		seq, ok := ev.Data["sequence_number"].(float64)
		if !ok {
			t.Fatalf("event %d (%s) missing sequence_number", i, ev.Event)
		}
		if int(seq) != i {
			t.Errorf("event %d (%s) sequence_number = %v, want %d", i, ev.Event, seq, i)
		}
		if ev.Data["type"] != ev.Event {
			t.Errorf("event %d type = %v, want %s", i, ev.Data["type"], ev.Event)
		}
	}

	var textDelta string
	var argDeltas []string
	var argDone string
	for _, ev := range events {
		switch ev.Event {
		case "response.output_text.delta":
			textDelta, _ = ev.Data["delta"].(string)
		case "response.function_call_arguments.delta":
			d, _ := ev.Data["delta"].(string)
			argDeltas = append(argDeltas, d)
		case "response.function_call_arguments.done":
			argDone, _ = ev.Data["arguments"].(string)
		}
	}
	if textDelta != "<b>&" {
		t.Errorf("text delta = %q, want <b>&", textDelta)
	}
	if len(argDeltas) != 2 || strings.Join(argDeltas, "") != `{"a":1}` {
		t.Errorf("argument deltas = %v, want two fragments of {\"a\":1}", argDeltas)
	}
	if argDone != `{"a":1}` {
		t.Errorf("arguments done = %q, want {\"a\":1}", argDone)
	}

	last := events[len(events)-1]
	if last.Event != "response.completed" {
		t.Fatalf("last event = %q, want response.completed", last.Event)
	}
	resp, _ := last.Data["response"].(map[string]any)
	if resp == nil {
		t.Fatalf("response.completed missing response object")
	}
	if resp["status"] != "completed" || resp["id"] != "resp_abc" {
		t.Errorf("final response = %v/%v, want completed/resp_abc", resp["status"], resp["id"])
	}
	output, _ := resp["output"].([]any)
	if len(output) != 3 {
		t.Fatalf("len(output) = %d, want 3", len(output))
	}
	wantTypes := []string{"reasoning", "message", "function_call"}
	for i, want := range wantTypes {
		item, _ := output[i].(map[string]any)
		if item == nil || item["type"] != want {
			t.Errorf("output[%d] = %v, want type %s", i, output[i], want)
		}
	}
	usage, _ := resp["usage"].(map[string]any)
	if usage == nil {
		t.Fatalf("final response missing usage")
	}
	if usage["input_tokens"] != float64(10) || usage["output_tokens"] != float64(5) {
		t.Errorf("usage = %v, want input 10/output 5", usage)
	}
	details, _ := usage["input_tokens_details"].(map[string]any)
	if details == nil || details["cached_tokens"] != float64(4) {
		t.Errorf("input_tokens_details = %v, want cached 4", usage["input_tokens_details"])
	}

	// 流式文本同样不允许 HTML 转义
	if strings.Contains(string(raw), `\u003c`) || strings.Contains(string(raw), `\u0026`) {
		t.Errorf("stream contains HTML-escaped sequences: %s", string(raw))
	}
}

// TestResponsesNormalizePromptCacheKey prompt_cache_key 必须透传到规范 Chat 请求体（提升上游缓存路由命中率）
func TestResponsesNormalizePromptCacheKey(t *testing.T) {
	c := &responsesCodec{}

	out, err := c.NormalizeRequest([]byte(`{
		"model": "gpt-6-sol",
		"input": "hi",
		"prompt_cache_key": "conv-123"
	}`), NormalizeOptions{})
	if err != nil {
		t.Fatalf("NormalizeRequest error: %v", err)
	}
	var got struct {
		PromptCacheKey string `json:"prompt_cache_key"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if got.PromptCacheKey != "conv-123" {
		t.Errorf("prompt_cache_key = %q, want conv-123", got.PromptCacheKey)
	}

	// 未携带时不注入该字段
	out, err = c.NormalizeRequest([]byte(`{"model":"gpt-6-sol","input":"hi"}`), NormalizeOptions{})
	if err != nil {
		t.Fatalf("NormalizeRequest error: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if _, ok := raw["prompt_cache_key"]; ok {
		t.Errorf("prompt_cache_key should be absent when not provided")
	}
}
