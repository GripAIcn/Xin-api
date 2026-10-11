package handler

import (
	"context"
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
	channelRepo   postgresql.ChannelRepo
	channelTester *service.ChannelTester
	redis         *redis.Client
	validate      *validator.Validate
}

func NewChannelHandler(repo postgresql.ChannelRepo, rdb *redis.Client) *ChannelHandler {
	return &ChannelHandler{
		channelRepo:   repo,
		channelTester: service.NewChannelTester(repo),
		redis:         rdb,
		validate:      validator.New(),
	}
}

// CreateChannel 添加上游渠道供应商
func (h *ChannelHandler) CreateChannel(c *gin.Context) {
	var req form.CreateChannelReq // 包含 Name, ModelMapping, BaseURL, APIKey, Weight
	if !h.bindAndValidate(c, &req) {
		return
	}

	channel := &model.Channel{
		Name:         req.Name,
		ModelMapping: req.ModelMapping,
		BaseURL:      req.BaseURL,
		APIKey:       req.APIKey,
		Weight:       req.Weight,
		Status:       model.ChannelStatusActive, // 默认正常
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

	response.AdminSuccess(c, "渠道删除成功")
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
		ID:           id,
		Name:         req.Name,
		ModelMapping: req.ModelMapping,
		BaseURL:      req.BaseURL,
		APIKey:       req.APIKey,
		Weight:       req.Weight,
		Status:       req.Status, // 允许编辑时调整状态
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
