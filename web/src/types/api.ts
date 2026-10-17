// === API Response ===

export interface AdminResponse<T> {
  code: number
  message: string
  data?: T
}

// === Auth ===

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  username: string
  role: string
  must_change_password: boolean
}

export interface UpdatePasswordRequest {
  old_password: string
  new_password: string
}

// === User ===

export interface User {
  id: number
  username: string
  email?: string
  phone?: string
  role: string
  status: string
  must_change_password: boolean
  balance: number
  currency: string
  quota?: number
  used_quota: number
  last_login_at?: string
  created_at: string
  updated_at: string
}

export interface CreateUserRequest {
  username: string
  password: string
  email?: string
  phone?: string
}

export interface UpdateProfileRequest {
  old_password: string
  email?: string
  phone?: string
}

export interface UpdateUserStatusRequest {
  status: string
}

export interface AdminResetPasswordRequest {
  new_password: string
}

export interface RechargeBalanceRequest {
  amount: number
}

// === Channel ===

export interface Channel {
  id: number
  name: string
  provider: string
  protocol: string[]
  base_url: string
  api_key: string
  weight: number
  priority: number
  tags: string[]
  status: string
  timeout: number
  max_retries: number
  last_error_at?: string
  circuit_opened_at?: string
  created_at: string
  updated_at: string
}

export interface CreateChannelRequest {
  name: string
  provider: string
  protocol: string[]
  base_url: string
  api_key: string
  weight: number
  priority?: number
  tags?: string[]
  timeout?: number
  max_retries?: number
}

export interface UpdateChannelRequest {
  name: string
  provider: string
  protocol: string[]
  base_url: string
  api_key: string
  weight: number
  priority?: number
  tags?: string[]
  timeout?: number
  max_retries?: number
  status: string
}

// === Channel Test ===

export interface TestChannelRequest {
  model?: string
}

export interface ModelTestResult {
  model: string
  success: boolean
  response_time_ms: number
  error_msg?: string
}

export interface ChannelTestResult {
  channel_id: number
  channel_name: string
  base_url: string
  results: ModelTestResult[]
}

// === Channel Model ===

export type ChannelModelStatus = 'active' | 'disabled'

// 渠道下的可用模型（渠道 × 模型）：name 为对外（逻辑）模型名，upstream_model 为上游真实模型名
export interface ChannelModel {
  id: number
  channel_id: number
  name: string
  upstream_model: string
  context_length?: number
  max_output_tokens?: number
  capabilities: string[]
  status: ChannelModelStatus
  // 绑定的价格方案 id / 方案名（未绑定时缺省）
  model_price_id?: number
  price_name?: string | null
  created_at: string
  updated_at: string
}

export interface CreateChannelModelRequest {
  name: string
  upstream_model: string
  context_length?: number
  max_output_tokens?: number
  capabilities?: string[]
  status?: ChannelModelStatus
  // 绑定的价格方案 id（缺省=不改变绑定 / 0=解绑 / >0=绑定）
  model_price_id?: number
}

export interface UpdateChannelModelRequest {
  name: string
  upstream_model: string
  context_length?: number
  max_output_tokens?: number
  capabilities?: string[]
  status?: ChannelModelStatus
  // 绑定的价格方案 id（缺省=不改变绑定 / 0=解绑 / >0=绑定）
  model_price_id?: number
}

export interface UpdateChannelModelStatusRequest {
  status: ChannelModelStatus
}

// === 模型管理（按逻辑模型名聚合）===

// 逻辑模型在某渠道下的明细行（GET /v1/models 的 channels 元素）
export interface ModelChannelEntry {
  model_id: number
  channel_id: number
  channel_name: string
  channel_provider: string
  channel_status: string // active | disabled | circuit_open
  upstream_model: string
  status: ChannelModelStatus // 模型自身状态
  context_length?: number
  max_output_tokens?: number
  capabilities: string[]
  // 绑定的价格方案 id / 方案名（未绑定时缺省）
  model_price_id?: number | null
  price_name?: string | null
}

// 按逻辑模型名聚合的分组（GET /v1/models 的元素）
export interface ModelGroup {
  name: string
  channel_count: number
  active_count: number
  channels: ModelChannelEntry[]
}

// === 价格方案 ===

export type ModelPriceStatus = 'active' | 'disabled'

// 价格方案（model_prices）：单位 currency / 1M tokens
export interface ModelPrice {
  id: number
  name: string
  input_price: number
  output_price: number
  cached_input_price: number
  cache_creation_price: number
  currency: string
  status: ModelPriceStatus
  created_at: string
  updated_at: string
}

