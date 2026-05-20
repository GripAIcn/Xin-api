package model

import (
	"time"

	"gorm.io/gorm"
)

// Group 业务项目组，对应 groups 表
type Group struct {
	ID        int64          `gorm:"column:id;primaryKey;autoIncrement"`
	Name      string         `gorm:"column:name;type:varchar(100);not null"`
	Status    int            `gorm:"column:status;default:1;comment:状态 1-启用 0-禁用"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// ApiKey 业务客户端对外分发凭证，对应 api_keys 表
type ApiKey struct {
	Key       string         `gorm:"column:key;type:varchar(255);primaryKey"`
	GroupID   int64          `gorm:"column:group_id;not null;index;comment:所属项目组ID"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
	// 外键关联
	Group Group `gorm:"foreignKey:GroupID;references:ID;constraint:OnDelete:CASCADE"`
}
