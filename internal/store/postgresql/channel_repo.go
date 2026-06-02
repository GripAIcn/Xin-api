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
	// 第一步：获取该项目组下所有正常状态的渠道（不限制模型）
	var allChannels []model.Channel
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND status = 1", groupID).
		Order("weight DESC").
		Find(&allChannels).Error
	if err != nil {
		return nil, err
	}

	if len(allChannels) == 0 {
		return nil, nil
	}

	// 第二步：优先精确匹配指定模型的渠道
	var matchedChannels []model.Channel
	for _, ch := range allChannels {
		models := strings.Split(ch.ModelMapping, ",")
		for _, m := range models {
			if strings.TrimSpace(m) == modelName {
				matchedChannels = append(matchedChannels, ch)
				break
			}
		}
	}

	// 第三步：若找到精确匹配，直接返回
	if len(matchedChannels) > 0 {
		return matchedChannels, nil
	}

	// 第四步：容错策略 - 返回所有可用渠道（让上层按权重选择）
	// 场景：用户请求 "deepseek-v4-flash" 但渠道配置的是 "deepseek-chat"
	// 避免直接报错，提高网关可用性
	return allChannels, nil
}
