package form

// LogListQuery 日志列表查询入参
type LogListQuery struct {
	Page      int     `form:"page" json:"page" validate:"omitempty,gte=1"`
	PageSize  int     `form:"page_size" json:"page_size" validate:"omitempty,gte=1,lte=100"`
	LogType   string  `form:"log_type" json:"log_type" validate:"omitempty,oneof=consumption error system management login refund recharge"`
	Model     string  `form:"model" json:"model"`
	ChannelID *int64  `form:"channel_id" json:"channel_id" validate:"omitempty,gte=0"`
	UserID    *int64  `form:"user_id" json:"user_id" validate:"omitempty,gte=0"`
	APIKeyID  *int64  `form:"api_key_id" json:"api_key_id" validate:"omitempty,gte=0"`
	RequestID string  `form:"request_id" json:"request_id"`
	Status    string  `form:"status" json:"status" validate:"omitempty,oneof=success error"`
	StartTime *string `form:"start_time" json:"start_time"`
	EndTime   *string `form:"end_time" json:"end_time"`
}
