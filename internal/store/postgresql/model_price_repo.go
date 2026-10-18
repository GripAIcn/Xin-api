package postgresql

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"Xin-api/internal/model"
)

type ModelPriceRepo interface {
	// Create 新增价格方案
	Create(ctx context.Context, p *model.ModelPrice) error
	// List 获取全部价格方案（含绑定模型数 bound_count，仅统计未软删绑定）
	List(ctx context.Context) ([]model.ModelPriceWithCount, error)
	// GetByID 根据 ID 查询价格方案；未找到返回 (nil, nil)
	GetByID(ctx context.Context, id int64) (*model.ModelPrice, error)
	// GetByName 根据名称查询价格方案；未找到返回 (nil, nil)
	GetByName(ctx context.Context, name string) (*model.ModelPrice, error)
	// Update 整体更新价格方案
	Update(ctx context.Context, p *model.ModelPrice) error
	// UpdateStatus 修改价格方案状态（active/disabled）
	UpdateStatus(ctx context.Context, id int64, status string) error
	// Delete 删除价格方案，并在同一事务内软删其全部绑定
	Delete(ctx context.Context, id int64) error
	// ListBoundModels 查询绑定到指定价格方案的全部「渠道 × 模型」明细
	ListBoundModels(ctx context.Context, priceID int64) ([]model.PriceBoundModelEntry, error)
	// Exists 判断价格方案是否存在
	Exists(ctx context.Context, id int64) (bool, error)
}

type modelPriceRepo struct {
	db *gorm.DB
}

// NewModelPriceRepo 创建价格方案仓储
func NewModelPriceRepo(db *gorm.DB) ModelPriceRepo {
	return &modelPriceRepo{db: db}
}

func (r *modelPriceRepo) Create(ctx context.Context, p *model.ModelPrice) error {
	return gorm.G[model.ModelPrice](r.db).Create(ctx, p)
}

// List 获取全部价格方案，LEFT JOIN 绑定表统计未软删绑定数
func (r *modelPriceRepo) List(ctx context.Context) ([]model.ModelPriceWithCount, error) {
	var rows []model.ModelPriceWithCount
	err := r.db.WithContext(ctx).
		Model(&model.ModelPrice{}).
		Select("model_prices.id AS id, model_prices.name AS name, model_prices.input_price AS input_price, " +
			"model_prices.output_price AS output_price, model_prices.cached_input_price AS cached_input_price, " +
			"model_prices.cache_creation_price AS cache_creation_price, model_prices.currency AS currency, " +
			"model_prices.status AS status, model_prices.created_at AS created_at, model_prices.updated_at AS updated_at, " +
			"COUNT(b.id) AS bound_count").
		Joins("LEFT JOIN model_price_bindings b ON b.model_price_id = model_prices.id AND b.deleted_at IS NULL").
		Group("model_prices.id").
		Order("model_prices.id DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []model.ModelPriceWithCount{}, nil
	}
	return rows, nil
}

func (r *modelPriceRepo) GetByID(ctx context.Context, id int64) (*model.ModelPrice, error) {
	p, err := gorm.G[model.ModelPrice](r.db).Where("id = ?", id).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *modelPriceRepo) GetByName(ctx context.Context, name string) (*model.ModelPrice, error) {
	p, err := gorm.G[model.ModelPrice](r.db).Where("name = ?", name).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *modelPriceRepo) Update(ctx context.Context, p *model.ModelPrice) error {
	return r.db.WithContext(ctx).Save(p).Error
}

func (r *modelPriceRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	rows, err := gorm.G[model.ModelPrice](r.db).Where("id = ?", id).Update(ctx, "status", status)
	if err != nil || rows == 0 {
		return err
	}
	return nil
}

// Delete 删除价格方案，避免残留绑定：同一事务内先软删绑定再删除方案行
func (r *modelPriceRepo) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("model_price_id = ?", id).Delete(&model.ModelPriceBinding{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&model.ModelPrice{}).Error
	})
}

// ListBoundModels 查询绑定到指定价格方案的全部「渠道 × 模型」明细
// 说明：以 model_price_bindings 为主表，GORM 自动追加其 deleted_at IS NULL；
// models / channels 的软删条件写在 JOIN 内。
func (r *modelPriceRepo) ListBoundModels(ctx context.Context, priceID int64) ([]model.PriceBoundModelEntry, error) {
	var rows []model.PriceBoundModelEntry
	err := r.db.WithContext(ctx).
		Model(&model.ModelPriceBinding{}).
		Select("model_price_bindings.id AS binding_id, models.id AS model_id, models.channel_id AS channel_id, " +
			"channels.name AS channel_name, channels.provider AS channel_provider, models.name AS model_name, " +
			"models.upstream_model AS upstream_model, models.status AS model_status, channels.status AS channel_status, " +
			"models.ratio AS ratio").
		Joins("JOIN models ON models.id = model_price_bindings.model_id AND models.deleted_at IS NULL").
		Joins("JOIN channels ON channels.id = models.channel_id AND channels.deleted_at IS NULL").
		Where("model_price_bindings.model_price_id = ?", priceID).
		Order("channels.priority DESC, channels.weight DESC, models.name ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		return []model.PriceBoundModelEntry{}, nil
	}
	return rows, nil
}

// Exists 判断价格方案是否存在
func (r *modelPriceRepo) Exists(ctx context.Context, id int64) (bool, error) {
	var n int64
	if err := r.db.WithContext(ctx).Model(&model.ModelPrice{}).Where("id = ?", id).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}
