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
