import { Section, SectionHeading } from '@/components/landing/Section'
import Reveal from '@/components/landing/Reveal'
import { observabilityMetrics } from '@/components/landing/content'

const dashboardStats = [
  { label: '成功率', value: '99.96%' },
  { label: 'P95 延迟', value: '1.24s' },
  { label: '费用', value: '¥ 3,842.60' },
]

const trendPoints = [
  [0, 132],
  [60, 118],
  [120, 124],
  [180, 96],
  [240, 104],
  [300, 72],
  [360, 80],
  [420, 54],
  [480, 62],
  [540, 34],
  [600, 42],
]

const trendLine = trendPoints.map(([x, y]) => `${x},${y}`).join(' ')
const trendArea = `0,180 ${trendLine} 600,180`

export default function ObservabilitySection() {
  return (
    <Section id="observability" className="border-t">
      <div className="grid items-start gap-12 lg:grid-cols-2 lg:gap-16">
        <Reveal>
          <SectionHeading
            eyebrow="可观测"
            title="每一次调用，都留下完整的证据链"
            description="从网关到上游供应商，请求量、成功率、Token、费用与延迟全程可查，问题定位与成本对账不再靠猜。"
          />
          <dl className="mt-10 divide-y divide-border border-t">
            {observabilityMetrics.map((metric) => (
              <div
                key={metric.label}
                className="flex flex-col gap-1 py-4 sm:flex-row sm:items-baseline sm:justify-between sm:gap-8"
              >
                <dt className="shrink-0 text-sm font-semibold">
                  {metric.label}
                </dt>
                <dd className="text-sm leading-6 text-muted-foreground sm:text-right">
                  {metric.desc}
                </dd>
              </div>
            ))}
          </dl>
        </Reveal>

        <Reveal>
          <div className="overflow-hidden rounded-2xl border bg-marketing-panel">
            <div className="flex items-start justify-between gap-4 border-b px-5 py-4">
              <div>
                <p className="text-xs font-medium text-muted-foreground">
                  请求量
                </p>
                <p className="mt-1 text-2xl font-semibold tracking-tight tabular-nums">
                  1,284,930
                </p>
              </div>
              <span className="rounded-full bg-success/10 px-2.5 py-1 text-xs font-medium text-success tabular-nums">
                +12.4%
              </span>
            </div>

            <div className="px-5 pt-5">
              <svg
                viewBox="0 0 600 180"
                preserveAspectRatio="none"
                role="img"
                aria-label="请求量趋势折线示意图（示例数据）"
                className="h-40 w-full text-primary"
              >
                <defs>
                  <linearGradient
                    id="observability-area"
                    x1="0"
                    y1="0"
                    x2="0"
                    y2="1"
                  >
                    <stop
                      offset="0%"
                      stopColor="currentColor"
                      stopOpacity="0.22"
                    />
                    <stop
                      offset="100%"
                      stopColor="currentColor"
                      stopOpacity="0"
                    />
                  </linearGradient>
                </defs>
                {[30, 70, 110, 150].map((y) => (
                  <line
                    key={y}
                    x1="0"
                    y1={y}
                    x2="600"
                    y2={y}
                    stroke="currentColor"
                    strokeOpacity="0.08"
                    strokeWidth="1"
                    vectorEffect="non-scaling-stroke"
                  />
                ))}
                <polygon
                  points={trendArea}
                  fill="url(#observability-area)"
                />
                <polyline
                  points={trendLine}
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  vectorEffect="non-scaling-stroke"
                />
              </svg>
              <div
                aria-hidden="true"
                className="mt-2 flex justify-between text-[11px] text-muted-foreground tabular-nums"
              >
                <span>09-10</span>
                <span>09-13</span>
                <span>09-16</span>
                <span>09-19</span>
                <span>09-22</span>
              </div>
            </div>

            <div className="mt-5 grid grid-cols-3 divide-x border-t">
              {dashboardStats.map((stat) => (
                <div key={stat.label} className="px-4 py-4 sm:px-5">
                  <p className="text-xs text-muted-foreground">{stat.label}</p>
                  <p className="mt-1 text-sm font-semibold tabular-nums">
                    {stat.value}
                  </p>
                </div>
              ))}
            </div>

            <p className="border-t px-5 py-3 text-[11px] text-muted-foreground">
              示例数据 · 仅用于展示看板形态
            </p>
          </div>
        </Reveal>
      </div>
    </Section>
  )
}
