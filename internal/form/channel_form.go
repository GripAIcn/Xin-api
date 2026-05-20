package form

// CreateChannelReq 添加上游渠道请求入参
type CreateChannelReq struct {
	GroupID      int64  `json:"group_id" binding:"required" validate:"required,gt=0"`         // 必须大于 0
	Name         string `json:"name" binding:"required" validate:"required,min=2,max=100"`    // 渠道名称
	ModelMapping string `json:"model_mapping" binding:"required" validate:"required"`         // 模型列表，如 "gpt-4o,deepseek-chat"
	BaseURL      string `json:"base_url" binding:"required" validate:"required,url"`          // 必须符合标准的 URL 格式
	APIKey       string `json:"api_key" binding:"required" validate:"required,min=1,max=255"` // 供应商密钥
	Weight       int    `json:"weight" binding:"required" validate:"required,min=1,max=100"`  // 负载权重范围 1-100
}

// UpdateChannelReq 覆盖更新渠道请求入参
type UpdateChannelReq struct {
	GroupID      int64  `json:"group_id" binding:"required" validate:"required,gt=0"`
	Name         string `json:"name" binding:"required" validate:"required,min=2,max=100"`
	ModelMapping string `json:"model_mapping" binding:"required" validate:"required"`
	BaseURL      string `json:"base_url" binding:"required" validate:"required,url"`
	APIKey       string `json:"api_key" binding:"required" validate:"required,min=1,max=255"`
	Weight       int    `json:"weight" binding:"required" validate:"required,min=1,max=100"`
	Status       int    `json:"status" binding:"oneof=0 1 2" validate:"oneof=0 1 2"` // 状态允许修改为 0-手动关闭, 1-正常, 2-自动熔断
}
