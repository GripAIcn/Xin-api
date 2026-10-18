import { useEffect, useState, type MouseEvent } from 'react'
import { Link, useLocation } from 'react-router-dom'
import { Menu } from 'lucide-react'

import ThemeToggle from '@/components/ThemeToggle'
import BrandLogo from '@/components/brand/BrandLogo'
import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import { siteNavItems } from '@/components/landing/content'
import { useLandingCta } from '@/components/landing/useLandingCta'
import { cn } from '@/lib/utils'

const GITHUB_URL = 'https://github.com/GripAIcn/Xin-api'

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

export default function LandingHeader() {
  const cta = useLandingCta()
  const { pathname } = useLocation()
  const [open, setOpen] = useState(false)
  const [scrolled, setScrolled] = useState(false)

  const isHome = pathname === '/'

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 8)
    onScroll()
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => window.removeEventListener('scroll', onScroll)
  }, [])

  const handleHomeClick = (event: MouseEvent<HTMLAnchorElement>) => {
    setOpen(false)
    if (!isHome) return
    event.preventDefault()
    const reduceMotion = window.matchMedia(
      '(prefers-reduced-motion: reduce)',
    ).matches
    window.scrollTo({ top: 0, behavior: reduceMotion ? 'auto' : 'smooth' })
  }

  const itemClassName = (active: boolean) =>
    cn(
      'relative flex h-16 items-center px-3 text-[15px] transition-colors hover:text-foreground',
      active ? 'text-foreground' : 'text-muted-foreground',
    )

  const mobileItemClassName =
    'flex min-h-11 items-center rounded-md px-3 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground'

  return (
    <header
      className={cn(
        'sticky top-0 z-50 border-b bg-background/85 backdrop-blur transition-[border-color,box-shadow] duration-200',
        scrolled ? 'border-border shadow-sm' : 'border-transparent',
      )}
    >
      <div className="mx-auto flex h-16 w-full max-w-[1440px] items-center gap-6 px-6 lg:px-10">
        <Link to="/" className="flex shrink-0 items-center" onClick={handleHomeClick}>
          <BrandLogo variant="lockup" size="md" />
        </Link>

        <nav
          aria-label="主导航"
          className="hidden flex-1 items-center justify-center lg:flex"
        >
          {siteNavItems.map((item) => {
            const active = pathname === item.href
            const content = (
              <>
                {item.label}
                {active && (
                  <span
                    aria-hidden="true"
                    className="absolute inset-x-3 bottom-0 h-0.5 rounded-full bg-primary"
                  />
                )}
              </>
            )

            if (item.href === '/') {
              return (
                <a
                  key={item.href}
                  href="/"
                  aria-current={active ? 'page' : undefined}
                  onClick={handleHomeClick}
                  className={itemClassName(active)}
                >
                  {content}
                </a>
              )
            }

            return (
              <Link
                key={item.href}
                to={item.href}
                aria-current={active ? 'page' : undefined}
                className={itemClassName(active)}
              >
                {content}
              </Link>
            )
          })}
        </nav>

        <div className="ml-auto hidden items-center gap-1 lg:flex">
          <Button variant="ghost" size="icon" asChild>
            <a
              href={GITHUB_URL}
              target="_blank"
              rel="noreferrer"
              aria-label="GitHub"
            >
              <GithubIcon className="size-5" />
            </a>
          </Button>
          <ThemeToggle />
          <Button size="sm" asChild className="ml-2 h-9 rounded-full px-5">
            <Link to={cta.to}>{cta.label}</Link>
          </Button>
        </div>

        <div className="ml-auto flex items-center gap-1 lg:hidden">
          <Button variant="ghost" size="icon" asChild>
            <a
              href={GITHUB_URL}
              target="_blank"
              rel="noreferrer"
              aria-label="GitHub"
              className="size-11"
            >
              <GithubIcon className="size-5" />
            </a>
          </Button>
          <Sheet open={open} onOpenChange={setOpen}>
            <SheetTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="size-11"
                aria-label="打开导航菜单"
              >
                <Menu className="size-5" aria-hidden="true" />
              </Button>
            </SheetTrigger>
            <SheetContent side="right" className="w-80">
              <SheetHeader>
                <SheetTitle>导航</SheetTitle>
              </SheetHeader>
              <div className="flex flex-col gap-1 px-4">
                {siteNavItems.map((item) => {
                  if (item.href === '/') {
                    return (
                      <a
                        key={item.href}
                        href="/"
                        onClick={handleHomeClick}
                        className={mobileItemClassName}
                      >
                        {item.label}
                      </a>
                    )
                  }

                  return (
                    <Link
                      key={item.href}
                      to={item.href}
                      onClick={() => setOpen(false)}
                      className={mobileItemClassName}
                    >
                      {item.label}
                    </Link>
                  )
                })}
              </div>
              <div className="mt-auto flex flex-col gap-3 px-4 pb-6">
                <div className="flex items-center justify-between rounded-lg border px-3">
                  <span className="text-sm text-muted-foreground">主题</span>
                  <ThemeToggle />
                </div>
                <Button
                  className="h-11 w-full rounded-full"
                  asChild
                  onClick={() => setOpen(false)}
                >
                  <Link to={cta.to}>{cta.label}</Link>
                </Button>
              </div>
            </SheetContent>
          </Sheet>
        </div>
      </div>
    </header>
  )
}
