import apiClient, { handleApiResponse } from '@/lib/api-client'
import type { LogDetail, LogItem, LogListQuery, PageData } from '@/types/api'

// 分页查询日志列表（管理员可见全部，普通用户仅本人）
export function fetchLogs(query: LogListQuery) {
  return apiClient
    .get('/v1/logs', { params: query })
    .then(res => handleApiResponse<PageData<LogItem>>(res))
}

// 查询单条日志详情（含计费明细）
export function fetchLogDetail(id: number) {
  return apiClient
    .get(`/v1/logs/${id}`)
    .then(res => handleApiResponse<LogDetail>(res))
}
