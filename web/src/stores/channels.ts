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
  fetchAll: () => Promise<Channel[]>
  create: (data: CreateChannelRequest) => Promise<Channel>
  update: (id: number, data: UpdateChannelRequest) => Promise<Channel>
  remove: (id: number) => Promise<void>
  testChannel: (id: number, model?: string) => Promise<ChannelTestResult>
  testAllChannels: () => Promise<ChannelTestResult[]>
}

export const useChannelStore = create<ChannelState>()((set) => ({
  channels: [],

  fetchAll: async () => {
    const data = await channelsApi.fetchChannels()
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

  // 测试所有渠道
  testAllChannels: async () => {
    return channelsApi.testAllChannels()
  },
}))