import { create } from 'zustand'
import type {
  Channel,
  ChannelTestResult,
  CreateChannelRequest,
  UpdateChannelRequest,
} from '@/types/api'
import * as channelsApi from '@/api/modules/channels'

interface ChannelState {
  channels: Channel[]
  fetchByGroup: (groupId?: number) => Promise<Channel[]>
  create: (data: CreateChannelRequest) => Promise<Channel>
  update: (id: number, data: UpdateChannelRequest) => Promise<Channel>
  remove: (id: number) => Promise<void>
  testChannel: (id: number, model?: string) => Promise<ChannelTestResult>
  testGroupChannels: (groupId: number) => Promise<ChannelTestResult[]>
}

export const useChannelStore = create<ChannelState>()((set) => ({
  channels: [],

  fetchByGroup: async (groupId) => {
    const data = await channelsApi.fetchChannelsByGroup(groupId)
    set({ channels: data || [] })
    return data
  },

  create: async (data) => {
    const result = await channelsApi.createChannel(data)
    set((state) => ({ channels: [...state.channels, result] }))
    return result
  },

  update: async (id, data) => {
    const result = await channelsApi.updateChannel(id, data)
    set((state) => ({
      channels: state.channels.map((c) => (c.id === id ? result : c)),
    }))
    return result
  },

  remove: async (id) => {
    await channelsApi.deleteChannel(id)
    set((state) => ({ channels: state.channels.filter((c) => c.id !== id) }))
  },

  // 测试单个渠道
  testChannel: async (channelId, model) => {
    return channelsApi.testChannel(channelId, model ? { model } : undefined)
  },

  // 测试项目组所有渠道
  testGroupChannels: async (groupId) => {
    return channelsApi.testGroupChannels(groupId)
  },
}))