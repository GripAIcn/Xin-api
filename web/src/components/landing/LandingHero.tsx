import { Link } from 'react-router-dom'
import { ArrowRight } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Section } from '@/components/landing/Section'
import CodePreview from '@/components/landing/CodePreview'
import { useLandingCta } from '@/components/landing/useLandingCta'

export default function LandingHero() {
  const cta = useLandingCta()

  return (
    <Section className="relative overflow-hidden pt-16 pb-16 md:pt-24 md:pb-24">
      <div
        aria-hidden="true"
        className="pointer-events-none absolute -top-56 right-[-12%] size-[42rem] rounded-full bg-marketing-glow blur-3xl"
      />

      <div className="relative grid gap-12 lg:grid-cols-12 lg:items-center lg:gap-10">
        <div className="min-w-0 max-w-xl lg:col-span-5">
          <span className="inline-flex items-center gap-2 rounded-full border bg-background/70 px-3 py-1 text-xs font-medium text-muted-foreground">
            <span
              aria-hidden="true"
              className="size-1.5 rounded-full bg-primary"
            />
            兼容 OpenAI · Anthropic · Gemini
          </span>

          <h1 className="mt-6 text-[clamp(2.5rem,5vw,4rem)] leading-[1.05] font-semibold tracking-tight">
            一个端点，
            <br />
            <span className="text-primary">接入所有大模型</span>
          </h1>

          <p className="mt-6 text-base leading-7 text-muted-foreground md:text-lg">
            统一 <code className="font-mono text-[0.95em]">/v1/chat/completions</code>{' '}
            端点，内置负载均衡、熔断限流与全链路追踪，无需改动现有调用代码。
          </p>

          <div className="mt-8 flex flex-col gap-3 sm:flex-row">
            <Button
              size="lg"
              asChild
              className="h-11 rounded-full px-7 text-base has-[>svg]:px-7"
            >
              <Link to={cta.to}>
                {cta.label}
                <ArrowRight className="size-4" />
              </Link>
            </Button>
            <Button
              variant="ghost"
              size="lg"
              asChild
              className="h-11 rounded-full px-6 text-base"
            >
              <a href="#features">查看功能介绍</a>
            </Button>
          </div>

          <p className="mt-6 text-sm text-muted-foreground">
            开源自托管 · 无需修改现有 OpenAI SDK 代码
          </p>
        </div>

        <div className="min-w-0 lg:col-span-7">
          <CodePreview />
        </div>
      </div>
    </Section>
  )
}
