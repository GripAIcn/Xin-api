import { create } from 'zustand'
import type { LoginResponse, UpdateProfileRequest } from '@/types/api'
import * as authApi from '@/api/modules/auth'
import apiClient from '@/lib/api-client'

interface AuthState {
  token: string | null
  username: string | null
  role: string | null
  isAdmin: boolean
  isValidating: boolean
  isAuthenticated: boolean
  login: (username: string, password: string) => Promise<void>
  logout: () => void
  updatePassword: (data: {
    old_password: string
    new_password: string
  }) => Promise<void>
  updateProfile: (data: UpdateProfileRequest) => Promise<void>
  recoverFromStorage: () => void
  validateToken: () => Promise<boolean>
}

// SSR 安全：构建期预渲染（非浏览器环境）无 localStorage，回退为 null；
// 浏览器环境下行为与直接读取完全一致，不影响水合
function readStoredAuth(key: string): string | null {
  return typeof localStorage !== 'undefined' ? localStorage.getItem(key) : null
}

export const useAuthStore = create<AuthState>()((set, get) => ({
  token: readStoredAuth('token'),
  username: readStoredAuth('username'),
  role: readStoredAuth('role'),
  isAdmin: readStoredAuth('role') === 'admin',
  isValidating: false,
  isAuthenticated: !!readStoredAuth('token'),

  login: async (usernameVal, password) => {
    const res: LoginResponse = await authApi.login({
      username: usernameVal,
      password,
    })
    localStorage.setItem('token', res.token)
    localStorage.setItem('username', res.username)
    localStorage.setItem('role', res.role)
    set({
      token: res.token,
      username: res.username,
      role: res.role,
      isAdmin: res.role === 'admin',
      isAuthenticated: true,
    })
  },

  logout: () => {
    localStorage.removeItem('token')
    localStorage.removeItem('username')
    localStorage.removeItem('role')
    set({
      token: null,
      username: null,
      role: null,
      isAdmin: false,
      isAuthenticated: false,
    })
    if (window.location.pathname !== '/login') {
      window.location.assign('/login')
    }
  },

  updatePassword: async (data) => {
    await authApi.updatePassword(data)
  },

  updateProfile: async (data) => {
    await authApi.updateProfile(data)
  },

  recoverFromStorage: () => {
    const storedToken = localStorage.getItem('token')
    const storedUsername = localStorage.getItem('username')
    if (storedToken && storedUsername) {
      const storedRole = localStorage.getItem('role')
      set({
        token: storedToken,
        username: storedUsername,
        role: storedRole,
        isAdmin: storedRole === 'admin',
        isAuthenticated: true,
      })
    }
  },

  // 验证 token 是否有效（调用需要认证的 API）
  validateToken: async () => {
    if (!get().token) {
      return false
    }

    set({ isValidating: true })
    try {
      // 使用 GET /v1/channels 作为验证接口
      await apiClient.get('/v1/channels')
      return true
    } catch {
      // 验证失败（包括网络错误、401 等），清除 token 并跳转到登录页
      get().logout()
      return false
    } finally {
      set({ isValidating: false })
    }
  },
}))