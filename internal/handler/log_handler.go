package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"Xin-api/internal/form"
	"Xin-api/internal/middleware"
	"Xin-api/internal/model"
	"Xin-api/internal/store/postgresql"
	"Xin-api/pkg/response"
)

// LogHandler 请求日志查询处理器
type LogHandler struct {
	logRepo  postgresql.LogRepo
	validate *validator.Validate
}

// NewLogHandler 构造函数注入依赖
func NewLogHandler(logRepo postgresql.LogRepo) *LogHandler {
	return &LogHandler{
		logRepo:  logRepo,
		validate: validator.New(),
	}
}

// List 分页查询请求日志列表。
// 权限约定：管理员可查看全部日志；普通用户被强制限定为仅能查看本人日志。
func (h *LogHandler) List(c *gin.Context) {
	// 1. 参数绑定与校验（GET 查询串）
	var q form.LogListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}
	if err := h.validate.Struct(q); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	ctx := c.Request.Context()

	// 2. 管理员：可查询全部
	if c.GetString(middleware.CtxRoleKey) == model.UserRoleAdmin {
		data, err := h.logRepo.List(ctx, q, nil)
		if err != nil {
			response.AdminFail(c, response.InternalError)
			return
		}
		response.AdminSuccess(c, data)
		return
	}

	// 3. 普通用户：强制以当前登录用户为查询范围，防止越权
	uid, ok := currentUserID(c)
	if !ok {
		response.AdminFail(c, response.Unauthorized)
		return
	}
	data, err := h.logRepo.List(ctx, q, &uid)
	if err != nil {
		response.AdminFail(c, response.InternalError)
		return
	}
	// 普通用户不可见上游渠道信息
	for i := range data.List {
		sanitizeLogItem(&data.List[i])
	}
	response.AdminSuccess(c, data)
}

// sanitizeLogItem 清空列表项中的上游渠道信息（仅用于非管理员视图）
func sanitizeLogItem(item *model.LogListItem) {
	item.ChannelID = nil
	item.ChannelName = nil
	item.UpstreamReqID = nil
}

// ListBillings 分页查询账单流水（消费 / 充值 / 退款合并）。
// 权限约定：管理员可查看全部流水；普通用户被强制限定为仅能查看本人流水。
func (h *LogHandler) ListBillings(c *gin.Context) {
	// 1. 参数绑定与校验（GET 查询串）
	var q form.BillingLogQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}
	if err := h.validate.Struct(q); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	ctx := c.Request.Context()

	// 2. 管理员：可查询全部
	if c.GetString(middleware.CtxRoleKey) == model.UserRoleAdmin {
		data, err := h.logRepo.ListBilling(ctx, q, nil)
		if err != nil {
			response.AdminFail(c, response.InternalError)
			return
		}
		response.AdminSuccess(c, data)
		return
	}

	// 3. 普通用户：强制以当前登录用户为查询范围，防止越权
	uid, ok := currentUserID(c)
	if !ok {
		response.AdminFail(c, response.Unauthorized)
		return
	}
	data, err := h.logRepo.ListBilling(ctx, q, &uid)
	if err != nil {
		response.AdminFail(c, response.InternalError)
		return
	}
	response.AdminSuccess(c, data)
}

// Get 查询单条日志详情。
// 权限约定：管理员可查看任意日志；普通用户仅可查看归属于本人的日志。
func (h *LogHandler) Get(c *gin.Context) {
	// 1. 解析路径参数 id
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	ctx := c.Request.Context()

	// 2. 查询详情（未找到返回 nil）
	detail, err := h.logRepo.GetByID(ctx, id)
	if err != nil {
		response.AdminFail(c, response.InternalError)
		return
	}
	if detail == nil {
		response.AdminFail(c, response.NotFound)
		return
	}

	// 3. 非管理员：仅允许查看属于本人的日志
	if c.GetString(middleware.CtxRoleKey) != model.UserRoleAdmin {
		uid, ok := currentUserID(c)
		if !ok {
			response.AdminFail(c, response.Unauthorized)
			return
		}
		// 无归属日志（如用户不存在的登录失败）一律不向普通用户开放；
		// 统一返回 NotFound，避免通过 403/404 差异探测日志是否存在
		if detail.UserID == nil || *detail.UserID != uid {
			response.AdminFail(c, response.NotFound)
			return
		}
		// 普通用户不可见上游渠道与路由细节
		sanitizeLogItem(&detail.LogListItem)
		detail.UpstreamModel = nil
		detail.Provider = nil
		detail.GatewayNode = nil
		detail.RetryChannels = []int64{}
		detail.RouteStrategy = nil
		detail.RouteWeight = nil
		if detail.Billing != nil {
			detail.Billing.ChannelID = nil
		}
	}

	response.AdminSuccess(c, detail)
}
