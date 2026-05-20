package postgresql

import (
	"context"
	"errors"
	"strings"

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
	// UpdateStatus 修改渠道状态 (1-正常 0-手动关闭 2-自动熔断)——常用于健康检查熔断器
	UpdateStatus(ctx context.Context, id int64, status int) error
	// GetByID 根据 ID 查询渠道详情
	GetByID(ctx context.Context, id int64) (*model.Channel, error)
	// ListByGroupID 获取某个项目组名下的所有渠道（包含关闭和熔断的，供管理后台列表展示）
	ListByGroupID(ctx context.Context, groupID int64) ([]model.Channel, error)
	// ListActiveByGroupAndModel ⚡️核心流量方法：根据项目组和请求的模型，捞出所有当前可用的物理渠道（状态为 1-正常）
	ListActiveByGroupAndModel(ctx context.Context, groupID int64, modelName string) ([]model.Channel, error)
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

func (r *channelRepo) UpdateStatus(ctx context.Context, id int64, status int) error {
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

func (r *channelRepo) ListByGroupID(ctx context.Context, groupID int64) ([]model.Channel, error) {
	channels, err := gorm.G[model.Channel](r.db).Where("group_id = ?", groupID).Find(ctx)
	return channels, err
}

func (r *channelRepo) ListActiveByGroupAndModel(ctx context.Context, groupID int64, modelName string) ([]model.Channel, error) {
	var channels []model.Channel

	// ⚡️ 生产级语义：
	// 1. group_id = ? 锁定项目组
	// 2. status = 1 必须是正常态（过滤掉手动关闭和自动熔断的渠道）
	// 3. model_mapping LIKE ? 模糊匹配支持的模型 (由于存储格式是逗号分隔，如 "gpt-4o,deepseek-chat"，使用 ILIKE 或 LIKE 进行安全匹配)
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND status = 1 AND model_mapping LIKE ?", groupID, "%"+modelName+"%").
		Order("weight DESC"). // 按权重降序，方便上层配合算法进行负载均衡
		Find(&channels).Error

	if err != nil {
		return nil, err
	}

	// 内存二次精细化过滤（防范例如请求 "gpt-4" 却匹配到了 "gpt-4o" 的边界边缘情况）
	var filtered []model.Channel
	for _, ch := range channels {
		models := strings.Split(ch.ModelMapping, ",")
		for _, m := range models {
			if strings.TrimSpace(m) == modelName {
				filtered = append(filtered, ch)
				break
			}
		}
	}

	return filtered, nil
}
