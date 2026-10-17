import { useEffect, type ReactNode } from 'react'
import type { RouteObject } from 'react-router-dom'

import DocsView from '@/pages/DocsView'
import LandingView from '@/pages/LandingView'

// 渲染时设置 document.title
export function Page({
  title,
  children,
}: {
  title?: string
  children: ReactNode
}) {
  useEffect(() => {
    document.title = title
      ? `${title} - Xin API`
      : 'Xin API - 大模型 API 统一管理、全链路可观测与智能分发网关'
  }, [title])
  return <>{children}</>
}

/**
 * 公开路由（无需登录）：官网首页与使用文档。
 * 独立成模块，供构建期预渲染与客户端路由共用，保证两侧渲染树一致。
 */
export const publicRoutes: RouteObject[] = [
  {
    path: '/',
    element: (
      <Page title="">
        <LandingView />
      </Page>
    ),
  },
  {
    path: '/docs',
    element: (
      <Page title="使用文档">
        <DocsView />
      </Page>
    ),
  },
]
