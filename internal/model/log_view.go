package model

import "time"

// LogListItem 日志列表视图
type LogListItem struct {
	ID            int64     `gorm:"column:id" json:"id"`
	RequestID     string    `gorm:"column:request_id" json:"request_id"`
	UpstreamReqID *string   `gorm:"column:upstream_req_id" json:"upstream_req_id,omitempty"`
	LogType       string    `gorm:"column:log_type" json:"log_type"`
	Status        string    `gorm:"column:status" json:"status"`
	Model         string    `gorm:"column:model" json:"model"`
	ChannelID     *int64    `gorm:"column:channel_id" json:"channel_id,omitempty"`
	ChannelName   *string   `gorm:"column:channel_name" json:"channel_name,omitempty"`
	APIKeyID      *int64    `gorm:"column:api_key_id" json:"api_key_id,omitempty"`
	UserID        *int64    `gorm:"column:user_id" json:"user_id,omitempty"`
	UserName      *string   `gorm:"column:user_name" json:"user_name,omitempty"`
	InputTokens   int64     `gorm:"column:input_tokens" json:"input_tokens"`
	OutputTokens  int64     `gorm:"column:output_tokens" json:"output_tokens"`
	TotalTokens   int64     `gorm:"column:total_tokens" json:"total_tokens"`
	TotalCost     Money     `gorm:"column:total_cost" json:"total_cost"`
	Currency      string    `gorm:"column:currency" json:"currency"`
	LatencyMs     *int64    `gorm:"column:latency_ms" json:"latency_ms,omitempty"`
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
}

// PriceSnapshot 计价单价快照（billing_records 单价 + ratio + 币种）
type PriceSnapshot struct {
	InputPrice         Money   `json:"input_price"`
	OutputPrice        Money   `json:"output_price"`
	CachedInputPrice   Money   `json:"cached_input_price"`
	CacheCreationPrice Money   `json:"cache_creation_price"`
	Ratio              float64 `json:"ratio"`
	Currency           string  `json:"currency"`
}

// BillingDetail 计费明细（消费记录，来自 billing_records）
type BillingDetail struct {
	ID                  int64         `json:"id"`
	RequestID           string        `json:"request_id"`
	Model               string        `json:"model"`
	ChannelID           *int64        `json:"channel_id,omitempty"`
	InputTokens         int64         `json:"input_tokens"`
	OutputTokens        int64         `json:"output_tokens"`
	TotalTokens         int64         `json:"total_tokens"`
	CachedInputTokens   int64         `json:"cached_input_tokens"`
	CacheCreationTokens int64         `json:"cache_creation_tokens"`
	Price               PriceSnapshot `json:"price"`
	InputCost           Money         `json:"input_cost"`
	OutputCost          Money         `json:"output_cost"`
	CachedInputCost     Money         `json:"cached_input_cost"`
	CacheCreationCost   Money         `json:"cache_creation_cost"`
	TotalCost           Money         `json:"total_cost"`
	BalanceBefore       *Money        `json:"balance_before,omitempty"`
	BalanceAfter        *Money        `json:"balance_after,omitempty"`
	Status              string        `json:"status"`
	CreatedAt           time.Time     `json:"created_at"`
}

// LogDetail 日志详情视图（内嵌列表视图 + 详情字段 + 计费明细）
type LogDetail struct {
	LogListItem
	UpstreamModel       *string        `json:"upstream_model,omitempty"`
	Provider            *string        `json:"provider,omitempty"`
	Protocol            *string        `json:"protocol,omitempty"`
	StatusCode          *int           `json:"status_code,omitempty"`
	FinishReason        *string        `json:"finish_reason,omitempty"`
	IsStream            bool           `json:"is_stream"`
	IsCancelled         bool           `json:"is_cancelled"`
	CancelledBy         *string        `json:"cancelled_by,omitempty"`
	RetryCount          int            `json:"retry_count"`
	RetryChannels       []int64        `json:"retry_channels"`
	UsageSource         string         `json:"usage_source"`
	RouteStrategy       *string        `json:"route_strategy,omitempty"`
	RouteWeight         *int           `json:"route_weight,omitempty"`
	CachedInputTokens   int64          `json:"cached_input_tokens"`
	CacheCreationTokens int64          `json:"cache_creation_tokens"`
	ReasoningTokens     int64          `json:"reasoning_tokens"`
	TTFTMs              *int64         `json:"ttft_ms,omitempty"`
	GenerationMs        *int64         `json:"generation_ms,omitempty"`
	ErrorStage          *string        `json:"error_stage,omitempty"`
	ErrorType           *string        `json:"error_type,omitempty"`
	ErrorCode           *string        `json:"error_code,omitempty"`
	ErrorMessage        *string        `json:"error_message,omitempty"`
	ClientIP            *string        `json:"client_ip,omitempty"`
	GatewayNode         *string        `json:"gateway_node,omitempty"`
	StartedAt           *time.Time     `json:"started_at,omitempty"`
	CompletedAt         *time.Time     `json:"completed_at,omitempty"`
	Billing             *BillingDetail `json:"billing,omitempty"`
}
