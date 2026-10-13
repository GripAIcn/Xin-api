package model

import "time"

// RequestPayload 请求 / 响应原文（默认关闭），对应 request_payloads 表
// 说明：与 request_logs 1:1，created_at 取父行同值（无 autoCreateTime）。
type RequestPayload struct {
	ID           int64     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	RequestID    string    `gorm:"column:request_id;type:varchar(64);not null;uniqueIndex:uq_request_payloads_request,priority:1;comment:父行 request_logs.request_id" json:"request_id"`
	CreatedAt    time.Time `gorm:"column:created_at;not null;uniqueIndex:uq_request_payloads_request,priority:2;comment:取父行 request_logs.created_at 同值" json:"created_at"`
	RequestBody  *string   `gorm:"column:request_body;type:text;comment:请求原文" json:"request_body,omitempty"`
	ResponseBody *string   `gorm:"column:response_body;type:text;comment:响应原文" json:"response_body,omitempty"`
}
