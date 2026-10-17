package model

import "time"

// ChannelMetricSummary 指标跨窗口二次聚合结果（按渠道 / 模型 / 渠道 × 模型维度）
type ChannelMetricSummary struct {
	ChannelID       *int64  `gorm:"column:channel_id" json:"channel_id"`
	ChannelName     *string `gorm:"column:channel_name" json:"channel_name"`
	ChannelProvider *string `gorm:"column:channel_provider" json:"channel_provider"`
	ChannelStatus   *string `gorm:"column:channel_status" json:"channel_status"`
	Model           *string `gorm:"column:model" json:"model"`

	RequestCount int64   `gorm:"column:request_count" json:"request_count"`
	SuccessCount int64   `gorm:"column:success_count" json:"success_count"`
	ErrorCount   int64   `gorm:"column:error_count" json:"error_count"`
	TimeoutCount int64   `gorm:"column:timeout_count" json:"timeout_count"`
	SuccessRate  float64 `gorm:"column:success_rate" json:"success_rate"`

	AvgLatencyMs float64 `gorm:"column:avg_latency_ms" json:"avg_latency_ms"`
	AvgTTFTMs    float64 `gorm:"column:avg_ttft_ms" json:"avg_ttft_ms"`
	TPOT         float64 `gorm:"column:tpot" json:"tpot"`
	RPM          float64 `gorm:"column:rpm" json:"rpm"`
	TPM          float64 `gorm:"column:tpm" json:"tpm"`

	TotalInputTokens  int64   `gorm:"column:total_input_tokens" json:"total_input_tokens"`
	TotalOutputTokens int64   `gorm:"column:total_output_tokens" json:"total_output_tokens"`
	TotalTokens       int64   `gorm:"column:total_tokens" json:"total_tokens"`
	TotalCost         float64 `gorm:"column:total_cost" json:"total_cost"`
	Currency          string  `gorm:"column:currency" json:"currency"`

	LastWindowStart *time.Time `gorm:"column:last_window_start" json:"last_window_start"`
}

// ChannelMetricTrendPoint 指标趋势分桶点
type ChannelMetricTrendPoint struct {
	BucketStart  time.Time `gorm:"column:bucket_start" json:"bucket_start"`
	RequestCount int64     `gorm:"column:request_count" json:"request_count"`
	SuccessCount int64     `gorm:"column:success_count" json:"success_count"`
	AvgTTFTMs    float64   `gorm:"column:avg_ttft_ms" json:"avg_ttft_ms"`
	AvgLatencyMs float64   `gorm:"column:avg_latency_ms" json:"avg_latency_ms"`
	TPOT         float64   `gorm:"column:tpot" json:"tpot"`
	SuccessRate  float64   `gorm:"column:success_rate" json:"success_rate"`
}
