import apiClient, { handleApiResponse } from '../client'
import type { ApiKey, CreateApiKeyRequest, DeleteApiKeyRequest } from '@/types/api'

export function fetchApiKeys(groupId?: number) {
  const url = groupId ? `/v1/apikeys?group_id=${groupId}` : '/v1/apikeys'
  return apiClient.get(url).then(res => handleApiResponse<ApiKey[]>(res))
}

export function createApiKey(data: CreateApiKeyRequest) {
  return apiClient.post('/v1/apikeys', data).then(res => handleApiResponse<ApiKey>(res))
}

export function deleteApiKey(data: DeleteApiKeyRequest) {
  // 使用 POST /v1/apikeys/delete 以便从 body 读取 key
  return apiClient.post('/v1/apikeys/delete', data).then(res => handleApiResponse<null>(res))
}
