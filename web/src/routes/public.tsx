import { useEffect, type ReactNode } from 'react'
import type { RouteObject } from 'react-router-dom'

import ContactView from '@/pages/ContactView'
import DocsView from '@/pages/DocsView'
import FaqView from '@/pages/FaqView'
import FeaturesView from '@/pages/FeaturesView'
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
 * 公开路由（无需登录）：官网首页、功能介绍、使用文档、常见问题与联系我们。
 * 独立成模块，供构建期预渲染与客户端路由共用，保证两侧渲染树一致。
 * 每个路由都要在 scripts/prerender.ts 的 PUBLIC_PAGES 中登记，才能产出可被收录的独立页面。
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
    path: '/features',
    element: (
      <Page title="功能介绍">
        <FeaturesView />
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
  {
    path: '/faq',
    element: (
      <Page title="常见问题">
        <FaqView />
      </Page>
    ),
  },
  {
    path: '/contact',
    element: (
      <Page title="联系我们">
        <ContactView />
      </Page>
    ),
  },
]
