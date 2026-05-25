import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { Channel, CreateChannelRequest, UpdateChannelRequest } from '@/types/api'
import * as channelsApi from '@/api/modules/channels'

export const useChannelStore = defineStore('channels', () => {
  const channels = ref<Channel[]>([])
  const loading = ref(false)

  async function fetchByGroup(groupId?: number) {
    loading.value = true
    try {
      channels.value = await channelsApi.listChannelsByGroup(groupId)
    } catch (error: any) {
      // 重新抛出错误以便组件可以处理
      throw error
    } finally {
      loading.value = false
    }
  }

  async function create(data: CreateChannelRequest) {
    const channel = await channelsApi.createChannel(data)
    channels.value.push(channel)
  }

  async function update(id: number, data: UpdateChannelRequest) {
    const updated = await channelsApi.updateChannel(id, data)
    const index = channels.value.findIndex((c) => c.id === id)
    if (index !== -1) {
      channels.value[index] = updated
    }
  }

  async function remove(id: number) {
    await channelsApi.deleteChannel(id)
    channels.value = channels.value.filter((c) => c.id !== id)
  }

  return {
    channels,
    loading,
    fetchByGroup,
    create,
    update,
    remove,
  }
})
