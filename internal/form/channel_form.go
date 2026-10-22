package form

// CreateChannelReq 添加上游渠道请求入参
type CreateChannelReq struct {
	Name       string   `json:"name" binding:"required" validate:"required,min=2,max=100"`                                                                                       // 渠道名称
	Provider   string   `json:"provider" binding:"required,oneof=openai deepseek anthropic gemini" validate:"required,oneof=openai deepseek anthropic gemini"`                   // 供应商类型：openai/deepseek/anthropic/gemini
	Protocol   []string `json:"protocol" binding:"required,min=1,dive,oneof=chat responses messages gemini" validate:"required,min=1,dive,oneof=chat responses messages gemini"` // 支持协议数组：chat/responses/messages/gemini
	BaseURL    string   `json:"base_url" binding:"required" validate:"required,url"`                                                                                             // 必须符合标准的 URL 格式
	APIKey     string   `json:"api_key" binding:"required" validate:"required,min=1,max=255"`                                                                                    // 供应商密钥
	Weight     int      `json:"weight" binding:"required" validate:"required,min=1,max=100"`                                                                                     // 负载权重范围 1-100
	Priority   int      `json:"priority"`                                                                                                                                        // 路由优先级，数值越大越优先
	Tags       []string `json:"tags"`                                                                                                                                            // 路由标签：fast/cheap/stable
	Timeout    int      `json:"timeout" validate:"omitempty,min=1"`                                                                                                              // 请求超时（秒）
	MaxRetries int      `json:"max_retries" validate:"omitempty,min=0"`                                                                                                          // 失败重试次数
}

// UpdateChannelReq 覆盖更新渠道请求入参
type UpdateChannelReq struct {
	Name       string   `json:"name" binding:"required" validate:"required,min=2,max=100"`
	Provider   string   `json:"provider" binding:"required,oneof=openai deepseek anthropic gemini" validate:"required,oneof=openai deepseek anthropic gemini"`
	Protocol   []string `json:"protocol" binding:"required,min=1,dive,oneof=chat responses messages gemini" validate:"required,min=1,dive,oneof=chat responses messages gemini"`
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

// ChannelModelBase 渠道模型公共字段（新增/编辑复用）
// 说明：模型价格改由「价格方案」（model_prices + model_price_bindings）承载，通过 ModelPriceID 绑定；
// 语义：缺省=不改变绑定；0=解绑；>0=绑定到该价格方案。
type ChannelModelBase struct {
	Name            string   `json:"name" binding:"required" validate:"required,min=1,max=128"`           // 对外（逻辑）模型名
	UpstreamModel   string   `json:"upstream_model" binding:"required" validate:"required,min=1,max=128"` // 该渠道上游真实模型名
	ContextLength   *int     `json:"context_length" validate:"omitempty,min=1"`                           // 上下文窗口长度（tokens）
	MaxOutputTokens *int     `json:"max_output_tokens" validate:"omitempty,min=1"`                        // 单次最大输出 tokens
	Capabilities    []string `json:"capabilities"`                                                        // 能力标签 vision/tools/json/reasoning/audio
	ModelPriceID    *int64   `json:"model_price_id" validate:"omitempty,gte=0"`                           // 绑定的价格方案 model_prices.id（缺省=不改绑定 / 0=解绑 / >0=绑定）
	Ratio           *float64 `json:"ratio" validate:"omitempty,gt=0,lte=100"`                             // 计费倍率：1=官方价，0.8=八折；缺省且选择了计价表时默认 1
}

// CreateChannelModelReq 为渠道新增模型请求入参
type CreateChannelModelReq struct {
	ChannelModelBase
	Status string `json:"status" binding:"omitempty,oneof=active disabled" validate:"omitempty,oneof=active disabled"` // 默认 active
}

// UpdateChannelModelReq 编辑渠道模型请求入参
type UpdateChannelModelReq struct {
	ChannelModelBase
	Status string `json:"status" binding:"omitempty,oneof=active disabled" validate:"omitempty,oneof=active disabled"`
}

// UpdateChannelModelStatusReq 启用/禁用渠道模型请求入参
type UpdateChannelModelStatusReq struct {
	Status string `json:"status" binding:"required,oneof=active disabled" validate:"required,oneof=active disabled"` // active-启用 disabled-禁用
}
