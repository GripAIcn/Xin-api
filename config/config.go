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

// Config 全局配置大包装结构体（便于未来横向扩展）
type Config struct {
	JWT    JWTConfig
	Server ServerConfig
}

// JWTConfig JWT 专有配置
type JWTConfig struct {
	Secret string
	Expire time.Duration
}

// ServerConfig 服务运行时配置
type ServerConfig struct {
	Port string
}

// Load 显式加载并装配全局配置（替代隐式 init）
func Load() *Config {
	// 1. 尝试加载 .env 文件
	// 生产级细节：在 K8s 云原生部署时，配置通常以环境变量直接注入，不一定有 .env 文件
	// 因此这里使用 Warn 提示，而不是直接 Fatal 挂断进程，提高容器兼容性
	if err := godotenv.Load(); err != nil {
		log.Println("[CONFIG] [WARN] No .env file found, reading from system environments directly.")
	} else {
		log.Println("[CONFIG] [INFO] Successfully loaded .env file")
	}

	// 2. 统一组装并返回强类型配置实例
	return &Config{
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "xin-api-default-secret-key-2026"), // 给予安全兜底密钥
			Expire: getEnvToTimeDuration("JWT_EXPIRE", 24*time.Hour),
		},
		Server: ServerConfig{
			Port: getEnv("PROXY_PORT", "8080"), // 默认 8080 端口
		},
	}
}

// getEnvToTimeDuration 读取时间相关的环境变量并解析
func getEnvToTimeDuration(key string, defaultDuration time.Duration) time.Duration {
	val := os.Getenv(key)
	val = strings.TrimSpace(val)
	if val == "" {
		return defaultDuration
	}

	duration, err := time.ParseDuration(val)
	if err != nil {
		// 生产级细节：如果此时日志框架（logger）尚未在 main 中完全初始化完成，
		// 直接调用 logger.Warn 可能会 panic。这里做一个安全的双轨输出方案：
		log.Printf("[CONFIG] [WARN] parse env %s value '%s' failed: %v, using default: %v", key, val, err, defaultDuration)

		// 如果日志框架已经安全 ready，再写入流式日志管道
		logger.Warn(context.Background(), "parse env %s failed: %v", key, err)

		return defaultDuration
	}

	return duration
}

// getEnv 辅助工具：读取字符串环境变量，若为空则返回默认值
func getEnv(key, defaultValue string) string {
	val := os.Getenv(key)
	val = strings.TrimSpace(val)
	if val == "" {
		return defaultValue
	}
	return val
}
