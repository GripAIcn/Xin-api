import apiClient, { handleApiResponse } from '@/api/client'
import type {
  Channel,
  CreateChannelRequest,
  UpdateChannelRequest,
} from '@/types/api'

export async function listChannelsByGroup(
  groupId?: number,
): Promise<Channel[]> {
  // 如果有 groupId，作为查询参数传递
  if (groupId !== undefined && groupId !== null && groupId > 0) {
    const response = await apiClient.get('/v1/channels', {
      params: { group_id: groupId }
    })
    return handleApiResponse<Channel[]>(response)
  }
  // 没有 groupId 时，不传递任何参数
  const response = await apiClient.get('/v1/channels')
  return handleApiResponse<Channel[]>(response)
}

export async function createChannel(
  data: CreateChannelRequest,
): Promise<Channel> {
  const response = await apiClient.post('/v1/channels', data)
  return handleApiResponse<Channel>(response)
}

export async function updateChannel(
  id: number,
  data: UpdateChannelRequest,
): Promise<Channel> {
  const response = await apiClient.put(`/v1/channels/${id}`, data)
  return handleApiResponse<Channel>(response)
}

export async function deleteChannel(id: number): Promise<void> {
  const response = await apiClient.delete(`/v1/channels/${id}`)
  handleApiResponse(response)
}
