import apiClient, { handleApiResponse } from '@/lib/api-client'
import type {
  AggregateChannelMetricsRequest,
  AggregateChannelMetricsResult,
  ChannelMetricQuery,
  ChannelMetricSummary,
  ChannelMetricTrendPoint,
  ChannelMetricTrendQuery,
} from '@/types/api'

// 按渠道 / 模型维度汇总指标
export function fetchChannelMetrics(query: ChannelMetricQuery) {
  return apiClient
    .get('/v1/channel-metrics', { params: query })
    .then(res => handleApiResponse<ChannelMetricSummary[]>(res))
}

// 查询 TTFT / 延迟走势
export function fetchChannelMetricTrend(query: ChannelMetricTrendQuery) {
  return apiClient
    .get('/v1/channel-metrics/trend', { params: query })
    .then(res => handleApiResponse<ChannelMetricTrendPoint[]>(res))
}

// 手动按时间范围聚合写入
export function aggregateChannelMetrics(data: AggregateChannelMetricsRequest) {
  return apiClient
    .post('/v1/channel-metrics/aggregate', data)
    .then(res => handleApiResponse<AggregateChannelMetricsResult>(res))
}
