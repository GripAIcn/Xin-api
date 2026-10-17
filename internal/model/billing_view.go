package model

import "time"

// 账单流水来源枚举，对应 BillingLogItem.Source
const (
	BillingSourceConsumption = "consumption" // 消费（出账，取自 billing_records）
	BillingSourceRecharge    = "recharge"    // 充值（入账，取自 request_logs）
	BillingSourceRefund      = "refund"      // 退款（入账，取自 request_logs）
)

// BillingLogItem 账务流水视图：将「消费」（billing_records）与「充值/退款」（request_logs）
// 合并后的扁平流水行。
// 说明：采用扁平结构而非嵌套 BillingDetail / PriceSnapshot，避免 GORM Scan 的嵌套映射问题；
// 每列均带 gorm column tag，与 ListBilling 中 UNION ALL 的列别名一一对应。
// amount 带符号：消费为负（-total_cost）、充值/退款为正（+total_cost）。
type BillingLogItem struct {
	ID                  int64     `gorm:"column:id" json:"id"`
	Source              string    `gorm:"column:source" json:"source"`
	RequestID           string    `gorm:"column:request_id" json:"request_id"`
	UserID              *int64    `gorm:"column:user_id" json:"user_id,omitempty"`
	UserName            *string   `gorm:"column:user_name" json:"user_name,omitempty"`
	Model               string    `gorm:"column:model" json:"model"`
	Currency            string    `gorm:"column:currency" json:"currency"`
	Amount              Money     `gorm:"column:amount" json:"amount"`
	BalanceBefore       *Money    `gorm:"column:balance_before" json:"balance_before,omitempty"`
	BalanceAfter        *Money    `gorm:"column:balance_after" json:"balance_after,omitempty"`
	Status              string    `gorm:"column:status" json:"status"`
	InputTokens         int64     `gorm:"column:input_tokens" json:"input_tokens"`
	OutputTokens        int64     `gorm:"column:output_tokens" json:"output_tokens"`
	TotalTokens         int64     `gorm:"column:total_tokens" json:"total_tokens"`
	CachedInputTokens   int64     `gorm:"column:cached_input_tokens" json:"cached_input_tokens"`
	CacheCreationTokens int64     `gorm:"column:cache_creation_tokens" json:"cache_creation_tokens"`
	InputPrice          Money     `gorm:"column:input_price" json:"input_price"`
	OutputPrice         Money     `gorm:"column:output_price" json:"output_price"`
	CachedInputPrice    Money     `gorm:"column:cached_input_price" json:"cached_input_price"`
	CacheCreationPrice  Money     `gorm:"column:cache_creation_price" json:"cache_creation_price"`
	Ratio               float64   `gorm:"column:ratio" json:"ratio"`
	InputCost           Money     `gorm:"column:input_cost" json:"input_cost"`
	OutputCost          Money     `gorm:"column:output_cost" json:"output_cost"`
	CachedInputCost     Money     `gorm:"column:cached_input_cost" json:"cached_input_cost"`
	CacheCreationCost   Money     `gorm:"column:cache_creation_cost" json:"cache_creation_cost"`
	CreatedAt           time.Time `gorm:"column:created_at" json:"created_at"`
}
