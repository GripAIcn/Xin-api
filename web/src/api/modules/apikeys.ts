import apiClient, { handleApiResponse } from '@/lib/api-client'
import type { ApiKey, DeleteApiKeyRequest } from '@/types/api'

export function fetchApiKeys() {
  return apiClient.get('/v1/apikeys').then(res => handleApiResponse<ApiKey[]>(res))
}

export function createApiKey() {
  return apiClient.post('/v1/apikeys', {}).then(res => handleApiResponse<ApiKey>(res))
}

export function deleteApiKey(data: DeleteApiKeyRequest) {
  // 使用 POST /v1/apikeys/delete 以便从 body 读取 key
  return apiClient.post('/v1/apikeys/delete', data).then(res => handleApiResponse<null>(res))
}
