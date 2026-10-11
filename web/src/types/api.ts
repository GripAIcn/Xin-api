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
  model_mapping: string
  base_url: string
  api_key: string
  weight: number
  status: number
  created_at: string
  updated_at: string
}

export interface CreateChannelRequest {
  name: string
  model_mapping: string
  base_url: string
  api_key: string
  weight: number
}

export interface UpdateChannelRequest {
  name: string
  model_mapping: string
  base_url: string
  api_key: string
  weight: number
  status: number
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
  key: string
  created_at: string
}

export interface DeleteApiKeyRequest {
  key: string
}
