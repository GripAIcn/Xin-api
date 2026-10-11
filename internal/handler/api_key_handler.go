package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"Xin-api/internal/form"
	"Xin-api/internal/model"
	"Xin-api/internal/store/postgresql"
	"Xin-api/pkg/response"
)

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

// CreateApiKey 分发高强度密钥凭证
func (h *ApiKeyHandler) CreateApiKey(c *gin.Context) {
	// ⚡️ 生产亮点：后端自动生成 32 字节高强度安全的随机密钥，拼接特定业务前缀
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		logger.Error(c.Request.Context(), "generate secure random key failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	// 生成类似: sk-xin-4a6f9c8d3e2b1a0f9b8c7d6e5f4a3b2c 的高强度密钥
	generatedKey := fmt.Sprintf("sk-xin-%s", hex.EncodeToString(randomBytes))

	apiKey := &model.ApiKey{
		Key: generatedKey,
	}

	// 写入数据库
	if err := h.apiKeyRepo.Create(c.Request.Context(), apiKey); err != nil {
		logger.Error(c.Request.Context(), "create api key failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	// 返回清洗后的 DTO 对象，隐藏不必要的 ORM 级 DeletedAt 信息
	response.AdminSuccess(c, form.ApiKeyResponse{
		Key:       apiKey.Key,
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

// ListApiKeys 获取全量可用密钥列表
func (h *ApiKeyHandler) ListApiKeys(c *gin.Context) {
	keys, err := h.apiKeyRepo.ListAll(c.Request.Context())
	if err != nil {
		logger.Error(c.Request.Context(), "list api keys failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	// 批量转化为响应 DTO 数组
	var resp []form.ApiKeyResponse
	for _, k := range keys {
		resp = append(resp, form.ApiKeyResponse{
			Key:       k.Key,
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
