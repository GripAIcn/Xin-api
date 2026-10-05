import {
  Card,
  CardContent,
  CardHeader,
} from '@/components/ui/card'
import { features } from '@/components/landing/content'

export default function FeatureGrid() {
  return (
    <section id="features" className="scroll-mt-20 py-20">
      <div className="mx-auto w-full max-w-6xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <span className="rounded-full bg-primary/10 px-3 py-1 text-xs font-medium text-primary">
            核心能力
          </span>
          <h2 className="mt-4 text-3xl font-bold tracking-tight">功能特性</h2>
          <p className="mt-3 text-muted-foreground">
            从统一接入到高可用防护，网关能力开箱即用
          </p>
        </div>
        <div className="mt-12 grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
          {features.map((feature) => {
            const Icon = feature.icon
            return (
              <Card
                key={feature.title}
                className="h-full rounded-2xl shadow-sm transition hover:border-primary/50 hover:shadow-sm"
              >
                <CardHeader className="gap-4">
                  <span
                    aria-hidden="true"
                    className="flex size-10 items-center justify-center rounded-lg bg-muted"
                  >
                    <Icon className="size-5" />
                  </span>
                  <h3 className="text-base font-semibold leading-none">
                    {feature.title}
                  </h3>
                </CardHeader>
                <CardContent>
                  <p className="text-sm leading-relaxed text-muted-foreground">
                    {feature.desc}
                  </p>
                </CardContent>
              </Card>
            )
          })}
        </div>
      </div>
    </section>
  )
}
