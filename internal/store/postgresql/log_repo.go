package postgresql

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"Xin-api/internal/form"
	"Xin-api/internal/model"
	"Xin-api/pkg/response"
)

type LogRepo interface {
	// Create 写入一条请求日志（用于登录/充值等仅日志场景）
	Create(ctx context.Context, log *model.RequestLog) error
	// CreateWithBilling 同一事务内写入请求日志并扣减用户余额、落计费流水
	CreateWithBilling(ctx context.Context, log *model.RequestLog, billing *model.BillingRecord) error
	// CreatePayload 写入请求/响应原文（需先写入对应 request_logs 行）
	CreatePayload(ctx context.Context, payload *model.RequestPayload) error
	// CreateAudit 写入一条控制面审计日志
	CreateAudit(ctx context.Context, audit *model.AuditLog) error
	// List 分页查询日志列表；scopeUserID 非空时强制限定该用户
	List(ctx context.Context, q form.LogListQuery, scopeUserID *int64) (*response.PageData[model.LogListItem], error)
	// ListBilling 分页查询账单流水（消费 billing_records + 充值/退款 request_logs 合并）；scopeUserID 非空时强制限定该用户
	ListBilling(ctx context.Context, q form.BillingLogQuery, scopeUserID *int64) (*response.PageData[model.BillingLogItem], error)
	// GetByID 查询日志详情（含渠道名/用户名与计费明细）；未找到返回 (nil, nil)
	GetByID(ctx context.Context, id int64) (*model.LogDetail, error)
	// ResolvePrice 按「渠道 × 对外模型名」解析计价单价快照；未命中返回 (nil, nil)
	ResolvePrice(ctx context.Context, channelID int64, externalModel string) (*model.PriceSnapshot, error)
}

type logRepo struct {
	db *gorm.DB
}

// NewLogRepo 创建日志仓储
func NewLogRepo(db *gorm.DB) LogRepo {
	return &logRepo{db: db}
}

func (r *logRepo) Create(ctx context.Context, log *model.RequestLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *logRepo) CreatePayload(ctx context.Context, payload *model.RequestPayload) error {
	return r.db.WithContext(ctx).Create(payload).Error
}

func (r *logRepo) CreateAudit(ctx context.Context, audit *model.AuditLog) error {
	return r.db.WithContext(ctx).Create(audit).Error
}

// CreateWithBilling 同一事务内：写日志 → 锁定用户余额 → 条件扣减（防负）→ 写计费流水。
// 说明：billing_records.request_id 有 UNIQUE 约束，重复写入会报错（正常防护）；
// 不显式写入 total_tokens（数据库生成列，只读）。
func (r *logRepo) CreateWithBilling(ctx context.Context, log *model.RequestLog, billing *model.BillingRecord) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(log).Error; err != nil {
			return err
		}

		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", billing.UserID).First(&user).Error; err != nil {
			return err
		}

		before := user.Balance
		after := model.Money{Decimal: before.Decimal.Sub(billing.TotalCost.Decimal)}
		billing.BalanceBefore = &before
		billing.BalanceAfter = &after

		// 条件更新防止扣成负数；余额不足（RowsAffected == 0）时安全兜底，不返回错误
		res := tx.Model(&model.User{}).
			Where("id = ? AND balance >= ?", billing.UserID, billing.TotalCost).
			Update("balance", gorm.Expr("balance - ?", billing.TotalCost))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			billing.BalanceAfter = &before
			billing.Status = model.BillingStatusFailed
		}

		return tx.Create(billing).Error
	})
}

