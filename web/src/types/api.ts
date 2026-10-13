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
}

export interface UpdatePasswordRequest {
  old_password: string
  new_password: string
}

export interface UpdateAccountRequest {
  username: string
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
