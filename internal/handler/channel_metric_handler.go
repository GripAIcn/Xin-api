package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	"Xin-api/internal/form"
	"Xin-api/internal/service"
	"Xin-api/internal/store/postgresql"
	"Xin-api/pkg/response"
)

// ChannelMetricHandler 渠道指标查询与聚合处理器（仅管理员）
type ChannelMetricHandler struct {
	repo       postgresql.ChannelMetricRepo
	aggregator *service.ChannelMetricAggregator
	validate   *validator.Validate
}

// NewChannelMetricHandler 构造函数注入依赖
func NewChannelMetricHandler(repo postgresql.ChannelMetricRepo, aggregator *service.ChannelMetricAggregator) *ChannelMetricHandler {
	return &ChannelMetricHandler{
		repo:       repo,
		aggregator: aggregator,
		validate:   validator.New(),
	}
}

// List 按维度查询指标汇总
func (h *ChannelMetricHandler) List(c *gin.Context) {
	var q form.ChannelMetricQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}
	if err := h.validate.Struct(q); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	rows, err := h.repo.Summarize(c.Request.Context(), q)
	if err != nil {
		response.AdminFail(c, response.InternalError)
		return
	}
	response.AdminSuccess(c, rows)
}

// Trend 查询 TTFT / 延迟时间走势
func (h *ChannelMetricHandler) Trend(c *gin.Context) {
	var q form.ChannelMetricTrendQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}
	if err := h.validate.Struct(q); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	rows, err := h.repo.Trend(c.Request.Context(), q)
	if err != nil {
		response.AdminFail(c, response.InternalError)
		return
	}
	response.AdminSuccess(c, rows)
}

// aggregateResult 手动聚合返回体
type aggregateResult struct {
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Rows       int64     `json:"rows"`
	DurationMs int64     `json:"duration_ms"`
}

// Aggregate 按传入时间区间同步重算指标
func (h *ChannelMetricHandler) Aggregate(c *gin.Context) {
	var req form.ChannelMetricAggregateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	start, ok := parseMetricTime(req.StartTime)
	if !ok {
		response.AdminFail(c, response.InvalidParams)
		return
	}
	end, ok := parseMetricTime(req.EndTime)
	if !ok || !start.Before(end) {
		response.AdminFail(c, response.InvalidParams)
		return
	}

	begin := time.Now()
	rows, err := h.aggregator.AggregateRange(c.Request.Context(), start, end)
	if err != nil {
		response.AdminFail(c, response.InternalError)
		return
	}
	response.AdminSuccess(c, aggregateResult{
		StartTime:  start,
		EndTime:    end,
		Rows:       rows,
		DurationMs: time.Since(begin).Milliseconds(),
	})
}

// parseMetricTime 依次尝试多种时间布局解析入参时间
func parseMetricTime(s string) (time.Time, bool) {
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
