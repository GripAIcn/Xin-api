import { create } from 'zustand'
import type { ApiKey, CreateApiKeyResponse } from '@/types/api'
import * as apiKeysApi from '@/api/modules/apikeys'

interface ApiKeyState {
  apiKeys: ApiKey[]
  fetchAll: () => Promise<ApiKey[]>
  create: (name?: string) => Promise<CreateApiKeyResponse>
  remove: (id: number) => Promise<void>
}

// 仅保留元数据，避免明文 key 残留在列表中
function toApiKeyMeta(data: CreateApiKeyResponse): ApiKey {
  return {
    id: data.id,
    name: data.name,
    user_id: data.user_id,
    status: data.status,
    expires_at: data.expires_at,
    last_used_at: data.last_used_at,
    created_at: data.created_at,
  }
}

export const useApiKeyStore = create<ApiKeyState>()((set) => ({
  apiKeys: [],

  fetchAll: async () => {
    const data = await apiKeysApi.fetchApiKeys()
    set({ apiKeys: data || [] })
    return data
  },

  create: async (name) => {
    const data = await apiKeysApi.createApiKey({ name })
    set((state) => ({ apiKeys: [...state.apiKeys, toApiKeyMeta(data)] }))
    return data
  },

  remove: async (id) => {
    await apiKeysApi.deleteApiKey({ id })
    set((state) => ({ apiKeys: state.apiKeys.filter((k) => k.id !== id) }))
  },
}))
