package adapter

import (
	"encoding/json"
	"fmt"
	"strings"
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

// RawRequestBuilder 可选能力：入站协议与上游协议一致时，直接复用客户端原始请求体
// 归一化为 OpenAI Chat 会丢失协议私有字段（如 Messages 的 cache_control / metadata），
// 因此协议同构的链路走原样透传，仅改写 model 与鉴权头
type RawRequestBuilder interface {
	// SupportsRawPassthrough 入站协议是否可按原始请求体透传
	SupportsRawPassthrough(protocol string) bool

	// BuildUpstreamRequestFromRaw 以客户端原始请求体构造上游请求
	// upstreamModel 非空时改写 model；clientHeaders 为需沿用的客户端协议级请求头
	BuildUpstreamRequestFromRaw(rawBody []byte, upstreamModel, apiKey string, clientHeaders map[string]string) (*UpstreamRequest, error)
}

// AdapterFactory 适配器构造器
// 适配器可能持有跨 chunk 的流式转换状态（如工具调用的块序号映射），因此注册表只存构造器，
// 每次转发实例化一份，保证同渠道并发请求之间状态不串扰（与 inbound.Registry 同构）
type AdapterFactory func() ProviderAdapter

// Registry 适配器注册表，按 provider 查找适配器构造器
type Registry struct {
	adapters map[string]AdapterFactory
}

// NewRegistry 创建适配器注册表
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]AdapterFactory),
	}
}

// Register 按适配器 Name() 注册
func (r *Registry) Register(factory AdapterFactory) {
	r.adapters[factory().Name()] = factory
}

// RegisterAlias 以别名注册适配器（如 deepseek 复用 OpenAI 兼容适配器）
func (r *Registry) RegisterAlias(alias string, factory AdapterFactory) {
	r.adapters[alias] = factory
}

// GetAdapter 根据渠道 provider 实例化适配器
func (r *Registry) GetAdapter(provider string) (ProviderAdapter, error) {
	factory, ok := r.adapters[provider]
	if !ok {
		return nil, fmt.Errorf("no adapter found for provider: %s", provider)
	}
	return factory(), nil
}

// chatContentPart 规范化后的 OpenAI Chat content part（文本或图片）
type chatContentPart struct {
	Type     string // text / image_url
	Text     string
	ImageURL string
}

// parseChatContentParts 解析 OpenAI message.content：
// content 可能是字符串，也可能是 [{type:"text",...},{type:"image_url",...}] 数组
func parseChatContentParts(raw json.RawMessage) []chatContentPart {
	trimmed := trimSpace(raw)
	if len(trimmed) == 0 {
		return nil
	}
	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(trimmed, &s); err == nil && s != "" {
			return []chatContentPart{{Type: "text", Text: s}}
		}
		return nil
	}
	if trimmed[0] != '[' {
		return nil
	}

	var blocks []struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		ImageURL struct {
			URL string `json:"url"`
		} `json:"image_url"`
	}
	if err := json.Unmarshal(trimmed, &blocks); err != nil {
		return nil
	}

	parts := make([]chatContentPart, 0, len(blocks))
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if b.Text != "" {
				parts = append(parts, chatContentPart{Type: "text", Text: b.Text})
			}
		case "image_url":
			if b.ImageURL.URL != "" {
				parts = append(parts, chatContentPart{Type: "image_url", ImageURL: b.ImageURL.URL})
			}
		}
	}
	return parts
}

// chatContentText 提取 content 中的纯文本部分
func chatContentText(raw json.RawMessage) string {
	var sb strings.Builder
	for _, p := range parseChatContentParts(raw) {
		if p.Type == "text" {
			sb.WriteString(p.Text)
		}
	}
	return sb.String()
}

// splitDataURI 将 data:<mediaType>;base64,<data> 拆分为 mediaType 与 base64 数据
func splitDataURI(url string) (mediaType, data string, ok bool) {
	if !strings.HasPrefix(url, "data:") {
		return "", "", false
	}
	rest := url[len("data:"):]
	comma := strings.IndexByte(rest, ',')
	if comma < 0 {
		return "", "", false
	}
	meta := rest[:comma]
	if !strings.HasSuffix(meta, ";base64") {
		return "", "", false
	}
	mediaType = strings.TrimSuffix(meta, ";base64")
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	return mediaType, rest[comma+1:], true
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
