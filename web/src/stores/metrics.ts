import { create } from 'zustand'
import type {
  AggregateChannelMetricsResult,
  ChannelMetricQuery,
  ChannelMetricSummary,
  ChannelMetricTrendPoint,
} from '@/types/api'
import * as metricsApi from '@/api/modules/metrics'

interface MetricState {
  summaries: ChannelMetricSummary[]
  trend: ChannelMetricTrendPoint[]
  loading: boolean
  error: string | null
  // 并行加载维度汇总与趋势
  fetchAll: (query: ChannelMetricQuery) => Promise<void>
  aggregate: (
    startTime: string,
    endTime: string,
  ) => Promise<AggregateChannelMetricsResult>
}

// 请求序号：筛选条件快速切换时，只保留最后一次请求的结果
let requestSeq = 0

export const useMetricStore = create<MetricState>()((set) => ({
  summaries: [],
  trend: [],
  loading: false,
  error: null,

  fetchAll: async (query) => {
    const seq = ++requestSeq
    set({ loading: true, error: null })
    try {
      const [summaries, trend] = await Promise.all([
        metricsApi.fetchChannelMetrics(query),
        metricsApi.fetchChannelMetricTrend({
          range: query.range,
          channel_id: query.channel_id,
          model: query.model,
        }),
      ])
      if (seq !== requestSeq) return
      set({
        summaries: summaries ?? [],
        trend: trend ?? [],
        loading: false,
      })
    } catch (error) {
      if (seq !== requestSeq) return
      set({
        summaries: [],
        trend: [],
        loading: false,
        error: error instanceof Error ? error.message : '加载失败',
      })
    }
  },

  aggregate: (startTime, endTime) =>
    metricsApi.aggregateChannelMetrics({
      start_time: startTime,
      end_time: endTime,
    }),
}))
