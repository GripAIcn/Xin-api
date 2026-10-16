package model

// ModelPriceWithCount 价格方案 + 绑定模型数（列表展示用）
// BoundCount 统计该方案下未软删的绑定数量。
type ModelPriceWithCount struct {
	ModelPrice
	BoundCount int64 `gorm:"column:bound_count" json:"bound_count"`
}

// PriceBoundModelEntry 绑定到某价格方案的「渠道 × 模型」明细
type PriceBoundModelEntry struct {
	BindingID       int64  `gorm:"column:binding_id" json:"binding_id"`
	ModelID         int64  `gorm:"column:model_id" json:"model_id"`
	ChannelID       int64  `gorm:"column:channel_id" json:"channel_id"`
	ChannelName     string `gorm:"column:channel_name" json:"channel_name"`
	ChannelProvider string `gorm:"column:channel_provider" json:"channel_provider"`
	ModelName       string `gorm:"column:model_name" json:"model_name"`
	UpstreamModel   string `gorm:"column:upstream_model" json:"upstream_model"`
	ModelStatus     string `gorm:"column:model_status" json:"model_status"`
	ChannelStatus   string `gorm:"column:channel_status" json:"channel_status"`
}
