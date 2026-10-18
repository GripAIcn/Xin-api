package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"Xin-api/config"
	"Xin-api/internal/middleware"
	"Xin-api/internal/router"
	"Xin-api/internal/service"
	"Xin-api/internal/store/postgresql"
	storeRedis "Xin-api/internal/store/redis"
)

func main() {
	// 1. 加载全局静态配置中心（OLTP 静态业务配置）
	// 假设你的 config 包提供了全局加载函数，如 config.Load()
	cfg := config.Load()

	// 2. 初始化 PostgreSQL 核心驱动
	db := postgresql.NewPostgres(cfg.Billing.USDCNYRate, cfg.Admin.InitPassword)

	// 3. 初始化 Redis 分布式协同层
	rdb := storeRedis.NewRedisClient(cfg.Redis)

	// 4. 数据面组件初始化
	channelRepo := postgresql.NewChannelRepo(db)

	breaker := service.NewCircuitBreaker(rdb, channelRepo, cfg.CircuitBreaker)

	// 启动熔断恢复协程
	go breaker.RecoveryLoop(context.Background(), cfg.CircuitBreaker.RecoveryInterval)

	// 5.1 渠道指标聚合：启动回填最近 7 天，之后每 60 秒增量重算
	metricRepo := postgresql.NewChannelMetricRepo(db)
	metricAggregator := service.NewChannelMetricAggregator(metricRepo)
	metricCtx, cancelMetrics := context.WithCancel(context.Background())
	defer cancelMetrics()
	go metricAggregator.Run(metricCtx)

	// 6. 构建无状态控制面 Engine
	//gin.SetMode(gin.ReleaseMode) // 生产环境切换为 Release 模式以榨干 Gin 的性能
	r := gin.New()
	r.Use(gin.Recovery()) // 引入崩溃恢复切面，防止单点故障引发多米诺骨牌级雪崩

	// 请求上下文中间件：生成 xin_req_* 请求 ID 并回写 X-Request-Id 响应头
	r.Use(middleware.RequestContextMiddleware())

	// CORS 中间件 — 允许前端开发服务器跨域访问
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:9090", "http://127.0.0.1:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "x-goog-api-key"},
		ExposeHeaders:    []string{middleware.HeaderRequestID},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// 6. 【控制面】动态路由与核心鉴权切面总装
	router.SetupRouter(r, db, *cfg, breaker, rdb, metricAggregator)

	// 7. 静态前端 SPA (生产模式)
	// 静态资源缓存策略：必须在静态路由注册前声明。
	// /assets 等文件名带内容哈希，可长期强缓存；HTML 入口必须每次回源校验，
	// 否则发版后会继续引用已被删除的旧 chunk。
	r.Use(func(c *gin.Context) {
		path := c.Request.URL.Path
		switch {
		case strings.HasPrefix(path, "/assets/"),
			strings.HasPrefix(path, "/fonts/"),
			strings.HasPrefix(path, "/brand/"):
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		case path == "/og.png", path == "/favicon.ico", path == "/apple-touch-icon.png":
			c.Header("Cache-Control", "public, max-age=604800")
		case path == "/", path == "/features", path == "/docs", path == "/faq", path == "/contact":
			c.Header("Cache-Control", "no-cache")
		}
		c.Next()
	})
	r.StaticFile("/", "./web/dist/index.html")
	// 预渲染的公开页面：与 web/scripts/prerender.ts 的 PUBLIC_PAGES 保持一致。
	// 必须显式注册，否则会被 NoRoute 兜底成控制台外壳（带 noindex），无法被搜索引擎收录
	for _, pagePath := range []string{"/features", "/docs", "/faq", "/contact"} {
		r.StaticFile(pagePath, "./web/dist"+pagePath+"/index.html")
	}
	r.Static("/assets", "./web/dist/assets")
	// 本地字体等 public 静态资源，避免被 NoRoute 回退为 HTML
	r.Static("/fonts", "./web/dist/fonts")
	// 根目录静态文件：SEO 与站点图标，必须在 NoRoute 之前显式注册，
	// 否则会被 SPA 兜底成 HTML 外壳（Content-Type 也会变成 text/html）
	r.StaticFile("/robots.txt", "./web/dist/robots.txt")
	r.StaticFile("/sitemap.xml", "./web/dist/sitemap.xml")
	r.StaticFile("/favicon.ico", "./web/dist/favicon.ico")
	r.StaticFile("/apple-touch-icon.png", "./web/dist/apple-touch-icon.png")
	r.StaticFile("/og.png", "./web/dist/og.png")
	r.Static("/brand", "./web/dist/brand")
	r.NoRoute(func(c *gin.Context) {
		// 带静态后缀的路径一律不兜底：避免把 HTML 外壳当作资源返回（软 404 / 重复内容），
		// 同时保证站长平台的文件验证（如 baidu_verify_xxx.html）能独立生效
		if filepath.Ext(c.Request.URL.Path) != "" {
			c.Status(http.StatusNotFound)
			return
		}
		c.File("./web/dist/app.html")
	})

	// ==========================================
	// 8. 生产级优化：优雅启停 (Graceful Shutdown)
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
	cancelMetrics() // 停止渠道指标聚合协程

	// 给未能处理完的请求 5 秒的宽限撤退时间 (Context Timeout)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("[CONTROL-PLANE] Xin-api Server exited safely.")
}
