package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"Xin-api/internal/form"
	"Xin-api/internal/middleware"
	"Xin-api/internal/model"
	"Xin-api/internal/store/postgresql"
	"Xin-api/pkg/response"
)

// DashboardHandler 仪表盘统计处理器
type DashboardHandler struct {
	repo     postgresql.DashboardRepo
	validate *validator.Validate
}

// NewDashboardHandler 构造函数注入依赖
func NewDashboardHandler(repo postgresql.DashboardRepo) *DashboardHandler {
	return &DashboardHandler{repo: repo, validate: validator.New()}
}

// Stats 查询仪表盘统计（概览 / 趋势 / 排行）。
// 权限约定：管理员统计全部；普通用户强制限定为仅统计本人数据。
func (h *DashboardHandler) Stats(c *gin.Context) {
	var q form.DashboardQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}
	if err := h.validate.Struct(q); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	ctx := c.Request.Context()

	var scopeUserID *int64
	if c.GetString(middleware.CtxRoleKey) != model.UserRoleAdmin {
		uid, ok := currentUserID(c)
		if !ok {
			response.AdminFail(c, response.Unauthorized)
			return
		}
		scopeUserID = &uid
	}

	data, err := h.repo.Stats(ctx, q, scopeUserID)
	if err != nil {
		response.AdminFail(c, response.InternalError)
		return
	}
	response.AdminSuccess(c, data)
}
