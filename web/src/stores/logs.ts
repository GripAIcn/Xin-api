import { create } from 'zustand'
import type { LogItem, LogListQuery } from '@/types/api'
import * as logsApi from '@/api/modules/logs'

interface LogsState {
  list: LogItem[]
  total: number
  page: number
  pageSize: number
  // 筛选条件（不含分页，分页由 page / pageSize 单独维护）
  filters: LogListQuery
  loading: boolean
  error: string | null
  // 合并筛选条件与分页参数拉取列表
  fetchList: () => Promise<void>
  // 局部更新筛选条件并回到第一页
  setFilter: (partial: LogListQuery) => void
  resetFilters: () => void
  setPage: (page: number) => void
  setPageSize: (pageSize: number) => void
}

export const useLogsStore = create<LogsState>()((set, get) => ({
  list: [],
  total: 0,
  page: 1,
  pageSize: 20,
  filters: {},
  loading: false,
  error: null,

  fetchList: async () => {
    set({ loading: true, error: null })
    const { filters, page, pageSize } = get()
    try {
      const data = await logsApi.fetchLogs({
        ...filters,
        page,
        page_size: pageSize,
      })
      set({
        list: data?.list ?? [],
        total: data?.total ?? 0,
        page: data?.page ?? page,
        pageSize: data?.page_size ?? pageSize,
        loading: false,
      })
    } catch (e) {
      set({
        loading: false,
        error: e instanceof Error && e.message ? e.message : '加载日志列表失败',
      })
      throw e
    }
  },

  setFilter: (partial) => {
    set((state) => ({ filters: { ...state.filters, ...partial }, page: 1 }))
  },

  resetFilters: () => {
    set({ filters: {}, page: 1 })
  },

  setPage: (page) => {
    set({ page: page < 1 ? 1 : page })
  },

  setPageSize: (pageSize) => {
    set({ pageSize, page: 1 })
  },
}))
