import apiClient, { handleApiResponse } from '@/api/client'
import type {
  Group,
  CreateGroupRequest,
  UpdateGroupRequest,
  UpdateGroupStatusRequest,
} from '@/types/api'

export async function listGroups(): Promise<Group[]> {
  const response = await apiClient.get('/v1/groups')
  return handleApiResponse<Group[]>(response)
}

export async function createGroup(
  data: CreateGroupRequest,
): Promise<Group> {
  const response = await apiClient.post('/v1/groups', data)
  return handleApiResponse<Group>(response)
}

export async function updateGroup(
  id: number,
  data: UpdateGroupRequest,
): Promise<Group> {
  const response = await apiClient.put(`/v1/groups/${id}`, data)
  return handleApiResponse<Group>(response)
}

export async function updateGroupStatus(
  id: number,
  data: UpdateGroupStatusRequest,
): Promise<Group> {
  const response = await apiClient.put(`/v1/groups/${id}/status`, data)
  return handleApiResponse<Group>(response)
}

export async function deleteGroup(id: number): Promise<void> {
  const response = await apiClient.delete(`/v1/groups/${id}`)
  handleApiResponse(response)
}
