package form

// DashboardQuery 仪表盘统计查询入参
type DashboardQuery struct {
	// Range 统计范围：today 今日 / 7d 近 7 天 / 30d 近 30 天；缺省 today
	Range string `form:"range" json:"range" validate:"omitempty,oneof=today 7d 30d"`
}
