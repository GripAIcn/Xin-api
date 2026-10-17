package middleware

import (
	"github.com/gin-gonic/gin"

	"Xin-api/internal/requestctx"
)

const (
	// CtxRequestIDKey 请求 ID 在 gin.Context 中的键
	CtxRequestIDKey = "request_id"
	// HeaderRequestID 响应头中回写的请求 ID
	HeaderRequestID = "X-Request-Id"
)

// RequestContextMiddleware 为每个请求生成 xin_req_* 请求 ID，
// 装配 RequestContext 并回写响应头，供日志与排障串联同一次请求。
func RequestContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := requestctx.NewRequestID()
		c.Set(CtxRequestIDKey, requestID)
		requestctx.Set(c, requestctx.New(requestID))
		c.Header(HeaderRequestID, requestID)
		c.Next()
	}
}
