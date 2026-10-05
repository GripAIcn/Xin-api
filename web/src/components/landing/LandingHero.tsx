import { Link } from 'react-router-dom'
import { ArrowRight } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { useLandingCta } from '@/components/landing/useLandingCta'

const compatModels = ['兼容 OpenAI', '兼容 Anthropic Claude', '兼容 DeepSeek']

export default function LandingHero() {
  const cta = useLandingCta()

  return (
    <section className="relative overflow-hidden bg-[#0a0a14] py-24 text-white md:py-32">
      <div
        aria-hidden="true"
        className="pointer-events-none absolute -top-40 left-1/2 size-[36rem] -translate-x-1/2 rounded-full bg-primary/25 blur-3xl"
      />
      <div
        aria-hidden="true"
        className="pointer-events-none absolute right-[8%] top-32 size-72 rounded-full bg-sky-500/10 blur-3xl"
      />
      <div className="relative mx-auto w-full max-w-6xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto flex max-w-3xl flex-col items-center text-center">
          <span className="rounded-full border border-white/15 bg-white/10 px-4 py-1.5 text-sm text-white/80">
            高性能 LLM API 网关
          </span>
          <h1 className="mt-6 text-balance text-4xl font-bold tracking-tight md:text-5xl lg:text-6xl">
            大模型 API 统一接入、
            <span className="bg-gradient-to-r from-sky-400 to-violet-400 bg-clip-text text-transparent">
              全链路可观测
            </span>
            与智能分发网关
          </h1>
          <p className="mt-6 max-w-2xl text-base leading-relaxed text-white/70 md:text-lg">
            无缝兼容 OpenAI、Anthropic Claude、DeepSeek
            等主流大模型，统一 /v1/chat/completions
            端点；负载均衡、熔断限流、多租户管理开箱即用。
          </p>
          <div className="mt-10 flex w-full flex-col items-center justify-center gap-3 sm:w-auto sm:flex-row">
            <Button
              size="lg"
              asChild
              className="h-11 rounded-full px-8 text-base has-[>svg]:px-8"
            >
              <Link to={cta.to}>
                {cta.label}
                <ArrowRight className="size-4" />
              </Link>
            </Button>
            <Button
              variant="outline"
              size="lg"
              asChild
              className="h-11 rounded-full border-white/25 bg-transparent px-8 text-base text-white hover:bg-white/10 hover:text-white"
            >
              <a href="#features">查看功能特性</a>
            </Button>
          </div>
          <div className="mt-8 flex flex-wrap items-center justify-center gap-2">
            {compatModels.map((model) => (
              <span
                key={model}
                className="rounded-full border border-white/15 bg-white/5 px-3 py-1 text-xs text-white/70"
              >
                {model}
              </span>
            ))}
          </div>
        </div>
      </div>
    </section>
  )
}
