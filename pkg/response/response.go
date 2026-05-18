package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ==========================================
// 1. 控制面 (Control Plane) 标准响应结构
// ==========================================

// AdminResponse 控制面统一 JSON 返回体
type AdminResponse[T any] struct {
	Code    int    `json:"code"`           // 业务自定义状态码
	Message string `json:"message"`        // 提示信息
	Data    T      `json:"data,omitempty"` // 泛型数据内容
}

// PageData 通用分页结构体（便于大盘及列表展示）
type PageData[T any] struct {
	List     []T   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// AdminSuccess 控制面成功返回
func AdminSuccess[T any](c *gin.Context, data T) {
	c.JSON(http.StatusOK, AdminResponse[T]{
		Code:    Success,
		Message: Msg(Success),
		Data:    data,
	})
}

// AdminFail 控制面业务异常返回
func AdminFail(c *gin.Context, bizCode int) {
	c.JSON(http.StatusOK, AdminResponse[any]{
		Code:    bizCode,
		Message: Msg(bizCode),
	})
}

// ==========================================
// 2. 数据面 (Data Plane) OpenAI 兼容错误结构
// ==========================================

// OpenAIErrorBody 严格契合 OpenAI 官方 SDK 的错误内嵌结构
type OpenAIErrorBody struct {
	Message string  `json:"message"`         // 错误描述
	Type    string  `json:"type"`            // 错误类型，例如 invalid_request_error, insuﬃcient_quota
	Param   *string `json:"param,omitempty"` // 触发错误的参数项
	Code    string  `json:"code,omitempty"`  // 字符串型错误标识码
}

// OpenAIErrorResponse 数据面统一标准错误返回体
type OpenAIErrorResponse struct {
	Error OpenAIErrorBody `json:"error"`
}

// DataPlaneError 数据面专属错误注入器（自动映射 HTTP 状态码并 Abort）
func DataPlaneError(c *gin.Context, bizCode int) {
	httpStatus := MapToHTTPStatus(bizCode)
	errType := "api_error"
	errCode := "gateway_internal_error"

	// 根据不同的网关业务状态，转换为标准的 OpenAI 错误范式
	switch bizCode {
	case Unauthorized:
		errType = "authentication_error"
		errCode = "invalid_api_key"
	case RateLimited:
		errType = "requests_error"
		errCode = "rate_limit_exceeded"
	case CircuitBreakerOpen, UpstreamTimeout:
		errType = "server_error"
		errCode = "upstream_service_unavailable"
	case InvalidParams:
		errType = "invalid_request_error"
		errCode = "parse_body_failed"
	}

	c.JSON(httpStatus, OpenAIErrorResponse{
		Error: OpenAIErrorBody{
			Message: Msg(bizCode),
			Type:    errType,
			Code:    errCode,
		},
	})

	// 核心：直接阻断 Gin 中间件链条后续逻辑，防止错上加错
	c.Abort()
}
