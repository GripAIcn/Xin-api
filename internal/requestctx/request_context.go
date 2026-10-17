// Package requestctx 维护数据面单次请求的生命周期上下文。
// 目标：request_id / user_id / api_key_id / model / protocol / channel / provider 等
// 只在入口处装配一次，后续链路统一从这里读取，避免长参数列表层层传递。
package requestctx

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ginContextKey RequestContext 在 gin.Context 中的存储键
const ginContextKey = "xin_request_context"

// RequestIDPrefix Xin API 请求 ID 前缀
const RequestIDPrefix = "xin_req_"

// RequestContext 单次请求的生命周期上下文
type RequestContext struct {
	RequestID string

	UserID   int64
	APIKeyID int64

	Model    string
	Protocol string

	ChannelID int64
	Provider  string

	StartTime time.Time

	IsStream bool
}

// NewRequestID 生成全局唯一的 Xin API 请求 ID（时间有序 UUIDv7，去掉连字符）
func NewRequestID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return RequestIDPrefix + strings.ReplaceAll(uuid.NewString(), "-", "")
	}
	return RequestIDPrefix + strings.ReplaceAll(id.String(), "-", "")
}

// New 创建请求上下文；StartTime 取当前时间
func New(requestID string) *RequestContext {
	return &RequestContext{
		RequestID: requestID,
		StartTime: time.Now(),
	}
}

// Set 将请求上下文写入 gin.Context
func Set(c *gin.Context, rc *RequestContext) {
	c.Set(ginContextKey, rc)
}

// Get 读取请求上下文；不存在返回 nil
func Get(c *gin.Context) *RequestContext {
	v, ok := c.Get(ginContextKey)
	if !ok {
		return nil
	}
	rc, _ := v.(*RequestContext)
	return rc
}

// Ensure 返回当前请求上下文；缺失时兜底创建，保证 Handler 内可直接使用
func Ensure(c *gin.Context) *RequestContext {
	if rc := Get(c); rc != nil {
		return rc
	}
	rc := New(NewRequestID())
	Set(c, rc)
	return rc
}
