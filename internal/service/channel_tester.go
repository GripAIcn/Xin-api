package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"Xin-api/internal/form"
	"Xin-api/internal/model"
	"Xin-api/internal/store/postgresql"
)

// ModelTestResult 单个模型测试结果
type ModelTestResult struct {
	Model        string `json:"model"`               // 模型名称
	Success      bool   `json:"success"`             // 是否成功
	ResponseTime int64  `json:"response_time_ms"`    // 响应时间（毫秒）
	ErrorMsg     string `json:"error_msg,omitempty"` // 错误信息
}

// ChannelTestResult 单个渠道测试结果
type ChannelTestResult struct {
	ChannelID   int64             `json:"channel_id"`   // 渠道ID
	ChannelName string            `json:"channel_name"` // 渠道名称
	BaseURL     string            `json:"base_url"`     // 渠道地址
	Results     []ModelTestResult `json:"results"`      // 各模型测试结果
}

// ChannelTester 渠道测试服务
type ChannelTester struct {
	channelRepo postgresql.ChannelRepo
	httpClient  *http.Client
}

// NewChannelTester 创建渠道测试服务
func NewChannelTester(repo postgresql.ChannelRepo) *ChannelTester {
	return &ChannelTester{
		channelRepo: repo,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// TestChannel 测试单个渠道的指定模型或所有模型
// 如果 modelName 为空字符串，则测试该渠道支持的所有模型
func (t *ChannelTester) TestChannel(ctx context.Context, channelID int64, modelName string) (*ChannelTestResult, error) {
	channel, err := t.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if channel == nil {
		return nil, fmt.Errorf("channel not found")
	}

	result := &ChannelTestResult{
		ChannelID:   channel.ID,
		ChannelName: channel.Name,
		BaseURL:     channel.BaseURL,
		Results:     []ModelTestResult{},
	}

	// 确定要测试的模型列表
	var modelsToTest []string
	if modelName != "" {
		// 测试指定模型
		modelsToTest = []string{modelName}
	} else {
		// 测试该渠道支持的所有模型
		models := strings.Split(channel.ModelMapping, ",")
		for _, m := range models {
			m = strings.TrimSpace(m)
			if m != "" {
				modelsToTest = append(modelsToTest, m)
			}
		}
	}

	// 并发测试所有模型，但限制并发数
	result.Results = t.testModelsConcurrent(ctx, channel.BaseURL, channel.APIKey, modelsToTest)

	return result, nil
}

// TestGroupAllChannels 测试项目组下所有渠道的所有模型
func (t *ChannelTester) TestGroupAllChannels(ctx context.Context, groupID int64) ([]ChannelTestResult, error) {
	channels, err := t.channelRepo.ListByGroupID(ctx, groupID)
	if err != nil {
		return nil, err
	}

	var results []ChannelTestResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 限制并发渠道测试数量
	semaphore := make(chan struct{}, 5)

	for _, channel := range channels {
		// 只测试正常状态的渠道
		if channel.Status != 1 {
			continue
		}

		wg.Add(1)
		go func(ch model.Channel) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result, err := t.TestChannel(ctx, ch.ID, "")
			if err != nil {
				result = &ChannelTestResult{
					ChannelID:   ch.ID,
					ChannelName: ch.Name,
					BaseURL:     ch.BaseURL,
					Results: []ModelTestResult{
						{
							Model:    "test",
							Success:  false,
							ErrorMsg: err.Error(),
						},
					},
				}
			}

			mu.Lock()
			results = append(results, *result)
			mu.Unlock()
		}(channel)
	}

	wg.Wait()
	return results, nil
}

// testModelsConcurrent 并发测试多个模型
func (t *ChannelTester) testModelsConcurrent(ctx context.Context, baseURL, apiKey string, models []string) []ModelTestResult {
	var results []ModelTestResult
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 限制并发数为 3，避免对上游造成过大压力
	semaphore := make(chan struct{}, 3)

	for _, model := range models {
		wg.Add(1)
		go func(m string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := t.testSingleModel(ctx, baseURL, apiKey, m)

			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(model)
	}

	wg.Wait()
	return results
}

func (t *ChannelTester) testSingleModel(ctx context.Context, baseURL, apiKey, modelName string) ModelTestResult {
	start := time.Now()

	// 构造测试请求
	reqBody := form.ChatCompletionRequest{
		Model: modelName,
		Messages: []form.ChatMessage{
			{Role: "user", Content: "hi"},
		},
		MaxTokens: 5,
		Stream:    false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return ModelTestResult{
			Model:        modelName,
			Success:      false,
			ResponseTime: time.Since(start).Milliseconds(),
			ErrorMsg:     fmt.Sprintf("marshal request failed: %v", err),
		}
	}

	// 构建请求 URL
	url := baseURL
	if !strings.HasSuffix(url, "/v1/chat/completions") {
		if strings.HasSuffix(url, "/") {
			url = url + "v1/chat/completions"
		} else {
			url = url + "/v1/chat/completions"
		}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return ModelTestResult{
			Model:        modelName,
			Success:      false,
			ResponseTime: time.Since(start).Milliseconds(),
			ErrorMsg:     fmt.Sprintf("create request failed: %v", err),
		}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return ModelTestResult{
			Model:        modelName,
			Success:      false,
			ResponseTime: time.Since(start).Milliseconds(),
			ErrorMsg:     fmt.Sprintf("request failed: %v", err),
		}
	}
	defer resp.Body.Close()

	responseTime := time.Since(start).Milliseconds()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ModelTestResult{
			Model:        modelName,
			Success:      false,
			ResponseTime: responseTime,
			ErrorMsg:     fmt.Sprintf("read response failed: %v", err),
		}
	}

	// 判断成功条件：HTTP 200 且响应体不为空
	success := resp.StatusCode == http.StatusOK && len(body) > 0

	var errorMsg string
	if !success {
		if resp.StatusCode != http.StatusOK {
			errorMsg = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
		} else {
			errorMsg = "empty response"
		}
	}

	return ModelTestResult{
		Model:        modelName,
		Success:      success,
		ResponseTime: responseTime,
		ErrorMsg:     errorMsg,
	}
}
