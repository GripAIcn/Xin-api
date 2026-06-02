import apiClient, { handleApiResponse } from '../client'
import type { Channel, CreateChannelRequest, UpdateChannelRequest, TestChannelRequest, ChannelTestResult } from '@/types/api'

export function fetchChannelsByGroup(groupId?: number) {
  const url = groupId ? `/v1/channels?group_id=${groupId}` : '/v1/channels'
  return apiClient.get(url).then(res => handleApiResponse<Channel[]>(res))
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

// 测试项目组所有渠道
export function testGroupChannels(groupId: number) {
  return apiClient.post(`/v1/groups/${groupId}/test-channels`).then(res => handleApiResponse<ChannelTestResult[]>(res))
}
