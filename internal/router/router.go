package router

import (
	"github.com/gin-gonic/gin"
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
	breaker *service.CircuitBreaker) {
	// 1. 初始化 Handler（注入底层的 DB 仓库和配置）
	userRepo := postgresql.NewUserRepo(db)
	userHandler := handler.NewUserHandler(userRepo, cfg.JWT)
	apiKeyRepo := postgresql.NewApiKeyRepo(db)
	apiKeyHandler := handler.NewApiKeyHandler(apiKeyRepo)
	groupRepo := postgresql.NewGroupRepo(db)
	groupHandler := handler.NewGroupHandler(groupRepo)
	channelRepo := postgresql.NewChannelRepo(db)
	channelHandler := handler.NewChannelHandler(channelRepo)
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
		protectedUser := v1.Group("/users").Use(middleware.JWTAuth(cfg.JWT.Secret)) // ⚡️ 注入你刚才写的鉴权中间件
		{
			protectedUser.POST("", userHandler.CreateUser)             // 添加账户名和密码
			protectedUser.PUT("/password", userHandler.UpdatePassword) // 修改当前登录用户的密码
			protectedUser.PUT("/account", userHandler.UpdateAccount)   // 修改当前登录用户的账号名
		}

		protected := v1.Group("").Use(middleware.JWTAuth(cfg.JWT.Secret))
		{
			// 项目组路由
			protected.POST("/groups", groupHandler.CreateGroup)
			protected.GET("/groups", groupHandler.ListGroups)
			protected.PUT("/groups/:id", groupHandler.UpdateGroup)
			protected.PUT("/groups/:id/status", groupHandler.UpdateStatus)
			protected.DELETE("/groups/:id", groupHandler.DeleteGroup)

			// 供应商渠道路由
			protected.POST("/channels", channelHandler.CreateChannel)
			protected.GET("/channels", channelHandler.ListChannelsByGroup)
			protected.PUT("/channels/:id", channelHandler.UpdateChannel)
			protected.DELETE("/channels/:id", channelHandler.DeleteChannel)

			protected.POST("/apikeys", apiKeyHandler.CreateApiKey)        // 为组分发新 Key
			protected.GET("/apikeys", apiKeyHandler.ListApiKeys)          // 查询组名下的 Key 列表
			protected.POST("/apikeys/delete", apiKeyHandler.DeleteApiKey) // 吊销/软删除指定的 Key（使用 POST 以便从 body 读取 key）
		}

		// ==========================================
		// 数据面路由 (Data Plane)：使用 ApiKey 认证
		// ==========================================
		adapterGroup := adapter.NewRegistry()
		adapterGroup.Register(adapter.NewOpenAIAdapter())

		chatHandler := handler.NewChatHandler(
			postgresql.NewChannelRepo(db),
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
