import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { ApiKey } from '@/types/api'
import * as apiKeysApi from '@/api/modules/apikeys'

export const useApiKeyStore = defineStore('apikeys', () => {
  const apiKeys = ref<ApiKey[]>([])

  async function fetchAll(groupId?: number) {
    const data = await apiKeysApi.fetchApiKeys(groupId)
    apiKeys.value = data || []
    return data
  }

  async function create(groupId: number) {
    const data = await apiKeysApi.createApiKey({ group_id: groupId })
    apiKeys.value.push(data)
    return data
  }

  async function remove(key: string) {
    await apiKeysApi.deleteApiKey(key)
    apiKeys.value = apiKeys.value.filter(k => k.key !== key)
  }

  return {
    apiKeys,
    fetchAll,
    create,
    remove,
  }
})
