import apiClient, { handleApiResponse } from '@/lib/api-client'
import type { ApiKey, CreateApiKeyRequest, CreateApiKeyResponse, DeleteApiKeyRequest } from '@/types/api'

export function fetchApiKeys() {
  return apiClient.get('/v1/apikeys').then(res => handleApiResponse<ApiKey[]>(res))
}

// 创建凭证：明文 key 仅在响应中返回一次
export function createApiKey(data?: CreateApiKeyRequest) {
  return apiClient.post('/v1/apikeys', data ?? {}).then(res => handleApiResponse<CreateApiKeyResponse>(res))
}

export function deleteApiKey(data: DeleteApiKeyRequest) {
  // 使用 POST /v1/apikeys/delete 以便从 body 读取 id
  return apiClient.post('/v1/apikeys/delete', data).then(res => handleApiResponse<null>(res))
}
