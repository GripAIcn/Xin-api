import { create } from 'zustand'
import type {
  Channel,
  ChannelModel,
  ChannelTestResult,
  CreateChannelModelRequest,
  CreateChannelRequest,
  UpdateChannelModelRequest,
  UpdateChannelRequest,
} from '@/types/api'
import * as channelsApi from '@/api/modules/channels'

interface ChannelState {
  channels: Channel[]
  // 当前对话框正在查看的渠道模型列表
  models: ChannelModel[]
  modelsChannelId: number | null
  fetchAll: () => Promise<Channel[]>
  create: (data: CreateChannelRequest) => Promise<Channel>
  update: (id: number, data: UpdateChannelRequest) => Promise<Channel>
  remove: (id: number) => Promise<void>
  testChannel: (id: number, model?: string) => Promise<ChannelTestResult>
  testAllChannels: () => Promise<ChannelTestResult[]>
  fetchModels: (channelId: number) => Promise<ChannelModel[]>
  createModel: (
    channelId: number,
    data: CreateChannelModelRequest,
  ) => Promise<ChannelModel>
  updateModel: (
    channelId: number,
    modelId: number,
    data: UpdateChannelModelRequest,
  ) => Promise<ChannelModel>
  removeModel: (channelId: number, modelId: number) => Promise<void>
  updateModelStatus: (
    channelId: number,
    modelId: number,
    status: ChannelModel['status'],
  ) => Promise<void>
  fetchUpstreamModels: (channelId: number) => Promise<string[]>
}

export const useChannelStore = create<ChannelState>()((set) => ({
  channels: [],
  models: [],
  modelsChannelId: null,

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

  // 加载指定渠道的全部模型
  fetchModels: async (channelId) => {
    const data = await channelsApi.fetchChannelModels(channelId)
    set({ models: data || [], modelsChannelId: channelId })
    return data
  },

  createModel: async (channelId, data) => {
    const result = await channelsApi.createChannelModel(channelId, data)
    set((state) => ({ models: [...state.models, result] }))
    return result
  },

  updateModel: async (channelId, modelId, data) => {
    const result = await channelsApi.updateChannelModel(channelId, modelId, data)
    set((state) => ({
      models: state.models.map((m) => (m.id === modelId ? result : m)),
    }))
    return result
  },

  removeModel: async (channelId, modelId) => {
    await channelsApi.deleteChannelModel(channelId, modelId)
    set((state) => ({ models: state.models.filter((m) => m.id !== modelId) }))
  },

  updateModelStatus: async (channelId, modelId, status) => {
    const result = await channelsApi.updateChannelModelStatus(channelId, modelId, {
      status,
    })
    set((state) => ({
      models: state.models.map((m) =>
        m.id === modelId ? (result ?? { ...m, status }) : m,
      ),
    }))
  },

  fetchUpstreamModels: (channelId) => {
    return channelsApi.fetchUpstreamModels(channelId)
  },
}))
