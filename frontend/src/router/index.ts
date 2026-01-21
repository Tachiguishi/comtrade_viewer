import { createRouter, createWebHistory } from 'vue-router'
import UploadView from '../views/UploadView.vue'
import ViewerView from '../views/ViewerView.vue'
import WaveCanvasView from '../views/CanvasView.vue'
import LoginView from '../views/LoginView.vue'
import { useAuthStore } from '../stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      redirect: '/upload',
    },
    {
      path: '/upload',
      name: 'upload',
      component: UploadView,
    },
    {
      path: '/login',
      name: 'login',
      component: LoginView,
    },
    {
      path: '/viewer',
      name: 'viewer',
      component: ViewerView,
    },
    {
      path: '/canvas',
      name: 'canvas',
      component: WaveCanvasView,
    },
  ],
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  if (to.name !== 'login' && !auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.name === 'login' && auth.isAuthenticated) {
    const redirect = (to.query.redirect as string) || '/upload'
    return { path: redirect }
  }
  return true
})

export default router
