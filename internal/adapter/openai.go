package adapter

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type openaiAdapter struct{}

// NewOpenAIAdapter 创建 OpenAI 兼容适配器（deepseek 等 OpenAI 兼容供应商复用此实现）
func NewOpenAIAdapter() ProviderAdapter {
	return &openaiAdapter{}
}

func (a *openaiAdapter) Name() string {
	return "openai"
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

// BuildUpstreamRequest 直连上游 /v1/chat/completions，仅把 model 改写为上游名，其余原样透传
func (a *openaiAdapter) BuildUpstreamRequest(body []byte, upstreamModel, apiKey string) (*UpstreamRequest, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("openai: parse request body failed: %w", err)
	}
	if upstreamModel != "" {
		m, err := json.Marshal(upstreamModel)
		if err != nil {
			return nil, err
		}
		raw["model"] = m
	}
	newBody, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	return &UpstreamRequest{
		Method:   http.MethodPost,
		Endpoint: "/v1/chat/completions",
		Body:     newBody,
		Headers:  map[string]string{"Authorization": "Bearer " + apiKey},
	}, nil
}

// ConvertResponse OpenAI 兼容响应原样透传，仅把 model 回写为对外名
func (a *openaiAdapter) ConvertResponse(body []byte, externalModel string) ([]byte, error) {
	return rewriteJSONModel(body, externalModel)
}

// ConvertStreamChunk SSE 事件透传，仅把 data 负载内的 model 回写为对外名
func (a *openaiAdapter) ConvertStreamChunk(chunk []byte, externalModel string) ([]byte, error) {
	data := extractSSEData(chunk)
	if len(data) == 0 || string(data) == "[DONE]" {
		return chunk, nil
	}
	newData, err := rewriteJSONModel(data, externalModel)
	if err != nil {
		return chunk, nil
	}
	return openaiSSE(newData), nil
}

// UpstreamModelsRequest 构造 GET {base}/v1/models 请求
func (a *openaiAdapter) UpstreamModelsRequest(apiKey string) *UpstreamRequest {
	return &UpstreamRequest{
		Method:   http.MethodGet,
		Endpoint: "/v1/models",
		Body:     nil,
		Headers:  map[string]string{"Authorization": "Bearer " + apiKey},
	}
}

// rewriteJSONModel 若 JSON 对象含 model 字段，则改写为 externalModel；解析失败时原样返回
func rewriteJSONModel(body []byte, externalModel string) ([]byte, error) {
	if externalModel == "" {
		return body, nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return body, nil
	}
	if _, ok := raw["model"]; !ok {
		return body, nil
	}
	m, err := json.Marshal(externalModel)
	if err != nil {
		return body, nil
	}
	raw["model"] = m
	out, err := json.Marshal(raw)
	if err != nil {
		return body, nil
	}
	return out, nil
}
