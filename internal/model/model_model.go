package model

import (
	"time"

	"gorm.io/gorm"
)

// 模型状态枚举，对应 models.status
const (
	ModelStatusActive   = "active"   // 启用
	ModelStatusDisabled = "disabled" // 下架
)

// Model 渠道下的可用模型（渠道 × 模型），对应 models 表
// 说明：对外（逻辑）模型名见 name，实际上游模型名见 upstream_model；某渠道存在一条记录即表示其支持该模型。
type Model struct {
	ID              int64       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ChannelID       int64       `gorm:"column:channel_id;not null;index:idx_models_channel;uniqueIndex:uq_models_channel_name_active,priority:1,where:deleted_at IS NULL;comment:所属渠道 channels.id" json:"channel_id"`
	Name            string      `gorm:"column:name;type:varchar(128);not null;index:idx_models_name;uniqueIndex:uq_models_channel_name_active,priority:2,where:deleted_at IS NULL;comment:对外（逻辑）模型名" json:"name"`
	UpstreamModel   string      `gorm:"column:upstream_model;type:varchar(128);not null;comment:该渠道上游真实模型名" json:"upstream_model"`
	ContextLength   *int        `gorm:"column:context_length;comment:上下文窗口长度（tokens）" json:"context_length,omitempty"`
	MaxOutputTokens *int        `gorm:"column:max_output_tokens;comment:单次最大输出 tokens" json:"max_output_tokens,omitempty"`
	Capabilities    StringArray `gorm:"column:capabilities;type:text[];default:'{}';index:idx_models_capabilities,type:gin;comment:能力标签 vision/tools/json/reasoning/audio" json:"capabilities"`
	Status          string      `gorm:"column:status;type:varchar(32);not null;default:active;check:chk_models_status,status = 'active' OR status = 'disabled';comment:状态 active-启用 disabled-下架" json:"status"`
	// 预留价格字段：为后续「模型价格表」功能占位，本期可空、可维护、不参与计费
	InputPrice         *float64       `gorm:"column:input_price;comment:预留-输入单价（待接入计费）" json:"input_price,omitempty"`
	OutputPrice        *float64       `gorm:"column:output_price;comment:预留-输出单价（待接入计费）" json:"output_price,omitempty"`
	CachedInputPrice   *float64       `gorm:"column:cached_input_price;comment:预留-缓存命中输入单价（待接入计费）" json:"cached_input_price,omitempty"`
	CacheCreationPrice *float64       `gorm:"column:cache_creation_price;comment:预留-缓存创建单价（待接入计费）" json:"cache_creation_price,omitempty"`
	Currency           string         `gorm:"column:currency;type:varchar(16);comment:预留-币种（如 USD/CNY）" json:"currency,omitempty"`
	CreatedAt          time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"column:deleted_at;index:idx_models_deleted_at" json:"-"`
}
