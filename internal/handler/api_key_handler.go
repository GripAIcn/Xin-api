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
// 明文 Key 仅在本次响应中返回一次，落库只存 SHA-256 哈希
func (h *ApiKeyHandler) CreateApiKey(c *gin.Context) {
	var req form.CreateApiKeyReq
	if c.Request.ContentLength > 0 {
		if !h.bindAndValidate(c, &req) {
			return
		}
	}

	// api_keys.user_id 非空：从 JWT 上下文取当前登录用户 id
	uidRaw, exists := c.Get("user_id")
	if !exists {
		response.AdminFail(c, response.Unauthorized)
		return
	}
	uidFloat, ok := uidRaw.(float64) // JWT 数字解析通常为 float64
	if !ok {
		response.AdminFail(c, response.Unauthorized)
		return
	}

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
		KeyHash:   model.HashApiKey(generatedKey), // 落库存哈希，不存明文
		Name:      req.Name,
		UserID:    int64(uidFloat),
		Status:    model.ApiKeyStatusActive,
		ExpiresAt: req.ExpiresAt,
	}

	// 写入数据库
	if err := h.apiKeyRepo.Create(c.Request.Context(), apiKey); err != nil {
		logger.Error(c.Request.Context(), "create api key failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	// 明文 Key 仅在创建时返回一次
	response.AdminSuccess(c, form.CreateApiKeyResp{
		Key:            generatedKey,
		ApiKeyResponse: toApiKeyResponse(apiKey),
	})
}

// DeleteApiKey 按 id 软删除指定的密钥凭证
func (h *ApiKeyHandler) DeleteApiKey(c *gin.Context) {
	var req form.DeleteApiKeyReq
	if !h.bindAndValidate(c, &req) {
		return
	}

	if err := h.apiKeyRepo.DeleteSoft(c.Request.Context(), req.ID); err != nil {
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

	// 批量转化为响应 DTO 数组（不暴露明文与哈希）
	var resp []form.ApiKeyResponse
	for _, k := range keys {
		resp = append(resp, toApiKeyResponse(&k))
	}

	response.AdminSuccess(c, resp)
}

// toApiKeyResponse 将模型转换为对外响应 DTO
func toApiKeyResponse(k *model.ApiKey) form.ApiKeyResponse {
	return form.ApiKeyResponse{
		ID:         k.ID,
		Name:       k.Name,
		UserID:     k.UserID,
		Status:     k.Status,
		ExpiresAt:  k.ExpiresAt,
		LastUsedAt: k.LastUsedAt,
		CreatedAt:  k.CreatedAt,
	}
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
