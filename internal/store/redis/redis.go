package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"Xin-api/config"
)

func NewRedisClient(cfg config.RedisConfig) *redis.Client {
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		panic(fmt.Sprintf("连接 Redis 失败 (%s): %v", addr, err))
	}

	log.Printf("[REDIS] 连接成功 %s (db=%d pool=%d)", addr, cfg.DB, cfg.PoolSize)
	return rdb
}