// 价格方案 + 绑定模型数（GET /v1/prices 的元素）
export interface ModelPriceWithCount extends ModelPrice {
  bound_count: number
}

export interface CreateModelPriceRequest {
  name: string
  input_price: number
  output_price: number
  cached_input_price: number
  cache_creation_price: number
  currency: string
}

export interface UpdateModelPriceRequest {
  name: string
  input_price: number
  output_price: number
  cached_input_price: number
  cache_creation_price: number
  currency: string
  status?: ModelPriceStatus
}

export interface UpdateModelPriceStatusRequest {
  status: ModelPriceStatus
}

// 绑定到某价格方案的「渠道 × 模型」明细（GET /v1/prices/:id/models 的元素）
export interface PriceBoundModelEntry {
  binding_id: number
  model_id: number
  channel_id: number
  channel_name: string
  channel_provider: string
  model_name: string
  upstream_model: string
  model_status: string
  channel_status: string
}

// === API Key ===

export interface ApiKey {
  id: number
  name: string
  user_id: number
  status: string
  expires_at?: string
  last_used_at?: string
  created_at: string
}

export interface CreateApiKeyRequest {
  name?: string
  expires_at?: string
}

// 创建响应：明文 key 仅在创建时返回一次，其余为元数据
export interface CreateApiKeyResponse extends ApiKey {
  key: string
}

export interface DeleteApiKeyRequest {
  id: number
}

// === 分页 ===

// 控制面通用分页返回体（对应后端 response.PageData）
export interface PageData<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

// === 日志（使用日志 + 消费记录）===

export type LogType =
  | 'consumption'
  | 'error'
  | 'system'
  | 'management'
  | 'login'
  | 'refund'
  | 'recharge'

export type LogStatus = 'success' | 'error'

// 日志列表行（GET /v1/logs 的 list 元素）
export interface LogItem {
  id: number
  request_id: string
  upstream_req_id?: string
  log_type: LogType
  status: LogStatus
  model: string
  channel_id?: number
  channel_name?: string
  api_key_id?: number
  user_id?: number
  user_name?: string
  input_tokens: number
  output_tokens: number
  total_tokens: number
  total_cost: number
  currency: string
  latency_ms?: number
  created_at: string
}

// 计价单价快照（billing_records 单价 + ratio + 币种）
export interface PriceSnapshot {
  input_price: number
  output_price: number
  cached_input_price: number
  cache_creation_price: number
  ratio: number
  currency: string
}

// 计费明细（消费记录，来自 billing_records）
export interface BillingDetail {
  id: number
  request_id: string
  model: string
  channel_id?: number
  input_tokens: number
  output_tokens: number
  total_tokens: number
  cached_input_tokens: number
  cache_creation_tokens: number
  price: PriceSnapshot
  input_cost: number
  output_cost: number
  cached_input_cost: number
  cache_creation_cost: number
  total_cost: number
  settle_currency: string
  settle_cost: number
  exchange_rate?: number
  balance_before?: number
  balance_after?: number
  status: string
  created_at: string
}

// 日志详情（GET /v1/logs/:id）
export interface LogDetail extends LogItem {
  upstream_model?: string
  provider?: string
  protocol?: string
  status_code?: number
  finish_reason?: string
  is_stream: boolean
  is_cancelled: boolean
  cancelled_by?: string
  retry_count: number
  retry_channels: number[]
  usage_source: string
  route_strategy?: string
  route_weight?: number
  cached_input_tokens: number
  cache_creation_tokens: number
  reasoning_tokens: number
  ttft_ms?: number
  generation_ms?: number
  error_stage?: string
  error_type?: string
  error_code?: string
  error_message?: string
  client_ip?: string
  gateway_node?: string
  started_at?: string
  completed_at?: string
  billing?: BillingDetail
}

// 日志列表查询参数（字段名与后端 query 一致，可直接作为 query 发送）
export interface LogListQuery {
  page?: number
  page_size?: number
  log_type?: LogType
  model?: string
  channel_id?: number
  user_id?: number
  api_key_id?: number
  request_id?: string
  provider?: string
  protocol?: string
  is_stream?: boolean
  upstream_req_id?: string
  status?: LogStatus
  start_time?: string
  end_time?: string
}

// === 账单日志（消费 / 充值 / 退款资金流水）===

