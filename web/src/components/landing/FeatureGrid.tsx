import Reveal from '@/components/landing/Reveal'
import { Section, SectionHeading } from '@/components/landing/Section'
import {
  coreCapabilities,
  moreCapabilities,
} from '@/components/landing/content'

export default function FeatureGrid() {
  return (
    <Section id="features" className="scroll-mt-16">
      <Reveal>
        <SectionHeading
          eyebrow="核心能力"
          title="从接入到可观测，一个网关全部覆盖"
          description="统一协议、智能选路、高可用防护与全链路追踪，无需在多个系统之间来回拼装。"
        />
      </Reveal>

      <div className="mt-14 grid gap-5 sm:grid-cols-2 xl:grid-cols-4">
        {coreCapabilities.map((capability) => {
          const Icon = capability.icon
          return (
            <Reveal key={capability.index}>
              <article className="group relative flex h-full flex-col overflow-hidden rounded-3xl border bg-card p-6 transition-[border-color,transform] duration-200 hover:-translate-y-0.5 hover:border-primary/40">
                <span
                  aria-hidden="true"
                  className="pointer-events-none absolute top-3 right-5 font-mono text-6xl font-semibold tracking-tighter text-foreground/5 tabular-nums select-none"
                >
                  {capability.index}
                </span>
                <span className="flex size-11 items-center justify-center rounded-xl border border-primary/15 bg-primary/10 text-primary">
                  <Icon className="size-5" aria-hidden="true" />
                </span>
                <h3 className="mt-6 text-lg font-semibold tracking-tight">
                  {capability.title}
                </h3>
                <p className="mt-2 text-sm leading-6 text-muted-foreground">
                  {capability.desc}
                </p>
                <ul className="mt-auto flex flex-wrap gap-1.5 pt-6">
                  {capability.keywords.map((keyword) => (
                    <li
                      key={keyword}
                      className="rounded-full border bg-muted/40 px-2.5 py-1 text-[11px] text-muted-foreground"
                    >
                      {keyword}
                    </li>
                  ))}
                </ul>
              </article>
            </Reveal>
          )
        })}
      </div>

      <div className="mt-20 border-t pt-14">
        <Reveal>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <h3 className="text-xl font-semibold tracking-tight">更多能力</h3>
              <p className="mt-2 text-sm text-muted-foreground">
                控制台里的日常运维，也一并收好。
              </p>
            </div>
          </div>
        </Reveal>

        <div className="mt-8 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
          {moreCapabilities.map((capability) => {
            const Icon = capability.icon
            return (
              <Reveal key={capability.title}>
                <div className="flex h-full gap-3 rounded-2xl border bg-card p-5 transition-[border-color,transform] duration-200 hover:-translate-y-0.5 hover:border-primary/40">
                  <span className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
                    <Icon className="size-4" aria-hidden="true" />
                  </span>
                  <div>
                    <h4 className="text-sm font-semibold">{capability.title}</h4>
                    <p className="mt-1 text-xs leading-5 text-muted-foreground">
                      {capability.desc}
                    </p>
                  </div>
                </div>
              </Reveal>
            )
          })}
        </div>
      </div>
    </Section>
  )
}
