import { Link } from 'react-router-dom'

const GITHUB_URL = 'https://github.com/GripAIcn/Xin-api'

const navItems = [
  { label: '功能特性', href: '#features' },
  { label: '工作原理', href: '#how-it-works' },
  { label: '常见问题', href: '#faq' },
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

export default function LandingFooter() {
  return (
    <footer className="border-t">
      <div className="mx-auto w-full max-w-6xl px-4 py-10 sm:px-6 lg:px-8">
        <div className="grid gap-8 md:grid-cols-3">
          <div>
            <div className="flex items-center gap-2 text-base font-semibold tracking-tight">
              <span
                aria-hidden="true"
                className="flex size-8 items-center justify-center rounded-md border border-primary/20 bg-primary text-sm font-bold text-primary-foreground"
              >
                X
              </span>
              Xin API
            </div>
            <p className="mt-3 text-sm text-muted-foreground">
              高性能大模型 API 统一网关
            </p>
          </div>

          <nav aria-label="页脚导航">
            <h2 className="text-sm font-semibold">导航</h2>
            <ul className="mt-3">
              {navItems.map((item) => (
                <li key={item.href}>
                  <a
                    href={item.href}
                    className="flex min-h-11 items-center text-sm text-muted-foreground transition-colors hover:text-foreground md:min-h-0"
                  >
                    {item.label}
                  </a>
                </li>
              ))}
            </ul>
          </nav>

          <div>
            <h2 className="text-sm font-semibold">资源</h2>
            <ul className="mt-3">
              <li>
                <a
                  href={GITHUB_URL}
                  target="_blank"
                  rel="noreferrer"
                  className="flex min-h-11 items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground md:min-h-0"
                >
                  <GithubIcon className="size-4" />
                  GitHub
                </a>
              </li>
              <li>
                <Link
                  to="/login"
                  className="flex min-h-11 items-center text-sm text-muted-foreground transition-colors hover:text-foreground md:min-h-0"
                >
                  控制台
                </Link>
              </li>
            </ul>
          </div>
        </div>

        <div className="mt-8 border-t pt-6 text-xs text-muted-foreground">
          © 2026 Xin API. All rights reserved.
        </div>
      </div>
    </footer>
  )
}
