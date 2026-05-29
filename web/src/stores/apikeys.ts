import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { ApiKey } from '@/types/api'
import { createApiKey as createApiKeyApi } from '@/api/modules/apikeys'
import * as apikeysApi from '@/api/modules/apikeys'

export const useApiKeyStore = defineStore('apikeys', () => {
  const apiKeys = ref<ApiKey[]>([])
  const loading = ref(false)

  async function fetchAll(groupId?: number) {
    loading.value = true
    try {
      const result = await apikeysApi.listApiKeys(groupId)
      apiKeys.value = result || []
    } catch (error: any) {
      apiKeys.value = []
      throw error
    } finally {
      loading.value = false
    }
  }

  async function create(groupId: number): Promise<ApiKey> {
    const newKey = await createApiKeyApi({ group_id: groupId })
    apiKeys.value.push(newKey)
    return newKey
  }

  async function remove(key: string) {
    await apikeysApi.deleteApiKey(key)
    apiKeys.value = apiKeys.value.filter((k) => k.key !== key)
  }

  return {
    apiKeys,
    loading,
    fetchAll,
    create,
    remove,
  }
})
