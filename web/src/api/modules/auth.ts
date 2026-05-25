import apiClient, { handleApiResponse } from '@/api/client'
import type {
  LoginRequest,
  LoginResponse,
  UpdatePasswordRequest,
  UpdateAccountRequest,
} from '@/types/api'

export async function login(data: LoginRequest): Promise<LoginResponse> {
  const response = await apiClient.post('/v1/auth/login', data)
  return handleApiResponse<LoginResponse>(response)
}

export async function updatePassword(
  data: UpdatePasswordRequest,
): Promise<void> {
  const response = await apiClient.put('/v1/users/password', data)
  handleApiResponse(response)
}

export async function updateAccount(data: UpdateAccountRequest): Promise<void> {
  const response = await apiClient.put('/v1/users/account', data)
  handleApiResponse(response)
}
