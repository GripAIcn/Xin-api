import apiClient, { handleApiResponse } from '@/lib/api-client'
import type { Channel, CreateChannelRequest, UpdateChannelRequest, TestChannelRequest, ChannelTestResult } from '@/types/api'

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
