import { create } from 'zustand'
import type { BillingLogItem, BillingLogQuery } from '@/types/api'
import * as billingApi from '@/api/modules/billing'

interface BillingState {
  list: BillingLogItem[]
  total: number
  page: number
  pageSize: number
  // 筛选条件（不含分页，分页由 page / pageSize 单独维护）
  filters: BillingLogQuery
  loading: boolean
  error: string | null
  // 合并筛选条件与分页参数拉取列表
  fetchList: () => Promise<void>
  // 局部更新筛选条件并回到第一页
  setFilter: (partial: BillingLogQuery) => void
  resetFilters: () => void
  setPage: (page: number) => void
  setPageSize: (pageSize: number) => void
}

export const useBillingStore = create<BillingState>()((set, get) => ({
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
      const data = await billingApi.fetchBillingLogs({
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
        error: e instanceof Error && e.message ? e.message : '加载账单列表失败',
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
