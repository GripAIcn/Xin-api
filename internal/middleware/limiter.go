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

// LimitMiddleware 返回一个 Gin 中间件，按业务组进行分布式令牌桶限流。
// maxTokens：桶容量；refillRate：每秒补充令牌数；reqTokens：每次请求消耗令牌数。
func (dl *DistributedLimiter) LimitMiddleware(maxTokens, refillRate, reqTokens int) gin.HandlerFunc {
	return func(c *gin.Context) {
		groupID := c.GetString("group_id")
		if groupID == "" {
			response.DataPlaneError(c, response.InternalError)
			return
		}

		redisKey := "XinAPI:limiter:group:" + groupID
		now := time.Now().Unix()
		args := []any{maxTokens, refillRate, reqTokens, now}

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		res, err := dl.rdb.Eval(ctx, tokenBucketScript, []string{redisKey}, args...).Result()
		if err != nil || res.(int64) == 0 {
			response.DataPlaneError(c, response.RateLimited)
			return
		}

		c.Next()
	}
}
