import apiClient, { handleApiResponse } from '@/api/client'
import type {
  ApiKey,
  CreateApiKeyRequest,
} from '@/types/api'

export async function listApiKeys(groupId?: number): Promise<ApiKey[]> {
  // 如果有 groupId，作为查询参数传递
  if (groupId !== undefined && groupId !== null && groupId > 0) {
    const response = await apiClient.get('/v1/apikeys', {
      params: { group_id: groupId }
    })
    return handleApiResponse<ApiKey[]>(response)
  }
  // 没有 groupId 时，不传递任何参数
  const response = await apiClient.get('/v1/apikeys')
  return handleApiResponse<ApiKey[]>(response)
}

export async function createApiKey(
  data: CreateApiKeyRequest,
): Promise<ApiKey> {
  const response = await apiClient.post('/v1/apikeys', data)
  return handleApiResponse<ApiKey>(response)
}

export async function deleteApiKey(key: string): Promise<void> {
  const response = await apiClient.delete(`/v1/apikeys/${key}`)
  handleApiResponse(response)
}
