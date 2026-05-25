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
      groups.value = await groupsApi.listGroups()
    } finally {
      loading.value = false
    }
  }

  async function create(name: string) {
    const group = await groupsApi.createGroup({ name })
    groups.value.push(group)
  }

  async function update(id: number, name: string) {
    const updated = await groupsApi.updateGroup(id, { name })
    const index = groups.value.findIndex((g) => g.id === id)
    if (index !== -1) {
      groups.value[index] = updated
    }
  }

  async function toggleStatus(id: number, currentStatus: number) {
    const newStatus = currentStatus === 1 ? 0 : 1
    const updated = await groupsApi.updateGroupStatus(id, {
      status: newStatus,
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
