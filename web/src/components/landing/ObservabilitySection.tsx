import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import { v2Features } from '@/components/landing/content'

export default function ObservabilitySection() {
  return (
    <section className="py-20">
      <div className="mx-auto w-full max-w-6xl px-4 sm:px-6 lg:px-8">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
          <div className="max-w-2xl">
            <h2 className="text-3xl font-bold tracking-tight">可观测与计费</h2>
            <p className="mt-3 text-muted-foreground">
              让每一次调用可审计、可计量、可对账
            </p>
          </div>
          <Badge variant="secondary" className="px-3 py-1">
            V2 规划中
          </Badge>
        </div>
        <div className="mt-12 grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
          {v2Features.map((feature) => {
            const Icon = feature.icon
            return (
              <Card key={feature.title} className="h-full gap-3">
                <div className="flex items-center justify-between gap-2 px-6">
                  <span
                    aria-hidden="true"
                    className="flex size-10 items-center justify-center rounded-lg bg-muted"
                  >
                    <Icon className="size-5" />
                  </span>
                  <Badge variant="secondary" className="text-[11px]">
                    V2 规划中
                  </Badge>
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
