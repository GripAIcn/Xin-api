package model

import "time"

// 汇率来源枚举，对应 exchange_rates.source
const (
	ExchangeRateSourceManual = "manual" // 人工维护
	ExchangeRateSourceAuto   = "auto"   // 自动同步
)

// 币种常量
const (
	CurrencyUSD = "USD"
	CurrencyCNY = "CNY"
)

// ExchangeRate 汇率配置，对应 exchange_rates 表
type ExchangeRate struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	FromCurrency string    `gorm:"column:from_currency;type:varchar(16);not null;uniqueIndex:idx_exchange_rates_pair,priority:1;comment:源币种" json:"from_currency"`
	ToCurrency   string    `gorm:"column:to_currency;type:varchar(16);not null;uniqueIndex:idx_exchange_rates_pair,priority:2;comment:目标币种" json:"to_currency"`
	Rate         Money     `gorm:"column:rate;type:numeric(20,8);not null;check:chk_exchange_rates_rate_positive,rate > 0;comment:汇率(1 源币种 = rate 目标币种)" json:"rate"`
	Source       string    `gorm:"column:source;type:varchar(16);not null;default:manual;comment:来源 manual-人工 auto-自动" json:"source"`
	UpdatedBy    *int64    `gorm:"column:updated_by;comment:最后更新管理员" json:"updated_by,omitempty"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}
