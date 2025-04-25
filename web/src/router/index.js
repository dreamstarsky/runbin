import {
  createRouter,
  createWebHistory
} from 'vue-router'
import CodePage from '../pages/CodePage.vue'
const routes = [{
  path: '/code/:id?',
  name: 'code',
  component: CodePage,
  props: true
}, {
  path: '/',
  redirect: '/code',
}]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router