package model

// ModelChannelEntry 「渠道 × 模型」明细行（JOIN 渠道信息），供模型管理页按逻辑模型名聚合展示。
// 说明：Name 为对外（逻辑）模型名，仅用于上层按名分组，不单独序列化到明细中（分组结果见 ModelGroup.Name）。
type ModelChannelEntry struct {
	Name               string      `gorm:"column:name" json:"-"`
	ModelID            int64       `gorm:"column:model_id" json:"model_id"`
	ChannelID          int64       `gorm:"column:channel_id" json:"channel_id"`
	ChannelName        string      `gorm:"column:channel_name" json:"channel_name"`
	ChannelProvider    string      `gorm:"column:channel_provider" json:"channel_provider"`
	ChannelStatus      string      `gorm:"column:channel_status" json:"channel_status"`
	UpstreamModel      string      `gorm:"column:upstream_model" json:"upstream_model"`
	Status             string      `gorm:"column:status" json:"status"`
	ContextLength      *int        `gorm:"column:context_length" json:"context_length,omitempty"`
	MaxOutputTokens    *int        `gorm:"column:max_output_tokens" json:"max_output_tokens,omitempty"`
	Capabilities       StringArray `gorm:"column:capabilities" json:"capabilities"`
	InputPrice         *float64    `gorm:"column:input_price" json:"input_price,omitempty"`
	OutputPrice        *float64    `gorm:"column:output_price" json:"output_price,omitempty"`
	CachedInputPrice   *float64    `gorm:"column:cached_input_price" json:"cached_input_price,omitempty"`
	CacheCreationPrice *float64    `gorm:"column:cache_creation_price" json:"cache_creation_price,omitempty"`
	Currency           string      `gorm:"column:currency" json:"currency,omitempty"`
}

// ModelGroup 按逻辑模型名聚合后的分组（同一 name 下的全部渠道明细）
type ModelGroup struct {
	Name         string              `json:"name"`
	ChannelCount int                 `json:"channel_count"`
	ActiveCount  int                 `json:"active_count"`
	Channels     []ModelChannelEntry `json:"channels"`
}
