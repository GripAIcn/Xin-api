import axios, { AxiosError, type AxiosResponse } from 'axios'
import type { AdminResponse } from '@/types/api'
import { useAuthStore } from '@/stores/auth'
import router from '@/router'

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
    // 后端业务状态码：20000 表示成功
    if (data.code !== undefined && data.code !== 20000) {
      return Promise.reject(new Error(data.message || 'Request failed'))
    }
    return response
  },
  (error: AxiosError<AdminResponse<unknown>>) => {
    // 401 错误：token 无效或过期
    if (error.response?.status === 401) {
      const authStore = useAuthStore()
      authStore.logout()
    }
    
    // 网络错误（后端掉线）：如果当前在需要认证的页面，则跳转到登录页
    if (!error.response && error.code === 'ERR_NETWORK' || error.code === 'ECONNABORTED') {
      // 检查当前路由是否需要认证
      const currentPath = router.currentRoute.value.path
      const requiresAuth = router.currentRoute.value.meta.requiresAuth
      const isLoginPage = currentPath === '/login'
      
      // 如果在需要认证的页面（非登录页），则清除状态并跳转
      if (requiresAuth && !isLoginPage) {
        const authStore = useAuthStore()
        authStore.logout()
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
