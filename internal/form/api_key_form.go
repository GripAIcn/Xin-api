package form

import "time"

// ApiKeyResponse 返回给前端的凭证详情（隐藏不需要展示的 GORM 内部元数据）
type ApiKeyResponse struct {
	Key       string    `json:"key"`
	CreatedAt time.Time `json:"created_at"`
}

// DeleteApiKeyReq 删除 API Key 请求入参
type DeleteApiKeyReq struct {
	Key string `json:"key" binding:"required" validate:"required"` // API Key 字符串
}
