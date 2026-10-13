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

func (r *channelRepo) DeleteSoft(ctx context.Context, id int64) error {
	rows, err := gorm.G[model.Channel](r.db).Where("id = ?", id).Delete(ctx)
	if err != nil || rows == 0 {
		return err
	}
	return nil
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
		Joins("JOIN models ON models.channel_id = channels.id AND models.deleted_at IS NULL").
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
