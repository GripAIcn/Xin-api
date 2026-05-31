import apiClient, { handleApiResponse } from '../client'
import type { Group, CreateGroupRequest, UpdateGroupRequest, UpdateGroupStatusRequest } from '@/types/api'

export function fetchGroups() {
  return apiClient.get('/v1/groups').then(res => handleApiResponse<Group[]>(res))
}

export function createGroup(data: CreateGroupRequest) {
  return apiClient.post('/v1/groups', data).then(res => handleApiResponse<Group>(res))
}

export function updateGroup(id: number, data: UpdateGroupRequest) {
  return apiClient.put(`/v1/groups/${id}`, data).then(res => handleApiResponse<Group>(res))
}

export function updateGroupStatus(id: number, data: UpdateGroupStatusRequest) {
  return apiClient.put(`/v1/groups/${id}/status`, data).then(res => handleApiResponse<Group>(res))
}

export function deleteGroup(id: number) {
  return apiClient.delete(`/v1/groups/${id}`).then(res => handleApiResponse<null>(res))
}
