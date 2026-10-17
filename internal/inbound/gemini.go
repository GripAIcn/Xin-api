package inbound

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"Xin-api/pkg/response"
)

// geminiCodec Google Gemini generateContent 入站协议 ⇄ OpenAI Chat
type geminiCodec struct{}

func (c *geminiCodec) Name() string { return ProtocolGemini }

// NormalizeRequest Gemini generateContent 请求 → 规范 OpenAI Chat 请求
// 模型名来自 URL 路径；contents/systemInstruction → messages；generationConfig → 采样参数
func (c *geminiCodec) NormalizeRequest(body []byte, opts NormalizeOptions) ([]byte, error) {
	if opts.PathModel == "" {
		return nil, fmt.Errorf("gemini: model field is required in path")
	}
	var req struct {
		Contents          []geminiContent `json:"contents"`
		SystemInstruction *geminiContent  `json:"systemInstruction"`
		GenerationConfig  struct {
			MaxOutputTokens *int     `json:"maxOutputTokens"`
			Temperature     *float64 `json:"temperature"`
			TopP            *float64 `json:"topP"`
		} `json:"generationConfig"`
	}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, fmt.Errorf("gemini: parse request body failed: %w", err)
		}
	}

	messages := make([]map[string]any, 0, len(req.Contents)+1)
	if req.SystemInstruction != nil {
		if text := req.SystemInstruction.text(); text != "" {
			messages = append(messages, map[string]any{"role": "system", "content": text})
		}
	}
	for _, ct := range req.Contents {
		role := "user"
		if ct.Role == "model" {
			role = "assistant"
		}
		messages = append(messages, map[string]any{"role": role, "content": ct.text()})
	}

	out := map[string]any{
		"model":    opts.PathModel,
		"messages": messages,
		"stream":   opts.Stream,
	}
	if req.GenerationConfig.MaxOutputTokens != nil {
		out["max_tokens"] = *req.GenerationConfig.MaxOutputTokens
	}
	if req.GenerationConfig.Temperature != nil {
		out["temperature"] = *req.GenerationConfig.Temperature
	}
	if req.GenerationConfig.TopP != nil {
		out["top_p"] = *req.GenerationConfig.TopP
	}
	return json.Marshal(out)
}

// EncodeResponse OpenAI Chat 非流式响应 → Gemini generateContent 响应
func (c *geminiCodec) EncodeResponse(openAIResponse []byte) ([]byte, error) {
	var cc chatCompletion
	if err := json.Unmarshal(openAIResponse, &cc); err != nil {
		return openAIResponse, nil
	}
	text := ""
	finishReason := "STOP"
	if len(cc.Choices) > 0 {
		text = cc.Choices[0].Message.Content
		if cc.Choices[0].FinishReason != nil && *cc.Choices[0].FinishReason != "" {
			finishReason = mapFinishReasonToGemini(*cc.Choices[0].FinishReason)
		}
	}
	out := map[string]any{
		"candidates": []map[string]any{{
			"content":      map[string]any{"role": "model", "parts": []map[string]any{{"text": text}}},
			"finishReason": finishReason,
		}},
	}
	if cc.Usage != nil {
		out["usageMetadata"] = geminiUsage(cc.Usage)
	}
	if cc.Model != "" {
		out["modelVersion"] = cc.Model
	}
	return json.Marshal(out)
}

// EncodeStreamChunk OpenAI SSE chunk → Gemini alt=sse 事件块
func (c *geminiCodec) EncodeStreamChunk(chunk []byte) ([]byte, error) {
	payload := extractSSEPayload(chunk)
	if len(payload) == 0 || string(payload) == "[DONE]" {
		return nil, nil
	}
	var ch chatChunk
	if err := json.Unmarshal(payload, &ch); err != nil {
		return nil, nil
	}

	out := map[string]any{}
	if len(ch.Choices) > 0 {
		text := ch.Choices[0].Delta.Content
		fr := ch.Choices[0].FinishReason
		if text != "" || (fr != nil && *fr != "") {
			candidate := map[string]any{
				"content": map[string]any{"role": "model", "parts": []map[string]any{{"text": text}}},
			}
			if fr != nil && *fr != "" {
				candidate["finishReason"] = mapFinishReasonToGemini(*fr)
			}
			out["candidates"] = []map[string]any{candidate}
		}
	}
	if ch.Usage != nil {
		out["usageMetadata"] = geminiUsage(ch.Usage)
	}
	if len(out) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(out)
	if err != nil {
		return nil, nil
	}
	return sseEvent("", data), nil
}

// LocalError Gemini 原生错误体 {"error":{"code","message","status"}}
func (c *geminiCodec) LocalError(bizCode int) (int, []byte) {
	status := response.MapToHTTPStatus(bizCode)
	return status, geminiErrorBody(status, response.Msg(bizCode))
}

// UpstreamError 上游非 200 → Gemini 原生错误体（保留上游 message）
func (c *geminiCodec) UpstreamError(upstreamStatus int, upstreamBody []byte) (int, []byte) {
	status := upstreamStatus
	if status < http.StatusBadRequest {
		status = http.StatusBadGateway
	}
	msg := extractErrorMessage(upstreamBody)
	if msg == "" {
		msg = http.StatusText(status)
	}
	return status, geminiErrorBody(status, msg)
}

// geminiContent Gemini contents 单条消息
type geminiContent struct {
	Role  string `json:"role"`
	Parts []struct {
		Text string `json:"text"`
	} `json:"parts"`
}

// text 拼接该消息所有 part 的文本
func (c geminiContent) text() string {
	var b strings.Builder
	for _, p := range c.Parts {
		b.WriteString(p.Text)
	}
	return b.String()
}

// geminiUsage OpenAI usage → Gemini usageMetadata
func geminiUsage(u *chatUsage) map[string]any {
	return map[string]any{
		"promptTokenCount":     u.PromptTokens,
		"candidatesTokenCount": u.CompletionTokens,
		"totalTokenCount":      u.TotalTokens,
	}
}

// geminiErrorBody 组装 Gemini 原生错误体
func geminiErrorBody(status int, message string) []byte {
	body := map[string]any{
		"error": map[string]any{
			"code":    status,
			"message": message,
			"status":  geminiErrorStatus(status),
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return []byte(`{"error":{"code":500,"message":"gateway error","status":"INTERNAL"}}`)
	}
	return data
}

// geminiErrorStatus HTTP 状态码 → Gemini 的 status 枚举
func geminiErrorStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "INVALID_ARGUMENT"
	case http.StatusUnauthorized:
		return "UNAUTHENTICATED"
	case http.StatusForbidden:
		return "PERMISSION_DENIED"
	case http.StatusNotFound:
		return "NOT_FOUND"
	case http.StatusTooManyRequests:
		return "RESOURCE_EXHAUSTED"
	case http.StatusServiceUnavailable:
		return "UNAVAILABLE"
	case http.StatusGatewayTimeout:
		return "DEADLINE_EXCEEDED"
	default:
		return "INTERNAL"
	}
}

// mapFinishReasonToGemini OpenAI finish_reason → Gemini finishReason
func mapFinishReasonToGemini(reason string) string {
	switch reason {
	case "length":
		return "MAX_TOKENS"
	case "content_filter":
		return "SAFETY"
	default:
		return "STOP"
	}
}
