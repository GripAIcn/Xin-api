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

type ChannelHandler struct {
	channelRepo postgresql.ChannelRepo
	validate    *validator.Validate
}

func NewChannelHandler(repo postgresql.ChannelRepo) *ChannelHandler {
	return &ChannelHandler{
		channelRepo: repo,
		validate:    validator.New(),
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
