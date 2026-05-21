package response

import "net/http"

// 业务状态码定义
const (
	Success             = 20000
	InvalidParams       = 40001
	Unauthorized        = 40101
	GroupQuotaExhausted = 40201
	Forbidden           = 40301
	URLNotFound         = 40401

	RateLimited        = 42901
	ServiceUnavailable = 50300
	CircuitBreakerOpen = 50301
	UpstreamTimeout    = 50401
	InternalError      = 50001
)

var msgMap = map[int]string{
	Success:             "success",
	InvalidParams:       "请求参数解析失败或格式不正确",
	Unauthorized:        "无效的 API-Key/Token，鉴权未通过",
	GroupQuotaExhausted: "当前业务组代币额度已耗尽",
	Forbidden:           "鉴权成功但无权限访问该物理渠道",
	URLNotFound:         "请求的路由路径不存在",
	RateLimited:         "已触发全局限流保护，当前业务组令牌桶枯竭",
	ServiceUnavailable:  "当前没有可用的上游渠道，请检查渠道配置或等待熔断恢复",
	CircuitBreakerOpen:  "上游物理渠道异常率触发阈值，熔断状态机已瞬间隔离",
	UpstreamTimeout:     "上游大模型供应商响应超时，请检查渠道健康度",
	InternalError:       "内部核心故障，请检查基础设施",
}

// Msg 获取状态码描述
func Msg(code int) string {
	if msg, exists := msgMap[code]; exists {
		return msg
	}
	return "Unknown Gateway Error"
}

// MapToHTTPStatus 将业务状态码映射到标准的 HTTP 状态码（供数据面使用）
func MapToHTTPStatus(bizCode int) int {
	switch bizCode {
	case Success:
		return http.StatusOK
	case InvalidParams:
		return http.StatusBadRequest
	case Unauthorized:
		return http.StatusUnauthorized
	case Forbidden:
		return http.StatusForbidden
	case RateLimited:
		return http.StatusTooManyRequests
	case ServiceUnavailable:
		return http.StatusServiceUnavailable
	case CircuitBreakerOpen:
		return http.StatusServiceUnavailable
	case UpstreamTimeout:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}
