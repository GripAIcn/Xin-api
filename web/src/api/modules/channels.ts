import apiClient, { handleApiResponse } from '@/lib/api-client'
import type {
  Channel,
  ChannelModel,
  CreateChannelModelRequest,
  CreateChannelRequest,
  TestChannelRequest,
  UpdateChannelModelRequest,
  UpdateChannelModelStatusRequest,
  UpdateChannelRequest,
  ChannelTestResult,
} from '@/types/api'

export function fetchChannels() {
  return apiClient.get('/v1/channels').then(res => handleApiResponse<Channel[]>(res))
}

export function createChannel(data: CreateChannelRequest) {
  return apiClient.post('/v1/channels', data).then(res => handleApiResponse<Channel>(res))
}

export function updateChannel(id: number, data: UpdateChannelRequest) {
  return apiClient.put(`/v1/channels/${id}`, data).then(res => handleApiResponse<Channel>(res))
}

export function deleteChannel(id: number) {
  return apiClient.delete(`/v1/channels/${id}`).then(res => handleApiResponse<null>(res))
}

// 测试单个渠道
export function testChannel(id: number, data?: TestChannelRequest) {
  return apiClient.post(`/v1/channels/${id}/test`, data || {}).then(res => handleApiResponse<ChannelTestResult>(res))
}

// 测试所有渠道
export function testAllChannels() {
  return apiClient.post('/v1/channels/test', {}).then(res => handleApiResponse<ChannelTestResult[]>(res))
}

// === 渠道模型管理 ===

// 获取渠道全部模型（含 disabled）
export function fetchChannelModels(channelId: number) {
  return apiClient.get(`/v1/channels/${channelId}/models`).then(res => handleApiResponse<ChannelModel[]>(res))
}

// 为渠道新增模型
export function createChannelModel(channelId: number, data: CreateChannelModelRequest) {
  return apiClient.post(`/v1/channels/${channelId}/models`, data).then(res => handleApiResponse<ChannelModel>(res))
}

// 编辑渠道模型
export function updateChannelModel(channelId: number, modelId: number, data: UpdateChannelModelRequest) {
  return apiClient.put(`/v1/channels/${channelId}/models/${modelId}`, data).then(res => handleApiResponse<ChannelModel>(res))
}

// 软删除渠道模型
export function deleteChannelModel(channelId: number, modelId: number) {
  return apiClient.delete(`/v1/channels/${channelId}/models/${modelId}`).then(res => handleApiResponse<null>(res))
}

// 启用/禁用渠道模型
export function updateChannelModelStatus(channelId: number, modelId: number, data: UpdateChannelModelStatusRequest) {
  return apiClient.put(`/v1/channels/${channelId}/models/${modelId}/status`, data).then(res => handleApiResponse<ChannelModel>(res))
}

// 拉取上游可用模型 id 列表
export function fetchUpstreamModels(channelId: number) {
  return apiClient.get(`/v1/channels/${channelId}/models/available`).then(res => handleApiResponse<string[]>(res))
}
