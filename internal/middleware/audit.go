package middleware

import (
	"net/http"
	"strings"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/gin-gonic/gin"

	"Xin-api/internal/model"
	"Xin-api/internal/store/postgresql"
)

// auditActions 控制面变更接口到审计动作名的映射
var auditActions = map[string]string{
	http.MethodPost + " /v1/users":                              "create_user",
	http.MethodPut + " /v1/users/:id/status":                    "update_user_status",
	http.MethodPost + " /v1/users/:id/reset-password":           "reset_password",
	http.MethodPost + " /v1/users/:id/recharge":                 "recharge_balance",
	http.MethodPut + " /v1/users/password":                      "update_password",
	http.MethodPut + " /v1/users/profile":                       "update_profile",
	http.MethodPost + " /v1/apikeys":                            "create_key",
	http.MethodPost + " /v1/apikeys/delete":                     "delete_key",
	http.MethodPost + " /v1/channels":                           "create_channel",
	http.MethodPut + " /v1/channels/:id":                        "update_channel",
	http.MethodDelete + " /v1/channels/:id":                     "delete_channel",
	http.MethodPost + " /v1/channels/:id/test":                  "test_channel",
	http.MethodPost + " /v1/channels/:id/models":                "create_model",
	http.MethodPut + " /v1/channels/:id/models/:modelId":        "update_model",
	http.MethodDelete + " /v1/channels/:id/models/:modelId":     "delete_model",
	http.MethodPut + " /v1/channels/:id/models/:modelId/status": "update_model_status",
	http.MethodPost + " /v1/channels/test":                      "test_all_channels",
	http.MethodPost + " /v1/prices":                             "create_price",
	http.MethodPut + " /v1/prices/:id":                          "update_price",
	http.MethodPut + " /v1/prices/:id/status":                   "update_price_status",
	http.MethodDelete + " /v1/prices/:id":                       "delete_price",
	http.MethodPost + " /v1/channel-metrics/aggregate":          "aggregate_channel_metrics",
}

// auditTargetTypes 路由路径段到目标对象类型的映射
var auditTargetTypes = map[string]string{
	"users":           "user",
	"apikeys":         "api_key",
	"channels":        "channel",
	"prices":          "model_price",
	"channel-metrics": "channel_metric",
}

// AuditLog 控制面变更自动审计：请求完成后对非只读请求写入 audit_logs
func AuditLog(repo postgresql.LogRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if repo == nil {
			return
		}
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			return
		}

		path := c.FullPath()
		if path == "" {
			return
		}

		action := auditActions[c.Request.Method+" "+path]
		if action == "" {
			action = strings.ToLower(c.Request.Method) + ":" + path
		}

		audit := &model.AuditLog{
			Action: action,
			Result: model.AuditResultSuccess,
		}
		if c.Writer.Status() >= http.StatusBadRequest {
			audit.Result = model.AuditResultFailure
		}
		if userID, ok := contextUserID(c); ok {
			audit.UserID = &userID
		}
		if username := c.GetString(CtxUsernameKey); username != "" {
			audit.UserName = &username
		}
		if targetType := auditTargetType(path); targetType != "" {
			audit.TargetType = &targetType
		}
		if targetID := c.Param("id"); targetID != "" {
			audit.TargetID = &targetID
		} else if targetID := c.Param("modelId"); targetID != "" {
			audit.TargetID = &targetID
		}
		if ip := c.ClientIP(); ip != "" {
			audit.ClientIP = &ip
		}
		if ua := c.Request.UserAgent(); ua != "" {
			audit.UserAgent = &ua
		}

		if err := repo.CreateAudit(c.Request.Context(), audit); err != nil {
			logger.Error(c.Request.Context(), "write audit log failed: %v", err)
		}
	}
}

// contextUserID 读取 JWT 注入的 user_id（JSON 数字解析为 float64）
func contextUserID(c *gin.Context) (int64, bool) {
	v, ok := c.Get(CtxUserIDKey)
	if !ok {
		return 0, false
	}
	switch id := v.(type) {
	case float64:
		return int64(id), true
	case int64:
		return id, true
	case int:
		return int64(id), true
	default:
		return 0, false
	}
}

// auditTargetType 从路由路径中提取目标对象类型
func auditTargetType(path string) string {
	for _, seg := range strings.Split(strings.Trim(path, "/"), "/") {
		if targetType, ok := auditTargetTypes[seg]; ok {
			return targetType
		}
	}
	return ""
}
