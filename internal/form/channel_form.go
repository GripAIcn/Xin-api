package form

// CreateChannelReq 添加上游渠道请求入参
type CreateChannelReq struct {
	Name       string   `json:"name" binding:"required" validate:"required,min=2,max=100"`    // 渠道名称
	Provider   string   `json:"provider" binding:"required" validate:"required,min=1,max=64"` // 供应商类型：openai/anthropic/gemini/deepseek
	Protocol   []string `json:"protocol" binding:"required" validate:"required,min=1,dive"`   // 支持协议数组：chat/responses/gemini
	BaseURL    string   `json:"base_url" binding:"required" validate:"required,url"`          // 必须符合标准的 URL 格式
	APIKey     string   `json:"api_key" binding:"required" validate:"required,min=1,max=255"` // 供应商密钥
	Weight     int      `json:"weight" binding:"required" validate:"required,min=1,max=100"`  // 负载权重范围 1-100
	Priority   int      `json:"priority"`                                                     // 路由优先级，数值越大越优先
	Tags       []string `json:"tags"`                                                         // 路由标签：fast/cheap/stable
	Timeout    int      `json:"timeout" validate:"omitempty,min=1"`                           // 请求超时（秒）
	MaxRetries int      `json:"max_retries" validate:"omitempty,min=0"`                       // 失败重试次数
}

// UpdateChannelReq 覆盖更新渠道请求入参
type UpdateChannelReq struct {
	Name       string   `json:"name" binding:"required" validate:"required,min=2,max=100"`
	Provider   string   `json:"provider" binding:"required" validate:"required,min=1,max=64"`
	Protocol   []string `json:"protocol" binding:"required" validate:"required,min=1,dive"`
	BaseURL    string   `json:"base_url" binding:"required" validate:"required,url"`
	APIKey     string   `json:"api_key" binding:"required" validate:"required,min=1,max=255"`
	Weight     int      `json:"weight" binding:"required" validate:"required,min=1,max=100"`
	Priority   int      `json:"priority"`
	Tags       []string `json:"tags"`
	Timeout    int      `json:"timeout" validate:"omitempty,min=1"`
	MaxRetries int      `json:"max_retries" validate:"omitempty,min=0"`
	Status     string   `json:"status" binding:"oneof=active disabled circuit_open" validate:"oneof=active disabled circuit_open"` // 状态：active-正常 disabled-手动关闭 circuit_open-自动熔断
}

// TestChannelReq 测试单个渠道请求入参
type TestChannelReq struct {
	Model string `json:"model"` // 可选，指定测试某个模型；为空则测试该渠道支持的所有模型
}
