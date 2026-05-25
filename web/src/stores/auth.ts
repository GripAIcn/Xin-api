import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import type { LoginResponse } from '@/types/api'
import * as authApi from '@/api/modules/auth'
import router from '@/router'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem('token'))
  const username = ref<string | null>(localStorage.getItem('username'))

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

  return {
    token,
    username,
    isAuthenticated,
    login,
    logout,
    updateUsername,
    updatePassword,
    updateAccount,
    recoverFromStorage,
  }
})
