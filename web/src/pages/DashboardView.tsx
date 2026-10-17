import { useEffect, type ReactNode } from 'react'
import {
  Activity,
  CheckCircle2,
  Hash,
  JapaneseYen,
  Timer,
  type LucideIcon,
} from 'lucide-react'

import DashboardTrendChart, {
  type TrendDatum,
} from '@/components/DashboardTrendChart'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { formatTime } from '@/lib/format'
import { useAuthStore } from '@/stores/auth'
import { useDashboardStore } from '@/stores/dashboard'
import type { DashboardRange, DashboardTrendPoint } from '@/types/api'

const RANGE_TABS: { value: DashboardRange; label: string }[] = [
  { value: 'today', label: '今日' },
  { value: '7d', label: '近 7 天' },
  { value: '30d', label: '近 30 天' },
]

const percent = (rate: number) => `${(rate * 100).toFixed(2)}%`

const numberText = (value: number) => value.toLocaleString('en-US')

// Token 等大数值的紧凑展示
function compactNumber(value: number): string {
  if (value >= 1e9) return `${(value / 1e9).toFixed(2)}B`
  if (value >= 1e6) return `${(value / 1e6).toFixed(2)}M`
  if (value >= 1e3) return `${(value / 1e3).toFixed(2)}K`
  return String(value)
}

function formatMoney(value: number, currency: string): string {
  const symbol =
    currency === 'USD' ? '$' : currency === 'CNY' ? '¥' : ''
  if (value === 0) return `${symbol}0.00`
  if (value < 1) return `${symbol}${value.toFixed(4)}`
  return `${symbol}${value.toFixed(2)}`
}

