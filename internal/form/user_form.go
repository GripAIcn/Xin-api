package form

import (
	"time"

	"Xin-api/internal/model"
)

// UserLoginReq 登录请求结构
type UserLoginReq struct {
	Username string `json:"username" validate:"required,min=5,max=10"`
	Password string `json:"password" validate:"required,min=8,max=20"`
}

// UserLoginResponse 登录响应结构
type UserLoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// UpdatePasswordReq 修改密码请求
type UpdatePasswordReq struct {
	OldPassword string `json:"old_password" validate:"required,min=8,max=20"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=20"`
}

// CreateUserReq 管理员新建用户请求（默认创建普通用户）
type CreateUserReq struct {
	Username string  `json:"username" validate:"required,min=5,max=10"`
	Password string  `json:"password" validate:"required,min=8,max=20"`
	Email    *string `json:"email" validate:"omitempty,email,max=255"`
	Phone    *string `json:"phone" validate:"omitempty,max=32"`
}

// UpdateProfileReq 用户修改自身资料（邮箱/手机号，需校验原始密码）
type UpdateProfileReq struct {
	OldPassword string  `json:"old_password" validate:"required,min=8,max=20"`
	Email       *string `json:"email" validate:"omitempty,email,max=255"`
	Phone       *string `json:"phone" validate:"omitempty,max=32"`
}

// UpdateUserStatusReq 管理员启用/禁用用户请求
type UpdateUserStatusReq struct {
	Status string `json:"status" validate:"required,oneof=active disabled"`
}

// AdminResetPasswordReq 管理员重置用户密码请求
type AdminResetPasswordReq struct {
	NewPassword string `json:"new_password" validate:"required,min=8,max=20"`
}

// RechargeBalanceReq 管理员为用户充值请求（仅增加余额）
// 金额使用 decimal 解析，具体 >0 校验在 handler 中完成。
type RechargeBalanceReq struct {
	Amount model.Money `json:"amount" validate:"required"`
}

// UserResponse 用户安全字段（不含密码）
type UserResponse struct {
	ID          uint         `json:"id"`
	Username    string       `json:"username"`
	Email       *string      `json:"email,omitempty"`
	Phone       *string      `json:"phone,omitempty"`
	Role        string       `json:"role"`
	Status      string       `json:"status"`
	Balance     model.Money  `json:"balance"`
	Currency    string       `json:"currency"`
	Quota       *model.Money `json:"quota,omitempty"`
	UsedQuota   model.Money  `json:"used_quota"`
	LastLoginAt *time.Time   `json:"last_login_at,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}
