package form

import "Xin-api/internal/model"

// CreateModelPriceReq 新增价格方案请求入参
// 金额使用 decimal 解析，非负校验在 handler 中完成。
type CreateModelPriceReq struct {
	Name               string      `json:"name" binding:"required" validate:"required,min=1,max=128"` // 价格方案名（全局唯一）
	InputPrice         model.Money `json:"input_price"`                                               // 普通输入单价
	OutputPrice        model.Money `json:"output_price"`                                              // 输出单价
	CachedInputPrice   model.Money `json:"cached_input_price"`                                        // 输入缓存命中单价
	CacheCreationPrice model.Money `json:"cache_creation_price"`                                      // 输入缓存创建单价
	Currency           string      `json:"currency" validate:"required,min=1,max=16"`                 // 币种（默认 USD）
}

// UpdateModelPriceReq 编辑价格方案请求入参（整体覆盖）
type UpdateModelPriceReq struct {
	Name               string      `json:"name" binding:"required" validate:"required,min=1,max=128"`
	InputPrice         model.Money `json:"input_price"`
	OutputPrice        model.Money `json:"output_price"`
	CachedInputPrice   model.Money `json:"cached_input_price"`
	CacheCreationPrice model.Money `json:"cache_creation_price"`
	Currency           string      `json:"currency" validate:"required,min=1,max=16"`
	Status             string      `json:"status" binding:"omitempty,oneof=active disabled" validate:"omitempty,oneof=active disabled"` // 可选，非空时覆盖
}

// UpdateModelPriceStatusReq 启用/禁用价格方案请求入参
type UpdateModelPriceStatusReq struct {
	Status string `json:"status" binding:"required,oneof=active disabled" validate:"required,oneof=active disabled"` // active-启用 disabled-禁用
}
