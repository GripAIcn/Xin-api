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
	logRepo := postgresql.NewLogRepo(db)
	userHandler := handler.NewUserHandler(userRepo, cfg.JWT, logRepo)
	logHandler := handler.NewLogHandler(logRepo)
	apiKeyRepo := postgresql.NewApiKeyRepo(db)
	apiKeyHandler := handler.NewApiKeyHandler(apiKeyRepo)
	channelRepo := postgresql.NewChannelRepo(db)
	modelPriceRepo := postgresql.NewModelPriceRepo(db)
	channelHandler := handler.NewChannelHandler(channelRepo, modelPriceRepo, rdb)
	modelPriceHandler := handler.NewModelPriceHandler(modelPriceRepo)
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
			protectedUser.GET("/profile", userHandler.GetProfile)      // 获取当前登录用户资料（亦用于前端 token 校验）
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

		protected := v1.Group("")
		protected.Use(middleware.JWTAuth(cfg.JWT.Secret))
		{
			// 请求日志查询：所有登录用户可访问，仓储层按当前用户隔离，管理员可查全部
			protected.GET("/logs", logHandler.List)    // 日志列表（分页 + 多维过滤）
			protected.GET("/logs/:id", logHandler.Get) // 日志详情（含计费明细）
			protected.GET("/billing-logs", logHandler.ListBillings) // 账单日志（消费/充值/退款合并，分页 + 多维过滤）

			// API Key：所有登录用户均可管理，但仓储/业务层按当前用户 user_id 隔离
			protected.POST("/apikeys", apiKeyHandler.CreateApiKey)        // 分发新 Key
			protected.GET("/apikeys", apiKeyHandler.ListApiKeys)          // 查询当前用户的 Key 列表
			protected.POST("/apikeys/delete", apiKeyHandler.DeleteApiKey) // 吊销/软删除当前用户名下的 Key

			// 供应商渠道：全局上游资源，仅管理员可管理
			adminChannel := protected.Group("").Use(middleware.RequireAdmin())
			{
				adminChannel.POST("/channels", channelHandler.CreateChannel)
				adminChannel.GET("/channels", channelHandler.ListChannels)
				adminChannel.PUT("/channels/:id", channelHandler.UpdateChannel)
				adminChannel.DELETE("/channels/:id", channelHandler.DeleteChannel)
				adminChannel.POST("/channels/:id/test", channelHandler.TestChannel) // 测试单个渠道

				// 渠道模型管理
				adminChannel.POST("/channels/:id/models", channelHandler.CreateChannelModel)              // 为渠道新增模型
				adminChannel.GET("/channels/:id/models", channelHandler.ListChannelModels)                // 列出渠道下全部模型（含禁用）
				adminChannel.GET("/channels/:id/models/available", channelHandler.FetchUpstreamModels)    // 获取上游可用模型
				adminChannel.PUT("/channels/:id/models/:modelId", channelHandler.UpdateChannelModel)      // 编辑模型
				adminChannel.DELETE("/channels/:id/models/:modelId", channelHandler.DeleteChannelModel)   // 软删除模型
				adminChannel.PUT("/channels/:id/models/:modelId/status", channelHandler.UpdateChannelModelStatus) // 启用/禁用模型

				// 模型管理：按逻辑模型名聚合查询全部渠道模型
				adminChannel.GET("/models", channelHandler.ListModels) // 按逻辑模型名聚合查询（模型管理页）

				// 价格方案管理
				adminChannel.POST("/prices", modelPriceHandler.CreatePrice)              // 新增价格方案
				adminChannel.GET("/prices", modelPriceHandler.ListPrices)                // 价格方案列表（含绑定模型数）
				adminChannel.GET("/prices/:id", modelPriceHandler.GetPrice)              // 价格方案详情
				adminChannel.PUT("/prices/:id", modelPriceHandler.UpdatePrice)           // 编辑价格方案
				adminChannel.PUT("/prices/:id/status", modelPriceHandler.UpdatePriceStatus) // 启用/禁用价格方案
				adminChannel.DELETE("/prices/:id", modelPriceHandler.DeletePrice)        // 删除价格方案
				adminChannel.GET("/prices/:id/models", modelPriceHandler.ListPriceModels) // 查看绑定该方案的模型

				// 渠道批量测试
				adminChannel.POST("/channels/test", channelHandler.TestAllChannels) // 测试所有渠道
			}
		}

		// ==========================================
		// 数据面路由 (Data Plane)：使用 ApiKey 认证
		// ==========================================
		adapterGroup := adapter.NewRegistry()
		adapterGroup.Register(adapter.NewOpenAIAdapter())
		adapterGroup.RegisterAlias("deepseek", adapter.NewOpenAIAdapter())
		adapterGroup.Register(adapter.NewAnthropicAdapter())
		adapterGroup.Register(adapter.NewGeminiAdapter())

		chatHandler := handler.NewChatHandler(
			channelRepo,
			rdb,
			cfg.Cache.ChannelExpireTime,
			balancer, breaker, streamProxy,
			adapterGroup,
			logRepo,
			userRepo,
		)
		dataPlane := v1.Group("")
		dataPlane.Use(middleware.ApiKeyAuth(apiKeyRepo))
		{
			dataPlane.POST("/chat/completions", chatHandler.HandleDataPlane)
		}
	}
}
