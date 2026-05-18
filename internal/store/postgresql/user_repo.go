package postgresql

import (
	"context"
	"errors"

	"Xin-api/internal/model"

	"gorm.io/gorm"
)

type UserRepo interface {
	// Create 新增用户（返回 user_id）
	Create(ctx context.Context, user *model.User) error
	// UpdatePassword 修改密码（通过 user_id）
	UpdatePassword(ctx context.Context, userID uint, newPasswordHash string) error
	// UpdateAccount 修改账号信息（如 username 或 email，按需）
	UpdateAccount(ctx context.Context, userID uint, username string) error
	// GetIdByUsername 根据用户名查询用户（用于登录验证）
	GetIdByUsername(ctx context.Context, username string) (*model.User, error)
}

type userRepo struct {
	db *gorm.DB
}

// NewUserRepo 创建用户仓储
func NewUserRepo(db *gorm.DB) UserRepo {
	return &userRepo{db: db}
}

func (r *userRepo) Create(ctx context.Context, user *model.User) error {
	return gorm.G[model.User](r.db).Create(ctx, user)
}

func (r *userRepo) UpdatePassword(ctx context.Context, userID uint, newPasswordHash string) error {
	rows, err := gorm.G[model.User](r.db).Where("id = ?", userID).Update(ctx, "password", newPasswordHash)
	if err != nil || rows == 0 {
		return err
	}
	return nil
}

func (r *userRepo) UpdateAccount(ctx context.Context, userID uint, username string) error {
	rows, err := gorm.G[model.User](r.db).Where("id = ?", userID).Update(ctx, "username", username)
	if err != nil || rows == 0 {
		return err
	}
	return nil
}

func (r *userRepo) GetIdByUsername(ctx context.Context, username string) (*model.User, error) {
	user, err := gorm.G[model.User](r.db).Where("username = ?", username).First(ctx)
	if err != nil || errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &user, nil
}
