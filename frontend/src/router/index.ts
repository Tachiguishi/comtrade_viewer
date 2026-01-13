import { createRouter, createWebHistory } from 'vue-router'
import UploadView from '../views/UploadView.vue'
import ViewerView from '../views/ViewerView.vue'

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
      path: '/viewer',
      name: 'viewer',
      component: ViewerView,
    },
  ],
})

export default router
