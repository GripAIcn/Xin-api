package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"Xin-api/config"
	"Xin-api/internal/store/postgresql"
)

// ChannelBreaker 熔断器接口
type ChannelBreaker interface {
	IsHealthy(ctx context.Context, channelID int64) (bool, error)
	Record(ctx context.Context, channelID int64, err error)
}

const (
	cbFailCountKey   = "cb:%d:fail_count"
	cbLastFailKey    = "cb:%d:last_fail_time"
	cbLastSuccessKey = "cb:%d:last_success_time"
)

// CircuitBreaker Redis 支持的被动式熔断器
type CircuitBreaker struct {
	rdb         *redis.Client
	channelRepo postgresql.ChannelRepo
	cfg         config.CircuitBreakerConfig
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(rdb *redis.Client, channelRepo postgresql.ChannelRepo, cfg config.CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		rdb:         rdb,
		channelRepo: channelRepo,
		cfg:         cfg,
	}
}

// IsHealthy 检查渠道是否健康（查 DB 中 status 是否为 1）
func (cb *CircuitBreaker) IsHealthy(ctx context.Context, channelID int64) (bool, error) {
	ch, err := cb.channelRepo.GetByID(ctx, channelID)
	if err != nil || ch == nil {
		return false, err
	}
	return ch.Status == 1, nil
}

// Record 记录请求结果
// 成功：重置失败计数，更新最后成功时间
// 失败：递增失败计数，达阈值则将渠道 status 更新为 2（自动熔断）
func (cb *CircuitBreaker) Record(ctx context.Context, channelID int64, err error) {
	if err == nil {
		cb.rdb.Del(ctx, fmt.Sprintf(cbFailCountKey, channelID))
		cb.rdb.Set(ctx, fmt.Sprintf(cbLastSuccessKey, channelID), time.Now().Unix(), 0)
		return
	}

	failKey := fmt.Sprintf(cbFailCountKey, channelID)
	count := cb.rdb.Incr(ctx, failKey).Val()
	cb.rdb.Set(ctx, fmt.Sprintf(cbLastFailKey, channelID), time.Now().Unix(), 0)
	log.Printf("[CB] channel %d fail count: %d", channelID, count)

	if count >= int64(cb.cfg.FailureThreshold) {
		log.Printf("[CB] channel %d failure threshold reached, opening circuit", channelID)
		if updateErr := cb.channelRepo.UpdateStatus(ctx, channelID, 2); updateErr != nil {
			log.Printf("[CB] update channel %d status to 2 failed: %v", channelID, updateErr)
		} else {
			log.Printf("[CB] channel %d circuit opened (status=2)", channelID)
			cb.rdb.Del(ctx, failKey)
		}
	}
}

// RecoveryLoop 后台协程：定期扫描熔断渠道并尝试恢复
func (cb *CircuitBreaker) RecoveryLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("[CB] recovery loop stopped")
			return
		case <-ticker.C:
			cb.recover(ctx)
		}
	}
}

// recover 扫描 Redis 中记录的熔断渠道，冷却期到期后尝试恢复为 Status=1
func (cb *CircuitBreaker) recover(ctx context.Context) {
	keys, err := cb.rdb.Keys(ctx, "cb:*:last_fail_time").Result()
	if err != nil {
		return
	}
	for _, key := range keys {
		var channelID int64
		fmt.Sscanf(key, "cb:%d:last_fail_time", &channelID)
		if channelID == 0 {
			continue
		}
		lastFailStr, err := cb.rdb.Get(ctx, key).Result()
		if err != nil {
			continue
		}
		var lastFailTime int64
		fmt.Sscanf(lastFailStr, "%d", &lastFailTime)
		since := time.Since(time.Unix(lastFailTime, 0))

		if since >= cb.cfg.RecoveryInterval {
			ch, err := cb.channelRepo.GetByID(ctx, channelID)
			if err != nil || ch == nil || ch.Status != 2 {
				continue
			}
			if updateErr := cb.channelRepo.UpdateStatus(ctx, channelID, 1); updateErr == nil {
				log.Printf("[CB] channel %d recovered (status=1)", channelID)
				cb.rdb.Del(ctx, fmt.Sprintf(cbFailCountKey, channelID))
				cb.rdb.Del(ctx, key)
			}
		}
	}
}
