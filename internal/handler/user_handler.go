package handler

import (
	"time"

	"github.com/bytedance/gopkg/util/logger"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"Xin-api/config"
	"Xin-api/internal/form"
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
	})
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var req form.UserLoginReq
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
	}

	if err := h.userRepo.Create(c.Request.Context(), newUser); err != nil {
		logger.Error(c.Request.Context(), "create user failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	response.AdminSuccess(c, "账户创建成功")
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

func (h *UserHandler) UpdateAccount(c *gin.Context) {
	var req form.UpdateAccountReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	currentUserID, exists := c.Get("user_id")
	if !exists {
		response.AdminFail(c, response.Unauthorized)
		return
	}

	// 检查新改的用户名是否被别人占用了
	existUser, err := h.userRepo.GetIdByUsername(c.Request.Context(), req.Username)
	if err != nil {
		response.AdminFail(c, response.InternalError)
		return
	}

	uID := uint(currentUserID.(float64))
	if existUser != nil && existUser.ID != uID {
		response.AdminFail(c, response.InvalidParams) // 用户名已被占
		return
	}

	// 更新账户名
	if err := h.userRepo.UpdateAccount(c.Request.Context(), uID, req.Username); err != nil {
		logger.Error(c.Request.Context(), "update account failed: %v", err)
		response.AdminFail(c, response.InternalError)
		return
	}

	response.AdminSuccess(c, "账号信息更新成功")
}

// generateJWT 创建 JWT Token
func (h *UserHandler) generateJWT(user *model.User) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"iat":      now.Unix(),
		"exp":      now.Add(h.jwtCfg.Expire).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtCfg.Secret))
}
