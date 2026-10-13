import { create } from 'zustand'
import type {
  User,
  CreateUserRequest,
  UpdateUserStatusRequest,
  AdminResetPasswordRequest,
  RechargeBalanceRequest,
} from '@/types/api'
import * as usersApi from '@/api/modules/users'

interface UsersState {
  users: User[]
  loading: boolean
  fetchAll: () => Promise<void>
  create: (data: CreateUserRequest) => Promise<void>
  updateStatus: (id: number, data: UpdateUserStatusRequest) => Promise<void>
  resetPassword: (id: number, data: AdminResetPasswordRequest) => Promise<void>
  recharge: (id: number, data: RechargeBalanceRequest) => Promise<void>
}

export const useUsersStore = create<UsersState>()((set, get) => ({
  users: [],
  loading: false,

  fetchAll: async () => {
    set({ loading: true })
    try {
      const users = await usersApi.fetchUsers()
      set({ users, loading: false })
    } catch (error) {
      set({ loading: false })
      throw error
    }
  },

  create: async (data) => {
    await usersApi.createUser(data)
    await get().fetchAll()
  },

  updateStatus: async (id, data) => {
    await usersApi.updateUserStatus(id, data)
    await get().fetchAll()
  },

  resetPassword: async (id, data) => {
    await usersApi.resetUserPassword(id, data)
  },

  recharge: async (id, data) => {
    await usersApi.rechargeBalance(id, data)
    await get().fetchAll()
  },
}))
