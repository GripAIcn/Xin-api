import { create } from 'zustand'
import type { ModelGroup } from '@/types/api'
import * as modelsApi from '@/api/modules/models'

interface ModelState {
  // 按逻辑模型名聚合的分组列表
  groups: ModelGroup[]
  loading: boolean
  fetchAll: () => Promise<ModelGroup[]>
  // 本地直接覆盖分组列表（用于前端本地刷新）
  setGroups: (groups: ModelGroup[]) => void
}

export const useModelStore = create<ModelState>()((set) => ({
  groups: [],
  loading: false,

  // 拉取全部模型分组：失败时复位 loading 并向上抛出，由调用方 toast
  fetchAll: async () => {
    set({ loading: true })
    try {
      const data = await modelsApi.fetchModelGroups()
      set({ groups: data || [], loading: false })
      return data
    } catch (err) {
      set({ loading: false })
      throw err
    }
  },

  setGroups: (groups) => {
    set({ groups })
  },
}))
