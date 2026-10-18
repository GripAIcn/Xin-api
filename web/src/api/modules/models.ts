import apiClient, { handleApiResponse } from '@/lib/api-client'
import type { ModelGroup } from '@/types/api'

// 按逻辑模型名聚合查询全部渠道模型（模型管理页）
export function fetchModelGroups() {
  return apiClient.get('/v1/admin/models').then(res => handleApiResponse<ModelGroup[]>(res))
}
