package form

// UserLoginReq 登录请求结构
type UserLoginReq struct {
	Username string `json:"username" validate:"required,min=5,max=10"`
	Password string `json:"password" validate:"required,min=8,max=20"`
}

// UserLoginResponse 登录响应结构
type UserLoginResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
}

// UpdatePasswordReq 修改密码请求
type UpdatePasswordReq struct {
	OldPassword string `json:"old_password" validate:"required,min=8,max=20"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=20"`
}

// UpdateAccountReq 修改账号信息请求
type UpdateAccountReq struct {
	Username string `json:"username" validate:"required,min=5,max=10"`
}
