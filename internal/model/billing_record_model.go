package model

import "time"

// 计费流水状态枚举，对应 billing_records.status
const (
	BillingStatusSettled  = "settled"  // 已结算
	BillingStatusFailed   = "failed"   // 结算失败
	BillingStatusRefunded = "refunded" // 已退款
)

// BillingRecord 计费流水，对应 billing_records 表
// 说明：total_tokens 为数据库生成列（只读）；价格与倍率均为快照。
type BillingRecord struct {
	ID                  int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	RequestID           string    `gorm:"column:request_id;type:varchar(64);not null;unique;comment:Xin API 请求 ID" json:"request_id"`
	UserID              int64     `gorm:"column:user_id;not null;comment:所属用户" json:"user_id"`
	APIKeyID            *int64    `gorm:"column:api_key_id;comment:所属 Key" json:"api_key_id,omitempty"`
	Model               string    `gorm:"column:model;type:varchar(128);not null;comment:对外模型名快照" json:"model"`
	ChannelID           *int64    `gorm:"column:channel_id;comment:渠道快照（不建外键）" json:"channel_id,omitempty"`
	InputTokens         int64     `gorm:"column:input_tokens;default:0;comment:总输入 tokens" json:"input_tokens"`
	OutputTokens        int64     `gorm:"column:output_tokens;default:0;comment:输出 tokens" json:"output_tokens"`
	TotalTokens         int64     `gorm:"column:total_tokens;type:bigint GENERATED ALWAYS AS (input_tokens + output_tokens) STORED;->;comment:总 tokens（生成列）" json:"total_tokens"`
	CachedInputTokens   int64     `gorm:"column:cached_input_tokens;default:0;comment:缓存命中输入 tokens" json:"cached_input_tokens"`
	CacheCreationTokens int64     `gorm:"column:cache_creation_tokens;default:0;comment:缓存创建输入 tokens" json:"cache_creation_tokens"`
	Currency            string    `gorm:"column:currency;type:varchar(16);not null;comment:币种快照" json:"currency"`
	InputPrice          float64   `gorm:"column:input_price;type:numeric(20,8);not null;comment:普通输入单价快照" json:"input_price"`
	OutputPrice         float64   `gorm:"column:output_price;type:numeric(20,8);not null;comment:输出单价快照" json:"output_price"`
	CachedInputPrice    float64   `gorm:"column:cached_input_price;type:numeric(20,8);not null;comment:输入缓存命中单价快照" json:"cached_input_price"`
	CacheCreationPrice  float64   `gorm:"column:cache_creation_price;type:numeric(20,8);not null;comment:输入缓存创建单价快照" json:"cache_creation_price"`
	Ratio               float64   `gorm:"column:ratio;type:numeric(12,4);not null;default:1;comment:加价倍率快照" json:"ratio"`
	InputCost           float64   `gorm:"column:input_cost;type:numeric(20,8);not null;comment:输入总成本" json:"input_cost"`
	OutputCost          float64   `gorm:"column:output_cost;type:numeric(20,8);not null;comment:输出成本" json:"output_cost"`
	CachedInputCost     float64   `gorm:"column:cached_input_cost;type:numeric(20,8);not null;comment:输入缓存命中成本" json:"cached_input_cost"`
	CacheCreationCost   float64   `gorm:"column:cache_creation_cost;type:numeric(20,8);not null;comment:输入缓存创建成本" json:"cache_creation_cost"`
	TotalCost           float64   `gorm:"column:total_cost;type:numeric(20,8);not null;comment:总成本" json:"total_cost"`
	BalanceBefore       *float64  `gorm:"column:balance_before;type:numeric(20,8);comment:结算前余额" json:"balance_before,omitempty"`
	BalanceAfter        *float64  `gorm:"column:balance_after;type:numeric(20,8);comment:结算后余额" json:"balance_after,omitempty"`
	Status              string    `gorm:"column:status;type:varchar(32);not null;default:settled;comment:状态 settled-已结算 failed-结算失败 refunded-已退款" json:"status"`
	CreatedAt           time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}
