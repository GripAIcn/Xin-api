package postgresql

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"Xin-api/internal/model"
)

type GroupRepo interface {
	// Create 创建项目组
	Create(ctx context.Context, group *model.Group) error
	// DeleteSoft 软删除项目组（由于模型包含 gorm.DeletedAt，默认触发软删除）
	DeleteSoft(ctx context.Context, id int64) error
	// UpdateName 修改项目组名称
	UpdateName(ctx context.Context, id int64, newName string) error
	// UpdateStatus 修改项目组状态 (1-启用 0-禁用)
	UpdateStatus(ctx context.Context, id int64, status int) error
	// GetByID 根据 ID 查询项目组
	GetByID(ctx context.Context, id int64) (*model.Group, error)
	// ListAll 获取所有未删除的项目组
	ListAll(ctx context.Context) ([]model.Group, error)
}

type groupRepo struct {
	db *gorm.DB
}

// NewGroupRepo 创建项目组仓储
func NewGroupRepo(db *gorm.DB) GroupRepo {
	return &groupRepo{db: db}
}

func (r *groupRepo) Create(ctx context.Context, group *model.Group) error {
	return gorm.G[model.Group](r.db).Create(ctx, group)
}

func (r *groupRepo) DeleteSoft(ctx context.Context, id int64) error {
	// GORM 会自动将此操作转化为 UPDATE groups SET deleted_at = ... WHERE id = ?
	rows, err := gorm.G[model.Group](r.db).Where("id = ?", id).Delete(ctx)
	if err != nil || rows == 0 {
		return err
	}
	return nil
}

func (r *groupRepo) UpdateName(ctx context.Context, id int64, newName string) error {
	rows, err := gorm.G[model.Group](r.db).Where("id = ?", id).Update(ctx, "name", newName)
	if err != nil || rows == 0 {
		return err
	}
	return nil
}

func (r *groupRepo) UpdateStatus(ctx context.Context, id int64, status int) error {
	rows, err := gorm.G[model.Group](r.db).Where("id = ?", id).Update(ctx, "status", status)
	if err != nil || rows == 0 {
		return err
	}
	return nil
}

func (r *groupRepo) GetByID(ctx context.Context, id int64) (*model.Group, error) {
	group, err := gorm.G[model.Group](r.db).Where("id = ?", id).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 未找到不视作系统致命错误，回传空指针
		}
		return nil, err
	}
	return &group, nil
}

func (r *groupRepo) ListAll(ctx context.Context) ([]model.Group, error) {
	// Find 会自动过滤掉 deleted_at 不为空的记录
	groups, err := gorm.G[model.Group](r.db).Find(ctx)
	return groups, err
}

type ApiKeyRepo interface {
	// Create 分发一个 API Key
	Create(ctx context.Context, apiKey *model.ApiKey) error
	// DeleteSoft 软删除指定的 API Key
	DeleteSoft(ctx context.Context, key string) error
	// GetByKey 根据特定的秘钥串获取详情
	GetByKey(ctx context.Context, key string) (*model.ApiKey, error)
	// GetWithGroup 验证与网关数据面高频调用：获取 Key 详情并联动预加载(Preload)其对应的项目组信息
	GetWithGroup(ctx context.Context, key string) (*model.ApiKey, error)
	// ListByGroupID 获取某个项目组名下的所有可用密钥
	ListByGroupID(ctx context.Context, groupID int64) ([]model.ApiKey, error)
}

type apiKeyRepo struct {
	db *gorm.DB
}

// NewApiKeyRepo 创建凭证仓储
func NewApiKeyRepo(db *gorm.DB) ApiKeyRepo {
	return &apiKeyRepo{db: db}
}

func (r *apiKeyRepo) Create(ctx context.Context, apiKey *model.ApiKey) error {
	return gorm.G[model.ApiKey](r.db).Create(ctx, apiKey)
}

func (r *apiKeyRepo) DeleteSoft(ctx context.Context, key string) error {
	rows, err := gorm.G[model.ApiKey](r.db).Where("key = ?", key).Delete(ctx)
	if err != nil {
		return err
	}
	if rows == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *apiKeyRepo) GetByKey(ctx context.Context, key string) (*model.ApiKey, error) {
	apiKey, err := gorm.G[model.ApiKey](r.db).Where("key = ?", key).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &apiKey, nil
}

func (r *apiKeyRepo) GetWithGroup(ctx context.Context, key string) (*model.ApiKey, error) {
	var apiKey model.ApiKey
	// ⚡️ 生产亮点：利用 Preload 自动完成 JOIN 关联，一并查出所属的 Group 详情与状态
	err := r.db.WithContext(ctx).Preload("Group").Where("key = ?", key).First(&apiKey).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &apiKey, nil
}

func (r *apiKeyRepo) ListByGroupID(ctx context.Context, groupID int64) ([]model.ApiKey, error) {
	keys, err := gorm.G[model.ApiKey](r.db).Where("group_id = ?", groupID).Find(ctx)
	return keys, err
}
