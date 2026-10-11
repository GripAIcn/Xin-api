package form

// CreateChannelReq 添加上游渠道请求入参
type CreateChannelReq struct {
	Name         string `json:"name" binding:"required" validate:"required,min=2,max=100"`    // 渠道名称
	ModelMapping string `json:"model_mapping" binding:"required" validate:"required"`         // 模型列表，如 "gpt-4o,deepseek-chat"
	BaseURL      string `json:"base_url" binding:"required" validate:"required,url"`          // 必须符合标准的 URL 格式
	APIKey       string `json:"api_key" binding:"required" validate:"required,min=1,max=255"` // 供应商密钥
	Weight       int    `json:"weight" binding:"required" validate:"required,min=1,max=100"`  // 负载权重范围 1-100
}

// UpdateChannelReq 覆盖更新渠道请求入参
type UpdateChannelReq struct {
	Name         string `json:"name" binding:"required" validate:"required,min=2,max=100"`
	ModelMapping string `json:"model_mapping" binding:"required" validate:"required"`
	BaseURL      string `json:"base_url" binding:"required" validate:"required,url"`
	APIKey       string `json:"api_key" binding:"required" validate:"required,min=1,max=255"`
	Weight       int    `json:"weight" binding:"required" validate:"required,min=1,max=100"`
	Status       string `json:"status" binding:"oneof=active disabled circuit_open" validate:"oneof=active disabled circuit_open"` // 状态：active-正常 disabled-手动关闭 circuit_open-自动熔断
}

// TestChannelReq 测试单个渠道请求入参
type TestChannelReq struct {
	Model string `json:"model"` // 可选，指定测试某个模型；为空则测试该渠道支持的所有模型
}
