package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"Xin-api/config"
	"Xin-api/internal/middleware"
	"Xin-api/internal/router"
	"Xin-api/internal/store/postgresql"
	storeRedis "Xin-api/internal/store/redis"
)

func main() {
	// 1. 加载全局静态配置中心（OLTP 静态业务配置）
	// 假设你的 config 包提供了全局加载函数，如 config.Load()
	cfg := config.Load()

	// 2. 初始化 PostgreSQL 核心驱动
	db := postgresql.NewPostgres()

	// 3. 初始化 Redis 分布式协同层
	rdb := storeRedis.NewRedisClient(cfg.Redis)

	// 4. 构造分布式限流器（数据面使用，暂不挂载路由）
	_ = middleware.NewDistributedLimiter(rdb)

	// 5. 构建无状态控制面 Engine
	gin.SetMode(gin.ReleaseMode) // 生产环境切换为 Release 模式以榨干 Gin 的性能
	r := gin.New()
	r.Use(gin.Recovery()) // 引入崩溃恢复切面，防止单点故障引发多米诺骨牌级雪崩

	// 6. 【控制面】动态路由与核心鉴权切面总装
	router.SetupRouter(r, db, cfg.JWT)

	// ==========================================
	// 7. 生产级优化：优雅启停 (Graceful Shutdown)
	// ==========================================
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port, // 比如 :8080
		Handler: r,
	}

	// 开启常驻协程监听 HTTP 请求，避免阻塞主线程
	go func() {
		log.Printf("[CONTROL-PLANE] Xin-api Admin Server is running on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("网关控制面意外退役: %v", err)
		}
	}()

	// 等待系统中断信号（如 K8s 扩缩容、Ctrl+C、kill 进程）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[CONTROL-PLANE] Shutting down server safely...")

	// 给未能处理完的请求 5 秒的宽限撤退时间 (Context Timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("[CONTROL-PLANE] Xin-api Server exited safely.")
}
