package inbound

import (
	"encoding/json"
	"testing"
)

// geminiNormOut Gemini 归一化断言结构
type geminiNormOut struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"messages"`
}

func TestGeminiNormalizeImageInput(t *testing.T) {
	in := []byte(`{
		"contents":[{"role":"user","parts":[
			{"text":"look at <this>"},
			{"inlineData":{"mimeType":"image/png","data":"aGVsbG8="}},
			{"fileData":{"mimeType":"image/jpeg","fileUri":"https://example.com/a.jpg"}}
		]}],
		"generationConfig":{"maxOutputTokens":128}
	}`)
	c := &geminiCodec{}
	out, err := c.NormalizeRequest(in, NormalizeOptions{PathModel: "gemini-2.0-flash"})
	if err != nil {
		t.Fatalf("NormalizeRequest error: %v", err)
	}

	var got geminiNormOut
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	if got.Model != "gemini-2.0-flash" {
		t.Errorf("model = %q, want gemini-2.0-flash", got.Model)
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
	if parts[0].Type != "text" || parts[0].Text != "look at <this>" {
		t.Errorf("parts[0] = %+v, want text", parts[0])
	}
	if parts[1].Type != "image_url" || parts[1].ImageURL.URL != "data:image/png;base64,aGVsbG8=" {
		t.Errorf("parts[1] = %+v, want inlineData data URI", parts[1])
	}
	if parts[2].Type != "image_url" || parts[2].ImageURL.URL != "https://example.com/a.jpg" {
		t.Errorf("parts[2] = %+v, want fileData uri", parts[2])
	}

	// 纯文本内容仍保持字符串形式
	in = []byte(`{"contents":[{"role":"user","parts":[{"text":"hi"},{"text":" there"}]}]}`)
	out, err = c.NormalizeRequest(in, NormalizeOptions{PathModel: "gemini-2.0-flash"})
	if err != nil {
		t.Fatalf("NormalizeRequest error: %v", err)
	}
	var textOnly geminiNormOut
	if err := json.Unmarshal(out, &textOnly); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	var text string
	if err := json.Unmarshal(textOnly.Messages[0].Content, &text); err != nil {
		t.Fatalf("content is not a string: %v", err)
	}
	if text != "hi there" {
		t.Errorf("content = %q, want %q", text, "hi there")
	}
}
