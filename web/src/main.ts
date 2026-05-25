import './assets/main.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from './App.vue'
import router from './router'
import { useAuthStore } from '@/stores/auth'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)

// 在挂载应用前恢复认证状态
const authStore = useAuthStore()
authStore.recoverFromStorage()

app.use(router)

app.mount('#app')
