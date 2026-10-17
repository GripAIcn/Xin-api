package model

import "time"

// 请求日志类型枚举，对应 request_logs.log_type
const (
	LogTypeConsumption = "consumption" // 消费
	LogTypeError       = "error"       // 错误
	LogTypeSystem      = "system"      // 系统
	LogTypeManagement  = "management"  // 管理
	LogTypeLogin       = "login"       // 登录
	LogTypeRefund      = "refund"      // 退款
	LogTypeRecharge    = "recharge"    // 充值
)

// 请求日志状态枚举，对应 request_logs.status
const (
	LogStatusSuccess = "success"
	LogStatusError   = "error"
)

// 错误归因阶段枚举，对应 request_logs.error_stage
const (
	ErrorStageUpstream = "upstream" // 上游
	ErrorStageLocal    = "local"    // 本地
)

// 用量来源枚举，对应 request_logs.usage_source
const (
	UsageSourceUpstream  = "upstream"  // 上游返回
	UsageSourceEstimated = "estimated" // 本地估算
)

// 中断方枚举，对应 request_logs.cancelled_by
const (
	CancelledByClient   = "client"   // 客户端中断
	CancelledByUpstream = "upstream" // 上游中断
)

// RequestLog 请求日志，对应 request_logs 表
// 说明：total_tokens 为数据库生成列（只读）。
type RequestLog struct {
	ID                  int64      `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	RequestID           string     `gorm:"column:request_id;type:varchar(64);not null;uniqueIndex:uq_request_logs_request,priority:1;comment:Xin API 侧请求 ID" json:"request_id"`
	UpstreamReqID       *string    `gorm:"column:upstream_req_id;type:varchar(128);comment:上游返回 ID" json:"upstream_req_id,omitempty"`
	UserID              *int64     `gorm:"column:user_id;comment:所属用户（不建外键）" json:"user_id,omitempty"`
	APIKeyID            *int64     `gorm:"column:api_key_id;comment:所属 Key（不建外键）" json:"api_key_id,omitempty"`
	Model               string     `gorm:"column:model;type:varchar(128);not null;comment:对外（逻辑）模型名" json:"model"`
	UpstreamModel       *string    `gorm:"column:upstream_model;type:varchar(128);comment:实际上游模型名" json:"upstream_model,omitempty"`
	ChannelID           *int64     `gorm:"column:channel_id;comment:命中渠道" json:"channel_id,omitempty"`
	Provider            *string    `gorm:"column:provider;type:varchar(64);comment:供应商" json:"provider,omitempty"`
	Protocol            *string    `gorm:"column:protocol;type:varchar(64);comment:请求协议 chat/responses/gemini" json:"protocol,omitempty"`
	LogType             string     `gorm:"column:log_type;type:varchar(32);not null;comment:日志类型" json:"log_type"`
	Status              string     `gorm:"column:status;type:varchar(32);not null;comment:状态 success/error" json:"status"`
	StatusCode          *int       `gorm:"column:status_code;comment:HTTP 状态码" json:"status_code,omitempty"`
	FinishReason        *string    `gorm:"column:finish_reason;type:varchar(64);comment:上游结束原因" json:"finish_reason,omitempty"`
	IsStream            bool       `gorm:"column:is_stream;default:false;comment:是否流式" json:"is_stream"`
	IsCancelled         bool       `gorm:"column:is_cancelled;default:false;comment:是否被中断" json:"is_cancelled"`
	CancelledBy         *string    `gorm:"column:cancelled_by;type:varchar(32);comment:中断方 client/upstream" json:"cancelled_by,omitempty"`
	RetryCount          int        `gorm:"column:retry_count;not null;default:0;comment:实际重试次数" json:"retry_count"`
	RetryChannels       Int64Array `gorm:"column:retry_channels;type:bigint[];default:'{}';comment:重试跨过的渠道 ID 序列" json:"retry_channels"`
	UsageSource         string     `gorm:"column:usage_source;type:varchar(32);not null;default:upstream;comment:用量来源 upstream/estimated" json:"usage_source"`
	RouteStrategy       *string    `gorm:"column:route_strategy;type:varchar(32);comment:选路策略快照" json:"route_strategy,omitempty"`
	RouteWeight         *int       `gorm:"column:route_weight;comment:命中渠道权重快照" json:"route_weight,omitempty"`
	InputTokens         int64      `gorm:"column:input_tokens;default:0;comment:总输入 tokens" json:"input_tokens"`
	OutputTokens        int64      `gorm:"column:output_tokens;default:0;comment:输出 tokens" json:"output_tokens"`
	TotalTokens         int64      `gorm:"column:total_tokens;type:bigint GENERATED ALWAYS AS (input_tokens + output_tokens) STORED;->;comment:总 tokens（生成列）" json:"total_tokens"`
	CachedInputTokens   int64      `gorm:"column:cached_input_tokens;default:0;comment:缓存命中输入 tokens" json:"cached_input_tokens"`
	CacheCreationTokens int64      `gorm:"column:cache_creation_tokens;default:0;comment:缓存创建输入 tokens" json:"cache_creation_tokens"`
	ReasoningTokens     int64      `gorm:"column:reasoning_tokens;default:0;comment:推理 tokens" json:"reasoning_tokens"`
	TTFTMs              *int64     `gorm:"column:ttft_ms;comment:首 token 延迟（ms）" json:"ttft_ms,omitempty"`
	LatencyMs           *int64     `gorm:"column:latency_ms;comment:总延迟（ms）" json:"latency_ms,omitempty"`
	GenerationMs        *int64     `gorm:"column:generation_ms;comment:生成耗时（ms）" json:"generation_ms,omitempty"`
	TotalCost           float64    `gorm:"column:total_cost;type:numeric(20,8);default:0;comment:本次请求计费金额" json:"total_cost"`
	Currency            string     `gorm:"column:currency;type:varchar(16);not null;default:USD;comment:成本币种快照" json:"currency"`
	ErrorStage          *string    `gorm:"column:error_stage;type:varchar(32);comment:错误归因阶段 upstream/local" json:"error_stage,omitempty"`
	ErrorType           *string    `gorm:"column:error_type;type:varchar(64);comment:错误类型细分" json:"error_type,omitempty"`
	ErrorCode           *string    `gorm:"column:error_code;type:varchar(64);comment:错误 code" json:"error_code,omitempty"`
	ErrorMessage        *string    `gorm:"column:error_message;type:text;comment:错误信息" json:"error_message,omitempty"`
	ClientIP            *string    `gorm:"column:client_ip;type:inet;comment:客户端 IP" json:"client_ip,omitempty"`
	GatewayNode         *string    `gorm:"column:gateway_node;type:varchar(128);comment:网关实例标识" json:"gateway_node,omitempty"`
	StartedAt           *time.Time `gorm:"column:started_at;comment:请求开始时间" json:"started_at,omitempty"`
	CompletedAt         *time.Time `gorm:"column:completed_at;comment:请求完成时间" json:"completed_at,omitempty"`
	CreatedAt           time.Time  `gorm:"column:created_at;autoCreateTime;uniqueIndex:uq_request_logs_request,priority:2" json:"created_at"`
}
