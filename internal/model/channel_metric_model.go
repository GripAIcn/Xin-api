package model

import "time"

// ChannelMetric 渠道指标聚合，对应 channel_metrics 表（按 渠道 × 模型 × 窗口 聚合）
// 说明：avg_latency_ms / avg_ttft_ms / tpot / rpm / tpm / total_tokens 均为数据库生成列（只读）。
type ChannelMetric struct {
	ChannelID         int64     `gorm:"column:channel_id;primaryKey;comment:渠道 ID" json:"channel_id"`
	Model             string    `gorm:"column:model;type:varchar(128);primaryKey;comment:对外（逻辑）模型名" json:"model"`
	WindowStart       time.Time `gorm:"column:window_start;primaryKey;comment:窗口起始时间" json:"window_start"`
	WindowSize        int       `gorm:"column:window_size;type:integer;not null;comment:基窗口长度（秒），固定 60" json:"window_size"`
	RequestCount      int64     `gorm:"column:request_count;default:0;check:chk_channel_metrics_counts,request_count = success_count + error_count + timeout_count;comment:总请求数" json:"request_count"`
	SuccessCount      int64     `gorm:"column:success_count;default:0;comment:成功数" json:"success_count"`
	ErrorCount        int64     `gorm:"column:error_count;default:0;comment:失败数" json:"error_count"`
	TimeoutCount      int64     `gorm:"column:timeout_count;default:0;comment:超时数" json:"timeout_count"`
	TotalInputTokens  int64     `gorm:"column:total_input_tokens;default:0;comment:输入 tokens 合计" json:"total_input_tokens"`
	TotalOutputTokens int64     `gorm:"column:total_output_tokens;default:0;comment:输出 tokens 合计" json:"total_output_tokens"`
	TotalTokens       int64     `gorm:"column:total_tokens;type:bigint GENERATED ALWAYS AS (total_input_tokens + total_output_tokens) STORED;->;comment:总 tokens（生成列）" json:"total_tokens"`
	SumLatencyMs      int64     `gorm:"column:sum_latency_ms;default:0;comment:成功请求延迟总和" json:"sum_latency_ms"`
	SumTTFTMs         int64     `gorm:"column:sum_ttft_ms;default:0;comment:TTFT 总和" json:"sum_ttft_ms"`
	TTFTCount         int64     `gorm:"column:ttft_count;default:0;comment:TTFT 样本数" json:"ttft_count"`
	SumGenerationMs   int64     `gorm:"column:sum_generation_ms;default:0;comment:生成耗时总和" json:"sum_generation_ms"`
	AvgLatencyMs      float64   `gorm:"column:avg_latency_ms;type:numeric(12,2) GENERATED ALWAYS AS (sum_latency_ms::numeric / NULLIF(success_count, 0)) STORED;->;comment:平均延迟（生成列）" json:"avg_latency_ms"`
	AvgTTFTMs         float64   `gorm:"column:avg_ttft_ms;type:numeric(12,2) GENERATED ALWAYS AS (sum_ttft_ms::numeric / NULLIF(ttft_count, 0)) STORED;->;comment:平均 TTFT（生成列）" json:"avg_ttft_ms"`
	TPOT              float64   `gorm:"column:tpot;type:numeric(12,2) GENERATED ALWAYS AS (sum_generation_ms::numeric / NULLIF(total_output_tokens, 0)) STORED;->;comment:每输出 token 平均生成耗时（生成列）" json:"tpot"`
	RPM               float64   `gorm:"column:rpm;type:numeric(12,2) GENERATED ALWAYS AS (request_count::numeric * 60 / NULLIF(window_size, 0)) STORED;->;comment:每分钟请求数（生成列）" json:"rpm"`
	TPM               float64   `gorm:"column:tpm;type:numeric(12,2) GENERATED ALWAYS AS ((total_input_tokens + total_output_tokens)::numeric * 60 / NULLIF(window_size, 0)) STORED;->;comment:每分钟 token 数（生成列）" json:"tpm"`
	TotalCost         Money     `gorm:"column:total_cost;type:numeric(20,8);default:0;comment:成本合计" json:"total_cost"`
	Currency          string    `gorm:"column:currency;type:varchar(16);not null;default:USD;comment:币种" json:"currency"`
	UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime;comment:最近更新时间" json:"updated_at"`
}
