package adapter

import "fmt"

// ProviderAdapter 上游供应商适配器接口
// 每个 API 规范（OpenAI、Anthropic 等）实现此接口
type ProviderAdapter interface {
	// Name 返回供应商标识，如 "openai"、"anthropic"
	Name() string

	// Endpoints 返回该适配器能处理的请求路径列表
	Endpoints() []string

	// ExtractModel 从请求体中提取模型名称，用于渠道路由
	ExtractModel(body []byte) (string, error)

	// UpstreamEndpoint 返回上游 API 的请求路径
	UpstreamEndpoint() string

	// ContentType 返回请求的 Content-Type
	ContentType() string
}

// Registry 适配器注册表，按请求路径查找适配器
type Registry struct {
	adapters map[string]ProviderAdapter
}

// NewRegistry 创建适配器注册表
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]ProviderAdapter),
	}
}

// Register 注册适配器
func (r *Registry) Register(adapter ProviderAdapter) {
	for _, path := range adapter.Endpoints() {
		r.adapters[path] = adapter
	}
}

// GetAdapter 根据请求路径获取适配器
func (r *Registry) GetAdapter(path string) (ProviderAdapter, error) {
	adapter, ok := r.adapters[path]
	if !ok {
		return nil, fmt.Errorf("no adapter found for path: %s", path)
	}
	return adapter, nil
}
