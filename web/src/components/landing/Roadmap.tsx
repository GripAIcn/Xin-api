import { ArrowRight, CheckCircle2, Clock, MapPin } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { cn } from '@/lib/utils'
import {
  roadmapStages,
  type RoadmapStage,
  type RoadmapTone,
} from '@/components/landing/content'

function StageIcon({ tone }: { tone: RoadmapTone }) {
  if (tone === 'default') {
    return (
      <CheckCircle2
        aria-hidden="true"
        className="mt-0.5 size-4 shrink-0 text-primary"
      />
    )
  }
  if (tone === 'secondary') {
    return (
      <Clock
        aria-hidden="true"
        className="mt-0.5 size-4 shrink-0 text-muted-foreground"
      />
    )
  }
  return (
    <MapPin
      aria-hidden="true"
      className="mt-0.5 size-4 shrink-0 text-muted-foreground"
    />
  )
}

export default function Roadmap() {
  return (
    <section id="roadmap" className="scroll-mt-20 bg-muted/30 py-20">
      <div className="mx-auto w-full max-w-6xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-bold tracking-tight">演进路线</h2>
          <p className="mt-3 text-muted-foreground">
            从统一网关到多维感知智能路由
          </p>
        </div>
        <div className="mt-12 grid gap-6 lg:grid-cols-3">
          {roadmapStages.map((stage: RoadmapStage, index) => (
            <div key={stage.version} className="relative h-full">
              <Card
                className={cn(
                  'h-full',
                  stage.tone === 'default' && 'border-primary/40'
                )}
              >
                <CardHeader className="gap-2">
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-sm font-semibold text-primary">
                      {stage.version}
                    </span>
                    <Badge variant={stage.tone}>{stage.badge}</Badge>
                  </div>
                  <h3 className="text-lg font-semibold leading-none">
                    {stage.name}
                  </h3>
                </CardHeader>
                <CardContent>
                  <ul className="space-y-2.5">
                    {stage.items.map((item) => (
                      <li
                        key={item}
                        className="flex items-start gap-2 text-sm leading-relaxed"
                      >
                        <StageIcon tone={stage.tone} />
                        <span>{item}</span>
                      </li>
                    ))}
                  </ul>
                </CardContent>
              </Card>
              {index < roadmapStages.length - 1 && (
                <ArrowRight
                  aria-hidden="true"
                  className="absolute -right-5 top-1/2 z-10 hidden size-5 -translate-y-1/2 text-muted-foreground lg:block"
                />
              )}
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
