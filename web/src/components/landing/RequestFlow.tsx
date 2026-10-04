import { Fragment } from 'react'
import {
  ArrowRight,
  ChevronDown,
  ChevronRight,
  Info,
} from 'lucide-react'

import { Card } from '@/components/ui/card'
import { cn } from '@/lib/utils'
import {
  flowSteps,
  flowTraceNote,
  quickSteps,
} from '@/components/landing/content'

export default function RequestFlow() {
  return (
    <section id="how-it-works" className="scroll-mt-20 bg-muted/30 py-20">
      <div className="mx-auto w-full max-w-6xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-bold tracking-tight">工作原理</h2>
          <p className="mt-3 text-muted-foreground">
            一次请求在网关中的完整生命周期
          </p>
        </div>

        <div className="mt-12 flex flex-col items-stretch gap-2 lg:flex-row lg:items-center lg:gap-3">
          {flowSteps.map((step, index) => {
            const isEndpoint = index === 0 || index === flowSteps.length - 1
            return (
              <Fragment key={step}>
                <div
                  className={cn(
                    'flex flex-1 items-center justify-center rounded-lg border px-3 py-3 text-center text-sm leading-snug',
                    isEndpoint
                      ? 'border-primary bg-primary font-medium text-primary-foreground'
                      : 'bg-card text-card-foreground'
                  )}
                >
                  {step}
                </div>
                {index < flowSteps.length - 1 && (
                  <>
                    <ChevronRight
                      aria-hidden="true"
                      className="hidden size-4 shrink-0 self-center text-muted-foreground lg:block"
                    />
                    <ChevronDown
                      aria-hidden="true"
                      className="mx-auto size-4 shrink-0 text-muted-foreground lg:hidden"
                    />
                  </>
                )}
              </Fragment>
            )
          })}
        </div>

        <div className="mt-8 flex items-start gap-3 rounded-lg border bg-card px-4 py-3.5">
          <Info
            aria-hidden="true"
            className="mt-0.5 size-4 shrink-0 text-muted-foreground"
          />
          <p className="text-sm leading-relaxed text-muted-foreground">
            {flowTraceNote}
          </p>
        </div>

        <h3 className="mt-16 text-center text-xl font-semibold tracking-tight">
          三步上手控制台
        </h3>
        <div className="mt-8 grid gap-6 md:grid-cols-3">
          {quickSteps.map((step, index) => (
            <div key={step.title} className="relative">
              <Card className="h-full gap-3">
                <span
                  aria-hidden="true"
                  className="text-3xl font-bold text-primary/30"
                >
                  {`0${index + 1}`}
                </span>
                <h3 className="text-lg font-semibold">{step.title}</h3>
                <p className="text-sm leading-relaxed text-muted-foreground">
                  {step.desc}
                </p>
              </Card>
              {index < quickSteps.length - 1 && (
                <ArrowRight
                  aria-hidden="true"
                  className="absolute -right-5 top-1/2 z-10 hidden size-5 -translate-y-1/2 text-muted-foreground md:block"
                />
              )}
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
