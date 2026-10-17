package config

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/joho/godotenv"
)

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	FailureThreshold int           // 连续失败次数阈值
	RecoveryInterval time.Duration // 自动恢复间隔
}

// ProxyConfig 上游代理配置
type ProxyConfig struct {
	RequestTimeout time.Duration // 上游请求超时
	MaxBodyMB      int           // 最大请求体 MB
}

// LogConfig 日志落库开关（请求/响应原文默认关闭，避免历史数据膨胀）
type LogConfig struct {
	RequestBody  bool // 是否落库 request_payloads.request_body
	ResponseBody bool // 是否落库 request_payloads.response_body
}

// Config 全局配置大包装结构体（便于未来横向扩展）
type Config struct {
	JWT            JWTConfig
	Server         ServerConfig
	Redis          RedisConfig
	Cache          CacheConfig
	CircuitBreaker CircuitBreakerConfig
	Proxy          ProxyConfig
	Log            LogConfig
	Billing        BillingConfig
	Admin          AdminConfig
}

// AdminConfig 初始管理员配置
type AdminConfig struct {
	InitPassword string // 首次初始化 admin 账号时使用的口令，为空则随机生成
}

// BillingConfig 计费配置
type BillingConfig struct {
	USDCNYRate float64 // USD→CNY 结算汇率初始值
}

// CacheConfig 缓存配置
type CacheConfig struct {
	ChannelExpireTime time.Duration // 渠道缓存过期时间
}

// JWTConfig JWT 专有配置
type JWTConfig struct {
	Secret string
	Expire time.Duration
}

// RedisConfig Redis 客户端配置
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	PoolSize int
}

// ServerConfig 服务运行时配置
type ServerConfig struct {
	Port        string
	GatewayNode string // 网关实例标识，落库 request_logs.gateway_node
}

// Load 显式加载并装配全局配置（替代隐式 init）
func Load() *Config {
	// 1. 尝试加载 .env 文件
	// 生产级细节：在 K8s 云原生部署时，配置通常以环境变量直接注入，不一定有 .env 文件
	// 因此这里使用 Warn 提示，而不是直接 Fatal 挂断进程，提高容器兼容性
	if err := godotenv.Load(".env"); err != nil {
		log.Println("[CONFIG] [WARN] No .env file found, reading from system environments directly.")
	} else {
		log.Println("[CONFIG] [INFO] Successfully loaded .env file")
	}

	// 2. 统一组装并返回强类型配置实例
	return &Config{
		JWT: JWTConfig{
			Secret: mustGetEnv("JWT_SECRET"), // 不再提供默认密钥，未配置则拒绝启动
			Expire: getEnvToTimeDuration("JWT_EXPIRE", 24*time.Hour),
		},
		Server: ServerConfig{
			Port:        getEnv("PROXY_PORT", "8080"), // 默认 8080 端口
			GatewayNode: getEnv("GATEWAY_NODE", defaultGatewayNode()),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "127.0.0.1"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvToInt("REDIS_DB", 0),
			PoolSize: getEnvToInt("REDIS_POOL_SIZE", 20),
		},
		Cache: CacheConfig{
			ChannelExpireTime: getEnvToTimeDuration("CACHE_CHANNEL_EXPIRE", 5*time.Minute),
		},
		CircuitBreaker: CircuitBreakerConfig{
			FailureThreshold: getEnvToInt("CB_FAILURE_THRESHOLD", 5),
			RecoveryInterval: getEnvToTimeDuration("CB_RECOVERY_INTERVAL", 60*time.Second),
		},
		Proxy: ProxyConfig{
			RequestTimeout: getEnvToTimeDuration("PROXY_REQUEST_TIMEOUT", 120*time.Second),
			MaxBodyMB:      getEnvToInt("PROXY_MAX_BODY_MB", 10),
		},
		Log: LogConfig{
			RequestBody:  getEnvToBool("LOG_REQUEST_BODY", false),
			ResponseBody: getEnvToBool("LOG_RESPONSE_BODY", false),
		},
		Billing: BillingConfig{
			USDCNYRate: getEnvToFloat("USD_CNY_RATE", 7.2),
		},
		Admin: AdminConfig{
			InitPassword: getEnv("ADMIN_INIT_PASSWORD", ""),
		},
	}
}

// mustGetEnv 读取必填环境变量，缺失时直接终止启动，避免使用可预测的默认密钥
func mustGetEnv(key string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		log.Fatalf("[CONFIG] [FATAL] required env %s is not set", key)
	}
	if len(val) < 32 {
		log.Printf("[CONFIG] [WARN] %s is shorter than 32 characters, please use a strong random secret", key)
	}
	return val
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

// getEnvToInt 读取整数型环境变量
func getEnvToInt(key string, defaultValue int) int {
	val := os.Getenv(key)
	val = strings.TrimSpace(val)
	if val == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		log.Printf("[CONFIG] [WARN] parse env %s value '%s' to int failed: %v, using default: %d", key, val, err, defaultValue)
		return defaultValue
	}
	return n
}

// getEnvToFloat 读取浮点型环境变量，非法或非正值回落默认值
func getEnvToFloat(key string, defaultValue float64) float64 {
	val := os.Getenv(key)
	val = strings.TrimSpace(val)
	if val == "" {
		return defaultValue
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil || f <= 0 {
		log.Printf("[CONFIG] [WARN] parse env %s value '%s' to float failed: %v, using default: %v", key, val, err, defaultValue)
		return defaultValue
	}
	return f
}

// getEnvToBool 读取布尔型环境变量
func getEnvToBool(key string, defaultValue bool) bool {
	val := os.Getenv(key)
	val = strings.TrimSpace(val)
	if val == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		log.Printf("[CONFIG] [WARN] parse env %s value '%s' to bool failed: %v, using default: %t", key, val, err, defaultValue)
		return defaultValue
	}
	return b
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

// defaultGatewayNode 默认使用主机名作为网关实例标识（K8s 下通常为 Pod 名）
func defaultGatewayNode() string {
	if hostname, err := os.Hostname(); err == nil && strings.TrimSpace(hostname) != "" {
		return hostname
	}
	return "xin-api"
}
