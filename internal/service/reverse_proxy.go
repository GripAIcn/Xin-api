package service

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"Xin-api/internal/model"
)

// StreamProxy SSE 流式代理
type StreamProxy struct {
	client *http.Client
}

// NewStreamProxy 创建流式代理
func NewStreamProxy(timeout time.Duration) *StreamProxy {
	return &StreamProxy{
		client: &http.Client{Timeout: timeout},
	}
}

// Forward 转发请求到上游渠道
// body 为原始请求体 JSON 字节，endpoint 为上游 API 路径（由 adapter 提供）
// 返回 nil 表示成功；返回非 nil 表示转发失败或上游返回非 200
func (p *StreamProxy) Forward(c *gin.Context, channel *model.Channel, body []byte, endpoint string) error {
	upstreamURL := channel.BaseURL + endpoint

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, upstreamURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+channel.APIKey)

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

	// SSE 流式转发
	c.Status(http.StatusOK)
	c.Header("Content-Type", resp.Header.Get("Content-Type"))
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	c.Stream(func(w io.Writer) bool {
		buf := make([]byte, 4096)
		n, _ := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := w.Write(buf[:n])
			if writeErr != nil {
				return false
			}
			return true
		}
		return false
	})

	return nil
}

// UpstreamError 上游返回错误
type UpstreamError struct {
	StatusCode int
	Body       []byte
}

func (e *UpstreamError) Error() string {
	return http.StatusText(e.StatusCode)
}
