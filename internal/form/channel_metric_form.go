package form

// ChannelMetricQuery 指标汇总查询入参
type ChannelMetricQuery struct {
	Range     string `form:"range" json:"range" validate:"omitempty,oneof=1h 6h 24h 7d"`
	Dimension string `form:"dimension" json:"dimension" validate:"omitempty,oneof=channel model channel_model"`
	ChannelID *int64 `form:"channel_id" json:"channel_id" validate:"omitempty,gte=0"`
	Model     string `form:"model" json:"model"`
}

// ChannelMetricTrendQuery 指标趋势查询入参
type ChannelMetricTrendQuery struct {
	Range     string `form:"range" json:"range" validate:"omitempty,oneof=1h 6h 24h 7d"`
	ChannelID *int64 `form:"channel_id" json:"channel_id" validate:"omitempty,gte=0"`
	Model     string `form:"model" json:"model"`
}

// ChannelMetricAggregateRequest 手动范围聚合入参
type ChannelMetricAggregateRequest struct {
	StartTime string `json:"start_time" validate:"required"`
	EndTime   string `json:"end_time" validate:"required"`
}
