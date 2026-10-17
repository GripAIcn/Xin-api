package model

import "time"

// DashboardOverview 仪表盘概览指标（所选时间范围内 request_logs 聚合）
type DashboardOverview struct {
	RequestCount int64   `gorm:"column:request_count" json:"request_count"`
	SuccessCount int64   `gorm:"column:success_count" json:"success_count"`
	ErrorCount   int64   `gorm:"column:error_count" json:"error_count"`
	SuccessRate  float64 `gorm:"column:success_rate" json:"success_rate"` // 0~1 小数，展示层自行换算百分比

	InputTokens  int64 `gorm:"column:input_tokens" json:"input_tokens"`
	OutputTokens int64 `gorm:"column:output_tokens" json:"output_tokens"`
	TotalTokens  int64 `gorm:"column:total_tokens" json:"total_tokens"`

	TotalCost    Money   `gorm:"column:total_cost" json:"total_cost"`
	Currency     string  `gorm:"column:currency" json:"currency"`
	AvgLatencyMs float64 `gorm:"column:avg_latency_ms" json:"avg_latency_ms"`
	AvgTTFTMs    float64 `gorm:"column:avg_ttft_ms" json:"avg_ttft_ms"`
}

// DashboardTrendPoint 仪表盘趋势分桶点（今日按小时、近 N 天按天）
type DashboardTrendPoint struct {
	BucketStart  time.Time `gorm:"column:bucket_start" json:"bucket_start"`
	RequestCount int64     `gorm:"column:request_count" json:"request_count"`
	SuccessCount int64     `gorm:"column:success_count" json:"success_count"`
	ErrorCount   int64     `gorm:"column:error_count" json:"error_count"`
	InputTokens  int64     `gorm:"column:input_tokens" json:"input_tokens"`
	OutputTokens int64     `gorm:"column:output_tokens" json:"output_tokens"`
	TotalTokens  int64     `gorm:"column:total_tokens" json:"total_tokens"`
	TotalCost    Money     `gorm:"column:total_cost" json:"total_cost"`
}

// DashboardModelRank 模型调用排行项
type DashboardModelRank struct {
	Model        string `gorm:"column:model" json:"model"`
	RequestCount int64  `gorm:"column:request_count" json:"request_count"`
	TotalTokens  int64  `gorm:"column:total_tokens" json:"total_tokens"`
	TotalCost    Money  `gorm:"column:total_cost" json:"total_cost"`
}

// DashboardChannelRank 渠道调用排行项
type DashboardChannelRank struct {
	ChannelID    int64   `gorm:"column:channel_id" json:"channel_id"`
	ChannelName  *string `gorm:"column:channel_name" json:"channel_name"`
	RequestCount int64   `gorm:"column:request_count" json:"request_count"`
	SuccessRate  float64 `gorm:"column:success_rate" json:"success_rate"`
	AvgLatencyMs float64 `gorm:"column:avg_latency_ms" json:"avg_latency_ms"`
}

// DashboardErrorRank 错误排行项
type DashboardErrorRank struct {
	ErrorType string `gorm:"column:error_type" json:"error_type"`
	Count     int64  `gorm:"column:count" json:"count"`
}

// DashboardStats 仪表盘统计聚合结果（GET /v1/dashboard/stats）
type DashboardStats struct {
	Range      string `json:"range"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`

	Overview    DashboardOverview      `json:"overview"`
	Trend       []DashboardTrendPoint  `json:"trend"`
	TopModels   []DashboardModelRank   `json:"top_models"`
	TopChannels []DashboardChannelRank `json:"top_channels"`
	TopErrors   []DashboardErrorRank   `json:"top_errors"`
}
