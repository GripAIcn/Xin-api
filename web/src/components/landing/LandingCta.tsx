import { Link } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import Reveal from '@/components/landing/Reveal'
import { Section } from '@/components/landing/Section'
import { useLandingCta } from '@/components/landing/useLandingCta'

export default function LandingCta() {
  const cta = useLandingCta()

  return (
    <Section className="border-t">
      <Reveal>
        <div className="relative overflow-hidden rounded-3xl bg-[#0a0a14] px-6 py-20 text-center text-white md:py-28">
          <div
            aria-hidden="true"
            className="pointer-events-none absolute -top-40 left-1/2 size-[36rem] -translate-x-1/2 rounded-full bg-primary/25 blur-3xl"
          />
          <div
            aria-hidden="true"
            className="pointer-events-none absolute inset-x-0 bottom-0 h-px bg-gradient-to-r from-transparent via-white/20 to-transparent"
          />
          <div className="relative mx-auto max-w-2xl">
            <h2 className="text-3xl leading-tight font-semibold tracking-tight md:text-4xl">
              立即开始使用 Xin API
            </h2>
            <p className="mt-4 text-base leading-7 text-white/65">
              登录控制台，几分钟内完成渠道接入，把多供应商的复杂度交给网关。
            </p>
            <Button
              size="lg"
              asChild
              className="mt-9 h-12 rounded-full bg-white px-8 text-base text-[#0a0a14] hover:bg-white/90"
            >
              <Link to={cta.to}>{cta.label}</Link>
            </Button>
          </div>
        </div>
      </Reveal>
    </Section>
  )
}
