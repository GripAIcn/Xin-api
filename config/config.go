package config

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/joho/godotenv"
)

type JWTConfig struct {
	Secret string
	Expire time.Duration
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	log.Println("Successfully loaded .env file")

	//{
	//Secret: os.Getenv("JWT_SECRET"),
	//	Expire: GetEnvToTimeDuration("JWT_EXPIRE", 24*time.Hour),
	//}

}

func GetEnvToTimeDuration(key string, defaultDuration time.Duration) time.Duration {
	val := os.Getenv(key)
	val = strings.TrimSpace(val)
	if val == "" {
		return defaultDuration // 环境变量未配置时，返回兜底默认值
	}

	duration, err := time.ParseDuration(val)
	if err != nil {
		// 生产级细节：解析失败时记录 Warn 日志，并用默认值兜底，保证网关不崩溃
		logger.Warn(context.Background(), "parse env %s value '%s' to duration failed: %v, using default: %v", key, val, err, defaultDuration)
		return defaultDuration
	}

	return duration
}
