import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { Group } from '@/types/api'
import * as groupsApi from '@/api/modules/groups'

export const useGroupStore = defineStore('groups', () => {
  const groups = ref<Group[]>([])
  const loading = ref(false)

  async function fetchAll() {
    loading.value = true
    try {
      const result = await groupsApi.listGroups()
      groups.value = result || []
    } catch (error) {
      groups.value = []
      throw error
    } finally {
      loading.value = false
    }
  }

  async function create(name: string) {
    const newGroup = await groupsApi.createGroup({ name })
    // 将新创建的项目组添加到列表中
    groups.value.unshift(newGroup)
  }

  async function update(id: number, name: string) {
    const updated = await groupsApi.updateGroup(id, { name })
    const index = groups.value.findIndex((g) => g.id === id)
    if (index !== -1) {
      groups.value[index] = updated
    }
  }

  async function toggleStatus(id: number, currentStatus: number) {
    const updated = await groupsApi.updateGroupStatus(id, {
      status: currentStatus === 1 ? 0 : 1,
    })
    const index = groups.value.findIndex((g) => g.id === id)
    if (index !== -1) {
      groups.value[index] = updated
    }
  }

  async function remove(id: number) {
    await groupsApi.deleteGroup(id)
    groups.value = groups.value.filter((g) => g.id !== id)
  }

  return {
    groups,
    loading,
    fetchAll,
    create,
    update,
    toggleStatus,
    remove,
  }
})
