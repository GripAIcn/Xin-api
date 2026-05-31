import apiClient, { handleApiResponse } from '../client'
import type { LoginRequest, LoginResponse, UpdatePasswordRequest, UpdateAccountRequest } from '@/types/api'

export function login(data: LoginRequest) {
  return apiClient.post('/v1/auth/login', data).then(res => handleApiResponse<LoginResponse>(res))
}

export function updatePassword(data: UpdatePasswordRequest) {
  return apiClient.put('/v1/users/password', data).then(res => handleApiResponse<null>(res))
}

export function updateAccount(data: UpdateAccountRequest) {
  return apiClient.put('/v1/users/account', data).then(res => handleApiResponse<null>(res))
}
