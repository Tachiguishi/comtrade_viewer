import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './assets/main.css'
import { useAuthStore } from './stores/auth'
import { setupApiAuth } from './api'

const app = createApp(App)
const pinia = createPinia()

const authStore = useAuthStore(pinia)
authStore.restore()

setupApiAuth(
  () => authStore.bearerToken,
  () => {
    authStore.handleUnauthorized()
    const current = router.currentRoute.value
    if (current.name !== 'login') {
      router.push({ name: 'login', query: { redirect: current.fullPath } })
    }
  },
)

app.use(pinia)
app.use(router)
app.mount('#app')
