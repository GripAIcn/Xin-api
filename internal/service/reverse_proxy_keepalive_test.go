package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"Xin-api/internal/adapter"
	"Xin-api/internal/model"
)

// fakeAdapter 透传上游 SSE 块，便于聚焦代理自身的保活/超时行为
type fakeAdapter struct{}

func (fakeAdapter) Name() string                        { return "fake" }
func (fakeAdapter) ExtractModel([]byte) (string, error) { return "fake-model", nil }
func (fakeAdapter) BuildUpstreamRequest(body []byte, _, _ string) (*adapter.UpstreamRequest, error) {
	return &adapter.UpstreamRequest{Method: http.MethodPost, Endpoint: "/v1/chat/completions", Body: body}, nil
}
func (fakeAdapter) ConvertResponse(body []byte, _ string) ([]byte, error) { return body, nil }
func (fakeAdapter) ConvertStreamChunk(chunk []byte, _ string) ([]byte, error) {
	return chunk, nil
}
func (fakeAdapter) UpstreamModelsRequest(string) *adapter.UpstreamRequest { return nil }

// fakeEncoder 模拟 chat 入站协议：恒等编码 + SSE 注释保活
type fakeEncoder struct{ keepalive []byte }

func (e fakeEncoder) EncodeResponse(b []byte) ([]byte, error)    { return b, nil }
func (e fakeEncoder) EncodeStreamChunk(b []byte) ([]byte, error) { return b, nil }
func (e fakeEncoder) KeepaliveEvent() []byte                     { return e.keepalive }

var pingEncoder = fakeEncoder{keepalive: []byte(": ping\n\n")}

// TestMergeMetaUsageFallback 验证 usage 兜底：输入侧为 0 时不标记 HasUsage（计费回落估算），
// 但输出用量仍保留；输入侧存在时正常采集缓存明细
func TestMergeMetaUsageFallback(t *testing.T) {
	var outputOnly ForwardResult
	outputOnly.mergeMeta(parseOpenAIMeta([]byte(
		`{"usage":{"prompt_tokens":0,"completion_tokens":7,"total_tokens":7}}`)), false)
	if outputOnly.HasUsage {
		t.Fatal("输入侧为 0 时不应标记 HasUsage")
	}
	if outputOnly.OutputTokens != 7 {
		t.Fatalf("output tokens = %d, want 7", outputOnly.OutputTokens)
	}

	var full ForwardResult
	full.mergeMeta(parseOpenAIMeta([]byte(
		`{"usage":{"prompt_tokens":160,"completion_tokens":5,"total_tokens":165,`+
			`"prompt_tokens_details":{"cached_tokens":100,"cache_creation_tokens":50}}}`)), false)
	if !full.HasUsage || full.InputTokens != 160 || full.OutputTokens != 5 {
		t.Fatalf("usage = %+v", full)
	}
	if full.CachedInputTokens != 100 || full.CacheCreationTokens != 50 {
		t.Fatalf("cache details = %+v", full)
	}
}

// closeNotifyRecorder 让 httptest.ResponseRecorder 满足 gin 依赖的 http.CloseNotifier
// （gin.Context.Stream 会做该断言，真实 net/http 响应writer 已实现）
type closeNotifyRecorder struct {
	*httptest.ResponseRecorder
	ch chan bool
}

func (r *closeNotifyRecorder) CloseNotify() <-chan bool { return r.ch }

// newTestContext 构造带 Request 的测试上下文（Forward 依赖 Request.Context）
func newTestContext() (*gin.Context, *closeNotifyRecorder) {
	gin.SetMode(gin.TestMode)
	w := &closeNotifyRecorder{ResponseRecorder: httptest.NewRecorder(), ch: make(chan bool)}
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader("{}"))
	return c, w
}

func testChannel(srv *httptest.Server) *model.Channel {
	return &model.Channel{BaseURL: srv.URL}
}

func testUpstreamRequest() *adapter.UpstreamRequest {
	return &adapter.UpstreamRequest{Method: http.MethodPost, Endpoint: "/v1/chat/completions", Body: []byte("{}")}
}

func dataChunk(text string) string {
	return `data: {"id":"c1","choices":[{"delta":{"content":"` + text + `"},"finish_reason":null}]}` + "\n\n"
}

// TestForwardStreamKeepaliveNotCountedAsFirstToken 静默期下发保活，且保活不计入 TTFT、不落入响应原文采集
func TestForwardStreamKeepaliveNotCountedAsFirstToken(t *testing.T) {
	silent := 200 * time.Millisecond
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		time.Sleep(silent) // 模拟上游思考期：有响应头但无任何事件
		_, _ = w.Write([]byte(dataChunk("hi")))
		w.(http.Flusher).Flush()
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer srv.Close()

	p := NewStreamProxy(5*time.Second, 50*time.Millisecond, 5*time.Second)
	c, w := newTestContext()
	res, err := p.Forward(c, testChannel(srv), testUpstreamRequest(), fakeAdapter{}, "fake-model", pingEncoder, true)
	if err != nil {
		t.Fatalf("Forward 返回错误: %v", err)
	}

	body := w.Body.String()
	if !strings.Contains(body, ": ping") {
		t.Fatalf("静默期未下发保活注释, body=%q", body)
	}
	if !strings.Contains(body, "hi") {
		t.Fatalf("未下发语义 chunk, body=%q", body)
	}
	if res.TTFTMs < silent.Milliseconds() {
		t.Fatalf("保活被误计为首 token: TTFT=%dms, 期望 >= %dms", res.TTFTMs, silent.Milliseconds())
	}
	if strings.Contains(string(res.ResponseBody), ": ping") {
		t.Fatalf("保活字节不应落入响应原文采集: %q", string(res.ResponseBody))
	}
	if !strings.Contains(string(res.ResponseBody), "hi") {
		t.Fatalf("语义内容应落入响应原文采集: %q", string(res.ResponseBody))
	}
}

