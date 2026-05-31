import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { Channel, CreateChannelRequest, UpdateChannelRequest } from '@/types/api'
import * as channelsApi from '@/api/modules/channels'

export const useChannelStore = defineStore('channels', () => {
  const channels = ref<Channel[]>([])

  async function fetchByGroup(groupId?: number) {
    const data = await channelsApi.fetchChannelsByGroup(groupId)
    channels.value = data || []
    return data
  }

  async function create(data: CreateChannelRequest) {
    const result = await channelsApi.createChannel(data)
    channels.value.push(result)
    return result
  }

  async function update(id: number, data: UpdateChannelRequest) {
    const result = await channelsApi.updateChannel(id, data)
    const index = channels.value.findIndex(c => c.id === id)
    if (index !== -1) {
      channels.value[index] = result
    }
    return result
  }

  async function remove(id: number) {
    await channelsApi.deleteChannel(id)
    channels.value = channels.value.filter(c => c.id !== id)
  }

  return {
    channels,
    fetchByGroup,
    create,
    update,
    remove,
  }
})
