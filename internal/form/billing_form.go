package form

// BillingLogQuery 账单日志列表查询入参
type BillingLogQuery struct {
	Page      int     `form:"page" json:"page" validate:"omitempty,gte=1"`
	PageSize  int     `form:"page_size" json:"page_size" validate:"omitempty,gte=1,lte=100"`
	Source    string  `form:"source" json:"source" validate:"omitempty,oneof=consumption recharge refund"`
	UserID    *int64  `form:"user_id" json:"user_id" validate:"omitempty,gte=0"`
	Model     string  `form:"model" json:"model"`
	RequestID string  `form:"request_id" json:"request_id"`
	StartTime *string `form:"start_time" json:"start_time"`
	EndTime   *string `form:"end_time" json:"end_time"`
}
