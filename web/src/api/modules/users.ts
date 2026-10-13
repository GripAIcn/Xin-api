import apiClient, { handleApiResponse } from '@/lib/api-client'
import type {
  User,
  CreateUserRequest,
  UpdateUserStatusRequest,
  AdminResetPasswordRequest,
  RechargeBalanceRequest,
} from '@/types/api'

// 管理员：获取用户列表
export function fetchUsers() {
  return apiClient.get('/v1/users').then(res => handleApiResponse<User[]>(res))
}

// 管理员：新建用户
export function createUser(data: CreateUserRequest) {
  return apiClient.post('/v1/users', data).then(res => handleApiResponse<null>(res))
}

// 管理员：启用/禁用用户
export function updateUserStatus(id: number, data: UpdateUserStatusRequest) {
  return apiClient.put(`/v1/users/${id}/status`, data).then(res => handleApiResponse<null>(res))
}

// 管理员：重置用户密码
export function resetUserPassword(id: number, data: AdminResetPasswordRequest) {
  return apiClient.post(`/v1/users/${id}/reset-password`, data).then(res => handleApiResponse<null>(res))
}

// 管理员：为用户充值（仅增加余额）
export function rechargeBalance(id: number, data: RechargeBalanceRequest) {
  return apiClient.post(`/v1/users/${id}/recharge`, data).then(res => handleApiResponse<null>(res))
}
