package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"Xin-api/internal/adapter"
	"Xin-api/internal/model"
)

// StreamProxy 上游请求与响应转换代理
type StreamProxy struct {
	client         *http.Client
	keepaliveInt   time.Duration // 下游 SSE 空闲保活间隔，0 表示禁用
	idleTimeout    time.Duration // 流式上游静默上限（读空闲看门狗），0 表示禁用
	requestTimeout time.Duration // 上游首字节（响应头）超时，同时作为非流式响应体读取的总超时
}

// ResponseEncoder 把适配器产出的 OpenAI Chat 响应（或 SSE 块）再编码为入站协议格式
// 由入站 Codec 实现；nil 表示透传（保持 OpenAI Chat 形态）
type ResponseEncoder interface {
	EncodeResponse(openAIResponse []byte) ([]byte, error)
	EncodeStreamChunk(openAISSESChunk []byte) ([]byte, error)
	// KeepaliveEvent 流式空闲保活事件（SSE 注释或协议原生 ping）；nil 表示该协议不发送
	KeepaliveEvent() []byte
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
// requestTimeout：上游首字节（响应头）超时，同时作为非流式响应体读取的总逾时
// keepaliveInterval：下游 SSE 空闲保活间隔，0 表示禁用
// idleTimeout：流式上游静默上限（读空闲看门狗），0 表示禁用
func NewStreamProxy(requestTimeout, keepaliveInterval, idleTimeout time.Duration) *StreamProxy {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// 上游首字节超时：替代原先覆盖「建连→body 读完」整段的 Client.Timeout
		ResponseHeaderTimeout: requestTimeout,
	}
	return &StreamProxy{
		// Timeout 置 0：整段超时改由分层超时（拨号/TLS/首字节）+ 流式读空闲看门狗承担，
		// 避免长推理等“持续有输出但总时长很长”的流被整段掐断
		client:         &http.Client{Transport: transport},
		keepaliveInt:   keepaliveInterval,
		idleTimeout:    idleTimeout,
		requestTimeout: requestTimeout,
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
	encoder ResponseEncoder,
	captureResponseBody bool,
) (*ForwardResult, error) {
	start := time.Now()
	upstreamURL := joinUpstreamURL(channel.BaseURL, upReq.Endpoint)

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
		latency := time.Since(start).Milliseconds()
		// 上游首字节（响应头）超时归入超时错误，便于上层映射 504/超时统计
		if isTimeoutErr(err) {
			return &ForwardResult{StatusCode: http.StatusGatewayTimeout, LatencyMs: latency},
				&ErrUpstreamTimeout{Stage: "header", Err: err}
		}
		return &ForwardResult{StatusCode: http.StatusBadGateway, LatencyMs: latency}, err
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
		return p.forwardStream(c, resp, adpt, externalModel, encoder, captureResponseBody)
	}

	// 非流式：读取完整响应并转换
	// Client 已关闭整段超时，此处按 requestTimeout 给响应体读取单独兜底，避免上游只回 header 时挂死
	readCtx, cancelRead := context.WithTimeout(c.Request.Context(), p.requestTimeout)
	defer cancelRead()
	stopTimeout := context.AfterFunc(readCtx, func() { _ = resp.Body.Close() })
	defer stopTimeout()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		result := &ForwardResult{
			StatusCode:    http.StatusOK,
			UpstreamReqID: extractUpstreamReqID(resp),
			LatencyMs:     time.Since(start).Milliseconds(),
		}
		if readCtx.Err() != nil {
			result.StatusCode = http.StatusGatewayTimeout
			return result, &ErrUpstreamTimeout{Stage: "body", Err: err}
		}
		return result, err
	}
	converted, err := adpt.ConvertResponse(respBody, externalModel)
	if err != nil {
		return &ForwardResult{
			StatusCode:    http.StatusOK,
			UpstreamReqID: extractUpstreamReqID(resp),
			LatencyMs:     time.Since(start).Milliseconds(),
		}, err
	}

	// 写回客户端前按入站协议再编码（计费元信息仍从 OpenAI 格式字节解析）
	mout := converted
	if encoder != nil {
		if encoded, encErr := encoder.EncodeResponse(converted); encErr == nil {
			mout = encoded
		}
	}
	c.Data(http.StatusOK, "application/json", mout)

	result := &ForwardResult{
		StatusCode:    http.StatusOK,
		UpstreamReqID: extractUpstreamReqID(resp),
		LatencyMs:     time.Since(start).Milliseconds(),
	}
	if captureResponseBody {
		result.ResponseBody = mout
	}
	// 解析转换后的 OpenAI 格式 body（字段缺失不影响结果，仅作为兜底/计费信息）
	result.mergeMeta(parseOpenAIMeta(converted), false)
	return result, nil
}

