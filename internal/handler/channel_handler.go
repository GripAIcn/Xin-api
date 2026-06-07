package handler

import (
	"context"
	"fmt"
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
	var req form.CreateChannelReq // 包含 GroupID, Name, ModelMapping, BaseURL, APIKey, Weight
	if !h.bindAndValidate(c, &req) {
		return
	}

	channel := &model.Channel{
		GroupID:      req.GroupID,
		Name:         req.Name,
		ModelMapping: req.ModelMapping,
		BaseURL:      req.BaseURL,
		APIKey:       req.APIKey,
		Weight:       req.Weight,
		Status:       1, // 默认正常
	}

	if err := h.channelRepo.Create(c.Request.Context(), channel); err != nil {
		logger.Error(c.Request.Context(), "create channel failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	// 清除缓存
	h.invalidateChannelCache(c.Request.Context(), channel.GroupID)

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

	// 先获取 groupID 用于清除缓存
	channel, _ := h.channelRepo.GetByID(c.Request.Context(), id)

	if err := h.channelRepo.DeleteSoft(c.Request.Context(), id); err != nil {
		logger.Error(c.Request.Context(), "delete channel failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	// 清除缓存
	if channel != nil {
		h.invalidateChannelCache(c.Request.Context(), channel.GroupID)
	}

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
		GroupID:      req.GroupID,
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
	h.invalidateChannelCache(c.Request.Context(), channel.GroupID)

	response.AdminSuccess(c, channel)
}

// ListChannelsByGroup 获取项目组名下的全量渠道
func (h *ChannelHandler) ListChannelsByGroup(c *gin.Context) {
	groupIDStr := c.Query("group_id")
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	channels, err := h.channelRepo.ListByGroupID(c.Request.Context(), groupID)
	if err != nil {
		logger.Error(c.Request.Context(), "list channels failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	response.AdminSuccess(c, channels)
}

// invalidateChannelCache 清除指定项目组的渠道缓存
func (h *ChannelHandler) invalidateChannelCache(ctx context.Context, groupID int64) {
	pattern := fmt.Sprintf("xin:channels:active:%d:*", groupID)
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

// TestGroupChannels 测试项目组下所有渠道
func (h *ChannelHandler) TestGroupChannels(c *gin.Context) {
	idStr := c.Param("id")
	groupID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	results, err := h.channelTester.TestGroupAllChannels(c.Request.Context(), groupID)
	if err != nil {
		logger.Error(c.Request.Context(), "test group channels failed: %v", err)
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
