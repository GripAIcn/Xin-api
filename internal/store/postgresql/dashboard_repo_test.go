package postgresql

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"Xin-api/internal/form"
	"Xin-api/internal/model"
)

// dashboardSQLMatcher 依据期望标记校验实际 SQL：
// 费用聚合查询必须关联计费流水与汇率表，并使用结算快照 / 失败不计数 / 历史兜底的表达式。
func dashboardSQLMatcher(expectedSQL, actualSQL string) error {
	var required []string
	switch expectedSQL {
	case "overview":
		required = append(required,
			"LEFT JOIN billing_records b ON b.request_id = l.request_id",
			"LEFT JOIN exchange_rates er ON er.from_currency = l.currency AND er.to_currency = 'CNY'",
			"WHEN b.status = 'failed' THEN 0",
			"NULLIF(b.settle_cost, 0)",
			"l.total_cost * COALESCE(er.rate, 1)",
			"COALESCE(AVG(l.ttft_ms), 0)::float8 AS avg_ttft_ms",
		)
	case "trend":
		required = append(required,
			"LEFT JOIN billing_records b ON b.request_id = l.request_id",
			"NULLIF(b.settle_cost, 0)",
			"GROUP BY to_timestamp(floor(extract(epoch FROM l.created_at)",
		)
	case "models":
		required = append(required,
			"LEFT JOIN billing_records b ON b.request_id = l.request_id",
			"NULLIF(b.settle_cost, 0)",
			"GROUP BY l.model ORDER BY request_count DESC LIMIT 5",
		)
	case "channels":
		required = append(required,
			"LEFT JOIN channels ch ON ch.id = l.channel_id",
			"GROUP BY l.channel_id, ch.name",
		)
	case "errors":
		required = append(required,
			"FROM request_logs WHERE log_type = 'error'",
			"GROUP BY COALESCE(error_type, 'unknown')",
		)
	default:
		return fmt.Errorf("未知的期望标记: %q", expectedSQL)
	}

	for _, fragment := range required {
		if !strings.Contains(actualSQL, fragment) {
			return fmt.Errorf("SQL 缺少片段 %q，实际 SQL: %s", fragment, actualSQL)
		}
	}
	return nil
}

func newDashboardTestRepo(t *testing.T) (DashboardRepo, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New(
		sqlmock.QueryMatcherOption(sqlmock.QueryMatcherFunc(dashboardSQLMatcher)),
	)
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger:               logger.Default.LogMode(logger.Silent),
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("打开 GORM 失败: %v", err)
	}
	return NewDashboardRepo(db), mock
}

func TestDashboardRepoStatsUsesCNYSettlement(t *testing.T) {
	repo, mock := newDashboardTestRepo(t)
	now := time.Now()

	mock.ExpectQuery("overview").WillReturnRows(sqlmock.NewRows([]string{
		"request_count", "success_count", "error_count", "success_rate",
		"input_tokens", "output_tokens", "total_tokens", "total_cost",
		"avg_latency_ms", "avg_ttft_ms",
	}).AddRow(int64(10), int64(9), int64(1), 0.9, int64(100), int64(200), int64(300), "72.50000000", 120.5, 30.2))

	mock.ExpectQuery("trend").WillReturnRows(sqlmock.NewRows([]string{
		"bucket_start", "request_count", "success_count", "error_count",
		"input_tokens", "output_tokens", "total_tokens", "total_cost",
	}).AddRow(now, int64(10), int64(9), int64(1), int64(100), int64(200), int64(300), "72.50000000"))

	mock.ExpectQuery("models").WillReturnRows(sqlmock.NewRows([]string{
		"model", "request_count", "total_tokens", "total_cost",
	}).AddRow("gpt-4o", int64(6), int64(180), "62.25000000"))

	mock.ExpectQuery("channels").WillReturnRows(sqlmock.NewRows([]string{
		"channel_id", "channel_name", "request_count", "success_rate", "avg_latency_ms",
	}).AddRow(int64(3), "主力渠道", int64(8), 0.875, 100.0))

	mock.ExpectQuery("errors").WillReturnRows(sqlmock.NewRows([]string{
		"error_type", "count",
	}).AddRow("timeout", int64(2)))

	stats, err := repo.Stats(context.Background(), form.DashboardQuery{}, nil)
	if err != nil {
		t.Fatalf("Stats 返回错误: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL 期望未满足: %v", err)
	}

	if stats.Range != "today" {
		t.Fatalf("Range = %s, 期望 today", stats.Range)
	}
	if stats.Overview.Currency != model.CurrencyCNY {
		t.Fatalf("币种 = %s, 期望 CNY", stats.Overview.Currency)
	}
	requireMoney(t, stats.Overview.TotalCost, "72.5")
	if len(stats.Trend) != 1 {
		t.Fatalf("趋势点数 = %d, 期望 1", len(stats.Trend))
	}
	requireMoney(t, stats.Trend[0].TotalCost, "72.5")
	if len(stats.TopModels) != 1 {
		t.Fatalf("模型排行数 = %d, 期望 1", len(stats.TopModels))
	}
	requireMoney(t, stats.TopModels[0].TotalCost, "62.25")
	if len(stats.TopChannels) != 1 || stats.TopChannels[0].ChannelName == nil || *stats.TopChannels[0].ChannelName != "主力渠道" {
		t.Fatalf("渠道排行 = %+v, 期望包含主力渠道", stats.TopChannels)
	}
	if len(stats.TopErrors) != 1 || stats.TopErrors[0].ErrorType != "timeout" {
		t.Fatalf("错误排行 = %+v, 期望 timeout", stats.TopErrors)
	}
}

