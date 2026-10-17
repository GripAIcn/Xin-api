package postgresql

import (
	"context"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"Xin-api/internal/form"
	"Xin-api/internal/model"
)

// DashboardRepo 仪表盘统计仓储：第一阶段直接对 request_logs 做范围聚合（不做预聚合表）
type DashboardRepo interface {
	// Stats 汇总所选时间范围的概览、趋势与排行；scopeUserID 非空时仅统计该用户
	Stats(ctx context.Context, q form.DashboardQuery, scopeUserID *int64) (*model.DashboardStats, error)
}

type dashboardRepo struct {
	db *gorm.DB
}

// NewDashboardRepo 创建仪表盘统计仓储
func NewDashboardRepo(db *gorm.DB) DashboardRepo {
	return &dashboardRepo{db: db}
}

// dashboardScope 依统计范围返回 [start, end) 与分桶步长（秒）。
// today 以服务器本地时区的当日 00:00 起、按小时分桶；7d / 30d 为滚动窗口、按天分桶。
func dashboardScope(rangeKey string) (time.Time, time.Time, int64) {
	now := time.Now()
	switch rangeKey {
	case "7d":
		return now.AddDate(0, 0, -7), now, 86400
	case "30d":
		return now.AddDate(0, 0, -30), now, 86400
	default: // today
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return start, now, 3600
	}
}

// dashboardWhere 组装数据面流量的过滤条件（仅消费/错误日志，按时间范围与可选用户范围）。
// alias 非空时给列名加表前缀（用于 JOIN 查询）。
func dashboardWhere(alias string, start, end time.Time, scopeUserID *int64) (string, []any) {
	prefix := ""
	if alias != "" {
		prefix = alias + "."
	}
	conds := []string{
		prefix + "log_type IN ('consumption', 'error')",
		prefix + "created_at >= ?",
		prefix + "created_at < ?",
	}
	args := []any{start, end}
	if scopeUserID != nil {
		conds = append(conds, prefix+"user_id = ?")
		args = append(args, *scopeUserID)
	}
	return strings.Join(conds, " AND "), args
}

