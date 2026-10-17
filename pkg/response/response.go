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

// AdminSuccessMsg 控制面无数据成功返回（用于删除、修改等仅需提示结果的操作）
func AdminSuccessMsg(c *gin.Context, message string) {
	if message == "" {
		message = Msg(Success)
	}
	c.JSON(http.StatusOK, AdminResponse[any]{
		Code:    Success,
		Message: message,
	})
}

// AdminFail 控制面业务异常返回（使用状态码默认文案）
func AdminFail(c *gin.Context, bizCode int) {
	AdminFailMsg(c, bizCode, Msg(bizCode))
}

// AdminFailMsg 控制面业务异常返回，支持按场景覆盖提示文案
func AdminFailMsg(c *gin.Context, bizCode int, message string) {
	if message == "" {
		message = Msg(bizCode)
	}
	c.JSON(http.StatusOK, AdminResponse[any]{
		Code:    bizCode,
		Message: message,
	})
}

// ==========================================
// 2. 数据面 (Data Plane) OpenAI 兼容错误结构
// ==========================================

// OpenAIErrorBody 严格契合 OpenAI 官方 SDK 的错误内嵌结构
type OpenAIErrorBody struct {
	Message string  `json:"message"`         // 错误描述
	Type    string  `json:"type"`            // 错误类型，例如 invalid_request_error, insufficient_quota
	Param   *string `json:"param,omitempty"` // 触发错误的参数项
	Code    string  `json:"code,omitempty"`  // 字符串型错误标识码
}

// OpenAIErrorResponse 数据面统一标准错误返回体
type OpenAIErrorResponse struct {
	Error OpenAIErrorBody `json:"error"`
}

// BuildOpenAIError 生成数据面 OpenAI 风格错误体与对应 HTTP 状态码（供数据面注入器与入站协议编码复用）
func BuildOpenAIError(bizCode int) (int, OpenAIErrorResponse) {
	return BuildOpenAIErrorMsg(bizCode, Msg(bizCode))
}

// BuildOpenAIErrorMsg 生成数据面 OpenAI 风格错误体，支持按场景覆盖提示文案
func BuildOpenAIErrorMsg(bizCode int, message string) (int, OpenAIErrorResponse) {
	if message == "" {
		message = Msg(bizCode)
	}

	httpStatus := MapToHTTPStatus(bizCode)
	errType, errCode := openAIErrorMeta(bizCode)

	return httpStatus, OpenAIErrorResponse{
		Error: OpenAIErrorBody{
			Message: message,
			Type:    errType,
			Code:    errCode,
		},
	}
}

// openAIErrorMeta 将网关业务状态码转换为标准的 OpenAI 错误 type / code
func openAIErrorMeta(bizCode int) (errType, errCode string) {
	switch bizCode {
	case Unauthorized:
		return "authentication_error", "invalid_api_key"
	case Forbidden:
		return "permission_error", "permission_denied"
	case URLNotFound, NotFound:
		return "invalid_request_error", "not_found"
	case Conflict:
		return "invalid_request_error", "resource_conflict"
	case InvalidParams:
		return "invalid_request_error", "parse_body_failed"
	case RateLimited:
		return "rate_limit_error", "rate_limit_exceeded"
	case ServiceUnavailable:
		return "server_error", "service_unavailable"
	case CircuitBreakerOpen:
		return "server_error", "circuit_breaker_open"
	case UpstreamTimeout:
		return "server_error", "upstream_timeout"
	default:
		return "server_error", "gateway_internal_error"
	}
}

// DataPlaneError 数据面专属错误注入器（自动映射 HTTP 状态码并 Abort）
func DataPlaneError(c *gin.Context, bizCode int) {
	DataPlaneErrorMessage(c, bizCode, "")
}

// DataPlaneErrorMessage 数据面专属错误注入器，支持按场景覆盖提示文案
func DataPlaneErrorMessage(c *gin.Context, bizCode int, message string) {
	httpStatus, body := BuildOpenAIErrorMsg(bizCode, message)
	c.JSON(httpStatus, body)

	// 核心：直接阻断 Gin 中间件链条后续逻辑，防止错上加错
	c.Abort()
}
