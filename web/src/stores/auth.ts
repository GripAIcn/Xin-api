import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import type { LoginResponse } from '@/types/api'
import * as authApi from '@/api/modules/auth'
import apiClient from '@/api/client'
import router from '@/router'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem('token'))
  const username = ref<string | null>(localStorage.getItem('username'))
  const isValidating = ref(false)

  const isAuthenticated = computed<boolean>(() => !!token.value)

  async function login(usernameVal: string, password: string) {
    const res: LoginResponse = await authApi.login({
      username: usernameVal,
      password,
    })
    token.value = res.token
    username.value = res.username
    localStorage.setItem('token', res.token)
    localStorage.setItem('username', res.username)
  }

  function logout() {
    token.value = null
    username.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('username')
    router.push('/login')
  }

  function updateUsername(newUsername: string) {
    username.value = newUsername
    localStorage.setItem('username', newUsername)
  }

  function recoverFromStorage() {
    const storedToken = localStorage.getItem('token')
    const storedUsername = localStorage.getItem('username')
    if (storedToken && storedUsername) {
      token.value = storedToken
      username.value = storedUsername
    }
  }

  async function updatePassword(data: { old_password: string; new_password: string }) {
    await authApi.updatePassword(data)
  }

  async function updateAccount(data: { username: string }) {
    await authApi.updateAccount(data)
    username.value = data.username
    localStorage.setItem('username', data.username)
  }

  // 验证 token 是否有效（调用需要认证的 API）
  async function validateToken(): Promise<boolean> {
    // 如果没有 token，直接返回 false
    if (!token.value) {
      return false
    }

    isValidating.value = true
    try {
      // 调用需要认证的 API 来验证 token 是否有效
      // 使用 GET /v1/groups 作为验证接口
      await apiClient.get('/v1/groups')
      return true
    } catch (error) {
      // 验证失败（包括网络错误、401 等），清除 token 并跳转到登录页
      logout()
      return false
    } finally {
      isValidating.value = false
    }
  }

  return {
    token,
    username,
    isAuthenticated,
    isValidating,
    login,
    logout,
    updateUsername,
    updatePassword,
    updateAccount,
    recoverFromStorage,
    validateToken,
  }
})
