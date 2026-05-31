import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { Group } from '@/types/api'
import * as groupsApi from '@/api/modules/groups'

export const useGroupStore = defineStore('groups', () => {
  const groups = ref<Group[]>([])

  async function fetchAll() {
    const data = await groupsApi.fetchGroups()
    groups.value = data || []
    return data
  }

  async function create(name: string) {
    const data = await groupsApi.createGroup({ name })
    groups.value.push(data)
    return data
  }

  async function update(id: number, name: string) {
    const data = await groupsApi.updateGroup(id, { name })
    const index = groups.value.findIndex(g => g.id === id)
    if (index !== -1) {
      groups.value[index] = data
    }
    return data
  }

  async function toggleStatus(id: number, currentStatus: number) {
    const newStatus = currentStatus === 1 ? 0 : 1
    const data = await groupsApi.updateGroupStatus(id, { status: newStatus })
    const index = groups.value.findIndex(g => g.id === id)
    if (index !== -1) {
      groups.value[index] = data
    }
    return data
  }

  async function remove(id: number) {
    await groupsApi.deleteGroup(id)
    groups.value = groups.value.filter(g => g.id !== id)
  }

  return {
    groups,
    fetchAll,
    create,
    update,
    toggleStatus,
    remove,
  }
})
