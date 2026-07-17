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
      path: '/read/:id',
      name: 'reader',
      component: () => import('@/views/Reader.vue'),
    },
  ],
})

export default router