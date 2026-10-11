import { create } from 'zustand'
import type { LoginResponse } from '@/types/api'
import * as authApi from '@/api/modules/auth'
import apiClient from '@/lib/api-client'

interface AuthState {
  token: string | null
  username: string | null
  isValidating: boolean
  isAuthenticated: boolean
  login: (username: string, password: string) => Promise<void>
  logout: () => void
  updateUsername: (username: string) => void
  updatePassword: (data: {
    old_password: string
    new_password: string
  }) => Promise<void>
  updateAccount: (data: { username: string }) => Promise<void>
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
  isValidating: false,
  isAuthenticated: !!readStoredAuth('token'),

  login: async (usernameVal, password) => {
    const res: LoginResponse = await authApi.login({
      username: usernameVal,
      password,
    })
    localStorage.setItem('token', res.token)
    localStorage.setItem('username', res.username)
    set({ token: res.token, username: res.username, isAuthenticated: true })
  },

  logout: () => {
    localStorage.removeItem('token')
    localStorage.removeItem('username')
    set({ token: null, username: null, isAuthenticated: false })
    if (window.location.pathname !== '/login') {
      window.location.assign('/login')
    }
  },

  updateUsername: (newUsername) => {
    localStorage.setItem('username', newUsername)
    set({ username: newUsername })
  },

  updatePassword: async (data) => {
    await authApi.updatePassword(data)
  },

  updateAccount: async (data) => {
    await authApi.updateAccount(data)
    localStorage.setItem('username', data.username)
    set({ username: data.username })
  },

  recoverFromStorage: () => {
    const storedToken = localStorage.getItem('token')
    const storedUsername = localStorage.getItem('username')
    if (storedToken && storedUsername) {
      set({
        token: storedToken,
        username: storedUsername,
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