// forwardStream 逐 SSE 事件读取上游流，经适配器转换后写回客户端，并采集转发信息
// 上游读取放在独立 goroutine，主循环 select 三路：上游事件 / 下游保活定时器 / 上游静默看门狗
func (p *StreamProxy) forwardStream(c *gin.Context, resp *http.Response, adpt adapter.ProviderAdapter, externalModel string, encoder ResponseEncoder, captureResponseBody bool) (*ForwardResult, error) {
	start := time.Now()
	// 先设置 SSE 响应头，再写入状态码，避免头被提前提交导致 Content-Type 丢失
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)

	result := &ForwardResult{
		StatusCode:    http.StatusOK,
		UpstreamReqID: extractUpstreamReqID(resp),
		IsStream:      true,
	}

	reader := bufio.NewReader(resp.Body)
	// stop 通知读协程退出：客户端断开 / 写出失败 / 静默超时时不再等上游。
	// Forward 返回时会关闭 resp.Body，正在阻塞的读取随之结束
	stop := make(chan struct{})
	defer close(stop)

	type readEvent struct {
		event []byte
		err   error
	}
	events := make(chan readEvent, 8)
	go func() {
		defer close(events)
		for {
			event, err := readSSEEvent(reader)
			select {
			case events <- readEvent{event: event, err: err}:
			case <-stop:
				return
			}
			if err != nil {
				return
			}
		}
	}()

	var keepalive []byte
	if encoder != nil {
		keepalive = encoder.KeepaliveEvent()
	}

	// 下游保活定时器：仅在空闲超过间隔时补发一个无语义事件，刷新反代空闲读超时
	var tickCh <-chan time.Time
	if p.keepaliveInt > 0 && len(keepalive) > 0 {
		ticker := time.NewTicker(p.keepaliveInt)
		defer ticker.Stop()
		tickCh = ticker.C
	}

	// 上游静默看门狗：每收到一个上游事件即重置，替代已关闭的整段超时
	var idleTimer *time.Timer
	var idleCh <-chan time.Time
	if p.idleTimeout > 0 {
		idleTimer = time.NewTimer(p.idleTimeout)
		defer idleTimer.Stop()
		idleCh = idleTimer.C
	}

	var firstChunkWritten bool
	var writeAborted bool
	var readFailed bool
	var idleTimeoutHit bool
	var captured bytes.Buffer
	lastWriteAt := start

	c.Stream(func(w io.Writer) bool {
		var ev readEvent
		select {
		case r, ok := <-events:
			if !ok {
				return false // 上游流已结束且事件已全部下发
			}
			resetTimer(idleTimer, p.idleTimeout)
			ev = r
		case <-tickCh:
			if time.Since(lastWriteAt) < p.keepaliveInt {
				return true // 仍有数据在流动，无需补保活
			}
			if _, writeErr := w.Write(keepalive); writeErr != nil {
				writeAborted = true
				return false
			}
			lastWriteAt = time.Now()
			return true
		case <-idleCh:
			idleTimeoutHit = true
			return false
		}

		if len(ev.event) > 0 {
			converted, convErr := adpt.ConvertStreamChunk(ev.event, externalModel)
			if convErr != nil {
				return false
			}
			// 仅语义块（携带 data 负载）参与写出/TTFT/采集：上游注释型保活既不下发也不污染指标
			if len(converted) > 0 && len(sseDataPayload(converted)) > 0 {
				// 解析转换后的 OpenAI chunk 元信息（最后一次非空值生效），必须编码前完成
				result.mergeMeta(parseOpenAIMeta(sseDataPayload(converted)), true)
				// 写回客户端前按入站协议再编码
				mout := converted
				if encoder != nil {
					if encoded, encErr := encoder.EncodeStreamChunk(converted); encErr == nil {
						mout = encoded
					}
				}
				if len(mout) > 0 {
					if !firstChunkWritten {
						firstChunkWritten = true
						result.TTFTMs = time.Since(start).Milliseconds()
					}
					// 按上限累积写回客户端的响应原文（用于 request_payloads，保活字节不落库）
					if captureResponseBody && captured.Len() < maxCapturedResponseBytes {
						remaining := maxCapturedResponseBytes - captured.Len()
						if len(mout) <= remaining {
							captured.Write(mout)
						} else {
							captured.Write(mout[:remaining])
						}
					}
					if _, writeErr := w.Write(mout); writeErr != nil {
						writeAborted = true
						return false
					}
					lastWriteAt = time.Now()
				}
			}
		}
		if ev.err != nil && ev.err != io.EOF {
			readFailed = true
		}
		return ev.err == nil
	})

	result.LatencyMs = time.Since(start).Milliseconds()
	if captureResponseBody && captured.Len() > 0 {
		result.ResponseBody = captured.Bytes()
	}

	switch {
	case c.Request.Context().Err() != nil, writeAborted:
		result.IsCancelled = true
		result.CancelledBy = "client"
	case readFailed, idleTimeoutHit:
		result.IsCancelled = true
		result.CancelledBy = "upstream"
	}

	// 上游静默超时：返回超时错误供上层做超时归因（响应已写出时上层不会再回写错误体）
	if idleTimeoutHit {
		return result, &ErrUpstreamTimeout{Stage: "stream_idle", Err: context.DeadlineExceeded}
	}

	return result, nil
}

