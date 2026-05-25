import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import AppLayout from '@/layouts/AppLayout.vue'

// 路由元信息类型扩展
declare module 'vue-router' {
  interface RouteMeta {
    guest?: boolean
    requiresAuth?: boolean
    title?: string
  }
}

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  scrollBehavior() {
    return { top: 0 }
  },
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { guest: true, title: '登录' },
    },
    {
      path: '/',
      component: AppLayout,
      meta: { requiresAuth: true },
      redirect: '/dashboard',
      children: [
        {
          path: 'dashboard',
          name: 'dashboard',
          component: () => import('@/views/DashboardView.vue'),
          meta: { title: '仪表盘' },
        },
        {
          path: 'groups',
          name: 'groups',
          component: () => import('@/views/GroupsView.vue'),
          meta: { title: '项目组' },
        },
        {
          path: 'groups/:id',
          name: 'group-detail',
          component: () => import('@/views/GroupDetailView.vue'),
          meta: { title: '项目组详情' },
        },
        {
          path: 'channels',
          name: 'channels',
          component: () => import('@/views/ChannelsView.vue'),
          meta: { title: '渠道管理' },
        },
        {
          path: 'apikeys',
          name: 'api-keys',
          component: () => import('@/views/ApiKeysView.vue'),
          meta: { title: 'API Key' },
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => import('@/views/SettingsView.vue'),
          meta: { title: '账户设置' },
        },
      ],
    },
  ],
})

// 检查用户是否已登录（优先使用 store，后备使用 localStorage）
function isAuthenticated(): boolean {
  const authStore = useAuthStore()
  if (authStore.isAuthenticated) {
    return true
  }
  // 后备检查：直接从 localStorage 检查
  const token = localStorage.getItem('token')
  return !!token
}

// 使用新的导航守卫 API，返回路径而不是调用 next()
router.beforeEach((to) => {
  const authenticated = isAuthenticated()

  // 需要认证但未登录
  if (to.meta.requiresAuth && !authenticated) {
    return '/login'
  }

  // 已登录但访问登录页
  if (to.meta.guest && authenticated) {
    return '/dashboard'
  }

  // 返回 true 或 undefined 表示继续导航
  return true
})

// 设置页面标题
router.afterEach((to) => {
  const title = to.meta.title
  if (title) {
    document.title = `${title} - Xin-api`
  } else {
    document.title = 'Xin-api 控制面板'
  }
})

export default router
