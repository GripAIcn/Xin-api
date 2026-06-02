import { ref } from 'vue'
import { defineStore } from 'pinia'
import type { Channel, CreateChannelRequest, UpdateChannelRequest, ChannelTestResult } from '@/types/api'
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

  // 测试单个渠道
  async function testChannel(channelId: number, model?: string) {
    const result = await channelsApi.testChannel(channelId, model ? { model } : undefined)
    return result
  }

  // 测试项目组所有渠道
  async function testGroupChannels(groupId: number) {
    const results = await channelsApi.testGroupChannels(groupId)
    return results
  }

  return {
    channels,
    fetchByGroup,
    create,
    update,
    remove,
    testChannel,
    testGroupChannels,
  }
})
