package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"Xin-api/internal/model"
	"Xin-api/internal/store/postgresql"
)

// stubApiKeyRepo 是 postgresql.ApiKeyRepo 的测试替身：
// 仅对 sk-test 的哈希返回一条启用中的密钥，其余查询返回 nil。
type stubApiKeyRepo struct{}

func (s *stubApiKeyRepo) Create(ctx context.Context, apiKey *model.ApiKey) error {
	return nil
}

func (s *stubApiKeyRepo) DeleteSoftByUser(ctx context.Context, id int64, userID int64) error {
	return nil
}

func (s *stubApiKeyRepo) GetByKeyHash(ctx context.Context, keyHash string) (*model.ApiKey, error) {
	if keyHash == model.HashApiKey("sk-test") {
		return &model.ApiKey{
			ID:      7,
			KeyHash: keyHash,
			Name:    "测试密钥",
			UserID:  42,
			Status:  model.ApiKeyStatusActive,
		}, nil
	}
	return nil, nil
}

func (s *stubApiKeyRepo) ListByUserID(ctx context.Context, userID int64) ([]model.ApiKey, error) {
	return nil, nil
}

// 确保 stub 满足仓储接口契约
var _ postgresql.ApiKeyRepo = (*stubApiKeyRepo)(nil)

// newTestContext 构造带有指定请求头与 URL 的 Gin 测试上下文
func newTestContext(target string, headers map[string]string) *gin.Context {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, target, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	c.Request = req
	return c
}

// TestExtractDataPlaneKeyPriority 验证数据面 Key 提取优先级
func TestExtractDataPlaneKeyPriority(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 1. 仅 x-api-key 头 → 返回该 key
	c := newTestContext("/v1/messages", map[string]string{"x-api-key": "only-apikey"})
	if got := extractDataPlaneKey(c); got != "only-apikey" {
		t.Fatalf("仅 x-api-key 时应返回该 key，期望 only-apikey，实际 %q", got)
	}

	// 2. Authorization Bearer 与 x-api-key 同时存在 → Authorization 优先
	c = newTestContext("/v1/messages", map[string]string{
		"Authorization": "Bearer abc",
		"x-api-key":     "xyz",
	})
	if got := extractDataPlaneKey(c); got != "abc" {
		t.Fatalf("Authorization 应优先于 x-api-key，期望 abc，实际 %q", got)
	}

	// 3. 仅 x-goog-api-key 头 → 返回该 key
	c = newTestContext("/v1/messages", map[string]string{"x-goog-api-key": "goog-key"})
	if got := extractDataPlaneKey(c); got != "goog-key" {
		t.Fatalf("仅 x-goog-api-key 时应返回该 key，期望 goog-key，实际 %q", got)
	}

	// 4. 不带任何 key → 返回空串
	c = newTestContext("/v1/messages", nil)
	if got := extractDataPlaneKey(c); got != "" {
		t.Fatalf("无任何凭证时期望返回空串，实际 %q", got)
	}

	// 5. scheme 大小写不敏感，且容忍首尾空白/换行
	c = newTestContext("/v1/messages", map[string]string{"Authorization": "bearer  sk-test  "})
	if got := extractDataPlaneKey(c); got != "sk-test" {
		t.Fatalf("应容忍 scheme 大小写与首尾空白，期望 sk-test，实际 %q", got)
	}
	c = newTestContext("/v1/messages", map[string]string{"x-api-key": " sk-test\n"})
	if got := extractDataPlaneKey(c); got != "sk-test" {
		t.Fatalf("应裁剪 x-api-key 首尾空白，期望 sk-test，实际 %q", got)
	}
}

// TestApiKeyAuthMessagesAnthropicError 验证 /v1/messages 入站的 Anthropic 错误体与鉴权成功路径
func TestApiKeyAuthMessagesAnthropicError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 用于捕获末端 handler 中读到的用户 ID
	var gotUserID int64

	router := gin.New()
	router.Use(ApiKeyAuth(&stubApiKeyRepo{}, nil))
	router.POST("/v1/messages", func(c *gin.Context) {
		// ApiKeyAuth 注入的是 int64，故直接断言为 int64（c.GetInt 仅识别 int）
		if v, ok := c.Get(CtxUserIDKey); ok {
			gotUserID, _ = v.(int64)
		}
		c.Status(http.StatusOK)
	})

	// 1. 不带任何 key → 401 且返回 Anthropic 错误体
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("无 key 时期望 401，实际 %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"type":"error"`) {
		t.Fatalf("期望 Anthropic 错误体含 \"type\":\"error\"，实际响应 %s", body)
	}
	if !strings.Contains(body, "authentication_error") {
		t.Fatalf("期望 Anthropic 错误体含 authentication_error，实际响应 %s", body)
	}

	// 2. 带有效 x-api-key: sk-test → 200 且上下文注入 UserID=42
	gotUserID = 0
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	req.Header.Set("x-api-key", "sk-test")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("有效 key 时期望 200，实际 %d，响应 %s", w.Code, w.Body.String())
	}
	if gotUserID != 42 {
		t.Fatalf("期望上下文 UserID 为 42，实际 %d", gotUserID)
	}
}

// TestApiKeyAuthChatKeepsOpenAIError 验证非 /v1/messages 入站仍返回 OpenAI 错误结构
func TestApiKeyAuthChatKeepsOpenAIError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(ApiKeyAuth(&stubApiKeyRepo{}, nil))
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("无 key 时期望 401，实际 %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"error"`) {
		t.Fatalf("期望 OpenAI 错误结构含 \"error\"，实际响应 %s", body)
	}
	if !strings.Contains(body, "invalid_api_key") {
		t.Fatalf("期望 OpenAI 错误结构含 invalid_api_key，实际响应 %s", body)
	}
}
