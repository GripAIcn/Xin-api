package model

import "time"

// ModelChannelEntry 「渠道 × 模型」明细行（JOIN 渠道信息），供模型管理页按逻辑模型名聚合展示。
// 说明：Name 为对外（逻辑）模型名，仅用于上层按名分组，不单独序列化到明细中（分组结果见 ModelGroup.Name）。
type ModelChannelEntry struct {
	Name            string      `gorm:"column:name" json:"-"`
	ModelID         int64       `gorm:"column:model_id" json:"model_id"`
	ChannelID       int64       `gorm:"column:channel_id" json:"channel_id"`
	ChannelName     string      `gorm:"column:channel_name" json:"channel_name"`
	ChannelProvider string      `gorm:"column:channel_provider" json:"channel_provider"`
	ChannelStatus   string      `gorm:"column:channel_status" json:"channel_status"`
	UpstreamModel   string      `gorm:"column:upstream_model" json:"upstream_model"`
	Status          string      `gorm:"column:status" json:"status"`
	ContextLength   *int        `gorm:"column:context_length" json:"context_length,omitempty"`
	MaxOutputTokens *int        `gorm:"column:max_output_tokens" json:"max_output_tokens,omitempty"`
	Capabilities    StringArray `gorm:"column:capabilities" json:"capabilities"`
	ModelPriceID    *int64      `gorm:"column:model_price_id" json:"model_price_id,omitempty"`
	PriceName       *string     `gorm:"column:price_name" json:"price_name,omitempty"`
}

// ChannelModelView 渠道下的模型明细（LEFT JOIN 价格方案绑定）。
// 字段与 models 表一致，额外带出当前绑定的 model_price_id / price_name，供渠道管理页展示与编辑预选。
type ChannelModelView struct {
	ID              int64       `gorm:"column:id" json:"id"`
	ChannelID       int64       `gorm:"column:channel_id" json:"channel_id"`
	Name            string      `gorm:"column:name" json:"name"`
	UpstreamModel   string      `gorm:"column:upstream_model" json:"upstream_model"`
	ContextLength   *int        `gorm:"column:context_length" json:"context_length,omitempty"`
	MaxOutputTokens *int        `gorm:"column:max_output_tokens" json:"max_output_tokens,omitempty"`
	Capabilities    StringArray `gorm:"column:capabilities" json:"capabilities"`
	Status          string      `gorm:"column:status" json:"status"`
	ModelPriceID    *int64      `gorm:"column:model_price_id" json:"model_price_id,omitempty"`
	PriceName       *string     `gorm:"column:price_name" json:"price_name,omitempty"`
	CreatedAt       time.Time   `gorm:"column:created_at" json:"created_at"`
	UpdatedAt       time.Time   `gorm:"column:updated_at" json:"updated_at"`
}

// ModelGroup 按逻辑模型名聚合后的分组（同一 name 下的全部渠道明细）
type ModelGroup struct {
	Name         string              `json:"name"`
	ChannelCount int                 `json:"channel_count"`
	ActiveCount  int                 `json:"active_count"`
	Channels     []ModelChannelEntry `json:"channels"`
}
