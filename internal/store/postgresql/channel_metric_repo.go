package postgresql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"Xin-api/internal/form"
	"Xin-api/internal/model"
)

// ChannelMetricRepo 渠道指标仓储：负责 channel_metrics 的聚合写入与读取
type ChannelMetricRepo interface {
	// AggregateRange 将 [start, end) 内的 request_logs 按 渠道 × 模型 × 60 秒窗口 聚合写入 channel_metrics
	AggregateRange(ctx context.Context, start, end time.Time) (int64, error)
	// Summarize 在 channel_metrics 上按维度做跨窗口二次聚合
	Summarize(ctx context.Context, q form.ChannelMetricQuery) ([]model.ChannelMetricSummary, error)
	// Trend 按时间分桶返回 TTFT / 延迟走势
	Trend(ctx context.Context, q form.ChannelMetricTrendQuery) ([]model.ChannelMetricTrendPoint, error)
}

type channelMetricRepo struct {
	db *gorm.DB
}

// NewChannelMetricRepo 创建渠道指标仓储
func NewChannelMetricRepo(db *gorm.DB) ChannelMetricRepo {
	return &channelMetricRepo{db: db}
}

// aggregateRangeSQL 从 request_logs 按 60 秒窗口聚合，幂等覆盖写入 channel_metrics。
// 只写非生成列；error_count / timeout_count 拆分保证 CHECK (request_count = success + error + timeout)。
const aggregateRangeSQL = `INSERT INTO channel_metrics (
	channel_id, model, window_start, window_size,
	request_count, success_count, error_count, timeout_count,
	total_input_tokens, total_output_tokens,
	sum_latency_ms, sum_ttft_ms, ttft_count, sum_generation_ms, generation_token_count,
	total_cost, currency, updated_at
)
SELECT
	l.channel_id,
	l.model,
	l.window_start,
	60,
	COUNT(*),
	COUNT(*) FILTER (WHERE l.status = 'success'),
	COUNT(*) FILTER (WHERE l.status <> 'success' AND COALESCE(l.error_type, '') <> 'timeout'),
	COUNT(*) FILTER (WHERE l.status <> 'success' AND l.error_type = 'timeout'),
	COALESCE(SUM(l.input_tokens), 0),
	COALESCE(SUM(l.output_tokens), 0),
	COALESCE(SUM(l.latency_ms) FILTER (WHERE l.status = 'success'), 0),
	COALESCE(SUM(l.ttft_ms) FILTER (WHERE l.status = 'success' AND l.ttft_ms IS NOT NULL), 0),
	COUNT(*) FILTER (WHERE l.status = 'success' AND l.ttft_ms IS NOT NULL),
	COALESCE(SUM(
		COALESCE(l.generation_ms, GREATEST(COALESCE(l.latency_ms, 0) - COALESCE(l.ttft_ms, 0), 0))
	) FILTER (WHERE l.status = 'success' AND l.ttft_ms IS NOT NULL AND COALESCE(l.output_tokens, 0) > 1), 0),
	-- TPOT 分母：仅流式成功且 N>1 的样本，每个样本按 N-1 个 token 间隔累计
	COALESCE(SUM(
		GREATEST(COALESCE(l.output_tokens, 0) - 1, 0)
	) FILTER (WHERE l.status = 'success' AND l.ttft_ms IS NOT NULL AND COALESCE(l.output_tokens, 0) > 1), 0),
	COALESCE(SUM(l.total_cost), 0),
	COALESCE(MAX(l.currency), 'USD'),
	NOW()
FROM (
	SELECT
		channel_id,
		model,
		to_timestamp(floor(extract(epoch FROM created_at) / 60) * 60) AS window_start,
		status,
		error_type,
		input_tokens,
		output_tokens,
		latency_ms,
		ttft_ms,
		generation_ms,
		total_cost,
		currency
	FROM request_logs
	WHERE channel_id IS NOT NULL
	  AND log_type IN ('consumption', 'error')
	  AND created_at >= ?
	  AND created_at < ?
) l
GROUP BY l.channel_id, l.model, l.window_start
ON CONFLICT (channel_id, model, window_start) DO UPDATE SET
	window_size = EXCLUDED.window_size,
	request_count = EXCLUDED.request_count,
	success_count = EXCLUDED.success_count,
	error_count = EXCLUDED.error_count,
	timeout_count = EXCLUDED.timeout_count,
	total_input_tokens = EXCLUDED.total_input_tokens,
	total_output_tokens = EXCLUDED.total_output_tokens,
	sum_latency_ms = EXCLUDED.sum_latency_ms,
	sum_ttft_ms = EXCLUDED.sum_ttft_ms,
	ttft_count = EXCLUDED.ttft_count,
	sum_generation_ms = EXCLUDED.sum_generation_ms,
	generation_token_count = EXCLUDED.generation_token_count,
	total_cost = EXCLUDED.total_cost,
	currency = EXCLUDED.currency,
	updated_at = NOW()`