// 趋势分桶标签：今日按小时、近 N 天按日期
function bucketLabel(iso: string, range: DashboardRange): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  const pad = (n: number) => String(n).padStart(2, '0')
  if (range === 'today') return `${pad(date.getHours())}:00`
  return `${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

interface OverviewCard {
  title: string
  value: string
  description: string
  icon: LucideIcon
  iconClass: string
}

export default function DashboardView() {
  const username = useAuthStore((s) => s.username)
  const isAdmin = useAuthStore((s) => s.isAdmin)
  const range = useDashboardStore((s) => s.range)
  const stats = useDashboardStore((s) => s.stats)
  const loading = useDashboardStore((s) => s.loading)
  const error = useDashboardStore((s) => s.error)
  const setRange = useDashboardStore((s) => s.setRange)
  const fetchStats = useDashboardStore((s) => s.fetchStats)

  useEffect(() => {
    void fetchStats(range)
  }, [fetchStats, range])

  const overview = stats?.overview
  // 费用与账户余额统一按人民币（CNY）展示
  const currency = overview?.currency ?? 'CNY'

  const buildTrend = (
    pick: (p: DashboardTrendPoint) => number,
    unit: string,
  ): TrendDatum[] =>
    (stats?.trend ?? []).map((p) => {
      const label = bucketLabel(p.bucket_start, range)
      const value = pick(p)
      return {
        label,
        value,
        title: `${label} · ${unit} ${unit === '费用' ? formatMoney(value, currency) : numberText(value)}`,
      }
    })

  const overviewCards: OverviewCard[] = [
    {
      title: '请求数',
      value: overview ? numberText(overview.request_count) : '-',
      description: overview
        ? `${numberText(overview.success_count)} 成功 · ${numberText(overview.error_count)} 失败`
        : '',
      icon: Activity,
      iconClass: 'bg-chart-1/10 text-chart-1',
    },
    {
      title: '成功率',
      value: overview ? percent(overview.success_rate) : '-',
      description: overview ? `共 ${numberText(overview.request_count)} 次请求` : '',
      icon: CheckCircle2,
      iconClass: 'bg-success/10 text-success',
    },
    {
      title: 'Token 用量',
      value: overview ? compactNumber(overview.total_tokens) : '-',
      description: overview
        ? `输入 ${compactNumber(overview.input_tokens)} · 输出 ${compactNumber(overview.output_tokens)}`
        : '',
      icon: Hash,
      iconClass: 'bg-chart-2/10 text-chart-2',
    },
    {
      title: '费用',
      value: overview ? formatMoney(overview.total_cost, currency) : '-',
      description: '所选范围累计计费',
      icon: JapaneseYen,
      iconClass: 'bg-warning/10 text-warning',
    },
    {
      title: '平均延迟',
      value: overview ? formatTime(overview.avg_latency_ms) : '-',
      description: overview ? `TTFT ${formatTime(overview.avg_ttft_ms)}` : '',
      icon: Timer,
      iconClass: 'bg-chart-3/10 text-chart-3',
    },
  ]

  const trendCharts: {
    title: string
    description: string
    data: TrendDatum[]
    color: string
    format: (v: number) => string
  }[] = [
    {
      title: '请求量趋势',
      description: range === 'today' ? '按小时统计' : '按天统计',
      data: buildTrend((p) => p.request_count, '请求数'),
      color: 'var(--color-chart-1)',
      format: (v) => compactNumber(v),
    },
    {
      title: 'Token 使用趋势',
      description: range === 'today' ? '按小时统计' : '按天统计',
      data: buildTrend((p) => p.total_tokens, 'Token'),
      color: 'var(--color-chart-2)',
      format: (v) => compactNumber(v),
    },
    {
      title: '费用趋势',
      description: range === 'today' ? '按小时统计' : '按天统计',
      data: buildTrend((p) => p.total_cost, '费用'),
      color: 'var(--color-chart-3)',
      format: (v) => formatMoney(v, currency),
    },
  ]

  const hasTrend = (stats?.trend ?? []).length > 0
  const topModels = stats?.top_models ?? []
  const topChannels = stats?.top_channels ?? []
  const topErrors = stats?.top_errors ?? []

  return (
    <div className="w-full space-y-6">
      {/* 欢迎区域 */}
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="mb-1 text-2xl font-semibold tracking-tight text-foreground">
            欢迎回来，{username || '管理员'}
          </h1>
          <p className="text-sm text-muted-foreground">
            Xin API 网关控制台 · 网关请求、用量与费用总览
          </p>
        </div>
        <Tabs
          value={range}
          onValueChange={(value) => setRange(value as DashboardRange)}
        >
          <TabsList>
            {RANGE_TABS.map((tab) => (
              <TabsTrigger key={tab.value} value={tab.value}>
                {tab.label}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </div>

      {error ? (
        <p className="rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          {error}
        </p>
      ) : null}

      {/* 概览指标 */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
        {overviewCards.map((card) => {
          const Icon = card.icon
          return (
            <Card key={card.title} className="gap-0 py-5">
              <CardContent className="px-5">
                {loading ? (
                  <div className="space-y-3">
                    <Skeleton className="h-4 w-16" />
                    <Skeleton className="h-8 w-24" />
                  </div>
                ) : (
                  <>
                    <div className="mb-3 flex items-center justify-between">
                      <p className="text-sm text-muted-foreground">
                        {card.title}
                      </p>
                      <span
                        className={`flex size-8 items-center justify-center rounded-lg ${card.iconClass}`}
                      >
                        <Icon className="size-4" />
                      </span>
                    </div>
                    <p className="text-2xl font-semibold leading-none tracking-tight text-foreground tabular-nums">
                      {card.value}
                    </p>
                    <p className="mt-2 min-h-4 text-xs text-muted-foreground">
                      {card.description}
                    </p>
                  </>
                )}
              </CardContent>
            </Card>
          )
        })}
      </div>

      {/* 趋势 */}
      <div className="grid gap-4 lg:grid-cols-3">
        {trendCharts.map((chart) => (
          <Card key={chart.title}>
            <CardHeader>
              <CardTitle className="text-base">{chart.title}</CardTitle>
              <CardDescription>{chart.description}</CardDescription>
            </CardHeader>
            <CardContent>
              {loading ? (
                <Skeleton className="h-56 w-full" />
              ) : hasTrend ? (
                <DashboardTrendChart
                  data={chart.data}
                  color={chart.color}
                  formatValue={chart.format}
                  ariaLabel={chart.title}
                />
              ) : (
                <p className="py-16 text-center text-sm text-muted-foreground">
                  暂无数据
                </p>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      {/* 排行 */}
      <div
        className={
          isAdmin ? 'grid gap-4 lg:grid-cols-3' : 'grid gap-4 lg:grid-cols-2'
        }
      >
        {/* 模型调用排行 */}
        <RankCard
          title="模型调用排行"
          description="按请求数 Top 5"
          loading={loading}
          empty={topModels.length === 0}
        >
          {topModels.map((row, index) => (
            <RankRow key={row.model} index={index} name={row.model}>
              {numberText(row.request_count)} 次 ·{' '}
              {compactNumber(row.total_tokens)} tokens ·{' '}
              {formatMoney(row.total_cost, currency)}
            </RankRow>
          ))}
        </RankCard>

        {/* 渠道调用排行（渠道属于上游资源，仅管理员可见） */}
        {isAdmin && (
          <RankCard
            title="渠道调用排行"
            description="按请求数 Top 5"
            loading={loading}
            empty={topChannels.length === 0}
          >
            {topChannels.map((row, index) => (
              <RankRow
                key={row.channel_id}
                index={index}
                name={row.channel_name ?? `#${row.channel_id}`}
              >
                {numberText(row.request_count)} 次 · {percent(row.success_rate)}{' '}
                · {formatTime(row.avg_latency_ms)}
              </RankRow>
            ))}
          </RankCard>
        )}

        {/* 错误排行 */}
        <RankCard
          title="错误排行"
          description="按错误类型 Top 5"
          loading={loading}
          empty={topErrors.length === 0}
        >
          {topErrors.map((row, index) => (
            <RankRow key={row.error_type} index={index} name={row.error_type}>
              {numberText(row.count)} 次
            </RankRow>
          ))}
        </RankCard>
      </div>
    </div>
  )
}

function RankCard({
  title,
  description,
  loading,
  empty,
  children,
}: {
  title: string
  description: string
  loading: boolean
  empty: boolean
  children: ReactNode
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="space-y-3">
            <Skeleton className="h-6 w-full" />
            <Skeleton className="h-6 w-full" />
            <Skeleton className="h-6 w-full" />
          </div>
        ) : empty ? (
          <p className="py-8 text-center text-sm text-muted-foreground">
            暂无数据
          </p>
        ) : (
          <div className="space-y-1">{children}</div>
        )}
      </CardContent>
    </Card>
  )
}

function RankRow({
  index,
  name,
  children,
}: {
  index: number
  name: string
  children: ReactNode
}) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-md px-2 py-2 text-sm odd:bg-muted/40">
      <div className="flex min-w-0 items-center gap-2">
        <span className="flex size-5 shrink-0 items-center justify-center rounded text-xs font-medium text-muted-foreground tabular-nums">
          {index + 1}
        </span>
        <span className="truncate font-medium text-foreground">{name}</span>
      </div>
      <span className="shrink-0 text-xs text-muted-foreground tabular-nums">
        {children}
      </span>
    </div>
  )
}
