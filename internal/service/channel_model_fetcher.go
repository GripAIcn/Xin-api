package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"Xin-api/internal/store/postgresql"
)

// ErrChannelNotFound 渠道不存在
var ErrChannelNotFound = errors.New("channel not found")

// ChannelModelFetcher 上游可用模型获取服务
// 调用上游 OpenAI 兼容的 GET /v1/models，返回模型 id 列表，供管理员登记模型时选择
type ChannelModelFetcher struct {
	channelRepo postgresql.ChannelRepo
	httpClient  *http.Client
}

// NewChannelModelFetcher 创建上游模型获取服务
func NewChannelModelFetcher(repo postgresql.ChannelRepo) *ChannelModelFetcher {
	return &ChannelModelFetcher{
		channelRepo: repo,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// FetchUpstreamModels 请求上游 GET {base_url}/v1/models，返回模型 id 列表
func (f *ChannelModelFetcher) FetchUpstreamModels(ctx context.Context, channelID int64) ([]string, error) {
	channel, err := f.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, ErrChannelNotFound
	}

	url := joinUpstreamURL(channel.BaseURL, "/v1/models")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+channel.APIKey)

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upstream returned %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("parse upstream models failed: %w", err)
	}

	ids := make([]string, 0, len(payload.Data))
	for _, d := range payload.Data {
		if d.ID != "" {
			ids = append(ids, d.ID)
		}
	}
	return ids, nil
}