// resetTimer 安全重置计时器：先排空可能已触发的信号，避免 Reset 后立即误触发
func resetTimer(t *time.Timer, d time.Duration) {
	if t == nil || d <= 0 {
		return
	}
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(d)
}

// isTimeoutErr 判断错误是否由超时引起（响应头超时 / context 超时 / 网络超时）
func isTimeoutErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
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

// joinUpstreamURL 拼接渠道 BaseURL 与上游端点路径
// 兼容 BaseURL 已带 /v1（如 https://relay.example.com/v1）或已含完整端点路径的写法，避免重复追加
func joinUpstreamURL(baseURL, endpoint string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	ep := "/" + strings.TrimLeft(strings.TrimSpace(endpoint), "/")
	if ep == "/" {
		ep = ""
	}
	if base == "" {
		return ep
	}
	if ep == "" {
		return base
	}
	// BaseURL 已包含完整端点，直接返回
	if strings.HasSuffix(base, ep) {
		return base
	}
	// BaseURL 已以 /v1 结尾，且端点以 /v1/ 开头时去掉端点的版本前缀
	if strings.HasSuffix(base, "/v1") && strings.HasPrefix(ep, "/v1/") {
		return base + strings.TrimPrefix(ep, "/v1")
	}
	return base + ep
}

// UpstreamError 上游返回错误
type UpstreamError struct {
	StatusCode int
	Body       []byte
}

func (e *UpstreamError) Error() string {
	return http.StatusText(e.StatusCode)
}

// ErrUpstreamTimeout 网关本地判定的上游超时（上游未按约定时限返回数据）
// 与 UpstreamError 区分：它不是上游返回的状态码，而是本地的超时归因
type ErrUpstreamTimeout struct {
	Stage string // header（响应头超时）/ body（非流式读取超时）/ stream_idle（流式静默超时）
	Err   error
}

func (e *ErrUpstreamTimeout) Error() string {
	if e.Err == nil {
		return "upstream timeout (" + e.Stage + ")"
	}
	return "upstream timeout (" + e.Stage + "): " + e.Err.Error()
}

func (e *ErrUpstreamTimeout) Unwrap() error { return e.Err }
