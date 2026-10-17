const stats = [
  {
    value: '3 种',
    label: '入站协议',
    detail: 'OpenAI · Anthropic · Gemini',
  },
  {
    value: '4 家',
    label: '上游供应商',
    detail: 'OpenAI · Claude · Gemini · DeepSeek',
  },
  {
    value: '4 维',
    label: '智能路由',
    detail: '健康度 · 延迟 · 成本 · 限流',
  },
  {
    value: '1 条',
    label: '追踪链路',
    detail: '单请求 xin_req_* Request ID',
  },
]

/** 紧贴 Hero 下沿的能力指标条，一行 4 项，移动端 2×2 */
export default function StatsStrip() {
  return (
    <section className="border-y bg-background">
      <div className="mx-auto w-full max-w-[1440px] px-6 lg:px-10">
        <dl className="grid grid-cols-2 gap-px bg-border lg:grid-cols-4">
          {stats.map((stat) => (
            <div
              key={stat.label}
              className="bg-background px-4 py-6 sm:px-6 sm:py-7"
            >
              <dd className="text-2xl font-semibold tracking-tight tabular-nums">
                {stat.value}
              </dd>
              <dt className="mt-1 text-sm font-medium">{stat.label}</dt>
              <p className="mt-1 text-xs text-muted-foreground">
                {stat.detail}
              </p>
            </div>
          ))}
        </dl>
      </div>
    </section>
  )
}
