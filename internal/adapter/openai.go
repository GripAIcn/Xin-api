package adapter

import (
	"encoding/json"
	"fmt"
)

const openaiContentType = "application/json"

type openaiAdapter struct{}

// NewOpenAIAdapter 创建 OpenAI 适配器
func NewOpenAIAdapter() ProviderAdapter {
	return &openaiAdapter{}
}

func (a *openaiAdapter) Name() string {
	return "openai"
}

func (a *openaiAdapter) Endpoints() []string {
	return []string{"/v1/chat/completions"}
}

func (a *openaiAdapter) ExtractModel(body []byte) (string, error) {
	var req struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return "", fmt.Errorf("openai: parse request body failed: %w", err)
	}
	if req.Model == "" {
		return "", fmt.Errorf("openai: model field is required")
	}
	return req.Model, nil
}

func (a *openaiAdapter) UpstreamEndpoint() string {
	return "/v1/chat/completions"
}

func (a *openaiAdapter) ContentType() string {
	return openaiContentType
}
