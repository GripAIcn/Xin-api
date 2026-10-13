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
	ID              int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ChannelID       int64          `gorm:"column:channel_id;not null;index;comment:所属渠道 channels.id" json:"channel_id"`
	Name            string         `gorm:"column:name;type:varchar(128);not null;index;comment:对外（逻辑）模型名" json:"name"`
	UpstreamModel   string         `gorm:"column:upstream_model;type:varchar(128);not null;comment:该渠道上游真实模型名" json:"upstream_model"`
	ContextLength   *int           `gorm:"column:context_length;comment:上下文窗口长度（tokens）" json:"context_length,omitempty"`
	MaxOutputTokens *int           `gorm:"column:max_output_tokens;comment:单次最大输出 tokens" json:"max_output_tokens,omitempty"`
	Capabilities    []string       `gorm:"column:capabilities;type:text[];default:'{}';comment:能力标签 vision/tools/json/reasoning/audio" json:"capabilities"`
	Status          string         `gorm:"column:status;type:varchar(32);not null;default:active;comment:状态 active-启用 disabled-下架" json:"status"`
	CreatedAt       time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}
