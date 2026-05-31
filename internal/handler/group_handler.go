package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"Xin-api/internal/form"
	"Xin-api/internal/model"
	"Xin-api/internal/store/postgresql"
	"Xin-api/pkg/response"
)

type GroupHandler struct {
	groupRepo postgresql.GroupRepo
	validate  *validator.Validate
}

func NewGroupHandler(repo postgresql.GroupRepo) *GroupHandler {
	return &GroupHandler{
		groupRepo: repo,
		validate:  validator.New(),
	}
}

// CreateGroup 创建业务项目组
func (h *GroupHandler) CreateGroup(c *gin.Context) {
	var req form.CreateGroupReq // 需在 form 包中定义，包含 Name
	if !h.bindAndValidate(c, &req) {
		return
	}

	group := &model.Group{
		Name:   req.Name,
		Status: 1, // 默认启用
	}

	if err := h.groupRepo.Create(c.Request.Context(), group); err != nil {
		logger.Error(c.Request.Context(), "create group failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	response.AdminSuccess(c, group)
}

// DeleteGroup 软删除项目组
func (h *GroupHandler) DeleteGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	if err := h.groupRepo.DeleteSoft(c.Request.Context(), id); err != nil {
		logger.Error(c.Request.Context(), "delete group failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	response.AdminSuccess(c, "项目组删除成功")
}

// UpdateGroup 更新项目组基础信息
func (h *GroupHandler) UpdateGroup(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	var req form.UpdateGroupReq
	if !h.bindAndValidate(c, &req) {
		return
	}

	if err := h.groupRepo.UpdateName(c.Request.Context(), id, req.Name); err != nil {
		logger.Error(c.Request.Context(), "update group name failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	group, err := h.groupRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		logger.Error(c.Request.Context(), "fetch updated group failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	response.AdminSuccess(c, group)
}

// UpdateStatus 启停项目组状态
func (h *GroupHandler) UpdateStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	var req form.UpdateGroupStatusReq // 包含 Status 字段
	if !h.bindAndValidate(c, &req) {
		return
	}

	if err := h.groupRepo.UpdateStatus(c.Request.Context(), id, req.Status); err != nil {
		logger.Error(c.Request.Context(), "update group status failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	group, err := h.groupRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		logger.Error(c.Request.Context(), "fetch updated group failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	response.AdminSuccess(c, group)
}

// ListGroups 获取全部项目组
func (h *GroupHandler) ListGroups(c *gin.Context) {
	groups, err := h.groupRepo.ListAll(c.Request.Context())
	if err != nil {
		logger.Error(c.Request.Context(), "list groups failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	response.AdminSuccess(c, groups)
}

// bindAndValidate 参数校验辅助方法
func (h *GroupHandler) bindAndValidate(c *gin.Context, req interface{}) bool {
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

type ApiKeyHandler struct {
	apiKeyRepo postgresql.ApiKeyRepo
	validate   *validator.Validate
}

func NewApiKeyHandler(repo postgresql.ApiKeyRepo) *ApiKeyHandler {
	return &ApiKeyHandler{
		apiKeyRepo: repo,
		validate:   validator.New(),
	}
}

// CreateApiKey 为指定项目组分发高强度密钥凭证
func (h *ApiKeyHandler) CreateApiKey(c *gin.Context) {
	var req form.CreateApiKeyReq
	if !h.bindAndValidate(c, &req) {
		return
	}

	// 1. ⚡️ 生产亮点：后端自动生成 32 字节高强度安全的随机密钥，拼接特定业务前缀
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		logger.Error(c.Request.Context(), "generate secure random key failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	// 生成类似: sk-xin-4a6f9c8d3e2b1a0f9b8c7d6e5f4a3b2c 的高强度密钥
	generatedKey := fmt.Sprintf("sk-xin-%s", hex.EncodeToString(randomBytes))

	apiKey := &model.ApiKey{
		Key:     generatedKey,
		GroupID: req.GroupID,
	}

	// 2. 写入数据库
	if err := h.apiKeyRepo.Create(c.Request.Context(), apiKey); err != nil {
		logger.Error(c.Request.Context(), "create api key failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	// 3. 返回清洗后的 DTO 对象，隐藏不必要的 ORM 级 DeletedAt 信息
	response.AdminSuccess(c, form.ApiKeyResponse{
		Key:       apiKey.Key,
		GroupID:   apiKey.GroupID,
		CreatedAt: apiKey.CreatedAt,
	})
}

// DeleteApiKey 软删除指定的密钥凭证
func (h *ApiKeyHandler) DeleteApiKey(c *gin.Context) {
	var req form.DeleteApiKeyReq
	if !h.bindAndValidate(c, &req) {
		return
	}

	if err := h.apiKeyRepo.DeleteSoft(c.Request.Context(), req.Key); err != nil {
		logger.Error(c.Request.Context(), "delete api key failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	response.AdminSuccess(c, "凭证已成功吊销")
}

// ListApiKeys 获取某个项目组名下的全量可用密钥列表
func (h *ApiKeyHandler) ListApiKeys(c *gin.Context) {
	groupIDStr := c.Query("group_id")
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	keys, err := h.apiKeyRepo.ListByGroupID(c.Request.Context(), groupID)
	if err != nil {
		logger.Error(c.Request.Context(), "list api keys failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	// 批量转化为响应 DTO 数组（按需，也可以直接返回 keys 数组）
	var resp []form.ApiKeyResponse
	for _, k := range keys {
		resp = append(resp, form.ApiKeyResponse{
			Key:       k.Key,
			GroupID:   k.GroupID,
			CreatedAt: k.CreatedAt,
		})
	}

	response.AdminSuccess(c, resp)
}

// bindAndValidate 参数校验辅助方法
func (h *ApiKeyHandler) bindAndValidate(c *gin.Context, req interface{}) bool {
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
