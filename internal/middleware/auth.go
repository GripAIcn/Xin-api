package middleware

import (
	"Xin-api/internal/store/postgresql"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"Xin-api/internal/inbound"
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
// Authorization: Bearer → x-api-key 头（Anthropic SDK / Claude Code 默认鉴权头）
// → x-goog-api-key 头（Gemini 官方 SDK）→ ?key= 查询参数（Gemini 兼容）
func extractDataPlaneKey(c *gin.Context) string {
	// 鉴权 scheme 大小写不敏感，且需容忍复制粘贴带入的首尾空白/换行
	if auth := strings.TrimSpace(c.GetHeader("Authorization")); auth != "" {
		if parts := strings.SplitN(auth, " ", 2); len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			if key := strings.TrimSpace(parts[1]); key != "" {
				return key
			}
		}
	}
	if key := strings.TrimSpace(c.GetHeader("x-api-key")); key != "" {
		return key
	}
	if key := strings.TrimSpace(c.GetHeader("x-goog-api-key")); key != "" {
		return key
	}
	return strings.TrimSpace(c.Query("key"))
}

// dataPlaneAuthError 数据面鉴权失败响应：
// Anthropic Messages 入站返回 Anthropic 错误体，其余入站保持 OpenAI 错误结构
func dataPlaneAuthError(c *gin.Context) {
	if c.Request != nil && c.Request.URL != nil && strings.HasSuffix(c.Request.URL.Path, "/v1/messages") {
		status := response.MapToHTTPStatus(response.Unauthorized)
		c.Data(status, "application/json", response.AnthropicErrorBody(status, response.Msg(response.Unauthorized)))
		c.Abort()
		return
	}
	response.DataPlaneError(c, response.Unauthorized)
}

// ApiKeyAuth 数据面 API Key 认证中间件
// 支持 Authorization: Bearer / x-api-key / x-goog-api-key / ?key= 四种取值方式，校验其有效性。
// 鉴权失败会写应用日志并落一条 error 请求日志，避免 401 请求在管理后台完全不可见。
func ApiKeyAuth(apiKeyRepo postgresql.ApiKeyRepo, logRepo postgresql.LogRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := extractDataPlaneKey(c)
		if key == "" {
			recordAuthFailure(c, logRepo, nil, "", "auth_missing_key", "请求未携带 API Key")
			dataPlaneAuthError(c)
			return
		}

		// 落库只存哈希：对来访明文 Key 求摘要后比对
		keyHash := model.HashApiKey(key)
		apiKey, err := apiKeyRepo.GetByKeyHash(c.Request.Context(), keyHash)
		if err != nil {
			recordAuthFailure(c, logRepo, nil, keyHash, "auth_lookup_error", "API Key 校验失败，请稍后重试")
			dataPlaneAuthError(c)
			return
		}
		if apiKey == nil {
			recordAuthFailure(c, logRepo, nil, keyHash, "auth_invalid_key", "API Key 不存在或已删除")
			dataPlaneAuthError(c)
			return
		}

		// 仅启用且未过期的 Key 可通过鉴权
		if apiKey.Status != model.ApiKeyStatusActive {
			recordAuthFailure(c, logRepo, apiKey, keyHash, "auth_key_disabled", "API Key 已被禁用")
			dataPlaneAuthError(c)
			return
		}
		if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
			recordAuthFailure(c, logRepo, apiKey, keyHash, "auth_key_expired", "API Key 已过期")
			dataPlaneAuthError(c)
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

// recordAuthFailure 记录一次数据面鉴权失败：
//   - 应用日志记录 Key 摘要指纹，便于与 api_keys.key_hash 对账（不落明文）；
//   - 落一条 error 请求日志，管理员/所属用户可在「使用日志」中看到被拒绝的请求。
func recordAuthFailure(c *gin.Context, logRepo postgresql.LogRepo, apiKey *model.ApiKey, keyHash, errType, errMsg string) {
	ctx := c.Request.Context()
	rc := requestctx.Get(c)
	requestID := ""
	startedAt := time.Now()
	if rc != nil {
		requestID = rc.RequestID
		startedAt = rc.StartTime
	}
	if requestID == "" {
		requestID = requestctx.NewRequestID()
	}

	path := ""
	if c.Request != nil && c.Request.URL != nil {
		path = c.Request.URL.Path
	}
	clientIP := c.ClientIP()

	logger.CtxWarnf(ctx, "data-plane auth failed: type=%s path=%s ip=%s key_hash_prefix=%s msg=%s",
		errType, path, clientIP, shortKeyHash(keyHash), errMsg)

	if logRepo == nil {
		return
	}

	now := time.Now()
	statusCode := response.MapToHTTPStatus(response.Unauthorized)
	stage := model.ErrorStageLocal
	message := errMsg
	protocol := dataPlaneProtocol(path)

	log := &model.RequestLog{
		RequestID:    requestID,
		LogType:      model.LogTypeError,
		Status:       model.LogStatusError,
		StatusCode:   &statusCode,
		UsageSource:  model.UsageSourceUpstream,
		Currency:     model.CurrencyUSD,
		ErrorStage:   &stage,
		ErrorType:    &errType,
		ErrorCode:    &errType,
		ErrorMessage: &message,
		StartedAt:    &startedAt,
		CompletedAt:  &now,
	}
	if protocol != "" {
		log.Protocol = &protocol
	}
	if clientIP != "" {
		log.ClientIP = &clientIP
	}
	// 已识别到 Key 时把归属写入日志，普通用户也能在「使用日志」中看到自己的失败请求
	if apiKey != nil {
		userID, apiKeyID := apiKey.UserID, apiKey.ID
		log.UserID = &userID
		log.APIKeyID = &apiKeyID
	}

	if err := logRepo.Create(ctx, log); err != nil {
		logger.CtxErrorf(ctx, "write auth failure log failed: %v", err)
	}
}

// dataPlaneProtocol 按数据面路径推断入站协议，用于日志分类
func dataPlaneProtocol(path string) string {
	switch {
	case strings.HasSuffix(path, "/chat/completions"):
		return inbound.ProtocolChat
	case strings.HasSuffix(path, "/responses"):
		return inbound.ProtocolResponses
	case strings.HasSuffix(path, "/messages"):
		return inbound.ProtocolMessages
	case strings.Contains(path, "/models/"):
		return inbound.ProtocolGemini
	}
	return ""
}

// shortKeyHash 截取哈希前缀用于日志对账，避免暴露明文 Key
func shortKeyHash(keyHash string) string {
	if len(keyHash) > 12 {
		return keyHash[:12]
	}
	return keyHash
}
