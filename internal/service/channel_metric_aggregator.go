package service

import (
	"context"
	"log"
	"time"

	"Xin-api/internal/store/postgresql"
)

const (
	// metricBackfillWindow 启动回填范围：最近 7 天
	metricBackfillWindow = 7 * 24 * time.Hour
	// metricBackfillShard 回填分片：按 6 小时顺序执行，避免单条 SQL 扫描过大
	metricBackfillShard = 6 * time.Hour
	// metricIncrementalWindow 周期重算范围：最近 2 分钟（覆盖迟到日志）
	metricIncrementalWindow = 2 * time.Minute
	// metricAggregateInterval 聚合触发周期
	metricAggregateInterval = time.Minute
)

// ChannelMetricAggregator 渠道指标聚合服务：启动回填 + 周期增量
type ChannelMetricAggregator struct {
	repo postgresql.ChannelMetricRepo
}

// NewChannelMetricAggregator 构造函数注入依赖
func NewChannelMetricAggregator(repo postgresql.ChannelMetricRepo) *ChannelMetricAggregator {
	return &ChannelMetricAggregator{repo: repo}
}

// Run 启动聚合协程：先回填最近 7 天，随后每 60 秒重算最近 2 分钟；ctx 取消后退出
func (a *ChannelMetricAggregator) Run(ctx context.Context) {
	a.Backfill(ctx, metricBackfillWindow)

	ticker := time.NewTicker(metricAggregateInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			end := time.Now().Truncate(time.Minute)
			if _, err := a.repo.AggregateRange(ctx, end.Add(-metricIncrementalWindow), end); err != nil {
				log.Printf("[METRICS] 增量聚合失败: %v", err)
			}
		}
	}
}

// Backfill 按 6 小时分片回填指定时间窗口；单分片失败仅记录日志并继续
func (a *ChannelMetricAggregator) Backfill(ctx context.Context, window time.Duration) {
	end := time.Now().Truncate(time.Minute)
	start := end.Add(-window)
	log.Printf("[METRICS] 开始回填渠道指标: %s ~ %s", start.Format(time.RFC3339), end.Format(time.RFC3339))

	for s := start; s.Before(end); s = s.Add(metricBackfillShard) {
		e := s.Add(metricBackfillShard)
		if e.After(end) {
			e = end
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
		rows, err := a.repo.AggregateRange(ctx, s, e)
		if err != nil {
			log.Printf("[METRICS] 回填分片失败 %s ~ %s: %v", s.Format(time.RFC3339), e.Format(time.RFC3339), err)
			continue
		}
		log.Printf("[METRICS] 回填分片完成 %s ~ %s, rows=%d", s.Format(time.RFC3339), e.Format(time.RFC3339), rows)
	}
	log.Println("[METRICS] 渠道指标回填结束")
}

// AggregateRange 手动聚合指定区间（供管理接口补数使用）
func (a *ChannelMetricAggregator) AggregateRange(ctx context.Context, start, end time.Time) (int64, error) {
	return a.repo.AggregateRange(ctx, start, end)
}
