import { createRouter, createWebHistory } from 'vue-router'
import BookShelf from '@/views/BookShelf.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'shelf',
      component: BookShelf,
    },
    {
      path: '/read',
      name: 'reader',
      component: () => import('@/views/Reader.vue'),
    },
    {
      path: '/read-complete',
      name: 'readComplete',
      component: () => import('@/views/ReadComplete.vue'),
    },
    {
      path: '/settings',
      name: 'settings',
      component: () => import('@/views/Settings.vue'),
    },
  ],
})

export default router
