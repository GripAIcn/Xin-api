package adapter

import (
	"encoding/json"
	"testing"
)

func TestGeminiBuildUpstreamRequestImageInput(t *testing.T) {
	body := `{
		"model": "gemini-2.0-flash",
		"messages": [
			{"role": "system", "content": "sys"},
			{"role": "user", "content": [
				{"type": "text", "text": "describe"},
				{"type": "image_url", "image_url": {"url": "data:image/png;base64,aGVsbG8="}},
				{"type": "image_url", "image_url": {"url": "https://example.com/a.jpg"}}
			]}
		]
	}`

	ad := NewGeminiAdapter()
	req, err := ad.BuildUpstreamRequest([]byte(body), "", "key")
	if err != nil {
		t.Fatalf("BuildUpstreamRequest: %v", err)
	}

	var out struct {
		SystemInstruction struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"systemInstruction"`
		Contents []struct {
			Role  string `json:"role"`
			Parts []struct {
				Text       string `json:"text"`
				InlineData *struct {
					MimeType string `json:"mimeType"`
					Data     string `json:"data"`
				} `json:"inlineData"`
				FileData *struct {
					FileURI string `json:"fileUri"`
				} `json:"fileData"`
			} `json:"parts"`
		} `json:"contents"`
	}
	if err := json.Unmarshal(req.Body, &out); err != nil {
		t.Fatalf("unmarshal upstream body: %v", err)
	}
	if len(out.SystemInstruction.Parts) != 1 || out.SystemInstruction.Parts[0].Text != "sys" {
		t.Fatalf("systemInstruction = %+v", out.SystemInstruction)
	}
	if len(out.Contents) != 1 || out.Contents[0].Role != "user" {
		t.Fatalf("contents = %+v", out.Contents)
	}

	parts := out.Contents[0].Parts
	if len(parts) != 3 {
		t.Fatalf("parts = %+v, want 3", parts)
	}
	if parts[0].Text != "describe" {
		t.Fatalf("text part = %+v", parts[0])
	}
	if parts[1].InlineData == nil || parts[1].InlineData.MimeType != "image/png" ||
		parts[1].InlineData.Data != "aGVsbG8=" {
		t.Fatalf("inlineData part = %+v", parts[1])
	}
	if parts[2].FileData == nil || parts[2].FileData.FileURI != "https://example.com/a.jpg" {
		t.Fatalf("fileData part = %+v", parts[2])
	}
}

func TestGeminiBuildUpstreamRequestPlainTextContent(t *testing.T) {
	body := `{"model":"gemini-2.0-flash","messages":[{"role":"user","content":"hi"}]}`

	ad := NewGeminiAdapter()
	req, err := ad.BuildUpstreamRequest([]byte(body), "", "key")
	if err != nil {
		t.Fatalf("BuildUpstreamRequest: %v", err)
	}

	var out struct {
		Contents []struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"contents"`
	}
	if err := json.Unmarshal(req.Body, &out); err != nil {
		t.Fatalf("unmarshal upstream body: %v", err)
	}
	if len(out.Contents) != 1 || len(out.Contents[0].Parts) != 1 || out.Contents[0].Parts[0].Text != "hi" {
		t.Fatalf("contents = %+v", out.Contents)
	}
}
