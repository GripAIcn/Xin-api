package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"Xin-api/internal/form"
	"Xin-api/internal/middleware"
	"Xin-api/internal/model"
	"Xin-api/pkg/response"
)

// fakeExchangeRateRepo 汇率仓储桩，隔离数据库依赖
type fakeExchangeRateRepo struct {
	getFn    func(ctx context.Context, fromCurrency, toCurrency string) (*model.ExchangeRate, error)
	upsertFn func(ctx context.Context, rate *model.ExchangeRate) error
	seedFn   func(ctx context.Context, initialRate model.Money) error

	getCalls    int
	upsertCalls []model.ExchangeRate
}

func (f *fakeExchangeRateRepo) Get(ctx context.Context, fromCurrency, toCurrency string) (*model.ExchangeRate, error) {
	f.getCalls++
	if f.getFn != nil {
		return f.getFn(ctx, fromCurrency, toCurrency)
	}
	return nil, nil
}

func (f *fakeExchangeRateRepo) Upsert(ctx context.Context, rate *model.ExchangeRate) error {
	f.upsertCalls = append(f.upsertCalls, *rate)
	if f.upsertFn != nil {
		return f.upsertFn(ctx, rate)
	}
	return nil
}

func (f *fakeExchangeRateRepo) EnsureUSDCNYSeed(ctx context.Context, initialRate model.Money) error {
	if f.seedFn != nil {
		return f.seedFn(ctx, initialRate)
	}
	return nil
}

// exchangeRateEnvelope 控制面响应结构，Data 可能因失败而缺失
type exchangeRateEnvelope struct {
	Code    int                        `json:"code"`
	Message string                     `json:"message"`
	Data    *form.ExchangeRateResponse `json:"data"`
}

// newExchangeRateTestRouter 构建仅包含汇率接口的测试路由
func newExchangeRateTestRouter(h *ExchangeRateHandler, withUserID bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	get := r.Group("/v1/exchange-rates")
	get.GET("/usd-cny", h.GetUSDCNY)

	handlers := make([]gin.HandlerFunc, 0, 2)
	if withUserID {
		handlers = append(handlers, func(c *gin.Context) {
			c.Set(middleware.CtxUserIDKey, float64(42))
		})
	}
	handlers = append(handlers, h.UpdateUSDCNY)
	get.PUT("/usd-cny", handlers...)
	return r
}

func sampleExchangeRate() *model.ExchangeRate {
	return &model.ExchangeRate{
		FromCurrency: model.CurrencyUSD,
		ToCurrency:   model.CurrencyCNY,
		Rate:         model.MoneyFromFloat(7.2),
		Source:       model.ExchangeRateSourceManual,
		UpdatedAt:    time.Date(2026, 10, 17, 10, 0, 0, 0, time.UTC),
	}
}

func performRequest(t *testing.T, r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeExchangeRateEnvelope(t *testing.T, w *httptest.ResponseRecorder) exchangeRateEnvelope {
	t.Helper()
	var env exchangeRateEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("响应解析失败: %v, body=%s", err, w.Body.String())
	}
	return env
}

func TestGetUSDCNY(t *testing.T) {
	t.Run("成功返回汇率", func(t *testing.T) {
		var gotFrom, gotTo string
		repo := &fakeExchangeRateRepo{
			getFn: func(_ context.Context, fromCurrency, toCurrency string) (*model.ExchangeRate, error) {
				gotFrom, gotTo = fromCurrency, toCurrency
				return sampleExchangeRate(), nil
			},
		}
		w := performRequest(t, newExchangeRateTestRouter(NewExchangeRateHandler(repo), false), http.MethodGet, "/v1/exchange-rates/usd-cny", "")

		if w.Code != http.StatusOK {
			t.Fatalf("HTTP 状态码 = %d, 期望 200", w.Code)
		}
		env := decodeExchangeRateEnvelope(t, w)
		if env.Code != response.Success {
			t.Fatalf("业务码 = %d, 期望 %d", env.Code, response.Success)
		}
		if env.Data == nil {
			t.Fatal("data 不应为空")
		}
		if gotFrom != model.CurrencyUSD || gotTo != model.CurrencyCNY {
			t.Fatalf("查询币种对 = %s→%s, 期望 USD→CNY", gotFrom, gotTo)
		}
		if env.Data.Rate.String() != "7.2" {
			t.Fatalf("汇率 = %s, 期望 7.2", env.Data.Rate.String())
		}
		if env.Data.Source != model.ExchangeRateSourceManual {
			t.Fatalf("来源 = %s, 期望 manual", env.Data.Source)
		}
		if !env.Data.UpdatedAt.Equal(sampleExchangeRate().UpdatedAt) {
			t.Fatalf("更新时间 = %v, 期望 %v", env.Data.UpdatedAt, sampleExchangeRate().UpdatedAt)
		}
	})

	t.Run("仓储查询失败返回内部错误", func(t *testing.T) {
		repo := &fakeExchangeRateRepo{
			getFn: func(context.Context, string, string) (*model.ExchangeRate, error) {
				return nil, errors.New("db down")
			},
		}
		w := performRequest(t, newExchangeRateTestRouter(NewExchangeRateHandler(repo), false), http.MethodGet, "/v1/exchange-rates/usd-cny", "")

		env := decodeExchangeRateEnvelope(t, w)
		if env.Code != response.InternalError {
			t.Fatalf("业务码 = %d, 期望 %d", env.Code, response.InternalError)
		}
		if env.Data != nil {
			t.Fatalf("失败响应不应包含 data, 实际 %+v", env.Data)
		}
	})

	t.Run("汇率不存在返回 NotFound", func(t *testing.T) {
		repo := &fakeExchangeRateRepo{
			getFn: func(context.Context, string, string) (*model.ExchangeRate, error) {
				return nil, nil
			},
		}
		w := performRequest(t, newExchangeRateTestRouter(NewExchangeRateHandler(repo), false), http.MethodGet, "/v1/exchange-rates/usd-cny", "")

		env := decodeExchangeRateEnvelope(t, w)
		if env.Code != response.NotFound {
			t.Fatalf("业务码 = %d, 期望 %d", env.Code, response.NotFound)
		}
	})
}

