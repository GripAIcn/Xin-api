package middleware

import (
	"Xin-api/internal/store/postgresql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"Xin-api/internal/model"
	"Xin-api/internal/requestctx"
	"Xin-api/pkg/response"
)

const (
	CtxUserIDKey   = "user_id"
	CtxUsernameKey = "username"
	CtxRoleKey     = "role"
	CtxAPIKeyIDKey = "api_key_id"
	// CtxMustChangePasswordKey 标记当前登录用户是否必须先修改密码
	CtxMustChangePasswordKey = "must_change_password"
)

// JWTAuth 控制面管理后台鉴权中间件。
// 除校验签名外，还会回查用户当前状态与角色，确保禁用用户和角色变更即时生效。
func JWTAuth(jwtSecret string, userRepo postgresql.UserRepo) gin.HandlerFunc {
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

		uid, ok := claimUserID(claims["user_id"])
		if !ok {
			response.AdminFail(c, response.Unauthorized)
			c.Abort()
			return
		}

		// 5. 回查数据库：Token 签发后用户可能被禁用或角色被调整
		user, err := userRepo.GetByID(c.Request.Context(), uid)
		if err != nil {
			response.AdminFail(c, response.InternalError)
			c.Abort()
			return
		}
		if user == nil || user.Status != model.UserStatusActive {
			response.AdminFail(c, response.Unauthorized)
			c.Abort()
			return
		}

		// 将常用用户信息存入上下文，方便后续 Handler 免密读取
		// 角色以数据库为准，避免旧 Token 在角色变更后仍保留管理员权限
		c.Set(CtxUserIDKey, claims["user_id"])
		c.Set(CtxUsernameKey, user.Username)
		c.Set(CtxRoleKey, user.Role)
		c.Set(CtxMustChangePasswordKey, user.MustChangePassword)

		c.Next() // 鉴权通过，放行继续执行后续路由
	}
}

// claimUserID 将 JWT 中的 user_id（JSON 数字解析后为 float64）转换为 uint
func claimUserID(v any) (uint, bool) {
	switch id := v.(type) {
	case float64:
		if id <= 0 || id != float64(uint(id)) {
			return 0, false
		}
		return uint(id), true
	case int64:
		if id <= 0 {
			return 0, false
		}
		return uint(id), true
	case uint:
		if id == 0 {
			return 0, false
		}
		return id, true
	default:
		return 0, false
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

// RequirePasswordChanged 强制改密中间件：用户被标记 must_change_password 时，
// 仅放行查询个人资料与修改密码，其余控制面接口一律拒绝。
func RequirePasswordChanged() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !c.GetBool(CtxMustChangePasswordKey) {
			c.Next()
			return
		}

		// 白名单：读取个人资料（前端 token 校验）、修改密码
		if (c.Request.Method == http.MethodGet && c.FullPath() == "/v1/users/profile") ||
			(c.Request.Method == http.MethodPut && c.FullPath() == "/v1/users/password") {
			c.Next()
			return
		}

		response.AdminFail(c, response.PasswordChangeRequired)
		c.Abort()
	}
}

// extractDataPlaneKey 按优先级提取数据面 API Key：
// Authorization: Bearer → x-goog-api-key 头（Gemini 官方 SDK）→ ?key= 查询参数（Gemini 兼容）
func extractDataPlaneKey(c *gin.Context) string {
	if auth := c.GetHeader("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	if key := c.GetHeader("x-goog-api-key"); key != "" {
		return key
	}
	return c.Query("key")
}

// ApiKeyAuth 数据面 API Key 认证中间件
// 支持 Authorization: Bearer / x-goog-api-key / ?key= 三种取值方式，校验其有效性
func ApiKeyAuth(apiKeyRepo postgresql.ApiKeyRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := extractDataPlaneKey(c)
		if key == "" {
			response.DataPlaneError(c, response.Unauthorized)
			return
		}

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

		c.Set(CtxUserIDKey, apiKey.UserID)
		c.Set(CtxAPIKeyIDKey, apiKey.ID)

		// 同步写入请求上下文，后续链路直接从 RequestContext 读取身份信息
		if rc := requestctx.Get(c); rc != nil {
			rc.UserID = apiKey.UserID
			rc.APIKeyID = apiKey.ID
		}

		c.Next()
	}
}
