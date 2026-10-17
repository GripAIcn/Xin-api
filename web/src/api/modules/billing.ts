import apiClient, { handleApiResponse } from '@/lib/api-client'
import type { BillingLogItem, BillingLogQuery, PageData } from '@/types/api'

// 分页查询账单流水（管理员可见全部，普通用户仅本人）
export function fetchBillingLogs(query: BillingLogQuery) {
  return apiClient
    .get('/v1/billing-logs', { params: query })
    .then(res => handleApiResponse<PageData<BillingLogItem>>(res))
}
