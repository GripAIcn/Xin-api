package form

import "time"

// CreateGroupReq 创建项目组请求入参
type CreateGroupReq struct {
	Name string `json:"name" binding:"required" validate:"required,min=2,max=100"` // 项目组名称必填，长度 2-100
}

// UpdateGroupReq 更新项目组名称入参
type UpdateGroupReq struct {
	Name string `json:"name" binding:"required" validate:"required,min=2,max=100"`
}

// UpdateGroupStatusReq 启停项目组状态入参
type UpdateGroupStatusReq struct {
	// ⚡️ 生产细节：int 类型直接写 required，当传 0 时会被判定为空导致校验失败。
	// 针对可能为 0 的状态值，binding 需使用 "oneof=0 1"（只能是 0 或 1）
	Status int `json:"status" binding:"oneof=0 1" validate:"oneof=0 1"`
}

// CreateApiKeyReq 创建凭证请求入参
type CreateApiKeyReq struct {
	GroupID int64 `json:"group_id" binding:"required" validate:"required,gt=0"` // 所属项目组ID，必须大于0
}

// ApiKeyResponse 返回给前端的凭证详情（隐藏不需要展示的 GORM 内部元数据）
type ApiKeyResponse struct {
	Key       string    `json:"key"`
	GroupID   int64     `json:"group_id"`
	CreatedAt time.Time `json:"created_at"`
}
