package service

import (
	"bufio"
	"bytes"
	"encoding/json"
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

// ForwardResult 单次转发的上游信息快照
type ForwardResult struct {
	StatusCode          int    // 上游/本地 HTTP 状态码
	UpstreamReqID       string // 上游请求 ID
	InputTokens         int64
	OutputTokens        int64
	CachedInputTokens   int64
	CacheCreationTokens int64
	ReasoningTokens     int64
	HasUsage            bool  // 是否成功解析到上游 usage
	TTFTMs              int64 // 首 token 延迟（毫秒）
	LatencyMs           int64 // 总耗时（毫秒）
	FinishReason        string
	IsStream            bool
	IsCancelled         bool
	CancelledBy         string // "client" 或 "upstream"
	ResponseBody        []byte // 转换后写回客户端的响应原文（仅在开启采集时填充）
}

// maxCapturedResponseBytes 单次请求最多保留的响应原文字节数，避免流式响应无限膨胀
const maxCapturedResponseBytes = 1 << 20

// NewStreamProxy 创建代理
func NewStreamProxy(timeout time.Duration) *StreamProxy {
	return &StreamProxy{
		client: &http.Client{Timeout: timeout},
	}
}

// Forward 使用适配器构造的上游请求转发到渠道，并按需把响应转换回 OpenAI 格式
// upReq 由适配器 BuildUpstreamRequest 产出；adpt 用于响应转换；externalModel 为对外名
// 无论成功或失败都尽量返回非 nil 的 *ForwardResult；err 非 nil 表示转发失败或上游非 200
func (p *StreamProxy) Forward(
	c *gin.Context,
	channel *model.Channel,
	upReq *adapter.UpstreamRequest,
	adpt adapter.ProviderAdapter,
	externalModel string,
	captureResponseBody bool,
) (*ForwardResult, error) {
	start := time.Now()
	upstreamURL := strings.TrimRight(channel.BaseURL, "/") + upReq.Endpoint

	req, err := http.NewRequestWithContext(c.Request.Context(), upReq.Method, upstreamURL, bytes.NewReader(upReq.Body))
	if err != nil {
		return &ForwardResult{StatusCode: http.StatusInternalServerError}, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range upReq.Headers {
		req.Header.Set(k, v)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return &ForwardResult{StatusCode: http.StatusBadGateway, LatencyMs: time.Since(start).Milliseconds()}, err
	}
	defer resp.Body.Close()

	// 非 200 状态码视为上游错误；不在此处写出响应，交由调用方决定是否重试/透传
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		result := &ForwardResult{
			StatusCode:    resp.StatusCode,
			UpstreamReqID: extractUpstreamReqID(resp),
			LatencyMs:     time.Since(start).Milliseconds(),
		}
		if captureResponseBody {
			result.ResponseBody = respBody
		}
		return result, &UpstreamError{StatusCode: resp.StatusCode, Body: respBody}
	}

	// 流式：上游返回 SSE 时逐事件转换下发
	if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		return p.forwardStream(c, resp, adpt, externalModel, captureResponseBody)
	}

	// 非流式：读取完整响应并转换
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ForwardResult{
			StatusCode:    http.StatusOK,
			UpstreamReqID: extractUpstreamReqID(resp),
			LatencyMs:     time.Since(start).Milliseconds(),
		}, err
	}
	converted, err := adpt.ConvertResponse(respBody, externalModel)
	if err != nil {
		return &ForwardResult{
			StatusCode:    http.StatusOK,
			UpstreamReqID: extractUpstreamReqID(resp),
			LatencyMs:     time.Since(start).Milliseconds(),
		}, err
	}
	c.Data(http.StatusOK, "application/json", converted)

	result := &ForwardResult{
		StatusCode:    http.StatusOK,
		UpstreamReqID: extractUpstreamReqID(resp),
		LatencyMs:     time.Since(start).Milliseconds(),
	}
	if captureResponseBody {
		result.ResponseBody = converted
	}
	// 解析转换后的 OpenAI 格式 body（字段缺失不影响结果，仅作为兜底/计费信息）
	result.mergeMeta(parseOpenAIMeta(converted), false)
	return result, nil
}