func TestUpdateUSDCNY(t *testing.T) {
	t.Run("成功更新并返回最新汇率", func(t *testing.T) {
		repo := &fakeExchangeRateRepo{
			getFn: func(context.Context, string, string) (*model.ExchangeRate, error) {
				rate := sampleExchangeRate()
				rate.Rate = model.MoneyFromFloat(7.35)
				return rate, nil
			},
		}
		w := performRequest(t, newExchangeRateTestRouter(NewExchangeRateHandler(repo), true), http.MethodPut, "/v1/exchange-rates/usd-cny", `{"rate":7.35}`)

		env := decodeExchangeRateEnvelope(t, w)
		if env.Code != response.Success {
			t.Fatalf("业务码 = %d, 期望 %d, body=%s", env.Code, response.Success, w.Body.String())
		}
		if env.Data == nil || env.Data.Rate.String() != "7.35" {
			t.Fatalf("返回汇率不符合期望: %+v", env.Data)
		}
		if len(repo.upsertCalls) != 1 {
			t.Fatalf("Upsert 调用次数 = %d, 期望 1", len(repo.upsertCalls))
		}
		saved := repo.upsertCalls[0]
		if saved.FromCurrency != model.CurrencyUSD || saved.ToCurrency != model.CurrencyCNY {
			t.Fatalf("保存币种对 = %s→%s, 期望 USD→CNY", saved.FromCurrency, saved.ToCurrency)
		}
		if saved.Source != model.ExchangeRateSourceManual {
			t.Fatalf("保存来源 = %s, 期望 manual", saved.Source)
		}
		if saved.Rate.String() != "7.35" {
			t.Fatalf("保存汇率 = %s, 期望 7.35", saved.Rate.String())
		}
		if saved.UpdatedBy == nil || *saved.UpdatedBy != 42 {
			t.Fatalf("updated_by = %v, 期望 42", saved.UpdatedBy)
		}
	})

	t.Run("上下文缺少管理员ID时 updated_by 为空", func(t *testing.T) {
		repo := &fakeExchangeRateRepo{
			getFn: func(context.Context, string, string) (*model.ExchangeRate, error) {
				return sampleExchangeRate(), nil
			},
		}
		w := performRequest(t, newExchangeRateTestRouter(NewExchangeRateHandler(repo), false), http.MethodPut, "/v1/exchange-rates/usd-cny", `{"rate":7.2}`)

		env := decodeExchangeRateEnvelope(t, w)
		if env.Code != response.Success {
			t.Fatalf("业务码 = %d, 期望 %d", env.Code, response.Success)
		}
		if len(repo.upsertCalls) != 1 {
			t.Fatalf("Upsert 调用次数 = %d, 期望 1", len(repo.upsertCalls))
		}
		if repo.upsertCalls[0].UpdatedBy != nil {
			t.Fatalf("updated_by = %v, 期望 nil", repo.upsertCalls[0].UpdatedBy)
		}
	})

	t.Run("非法请求体返回参数错误", func(t *testing.T) {
		repo := &fakeExchangeRateRepo{}
		w := performRequest(t, newExchangeRateTestRouter(NewExchangeRateHandler(repo), true), http.MethodPut, "/v1/exchange-rates/usd-cny", `{"rate":"abc"}`)

		env := decodeExchangeRateEnvelope(t, w)
		if env.Code != response.InvalidParams {
			t.Fatalf("业务码 = %d, 期望 %d", env.Code, response.InvalidParams)
		}
		if len(repo.upsertCalls) != 0 {
			t.Fatalf("参数错误时不应写库, 实际调用 %d 次", len(repo.upsertCalls))
		}
	})

	invalidRateCases := []struct {
		name string
		body string
	}{
		{name: "汇率缺失", body: `{}`},
		{name: "汇率为零", body: `{"rate":0}`},
		{name: "汇率为负", body: `{"rate":-1}`},
		{name: "超过8位小数", body: `{"rate":7.123456789}`},
	}
	for _, tc := range invalidRateCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeExchangeRateRepo{}
			w := performRequest(t, newExchangeRateTestRouter(NewExchangeRateHandler(repo), true), http.MethodPut, "/v1/exchange-rates/usd-cny", tc.body)

			env := decodeExchangeRateEnvelope(t, w)
			if env.Code != response.InvalidParams {
				t.Fatalf("业务码 = %d, 期望 %d", env.Code, response.InvalidParams)
			}
			if len(repo.upsertCalls) != 0 {
				t.Fatalf("参数错误时不应写库, 实际调用 %d 次", len(repo.upsertCalls))
			}
		})
	}

	t.Run("恰好8位小数允许保存", func(t *testing.T) {
		repo := &fakeExchangeRateRepo{
			getFn: func(context.Context, string, string) (*model.ExchangeRate, error) {
				rate := sampleExchangeRate()
				rate.Rate = model.NewMoney(decimal.RequireFromString("7.12345678"))
				return rate, nil
			},
		}
		w := performRequest(t, newExchangeRateTestRouter(NewExchangeRateHandler(repo), true), http.MethodPut, "/v1/exchange-rates/usd-cny", `{"rate":7.12345678}`)

		env := decodeExchangeRateEnvelope(t, w)
		if env.Code != response.Success {
			t.Fatalf("业务码 = %d, 期望 %d, body=%s", env.Code, response.Success, w.Body.String())
		}
		if len(repo.upsertCalls) != 1 || repo.upsertCalls[0].Rate.String() != "7.12345678" {
			t.Fatalf("保存汇率不符合期望: %+v", repo.upsertCalls)
		}
	})

	t.Run("写库失败返回内部错误", func(t *testing.T) {
		repo := &fakeExchangeRateRepo{
			upsertFn: func(context.Context, *model.ExchangeRate) error {
				return errors.New("write failed")
			},
		}
		w := performRequest(t, newExchangeRateTestRouter(NewExchangeRateHandler(repo), true), http.MethodPut, "/v1/exchange-rates/usd-cny", `{"rate":7.2}`)

		env := decodeExchangeRateEnvelope(t, w)
		if env.Code != response.InternalError {
			t.Fatalf("业务码 = %d, 期望 %d", env.Code, response.InternalError)
		}
		if repo.getCalls != 0 {
			t.Fatalf("写库失败后不应再次查询, 实际查询 %d 次", repo.getCalls)
		}
	})

	t.Run("回查失败返回内部错误", func(t *testing.T) {
		repo := &fakeExchangeRateRepo{
			getFn: func(context.Context, string, string) (*model.ExchangeRate, error) {
				return nil, errors.New("read failed")
			},
		}
		w := performRequest(t, newExchangeRateTestRouter(NewExchangeRateHandler(repo), true), http.MethodPut, "/v1/exchange-rates/usd-cny", `{"rate":7.2}`)

		env := decodeExchangeRateEnvelope(t, w)
		if env.Code != response.InternalError {
			t.Fatalf("业务码 = %d, 期望 %d", env.Code, response.InternalError)
		}
		if len(repo.upsertCalls) != 1 {
			t.Fatalf("Upsert 调用次数 = %d, 期望 1", len(repo.upsertCalls))
		}
	})

	t.Run("回查不到记录返回 NotFound", func(t *testing.T) {
		repo := &fakeExchangeRateRepo{
			getFn: func(context.Context, string, string) (*model.ExchangeRate, error) {
				return nil, nil
			},
		}
		w := performRequest(t, newExchangeRateTestRouter(NewExchangeRateHandler(repo), true), http.MethodPut, "/v1/exchange-rates/usd-cny", `{"rate":7.2}`)

		env := decodeExchangeRateEnvelope(t, w)
		if env.Code != response.NotFound {
			t.Fatalf("业务码 = %d, 期望 %d", env.Code, response.NotFound)
		}
	})
}
