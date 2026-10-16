package postgresql

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"Xin-api/internal/model"
)

type ChannelRepo interface {
	// Create 创建上游渠道
	Create(ctx context.Context, channel *model.Channel) error
	// DeleteSoft 软删除上游渠道
	DeleteSoft(ctx context.Context, id int64) error
	// Update 完整更新渠道信息（常用于管理后台编辑页面整体提交）
	Update(ctx context.Context, channel *model.Channel) error
	// UpdateStatus 修改渠道状态 (active-正常 disabled-手动关闭 circuit_open-自动熔断)——常用于健康检查熔断器
	UpdateStatus(ctx context.Context, id int64, status string) error
	// GetByID 根据 ID 查询渠道详情
	GetByID(ctx context.Context, id int64) (*model.Channel, error)
	// ListAll 获取所有渠道（包含关闭和熔断的，供管理后台列表展示）
	ListAll(ctx context.Context) ([]model.Channel, error)
	// ListActiveByModel ⚡️核心流量方法：根据请求的模型，捞出所有当前可用的物理渠道（状态为 active）
	ListActiveByModel(ctx context.Context, modelName string) ([]model.Channel, error)
	// ListModelsByChannel 获取指定渠道下可用的模型（渠道 × 模型，来自 models 表）
	ListModelsByChannel(ctx context.Context, channelID int64) ([]model.Model, error)
	// ListModelsByChannelAll 获取指定渠道下全部模型（含 disabled，不含软删），供管理后台展示
	ListModelsByChannelAll(ctx context.Context, channelID int64) ([]model.Model, error)
	// GetModelByID 根据模型 ID 查询渠道模型记录（软删记录不返回）
	GetModelByID(ctx context.Context, id int64) (*model.Model, error)
	// GetModelByName 按 (channel_id, name) 查询渠道模型记录（软删记录不返回），用于判重
	GetModelByName(ctx context.Context, channelID int64, name string) (*model.Model, error)
	// GetUpstreamModel 按 (channel_id, name) 取该渠道登记的实际上游模型名；未登记或非 active 返回空串
	GetUpstreamModel(ctx context.Context, channelID int64, name string) (string, error)
	// CreateModel 为渠道新增模型（渠道 × 模型）
	CreateModel(ctx context.Context, m *model.Model) error
	// UpdateModel 更新模型（整体保存）
	UpdateModel(ctx context.Context, m *model.Model) error
	// UpdateModelStatus 修改模型状态（active/disabled）
	UpdateModelStatus(ctx context.Context, id int64, status string) error
	// DeleteModelSoft 软删除模型
	DeleteModelSoft(ctx context.Context, id int64) error
	// ListModelChannelRows 查询全部「渠道 × 模型」明细行（JOIN 渠道信息），供模型管理页按逻辑模型名聚合
	ListModelChannelRows(ctx context.Context) ([]model.ModelChannelEntry, error)
}

type channelRepo struct {
	db *gorm.DB
}

// NewChannelRepo 创建上游渠道仓储
func NewChannelRepo(db *gorm.DB) ChannelRepo {
	return &channelRepo{db: db}
}

func (r *channelRepo) Create(ctx context.Context, channel *model.Channel) error {
	return gorm.G[model.Channel](r.db).Create(ctx, channel)
}

// DeleteSoft 软删除渠道，并在同一事务内链式软删其名下全部 models，避免孤儿数据
func (r *channelRepo) DeleteSoft(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", id).Delete(&model.Channel{}).Error; err != nil {
			return err
		}
		return tx.Where("channel_id = ?", id).Delete(&model.Model{}).Error
	})
}

func (r *channelRepo) Update(ctx context.Context, channel *model.Channel) error {
	// Save 会根据主键 ID 是否存在来自动执行 UPDATE，并且会更新所有字段（包括零值如 status=0）
	return r.db.WithContext(ctx).Save(channel).Error
}

func (r *channelRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	rows, err := gorm.G[model.Channel](r.db).Where("id = ?", id).Update(ctx, "status", status)
	if err != nil || rows == 0 {
		return err
	}
	return nil
}

func (r *channelRepo) GetByID(ctx context.Context, id int64) (*model.Channel, error) {
	channel, err := gorm.G[model.Channel](r.db).Where("id = ?", id).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &channel, nil
}

func (r *channelRepo) ListAll(ctx context.Context) ([]model.Channel, error) {
	channels, err := gorm.G[model.Channel](r.db).Find(ctx)
	return channels, err
}

