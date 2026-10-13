package form

import "time"

// ApiKeyResponse 返回给前端的凭证详情（隐藏哈希等敏感字段）
type ApiKeyResponse struct {
	ID         int64      `json:"id"`
	Name       string     `json:"name"`
	UserID     int64      `json:"user_id"`
	Status     string     `json:"status"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// CreateApiKeyReq 创建凭证请求入参（字段均可选，未填则使用默认值）
type CreateApiKeyReq struct {
	Name      string     `json:"name" validate:"omitempty,max=100"` // 备注名
	ExpiresAt *time.Time `json:"expires_at"`                        // 过期时间，为空表示永不过期
}

// CreateApiKeyResp 创建凭证响应（明文 Key 仅在创建时返回一次）
type CreateApiKeyResp struct {
	Key string `json:"key"`
	ApiKeyResponse
}

// DeleteApiKeyReq 吊销 API Key 请求入参（按代理主键 id 定位）
type DeleteApiKeyReq struct {
	ID int64 `json:"id" binding:"required" validate:"required"` // API Key 代理主键
}