// List 分页查询日志列表
func (r *logRepo) List(ctx context.Context, q form.LogListQuery, scopeUserID *int64) (*response.PageData[model.LogListItem], error) {
	page := q.Page
	if page < 1 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// 不过滤 deleted_at，保证软删父表也不丢失日志行
	base := r.db.WithContext(ctx).
		Model(&model.RequestLog{}).
		Joins("LEFT JOIN channels ON channels.id = request_logs.channel_id").
		Joins("LEFT JOIN users ON users.id = request_logs.user_id")

	// 用户范围：scopeUserID 优先于入参 user_id
	if scopeUserID != nil {
		base = base.Where("request_logs.user_id = ?", *scopeUserID)
	} else if q.UserID != nil {
		base = base.Where("request_logs.user_id = ?", *q.UserID)
	}
	if q.LogType != "" {
		base = base.Where("request_logs.log_type = ?", q.LogType)
	}
	if q.Model != "" {
		base = base.Where("request_logs.model = ?", q.Model)
	}
	if q.ChannelID != nil {
		base = base.Where("request_logs.channel_id = ?", *q.ChannelID)
	}
	if q.APIKeyID != nil {
		base = base.Where("request_logs.api_key_id = ?", *q.APIKeyID)
	}
	if q.RequestID != "" {
		base = base.Where("request_logs.request_id = ?", q.RequestID)
	}
	if q.UpstreamReqID != "" {
		base = base.Where("request_logs.upstream_req_id = ?", q.UpstreamReqID)
	}
	if q.Provider != "" {
		base = base.Where("request_logs.provider = ?", q.Provider)
	}
	if q.Protocol != "" {
		base = base.Where("request_logs.protocol = ?", q.Protocol)
	}
	if q.IsStream != nil {
		base = base.Where("request_logs.is_stream = ?", *q.IsStream)
	}
	if q.Status != "" {
		base = base.Where("request_logs.status = ?", q.Status)
	}
	if start := parseTimeParam(q.StartTime); start != nil {
		base = base.Where("request_logs.created_at >= ?", *start)
	}
	if end := parseTimeParam(q.EndTime); end != nil {
		base = base.Where("request_logs.created_at <= ?", *end)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, err
	}

	var rows []model.LogListItem
	err := base.
		Select("request_logs.id AS id, request_logs.request_id AS request_id, " +
			"request_logs.upstream_req_id AS upstream_req_id, request_logs.log_type AS log_type, " +
			"request_logs.status AS status, request_logs.model AS model, request_logs.channel_id AS channel_id, " +
			"channels.name AS channel_name, request_logs.api_key_id AS api_key_id, request_logs.user_id AS user_id, " +
			"users.username AS user_name, request_logs.input_tokens AS input_tokens, " +
			"request_logs.output_tokens AS output_tokens, request_logs.total_tokens AS total_tokens, " +
			"request_logs.total_cost AS total_cost, request_logs.currency AS currency, " +
			"request_logs.latency_ms AS latency_ms, request_logs.created_at AS created_at").
		Order("request_logs.created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []model.LogListItem{}
	}

	return &response.PageData[model.LogListItem]{
		List:     rows,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// ListBilling 分页查询账单流水：将「消费」（billing_records）与「充值/退款」（request_logs）
// 在 SQL 层用 UNION ALL 合并为统一流水行，外层按 created_at 倒序分页；COUNT 对同一子查询。
// 全程只读，不改任何表结构（total_tokens 为生成列，仅 SELECT 读取）。
func (r *logRepo) ListBilling(ctx context.Context, q form.BillingLogQuery, scopeUserID *int64) (*response.PageData[model.BillingLogItem], error) {
	page := q.Page
	if page < 1 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// 用户范围：scopeUserID（普通用户强制本人）优先于入参 user_id
	scope := scopeUserID
	if scope == nil {
		scope = q.UserID
	}
	start := parseTimeParam(q.StartTime)
	end := parseTimeParam(q.EndTime)

	// 按 source 决定参与合并的分支：consumption 仅消费；recharge/refund 仅入账；缺省两分支并存
	includeConsumption := q.Source != model.BillingSourceRecharge && q.Source != model.BillingSourceRefund
	includeEntry := q.Source != model.BillingSourceConsumption

	selects := make([]string, 0, 2)
	args := make([]any, 0, 12)

	if includeConsumption {
		conds := make([]string, 0, 5)
		if scope != nil {
			conds = append(conds, "b.user_id = ?")
			args = append(args, *scope)
		}
		if q.Model != "" {
			conds = append(conds, "b.model = ?")
			args = append(args, q.Model)
		}
		if q.RequestID != "" {
			conds = append(conds, "b.request_id = ?")
			args = append(args, q.RequestID)
		}
		if start != nil {
			conds = append(conds, "b.created_at >= ?")
			args = append(args, *start)
		}
		if end != nil {
			conds = append(conds, "b.created_at <= ?")
			args = append(args, *end)
		}
		where := ""
		if len(conds) > 0 {
			where = " WHERE " + strings.Join(conds, " AND ")
		}
		selects = append(selects, billingConsumptionSelect+where)
	}

	if includeEntry {
		// 入账分支固定只取充值/退款；source 精确时收窄到单类型
		conds := make([]string, 0, 6)
		switch q.Source {
		case model.BillingSourceRecharge:
			conds = append(conds, "r.log_type = 'recharge'")
		case model.BillingSourceRefund:
			conds = append(conds, "r.log_type = 'refund'")
		default:
			conds = append(conds, "r.log_type IN ('recharge', 'refund')")
		}
		if scope != nil {
			conds = append(conds, "r.user_id = ?")
			args = append(args, *scope)
		}
		if q.Model != "" {
			conds = append(conds, "r.model = ?")
			args = append(args, q.Model)
		}
		if q.RequestID != "" {
			conds = append(conds, "r.request_id = ?")
			args = append(args, q.RequestID)
		}
		if start != nil {
			conds = append(conds, "r.created_at >= ?")
			args = append(args, *start)
		}
		if end != nil {
			conds = append(conds, "r.created_at <= ?")
			args = append(args, *end)
		}
		selects = append(selects, billingEntrySelect+" WHERE "+strings.Join(conds, " AND "))
	}

	union := strings.Join(selects, " UNION ALL ")

	var total int64
	if err := r.db.WithContext(ctx).
		Raw("SELECT COUNT(*) FROM ("+union+") t", args...).
		Scan(&total).Error; err != nil {
		return nil, err
	}

	listArgs := make([]any, 0, len(args)+2)
	listArgs = append(listArgs, args...)
	listArgs = append(listArgs, pageSize, (page-1)*pageSize)

	var rows []model.BillingLogItem
	if err := r.db.WithContext(ctx).
		Raw("SELECT * FROM ("+union+") t ORDER BY created_at DESC LIMIT ? OFFSET ?", listArgs...).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []model.BillingLogItem{}
	}

	return &response.PageData[model.BillingLogItem]{
		List:     rows,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// GetByID 查询日志详情
func (r *logRepo) GetByID(ctx context.Context, id int64) (*model.LogDetail, error) {
	var log model.RequestLog
	if err := r.db.WithContext(ctx).First(&log, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	detail := &model.LogDetail{
		LogListItem: model.LogListItem{
			ID:            log.ID,
			RequestID:     log.RequestID,
			UpstreamReqID: log.UpstreamReqID,
			LogType:       log.LogType,
			Status:        log.Status,
			Model:         log.Model,
			ChannelID:     log.ChannelID,
			APIKeyID:      log.APIKeyID,
			UserID:        log.UserID,
			InputTokens:   log.InputTokens,
			OutputTokens:  log.OutputTokens,
			TotalTokens:   log.TotalTokens,
			TotalCost:     log.TotalCost,
			Currency:      log.Currency,
			LatencyMs:     log.LatencyMs,
			CreatedAt:     log.CreatedAt,
		},
		UpstreamModel:       log.UpstreamModel,
		Provider:            log.Provider,
		Protocol:            log.Protocol,
		StatusCode:          log.StatusCode,
		FinishReason:        log.FinishReason,
		IsStream:            log.IsStream,
		IsCancelled:         log.IsCancelled,
		CancelledBy:         log.CancelledBy,
		RetryCount:          log.RetryCount,
		RetryChannels:       []int64(log.RetryChannels),
		UsageSource:         log.UsageSource,
		RouteStrategy:       log.RouteStrategy,
		RouteWeight:         log.RouteWeight,
		CachedInputTokens:   log.CachedInputTokens,
		CacheCreationTokens: log.CacheCreationTokens,
		ReasoningTokens:     log.ReasoningTokens,
		TTFTMs:              log.TTFTMs,
		GenerationMs:        log.GenerationMs,
		ErrorStage:          log.ErrorStage,
		ErrorType:           log.ErrorType,
		ErrorCode:           log.ErrorCode,
		ErrorMessage:        log.ErrorMessage,
		ClientIP:            log.ClientIP,
		GatewayNode:         log.GatewayNode,
		StartedAt:           log.StartedAt,
		CompletedAt:         log.CompletedAt,
	}

	// 补全渠道名 / 用户名（缺失保持 nil）
	type nameRow struct {
		ChannelName *string `gorm:"column:channel_name"`
		UserName    *string `gorm:"column:user_name"`
	}
	var nr nameRow
	if err := r.db.WithContext(ctx).
		Model(&model.RequestLog{}).
		Select("channels.name AS channel_name, users.username AS user_name").
		Joins("LEFT JOIN channels ON channels.id = request_logs.channel_id").
		Joins("LEFT JOIN users ON users.id = request_logs.user_id").
		Where("request_logs.id = ?", id).
		Scan(&nr).Error; err != nil {
		return nil, err
	}
	detail.ChannelName = nr.ChannelName
	detail.UserName = nr.UserName

	// 计费明细（缺失不报错）
	var b model.BillingRecord
	if err := r.db.WithContext(ctx).Where("request_id = ?", log.RequestID).First(&b).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return detail, nil
		}
		return nil, err
	}
	detail.Billing = toBillingDetail(&b)

	// 历史快照兜底：早期记录写入时未命中价格方案（单价全为 0），
	// 按当前绑定补全「单价」展示；成本与扣费保持记录原值，不做回溯结算。
	if b.InputPrice.IsZero() && b.OutputPrice.IsZero() && b.CachedInputPrice.IsZero() && b.CacheCreationPrice.IsZero() {
		var channelID int64
		if b.ChannelID != nil {
			channelID = *b.ChannelID
		}
		if p, err := r.ResolvePrice(ctx, channelID, b.Model); err == nil && p != nil {
			detail.Billing.Price = *p
		}
	}

	return detail, nil
}

// ResolvePrice 解析计价单价快照。
// 价格方案挂载在「渠道 × 模型」行上，但计费按逻辑模型名生效：
// 先精确匹配当前渠道的模型行，未命中再回退到同名逻辑模型在其他渠道上的绑定，
// 避免多渠道路由（或模型名大小写差异）导致漏计价。
func (r *logRepo) ResolvePrice(ctx context.Context, channelID int64, externalModel string) (*model.PriceSnapshot, error) {
	name := strings.TrimSpace(externalModel)
	if name == "" {
		return nil, nil
	}

	price, err := r.findPrice(ctx, name, &channelID)
	if err != nil || price != nil {
		return price, err
	}

	return r.findPrice(ctx, name, nil)
}

// findPrice 查询模型名对应的价格方案；channelID 非空时仅限该渠道。
// 先按对外模型名（models.name）匹配，再按上游模型名（models.upstream_model）匹配：
// 客户端可能直接用上游模型名请求（此时路由走兜底策略仍可成功），价格需要能对上。
func (r *logRepo) findPrice(ctx context.Context, name string, channelID *int64) (*model.PriceSnapshot, error) {
	for _, column := range []string{"m.name", "m.upstream_model"} {
		price, err := r.findPriceBy(ctx, column, name, channelID)
		if err != nil || price != nil {
			return price, err
		}
	}
	return nil, nil
}

// findPriceBy 按指定模型列查询价格方案；channelID 非空时仅限该渠道。
func (r *logRepo) findPriceBy(ctx context.Context, column, name string, channelID *int64) (*model.PriceSnapshot, error) {
	query := r.db.WithContext(ctx).Model(&model.ModelPrice{}).
		Select("model_prices.*").
		Joins("JOIN model_price_bindings b ON b.model_price_id = model_prices.id AND b.deleted_at IS NULL").
		Joins("JOIN models m ON m.id = b.model_id AND m.deleted_at IS NULL").
		Where("LOWER("+column+") = LOWER(?)", name)
	if channelID != nil {
		query = query.Where("m.channel_id = ?", *channelID)
	}

	var price model.ModelPrice
	if err := query.Order("b.id DESC").Limit(1).Scan(&price).Error; err != nil {
		return nil, err
	}
	if price.ID == 0 {
		return nil, nil
	}
	return &model.PriceSnapshot{
		InputPrice:         price.InputPrice,
		OutputPrice:        price.OutputPrice,
		CachedInputPrice:   price.CachedInputPrice,
		CacheCreationPrice: price.CacheCreationPrice,
		// 本项目 model_prices 无倍率字段，billing_records.ratio 默认 1 即按成本价
		Ratio:    1.0,
		Currency: price.Currency,
	}, nil
}

// toBillingDetail 将计费流水映射为对外计费明细视图
func toBillingDetail(b *model.BillingRecord) *model.BillingDetail {
	return &model.BillingDetail{
		ID:                  b.ID,
		RequestID:           b.RequestID,
		Model:               b.Model,
		ChannelID:           b.ChannelID,
		InputTokens:         b.InputTokens,
		OutputTokens:        b.OutputTokens,
		TotalTokens:         b.TotalTokens,
		CachedInputTokens:   b.CachedInputTokens,
		CacheCreationTokens: b.CacheCreationTokens,
		Price: model.PriceSnapshot{
			InputPrice:         b.InputPrice,
			OutputPrice:        b.OutputPrice,
			CachedInputPrice:   b.CachedInputPrice,
			CacheCreationPrice: b.CacheCreationPrice,
			Ratio:              b.Ratio,
			Currency:           b.Currency,
		},
		InputCost:         b.InputCost,
		OutputCost:        b.OutputCost,
		CachedInputCost:   b.CachedInputCost,
		CacheCreationCost: b.CacheCreationCost,
		TotalCost:         b.TotalCost,
		BalanceBefore:     b.BalanceBefore,
		BalanceAfter:      b.BalanceAfter,
		Status:            b.Status,
		CreatedAt:         b.CreatedAt,
	}
}

// billingConsumptionSelect 账单「消费」分支（来源 billing_records，出账）。
// 列别名必须在第一个 SELECT 中显式命名（UNION 的列名取第一分支），且与 BillingLogItem 的
// gorm column 名一一对应；amount 取负（-total_cost）表示资金减少。
const billingConsumptionSelect = `SELECT
	'consumption' AS source,
	b.id AS id,
	b.request_id AS request_id,
	b.user_id AS user_id,
	u.username AS user_name,
	b.model AS model,
	b.currency AS currency,
	(-1) * b.total_cost AS amount,
	b.balance_before AS balance_before,
	b.balance_after AS balance_after,
	b.status AS status,
	b.input_tokens AS input_tokens,
	b.output_tokens AS output_tokens,
	b.total_tokens AS total_tokens,
	b.cached_input_tokens AS cached_input_tokens,
	b.cache_creation_tokens AS cache_creation_tokens,
	b.input_price AS input_price,
	b.output_price AS output_price,
	b.cached_input_price AS cached_input_price,
	b.cache_creation_price AS cache_creation_price,
	b.ratio AS ratio,
	b.input_cost AS input_cost,
	b.output_cost AS output_cost,
	b.cached_input_cost AS cached_input_cost,
	b.cache_creation_cost AS cache_creation_cost,
	b.created_at AS created_at
FROM billing_records b
LEFT JOIN users u ON u.id = b.user_id`

// billingEntrySelect 账单「充值 / 退款」分支（来源 request_logs，入账）。
// 金额为正；request_logs 无余额快照与计价列，故 balance_* 取 NULL::numeric，
// tokens 取 0::bigint、单价与成本取 0::numeric、ratio 取 1::numeric（保持 UNION 各列类型一致）。
// 调用方需自行拼接 WHERE（其中必须包含 r.log_type 过滤）。
const billingEntrySelect = `SELECT
	r.log_type AS source,
	r.id AS id,
	r.request_id AS request_id,
	r.user_id AS user_id,
	u.username AS user_name,
	r.model AS model,
	r.currency AS currency,
	r.total_cost AS amount,
	NULL::numeric AS balance_before,
	NULL::numeric AS balance_after,
	r.status AS status,
	0::bigint AS input_tokens,
	0::bigint AS output_tokens,
	0::bigint AS total_tokens,
	0::bigint AS cached_input_tokens,
	0::bigint AS cache_creation_tokens,
	0::numeric AS input_price,
	0::numeric AS output_price,
	0::numeric AS cached_input_price,
	0::numeric AS cache_creation_price,
	1::numeric AS ratio,
	0::numeric AS input_cost,
	0::numeric AS output_cost,
	0::numeric AS cached_input_cost,
	0::numeric AS cache_creation_cost,
	r.created_at AS created_at
FROM request_logs r
LEFT JOIN users u ON u.id = r.user_id`

// parseTimeParam 依次尝试多种时间布局，解析失败返回 nil（忽略该条件）
func parseTimeParam(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, *s); err == nil {
			return &t
		}
	}
	return nil
}