// ListActiveByModel 根据模型选出支持该模型的可用渠道
// 模型支持关系由 models 表（渠道 × 模型）行式表达；无精确匹配时回退全部可用渠道。
func (r *channelRepo) ListActiveByModel(ctx context.Context, modelName string) ([]model.Channel, error) {
	var channels []model.Channel
	err := r.db.WithContext(ctx).
		Model(&model.Channel{}).
		Select("channels.*").
		Joins("JOIN models ON models.channel_id = channels.id AND models.deleted_at IS NULL AND models.status = ?", model.ModelStatusActive).
		Where("channels.status = ?", model.ChannelStatusActive).
		Where("models.name = ?", modelName).
		Order("channels.priority DESC, channels.weight DESC").
		Find(&channels).Error
	if err != nil {
		return nil, err
	}

	if len(channels) > 0 {
		return channels, nil
	}

	// 容错策略 - 返回所有可用渠道（让上层按权重选择）
	// 场景：用户请求 "deepseek-v4-flash" 但渠道尚未登记对应模型
	// 避免直接报错，提高网关可用性
	err = r.db.WithContext(ctx).
		Where("status = ?", model.ChannelStatusActive).
		Order("priority DESC, weight DESC").
		Find(&channels).Error
	return channels, err
}

func (r *channelRepo) ListModelsByChannel(ctx context.Context, channelID int64) ([]model.Model, error) {
	models, err := gorm.G[model.Model](r.db).
		Where("channel_id = ? AND status = ?", channelID, model.ModelStatusActive).
		Find(ctx)
	return models, err
}

// ListModelsByChannelAll 获取指定渠道下全部模型（含 disabled，不含软删）
func (r *channelRepo) ListModelsByChannelAll(ctx context.Context, channelID int64) ([]model.Model, error) {
	models, err := gorm.G[model.Model](r.db).
		Where("channel_id = ?", channelID).
		Order("id ASC").
		Find(ctx)
	return models, err
}

// GetModelByID 根据模型 ID 查询渠道模型记录
func (r *channelRepo) GetModelByID(ctx context.Context, id int64) (*model.Model, error) {
	m, err := gorm.G[model.Model](r.db).Where("id = ?", id).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// GetModelByName 按 (channel_id, name) 查询渠道模型记录
func (r *channelRepo) GetModelByName(ctx context.Context, channelID int64, name string) (*model.Model, error) {
	m, err := gorm.G[model.Model](r.db).
		Where("channel_id = ? AND name = ?", channelID, name).
		First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

// GetUpstreamModel 按 (channel_id, name) 取上游真实模型名（仅未软删且 active），未登记返回空串
func (r *channelRepo) GetUpstreamModel(ctx context.Context, channelID int64, name string) (string, error) {
	m, err := gorm.G[model.Model](r.db).
		Select("upstream_model").
		Where("channel_id = ? AND name = ? AND status = ?", channelID, name, model.ModelStatusActive).
		First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return m.UpstreamModel, nil
}

// CreateModel 为渠道新增模型
func (r *channelRepo) CreateModel(ctx context.Context, m *model.Model) error {
	return gorm.G[model.Model](r.db).Create(ctx, m)
}

// UpdateModel 更新模型（整体保存，允许清空价格等可空字段）
func (r *channelRepo) UpdateModel(ctx context.Context, m *model.Model) error {
	return r.db.WithContext(ctx).Save(m).Error
}

// UpdateModelStatus 修改模型状态
func (r *channelRepo) UpdateModelStatus(ctx context.Context, id int64, status string) error {
	rows, err := gorm.G[model.Model](r.db).Where("id = ?", id).Update(ctx, "status", status)
	if err != nil || rows == 0 {
		return err
	}
	return nil
}

// DeleteModelSoft 软删除模型
func (r *channelRepo) DeleteModelSoft(ctx context.Context, id int64) error {
	rows, err := gorm.G[model.Model](r.db).Where("id = ?", id).Delete(ctx)
	if err != nil || rows == 0 {
		return err
	}
	return nil
}

// ListModelChannelRows 查询全部「渠道 × 模型」明细行（JOIN 渠道信息），供模型管理页按逻辑模型名聚合
// 说明：以 models 为主表，GORM 自动追加 models.deleted_at IS NULL；从表 channels 的软删条件写在 JOIN 内。
func (r *channelRepo) ListModelChannelRows(ctx context.Context) ([]model.ModelChannelEntry, error) {
	var rows []model.ModelChannelEntry
	err := r.db.WithContext(ctx).
		Model(&model.Model{}).
		Select("models.name AS name, models.id AS model_id, models.channel_id AS channel_id, " +
			"channels.name AS channel_name, channels.provider AS channel_provider, channels.status AS channel_status, " +
			"models.upstream_model AS upstream_model, models.status AS status, " +
			"models.context_length AS context_length, models.max_output_tokens AS max_output_tokens, " +
			"models.capabilities AS capabilities, models.input_price AS input_price, " +
			"models.output_price AS output_price, models.cached_input_price AS cached_input_price, " +
			"models.cache_creation_price AS cache_creation_price, models.currency AS currency").
		Joins("JOIN channels ON channels.id = models.channel_id AND channels.deleted_at IS NULL").
		Order("models.name ASC, channels.priority DESC, channels.weight DESC").
		Scan(&rows).Error
	return rows, err
}
