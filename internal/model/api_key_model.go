package model

import (
	"time"

	"gorm.io/gorm"
)

// ApiKey 业务客户端对外分发凭证，对应 api_keys 表
type ApiKey struct {
	Key       string         `gorm:"column:key;type:varchar(255);primaryKey" json:"key"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
}
