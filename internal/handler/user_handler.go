package handler

import (
	"strconv"
	"time"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"Xin-api/config"
	"Xin-api/internal/form"
	"Xin-api/internal/middleware"
	"Xin-api/internal/model"
	"Xin-api/internal/store/postgresql"
	"Xin-api/pkg/response"
)

// UserHandler 用户相关接口处理器
type UserHandler struct {
	userRepo postgresql.UserRepo
	validate *validator.Validate
	jwtCfg   config.JWTConfig // JWT 配置（密钥、过期时间）
}

// NewUserHandler 构造函数注入依赖
func NewUserHandler(repo postgresql.UserRepo, jwtCfg config.JWTConfig) *UserHandler {
	return &UserHandler{
		userRepo: repo,
		validate: validator.New(),
		jwtCfg:   jwtCfg,
	}
}

// toUserResponse 将 model.User 映射为不含密码的安全响应结构
func toUserResponse(u model.User) form.UserResponse {
	return form.UserResponse{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		Phone:       u.Phone,
		Role:        u.Role,
		Status:      u.Status,
		Balance:     u.Balance,
		Currency:    u.Currency,
		Quota:       u.Quota,
		UsedQuota:   u.UsedQuota,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

func (h *UserHandler) Login(c *gin.Context) {
	var req form.UserLoginReq

	// 1. 参数绑定与校验
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error(c.Request.Context(), "bind login request failed: %v", err)
		// ⚡️ 优化：统一交由业务状态码管理，消除硬编码字符串
		response.AdminFail(c, response.InvalidParams)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		logger.Warn(c.Request.Context(), "validate login request failed: %v", err)
		response.AdminFail(c, response.InvalidParams)
		return
	}

	// 2. 查询用户
	user, err := h.userRepo.GetIdByUsername(c.Request.Context(), req.Username)
	if err != nil {
		logger.Error(c.Request.Context(), "query user failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	if user == nil {
		logger.Warn(c.Request.Context(), "user not found: %s", req.Username)
		response.AdminFail(c, response.Unauthorized) // 统一凭证错误响应
		return
	}

	// 2.1 禁用用户不可登录
	if user.Status == model.UserStatusDisabled {
		logger.Warn(c.Request.Context(), "disabled user login attempt: %s (id=%d)", user.Username, user.ID)
		response.AdminFail(c, response.Unauthorized)
		return
	}

	// 3. 验证密码（bcrypt 比较）
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		logger.Warn(c.Request.Context(), "password mismatch for user: %s", req.Username)
		response.AdminFail(c, response.Unauthorized)
		return
	}

	// 4. 生成 JWT Token
	token, err := h.generateJWT(user)
	if err != nil {
		logger.Error(c.Request.Context(), "generate jwt failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	// 5. 返回成功响应
	logger.Info(c.Request.Context(), "user login success: %s (id=%d)", user.Username, user.ID)

	response.AdminSuccess(c, form.UserLoginResponse{
		Token:    token,
		Username: user.Username,
		Role:     user.Role,
	})
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var req form.CreateUserReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	// 检查用户名是否已存在
	existUser, err := h.userRepo.GetIdByUsername(c.Request.Context(), req.Username)
	if err != nil {
		logger.Error(c.Request.Context(), "check username exist failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	if existUser != nil {
		// 业务冲突：用户名已存在
		response.AdminFail(c, response.InvalidParams)
		return
	}

	// 检查邮箱是否已被占用
	if req.Email != nil {
		existEmailUser, err := h.userRepo.GetByEmail(c.Request.Context(), *req.Email)
		if err != nil {
			logger.Error(c.Request.Context(), "check email exist failed: %v", err)
			response.AdminFail(c, response.InternalError)
			return
		}
		if existEmailUser != nil {
			response.AdminFail(c, response.InvalidParams)
			return
		}
	}

	// 使用 bcrypt 对新账号密码进行单向哈希加盐加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Error(c.Request.Context(), "hash password failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	newUser := &model.User{
		Username: req.Username,
		Password: string(hashedPassword),
		Email:    req.Email,
		Phone:    req.Phone,
		Role:     model.UserRoleUser,     // 默认创建为普通用户
		Status:   model.UserStatusActive, // 显式写入，满足 CHECK (status IN (...))
		Balance:  0,
		Currency: "USD",
		Quota:    nil,
	}

	if err := h.userRepo.Create(c.Request.Context(), newUser); err != nil {
		logger.Error(c.Request.Context(), "create user failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	response.AdminSuccess(c, "账户创建成功")
}

// ListUsers 获取全部用户列表（管理员）
func (h *UserHandler) ListUsers(c *gin.Context) {
	users, err := h.userRepo.ListAll(c.Request.Context())
	if err != nil {
		logger.Error(c.Request.Context(), "list users failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	list := make([]form.UserResponse, 0, len(users))
	for _, u := range users {
		list = append(list, toUserResponse(u))
	}

	response.AdminSuccess(c, list)
}

// UpdateUserStatus 管理员启用/禁用用户
func (h *UserHandler) UpdateUserStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	var req form.UpdateUserStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	// 确认用户存在
	user, err := h.userRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		logger.Error(c.Request.Context(), "query user failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	if user == nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	if err := h.userRepo.UpdateStatus(c.Request.Context(), uint(id), req.Status); err != nil {
		logger.Error(c.Request.Context(), "update user status failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	response.AdminSuccess[any](c, nil)
}

// AdminResetPassword 管理员重置指定用户密码
func (h *UserHandler) AdminResetPassword(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	var req form.AdminResetPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	// 确认用户存在
	user, err := h.userRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		logger.Error(c.Request.Context(), "query user failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	if user == nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		logger.Error(c.Request.Context(), "hash password failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	if err := h.userRepo.UpdatePassword(c.Request.Context(), uint(id), string(newHashedPassword)); err != nil {
		logger.Error(c.Request.Context(), "reset password failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	response.AdminSuccess[any](c, nil)
}

// RechargeBalance 管理员为用户充值（仅增加余额）
func (h *UserHandler) RechargeBalance(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	var req form.RechargeBalanceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	// 确认用户存在
	user, err := h.userRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		logger.Error(c.Request.Context(), "query user failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	if user == nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	if err := h.userRepo.RechargeBalance(c.Request.Context(), uint(id), req.Amount); err != nil {
		logger.Error(c.Request.Context(), "recharge balance failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	// 重新查询返回充值后的最新余额
	updated, err := h.userRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		logger.Error(c.Request.Context(), "query user after recharge failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	logger.Info(c.Request.Context(), "recharge balance success: user=%d amount=%f", uint(id), req.Amount)
	response.AdminSuccess(c, toUserResponse(*updated))
}

// GetProfile 获取当前登录用户的个人资料（所有登录用户可访问）
func (h *UserHandler) GetProfile(c *gin.Context) {
	currentUserID, exists := c.Get(middleware.CtxUserIDKey)
	if !exists {
		response.AdminFail(c, response.Unauthorized)
		return
	}
	uID := uint(currentUserID.(float64)) // JWT 解析数字通常为 float64

	user, err := h.userRepo.GetByID(c.Request.Context(), uID)
	if err != nil {
		logger.Error(c.Request.Context(), "query user failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	if user == nil {
		response.AdminFail(c, response.Unauthorized)
		return
	}

	response.AdminSuccess(c, toUserResponse(*user))
}

// UpdateProfile 普通用户修改自身邮箱/手机号（需校验原始密码）
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	var req form.UpdateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	// 从上下文提取当前操作人
	currentUserID, exists := c.Get(middleware.CtxUserIDKey)
	if !exists {
		response.AdminFail(c, response.Unauthorized)
		return
	}
	uID := uint(currentUserID.(float64)) // JWT 解析数字通常为 float64

	// 重新捞出用户，校验原始密码
	user, err := h.userRepo.GetByID(c.Request.Context(), uID)
	if err != nil {
		logger.Error(c.Request.Context(), "query user failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}
	if user == nil {
		response.AdminFail(c, response.Unauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		logger.Warn(c.Request.Context(), "user %d old password verification failed", uID)
		response.AdminFail(c, response.Unauthorized)
		return
	}

	// 邮箱变更需保证全局唯一
	if req.Email != nil {
		existEmailUser, err := h.userRepo.GetByEmail(c.Request.Context(), *req.Email)
		if err != nil {
			logger.Error(c.Request.Context(), "check email exist failed: %v", err)
			response.AdminFail(c, response.InternalError)
			return
		}
		if existEmailUser != nil && existEmailUser.ID != uID {
			response.AdminFail(c, response.InvalidParams)
			return
		}
	}

	if err := h.userRepo.UpdateProfile(c.Request.Context(), uID, req.Email, req.Phone); err != nil {
		logger.Error(c.Request.Context(), "update profile failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	response.AdminSuccess(c, "资料更新成功")
}

func (h *UserHandler) UpdatePassword(c *gin.Context) {
	var req form.UpdatePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	// 从中间件上下文中提取当前操作人的真实凭证，严防越权
	currentUserID, exists := c.Get("user_id")
	currentUsername := c.GetString("username")
	if !exists {
		response.AdminFail(c, response.Unauthorized)
		return
	}

	// 必须重新捞出数据库中的老密码哈希进行比对
	user, err := h.userRepo.GetIdByUsername(c.Request.Context(), currentUsername)
	if err != nil || user == nil {
		response.AdminFail(c, response.InternalError)
		return
	}

	// 校验旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		logger.Warn(c.Request.Context(), "user %s old password verification failed", currentUsername)
		response.AdminFail(c, response.Unauthorized) // 旧密码不对
		return
	}

	// 加密新密码
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		response.AdminFail(c, response.InternalError)
		return
	}

	// 执行修改（注意进行类型断言转换，通常为 uint 或 int64，依你的 model 而定）
	uID := uint(currentUserID.(float64)) // JWT 解析数字通常为 float64
	if err := h.userRepo.UpdatePassword(c.Request.Context(), uID, string(newHashedPassword)); err != nil {
		logger.Error(c.Request.Context(), "update password failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	response.AdminSuccess(c, "密码修改成功")
}

// generateJWT 创建 JWT Token
func (h *UserHandler) generateJWT(user *model.User) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
		"iat":      now.Unix(),
		"exp":      now.Add(h.jwtCfg.Expire).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtCfg.Secret))
}
