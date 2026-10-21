package adapter

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAnthropicBuildUpstreamRequestImageInput(t *testing.T) {
	body := `{
		"model": "claude-3-5-sonnet",
		"messages": [
			{"role": "system", "content": "sys"},
			{"role": "user", "content": [
				{"type": "text", "text": "describe"},
				{"type": "image_url", "image_url": {"url": "data:image/png;base64,aGVsbG8="}},
				{"type": "image_url", "image_url": {"url": "https://example.com/a.jpg"}}
			]}
		]
	}`

	ad := NewAnthropicAdapter()
	req, err := ad.BuildUpstreamRequest([]byte(body), "", "key")
	if err != nil {
		t.Fatalf("BuildUpstreamRequest: %v", err)
	}

	var out struct {
		System   string `json:"system"`
		Messages []struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(req.Body, &out); err != nil {
		t.Fatalf("unmarshal upstream body: %v", err)
	}
	if out.System != "sys" {
		t.Fatalf("system = %q, want %q", out.System, "sys")
	}
	if len(out.Messages) != 1 || out.Messages[0].Role != "user" {
		t.Fatalf("messages = %+v", out.Messages)
	}

	var blocks []struct {
		Type   string `json:"type"`
		Text   string `json:"text"`
		Source struct {
			Type      string `json:"type"`
			MediaType string `json:"media_type"`
			Data      string `json:"data"`
			URL       string `json:"url"`
		} `json:"source"`
	}
	if err := json.Unmarshal(out.Messages[0].Content, &blocks); err != nil {
		t.Fatalf("content is not a block array: %v", err)
	}
	if len(blocks) != 3 {
		t.Fatalf("blocks = %+v, want 3", blocks)
	}
	if blocks[0].Type != "text" || blocks[0].Text != "describe" {
		t.Fatalf("text block = %+v", blocks[0])
	}
	if blocks[1].Type != "image" || blocks[1].Source.Type != "base64" ||
		blocks[1].Source.MediaType != "image/png" || blocks[1].Source.Data != "aGVsbG8=" {
		t.Fatalf("base64 image block = %+v", blocks[1])
	}
	if blocks[2].Type != "image" || blocks[2].Source.Type != "url" ||
		blocks[2].Source.URL != "https://example.com/a.jpg" {
		t.Fatalf("url image block = %+v", blocks[2])
	}
}

func TestAnthropicBuildUpstreamRequestPlainTextContent(t *testing.T) {
	body := `{"model":"claude-3-5-sonnet","messages":[{"role":"user","content":[{"type":"text","text":"hi"},{"type":"text","text":" there"}]}]}`

	ad := NewAnthropicAdapter()
	req, err := ad.BuildUpstreamRequest([]byte(body), "", "key")
	if err != nil {
		t.Fatalf("BuildUpstreamRequest: %v", err)
	}

	var out struct {
		Messages []struct {
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(req.Body, &out); err != nil {
		t.Fatalf("unmarshal upstream body: %v", err)
	}
	var text string
	if err := json.Unmarshal(out.Messages[0].Content, &text); err != nil {
		t.Fatalf("content should stay a string: %v", err)
	}
	if text != "hi there" {
		t.Fatalf("content = %q, want %q", text, "hi there")
	}
}

func TestAnthropicBuildUpstreamRequestTools(t *testing.T) {
	body := `{
		"model": "claude-3-5-sonnet",
		"parallel_tool_calls": false,
		"tools": [{
			"type": "function",
			"function": {
				"name": "get_weather",
				"description": "查询天气",
				"parameters": {"type": "object", "properties": {"location": {"type": "string"}}, "required": ["location"]}
			}
		}],
		"tool_choice": "auto",
		"messages": [
			{"role": "user", "content": "北京天气如何"},
			{"role": "assistant", "content": "稍等", "tool_calls": [{"id": "call_1", "type": "function", "function": {"name": "get_weather", "arguments": "{\"location\":\"Beijing\"}"}}]},
			{"role": "tool", "tool_call_id": "call_1", "content": "晴 25度"},
			{"role": "tool", "tool_call_id": "call_2", "content": "{\"temp\": 26}"}
		]
	}`

	ad := NewAnthropicAdapter()
	req, err := ad.BuildUpstreamRequest([]byte(body), "", "key")
	if err != nil {
		t.Fatalf("BuildUpstreamRequest: %v", err)
	}

	var out struct {
		Tools []struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			InputSchema json.RawMessage `json:"input_schema"`
		} `json:"tools"`
		ToolChoice struct {
			Type                   string `json:"type"`
			Name                   string `json:"name"`
			DisableParallelToolUse bool   `json:"disable_parallel_tool_use"`
		} `json:"tool_choice"`
		Messages []struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(req.Body, &out); err != nil {
		t.Fatalf("unmarshal upstream body: %v", err)
	}

	// tools → Anthropic tools（parameters → input_schema）
	if len(out.Tools) != 1 || out.Tools[0].Name != "get_weather" || out.Tools[0].Description != "查询天气" {
		t.Fatalf("tools = %+v", out.Tools)
	}
	var schema map[string]any
	if err := json.Unmarshal(out.Tools[0].InputSchema, &schema); err != nil {
		t.Fatalf("input_schema not object: %v", err)
	}
	if props, ok := schema["properties"].(map[string]any); !ok || len(props) != 1 {
		t.Fatalf("input_schema properties = %+v", schema["properties"])
	}

	// tool_choice auto + parallel_tool_calls=false → disable_parallel_tool_use
	if out.ToolChoice.Type != "auto" || !out.ToolChoice.DisableParallelToolUse {
		t.Fatalf("tool_choice = %+v", out.ToolChoice)
	}

	// user / assistant / 合并后的 tool_result user 消息，共 3 条
	if len(out.Messages) != 3 {
		t.Fatalf("len(messages) = %d, want 3: %+v", len(out.Messages), out.Messages)
	}
	if out.Messages[0].Role != "user" {
		t.Fatalf("messages[0].role = %q", out.Messages[0].Role)
	}

	// assistant：文本块 + tool_use 块
	var assistantBlocks []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text"`
		ID    string          `json:"id"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	}
	if out.Messages[1].Role != "assistant" {
		t.Fatalf("messages[1].role = %q", out.Messages[1].Role)
	}
	if err := json.Unmarshal(out.Messages[1].Content, &assistantBlocks); err != nil {
		t.Fatalf("assistant content not blocks: %v", err)
	}
	if len(assistantBlocks) != 2 {
		t.Fatalf("assistant blocks = %+v, want 2", assistantBlocks)
	}
	if assistantBlocks[0].Type != "text" || assistantBlocks[0].Text != "稍等" {
		t.Fatalf("assistant block[0] = %+v", assistantBlocks[0])
	}
	if assistantBlocks[1].Type != "tool_use" || assistantBlocks[1].Name != "get_weather" ||
		assistantBlocks[1].ID != "call_1" {
		t.Fatalf("assistant block[1] = %+v", assistantBlocks[1])
	}
	var input map[string]any
	if err := json.Unmarshal(assistantBlocks[1].Input, &input); err != nil {
		t.Fatalf("tool_use input not object: %v", err)
	}
	if input["location"] != "Beijing" {
		t.Fatalf("tool_use input = %+v", input)
	}

	// 连续两条 tool 消息合并为一条 user 消息，含两个 tool_result 块
	var resultBlocks []struct {
		Type      string `json:"type"`
		ToolUseID string `json:"tool_use_id"`
		Content   string `json:"content"`
	}
	if out.Messages[2].Role != "user" {
		t.Fatalf("messages[2].role = %q, want user", out.Messages[2].Role)
	}
	if err := json.Unmarshal(out.Messages[2].Content, &resultBlocks); err != nil {
		t.Fatalf("tool_result content not blocks: %v", err)
	}
	if len(resultBlocks) != 2 {
		t.Fatalf("tool_result blocks = %+v, want 2", resultBlocks)
	}
	if resultBlocks[0].Type != "tool_result" || resultBlocks[0].ToolUseID != "call_1" ||
		resultBlocks[0].Content != "晴 25度" {
		t.Fatalf("tool_result[0] = %+v", resultBlocks[0])
	}
	if resultBlocks[1].ToolUseID != "call_2" || resultBlocks[1].Content != `{"temp": 26}` {
		t.Fatalf("tool_result[1] = %+v", resultBlocks[1])
	}
}

func TestAnthropicConvertResponseToolUse(t *testing.T) {
	body := `{
		"id": "msg_1",
		"model": "claude-3-5-sonnet",
		"content": [
			{"type": "text", "text": "我来查询"},
			{"type": "tool_use", "id": "toolu_1", "name": "get_weather", "input": {"location": "Shanghai"}}
		],
		"stop_reason": "tool_use",
		"usage": {"input_tokens": 10, "output_tokens": 5}
	}`

	ad := NewAnthropicAdapter()
	out, err := ad.ConvertResponse([]byte(body), "claude-3-5-sonnet")
	if err != nil {
		t.Fatalf("ConvertResponse: %v", err)
	}

	var got struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(got.Choices) != 1 {
		t.Fatalf("choices = %+v", got.Choices)
	}
	if got.Choices[0].Message.Content != "我来查询" {
		t.Fatalf("content = %q", got.Choices[0].Message.Content)
	}
	calls := got.Choices[0].Message.ToolCalls
	if len(calls) != 1 {
		t.Fatalf("tool_calls = %+v, want 1", calls)
	}
	if calls[0].ID != "toolu_1" || calls[0].Type != "function" || calls[0].Function.Name != "get_weather" {
		t.Fatalf("tool_call = %+v", calls[0])
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(calls[0].Function.Arguments), &args); err != nil {
		t.Fatalf("arguments not JSON: %v", err)
	}
	if args["location"] != "Shanghai" {
		t.Fatalf("arguments = %+v", args)
	}
	if got.Choices[0].FinishReason != "tool_calls" {
		t.Fatalf("finish_reason = %q, want tool_calls", got.Choices[0].FinishReason)
	}
}

func TestAnthropicConvertStreamToolUse(t *testing.T) {
	ad := NewAnthropicAdapter()

	// 同一实例跨 chunk 调用，验证 content_block 序号 → tool_calls index 的映射状态
	events := []string{
		`{"type":"message_start","message":{"id":"msg_1"}}`,
		`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"查询中"}}`,
		`{"type":"content_block_stop","index":0}`,
		`{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_1","name":"get_weather"}}`,
		`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"location\":"}}`,
		`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"\"Paris\"}"}}`,
		`{"type":"content_block_stop","index":1}`,
		`{"type":"message_delta","delta":{"stop_reason":"tool_use"}}`,
		`{"type":"message_stop"}`,
	}

	var collected []string
	for _, ev := range events {
		out, err := ad.ConvertStreamChunk([]byte("event: x\ndata: "+ev+"\n\n"), "claude-3-5-sonnet")
		if err != nil {
			t.Fatalf("ConvertStreamChunk(%s): %v", ev, err)
		}
		collected = append(collected, string(out))
	}

	// 首个事件带 role assistant
	if !strings.Contains(collected[0], `"role":"assistant"`) {
		t.Fatalf("message_start chunk = %s", collected[0])
	}
	// content_block_start(tool_use) 下发 tool_calls 起始增量：index 0 / id / name
	if !strings.Contains(collected[4], `"index":0`) || !strings.Contains(collected[4], `"id":"toolu_1"`) ||
		!strings.Contains(collected[4], `"name":"get_weather"`) {
		t.Fatalf("tool_use content_block_start chunk = %s", collected[4])
	}
	// 两个 input_json_delta 拼装为完整 arguments
	if !strings.Contains(collected[5], `"arguments":"{\"location\":"`) {
		t.Fatalf("input_json_delta chunk[0] = %s", collected[5])
	}
	if !strings.Contains(collected[6], `"arguments":"\"Paris\"}"`) {
		t.Fatalf("input_json_delta chunk[1] = %s", collected[6])
	}
	// message_delta → finish_reason tool_calls
	if !strings.Contains(collected[8], `"finish_reason":"tool_calls"`) {
		t.Fatalf("message_delta chunk = %s", collected[8])
	}
	// message_stop → [DONE]
	if !strings.Contains(collected[9], "[DONE]") {
		t.Fatalf("message_stop chunk = %s", collected[9])
	}
}

// TestAnthropicRawPassthrough 验证 Messages 入站按原始请求体透传：
// cache_control / metadata 等协议字段不丢失，仅改写 model 并补充鉴权头
func TestAnthropicRawPassthrough(t *testing.T) {
	body := `{
		"model": "claude-sonnet-4",
		"max_tokens": 1024,
		"system": [
			{"type": "text", "text": "sys1"},
			{"type": "text", "text": "sys2", "cache_control": {"type": "ephemeral"}}
		],
		"metadata": {"user_id": "user-1"},
		"tools": [{
			"name": "read_file",
			"description": "读取文件",
			"input_schema": {"type": "object", "properties": {}},
			"cache_control": {"type": "ephemeral"}
		}],
		"messages": [
			{"role": "user", "content": [
				{"type": "text", "text": "hi", "cache_control": {"type": "ephemeral"}}
			]}
		]
	}`

	ad := &anthropicAdapter{}
	if !ad.SupportsRawPassthrough("messages") {
		t.Fatal("messages 入站应支持原样透传")
	}
	if ad.SupportsRawPassthrough("chat") {
		t.Fatal("chat 入站不应走原样透传")
	}

	req, err := ad.BuildUpstreamRequestFromRaw([]byte(body), "claude-upstream", "key",
		map[string]string{"anthropic-beta": "prompt-caching-2024-07-31"})
	if err != nil {
		t.Fatalf("BuildUpstreamRequestFromRaw: %v", err)
	}

	type cacheControl struct {
		Type string `json:"type"`
	}
	var out struct {
		Model  string `json:"model"`
		System []struct {
			CacheControl *cacheControl `json:"cache_control"`
		} `json:"system"`
		Metadata struct {
			UserID string `json:"user_id"`
		} `json:"metadata"`
		Tools []struct {
			CacheControl *cacheControl `json:"cache_control"`
		} `json:"tools"`
		Messages []struct {
			Content []struct {
				CacheControl *cacheControl `json:"cache_control"`
			} `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(req.Body, &out); err != nil {
		t.Fatalf("unmarshal upstream body: %v", err)
	}
	if out.Model != "claude-upstream" {
		t.Fatalf("model = %q, want %q", out.Model, "claude-upstream")
	}
	if len(out.System) != 2 || out.System[1].CacheControl == nil {
		t.Fatalf("system cache_control 丢失: %+v", out.System)
	}
	if out.Metadata.UserID != "user-1" {
		t.Fatalf("metadata 丢失: %+v", out.Metadata)
	}
	if len(out.Tools) != 1 || out.Tools[0].CacheControl == nil {
		t.Fatalf("tools cache_control 丢失: %+v", out.Tools)
	}
	if len(out.Messages) != 1 || len(out.Messages[0].Content) != 1 ||
		out.Messages[0].Content[0].CacheControl == nil {
		t.Fatalf("messages cache_control 丢失: %+v", out.Messages)
	}
	if req.Headers["x-api-key"] != "key" || req.Headers["anthropic-version"] != "2023-06-01" {
		t.Fatalf("鉴权头异常: %+v", req.Headers)
	}
	if req.Headers["anthropic-beta"] != "prompt-caching-2024-07-31" {
		t.Fatalf("anthropic-beta 未沿用: %+v", req.Headers)
	}
}

// TestAnthropicConvertResponseCacheUsage 验证缓存读写字段映射为 OpenAI usage：
// prompt_tokens 为总输入（含缓存），cached_tokens / cache_creation_tokens 进入明细
func TestAnthropicConvertResponseCacheUsage(t *testing.T) {
	body := `{
		"id": "msg_1",
		"model": "claude-3-5-sonnet",
		"content": [{"type": "text", "text": "hi"}],
		"stop_reason": "end_turn",
		"usage": {
			"input_tokens": 10,
			"output_tokens": 5,
			"cache_read_input_tokens": 100,
			"cache_creation_input_tokens": 50
		}
	}`

	ad := NewAnthropicAdapter()
	out, err := ad.ConvertResponse([]byte(body), "claude-3-5-sonnet")
	if err != nil {
		t.Fatalf("ConvertResponse: %v", err)
	}

	var got struct {
		Usage struct {
			PromptTokens        int64 `json:"prompt_tokens"`
			CompletionTokens    int64 `json:"completion_tokens"`
			TotalTokens         int64 `json:"total_tokens"`
			PromptTokensDetails *struct {
				CachedTokens        int64 `json:"cached_tokens"`
				CacheCreationTokens int64 `json:"cache_creation_tokens"`
			} `json:"prompt_tokens_details"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got.Usage.PromptTokens != 160 || got.Usage.CompletionTokens != 5 || got.Usage.TotalTokens != 165 {
		t.Fatalf("usage = %+v", got.Usage)
	}
	if got.Usage.PromptTokensDetails == nil ||
		got.Usage.PromptTokensDetails.CachedTokens != 100 ||
		got.Usage.PromptTokensDetails.CacheCreationTokens != 50 {
		t.Fatalf("prompt_tokens_details = %+v", got.Usage.PromptTokensDetails)
	}
}

// TestAnthropicConvertStreamUsage 验证流式 usage 跨事件累积：
// message_start 给输入侧（含缓存读写），message_delta 给输出侧，最终独立 usage 块完整下发
func TestAnthropicConvertStreamUsage(t *testing.T) {
	ad := NewAnthropicAdapter()
	events := []string{
		`{"type":"message_start","message":{"id":"msg_1","usage":{"input_tokens":10,"output_tokens":1,"cache_creation_input_tokens":50,"cache_read_input_tokens":100}}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hi"}}`,
		`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":5}}`,
		`{"type":"message_stop"}`,
	}

	var chunks []string
	for _, ev := range events {
		out, err := ad.ConvertStreamChunk([]byte("event: x\ndata: "+ev+"\n\n"), "claude-3-5-sonnet")
		if err != nil {
			t.Fatalf("ConvertStreamChunk: %v", err)
		}
		for _, block := range strings.Split(string(out), "\n\n") {
			if strings.TrimSpace(block) != "" {
				chunks = append(chunks, block)
			}
		}
	}

	// message_start → role 块（带输入侧 usage）+ text 增量 + 结束块 + usage 块 + [DONE]
	if len(chunks) != 5 {
		t.Fatalf("chunks = %d, want 5; chunks=%v", len(chunks), chunks)
	}
	if !strings.Contains(chunks[0], `"prompt_tokens":160`) ||
		!strings.Contains(chunks[0], `"cached_tokens":100`) ||
		!strings.Contains(chunks[0], `"cache_creation_tokens":50`) {
		t.Fatalf("message_start chunk usage = %s", chunks[0])
	}
	if !strings.Contains(chunks[2], `"finish_reason":"stop"`) {
		t.Fatalf("message_delta chunk = %s", chunks[2])
	}
	// usage 块 choices 为空，携带累积后的完整 usage
	if !strings.Contains(chunks[3], `"choices":[]`) ||
		!strings.Contains(chunks[3], `"prompt_tokens":160`) ||
		!strings.Contains(chunks[3], `"completion_tokens":5`) {
		t.Fatalf("usage chunk = %s", chunks[3])
	}
	if !strings.Contains(chunks[4], "[DONE]") {
		t.Fatalf("final chunk = %s", chunks[4])
	}
}
