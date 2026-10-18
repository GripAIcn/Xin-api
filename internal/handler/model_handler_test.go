package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"Xin-api/internal/model"
	"Xin-api/internal/store/postgresql"
)

// stubModelChannelRepo 仅实现 ListEnabledModels，其余方法沿用嵌入接口（测试中不会被调用）
type stubModelChannelRepo struct {
	postgresql.ChannelRepo
	rows []model.EnabledModelInfo
	err  error
}

func (s *stubModelChannelRepo) ListEnabledModels(ctx context.Context) ([]model.EnabledModelInfo, error) {
	return s.rows, s.err
}

func newModelListRouter(repo postgresql.ChannelRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/v1/models", NewModelHandler(repo).ListModels)
	return r
}

// TestModelHandlerListModels 验证 OpenAI 兼容的模型列表响应结构
func TestModelHandlerListModels(t *testing.T) {
	created := time.Unix(1735689600, 0).UTC()
	router := newModelListRouter(&stubModelChannelRepo{
		rows: []model.EnabledModelInfo{
			{Name: "gpt-4o", CreatedAt: created},
			{Name: "claude-3", CreatedAt: created},
		},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, 期望 200，响应 %s", w.Code, w.Body.String())
	}

	var body struct {
		Object string `json:"object"`
		Data   []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("解析响应失败: %v，响应 %s", err, w.Body.String())
	}

	if body.Object != "list" {
		t.Fatalf("object = %q, 期望 list", body.Object)
	}
	if len(body.Data) != 2 {
		t.Fatalf("模型数量 = %d, 期望 2", len(body.Data))
	}
	for i, want := range []string{"gpt-4o", "claude-3"} {
		got := body.Data[i]
		if got.ID != want {
			t.Fatalf("data[%d].id = %q, 期望 %q", i, got.ID, want)
		}
		if got.Object != "model" {
			t.Fatalf("data[%d].object = %q, 期望 model", i, got.Object)
		}
		if got.Created != 1735689600 {
			t.Fatalf("data[%d].created = %d, 期望 1735689600", i, got.Created)
		}
		if got.OwnedBy != "xin-api" {
			t.Fatalf("data[%d].owned_by = %q, 期望 xin-api", i, got.OwnedBy)
		}
	}
}

// TestModelHandlerListModelsEmpty 验证无可用模型时返回空数组
func TestModelHandlerListModelsEmpty(t *testing.T) {
	router := newModelListRouter(&stubModelChannelRepo{rows: []model.EnabledModelInfo{}})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, 期望 200，响应 %s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got != `{"object":"list","data":[]}` {
		t.Fatalf("响应 = %s, 期望空列表结构", got)
	}
}

// TestModelHandlerListModelsError 验证查询失败时返回 OpenAI 风格错误
func TestModelHandlerListModelsError(t *testing.T) {
	router := newModelListRouter(&stubModelChannelRepo{err: errors.New("db down")})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("状态码 = %d, 期望 500，响应 %s", w.Code, w.Body.String())
	}
	if got := w.Body.String(); got == "" || !json.Valid(w.Body.Bytes()) {
		t.Fatalf("期望合法 JSON 错误体，实际 %q", got)
	}
}
