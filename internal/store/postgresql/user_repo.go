package postgresql

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"Xin-api/internal/model"
)

type UserRepo interface {
	// Create 新增用户（返回 user_id）
	Create(ctx context.Context, user *model.User) error
	// UpdatePassword 修改密码（通过 user_id）
	UpdatePassword(ctx context.Context, userID uint, newPasswordHash string) error
	// GetIdByUsername 根据用户名查询用户（用于登录验证）
	GetIdByUsername(ctx context.Context, username string) (*model.User, error)
	// GetByID 根据主键查询用户详情
	GetByID(ctx context.Context, id uint) (*model.User, error)
	// GetByEmail 根据邮箱查询用户
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	// ListAll 获取全部用户列表
	ListAll(ctx context.Context) ([]model.User, error)
	// UpdateProfile 更新用户资料（仅更新提供的邮箱与手机号）
	UpdateProfile(ctx context.Context, id uint, email *string, phone *string) error
	// UpdateStatus 更新用户状态
	UpdateStatus(ctx context.Context, id uint, status string) error
	// RechargeBalance 原子累加用户余额（只增不减）
	RechargeBalance(ctx context.Context, id uint, amount model.Money) error
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

func (r *userRepo) GetIdByUsername(ctx context.Context, username string) (*model.User, error) {
	user, err := gorm.G[model.User](r.db).Where("username = ?", username).First(ctx)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, nil
	}
	return &user, nil
}

func (r *userRepo) GetByID(ctx context.Context, id uint) (*model.User, error) {
	user, err := gorm.G[model.User](r.db).Where("id = ?", id).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user, err := gorm.G[model.User](r.db).Where("email = ?", email).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepo) ListAll(ctx context.Context) ([]model.User, error) {
	users, err := gorm.G[model.User](r.db).Find(ctx)
	return users, err
}

func (r *userRepo) UpdateProfile(ctx context.Context, id uint, email *string, phone *string) error {
	updates := map[string]any{}
	if email != nil {
		updates["email"] = *email
	}
	if phone != nil {
		updates["phone"] = *phone
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Updates(updates).Error
}

func (r *userRepo) UpdateStatus(ctx context.Context, id uint, status string) error {
	rows, err := gorm.G[model.User](r.db).Where("id = ?", id).Update(ctx, "status", status)
	if err != nil || rows == 0 {
		return err
	}
	return nil
}

func (r *userRepo) RechargeBalance(ctx context.Context, id uint, amount model.Money) error {
	return r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Update("balance", gorm.Expr("balance + ?", amount)).Error
}
