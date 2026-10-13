package model

import (
	"encoding/json"
	"time"
)

// 审计结果枚举，对应 audit_logs.result
const (
	AuditResultSuccess = "success"
	AuditResultFailure = "failure"
)

// AuditLog 控制面审计与登录日志，对应 audit_logs 表
// 说明：user_id 不建外键，审计独立留存。
type AuditLog struct {
	ID           int64           `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	UserID       *int64          `gorm:"column:user_id;comment:操作者用户 id" json:"user_id,omitempty"`
	UserName     *string         `gorm:"column:user_name;type:varchar(50);comment:操作者账号名快照" json:"user_name,omitempty"`
	Action       string          `gorm:"column:action;type:varchar(64);not null;comment:操作类型 create_key/update_channel/login 等" json:"action"`
	TargetType   *string         `gorm:"column:target_type;type:varchar(64);comment:目标对象类型" json:"target_type,omitempty"`
	TargetID     *string         `gorm:"column:target_id;type:varchar(64);comment:目标对象 id" json:"target_id,omitempty"`
	Before       json.RawMessage `gorm:"column:before;type:jsonb;comment:变更前快照" json:"before,omitempty"`
	After        json.RawMessage `gorm:"column:after;type:jsonb;comment:变更后快照" json:"after,omitempty"`
	Result       string          `gorm:"column:result;type:varchar(32);not null;default:success;comment:结果 success/failure" json:"result"`
	ErrorMessage *string         `gorm:"column:error_message;type:text;comment:失败原因" json:"error_message,omitempty"`
	ClientIP     *string         `gorm:"column:client_ip;type:inet;comment:客户端 IP" json:"client_ip,omitempty"`
	UserAgent    *string         `gorm:"column:user_agent;type:text;comment:客户端 UA" json:"user_agent,omitempty"`
	CreatedAt    time.Time       `gorm:"column:created_at;autoCreateTime" json:"created_at"`
}
