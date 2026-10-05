import { Card } from '@/components/ui/card'
import { v2Features } from '@/components/landing/content'

export default function ObservabilitySection() {
  return (
    <section className="py-20">
      <div className="mx-auto w-full max-w-6xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <span className="rounded-full bg-primary/10 px-3 py-1 text-xs font-medium text-primary">
            可观测
          </span>
          <h2 className="mt-4 text-3xl font-bold tracking-tight">
            可观测与计费
          </h2>
          <p className="mt-3 text-muted-foreground">
            让每一次调用可审计、可计量、可对账
          </p>
        </div>
        <div className="mt-12 grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
          {v2Features.map((feature) => {
            const Icon = feature.icon
            return (
              <Card
                key={feature.title}
                className="h-full gap-3 rounded-2xl shadow-sm"
              >
                <div className="px-6">
                  <span
                    aria-hidden="true"
                    className="flex size-10 items-center justify-center rounded-lg bg-muted"
                  >
                    <Icon className="size-5" />
                  </span>
                </div>
                <h3 className="px-6 text-base font-semibold leading-none">
                  {feature.title}
                </h3>
                <p className="px-6 text-sm leading-relaxed text-muted-foreground">
                  {feature.desc}
                </p>
              </Card>
            )
          })}
        </div>
      </div>
    </section>
  )
}
