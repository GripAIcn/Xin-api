package handler

import (
	"context"
	"errors"
	"strconv"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"

	"Xin-api/internal/form"
	"Xin-api/internal/model"
	"Xin-api/internal/service"
	"Xin-api/internal/store/postgresql"
	"Xin-api/pkg/response"
)

type ChannelHandler struct {
	channelRepo         postgresql.ChannelRepo
	priceRepo           postgresql.ModelPriceRepo
	channelTester       *service.ChannelTester
	channelModelFetcher *service.ChannelModelFetcher
	redis               *redis.Client
	validate            *validator.Validate
}

func NewChannelHandler(repo postgresql.ChannelRepo, priceRepo postgresql.ModelPriceRepo, rdb *redis.Client) *ChannelHandler {
	return &ChannelHandler{
		channelRepo:         repo,
		priceRepo:           priceRepo,
		channelTester:       service.NewChannelTester(repo),
		channelModelFetcher: service.NewChannelModelFetcher(repo),
		redis:               rdb,
		validate:            validator.New(),
	}
}

// CreateChannel 添加上游渠道供应商
func (h *ChannelHandler) CreateChannel(c *gin.Context) {
	var req form.CreateChannelReq // 包含 Name, Provider, Protocol, BaseURL, APIKey, Weight 等
	if !h.bindAndValidate(c, &req) {
		return
	}

	channel := &model.Channel{
		Name:       req.Name,
		Provider:   req.Provider,
		Protocol:   model.StringArray(req.Protocol),
		BaseURL:    req.BaseURL,
		APIKey:     req.APIKey,
		Weight:     req.Weight,
		Priority:   req.Priority,
		Tags:       model.StringArray(req.Tags),
		Timeout:    req.Timeout,
		MaxRetries: req.MaxRetries,
		Status:     model.ChannelStatusActive, // 默认正常
	}

	if err := h.channelRepo.Create(c.Request.Context(), channel); err != nil {
		logger.Error(c.Request.Context(), "create channel failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	// 清除缓存
	h.invalidateChannelCache(c.Request.Context())

	response.AdminSuccess(c, channel)
}

// DeleteChannel 软删除渠道
func (h *ChannelHandler) DeleteChannel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	if err := h.channelRepo.DeleteSoft(c.Request.Context(), id); err != nil {
		logger.Error(c.Request.Context(), "delete channel failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	// 清除缓存
	h.invalidateChannelCache(c.Request.Context())

	response.AdminSuccessMsg(c, "渠道删除成功")
}

// UpdateChannel 覆盖更新渠道（用于管理后台编辑保存动作）
func (h *ChannelHandler) UpdateChannel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	var req form.UpdateChannelReq
	if !h.bindAndValidate(c, &req) {
		return
	}

	// 先获取老数据进行校验，或直接覆盖
	channel := &model.Channel{
		ID:         id,
		Name:       req.Name,
		Provider:   req.Provider,
		Protocol:   model.StringArray(req.Protocol),
		BaseURL:    req.BaseURL,
		APIKey:     req.APIKey,
		Weight:     req.Weight,
		Priority:   req.Priority,
		Tags:       model.StringArray(req.Tags),
		Timeout:    req.Timeout,
		MaxRetries: req.MaxRetries,
		Status:     req.Status, // 允许编辑时调整状态
	}

	if err := h.channelRepo.Update(c.Request.Context(), channel); err != nil {
		logger.Error(c.Request.Context(), "update channel failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	// 清除缓存
	h.invalidateChannelCache(c.Request.Context())

	response.AdminSuccess(c, channel)
}

// ListChannels 获取全量渠道
func (h *ChannelHandler) ListChannels(c *gin.Context) {
	channels, err := h.channelRepo.ListAll(c.Request.Context())
	if err != nil {
		logger.Error(c.Request.Context(), "list channels failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	response.AdminSuccess(c, channels)
}

// ListModels 按逻辑模型名聚合查询全部渠道模型（模型管理页）
func (h *ChannelHandler) ListModels(c *gin.Context) {
	rows, err := h.channelRepo.ListModelChannelRows(c.Request.Context())
	if err != nil {
		logger.Error(c.Request.Context(), "list models failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	// 在 Go 内按 Name 分组（保持 SQL 的稳定顺序）
	groups := make([]model.ModelGroup, 0)
	index := make(map[string]int)
	for _, row := range rows {
		i, ok := index[row.Name]
		if !ok {
			groups = append(groups, model.ModelGroup{Name: row.Name})
			i = len(groups) - 1
			index[row.Name] = i
		}
		groups[i].Channels = append(groups[i].Channels, row)
	}
	for i := range groups {
		groups[i].ChannelCount = len(groups[i].Channels)
		active := 0
		for _, ch := range groups[i].Channels {
			if ch.Status == model.ModelStatusActive {
				active++
			}
		}
		groups[i].ActiveCount = active
	}

	response.AdminSuccess(c, groups)
}

// invalidateChannelCache 清除全部渠道缓存
func (h *ChannelHandler) invalidateChannelCache(ctx context.Context) {
	pattern := "xin:channels:active:*"
	iter := h.redis.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		h.redis.Del(ctx, iter.Val())
	}
}

// TestChannel 测试单个渠道
func (h *ChannelHandler) TestChannel(c *gin.Context) {
	idStr := c.Param("id")
	channelID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	var req form.TestChannelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		// 允许空 body，此时测试所有模型
		req.Model = ""
	}

	result, err := h.channelTester.TestChannel(c.Request.Context(), channelID, req.Model)
	if err != nil {
		logger.Error(c.Request.Context(), "test channel failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	response.AdminSuccess(c, result)
}

// TestAllChannels 测试所有渠道
func (h *ChannelHandler) TestAllChannels(c *gin.Context) {
	results, err := h.channelTester.TestAllChannels(c.Request.Context())
	if err != nil {
		logger.Error(c.Request.Context(), "test all channels failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	response.AdminSuccess(c, results)
}

// ==========================================
// 渠道模型管理（管理员）
// ==========================================

// CreateChannelModel 为渠道新增模型
func (h *ChannelHandler) CreateChannelModel(c *gin.Context) {
	channelID, ok := h.parsePathInt64(c, "id")
	if !ok {
		return
	}

	var req form.CreateChannelModelReq
	if !h.bindAndValidate(c, &req) {
		return
	}

	ctx := c.Request.Context()
	channel, err := h.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		logger.Error(ctx, "get channel failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	if channel == nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	// 同渠道对外名判重（应用层保证一致性）
	existing, err := h.channelRepo.GetModelByName(ctx, channelID, req.Name)
	if err != nil {
		logger.Error(ctx, "check duplicate model failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	if existing != nil {
		response.AdminFail(c, response.Conflict)
		return
	}

	status := req.Status
	if status == "" {
		status = model.ModelStatusActive
	}

	// 若提交了有效价格方案 id，先校验其存在
	if req.ModelPriceID != nil && *req.ModelPriceID > 0 {
		exists, err := h.priceRepo.Exists(ctx, *req.ModelPriceID)
		if err != nil {
			logger.Error(ctx, "check model price exists failed: %v", err)
			response.AdminFail(c, response.InternalError)
			return
		}
		if !exists {
			response.AdminFail(c, response.InvalidParams)
			return
		}
	}

	m := &model.Model{
		ChannelID:       channelID,
		Name:            req.Name,
		UpstreamModel:   req.UpstreamModel,
		ContextLength:   req.ContextLength,
		MaxOutputTokens: req.MaxOutputTokens,
		Capabilities:    model.StringArray(req.Capabilities),
		Status:          status,
	}

	if err := h.channelRepo.CreateModelWithBinding(ctx, m, req.ModelPriceID); err != nil {
		logger.Error(ctx, "create channel model failed: %v", err)
		response.AdminFail(c, response.InvalidParams)
		return
	}

	h.invalidateModelCache(ctx)
	response.AdminSuccess(c, m)
}

// ListChannelModels 列出渠道下全部模型（含 disabled）
func (h *ChannelHandler) ListChannelModels(c *gin.Context) {
	channelID, ok := h.parsePathInt64(c, "id")
	if !ok {
		return
	}

	ctx := c.Request.Context()
	channel, err := h.channelRepo.GetByID(ctx, channelID)
	if err != nil {
		logger.Error(ctx, "get channel failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	if channel == nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	models, err := h.channelRepo.ListModelsByChannelAll(ctx, channelID)
	if err != nil {
		logger.Error(ctx, "list channel models failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	response.AdminSuccess(c, models)
}

// UpdateChannelModel 编辑渠道模型
func (h *ChannelHandler) UpdateChannelModel(c *gin.Context) {
	channelID, ok := h.parsePathInt64(c, "id")
	if !ok {
		return
	}
	modelID, ok := h.parsePathInt64(c, "modelId")
	if !ok {
		return
	}

	var req form.UpdateChannelModelReq
	if !h.bindAndValidate(c, &req) {
		return
	}

	ctx := c.Request.Context()
	m, err := h.channelRepo.GetModelByID(ctx, modelID)
	if err != nil {
		logger.Error(ctx, "get channel model failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	if m == nil || m.ChannelID != channelID {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	// 改名时需保证同渠道对外名唯一
	if req.Name != m.Name {
		dup, err := h.channelRepo.GetModelByName(ctx, channelID, req.Name)
		if err != nil {
			logger.Error(ctx, "check duplicate model failed: %v", err)
			response.AdminFail(c, response.InternalError)
			return
		}
		if dup != nil {
			response.AdminFail(c, response.Conflict)
			return
		}
	}

	// 若提交了有效价格方案 id，先校验其存在
	if req.ModelPriceID != nil && *req.ModelPriceID > 0 {
		exists, err := h.priceRepo.Exists(ctx, *req.ModelPriceID)
		if err != nil {
			logger.Error(ctx, "check model price exists failed: %v", err)
			response.AdminFail(c, response.InternalError)
			return
		}
		if !exists {
			response.AdminFail(c, response.InvalidParams)
			return
		}
	}

	m.Name = req.Name
	m.UpstreamModel = req.UpstreamModel
	m.ContextLength = req.ContextLength
	m.MaxOutputTokens = req.MaxOutputTokens
	m.Capabilities = model.StringArray(req.Capabilities)
	if req.Status != "" {
		m.Status = req.Status
	}

	if err := h.channelRepo.UpdateModelWithBinding(ctx, m, req.ModelPriceID); err != nil {
		logger.Error(ctx, "update channel model failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	h.invalidateModelCache(ctx)
	response.AdminSuccess(c, m)
}

// DeleteChannelModel 软删除渠道模型
func (h *ChannelHandler) DeleteChannelModel(c *gin.Context) {
	channelID, ok := h.parsePathInt64(c, "id")
	if !ok {
		return
	}
	modelID, ok := h.parsePathInt64(c, "modelId")
	if !ok {
		return
	}

	ctx := c.Request.Context()
	m, err := h.channelRepo.GetModelByID(ctx, modelID)
	if err != nil {
		logger.Error(ctx, "get channel model failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	if m == nil || m.ChannelID != channelID {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	if err := h.channelRepo.DeleteModelSoftWithBinding(ctx, channelID, modelID); err != nil {
		logger.Error(ctx, "delete channel model failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	h.invalidateModelCache(ctx)
	response.AdminSuccessMsg(c, "模型删除成功")
}

// UpdateChannelModelStatus 启用/禁用渠道模型
func (h *ChannelHandler) UpdateChannelModelStatus(c *gin.Context) {
	channelID, ok := h.parsePathInt64(c, "id")
	if !ok {
		return
	}
	modelID, ok := h.parsePathInt64(c, "modelId")
	if !ok {
		return
	}

	var req form.UpdateChannelModelStatusReq
	if !h.bindAndValidate(c, &req) {
		return
	}

	ctx := c.Request.Context()
	m, err := h.channelRepo.GetModelByID(ctx, modelID)
	if err != nil {
		logger.Error(ctx, "get channel model failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	if m == nil || m.ChannelID != channelID {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	if err := h.channelRepo.UpdateModelStatus(ctx, modelID, req.Status); err != nil {
		logger.Error(ctx, "update channel model status failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	h.invalidateModelCache(ctx)

	m.Status = req.Status
	response.AdminSuccess(c, m)
}

// FetchUpstreamModels 获取上游可用模型列表
func (h *ChannelHandler) FetchUpstreamModels(c *gin.Context) {
	channelID, ok := h.parsePathInt64(c, "id")
	if !ok {
		return
	}

	ctx := c.Request.Context()
	ids, err := h.channelModelFetcher.FetchUpstreamModels(ctx, channelID)
	if err != nil {
		logger.Error(ctx, "fetch upstream models failed: %v", err)
		if errors.Is(err, service.ErrChannelNotFound) {
			response.AdminFail(c, response.InvalidParams)
			return
		}
		var upstreamErr *service.UpstreamHTTPError
		if errors.As(err, &upstreamErr) {
			response.AdminFailMsg(c, response.InternalError, upstreamErr.AdminMessage())
			return
		}
		response.AdminFail(c, response.InternalError)
		return
	}

	response.AdminSuccess(c, ids)
}

// invalidateModelCache 清除渠道与模型映射相关缓存
func (h *ChannelHandler) invalidateModelCache(ctx context.Context) {
	patterns := []string{"xin:channels:active:*", "xin:modelmap:*"}
	for _, pattern := range patterns {
		iter := h.redis.Scan(ctx, 0, pattern, 0).Iterator()
		for iter.Next(ctx) {
			h.redis.Del(ctx, iter.Val())
		}
	}
}

// parsePathInt64 解析路径中的整型参数
func (h *ChannelHandler) parsePathInt64(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil {
		response.AdminFail(c, response.InvalidParams)
		return 0, false
	}
	return id, true
}

// bindAndValidate 参数校验辅助方法
func (h *ChannelHandler) bindAndValidate(c *gin.Context, req interface{}) bool {
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
