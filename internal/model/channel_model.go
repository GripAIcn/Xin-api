package model

import (
	"time"

	"gorm.io/gorm"
)

// 渠道状态枚举，对应数据库 channels.status 的取值
const (
	ChannelStatusActive      = "active"       // 正常
	ChannelStatusDisabled    = "disabled"     // 手动关闭
	ChannelStatusCircuitOpen = "circuit_open" // 自动熔断
)

// Channel 上游物理供应商渠道，对应 channels 表
type Channel struct {
	ID           int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name         string         `gorm:"column:name;type:varchar(100);not null;comment:渠道名称" json:"name"`
	ModelMapping string         `gorm:"column:model_mapping;type:text;not null;comment:支持的模型列表，逗号分隔，例如 gpt-4o,deepseek-chat" json:"model_mapping"`
	BaseURL      string         `gorm:"column:base_url;type:varchar(255);not null;comment:供应商基础URL" json:"base_url"`
	APIKey       string         `gorm:"column:api_key;type:varchar(255);not null;comment:供应商鉴权key" json:"api_key"`
	Weight       int            `gorm:"column:weight;default:1;comment:负载权重 1-100" json:"weight"`
	Status       string         `gorm:"column:status;type:varchar(32);not null;default:active;comment:渠道状态 active-正常 disabled-手动关闭 circuit_open-自动熔断" json:"status"`
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}