func (r *channelMetricRepo) AggregateRange(ctx context.Context, start, end time.Time) (int64, error) {
	// 对齐到分钟边界，避免分片边界切裂同一窗口导致后续分片覆盖丢数据
	start = start.UTC().Truncate(time.Minute)
	end = end.UTC().Truncate(time.Minute)
	if !start.Before(end) {
		return 0, nil
	}
	res := r.db.WithContext(ctx).Exec(aggregateRangeSQL, start, end)
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

func (r *channelMetricRepo) Summarize(ctx context.Context, q form.ChannelMetricQuery) ([]model.ChannelMetricSummary, error) {
	dimension := q.Dimension
	if dimension == "" {
		dimension = "channel_model"
	}
	start, end := metricRange(q.Range)

	// 维度选择：决定分行键与渠道元信息列（模型维度不关联渠道分组，避免被打散）
	var groupBy string
	var selectDim string
	channelSelect := "NULL::varchar AS channel_name, NULL::varchar AS channel_provider, NULL::varchar AS channel_status"
	var groupChannel string
	switch dimension {
	case "channel":
		selectDim = "cm.channel_id AS channel_id, NULL::varchar AS model"
		groupBy = "cm.channel_id"
		groupChannel = ", ch.name, ch.provider, ch.status"
		channelSelect = "ch.name AS channel_name, ch.provider AS channel_provider, ch.status AS channel_status"
	case "model":
		selectDim = "NULL::bigint AS channel_id, cm.model AS model"
		groupBy = "cm.model"
	default:
		selectDim = "cm.channel_id AS channel_id, cm.model AS model"
		groupBy = "cm.channel_id, cm.model"
		groupChannel = ", ch.name, ch.provider, ch.status"
		channelSelect = "ch.name AS channel_name, ch.provider AS channel_provider, ch.status AS channel_status"
	}

	conds := []string{"cm.window_start >= ?", "cm.window_start < ?"}
	args := []any{start, end}
	if q.ChannelID != nil {
		conds = append(conds, "cm.channel_id = ?")
		args = append(args, *q.ChannelID)
	}
	if q.Model != "" {
		conds = append(conds, "cm.model = ?")
		args = append(args, q.Model)
	}

	sql := `SELECT ` + selectDim + `,
		COALESCE(SUM(cm.request_count), 0)::bigint AS request_count,
		COALESCE(SUM(cm.success_count), 0)::bigint AS success_count,
		COALESCE(SUM(cm.error_count), 0)::bigint AS error_count,
		COALESCE(SUM(cm.timeout_count), 0)::bigint AS timeout_count,
		COALESCE(SUM(cm.success_count)::numeric / NULLIF(SUM(cm.request_count), 0), 0) AS success_rate,
		COALESCE(SUM(cm.sum_latency_ms)::numeric / NULLIF(SUM(cm.success_count), 0), 0) AS avg_latency_ms,
		COALESCE(SUM(cm.sum_ttft_ms)::numeric / NULLIF(SUM(cm.ttft_count), 0), 0) AS avg_ttft_ms,
		COALESCE(SUM(cm.sum_generation_ms)::numeric / NULLIF(SUM(cm.generation_token_count), 0), 0) AS tpot,
		COALESCE(SUM(cm.request_count)::numeric * 60 / NULLIF(SUM(cm.window_size), 0), 0) AS rpm,
		COALESCE(SUM(cm.total_tokens)::numeric * 60 / NULLIF(SUM(cm.window_size), 0), 0) AS tpm,
		COALESCE(SUM(cm.total_input_tokens), 0)::bigint AS total_input_tokens,
		COALESCE(SUM(cm.total_output_tokens), 0)::bigint AS total_output_tokens,
		COALESCE(SUM(cm.total_tokens), 0)::bigint AS total_tokens,
		COALESCE(SUM(cm.total_cost), 0) AS total_cost,
		COALESCE(MAX(cm.currency), 'USD') AS currency,
		MAX(cm.window_start) AS last_window_start,
		` + channelSelect + `
	FROM channel_metrics cm
	LEFT JOIN channels ch ON ch.id = cm.channel_id
	WHERE ` + strings.Join(conds, " AND ") + `
	GROUP BY ` + groupBy + groupChannel + `
	ORDER BY SUM(cm.request_count) DESC`

	var rows []model.ChannelMetricSummary
	if err := r.db.WithContext(ctx).Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []model.ChannelMetricSummary{}
	}
	return rows, nil
}

