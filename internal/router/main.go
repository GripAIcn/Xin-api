package router

import (
	"Xin-api/config"
	"Xin-api/internal/handler"
	"Xin-api/internal/middleware"
	"Xin-api/internal/store/postgresql"

	"github.com/gin-gonic/gin"
)

// SetupRouter 统一注册控制面管理后台路由
func SetupRouter(r *gin.Engine, repo postgresql.UserRepo, jwtCfg config.JWTConfig) {
	// 1. 初始化 Handler（注入底层的 DB 仓库和配置）
	userHandler := handler.NewUserHandler(repo, jwtCfg)

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
		protected := v1.Group("/users")
		protected.Use(middleware.JWTAuth(jwtCfg.Secret)) // ⚡️ 注入你刚才写的鉴权中间件
		{
			protected.POST("", userHandler.CreateUser)             // 添加账户名和密码
			protected.PUT("/password", userHandler.UpdatePassword) // 修改当前登录用户的密码
			protected.PUT("/account", userHandler.UpdateAccount)   // 修改当前登录用户的账号名
		}
	}
}
