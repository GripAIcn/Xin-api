import { Link, useLocation } from 'react-router-dom'

import BrandLogo from '@/components/brand/BrandLogo'
import { CONTACT_EMAIL } from '@/components/landing/content'

const GITHUB_URL = 'https://github.com/GripAIcn/Xin-api'
const ISSUES_URL = 'https://github.com/GripAIcn/Xin-api/issues'
const LICENSE_URL = 'https://github.com/GripAIcn/Xin-api/blob/main/LICENSE'

interface FooterGroup {
  title: string
  items: { label: string; href: string; external?: boolean }[]
}

const footerGroups: FooterGroup[] = [
  {
    title: '产品',
    items: [
      { label: '功能介绍', href: '/features' },
      { label: '工作原理', href: '#how-it-works' },
      { label: '可观测', href: '#observability' },
    ],
  },
  {
    title: '能力',
    items: [
      { label: '统一接入', href: '/features' },
      { label: '智能路由', href: '/features' },
      { label: '高可用防护', href: '/features' },
    ],
  },
  {
    title: '资源',
    items: [
      { label: '使用文档', href: '/docs' },
      { label: '常见问题', href: '/faq' },
      { label: 'GitHub 仓库', href: GITHUB_URL, external: true },
    ],
  },
  {
    title: '联系我们',
    items: [
      { label: '联系方式', href: '/contact' },
      { label: CONTACT_EMAIL, href: `mailto:${CONTACT_EMAIL}`, external: true },
      { label: 'GitHub Issues', href: ISSUES_URL, external: true },
      { label: '登录控制台', href: '/login' },
    ],
  },
]

function GithubIcon({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="currentColor"
      className={className}
      aria-hidden="true"
    >
      <path d="M12 .5C5.65.5.5 5.65.5 12c0 5.08 3.29 9.39 7.86 10.91.58.11.79-.25.79-.55 0-.27-.01-1.17-.02-2.12-3.2.7-3.88-1.36-3.88-1.36-.52-1.33-1.28-1.68-1.28-1.68-1.05-.72.08-.7.08-.7 1.16.08 1.77 1.19 1.77 1.19 1.03 1.77 2.7 1.26 3.36.96.1-.75.4-1.26.73-1.55-2.55-.29-5.24-1.28-5.24-5.69 0-1.26.45-2.28 1.19-3.09-.12-.29-.52-1.46.11-3.05 0 0 .97-.31 3.18 1.18a11.1 11.1 0 0 1 2.9-.39c.98 0 1.97.13 2.9.39 2.2-1.49 3.17-1.18 3.17-1.18.63 1.59.23 2.76.11 3.05.74.81 1.19 1.83 1.19 3.09 0 4.42-2.69 5.39-5.25 5.68.41.36.78 1.06.78 2.14 0 1.55-.01 2.79-.01 3.17 0 .31.21.67.8.55A11.51 11.51 0 0 0 23.5 12C23.5 5.65 18.35.5 12 .5Z" />
    </svg>
  )
}

const linkClassName =
  'flex min-h-11 items-center text-sm text-muted-foreground transition-colors hover:text-foreground md:min-h-0'

export default function LandingFooter() {
  const { pathname } = useLocation()
  const isHome = pathname === '/'

  return (
    <footer className="border-t">
      <div className="mx-auto w-full max-w-[1440px] px-6 py-16 lg:px-10">
        <div className="grid gap-10 md:grid-cols-2 lg:grid-cols-6 lg:gap-8">
          <div className="lg:col-span-2">
            <BrandLogo variant="lockup" size="md" />
            <p className="mt-4 max-w-xs text-sm leading-6 text-muted-foreground">
              高性能大模型 API 统一网关，让多供应商接入、选路与计量回归简单。
            </p>
            <a
              href={GITHUB_URL}
              target="_blank"
              rel="noreferrer"
              className="mt-5 inline-flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground"
            >
              <GithubIcon className="size-4" />
              GripAIcn/Xin-api
            </a>
          </div>

          {footerGroups.map((group) => (
            <nav
              key={group.title}
              id={group.title === '联系我们' ? 'contact' : undefined}
              aria-label={`页脚 · ${group.title}`}
              className="scroll-mt-24"
            >
              <h2 className="text-sm font-semibold">{group.title}</h2>
              <ul className="mt-4 space-y-1">
                {group.items.map((item) =>
                  item.external ? (
                    <li key={item.label}>
                      <a
                        href={item.href}
                        target="_blank"
                        rel="noreferrer"
                        className={linkClassName}
                      >
                        {item.label}
                      </a>
                    </li>
                  ) : item.href.startsWith('#') ? (
                    <li key={item.label}>
                      <a
                        href={isHome ? item.href : `/${item.href}`}
                        className={linkClassName}
                      >
                        {item.label}
                      </a>
                    </li>
                  ) : (
                    <li key={item.label}>
                      <Link to={item.href} className={linkClassName}>
                        {item.label}
                      </Link>
                    </li>
                  ),
                )}
              </ul>
            </nav>
          ))}
        </div>

        <div className="mt-14 flex flex-col gap-2 border-t pt-6 text-xs text-muted-foreground sm:flex-row sm:items-center sm:justify-between">
          <p>© 2026 Xin API. All rights reserved.</p>
          <a
            href={LICENSE_URL}
            target="_blank"
            rel="noreferrer"
            className="transition-colors hover:text-foreground"
          >
            MIT License
          </a>
        </div>
      </div>
    </footer>
  )
}
