package model

import (
	"time"

	"gorm.io/gorm"
)

// ModelPriceBinding 价格方案 → 模型（渠道 × 模型）的绑定，对应 model_price_bindings 表
// 解除绑定使用软删，保留历史。
type ModelPriceBinding struct {
	ID           int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	ModelPriceID int64          `gorm:"column:model_price_id;not null;index;comment:绑定的价格方案 model_prices.id" json:"model_price_id"`
	ModelID      int64          `gorm:"column:model_id;not null;comment:绑定的渠道×模型 models.id" json:"model_id"`
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at" json:"-"`
}
