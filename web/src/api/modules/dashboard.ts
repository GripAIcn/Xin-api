import apiClient, { handleApiResponse } from '@/lib/api-client'
import type { DashboardRange, DashboardStats } from '@/types/api'

// 查询仪表盘统计（概览 / 趋势 / 排行）
export function fetchDashboardStats(range: DashboardRange) {
  return apiClient
    .get('/v1/dashboard/stats', { params: { range } })
    .then(res => handleApiResponse<DashboardStats>(res))
}