// forwardStream 逐 SSE 事件读取上游流，经适配器转换后写回客户端，并采集转发信息
func (p *StreamProxy) forwardStream(c *gin.Context, resp *http.Response, adpt adapter.ProviderAdapter, externalModel string, captureResponseBody bool) (*ForwardResult, error) {
	start := time.Now()
	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	result := &ForwardResult{
		StatusCode:    http.StatusOK,
		UpstreamReqID: extractUpstreamReqID(resp),
		IsStream:      true,
	}

	reader := bufio.NewReader(resp.Body)
	var firstChunkWritten bool
	var writeAborted bool
	var readFailed bool
	var captured bytes.Buffer

	c.Stream(func(w io.Writer) bool {
		event, readErr := readSSEEvent(reader)
		if len(event) > 0 {
			converted, convErr := adpt.ConvertStreamChunk(event, externalModel)
			if convErr != nil {
				return false
			}
			if len(converted) > 0 {
				if !firstChunkWritten {
					firstChunkWritten = true
					result.TTFTMs = time.Since(start).Milliseconds()
				}
				// 解析转换后的 chunk 元信息（最后一次非空值生效）
				result.mergeMeta(parseOpenAIMeta(sseDataPayload(converted)), true)
				// 按上限累积转换后的响应原文（用于 request_payloads）
				if captureResponseBody && captured.Len() < maxCapturedResponseBytes {
					remaining := maxCapturedResponseBytes - captured.Len()
					if len(converted) <= remaining {
						captured.Write(converted)
					} else {
						captured.Write(converted[:remaining])
					}
				}
				if _, writeErr := w.Write(converted); writeErr != nil {
					writeAborted = true
					return false
				}
			}
		}
		if readErr != nil && readErr != io.EOF {
			readFailed = true
		}
		return readErr == nil
	})

	result.LatencyMs = time.Since(start).Milliseconds()
	if captureResponseBody && captured.Len() > 0 {
		result.ResponseBody = captured.Bytes()
	}

	switch {
	case c.Request.Context().Err() != nil, writeAborted:
		result.IsCancelled = true
		result.CancelledBy = "client"
	case readFailed:
		result.IsCancelled = true
		result.CancelledBy = "upstream"
	}

	return result, nil
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

// extractUpstreamReqID 从响应头中提取上游请求 ID（不区分大小写）
func extractUpstreamReqID(resp *http.Response) string {
	if resp == nil {
		return ""
	}
	for _, key := range []string{"x-request-id", "request-id", "x-upstream-request-id", "openai-request-id"} {
		if v := resp.Header.Get(key); v != "" {
			return v
		}
	}
	return ""
}

// sseDataPayload 提取 SSE 块中首个 data: 行的负载（无则返回 nil）
func sseDataPayload(chunk []byte) []byte {
	for _, line := range bytes.Split(chunk, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) >= 5 && string(line[:5]) == "data:" {
			return bytes.TrimSpace(line[5:])
		}
	}
	return nil
}

// openaiUsage 转换后（OpenAI 格式）响应中的 usage 字段
type openaiUsage struct {
	PromptTokens        int64 `json:"prompt_tokens"`
	CompletionTokens    int64 `json:"completion_tokens"`
	PromptTokensDetails *struct {
		CachedTokens int64 `json:"cached_tokens"`
	} `json:"prompt_tokens_details"`
	CompletionTokensDetails *struct {
		ReasoningTokens int64 `json:"reasoning_tokens"`
	} `json:"completion_tokens_details"`
}

// openaiMeta 转换后（OpenAI 格式）响应/流式 chunk 中与计费相关的元信息
type openaiMeta struct {
	ID      string       `json:"id"`
	Usage   *openaiUsage `json:"usage"`
	Choices []struct {
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// parseOpenAIMeta 解析 OpenAI 格式 JSON（响应体或流式负载）；解析失败返回 nil
func parseOpenAIMeta(data []byte) *openaiMeta {
	if len(data) == 0 {
		return nil
	}
	var meta openaiMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil
	}
	return &meta
}

// mergeMeta 把解析到的元信息合并进结果；overrideID 为真时用 body 内 id 覆盖已有值
func (r *ForwardResult) mergeMeta(meta *openaiMeta, overrideID bool) {
	if meta == nil {
		return
	}
	if meta.ID != "" && (overrideID || r.UpstreamReqID == "") {
		r.UpstreamReqID = meta.ID
	}
	if meta.Usage != nil {
		r.HasUsage = true
		r.InputTokens = meta.Usage.PromptTokens
		r.OutputTokens = meta.Usage.CompletionTokens
		if meta.Usage.PromptTokensDetails != nil {
			r.CachedInputTokens = meta.Usage.PromptTokensDetails.CachedTokens
		}
		if meta.Usage.CompletionTokensDetails != nil {
			r.ReasoningTokens = meta.Usage.CompletionTokensDetails.ReasoningTokens
		}
	}
	if len(meta.Choices) > 0 && meta.Choices[0].FinishReason != nil && *meta.Choices[0].FinishReason != "" {
		r.FinishReason = *meta.Choices[0].FinishReason
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
