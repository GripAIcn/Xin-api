package form

// CreateModelPriceReq 新增价格方案请求入参
type CreateModelPriceReq struct {
	Name               string  `json:"name" binding:"required" validate:"required,min=1,max=128"` // 价格方案名（全局唯一）
	InputPrice         float64 `json:"input_price" validate:"gte=0"`                               // 普通输入单价
	OutputPrice        float64 `json:"output_price" validate:"gte=0"`                              // 输出单价
	CachedInputPrice   float64 `json:"cached_input_price" validate:"gte=0"`                        // 输入缓存命中单价
	CacheCreationPrice float64 `json:"cache_creation_price" validate:"gte=0"`                      // 输入缓存创建单价
	Currency           string  `json:"currency" validate:"required,min=1,max=16"`                  // 币种（默认 USD）
}

// UpdateModelPriceReq 编辑价格方案请求入参（整体覆盖）
type UpdateModelPriceReq struct {
	Name               string  `json:"name" binding:"required" validate:"required,min=1,max=128"`
	InputPrice         float64 `json:"input_price" validate:"gte=0"`
	OutputPrice        float64 `json:"output_price" validate:"gte=0"`
	CachedInputPrice   float64 `json:"cached_input_price" validate:"gte=0"`
	CacheCreationPrice float64 `json:"cache_creation_price" validate:"gte=0"`
	Currency           string  `json:"currency" validate:"required,min=1,max=16"`
	Status             string  `json:"status" binding:"omitempty,oneof=active disabled" validate:"omitempty,oneof=active disabled"` // 可选，非空时覆盖
}

// UpdateModelPriceStatusReq 启用/禁用价格方案请求入参
type UpdateModelPriceStatusReq struct {
	Status string `json:"status" binding:"required,oneof=active disabled" validate:"required,oneof=active disabled"` // active-启用 disabled-禁用
}
