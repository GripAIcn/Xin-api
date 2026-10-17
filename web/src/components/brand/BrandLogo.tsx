import { useId } from 'react'

import { cn } from '@/lib/utils'

export type BrandLogoSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl' | '2xl'

const sizeStyles: Record<
  BrandLogoSize,
  { box: string; text: string; gap: string }
> = {
  xs: { box: 'size-4', text: 'text-[13px]', gap: 'gap-2' },
  sm: { box: 'size-5', text: 'text-sm', gap: 'gap-2' },
  md: { box: 'size-8', text: 'text-base', gap: 'gap-2.5' },
  lg: { box: 'size-9', text: 'text-lg', gap: 'gap-2.5' },
  xl: { box: 'size-14', text: 'text-2xl', gap: 'gap-3' },
  '2xl': { box: 'size-10', text: 'text-2xl', gap: 'gap-3' },
}

const COMPACT_SIZES: BrandLogoSize[] = ['xs', 'sm']

interface BrandLogoProps {
  /** mark 仅图标；lockup 为图标 + Xin API 字标 */
  variant?: 'mark' | 'lockup'
  size?: BrandLogoSize
  /** mono 为单色 currentColor；inverse 用于深色底 */
  tone?: 'default' | 'mono' | 'inverse'
  className?: string
}

function BrandMark({
  gradientId,
  mono,
  compact,
  className,
  decorative,
}: {
  gradientId: string
  mono: boolean
  compact: boolean
  className?: string
  decorative?: boolean
}) {
  const strokeWidth = compact ? 9 : 8.5

  if (mono) {
    return (
      <svg
        viewBox="0 0 64 64"
        fill="none"
        className={className}
        role={decorative ? undefined : 'img'}
        aria-label={decorative ? undefined : 'Xin API'}
        aria-hidden={decorative || undefined}
      >
        <path
          d="M17 17 47 47"
          stroke="currentColor"
          strokeWidth={strokeWidth}
          strokeLinecap="round"
        />
        <path
          d="M47 17 17 47"
          stroke="currentColor"
          strokeWidth={strokeWidth}
          strokeLinecap="round"
        />
      </svg>
    )
  }

  return (
    <svg
      viewBox="0 0 64 64"
      fill="none"
      className={className}
      role={decorative ? undefined : 'img'}
      aria-label={decorative ? undefined : 'Xin API'}
      aria-hidden={decorative || undefined}
    >
      <defs>
        <linearGradient
          id={gradientId}
          x1="14"
          y1="14"
          x2="50"
          y2="50"
          gradientUnits="userSpaceOnUse"
        >
          <stop offset="0" stopColor="#22D3EE" />
          <stop offset="0.55" stopColor="#3B82F6" />
          <stop offset="1" stopColor="#8B5CF6" />
        </linearGradient>
      </defs>
      <rect width="64" height="64" rx="14" fill="#0C1424" />
      <path
        d="M17 17 47 47"
        stroke={`url(#${gradientId})`}
        strokeWidth={strokeWidth}
        strokeLinecap="round"
      />
      <path
        d="M47 17 17 47"
        stroke={`url(#${gradientId})`}
        strokeWidth={strokeWidth}
        strokeLinecap="round"
      />
      {!compact && (
        <>
          <circle cx="17" cy="17" r="4.2" fill="#A5F3FC" />
          <circle cx="47" cy="47" r="4.2" fill="#C4B5FD" />
          <circle cx="32" cy="32" r="3.2" fill="#0C1424" />
          <circle cx="32" cy="32" r="2.2" fill="#93C5FD" />
        </>
      )}
    </svg>
  )
}

export default function BrandLogo({
  variant = 'lockup',
  size = 'md',
  tone = 'default',
  className,
}: BrandLogoProps) {
  const gradientId = `xin-brand-${useId().replace(/:/g, '')}`
  const preset = sizeStyles[size]
  const compact = COMPACT_SIZES.includes(size)

  const mark = (
    <BrandMark
      gradientId={gradientId}
      mono={tone === 'mono'}
      compact={compact}
      className={cn(preset.box, 'shrink-0')}
      decorative={variant === 'lockup'}
    />
  )

  if (variant === 'mark') {
    return <span className={cn('inline-flex', className)}>{mark}</span>
  }

  return (
    <span
      className={cn(
        'inline-flex items-center',
        preset.gap,
        tone === 'inverse' ? 'text-white' : 'text-foreground',
        tone === 'mono' && 'text-current',
        className,
      )}
    >
      {mark}
      <span
        className={cn(
          preset.text,
          'font-semibold tracking-tight whitespace-nowrap',
        )}
      >
        Xin
        <span
          className={cn(
            'font-normal',
            tone === 'default' && 'text-muted-foreground',
            tone === 'inverse' && 'text-white/60',
            tone === 'mono' && 'opacity-70',
          )}
        >
          {' '}
          API
        </span>
      </span>
    </span>
  )
}
