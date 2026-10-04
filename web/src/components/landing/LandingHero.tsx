import { Link } from 'react-router-dom'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useLandingCta } from '@/components/landing/useLandingCta'

const compatModels = ['兼容 OpenAI', '兼容 Anthropic Claude', '兼容 DeepSeek']

export default function LandingHero() {
  const cta = useLandingCta()

  return (
    <section className="py-20 md:py-24">
      <div className="mx-auto w-full max-w-6xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto flex max-w-3xl flex-col items-center text-center">
          <Badge variant="secondary" className="px-3 py-1">
            高性能 LLM API 网关
          </Badge>
          <h1 className="mt-6 text-balance text-4xl font-bold tracking-tight md:text-5xl lg:text-6xl">
            大模型 API 统一接入、
            <span className="text-primary">全链路可观测</span>
            与智能分发网关
          </h1>
          <p className="mt-6 max-w-2xl text-base leading-relaxed text-muted-foreground md:text-lg">
            无缝兼容 OpenAI、Anthropic Claude、DeepSeek
            等主流大模型，统一 /v1/chat/completions
            端点；负载均衡、熔断限流、多租户管理开箱即用。
          </p>
          <div className="mt-10 flex w-full flex-col items-center justify-center gap-3 sm:w-auto sm:flex-row">
            <Button
              size="lg"
              asChild
              className="h-11 w-full px-8 text-base sm:w-auto"
            >
              <Link to={cta.to}>{cta.label}</Link>
            </Button>
            <Button
              variant="outline"
              size="lg"
              asChild
              className="h-11 w-full px-8 text-base sm:w-auto"
            >
              <a href="#features">查看功能特性</a>
            </Button>
          </div>
          <div className="mt-8 flex flex-wrap items-center justify-center gap-2">
            {compatModels.map((model) => (
              <Badge key={model} variant="outline" className="px-3 py-1">
                {model}
              </Badge>
            ))}
          </div>
        </div>
      </div>
    </section>
  )
}