export type BillingLogSource = 'consumption' | 'recharge' | 'refund'

// 账单流水行（GET /v1/billing-logs 的 list 元素，后端扁平结构）
export interface BillingLogItem {
  id: number
  source: BillingLogSource
  request_id: string
  user_id?: number
  user_name?: string
  model: string
  currency: string
  settle_currency: string
  settle_cost: number
  exchange_rate?: number
  // 带符号金额：消费为负，充值 / 退款为正
  amount: number
  balance_before?: number
  balance_after?: number
  status: string
  input_tokens: number
  output_tokens: number
  total_tokens: number
  cached_input_tokens: number
  cache_creation_tokens: number
  input_price: number
  output_price: number
  cached_input_price: number
  cache_creation_price: number
  ratio: number
  input_cost: number
  output_cost: number
  cached_input_cost: number
  cache_creation_cost: number
  created_at: string
}

// 账单流水查询参数（字段名与后端 query 一致，可直接作为 query 发送）
export interface BillingLogQuery {
  page?: number
  page_size?: number
  source?: BillingLogSource
  user_id?: number
  model?: string
  request_id?: string
  start_time?: string
  end_time?: string
}

// === 渠道指标监控 ===

export type ChannelMetricRange = '1h' | '6h' | '24h' | '7d'
export type ChannelMetricDimension = 'channel' | 'model' | 'channel_model'

// 指标跨窗口二次聚合行（GET /v1/channel-metrics 的元素）
export interface ChannelMetricSummary {
  channel_id: number | null
  channel_name: string | null
  channel_provider: string | null
  channel_status: string | null
  model: string | null
  request_count: number
  success_count: number
  error_count: number
  timeout_count: number
  success_rate: number
  avg_latency_ms: number
  avg_ttft_ms: number
  tpot: number
  rpm: number
  tpm: number
  total_input_tokens: number
  total_output_tokens: number
  total_tokens: number
  total_cost: number
  currency: string
  last_window_start: string | null
}

// 指标趋势分桶点（GET /v1/channel-metrics/trend 的元素）
export interface ChannelMetricTrendPoint {
  bucket_start: string
  request_count: number
  success_count: number
  avg_ttft_ms: number
  avg_latency_ms: number
  tpot: number
  success_rate: number
}

// 指标汇总查询参数
export interface ChannelMetricQuery {
  range?: ChannelMetricRange
  dimension?: ChannelMetricDimension
  channel_id?: number
  model?: string
}

// 指标趋势查询参数
export interface ChannelMetricTrendQuery {
  range?: ChannelMetricRange
  channel_id?: number
  model?: string
}

// 手动范围聚合入参（POST /v1/channel-metrics/aggregate）
export interface AggregateChannelMetricsRequest {
  start_time: string
  end_time: string
}

export interface AggregateChannelMetricsResult {
  start_time: string
  end_time: string
  rows: number
  duration_ms: number
}

// === 仪表盘统计 ===

export type DashboardRange = 'today' | '7d' | '30d'

// 概览指标（所选时间范围聚合）
export interface DashboardOverview {
  request_count: number
  success_count: number
  error_count: number
  success_rate: number // 0~1 小数
  input_tokens: number
  output_tokens: number
  total_tokens: number
  total_cost: number
  currency: string
  avg_latency_ms: number
  avg_ttft_ms: number
}

// 趋势分桶点（今日按小时、近 N 天按天）
export interface DashboardTrendPoint {
  bucket_start: string
  request_count: number
  success_count: number
  error_count: number
  input_tokens: number
  output_tokens: number
  total_tokens: number
  total_cost: number
}

export interface DashboardModelRank {
  model: string
  request_count: number
  total_tokens: number
  total_cost: number
}

export interface DashboardChannelRank {
  channel_id: number
  channel_name: string | null
  request_count: number
  success_rate: number
  avg_latency_ms: number
}

export interface DashboardErrorRank {
  error_type: string
  count: number
}

// === 汇率 ===

export interface ExchangeRate {
  from_currency: string
  to_currency: string
  rate: number
  source: string
  updated_at: string
}

export interface UpdateExchangeRateRequest {
  rate: number
}

// GET /v1/dashboard/stats 返回体
export interface DashboardStats {
  range: string
  start_time: string
  end_time: string
  overview: DashboardOverview
  trend: DashboardTrendPoint[]
  top_models: DashboardModelRank[]
  top_channels: DashboardChannelRank[]
  top_errors: DashboardErrorRank[]
}
