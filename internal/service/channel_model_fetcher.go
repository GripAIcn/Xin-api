package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"Xin-api/internal/store/postgresql"
)

// ErrChannelNotFound 渠道不存在
var ErrChannelNotFound = errors.New("channel not found")

// UpstreamHTTPError 上游返回非 200 状态码，保留状态码与响应体供管理端展示
type UpstreamHTTPError struct {
	StatusCode int
	Body       string
}

func (e *UpstreamHTTPError) Error() string {
	return fmt.Sprintf("upstream returned %d: %s", e.StatusCode, e.Body)
}

// AdminMessage 生成透传到管理端的错误提示，优先提取上游 JSON 中的 code / message
func (e *UpstreamHTTPError) AdminMessage() string {
	body := strings.TrimSpace(e.Body)

	var payload struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(e.Body), &payload); err == nil {
		code, message := payload.Code, payload.Message
		if code == "" {
			code = payload.Error.Code
		}
		if message == "" {
			message = payload.Error.Message
		}
		if message != "" {
			if code != "" {
				return fmt.Sprintf("上游返回 %d（%s）：%s", e.StatusCode, code, message)
			}
			return fmt.Sprintf("上游返回 %d：%s", e.StatusCode, message)
		}
	}

	// 非 JSON 响应（如网关 HTML 页面）截断后原样返回
	if runes := []rune(body); len(runes) > 300 {
		body = string(runes[:300]) + "..."
	}
	if body == "" {
		return fmt.Sprintf("上游返回 %d", e.StatusCode)
	}
	return fmt.Sprintf("上游返回 %d：%s", e.StatusCode, body)
}

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
		return nil, &UpstreamHTTPError{StatusCode: resp.StatusCode, Body: string(body)}
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
