import apiClient, { handleApiResponse } from '@/lib/api-client'
import type { LoginRequest, LoginResponse, UpdatePasswordRequest, UpdateProfileRequest } from '@/types/api'

export function login(data: LoginRequest) {
  return apiClient.post('/v1/auth/login', data).then(res => handleApiResponse<LoginResponse>(res))
}

export function updatePassword(data: UpdatePasswordRequest) {
  return apiClient.put('/v1/users/password', data).then(res => handleApiResponse<null>(res))
}

export function updateProfile(data: UpdateProfileRequest) {
  return apiClient.put('/v1/users/profile', data).then(res => handleApiResponse<null>(res))
}