// Stats 一次性汇总概览 / 趋势 / 模型排行 / 渠道排行 / 错误排行
func (r *dashboardRepo) Stats(ctx context.Context, q form.DashboardQuery, scopeUserID *int64) (*model.DashboardStats, error) {
	rangeKey := q.Range
	if rangeKey == "" {
		rangeKey = "today"
	}
	start, end, bucketStep := dashboardScope(rangeKey)
	where, args := dashboardWhere("", start, end, scopeUserID)

	stats := &model.DashboardStats{
		Range:       rangeKey,
		StartTime:   start,
		EndTime:     end,
		Trend:       []model.DashboardTrendPoint{},
		TopModels:   []model.DashboardModelRank{},
		TopChannels: []model.DashboardChannelRank{},
		TopErrors:   []model.DashboardErrorRank{},
	}

	// 1. 概览（AVG(latency_ms) 自动忽略 NULL 行）
	overviewSQL := `SELECT
		COUNT(*)::bigint AS request_count,
		COUNT(*) FILTER (WHERE status = 'success')::bigint AS success_count,
		COUNT(*) FILTER (WHERE status = 'error')::bigint AS error_count,
		COALESCE(COUNT(*) FILTER (WHERE status = 'success')::numeric / NULLIF(COUNT(*), 0), 0)::float8 AS success_rate,
		COALESCE(SUM(input_tokens), 0)::bigint AS input_tokens,
		COALESCE(SUM(output_tokens), 0)::bigint AS output_tokens,
		COALESCE(SUM(total_tokens), 0)::bigint AS total_tokens,
		COALESCE(SUM(total_cost), 0) AS total_cost,
		COALESCE(MAX(currency), 'USD') AS currency,
		COALESCE(AVG(latency_ms), 0)::float8 AS avg_latency_ms,
		COALESCE(AVG(ttft_ms), 0)::float8 AS avg_ttft_ms
	FROM request_logs WHERE ` + where
	if err := r.db.WithContext(ctx).Raw(overviewSQL, args...).Scan(&stats.Overview).Error; err != nil {
		return nil, err
	}

	// 2. 趋势（基于 epoch 的分桶，与 channel_metrics 的 metricTrendBucket 约定一致）
	step := strconv.FormatInt(bucketStep, 10)
	bucketExpr := "to_timestamp(floor(extract(epoch FROM created_at) / " + step + ") * " + step + ")"
	trendSQL := `SELECT ` + bucketExpr + ` AS bucket_start,
		COUNT(*)::bigint AS request_count,
		COUNT(*) FILTER (WHERE status = 'success')::bigint AS success_count,
		COUNT(*) FILTER (WHERE status = 'error')::bigint AS error_count,
		COALESCE(SUM(input_tokens), 0)::bigint AS input_tokens,
		COALESCE(SUM(output_tokens), 0)::bigint AS output_tokens,
		COALESCE(SUM(total_tokens), 0)::bigint AS total_tokens,
		COALESCE(SUM(total_cost), 0) AS total_cost
	FROM request_logs WHERE ` + where + ` GROUP BY ` + bucketExpr + ` ORDER BY bucket_start ASC`
	if err := r.db.WithContext(ctx).Raw(trendSQL, args...).Scan(&stats.Trend).Error; err != nil {
		return nil, err
	}
	if stats.Trend == nil {
		stats.Trend = []model.DashboardTrendPoint{}
	}

	// 3. 模型调用排行（按请求数取前 5）
	modelSQL := `SELECT model,
		COUNT(*)::bigint AS request_count,
		COALESCE(SUM(total_tokens), 0)::bigint AS total_tokens,
		COALESCE(SUM(total_cost), 0) AS total_cost
	FROM request_logs WHERE ` + where + ` GROUP BY model ORDER BY request_count DESC LIMIT 5`
	if err := r.db.WithContext(ctx).Raw(modelSQL, args...).Scan(&stats.TopModels).Error; err != nil {
		return nil, err
	}
	if stats.TopModels == nil {
		stats.TopModels = []model.DashboardModelRank{}
	}

	// 4. 渠道调用排行（JOIN channels 取渠道名）
	channelWhere, channelArgs := dashboardWhere("l", start, end, scopeUserID)
	channelSQL := `SELECT l.channel_id AS channel_id, ch.name AS channel_name,
		COUNT(*)::bigint AS request_count,
		COALESCE(COUNT(*) FILTER (WHERE l.status = 'success')::numeric / NULLIF(COUNT(*), 0), 0)::float8 AS success_rate,
		COALESCE(AVG(l.latency_ms), 0)::float8 AS avg_latency_ms
	FROM request_logs l
	LEFT JOIN channels ch ON ch.id = l.channel_id
	WHERE l.channel_id IS NOT NULL AND ` + channelWhere + `
	GROUP BY l.channel_id, ch.name ORDER BY request_count DESC LIMIT 5`
	if err := r.db.WithContext(ctx).Raw(channelSQL, channelArgs...).Scan(&stats.TopChannels).Error; err != nil {
		return nil, err
	}
	if stats.TopChannels == nil {
		stats.TopChannels = []model.DashboardChannelRank{}
	}

	// 5. 错误排行（仅错误日志，按 error_type 归类）
	errorConds := []string{"log_type = 'error'", "created_at >= ?", "created_at < ?"}
	errorArgs := []any{start, end}
	if scopeUserID != nil {
		errorConds = append(errorConds, "user_id = ?")
		errorArgs = append(errorArgs, *scopeUserID)
	}
	errorSQL := `SELECT COALESCE(error_type, 'unknown') AS error_type,
		COUNT(*)::bigint AS count
	FROM request_logs WHERE ` + strings.Join(errorConds, " AND ") + `
	GROUP BY COALESCE(error_type, 'unknown') ORDER BY count DESC LIMIT 5`
	if err := r.db.WithContext(ctx).Raw(errorSQL, errorArgs...).Scan(&stats.TopErrors).Error; err != nil {
		return nil, err
	}
	if stats.TopErrors == nil {
		stats.TopErrors = []model.DashboardErrorRank{}
	}

	return stats, nil
}
