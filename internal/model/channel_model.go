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
// 说明：渠道支持的模型改为由 models 表（渠道 × 模型）行式表达，不再使用数组列。
type Channel struct {
	ID              int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name            string         `gorm:"column:name;type:varchar(100);not null;comment:渠道名称" json:"name"`
	Provider        string         `gorm:"column:provider;type:varchar(64);not null;comment:供应商类型 openai/anthropic/gemini/deepseek" json:"provider"`
	Protocol        StringArray    `gorm:"column:protocol;type:varchar(64)[];not null;comment:支持协议数组 chat/responses/gemini" json:"protocol"`
	BaseURL         string         `gorm:"column:base_url;type:varchar(255);not null;comment:供应商基础URL" json:"base_url"`
	APIKey          string         `gorm:"column:api_key;type:varchar(255);not null;comment:供应商鉴权key（应用层可逆加密）" json:"api_key"`
	Weight          int            `gorm:"column:weight;not null;default:1;comment:同级负载权重 1-100" json:"weight"`
	Priority        int            `gorm:"column:priority;not null;default:0;comment:路由优先级，数值越大越优先" json:"priority"`
	Tags            StringArray    `gorm:"column:tags;type:text[];default:'{}';comment:路由标签 fast/cheap/stable" json:"tags"`
	Status          string         `gorm:"column:status;type:varchar(32);not null;default:active;comment:状态 active-正常 disabled-手动关闭 circuit_open-自动熔断" json:"status"`
	Timeout         int            `gorm:"column:timeout;default:60;comment:请求超时（秒）" json:"timeout"`
	MaxRetries      int            `gorm:"column:max_retries;default:0;comment:失败重试次数" json:"max_retries"`
	LastErrorAt     *time.Time     `gorm:"column:last_error_at;comment:最近一次调用失败时间" json:"last_error_at,omitempty"`
	CircuitOpenedAt *time.Time     `gorm:"column:circuit_opened_at;comment:熔断开启时间，NULL=从未熔断" json:"circuit_opened_at,omitempty"`
	CreatedAt       time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}
