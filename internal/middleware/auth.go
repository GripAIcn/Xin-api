package middleware

import (
	"Xin-api/internal/store/postgresql"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"Xin-api/internal/model"
	"Xin-api/pkg/response"
)

const (
	CtxUserIDKey   = "user_id"
	CtxUsernameKey = "username"
	CtxRoleKey     = "role"
)

// JWTAuth 控制面管理后台鉴权中间件
func JWTAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从 Header 中获取 Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.AdminFail(c, response.Unauthorized)
			c.Abort() // ⚡️ 核心：立刻阻断后续 Handler 的执行
			return
		}

		// 2. 按空格分割，验证 Bearer 格式
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			response.AdminFail(c, response.Unauthorized)
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 3. 解析并验证 Token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// 验证加密算法是否匹配 UserHandler 中签发的 HS256
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})

		// Token 解析失败或过期
		if err != nil || !token.Valid {
			response.AdminFail(c, response.Unauthorized)
			c.Abort()
			return
		}

		// 4. 提取 Claims 并注入到 Gin 上下文 (Context) 中
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			response.AdminFail(c, response.Unauthorized)
			c.Abort()
			return
		}

		// 将常用用户信息存入上下文，方便后续 Handler 免密读取
		c.Set(CtxUserIDKey, claims["user_id"])
		c.Set(CtxUsernameKey, claims["username"])
		c.Set(CtxRoleKey, claims["role"])

		c.Next() // 鉴权通过，放行继续执行后续路由
	}
}

// RequireAdmin 管理员权限中间件：仅放行 role=admin 的用户
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString(CtxRoleKey) != model.UserRoleAdmin {
			response.AdminFail(c, response.Forbidden)
			c.Abort()
			return
		}
		c.Next()
	}
}

// ApiKeyAuth 数据面 API Key 认证中间件
// 从 Authorization: Bearer sk-xin-xxx 提取 key，校验其有效性
func ApiKeyAuth(apiKeyRepo postgresql.ApiKeyRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			response.DataPlaneError(c, response.Unauthorized)
			return
		}

		key := strings.TrimPrefix(auth, "Bearer ")
		// 落库只存哈希：对来访明文 Key 求摘要后比对
		apiKey, err := apiKeyRepo.GetByKeyHash(c.Request.Context(), model.HashApiKey(key))
		if err != nil || apiKey == nil {
			response.DataPlaneError(c, response.Unauthorized)
			return
		}

		// 仅启用且未过期的 Key 可通过鉴权
		if apiKey.Status != model.ApiKeyStatusActive {
			response.DataPlaneError(c, response.Unauthorized)
			return
		}
		if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
			response.DataPlaneError(c, response.Unauthorized)
			return
		}

		c.Next()
	}
}
