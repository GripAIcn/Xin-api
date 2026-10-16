package adapter

import (
	"fmt"
)

// UpstreamRequest 上游请求抽象：由适配器根据入站 OpenAI Chat 请求构造
// reverse_proxy 直接使用其 Method/Endpoint/Body/Headers 发起请求，不再自行拼装
type UpstreamRequest struct {
	Method   string            // HTTP 方法，通常为 POST
	Endpoint string            // 上游 API 路径（相对渠道 BaseURL），如 /v1/chat/completions
	Body     []byte            // 已完成协议转换的请求体
	Headers  map[string]string // 需附加的请求头（含鉴权头）
}

// ProviderAdapter 上游供应商适配器接口
// 每个 API 规范（OpenAI、Anthropic、Gemini 等）实现此接口；注册表按渠道 provider 查找
type ProviderAdapter interface {
	// Name 返回供应商标识，如 "openai"、"anthropic"、"gemini"
	Name() string

	// ExtractModel 从入站（OpenAI Chat）请求体中提取对外模型名，用于渠道路由
	ExtractModel(body []byte) (string, error)

	// BuildUpstreamRequest 将入站 OpenAI Chat 请求体转换为上游原生请求
	// upstreamModel 为该渠道登记的实际上游模型名；apiKey 为该渠道密钥
	BuildUpstreamRequest(body []byte, upstreamModel, apiKey string) (*UpstreamRequest, error)

	// ConvertResponse 将上游非流式响应体转换为 OpenAI Chat 响应体
	// externalModel 为客户端请求的对外模型名，用于回写响应 model 字段
	ConvertResponse(body []byte, externalModel string) ([]byte, error)

	// ConvertStreamChunk 将上游流式的单个 SSE 事件块转换为 OpenAI SSE 事件块
	// 返回 nil 表示该事件无需下发（如仅用于内部状态的事件）
	ConvertStreamChunk(chunk []byte, externalModel string) ([]byte, error)

	// UpstreamModelsRequest 构造「获取上游可用模型列表」的请求（通常 GET /v1/models）
	UpstreamModelsRequest(apiKey string) *UpstreamRequest
}

// Registry 适配器注册表，按 provider 查找适配器
type Registry struct {
	adapters map[string]ProviderAdapter
}

// NewRegistry 创建适配器注册表
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]ProviderAdapter),
	}
}

// Register 按适配器 Name() 注册
func (r *Registry) Register(adapter ProviderAdapter) {
	r.adapters[adapter.Name()] = adapter
}

// RegisterAlias 以别名注册适配器（如 deepseek 复用 OpenAI 兼容适配器）
func (r *Registry) RegisterAlias(alias string, adapter ProviderAdapter) {
	r.adapters[alias] = adapter
}

// GetAdapter 根据渠道 provider 获取适配器
func (r *Registry) GetAdapter(provider string) (ProviderAdapter, error) {
	adapter, ok := r.adapters[provider]
	if !ok {
		return nil, fmt.Errorf("no adapter found for provider: %s", provider)
	}
	return adapter, nil
}

// openaiSSE 将 JSON 负载包装为 OpenAI SSE 事件块
func openaiSSE(payload []byte) []byte {
	out := make([]byte, 0, len(payload)+8)
	out = append(out, "data: "...)
	out = append(out, payload...)
	out = append(out, '\n', '\n')
	return out
}

// extractSSEData 从 SSE 事件块中提取 data: 后的 JSON 负载（取首个 data: 行）
func extractSSEData(chunk []byte) []byte {
	for _, line := range splitLines(chunk) {
		if len(line) >= 5 && string(line[:5]) == "data:" {
			return trimSpace(line[5:])
		}
	}
	return nil
}

// splitLines 按 \n 切分并去除行尾 \r
func splitLines(b []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i := 0; i < len(b); i++ {
		if b[i] == '\n' {
			lines = append(lines, trimCR(b[start:i]))
			start = i + 1
		}
	}
	if start < len(b) {
		lines = append(lines, trimCR(b[start:]))
	}
	return lines
}

func trimCR(b []byte) []byte {
	if n := len(b); n > 0 && b[n-1] == '\r' {
		return b[:n-1]
	}
	return b
}

func trimSpace(b []byte) []byte {
	start, end := 0, len(b)
	for start < end && isSpace(b[start]) {
		start++
	}
	for end > start && isSpace(b[end-1]) {
		end--
	}
	return b[start:end]
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}
