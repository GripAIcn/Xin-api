import axios, { AxiosError, type AxiosResponse } from 'axios'
import type { AdminResponse } from '@/types/api'

// import.meta.env 是 Vite 注入的；SSR 预渲染脚本（tsx 直接执行）中为 undefined，
// 用可选链兜底为 '/'
const apiClient = axios.create({
  baseURL: import.meta.env?.VITE_API_URL || '/',
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

function clearStoredAuth(): void {
  localStorage.removeItem('token')
  localStorage.removeItem('username')
}

apiClient.interceptors.response.use(
  (response: AxiosResponse<AdminResponse<unknown>>) => {
    const data = response.data
    // 后端业务状态码：20000 表示成功
    if (data.code !== undefined && data.code !== 20000) {
      return Promise.reject(new Error(data.message || 'Request failed'))
    }
    return response
  },
  (error: AxiosError<AdminResponse<unknown>>) => {
    // 401 错误：token 无效或过期，清除登录态
    if (error.response?.status === 401) {
      clearStoredAuth()
      if (window.location.pathname !== '/login') {
        window.location.assign('/login')
      }
    }

    // 网络错误（后端掉线）：非登录页时清除登录态
    if (
      !error.response &&
      (error.code === 'ERR_NETWORK' || error.code === 'ECONNABORTED')
    ) {
      if (window.location.pathname !== '/login') {
        clearStoredAuth()
      }
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
  // 后端业务状态码：20000 表示成功
  if (code !== 20000) {
    throw new Error(message || 'Request failed')
  }
  return data as T
}

export default apiClient