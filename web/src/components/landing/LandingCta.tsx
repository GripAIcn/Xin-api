import { Link } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { useLandingCta } from '@/components/landing/useLandingCta'

export default function LandingCta() {
  const cta = useLandingCta()

  return (
    <section className="py-20">
      <div className="mx-auto w-full max-w-6xl px-4 sm:px-6 lg:px-8">
        <div className="rounded-3xl border border-primary/20 bg-primary px-6 py-16 text-center text-primary-foreground">
          <div className="mx-auto max-w-2xl">
            <h2 className="text-3xl font-bold tracking-tight">
              立即开始使用 Xin API
            </h2>
            <p className="mt-3 text-primary-foreground/80">
              登录控制台，几分钟内完成渠道接入
            </p>
            <Button
              variant="secondary"
              size="lg"
              asChild
              className="mt-8 h-11 px-8 text-base"
            >
              <Link to={cta.to}>{cta.label}</Link>
            </Button>
          </div>
        </div>
      </div>
    </section>
  )
}