func (r *channelMetricRepo) Trend(ctx context.Context, q form.ChannelMetricTrendQuery) ([]model.ChannelMetricTrendPoint, error) {
	start, end := metricRange(q.Range)
	bucket := metricTrendBucket(q.Range)

	conds := []string{"cm.window_start >= ?", "cm.window_start < ?"}
	args := []any{start, end}
	if q.ChannelID != nil {
		conds = append(conds, "cm.channel_id = ?")
		args = append(args, *q.ChannelID)
	}
	if q.Model != "" {
		conds = append(conds, "cm.model = ?")
		args = append(args, q.Model)
	}

	// 桶宽来自受控白名单，直接内联避免 GROUP BY 重复占位符错位
	bucketExpr := fmt.Sprintf("to_timestamp(floor(extract(epoch FROM cm.window_start) / %d) * %d)", bucket, bucket)
	sql := `SELECT ` + bucketExpr + ` AS bucket_start,
		COALESCE(SUM(cm.request_count), 0)::bigint AS request_count,
		COALESCE(SUM(cm.success_count), 0)::bigint AS success_count,
		COALESCE(SUM(cm.success_count)::numeric / NULLIF(SUM(cm.request_count), 0), 0) AS success_rate,
		COALESCE(SUM(cm.sum_ttft_ms)::numeric / NULLIF(SUM(cm.ttft_count), 0), 0) AS avg_ttft_ms,
		COALESCE(SUM(cm.sum_latency_ms)::numeric / NULLIF(SUM(cm.success_count), 0), 0) AS avg_latency_ms,
		COALESCE(SUM(cm.sum_generation_ms)::numeric / NULLIF(SUM(cm.generation_token_count), 0), 0) AS tpot
	FROM channel_metrics cm
	WHERE ` + strings.Join(conds, " AND ") + `
	GROUP BY ` + bucketExpr + `
	ORDER BY bucket_start ASC`

	var rows []model.ChannelMetricTrendPoint
	if err := r.db.WithContext(ctx).Raw(sql, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []model.ChannelMetricTrendPoint{}
	}
	return rows, nil
}

// metricRange 将 range 参数换算为查询区间 [now-range, now)
func metricRange(r string) (time.Time, time.Time) {
	end := time.Now()
	return end.Add(-metricRangeWindow(r)), end
}

func metricRangeWindow(r string) time.Duration {
	switch r {
	case "6h":
		return 6 * time.Hour
	case "24h":
		return 24 * time.Hour
	case "7d":
		return 7 * 24 * time.Hour
	default:
		return time.Hour
	}
}

// metricTrendBucket 趋势分桶宽度（秒）：1h→60、6h→300、24h→900、7d→3600
func metricTrendBucket(r string) int {
	switch r {
	case "6h":
		return 300
	case "24h":
		return 900
	case "7d":
		return 3600
	default:
		return 60
	}
}
