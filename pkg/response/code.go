// Package response 统一控制面与数据面的 HTTP 响应契约。
//
// 控制面（管理后台）使用 AdminResponse 包裹业务数据，业务失败统一以 HTTP 200 +
// 业务码返回，由前端根据 code 集中拦截；数据面（OpenAI / Gemini 兼容接口）则使用
// 各协议的官方错误结构，并映射到语义一致的 HTTP 状态码。
package response

import "net/http"

// 业务状态码定义
const (
	Success       = 20000
	InvalidParams = 40001
	Unauthorized  = 40101
	Forbidden     = 40301
	// PasswordChangeRequired 首次登录或密码被管理员重置后，必须先修改密码
	PasswordChangeRequired = 40302
	URLNotFound            = 40401
	NotFound               = 40402 // 业务资源不存在（如指定的日志、价格方案等记录缺失）
	Conflict               = 40901 // 业务资源冲突（如用户名、方案名重复）

	RateLimited        = 42901
	ServiceUnavailable = 50300
	CircuitBreakerOpen = 50301
	UpstreamTimeout    = 50401
	InternalError      = 50001
)

var msgMap = map[int]string{
	Success:                "success",
	InvalidParams:          "请求参数解析失败或格式不正确",
	Unauthorized:           "无效的 API-Key/Token，鉴权未通过",
	Forbidden:              "鉴权成功但无权限访问该资源",
	PasswordChangeRequired: "首次登录或密码被重置后，必须先修改密码",
	URLNotFound:            "请求的路由路径不存在",
	NotFound:               "请求的资源不存在或已被删除",
	Conflict:               "请求的资源已存在或当前状态冲突",
	RateLimited:            "已触发全局限流保护，当前令牌桶枯竭",
	ServiceUnavailable:     "当前没有可用的上游渠道，请检查渠道配置或等待熔断恢复",
	CircuitBreakerOpen:     "上游物理渠道异常率触发阈值，熔断状态机已瞬间隔离",
	UpstreamTimeout:        "上游大模型供应商响应超时，请检查渠道健康度",
	InternalError:          "内部核心故障，请检查基础设施",
}

// Msg 获取状态码描述
func Msg(code int) string {
	if msg, exists := msgMap[code]; exists {
		return msg
	}
	return "未知网关错误"
}

// httpStatusMap 业务状态码到 HTTP 状态码的映射表
var httpStatusMap = map[int]int{
	Success:                http.StatusOK,
	InvalidParams:          http.StatusBadRequest,
	Unauthorized:           http.StatusUnauthorized,
	Forbidden:              http.StatusForbidden,
	PasswordChangeRequired: http.StatusForbidden,
	URLNotFound:            http.StatusNotFound,
	NotFound:               http.StatusNotFound,
	Conflict:               http.StatusConflict,
	RateLimited:            http.StatusTooManyRequests,
	ServiceUnavailable:     http.StatusServiceUnavailable,
	CircuitBreakerOpen:     http.StatusServiceUnavailable,
	UpstreamTimeout:        http.StatusGatewayTimeout,
	InternalError:          http.StatusInternalServerError,
}

// MapToHTTPStatus 将业务状态码映射到标准的 HTTP 状态码（供数据面使用）
func MapToHTTPStatus(bizCode int) int {
	if status, exists := httpStatusMap[bizCode]; exists {
		return status
	}
	return http.StatusInternalServerError
}
