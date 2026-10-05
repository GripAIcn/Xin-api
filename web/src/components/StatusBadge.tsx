import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

const toneDotClasses = {
  success: 'bg-success',
  warning: 'bg-warning',
  danger: 'bg-destructive',
  muted: 'bg-muted-foreground/50',
  primary: 'bg-primary',
} as const

export type StatusTone = keyof typeof toneDotClasses

/** 彩色状态圆点 */
export function StatusDot({
  tone,
  className,
}: {
  tone: StatusTone
  className?: string
}) {
  return (
    <span
      aria-hidden
      className={cn(
        'size-1.5 shrink-0 rounded-full',
        toneDotClasses[tone],
        className,
      )}
    />
  )
}

/** 彩点 + 文字的统一状态徽章 */
export function DotBadge({
  tone,
  children,
  className,
}: {
  tone: StatusTone
  children: ReactNode
  className?: string
}) {
  return (
    <span
      className={cn(
        'inline-flex w-fit shrink-0 items-center gap-1.5 rounded-full border px-2 py-0.5 text-xs font-medium whitespace-nowrap',
        className,
      )}
    >
      <StatusDot tone={tone} />
      {children}
    </span>
  )
}
