import { ArrowRight, ChevronRight } from 'lucide-react'

import Reveal from '@/components/landing/Reveal'
import { Section, SectionHeading } from '@/components/landing/Section'
import { flowStages, flowTraceNote } from '@/components/landing/content'

export default function RequestFlow() {
  return (
    <Section id="how-it-works" className="scroll-mt-16 border-t">
      <Reveal>
        <SectionHeading
          eyebrow="工作原理"
          title="一次请求，四步走完网关链路"
          description="从鉴权到计量，每一步都留下可观测的轨迹。"
        />
      </Reveal>

      <Reveal>
        <ol className="mt-12 flex flex-col gap-4 md:grid md:grid-cols-2 lg:grid-cols-4 lg:gap-6">
          {flowStages.map((stage, index) => (
            <li key={stage.title} className="relative pl-7 md:pl-0">
              {index < flowStages.length - 1 && (
                <span
                  aria-hidden="true"
                  className="absolute top-10 left-[4.5px] h-[calc(100%+1rem)] w-px bg-border md:hidden"
                />
              )}
              <span
                aria-hidden="true"
                className="absolute top-6 left-0 size-2.5 rounded-full bg-primary md:hidden"
              />

              <div className="flex h-full flex-col rounded-2xl border bg-card p-5 transition-[border-color,transform] duration-200 hover:-translate-y-0.5 hover:border-primary/40">
                <div className="flex items-center gap-3">
                  <span className="font-mono text-xs text-primary">
                    {String(index + 1).padStart(2, '0')}
                  </span>
                  <h3 className="text-base font-semibold">{stage.title}</h3>
                </div>
                <p className="mt-2 text-xs text-muted-foreground">
                  {stage.desc}
                </p>
                <ul className="mt-4 space-y-2 border-t pt-4">
                  {stage.items.map((item) => (
                    <li
                      key={item}
                      className="flex items-start gap-2 text-sm leading-6 text-muted-foreground"
                    >
                      <ChevronRight
                        aria-hidden="true"
                        className="mt-1 size-3.5 shrink-0 text-primary"
                      />
                      {item}
                    </li>
                  ))}
                </ul>
              </div>

              {index < flowStages.length - 1 && (
                <ArrowRight
                  aria-hidden="true"
                  className="absolute top-1/2 -right-[15px] hidden size-5 -translate-y-1/2 text-muted-foreground/40 lg:block"
                />
              )}
            </li>
          ))}
        </ol>
      </Reveal>

      <Reveal>
        <div className="mt-8 flex flex-col gap-3 rounded-xl border bg-marketing-panel px-4 py-3 font-mono text-xs leading-6 text-muted-foreground md:flex-row md:items-center md:gap-2">
          <span className="shrink-0 font-medium text-primary">xin_req_*</span>
          <span aria-hidden="true" className="hidden md:inline">
            ↔
          </span>
          <span className="shrink-0 text-foreground">上游 request id</span>
          <span className="md:ml-3">{flowTraceNote}</span>
        </div>
      </Reveal>
    </Section>
  )
}
