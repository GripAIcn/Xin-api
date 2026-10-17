package form

import (
	"time"

	"Xin-api/internal/model"
)

// UpdateExchangeRateReq 更新汇率请求入参
// 汇率正数与精度校验在 handler 中完成。
type UpdateExchangeRateReq struct {
	Rate model.Money `json:"rate"` // 汇率（1 源币种 = rate 目标币种）
}

// ExchangeRateResponse 汇率配置响应
type ExchangeRateResponse struct {
	FromCurrency string      `json:"from_currency"` // 源币种
	ToCurrency   string      `json:"to_currency"`   // 目标币种
	Rate         model.Money `json:"rate"`          // 汇率
	Source       string      `json:"source"`        // 来源 manual-人工 auto-自动
	UpdatedAt    time.Time   `json:"updated_at"`    // 最后更新时间
}
