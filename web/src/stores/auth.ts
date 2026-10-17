import { create } from 'zustand'
import type { LoginResponse, UpdateProfileRequest, User } from '@/types/api'
import * as authApi from '@/api/modules/auth'
import apiClient, { handleApiResponse } from '@/lib/api-client'

interface AuthState {
  token: string | null
  username: string | null
  role: string | null
  profile: User | null
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
  fetchProfile: () => Promise<User | null>
  validateToken: () => Promise<boolean>
}

// SSR 安全：构建期预渲染（非浏览器环境）无 localStorage，回退为 null；
// 浏览器环境下行为与直接读取完全一致，不影响水合
function readStoredAuth(key: string): string | null {
  return typeof localStorage !== 'undefined' ? localStorage.getItem(key) : null
}

const storedToken = readStoredAuth('token')

export const useAuthStore = create<AuthState>()((set, get) => ({
  token: storedToken,
  username: readStoredAuth('username'),
  role: readStoredAuth('role'),
  profile: null,
  isAdmin: readStoredAuth('role') === 'admin',
  // 冷启动带 token 时先标记为校验中，路由需等待校验完成再判断角色
  isValidating: !!storedToken,
  isAuthenticated: !!storedToken,

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
      profile: null,
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

  // 拉取当前登录用户资料（同时用于 token 校验）；失败返回 null 而不抛出
  fetchProfile: async () => {
    try {
      const res = await apiClient.get('/v1/users/profile')
      const profile = handleApiResponse<User>(res)
      set({ profile })
      return profile
    } catch {
      return null
    }
  },

  // 验证 token 是否有效（调用需要认证的 API）
  validateToken: async () => {
    if (!get().token) {
      return false
    }

    set({ isValidating: true })
    try {
      const profile = await get().fetchProfile()
      if (profile) {
        // 以服务端返回的角色为准，避免本地缓存的角色过期
        const role = profile.role
        localStorage.setItem('role', role)
        set({ role, isAdmin: role === 'admin' })
        return true
      }
      // 验证失败（包括网络错误、401 等），清除 token 并跳转到登录页
      get().logout()
      return false
    } finally {
      set({ isValidating: false })
    }
  },
}))