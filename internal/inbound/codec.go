// Package inbound 实现数据面「入站协议编解码」层。
// 数据面统一以 OpenAI Chat 请求体为内部规范形态：
//   - 入站归一化：把 responses / gemini 请求体转换为规范 OpenAI Chat 请求体；
//   - 出站再编码：把 provider 适配器产出的 OpenAI Chat 响应（或 SSE 块）编码回入站协议格式。
//
// Codec 必须每请求实例化（流式编解码需跨 chunk 累积状态），故 Registry 只存构造器。
package inbound

import (
	"bytes"
	"encoding/json"
	"strings"

	"Xin-api/pkg/jsonx"
	"Xin-api/pkg/response"
)

// 入站协议名常量，与 request_logs.protocol / channels.protocol 取值一致
const (
	ProtocolChat      = "chat"
	ProtocolResponses = "responses"
	ProtocolGemini    = "gemini"
	ProtocolMessages  = "messages"
)

// NormalizeOptions 入站归一化的元信息（协议自身无法从 body 中获取的部分）
type NormalizeOptions struct {
	PathModel string // gemini：模型名来自 URL 路径
	Stream    bool   // gemini：由路径 :streamGenerateContent 或 ?alt=sse 决定
}

// Codec 入站协议编解码器
type Codec interface {
	Name() string
	// NormalizeRequest 将入站请求体归一化为规范 OpenAI Chat 请求体
	NormalizeRequest(body []byte, opts NormalizeOptions) (canonical []byte, err error)
	// EncodeResponse 将 OpenAI Chat 非流式响应编码为入站协议响应体
	EncodeResponse(openAIResponse []byte) ([]byte, error)
	// EncodeStreamChunk 将单个 OpenAI SSE 事件块编码为入站协议 SSE 事件块；nil 表示不写出
	EncodeStreamChunk(openAISSESChunk []byte) ([]byte, error)
	// KeepaliveEvent 空闲保活事件（SSE 注释或协议原生 ping）；nil 表示该协议不发送
	KeepaliveEvent() []byte
	// LocalError 按入站协议编码网关本地错误（返回 HTTP 状态码与响应体）
	LocalError(bizCode int) (status int, body []byte)
	// UpstreamError 按入站协议编码上游错误（传入上游原始状态码与响应体）
	UpstreamError(upstreamStatus int, upstreamBody []byte) (status int, body []byte)
}

// Registry 协议构造器注册表：键为协议名，值为工厂函数（每请求实例化，保证并发安全）
type Registry map[string]func() Codec

// NewRegistry 创建内置三种协议的编解码注册表
func NewRegistry() Registry {
	return Registry{
		ProtocolChat:      func() Codec { return &chatCodec{} },
		ProtocolResponses: func() Codec { return &responsesCodec{} },
		ProtocolGemini:    func() Codec { return &geminiCodec{} },
		ProtocolMessages:  func() Codec { return &messagesCodec{} },
	}
}

// New 实例化指定协议的编解码器；未知协议回退为 chat（恒等编解码）
func (r Registry) New(name string) Codec {
	if ctor, ok := r[name]; ok {
		return ctor()
	}
	return &chatCodec{}
}

// ==========================================
// 共享结构：OpenAI Chat 规范形态的响应 / 流式 chunk
// ==========================================

// chatUsage OpenAI Chat usage 字段
type chatUsage struct {
	PromptTokens            int64 `json:"prompt_tokens"`
	CompletionTokens        int64 `json:"completion_tokens"`
	TotalTokens             int64 `json:"total_tokens"`
	PromptTokensDetails     *chatPromptTokensDetails     `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *chatCompletionTokensDetails `json:"completion_tokens_details,omitempty"`
}

// chatPromptTokensDetails prompt_tokens 明细（缓存命中）
type chatPromptTokensDetails struct {
	CachedTokens int64 `json:"cached_tokens"`
}

// chatCompletionTokensDetails completion_tokens 明细（推理 token）
type chatCompletionTokensDetails struct {
	ReasoningTokens int64 `json:"reasoning_tokens"`
}

// chatFunctionCall 工具调用中的函数描述
type chatFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// chatToolCall OpenAI Chat 工具调用
type chatToolCall struct {
	Index    int              `json:"index,omitempty"`
	ID       string           `json:"id,omitempty"`
	Type     string           `json:"type,omitempty"`
	Function chatFunctionCall `json:"function"`
}

// chatMessage 同时用于 message 与 delta（字段一致）
type chatMessage struct {
	Role             string         `json:"role"`
	Content          string         `json:"content"`
	Refusal          string         `json:"refusal,omitempty"`
	ReasoningContent string         `json:"reasoning_content,omitempty"`
	ToolCalls        []chatToolCall `json:"tool_calls,omitempty"`
	ToolCallID       string         `json:"tool_call_id,omitempty"`
}

// chatCompletion OpenAI Chat 非流式响应
type chatCompletion struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      chatMessage `json:"message"`
		FinishReason *string     `json:"finish_reason"`
	} `json:"choices"`
	Usage *chatUsage `json:"usage"`
}

// chatChunk OpenAI Chat 流式 chunk
type chatChunk struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Delta        chatMessage `json:"delta"`
		FinishReason *string     `json:"finish_reason"`
	} `json:"choices"`
	Usage *chatUsage `json:"usage"`
}

// ==========================================
// SSE 工具
// ==========================================

// extractSSEPayload 提取 SSE 块中首个 data: 行的负载（无则返回 nil）
func extractSSEPayload(chunk []byte) []byte {
	for _, line := range bytes.Split(chunk, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) >= 5 && string(line[:5]) == "data:" {
			return bytes.TrimSpace(line[5:])
		}
	}
	return nil
}

// sseEvent 组装一个 SSE 事件块；event 为空时不写 event: 行
func sseEvent(event string, data []byte) []byte {
	var b bytes.Buffer
	if event != "" {
		b.WriteString("event: ")
		b.WriteString(event)
		b.WriteByte('\n')
	}
	b.WriteString("data: ")
	b.Write(data)
	b.WriteString("\n\n")
	return b.Bytes()
}

// ==========================================
// 错误编码（OpenAI 范式，chat / responses 复用）
// ==========================================

// openAILocalError 以 OpenAI 标准错误体编码网关本地错误
func openAILocalError(bizCode int) (int, []byte) {
	status, body := response.BuildOpenAIError(bizCode)
	data, err := jsonx.Marshal(body)
	if err != nil {
		return status, []byte(`{"error":{"message":"gateway error","type":"api_error","code":"gateway_internal_error"}}`)
	}
	return status, data
}

// extractErrorMessage 从上游错误体中提取人类可读 message，供 Gemini 错误再编码使用
func extractErrorMessage(body []byte) string {
	var payload struct {
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && len(payload.Error) > 0 {
		var obj struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(payload.Error, &obj) == nil && obj.Message != "" {
			return obj.Message
		}
		var s string
		if json.Unmarshal(payload.Error, &s) == nil && s != "" {
			return s
		}
	}
	text := strings.TrimSpace(string(body))
	if len(text) > 500 {
		text = text[:500]
	}
	return text
}
