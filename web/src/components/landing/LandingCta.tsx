import { Link } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { useLandingCta } from '@/components/landing/useLandingCta'

export default function LandingCta() {
  const cta = useLandingCta()

  return (
    <section className="py-20">
      <div className="mx-auto w-full max-w-6xl px-4 sm:px-6 lg:px-8">
        <div className="relative overflow-hidden rounded-3xl bg-[#0a0a14] px-6 py-16 text-center text-white">
          <div
            aria-hidden="true"
            className="pointer-events-none absolute -top-24 left-1/2 size-96 -translate-x-1/2 rounded-full bg-primary/25 blur-3xl"
          />
          <div className="relative mx-auto max-w-2xl">
            <h2 className="text-3xl font-bold tracking-tight">
              立即开始使用 Xin API
            </h2>
            <p className="mt-3 text-white/70">
              登录控制台，几分钟内完成渠道接入
            </p>
            <Button
              size="lg"
              asChild
              className="mt-8 h-11 rounded-full bg-white px-8 text-base text-[#0a0a14] hover:bg-white/90"
            >
              <Link to={cta.to}>{cta.label}</Link>
            </Button>
          </div>
        </div>
      </div>
    </section>
  )
}
