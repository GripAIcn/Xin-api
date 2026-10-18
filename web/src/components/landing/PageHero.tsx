import type { ReactNode } from 'react'

import { Section } from '@/components/landing/Section'

interface PageHeroProps {
  eyebrow: string
  title: string
  description: ReactNode
}

/** 公开子页面的页头：每个可索引子页面只保留一个 h1 */
export default function PageHero({
  eyebrow,
  title,
  description,
}: PageHeroProps) {
  return (
    <Section className="border-b bg-marketing-panel pt-20 pb-16 md:pt-24 md:pb-20">
      <div className="max-w-3xl">
        <p className="mb-3 text-sm font-semibold text-primary">{eyebrow}</p>
        <h1 className="text-4xl leading-tight font-semibold tracking-tight md:text-5xl">
          {title}
        </h1>
        <p className="mt-5 max-w-2xl text-base leading-7 text-muted-foreground">
          {description}
        </p>
      </div>
    </Section>
  )
}
