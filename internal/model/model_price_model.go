package model

import "time"

// 价格方案状态枚举，对应 model_prices.status
const (
	ModelPriceStatusActive   = "active"   // 启用
	ModelPriceStatusDisabled = "disabled" // 禁用
)

// ModelPrice 全局价格方案，对应 model_prices 表
// 单位：currency / 1M tokens。
type ModelPrice struct {
	ID                 int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name               string    `gorm:"column:name;type:varchar(128);not null;unique;comment:价格方案名（全局唯一）" json:"name"`
	InputPrice         float64   `gorm:"column:input_price;type:numeric(20,8);not null;default:0;comment:普通输入单价" json:"input_price"`
	OutputPrice        float64   `gorm:"column:output_price;type:numeric(20,8);not null;default:0;comment:输出单价" json:"output_price"`
	CachedInputPrice   float64   `gorm:"column:cached_input_price;type:numeric(20,8);not null;default:0;comment:输入缓存命中单价" json:"cached_input_price"`
	CacheCreationPrice float64   `gorm:"column:cache_creation_price;type:numeric(20,8);not null;default:0;comment:输入缓存创建单价" json:"cache_creation_price"`
	Currency           string    `gorm:"column:currency;type:varchar(16);not null;default:USD;comment:币种" json:"currency"`
	Status             string    `gorm:"column:status;type:varchar(32);not null;default:active;comment:状态 active-启用 disabled-禁用" json:"status"`
	CreatedAt          time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}
