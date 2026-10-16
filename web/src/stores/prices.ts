import { create } from 'zustand'
import type { ModelPriceWithCount } from '@/types/api'
import * as pricesApi from '@/api/modules/prices'

interface PriceState {
  // 价格方案列表（含绑定模型数）
  prices: ModelPriceWithCount[]
  loading: boolean
  fetchAll: () => Promise<ModelPriceWithCount[]>
  // 本地直接覆盖方案列表（用于前端本地刷新）
  setPrices: (prices: ModelPriceWithCount[]) => void
}

export const usePriceStore = create<PriceState>()((set) => ({
  prices: [],
  loading: false,

  // 拉取全部价格方案：失败时复位 loading 并向上抛出，由调用方 toast
  fetchAll: async () => {
    set({ loading: true })
    try {
      const data = await pricesApi.fetchPrices()
      set({ prices: data || [], loading: false })
      return data
    } catch (err) {
      set({ loading: false })
      throw err
    }
  },

  setPrices: (prices) => {
    set({ prices })
  },
}))
