package adapter

import (
	"encoding/json"
	"strings"
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

func TestGeminiBuildUpstreamRequestTools(t *testing.T) {
	body := `{
		"model": "gemini-2.0-flash",
		"tools": [{
			"type": "function",
			"function": {
				"name": "get_weather",
				"description": "查询天气",
				"parameters": {"type": "object", "properties": {"location": {"type": "string"}}}
			}
		}],
		"tool_choice": "required",
		"messages": [
			{"role": "user", "content": "北京天气如何"},
			{"role": "assistant", "content": "稍等", "tool_calls": [{"id": "call_1", "type": "function", "function": {"name": "get_weather", "arguments": "{\"location\":\"Beijing\"}"}}]},
			{"role": "tool", "tool_call_id": "call_1", "content": "晴 25度"},
			{"role": "tool", "tool_call_id": "call_unknown", "name": "get_time", "content": "{\"temp\": 26}"}
		]
	}`

	ad := NewGeminiAdapter()
	req, err := ad.BuildUpstreamRequest([]byte(body), "", "key")
	if err != nil {
		t.Fatalf("BuildUpstreamRequest: %v", err)
	}

	var out struct {
		Tools []struct {
			FunctionDeclarations []struct {
				Name        string          `json:"name"`
				Description string          `json:"description"`
				Parameters  json.RawMessage `json:"parameters"`
			} `json:"functionDeclarations"`
		} `json:"tools"`
		ToolConfig struct {
			FunctionCallingConfig struct {
				Mode string `json:"mode"`
			} `json:"functionCallingConfig"`
		} `json:"toolConfig"`
		Contents []struct {
			Role  string `json:"role"`
			Parts []struct {
				Text             string          `json:"text"`
				FunctionCall     json.RawMessage `json:"functionCall"`
				FunctionResponse json.RawMessage `json:"functionResponse"`
			} `json:"parts"`
		} `json:"contents"`
	}
	if err := json.Unmarshal(req.Body, &out); err != nil {
		t.Fatalf("unmarshal upstream body: %v", err)
	}

	// tools → functionDeclarations（parameters 改名不变）
	if len(out.Tools) != 1 || len(out.Tools[0].FunctionDeclarations) != 1 {
		t.Fatalf("tools = %+v", out.Tools)
	}
	decl := out.Tools[0].FunctionDeclarations[0]
	if decl.Name != "get_weather" || decl.Description != "查询天气" {
		t.Fatalf("declaration = %+v", decl)
	}
	var params map[string]any
	if err := json.Unmarshal(decl.Parameters, &params); err != nil {
		t.Fatalf("parameters not object: %v", err)
	}
	// tool_choice required → toolConfig.mode ANY
	if out.ToolConfig.FunctionCallingConfig.Mode != "ANY" {
		t.Fatalf("mode = %q, want ANY", out.ToolConfig.FunctionCallingConfig.Mode)
	}

	// contents: user / model(functionCall) / user(合并 functionResponse)
	if len(out.Contents) != 3 {
		t.Fatalf("len(contents) = %d, want 3: %+v", len(out.Contents), out.Contents)
	}
	if out.Contents[0].Role != "user" || out.Contents[0].Parts[0].Text != "北京天气如何" {
		t.Fatalf("contents[0] = %+v", out.Contents[0])
	}

	// model 消息：文本 part + functionCall part
	if out.Contents[1].Role != "model" || len(out.Contents[1].Parts) != 2 {
		t.Fatalf("contents[1] = %+v", out.Contents[1])
	}
	if out.Contents[1].Parts[0].Text != "稍等" {
		t.Fatalf("contents[1].parts[0] = %+v", out.Contents[1].Parts[0])
	}
	var fcall struct {
		Name string         `json:"name"`
		Args map[string]any `json:"args"`
	}
	if err := json.Unmarshal(out.Contents[1].Parts[1].FunctionCall, &fcall); err != nil {
		t.Fatalf("functionCall not object: %v", err)
	}
	if fcall.Name != "get_weather" || fcall.Args["location"] != "Beijing" {
		t.Fatalf("functionCall = %+v", fcall)
	}

	// 连续两条 tool 消息合并为一条 user 消息的两个 functionResponse
	if out.Contents[2].Role != "user" || len(out.Contents[2].Parts) != 2 {
		t.Fatalf("contents[2] = %+v", out.Contents[2])
	}
	for i, part := range out.Contents[2].Parts {
		var fresp struct {
			Name     string         `json:"name"`
			Response map[string]any `json:"response"`
		}
		if err := json.Unmarshal(part.FunctionResponse, &fresp); err != nil {
			t.Fatalf("functionResponse[%d] not object: %v", i, err)
		}
		if i == 0 {
			// 无 name 字段时按 tool_call_id 回查 assistant tool_calls 的函数名
			if fresp.Name != "get_weather" {
				t.Fatalf("functionResponse[0].name = %q, want get_weather", fresp.Name)
			}
			if fresp.Response["result"] != "晴 25度" {
				t.Fatalf("functionResponse[0].response = %+v", fresp.Response)
			}
		}
		if i == 1 {
			// 自带 name 字段时优先使用（对齐 new-api 的解析顺序）
			if fresp.Name != "get_time" {
				t.Fatalf("functionResponse[1].name = %q, want get_time", fresp.Name)
			}
			if fresp.Response["temp"] != float64(26) {
				t.Fatalf("functionResponse[1].response = %+v", fresp.Response)
			}
		}
	}
}

func TestGeminiConvertResponseFunctionCall(t *testing.T) {
	body := `{
		"candidates": [{
			"content": {"parts": [
				{"text": "我来查询"},
				{"functionCall": {"name": "get_weather", "args": {"location": "Shanghai"}}}
			]},
			"finishReason": "STOP"
		}],
		"usageMetadata": {"promptTokenCount": 10, "candidatesTokenCount": 5, "totalTokenCount": 15}
	}`

	ad := NewGeminiAdapter()
	out, err := ad.ConvertResponse([]byte(body), "gemini-2.0-flash")
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
	if calls[0].Type != "function" || calls[0].Function.Name != "get_weather" || calls[0].ID == "" {
		t.Fatalf("tool_call = %+v", calls[0])
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(calls[0].Function.Arguments), &args); err != nil {
		t.Fatalf("arguments not JSON: %v", err)
	}
	if args["location"] != "Shanghai" {
		t.Fatalf("arguments = %+v", args)
	}
	// 有函数调用时 finish_reason 按 OpenAI 语义为 tool_calls
	if got.Choices[0].FinishReason != "tool_calls" {
		t.Fatalf("finish_reason = %q, want tool_calls", got.Choices[0].FinishReason)
	}
}

func TestGeminiConvertStreamFunctionCall(t *testing.T) {
	ad := NewGeminiAdapter()

	chunkWithCall := `{"candidates":[{"content":{"parts":[{"functionCall":{"name":"get_weather","args":{"location":"Paris"}}}]}}]}`
	chunkRepeat := `{"candidates":[{"content":{"parts":[{"functionCall":{"name":"get_weather","args":{"location":"Paris"}}}]}}]}`
	chunkFinish := `{"candidates":[{"content":{"parts":[{"functionCall":{"name":"get_weather","args":{"location":"Paris"}}}]},"finishReason":"STOP"}]}`

	out1, err := ad.ConvertStreamChunk([]byte("data: "+chunkWithCall+"\n\n"), "gemini-2.0-flash")
	if err != nil {
		t.Fatalf("ConvertStreamChunk: %v", err)
	}
	out2, err := ad.ConvertStreamChunk([]byte("data: "+chunkRepeat+"\n\n"), "gemini-2.0-flash")
	if err != nil {
		t.Fatalf("ConvertStreamChunk: %v", err)
	}
	out3, err := ad.ConvertStreamChunk([]byte("data: "+chunkFinish+"\n\n"), "gemini-2.0-flash")
	if err != nil {
		t.Fatalf("ConvertStreamChunk: %v", err)
	}

	// 首个工具块下发 tool_calls 起始增量：index 0 / id / name / arguments
	s1 := string(out1)
	if !strings.Contains(s1, `"tool_calls":[{"function":{"arguments":"{\"location\":\"Paris\"}","name":"get_weather"},"id":"`) ||
		!strings.Contains(s1, `"index":0`) || !strings.Contains(s1, `"type":"function"`) {
		t.Fatalf("chunk1 = %s", s1)
	}

	// 同一 part 重发相同 args 不产生重复增量
	if strings.Contains(string(out2), `"tool_calls"`) {
		t.Fatalf("chunk2 should not repeat tool_calls: %s", string(out2))
	}

	// 流内 id 保持一致
	id := streamChunkID(t, s1)
	if got := streamChunkID(t, string(out2)); got != id {
		t.Fatalf("chunk id changed: %q vs %q", id, got)
	}

	// 有函数调用时 finish_reason 为 tool_calls，并追加 [DONE]
	s3 := string(out3)
	if !strings.Contains(s3, `"finish_reason":"tool_calls"`) {
		t.Fatalf("chunk3 = %s", s3)
	}
	if !strings.Contains(s3, "[DONE]") {
		t.Fatalf("chunk3 missing [DONE]: %s", s3)
	}
}

// streamChunkID 从 OpenAI SSE 事件块中解析出 id 字段
func streamChunkID(t *testing.T, sse string) string {
	t.Helper()
	data := extractSSEData([]byte(sse))
	var chunk struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &chunk); err != nil {
		t.Fatalf("unmarshal chunk: %v", err)
	}
	return chunk.ID
}