func TestDashboardRepoStatsQueryError(t *testing.T) {
	repo, mock := newDashboardTestRepo(t)

	mock.ExpectQuery("overview").WillReturnRows(sqlmock.NewRows([]string{
		"request_count", "success_count", "error_count", "success_rate",
		"input_tokens", "output_tokens", "total_tokens", "total_cost",
		"avg_latency_ms", "avg_ttft_ms",
	}).AddRow(int64(0), int64(0), int64(0), 0.0, int64(0), int64(0), int64(0), "0", 0.0, 0.0))
	mock.ExpectQuery("trend").WillReturnError(errors.New("查询趋势失败"))

	stats, err := repo.Stats(context.Background(), form.DashboardQuery{Range: "7d"}, nil)
	if err == nil {
		t.Fatal("期望返回错误, 实际为 nil")
	}
	if stats != nil {
		t.Fatalf("stats = %+v, 期望 nil", stats)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL 期望未满足: %v", err)
	}
}

func TestDashboardRepoStatsScopeUser(t *testing.T) {
	repo, mock := newDashboardTestRepo(t)
	userID := int64(42)

	// 用户范围查询应在每个聚合 SQL 中带上 user_id 过滤参数
	mock.ExpectQuery("overview").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), userID).WillReturnRows(sqlmock.NewRows([]string{
		"request_count", "success_count", "error_count", "success_rate",
		"input_tokens", "output_tokens", "total_tokens", "total_cost",
		"avg_latency_ms", "avg_ttft_ms",
	}).AddRow(int64(1), int64(1), int64(0), 1.0, int64(10), int64(20), int64(30), "7.20000000", 88.0, 12.0))
	mock.ExpectQuery("trend").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), userID).WillReturnRows(sqlmock.NewRows([]string{
		"bucket_start", "request_count", "success_count", "error_count",
		"input_tokens", "output_tokens", "total_tokens", "total_cost",
	}))
	mock.ExpectQuery("models").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), userID).WillReturnRows(sqlmock.NewRows([]string{
		"model", "request_count", "total_tokens", "total_cost",
	}))
	// 非管理员不查询渠道排行（渠道属于上游资源，不对普通用户暴露）
	mock.ExpectQuery("errors").WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), userID).WillReturnRows(sqlmock.NewRows([]string{
		"error_type", "count",
	}))

	stats, err := repo.Stats(context.Background(), form.DashboardQuery{Range: "30d"}, &userID)
	if err != nil {
		t.Fatalf("Stats 返回错误: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("SQL 期望未满足: %v", err)
	}
	if stats.Overview.Currency != model.CurrencyCNY {
		t.Fatalf("币种 = %s, 期望 CNY", stats.Overview.Currency)
	}
	requireMoney(t, stats.Overview.TotalCost, "7.2")
	if len(stats.TopChannels) != 0 {
		t.Fatalf("渠道排行 = %+v, 期望普通用户为空", stats.TopChannels)
	}
}
