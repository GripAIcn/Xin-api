import type { ReactNode } from 'react'

import { cn } from '@/lib/utils'

interface SectionProps {
  id?: string
  className?: string
  children: ReactNode
}

/** 营销页统一宽容器：1440px 上限 + 与导航对齐的水平留白 */
export function Section({ id, className, children }: SectionProps) {
  return (
    <section id={id} className={cn('w-full py-24 md:py-32', className)}>
      <div className="mx-auto w-full max-w-[1440px] px-6 lg:px-10">
        {children}
      </div>
    </section>
  )
}

interface SectionHeadingProps {
  eyebrow?: string
  title: ReactNode
  description?: ReactNode
  align?: 'left' | 'center'
  className?: string
}

/** 区块标题三件套：eyebrow / 标题 / 描述 */
export function SectionHeading({
  eyebrow,
  title,
  description,
  align = 'left',
  className,
}: SectionHeadingProps) {
  return (
    <div
      className={cn(
        'max-w-2xl',
        align === 'center' && 'mx-auto text-center',
        className,
      )}
    >
      {eyebrow && (
        <p className="mb-3 text-sm font-semibold text-primary">{eyebrow}</p>
      )}
      <h2 className="text-3xl leading-tight font-semibold tracking-tight md:text-[2rem]">
        {title}
      </h2>
      {description && (
        <p className="mt-4 text-base leading-7 text-muted-foreground">
          {description}
        </p>
      )}
    </div>
  )
}
