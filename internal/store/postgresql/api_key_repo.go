package postgresql

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"Xin-api/internal/model"
)

type ApiKeyRepo interface {
	// Create 分发一个 API Key（落库仅存 key_hash）
	Create(ctx context.Context, apiKey *model.ApiKey) error
	// DeleteSoftByUser 按代理主键软删除指定的 API Key，且仅限该 Key 的归属用户
	DeleteSoftByUser(ctx context.Context, id int64, userID int64) error
	// GetByKeyHash 根据 Key 的哈希摘要获取详情（鉴权用）
	GetByKeyHash(ctx context.Context, keyHash string) (*model.ApiKey, error)
	// ListByUserID 获取指定用户自己的全部可用密钥
	ListByUserID(ctx context.Context, userID int64) ([]model.ApiKey, error)
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

func (r *apiKeyRepo) DeleteSoftByUser(ctx context.Context, id int64, userID int64) error {
	rows, err := gorm.G[model.ApiKey](r.db).Where("id = ? AND user_id = ?", id, userID).Delete(ctx)
	if err != nil {
		return err
	}
	if rows == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *apiKeyRepo) GetByKeyHash(ctx context.Context, keyHash string) (*model.ApiKey, error) {
	apiKey, err := gorm.G[model.ApiKey](r.db).Where("key_hash = ?", keyHash).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &apiKey, nil
}

func (r *apiKeyRepo) ListByUserID(ctx context.Context, userID int64) ([]model.ApiKey, error) {
	keys, err := gorm.G[model.ApiKey](r.db).Where("user_id = ?", userID).Find(ctx)
	return keys, err
}
