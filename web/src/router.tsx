import { useEffect, type ReactNode } from 'react'
import { Navigate, Outlet, createBrowserRouter } from 'react-router-dom'

import AppLayout from '@/layouts/AppLayout'
import ApiKeysView from '@/pages/ApiKeysView'
import ChannelsView from '@/pages/ChannelsView'
import DashboardView from '@/pages/DashboardView'
import LandingView from '@/pages/LandingView'
import LoginView from '@/pages/LoginView'
import ModelsView from '@/pages/ModelsView'
import SettingsView from '@/pages/SettingsView'
import UsersView from '@/pages/UsersView'
import { useAuthStore } from '@/stores/auth'

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

// 管理员专属：非管理员则跳转仪表盘
function AdminRoute() {
  if (!useAuthStore.getState().isAdmin) {
    return <Navigate to="/dashboard" replace />
  }
  return <Outlet />
}

// 渲染时设置 document.title
function Page({ title, children }: { title?: string; children: ReactNode }) {
  useEffect(() => {
    document.title = title
      ? `${title} - Xin-api`
      : 'Xin API - 大模型 API 统一管理、全链路可观测与智能分发网关'
  }, [title])
  return <>{children}</>
}

export const router = createBrowserRouter([
  {
    path: '/',
    element: (
      <Page title="">
        <LandingView />
      </Page>
    ),
  },
  {
    element: <GuestRoute />,
    children: [
      {
        path: '/login',
        element: (
          <Page title="登录">
            <LoginView />
          </Page>
        ),
      },
    ],
  },
  {
    path: '/',
    element: <ProtectedRoute />,
    children: [
      {
        element: <AppLayout />,
        children: [
          {
            path: 'dashboard',
            element: (
              <Page title="仪表盘">
                <DashboardView />
              </Page>
            ),
          },
          {
            path: 'apikeys',
            element: (
              <Page title="API Key">
                <ApiKeysView />
              </Page>
            ),
          },
          {
            path: 'settings',
            element: (
              <Page title="账户设置">
                <SettingsView />
              </Page>
            ),
          },
          {
            element: <AdminRoute />,
            children: [
              {
                path: 'channels',
                element: (
                  <Page title="渠道管理">
                    <ChannelsView />
                  </Page>
                ),
              },
              {
                path: 'models',
                element: (
                  <Page title="模型管理">
                    <ModelsView />
                  </Page>
                ),
              },
              {
                path: 'users',
                element: (
                  <Page title="用户管理">
                    <UsersView />
                  </Page>
                ),
              },
            ],
          },
        ],
      },
    ],
  },
])

export default router