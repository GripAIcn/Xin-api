import { create } from 'zustand'
import type { DashboardRange, DashboardStats } from '@/types/api'
import * as dashboardApi from '@/api/modules/dashboard'

interface DashboardState {
  stats: DashboardStats | null
  range: DashboardRange
  loading: boolean
  error: string | null
  setRange: (range: DashboardRange) => void
  fetchStats: (range?: DashboardRange) => Promise<void>
}

// 请求序号：时间范围快速切换时，只保留最后一次请求的结果
let requestSeq = 0

export const useDashboardStore = create<DashboardState>()((set, get) => ({
  stats: null,
  range: 'today',
  loading: false,
  error: null,

  setRange: (range) => set({ range }),

  fetchStats: async (range) => {
    const nextRange = range ?? get().range
    const seq = ++requestSeq
    set({ loading: true, error: null, range: nextRange })
    try {
      const stats = await dashboardApi.fetchDashboardStats(nextRange)
      if (seq !== requestSeq) return
      set({ stats, loading: false })
    } catch (error) {
      if (seq !== requestSeq) return
      set({
        stats: null,
        loading: false,
        error: error instanceof Error ? error.message : '加载失败',
      })
    }
  },
}))
