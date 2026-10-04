import { create } from 'zustand'
import type { Group } from '@/types/api'
import * as groupsApi from '@/api/modules/groups'

interface GroupState {
  groups: Group[]
  fetchAll: () => Promise<Group[]>
  create: (name: string) => Promise<Group>
  update: (id: number, name: string) => Promise<Group>
  toggleStatus: (id: number, currentStatus: number) => Promise<Group>
  remove: (id: number) => Promise<void>
}

export const useGroupStore = create<GroupState>()((set) => ({
  groups: [],

  fetchAll: async () => {
    const data = await groupsApi.fetchGroups()
    set({ groups: data || [] })
    return data
  },

  create: async (name) => {
    const data = await groupsApi.createGroup({ name })
    set((state) => ({ groups: [...state.groups, data] }))
    return data
  },

  update: async (id, name) => {
    const data = await groupsApi.updateGroup(id, { name })
    set((state) => ({
      groups: state.groups.map((g) => (g.id === id ? data : g)),
    }))
    return data
  },

  toggleStatus: async (id, currentStatus) => {
    const newStatus = currentStatus === 1 ? 0 : 1
    const data = await groupsApi.updateGroupStatus(id, { status: newStatus })
    set((state) => ({
      groups: state.groups.map((g) => (g.id === id ? data : g)),
    }))
    return data
  },

  remove: async (id) => {
    await groupsApi.deleteGroup(id)
    set((state) => ({ groups: state.groups.filter((g) => g.id !== id) }))
  },
}))