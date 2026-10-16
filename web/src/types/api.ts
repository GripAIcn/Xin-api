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
