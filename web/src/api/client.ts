import axios, { AxiosError, type AxiosResponse } from 'axios'
import type { AdminResponse } from '@/types/api'

const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/',
  timeout: 30000,
  headers: { 'Content-Type': 'application/json' },
})

apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

apiClient.interceptors.response.use(
  (response: AxiosResponse<AdminResponse<unknown>>) => {
    const data = response.data
    if (data.code !== undefined && data.code !== 20000) {
      return Promise.reject(new Error(data.message || 'Request failed'))
    }
    return response
  },
  (error: AxiosError<AdminResponse<unknown>>) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('username')
      window.location.href = '/login'
    }
    const message =
      error.response?.data?.message || error.message || 'Network error'
    return Promise.reject(new Error(message))
  },
)

export function handleApiResponse<T>(response: {
  data: { code: number; message: string; data?: T }
}): T {
  const { code, message, data } = response.data
  if (code !== 20000) {
    throw new Error(message || 'Request failed')
  }
  return data as T
}

export default apiClient
