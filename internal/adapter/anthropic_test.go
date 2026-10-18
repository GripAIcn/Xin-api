package adapter

import (
	"encoding/json"
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
