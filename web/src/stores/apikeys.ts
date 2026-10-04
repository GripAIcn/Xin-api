import { create } from 'zustand'
import type { ApiKey } from '@/types/api'
import * as apiKeysApi from '@/api/modules/apikeys'

interface ApiKeyState {
  apiKeys: ApiKey[]
  fetchAll: (groupId?: number) => Promise<ApiKey[]>
  create: (groupId: number) => Promise<ApiKey>
  remove: (key: string) => Promise<void>
}

export const useApiKeyStore = create<ApiKeyState>()((set) => ({
  apiKeys: [],

  fetchAll: async (groupId) => {
    const data = await apiKeysApi.fetchApiKeys(groupId)
    set({ apiKeys: data || [] })
    return data
  },

  create: async (groupId) => {
    const data = await apiKeysApi.createApiKey({ group_id: groupId })
    set((state) => ({ apiKeys: [...state.apiKeys, data] }))
    return data
  },

  remove: async (key) => {
    // 根据 API 文档，key 通过请求体传递
    await apiKeysApi.deleteApiKey({ key })
    set((state) => ({ apiKeys: state.apiKeys.filter((k) => k.key !== key) }))
  },
}))