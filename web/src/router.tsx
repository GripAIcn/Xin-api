import { Suspense, lazy, type ReactNode } from 'react'
import { LoaderCircle } from 'lucide-react'
import {
  Navigate,
  Outlet,
  createBrowserRouter,
  type RouteObject,
} from 'react-router-dom'

import { Page, publicRoutes } from '@/routes/public'
import { useAuthStore } from '@/stores/auth'

// 控制台侧页面按路由动态加载，避免与官网首页打进同一个首屏包。
// 公开页面必须保持 eager：构建期预渲染使用 renderToString，不支持 lazy 组件。
const AppLayout = lazy(() => import('@/layouts/AppLayout'))
const LoginView = lazy(() => import('@/pages/LoginView'))
const DashboardView = lazy(() => import('@/pages/DashboardView'))
const ApiKeysView = lazy(() => import('@/pages/ApiKeysView'))
const LogsView = lazy(() => import('@/pages/LogsView'))
const BillingView = lazy(() => import('@/pages/BillingView'))
const SettingsView = lazy(() => import('@/pages/SettingsView'))
const ChannelsView = lazy(() => import('@/pages/ChannelsView'))
const ModelsView = lazy(() => import('@/pages/ModelsView'))
const MonitoringView = lazy(() => import('@/pages/MonitoringView'))
const PricingView = lazy(() => import('@/pages/PricingView'))
const UsersView = lazy(() => import('@/pages/UsersView'))

// 懒加载路由的兜底：首次进入控制台时等待对应 chunk 下载。
// 只包裹控制台路由，公开页面保持无 Suspense，保证与预渲染产物结构一致（否则水合不匹配）。
function withFallback(node: ReactNode) {
  return <Suspense fallback={<RouteLoading />}>{node}</Suspense>
}

function RouteLoading() {
  return (
    <div className="flex h-64 items-center justify-center text-muted-foreground">
      <LoaderCircle className="size-5 animate-spin" />
      <span className="sr-only">加载中</span>
    </div>
  )
}

// 检查用户是否已登录（优先使用 store，后备使用 localStorage）
function isAuthenticated(): boolean {
  if (useAuthStore.getState().isAuthenticated) return true
  return !!localStorage.getItem('token')
}

// 需要认证：未登录则跳转登录页
function ProtectedRoute() {
  if (!isAuthenticated()) {
    return <Navigate to="/login" replace />
  }
  return <Outlet />
}

// 访客专属：已登录则跳转仪表盘
function GuestRoute() {
  if (isAuthenticated()) {
    return <Navigate to="/dashboard" replace />
  }
  return <Outlet />
}

// 管理员专属：等待 token 校验完成后再判断角色，避免冷启动误判
function AdminRoute() {
  const isAdmin = useAuthStore((s) => s.isAdmin)
  const isValidating = useAuthStore((s) => s.isValidating)

  if (isValidating) {
    return (
      <div className="flex h-64 items-center justify-center text-muted-foreground">
        <LoaderCircle className="size-5 animate-spin" />
        <span className="sr-only">加载中</span>
      </div>
    )
  }

  if (!isAdmin) {
    return <Navigate to="/dashboard" replace />
  }
  return <Outlet />
}

export const routes: RouteObject[] = [
  ...publicRoutes,
  {
    element: <GuestRoute />,
    children: [
      {
        path: '/login',
        element: (
          <Page title="登录">{withFallback(<LoginView />)}</Page>
        ),
      },
    ],
  },
  {
    path: '/',
    element: <ProtectedRoute />,
    children: [
      {
        element: withFallback(<AppLayout />),
        children: [
          {
            path: 'dashboard',
            element: (
              <Page title="仪表盘">{withFallback(<DashboardView />)}</Page>
            ),
          },
          {
            path: 'apikeys',
            element: (
              <Page title="API Key">{withFallback(<ApiKeysView />)}</Page>
            ),
          },
          {
            path: 'logs',
            element: <Page title="日志">{withFallback(<LogsView />)}</Page>,
          },
          {
            path: 'billing',
            element: (
              <Page title="账单日志">{withFallback(<BillingView />)}</Page>
            ),
          },
          {
            path: 'settings',
            element: (
              <Page title="账户与余额">{withFallback(<SettingsView />)}</Page>
            ),
          },
          {
            element: <AdminRoute />,
            children: [
              {
                path: 'channels',
                element: (
                  <Page title="渠道管理">{withFallback(<ChannelsView />)}</Page>
                ),
              },
              {
                path: 'models',
                element: (
                  <Page title="模型管理">{withFallback(<ModelsView />)}</Page>
                ),
              },
              {
                path: 'monitoring',
                element: (
                  <Page title="渠道监控">
                    {withFallback(<MonitoringView />)}
                  </Page>
                ),
              },
              {
                path: 'pricing',
                element: (
                  <Page title="定价管理">{withFallback(<PricingView />)}</Page>
                ),
              },
              {
                path: 'users',
                element: (
                  <Page title="用户管理">{withFallback(<UsersView />)}</Page>
                ),
              },
            ],
          },
        ],
      },
    ],
  },
]

export const router = createBrowserRouter(routes)

export default router