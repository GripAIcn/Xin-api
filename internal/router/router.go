package router

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"Xin-api/config"
	"Xin-api/internal/adapter"
	"Xin-api/internal/handler"
	"Xin-api/internal/middleware"
	"Xin-api/internal/service"
	"Xin-api/internal/store/postgresql"
)

// SetupRouter 统一注册控制面管理后台路由
func SetupRouter(r *gin.Engine, db *gorm.DB, cfg config.Config,
	breaker *service.CircuitBreaker, rdb *redis.Client) {
	// 1. 初始化 Handler（注入底层的 DB 仓库和配置）
	userRepo := postgresql.NewUserRepo(db)
	userHandler := handler.NewUserHandler(userRepo, cfg.JWT)
	apiKeyRepo := postgresql.NewApiKeyRepo(db)
	apiKeyHandler := handler.NewApiKeyHandler(apiKeyRepo)
	channelRepo := postgresql.NewChannelRepo(db)
	channelHandler := handler.NewChannelHandler(channelRepo, rdb)
	balancer := service.NewWeightedRRBalancer()
	streamProxy := service.NewStreamProxy(cfg.Proxy.RequestTimeout)

	// 2. 创建 v1 版本的统一根路由组
	v1 := r.Group("/v1")
	{
		// ==========================================
		// 开放路由组 (Public Group)：不需要携带 Token
		// ==========================================
		public := v1.Group("/auth")
		{
			public.POST("/login", userHandler.Login) // 用户登录获取 Token
		}

		// ==========================================
		// 权限受控路由组 (Protected Group)：必须通过 JWT 拦截器
		// ==========================================
		protectedUser := v1.Group("/users")
		protectedUser.Use(middleware.JWTAuth(cfg.JWT.Secret)) // ⚡️ 注入你刚才写的鉴权中间件
		{
			// —— 自助接口（所有登录用户）——
			protectedUser.PUT("/password", userHandler.UpdatePassword) // 修改当前登录用户的密码
			protectedUser.PUT("/profile", userHandler.UpdateProfile)   // 修改当前登录用户的邮箱/手机号

			// —— 管理员接口（在 JWTAuth 基础上追加 RequireAdmin 校验）——
			adminUser := protectedUser.Group("").Use(middleware.RequireAdmin())
			{
				adminUser.POST("", userHandler.CreateUser)                        // 创建新账户
				adminUser.GET("", userHandler.ListUsers)                          // 查询用户列表
				adminUser.PUT("/:id/status", userHandler.UpdateUserStatus)        // 启用/禁用指定用户
				adminUser.POST("/:id/reset-password", userHandler.AdminResetPassword) // 重置指定用户密码
				adminUser.POST("/:id/recharge", userHandler.RechargeBalance)      // 为指定用户充值
			}
		}

		protected := v1.Group("").Use(middleware.JWTAuth(cfg.JWT.Secret))
		{
			// 供应商渠道路由
			protected.POST("/channels", channelHandler.CreateChannel)
			protected.GET("/channels", channelHandler.ListChannels)
			protected.PUT("/channels/:id", channelHandler.UpdateChannel)
			protected.DELETE("/channels/:id", channelHandler.DeleteChannel)
			protected.POST("/channels/:id/test", channelHandler.TestChannel) // 测试单个渠道

			// 渠道批量测试
			protected.POST("/channels/test", channelHandler.TestAllChannels) // 测试所有渠道

			protected.POST("/apikeys", apiKeyHandler.CreateApiKey)        // 分发新 Key
			protected.GET("/apikeys", apiKeyHandler.ListApiKeys)          // 查询 Key 列表
			protected.POST("/apikeys/delete", apiKeyHandler.DeleteApiKey) // 吊销/软删除指定的 Key（使用 POST 以便从 body 读取 key）
		}

		// ==========================================
		// 数据面路由 (Data Plane)：使用 ApiKey 认证
		// ==========================================
		adapterGroup := adapter.NewRegistry()
		adapterGroup.Register(adapter.NewOpenAIAdapter())

		chatHandler := handler.NewChatHandler(
			channelRepo,
			rdb,
			cfg.Cache.ChannelExpireTime,
			balancer, breaker, streamProxy,
			adapterGroup,
		)
		dataPlane := v1.Group("")
		dataPlane.Use(middleware.ApiKeyAuth(apiKeyRepo))
		{
			dataPlane.POST("/chat/completions", chatHandler.HandleDataPlane)
		}
	}
}
