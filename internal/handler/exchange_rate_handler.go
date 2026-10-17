package handler

import (
	"github.com/bytedance/gopkg/util/logger"
	"github.com/gin-gonic/gin"

	"Xin-api/internal/form"
	"Xin-api/internal/middleware"
	"Xin-api/internal/model"
	"Xin-api/internal/store/postgresql"
	"Xin-api/pkg/response"
)

// ExchangeRateHandler 汇率配置管理处理器（仅管理员，权限由路由中间件保证）
type ExchangeRateHandler struct {
	repo postgresql.ExchangeRateRepo
}

// NewExchangeRateHandler 创建汇率配置处理器
func NewExchangeRateHandler(repo postgresql.ExchangeRateRepo) *ExchangeRateHandler {
	return &ExchangeRateHandler{repo: repo}
}

// GetUSDCNY 获取 USD → CNY 汇率
func (h *ExchangeRateHandler) GetUSDCNY(c *gin.Context) {
	ctx := c.Request.Context()

	rate, err := h.repo.Get(ctx, model.CurrencyUSD, model.CurrencyCNY)
	if err != nil {
		logger.Error(ctx, "get usd-cny exchange rate failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	if rate == nil {
		response.AdminFail(c, response.NotFound)
		return
	}

	response.AdminSuccess(c, toExchangeRateResponse(rate))
}

// UpdateUSDCNY 人工更新 USD → CNY 汇率
func (h *ExchangeRateHandler) UpdateUSDCNY(c *gin.Context) {
	var req form.UpdateExchangeRateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn(c.Request.Context(), "bind request failed: %v", err)
		response.AdminFail(c, response.InvalidParams)
		return
	}

	// 汇率必须为正数，且最多 8 位小数（与 numeric(20,8) 精度一致）
	if !req.Rate.IsPositive() || !req.Rate.Sub(req.Rate.Round(8)).IsZero() {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	// 从上下文获取当前管理员 ID（JWT 数字解析为 float64），缺失时留空
	var updatedBy *int64
	if v, ok := c.Get(middleware.CtxUserIDKey); ok {
		if f, ok := v.(float64); ok {
			uid := int64(f)
			updatedBy = &uid
		}
	}

	ctx := c.Request.Context()
	if err := h.repo.Upsert(ctx, &model.ExchangeRate{
		FromCurrency: model.CurrencyUSD,
		ToCurrency:   model.CurrencyCNY,
		Rate:         req.Rate,
		Source:       model.ExchangeRateSourceManual,
		UpdatedBy:    updatedBy,
	}); err != nil {
		logger.Error(ctx, "update usd-cny exchange rate failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	// 重新查询，返回数据库中的最新值（updated_at 由仓储层自动维护）
	rate, err := h.repo.Get(ctx, model.CurrencyUSD, model.CurrencyCNY)
	if err != nil {
		logger.Error(ctx, "get usd-cny exchange rate failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	if rate == nil {
		response.AdminFail(c, response.NotFound)
		return
	}

	response.AdminSuccess(c, toExchangeRateResponse(rate))
}

// toExchangeRateResponse 组装汇率响应
func toExchangeRateResponse(rate *model.ExchangeRate) form.ExchangeRateResponse {
	return form.ExchangeRateResponse{
		FromCurrency: rate.FromCurrency,
		ToCurrency:   rate.ToCurrency,
		Rate:         rate.Rate,
		Source:       rate.Source,
		UpdatedAt:    rate.UpdatedAt,
	}
}