// TestForwardStreamKeepaliveDisabled 间隔为 0 时不发任何保活，行为与改造前一致
func TestForwardStreamKeepaliveDisabled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		time.Sleep(150 * time.Millisecond)
		_, _ = w.Write([]byte(dataChunk("hi")))
	}))
	defer srv.Close()

	p := NewStreamProxy(5*time.Second, 0, 5*time.Second)
	c, w := newTestContext()
	if _, err := p.Forward(c, testChannel(srv), testUpstreamRequest(), fakeAdapter{}, "fake-model", pingEncoder, false); err != nil {
		t.Fatalf("Forward 返回错误: %v", err)
	}
	if strings.Contains(w.Body.String(), ": ping") {
		t.Fatalf("禁用保活后不应出现心跳: %q", w.Body.String())
	}
}

// TestForwardStreamUpstreamCommentNotForwarded 上游注释型保活不下发、不计入 TTFT
func TestForwardStreamUpstreamCommentNotForwarded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		_, _ = w.Write([]byte(": upstream-keepalive\n\n")) // 上游注释型保活
		w.(http.Flusher).Flush()
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte(dataChunk("hi")))
	}))
	defer srv.Close()

	p := NewStreamProxy(5*time.Second, 0, 5*time.Second)
	c, w := newTestContext()
	res, err := p.Forward(c, testChannel(srv), testUpstreamRequest(), fakeAdapter{}, "fake-model", pingEncoder, false)
	if err != nil {
		t.Fatalf("Forward 返回错误: %v", err)
	}
	if strings.Contains(w.Body.String(), "upstream-keepalive") {
		t.Fatalf("上游注释型保活不应透传给客户端: %q", w.Body.String())
	}
	if res.TTFTMs < 100 {
		t.Fatalf("上游注释不应被计为首 token: TTFT=%dms", res.TTFTMs)
	}
}

// TestForwardStreamIdleTimeout 上游静默超过 idle 上限：主动结束流并归因 upstream
func TestForwardStreamIdleTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-r.Context().Done() // 保持连接但不发任何事件
	}))
	defer srv.Close()

	idle := 150 * time.Millisecond
	p := NewStreamProxy(5*time.Second, 50*time.Millisecond, idle)
	c, w := newTestContext()
	start := time.Now()
	res, err := p.Forward(c, testChannel(srv), testUpstreamRequest(), fakeAdapter{}, "fake-model", pingEncoder, false)
	elapsed := time.Since(start)

	var te *ErrUpstreamTimeout
	if !errors.As(err, &te) || te.Stage != "stream_idle" {
		t.Fatalf("期望 stream_idle 超时错误, got %v", err)
	}
	if elapsed < idle {
		t.Fatalf("提前结束: 耗时 %v, 期望 >= %v", elapsed, idle)
	}
	if !res.IsCancelled || res.CancelledBy != "upstream" {
		t.Fatalf("静默超时应归因 upstream: cancelled=%v by=%q", res.IsCancelled, res.CancelledBy)
	}
	if !strings.Contains(w.Body.String(), ": ping") {
		t.Fatalf("静默期应先下发保活: %q", w.Body.String())
	}
}

// TestForwardNonStreamBodyReadTimeout 非流式：上游只回 header 不回 body，按 requestTimeout 兜底
func TestForwardNonStreamBodyReadTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()

	timeout := 150 * time.Millisecond
	p := NewStreamProxy(timeout, 0, 0)
	c, _ := newTestContext()
	res, err := p.Forward(c, testChannel(srv), testUpstreamRequest(), fakeAdapter{}, "fake-model", pingEncoder, false)

	var te *ErrUpstreamTimeout
	if !errors.As(err, &te) || te.Stage != "body" {
		t.Fatalf("期望 body 读取超时错误, got %v", err)
	}
	if res.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("超时状态码 = %d, 期望 504", res.StatusCode)
	}
}

// TestForwardHeaderTimeout 上游首字节（响应头）超时：按分层超时归因并返回 504
func TestForwardHeaderTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(400 * time.Millisecond) // 超过 ResponseHeaderTimeout 才回 header
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := NewStreamProxy(150*time.Millisecond, 0, 0)
	c, _ := newTestContext()
	res, err := p.Forward(c, testChannel(srv), testUpstreamRequest(), fakeAdapter{}, "fake-model", pingEncoder, false)

	var te *ErrUpstreamTimeout
	if !errors.As(err, &te) || te.Stage != "header" {
		t.Fatalf("期望 header 超时错误, got %v", err)
	}
	if res.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("超时状态码 = %d, 期望 504", res.StatusCode)
	}
}

// TestResetTimerNilSafe 看门狗禁用（nil）时重置不应 panic
func TestResetTimerNilSafe(t *testing.T) {
	resetTimer(nil, time.Second)
	if !isTimeoutErr(context.DeadlineExceeded) {
		t.Fatal("context.DeadlineExceeded 应判定为超时")
	}
	if isTimeoutErr(nil) || isTimeoutErr(errors.New("boom")) {
		t.Fatal("非超时错误不应判定为超时")
	}
}
