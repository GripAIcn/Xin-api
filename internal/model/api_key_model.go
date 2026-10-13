package model

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"gorm.io/gorm"
)

// HashApiKey 计算 API Key 明文的 SHA-256 十六进制摘要（落库只存该摘要，鉴权时比对）
func HashApiKey(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

// API Key 状态枚举，对应 api_keys.status
const (
	ApiKeyStatusActive   = "active"   // 启用
	ApiKeyStatusDisabled = "disabled" // 禁用（临时停用，可恢复）
)

// ApiKey 业务客户端对外分发凭证，对应 api_keys 表
// 说明：不存明文 Key，仅存 key_hash；明文仅在创建时返回一次。
type ApiKey struct {
	ID         int64          `gorm:"column:id;primaryKey;autoIncrement;comment:代理主键" json:"id"`
	KeyHash    string         `gorm:"column:key_hash;type:varchar(255);not null;comment:Key 的哈希（不存明文）" json:"-"`
	Name       string         `gorm:"column:name;type:varchar(100);comment:备注名" json:"name"`
	UserID     int64          `gorm:"column:user_id;not null;comment:所属用户 users.id" json:"user_id"`
	Status     string         `gorm:"column:status;type:varchar(32);not null;default:active;comment:状态 active-启用 disabled-禁用" json:"status"`
	ExpiresAt  *time.Time     `gorm:"column:expires_at;comment:过期时间，NULL=永不过期" json:"expires_at,omitempty"`
	LastUsedAt *time.Time     `gorm:"column:last_used_at;comment:最近使用时间" json:"last_used_at,omitempty"`
	CreatedAt  time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}
