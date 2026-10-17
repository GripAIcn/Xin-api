package model

import "time"

// 用户角色枚举，对应 users.role
const (
	UserRoleAdmin = "admin" // 控制面管理员
	UserRoleUser  = "user"  // 普通用户
)

// 用户状态枚举，对应 users.status
const (
	UserStatusActive   = "active"   // 启用
	UserStatusDisabled = "disabled" // 禁用
)

// User 控制面管理员表，对应 users 表
type User struct {
	ID       uint    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Username string  `gorm:"column:username;type:varchar(50);not null;unique;check:chk_users_username_len,char_length(username) >= 5;comment:全局唯一登录账号名（长度不少于5）" json:"username"`
	Password string  `gorm:"column:password;type:varchar(255);not null;comment:bcrypt 单向加密加盐密文" json:"-"`
	Email    *string `gorm:"column:email;type:varchar(255);unique;comment:邮箱（唯一，可空）" json:"email,omitempty"`
	Phone    *string `gorm:"column:phone;type:varchar(32);comment:手机号" json:"phone,omitempty"`
	Role     string  `gorm:"column:role;type:varchar(32);not null;default:admin;check:chk_users_role,role = 'admin' OR role = 'user';comment:角色 admin-管理员 user-普通用户" json:"role"`
	Status   string  `gorm:"column:status;type:varchar(32);not null;default:active;check:chk_users_status,status = 'active' OR status = 'disabled';comment:状态 active-启用 disabled-禁用" json:"status"`
	// MustChangePassword 为 true 时，除查询资料和修改密码外的控制面接口一律拒绝访问
	MustChangePassword bool       `gorm:"column:must_change_password;not null;default:false;comment:登录后必须修改密码" json:"must_change_password"`
	Balance            Money      `gorm:"column:balance;type:numeric(20,8);not null;default:0;comment:余额" json:"balance"`
	Quota              *Money     `gorm:"column:quota;type:numeric(20,8);comment:总额度上限，NULL=不限额" json:"quota,omitempty"`
	UsedQuota          Money      `gorm:"column:used_quota;type:numeric(20,8);not null;default:0;comment:累计已用额度" json:"used_quota"`
	Currency           string     `gorm:"column:currency;type:varchar(16);not null;default:CNY;comment:余额币种（用户结算币种固定 CNY）" json:"currency"`
	RPMLimit           *int       `gorm:"column:rpm_limit;comment:账户级每分钟请求数上限，NULL=不限" json:"rpm_limit,omitempty"`
	TPMLimit           *int       `gorm:"column:tpm_limit;comment:账户级每分钟 tokens 上限，NULL=不限" json:"tpm_limit,omitempty"`
	LastLoginAt        *time.Time `gorm:"column:last_login_at;comment:最近登录时间" json:"last_login_at,omitempty"`
	CreatedAt          time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}
