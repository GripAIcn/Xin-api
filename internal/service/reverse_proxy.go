package service

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"Xin-api/internal/adapter"
	"Xin-api/internal/model"
)

// StreamProxy 上游请求与响应转换代理
type StreamProxy struct {
	client *http.Client
}

// NewStreamProxy 创建代理
func NewStreamProxy(timeout time.Duration) *StreamProxy {
	return &StreamProxy{
		client: &http.Client{Timeout: timeout},
	}
}

// Forward 使用适配器构造的上游请求转发到渠道，并按需把响应转换回 OpenAI 格式
// upReq 由适配器 BuildUpstreamRequest 产出；adpt 用于响应转换；externalModel 为对外名
// 返回 nil 表示成功；返回非 nil 表示转发失败或上游返回非 200
func (p *StreamProxy) Forward(
	c *gin.Context,
	channel *model.Channel,
	upReq *adapter.UpstreamRequest,
	adpt adapter.ProviderAdapter,
	externalModel string,
) error {
	upstreamURL := strings.TrimRight(channel.BaseURL, "/") + upReq.Endpoint

	req, err := http.NewRequestWithContext(c.Request.Context(), upReq.Method, upstreamURL, bytes.NewReader(upReq.Body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range upReq.Headers {
		req.Header.Set(k, v)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 非 200 状态码视为上游错误，透传响应体
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		c.Data(resp.StatusCode, "application/json", respBody)
		return &UpstreamError{StatusCode: resp.StatusCode, Body: respBody}
	}

	// 流式：上游返回 SSE 时逐事件转换下发
	if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		return p.forwardStream(c, resp, adpt, externalModel)
	}

	// 非流式：读取完整响应并转换
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	converted, err := adpt.ConvertResponse(respBody, externalModel)
	if err != nil {
		return err
	}
	c.Data(http.StatusOK, "application/json", converted)
	return nil
}

// forwardStream 逐 SSE 事件读取上游流，经适配器转换后写回客户端
func (p *StreamProxy) forwardStream(c *gin.Context, resp *http.Response, adpt adapter.ProviderAdapter, externalModel string) error {
	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	reader := bufio.NewReader(resp.Body)
	c.Stream(func(w io.Writer) bool {
		event, readErr := readSSEEvent(reader)
		if len(event) > 0 {
			converted, convErr := adpt.ConvertStreamChunk(event, externalModel)
			if convErr != nil {
				return false
			}
			if len(converted) > 0 {
				if _, writeErr := w.Write(converted); writeErr != nil {
					return false
				}
			}
		}
		return readErr == nil
	})

	return nil
}

// readSSEEvent 读取一个完整的 SSE 事件（以空行分隔）
func readSSEEvent(r *bufio.Reader) ([]byte, error) {
	var buf bytes.Buffer
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 {
			buf.Write(line)
		}
		if err != nil {
			return buf.Bytes(), err
		}
		// 空行（仅含换行/回车）表示事件结束
		if len(bytes.TrimSpace(line)) == 0 {
			return buf.Bytes(), nil
		}
	}
}

// UpstreamError 上游返回错误
type UpstreamError struct {
	StatusCode int
	Body       []byte
}

func (e *UpstreamError) Error() string {
	return http.StatusText(e.StatusCode)
}
