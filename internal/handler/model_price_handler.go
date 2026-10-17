package handler

import (
	"strconv"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"Xin-api/internal/form"
	"Xin-api/internal/model"
	"Xin-api/internal/store/postgresql"
	"Xin-api/pkg/response"
)

// ModelPriceHandler 价格方案管理处理器（仅管理员，权限由路由中间件保证）
type ModelPriceHandler struct {
	priceRepo postgresql.ModelPriceRepo
	validate  *validator.Validate
}

// NewModelPriceHandler 创建价格方案处理器
func NewModelPriceHandler(repo postgresql.ModelPriceRepo) *ModelPriceHandler {
	return &ModelPriceHandler{
		priceRepo: repo,
		validate:  validator.New(),
	}
}

// CreatePrice 新增价格方案
func (h *ModelPriceHandler) CreatePrice(c *gin.Context) {
	var req form.CreateModelPriceReq
	if !h.bindAndValidate(c, &req) {
		return
	}
	if !pricesNonNegative(req.InputPrice, req.OutputPrice, req.CachedInputPrice, req.CacheCreationPrice) {
		response.AdminFail(c, response.InvalidParams)
		return
	}
	ctx := c.Request.Context()

	// 方案名判重
	existing, err := h.priceRepo.GetByName(ctx, req.Name)
	if err != nil {
		logger.Error(ctx, "check duplicate model price failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	if existing != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	p := &model.ModelPrice{
		Name:               req.Name,
		InputPrice:         req.InputPrice,
		OutputPrice:        req.OutputPrice,
		CachedInputPrice:   req.CachedInputPrice,
		CacheCreationPrice: req.CacheCreationPrice,
		Currency:           currency,
		Status:             model.ModelPriceStatusActive,
	}

	if err := h.priceRepo.Create(ctx, p); err != nil {
		logger.Error(ctx, "create model price failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	response.AdminSuccess(c, p)
}

// ListPrices 获取全部价格方案（含绑定模型数）
func (h *ModelPriceHandler) ListPrices(c *gin.Context) {
	prices, err := h.priceRepo.List(c.Request.Context())
	if err != nil {
		logger.Error(c.Request.Context(), "list model prices failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	response.AdminSuccess(c, prices)
}

// GetPrice 获取价格方案详情
func (h *ModelPriceHandler) GetPrice(c *gin.Context) {
	id, ok := h.parsePathInt64(c, "id")
	if !ok {
		return
	}
	ctx := c.Request.Context()

	p, err := h.priceRepo.GetByID(ctx, id)
	if err != nil {
		logger.Error(ctx, "get model price failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	if p == nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	response.AdminSuccess(c, p)
}

// UpdatePrice 整体更新价格方案
func (h *ModelPriceHandler) UpdatePrice(c *gin.Context) {
	id, ok := h.parsePathInt64(c, "id")
	if !ok {
		return
	}

	var req form.UpdateModelPriceReq
	if !h.bindAndValidate(c, &req) {
		return
	}
	if !pricesNonNegative(req.InputPrice, req.OutputPrice, req.CachedInputPrice, req.CacheCreationPrice) {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	ctx := c.Request.Context()
	p, err := h.priceRepo.GetByID(ctx, id)
	if err != nil {
		logger.Error(ctx, "get model price failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	if p == nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	// 改名时需保证方案名全局唯一
	if req.Name != p.Name {
		dup, err := h.priceRepo.GetByName(ctx, req.Name)
		if err != nil {
			logger.Error(ctx, "check duplicate model price failed: %v", err)
			response.AdminFail(c, response.InternalError)
			return
		}
		if dup != nil && dup.ID != id {
			response.AdminFail(c, response.InvalidParams)
			return
		}
	}

	p.Name = req.Name
	p.InputPrice = req.InputPrice
	p.OutputPrice = req.OutputPrice
	p.CachedInputPrice = req.CachedInputPrice
	p.CacheCreationPrice = req.CacheCreationPrice
	if req.Currency != "" {
		p.Currency = req.Currency
	}
	if req.Status != "" {
		p.Status = req.Status
	}

	if err := h.priceRepo.Update(ctx, p); err != nil {
		logger.Error(ctx, "update model price failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	response.AdminSuccess(c, p)
}

// UpdatePriceStatus 启用/禁用价格方案
func (h *ModelPriceHandler) UpdatePriceStatus(c *gin.Context) {
	id, ok := h.parsePathInt64(c, "id")
	if !ok {
		return
	}

	var req form.UpdateModelPriceStatusReq
	if !h.bindAndValidate(c, &req) {
		return
	}

	ctx := c.Request.Context()
	p, err := h.priceRepo.GetByID(ctx, id)
	if err != nil {
		logger.Error(ctx, "get model price failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	if p == nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	if err := h.priceRepo.UpdateStatus(ctx, id, req.Status); err != nil {
		logger.Error(ctx, "update model price status failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	p.Status = req.Status
	response.AdminSuccess(c, p)
}

// DeletePrice 删除价格方案
func (h *ModelPriceHandler) DeletePrice(c *gin.Context) {
	id, ok := h.parsePathInt64(c, "id")
	if !ok {
		return
	}

	ctx := c.Request.Context()
	p, err := h.priceRepo.GetByID(ctx, id)
	if err != nil {
		logger.Error(ctx, "get model price failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	if p == nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	if err := h.priceRepo.Delete(ctx, id); err != nil {
		logger.Error(ctx, "delete model price failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	response.AdminSuccess(c, "价格方案删除成功")
}

// ListPriceModels 查询绑定到指定价格方案的「渠道 × 模型」明细
func (h *ModelPriceHandler) ListPriceModels(c *gin.Context) {
	id, ok := h.parsePathInt64(c, "id")
	if !ok {
		return
	}

	ctx := c.Request.Context()
	p, err := h.priceRepo.GetByID(ctx, id)
	if err != nil {
		logger.Error(ctx, "get model price failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	if p == nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	entries, err := h.priceRepo.ListBoundModels(ctx, id)
	if err != nil {
		logger.Error(ctx, "list price bound models failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	if entries == nil {
		entries = []model.PriceBoundModelEntry{}
	}

	response.AdminSuccess(c, entries)
}

// parsePathInt64 解析路径中的整型参数
func (h *ModelPriceHandler) parsePathInt64(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil {
		response.AdminFail(c, response.InvalidParams)
		return 0, false
	}
	return id, true
}

// pricesNonNegative 校验价格均非负（Money 为结构体，validator 的 gte=0 不适用）
func pricesNonNegative(prices ...model.Money) bool {
	for _, p := range prices {
		if p.IsNegative() {
			return false
		}
	}
	return true
}

// bindAndValidate 参数校验辅助方法
func (h *ModelPriceHandler) bindAndValidate(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		logger.Error(c.Request.Context(), "bind request failed: %v", err)
		response.AdminFail(c, response.InvalidParams)
		return false
	}
	if err := h.validate.Struct(req); err != nil {
		logger.Warn(c.Request.Context(), "validate fields failed: %v", err)
		response.AdminFail(c, response.InvalidParams)
		return false
	}
	return true
}
