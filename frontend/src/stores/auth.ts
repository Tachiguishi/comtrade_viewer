import { defineStore } from 'pinia'
import { login as loginApi, type LoginResponse } from '@/api'

const STORAGE_KEY = 'comtrade.auth'

type PersistedAuth = { token: string; expiresAt: number }

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: '' as string,
    expiresAt: 0 as number,
  }),
  getters: {
    isAuthenticated: (state) => {
      if (!state.token) return false
      if (!state.expiresAt) return true
      return state.expiresAt * 1000 > Date.now()
    },
    bearerToken: (state): string | null => (state.token ? state.token : null),
  },
  actions: {
    restore() {
      const raw = localStorage.getItem(STORAGE_KEY)
      if (!raw) return
      try {
        const parsed = JSON.parse(raw) as PersistedAuth
        this.token = parsed.token
        this.expiresAt = parsed.expiresAt
        if (!this.isAuthenticated) {
          this.logout()
        }
      } catch (err) {
        console.warn('failed to parse auth cache', err)
        this.logout()
      }
    },
    async login(username: string, password: string) {
      const res: LoginResponse = await loginApi({ username, password })
      this.token = res.token
      this.expiresAt = res.expiresAt
      this.persist()
      return res
    },
    logout() {
      this.token = ''
      this.expiresAt = 0
      localStorage.removeItem(STORAGE_KEY)
    },
    persist() {
      const payload: PersistedAuth = { token: this.token, expiresAt: this.expiresAt }
      localStorage.setItem(STORAGE_KEY, JSON.stringify(payload))
    },
    handleUnauthorized() {
      this.logout()
    },
  },
})
