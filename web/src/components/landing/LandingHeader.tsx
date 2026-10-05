import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Menu } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import { useLandingCta } from '@/components/landing/useLandingCta'

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

export default function LandingHeader() {
  const cta = useLandingCta()
  const [open, setOpen] = useState(false)

  return (
    <header className="sticky top-0 z-50 bg-transparent">
      <nav
        aria-label="主导航"
        className="mx-auto mt-3 flex h-14 w-full max-w-5xl items-center justify-between gap-4 rounded-full border bg-background/70 px-4 backdrop-blur-md sm:px-5"
      >
        <Link
          to="/"
          className="flex min-h-11 items-center gap-2 text-base font-semibold tracking-tight"
        >
          <span
            aria-hidden="true"
            className="flex size-8 items-center justify-center rounded-md border border-primary/20 bg-primary text-sm font-bold text-primary-foreground"
          >
            X
          </span>
          Xin API
        </Link>

        <div className="hidden items-center gap-1 md:flex">
          {navItems.map((item) => (
            <a
              key={item.href}
              href={item.href}
              className="rounded-md px-3 py-2 text-sm text-muted-foreground transition-colors hover:text-foreground"
            >
              {item.label}
            </a>
          ))}
          <a
            href={GITHUB_URL}
            target="_blank"
            rel="noreferrer"
            aria-label="GitHub"
            className="inline-flex size-9 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          >
            <GithubIcon className="size-5" />
          </a>
          <Button size="sm" asChild className="ml-2 h-9">
            <Link to={cta.to}>{cta.label}</Link>
          </Button>
        </div>

        <div className="flex items-center gap-1 md:hidden">
          <a
            href={GITHUB_URL}
            target="_blank"
            rel="noreferrer"
            aria-label="GitHub"
            className="inline-flex size-11 items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          >
            <GithubIcon className="size-5" />
          </a>
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
            <SheetContent side="right" className="w-72">
              <SheetHeader>
                <SheetTitle>导航</SheetTitle>
              </SheetHeader>
              <div className="flex flex-col gap-1 px-4">
                {navItems.map((item) => (
                  <a
                    key={item.href}
                    href={item.href}
                    onClick={() => setOpen(false)}
                    className="flex min-h-11 items-center rounded-md px-3 text-sm font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
                  >
                    {item.label}
                  </a>
                ))}
              </div>
              <div className="mt-auto flex flex-col gap-3 px-4 pb-6">
                <Button
                  className="h-11 w-full"
                  asChild
                  onClick={() => setOpen(false)}
                >
                  <Link to={cta.to}>{cta.label}</Link>
                </Button>
              </div>
            </SheetContent>
          </Sheet>
        </div>
      </nav>
    </header>
  )
}
