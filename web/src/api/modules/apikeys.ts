import apiClient, { handleApiResponse } from '../client'
import type { ApiKey, CreateApiKeyRequest } from '@/types/api'

export function fetchApiKeys(groupId?: number) {
  const url = groupId ? `/v1/apikeys?group_id=${groupId}` : '/v1/apikeys'
  return apiClient.get(url).then(res => handleApiResponse<ApiKey[]>(res))
}

export function createApiKey(data: CreateApiKeyRequest) {
  return apiClient.post('/v1/apikeys', data).then(res => handleApiResponse<ApiKey>(res))
}

export function deleteApiKey(key: string) {
  // 后端使用 POST 请求体传递 key
  return apiClient.post('/v1/apikeys/delete', { key }).then(res => handleApiResponse<null>(res))
}
