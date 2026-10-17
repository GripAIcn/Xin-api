package middleware

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"Xin-api/pkg/response"
)

const tokenBucketScript = `
local key = KEYS[1]
local max_tokens = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local req_tokens = tonumber(ARGV[3])
local now = tonumber(ARGV[4])

local bucket = redis.call('hmget', key, 'tokens', 'last_updated')
local tokens = tonumber(bucket[1])
local last_updated = tonumber(bucket[2])

if not tokens then
    tokens = max_tokens
    last_updated = now
else
    local elapsed = now - last_updated
    if elapsed > 0 then
        tokens = math.min(max_tokens, tokens + elapsed * refill_rate)
        last_updated = now
    end
end

if tokens >= req_tokens then
    tokens = tokens - req_tokens
    redis.call('hmset', key, 'tokens', tokens, 'last_updated', last_updated)
    redis.call('expire', key, 86400)
    return 1
else
    return 0
end
`

type DistributedLimiter struct {
	rdb *redis.Client
}

func NewDistributedLimiter(rdb *redis.Client) *DistributedLimiter {
	return &DistributedLimiter{rdb: rdb}
}

// LimitMiddleware 返回一个 Gin 中间件，进行分布式令牌桶限流。
// maxTokens：桶容量；refillRate：每秒补充令牌数；reqTokens：每次请求消耗令牌数。
func (dl *DistributedLimiter) LimitMiddleware(maxTokens, refillRate, reqTokens int) gin.HandlerFunc {
	return func(c *gin.Context) {
		ok, err := dl.allow(c.Request.Context(), "XinAPI:limiter:global", maxTokens, refillRate, reqTokens)
		if err != nil || !ok {
			response.DataPlaneError(c, response.RateLimited)
			return
		}

		c.Next()
	}
}

// LoginRateLimit 登录接口限流：按客户端 IP 维护独立令牌桶，防止暴力破解与撞库。
// 超限时使用控制面响应契约返回业务码并中断请求。
func (dl *DistributedLimiter) LoginRateLimit(maxTokens, refillRate, reqTokens int) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := "XinAPI:limiter:login:" + c.ClientIP()
		ok, err := dl.allow(c.Request.Context(), key, maxTokens, refillRate, reqTokens)
		if err != nil || !ok {
			response.AdminFail(c, response.RateLimited)
			c.Abort()
			return
		}

		c.Next()
	}
}

// allow 执行一次令牌桶扣减；返回 false 表示令牌不足
func (dl *DistributedLimiter) allow(ctx context.Context, key string, maxTokens, refillRate, reqTokens int) (bool, error) {
	now := time.Now().Unix()
	args := []any{maxTokens, refillRate, reqTokens, now}

	evalCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()

	res, err := dl.rdb.Eval(evalCtx, tokenBucketScript, []string{key}, args...).Result()
	if err != nil {
		return false, err
	}
	allowed, ok := res.(int64)
	if !ok {
		return false, nil
	}
	return allowed == 1, nil
}
