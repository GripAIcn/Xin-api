import { useEffect, useMemo, useState } from 'react'
import { Loader2 } from 'lucide-react'
import { toast } from 'sonner'

import MetricTrendChart from '@/components/MetricTrendChart'
import PageHeader from '@/components/PageHeader'
import { DotBadge, type StatusTone } from '@/components/StatusBadge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { formatDate, formatTime } from '@/lib/format'
import { useChannelStore } from '@/stores/channels'
import { useMetricStore } from '@/stores/metrics'
import type {
  ChannelMetricDimension,
  ChannelMetricRange,
  ChannelMetricSummary,
} from '@/types/api'

const RANGE_OPTIONS: { value: ChannelMetricRange; label: string }[] = [
  { value: '1h', label: '最近 1 小时' },
  { value: '6h', label: '最近 6 小时' },
  { value: '24h', label: '最近 24 小时' },
  { value: '7d', label: '最近 7 天' },
]

const DIMENSION_OPTIONS: { value: ChannelMetricDimension; label: string }[] = [
  { value: 'channel', label: '按渠道' },
  { value: 'model', label: '按模型' },
  { value: 'channel_model', label: '渠道 × 模型' },
]

const ALL_CHANNELS = 'all'

const percent = (rate: number) => `${(rate * 100).toFixed(2)}%`

const successTone = (rate: number): StatusTone => {
  if (rate >= 0.99) return 'success'
  if (rate >= 0.95) return 'warning'
  return 'danger'
}

const numberText = (value: number) => value.toLocaleString('en-US')

const channelText = (row: ChannelMetricSummary) => {
  if (row.channel_name) return row.channel_name
  if (row.channel_id != null) return `#${row.channel_id}`
  return '-'
}

interface OverviewCard {
  title: string
  value: string
  description: string
}

export default function MonitoringView() {
  const summaries = useMetricStore((s) => s.summaries)
  const trend = useMetricStore((s) => s.trend)
  const loading = useMetricStore((s) => s.loading)
  const error = useMetricStore((s) => s.error)
  const fetchAll = useMetricStore((s) => s.fetchAll)

  const channels = useChannelStore((s) => s.channels)
  const fetchChannels = useChannelStore((s) => s.fetchAll)

  const [range, setRange] = useState<ChannelMetricRange>('1h')
  const [dimension, setDimension] =
    useState<ChannelMetricDimension>('channel')
  const [channelId, setChannelId] = useState<string>(ALL_CHANNELS)

  // 渠道筛选下拉数据（仅用于选项，失败不影响指标展示）
  useEffect(() => {
    void fetchChannels().catch(() => {
      /* silently fail */
    })
  }, [fetchChannels])

  useEffect(() => {
    void fetchAll({
      range,
      dimension,
      channel_id: channelId === ALL_CHANNELS ? undefined : Number(channelId),
    })
  }, [fetchAll, range, dimension, channelId])

  useEffect(() => {
    if (error) toast.error(error)
  }, [error])

  // 概览指标：按成功数加权，避免不同渠道请求量差异导致均值失真
  const overview = useMemo<OverviewCard[]>(() => {
    const totalRequests = summaries.reduce((sum, r) => sum + r.request_count, 0)
    const totalSuccess = summaries.reduce((sum, r) => sum + r.success_count, 0)
    const weighted = (pick: (row: ChannelMetricSummary) => number) => {
      if (totalSuccess <= 0) return 0
      return (
        summaries.reduce((sum, r) => sum + pick(r) * r.success_count, 0) /
        totalSuccess
      )
    }

    const avgTtft = weighted((r) => r.avg_ttft_ms)
    const avgLatency = weighted((r) => r.avg_latency_ms)

    return [
      {
        title: '总请求数',
        value: numberText(totalRequests),
        description: '当前时间范围内',
      },
      {
        title: '成功率',
        value: totalRequests > 0 ? percent(totalSuccess / totalRequests) : '-',
        description: `${numberText(totalSuccess)} 次成功`,
      },
      {
        title: '平均 TTFT',
        value: avgTtft > 0 ? formatTime(avgTtft) : '-',
        description: '首字耗时（成功请求）',
      },
      {
        title: '平均延迟',
        value: avgLatency > 0 ? formatTime(avgLatency) : '-',
        description: '端到端耗时（成功请求）',
      },
    ]
  }, [summaries])

  const showChannelColumn = dimension !== 'model'
  const showModelColumn = dimension !== 'channel'
  // 10 个固定列 + 渠道 / 模型标识列
  const columnCount = 10 + Number(showChannelColumn) + Number(showModelColumn)

  const rangeLabel =
    RANGE_OPTIONS.find((o) => o.value === range)?.label ?? ''

  const emptyContent = (
    <p className="py-8 text-center text-sm text-muted-foreground">
      暂无指标数据，聚合任务将按分钟生成
    </p>
  )

  return (
    <div className="mx-auto w-full max-w-7xl space-y-5">
      <PageHeader
        title="渠道监控"
        description="基于请求日志按分钟聚合的渠道 / 模型性能指标，启动时自动回填最近 7 天"
      />

      {/* 筛选栏 */}
      <Card className="py-0">
        <CardContent className="flex flex-wrap items-center gap-3 px-4 py-4">
          <div className="flex flex-col gap-1.5">
            <span className="text-xs text-muted-foreground">时间范围</span>
            <Select
              value={range}
              onValueChange={(v) => setRange(v as ChannelMetricRange)}
            >
              <SelectTrigger className="w-36">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {RANGE_OPTIONS.map((o) => (
                  <SelectItem key={o.value} value={o.value}>
                    {o.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="flex flex-col gap-1.5">
            <span className="text-xs text-muted-foreground">汇总维度</span>
            <Tabs
              value={dimension}
              onValueChange={(v) => setDimension(v as ChannelMetricDimension)}
            >
              <TabsList>
                {DIMENSION_OPTIONS.map((o) => (
                  <TabsTrigger key={o.value} value={o.value}>
                    {o.label}
                  </TabsTrigger>
                ))}
              </TabsList>
            </Tabs>
          </div>

          <div className="flex flex-col gap-1.5">
            <span className="text-xs text-muted-foreground">渠道</span>
            <Select value={channelId} onValueChange={setChannelId}>
              <SelectTrigger className="w-44">
                <SelectValue placeholder="全部渠道" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={ALL_CHANNELS}>全部渠道</SelectItem>
                {channels.map((c) => (
                  <SelectItem key={c.id} value={String(c.id)}>
                    {c.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      {/* 概览卡片 */}
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {overview.map((card) => (
          <Card key={card.title} className="gap-2 py-5">
            <CardContent className="px-5">
              {loading ? (
                <div className="space-y-3">
                  <Skeleton className="h-4 w-20" />
                  <Skeleton className="h-9 w-24" />
                </div>
              ) : (
                <>
                  <p className="mb-1.5 text-sm text-muted-foreground">
                    {card.title}
                  </p>
                  <p className="text-3xl font-semibold leading-none tracking-tight text-foreground tabular-nums">
                    {card.value}
                  </p>
                  <p className="mt-2 text-xs text-muted-foreground">
                    {card.description}
                  </p>
                </>
              )}
            </CardContent>
          </Card>
        ))}
      </div>

      {/* 趋势图 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">TTFT / 延迟走势</CardTitle>
          <CardDescription>
            {rangeLabel}
            {channelId === ALL_CHANNELS
              ? ''
              : ` · ${
                  channels.find((c) => String(c.id) === channelId)?.name ??
                  `渠道 #${channelId}`
                }`}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <Skeleton className="h-64 w-full" />
          ) : trend.length === 0 ? (
            emptyContent
          ) : (
            <MetricTrendChart points={trend} range={range} />
          )}
        </CardContent>
      </Card>

      {/* 明细表 */}
      <Card className="py-0">
        <CardContent className="px-0">
          <Table>
            <TableHeader>
              <TableRow>
                {showChannelColumn && (
                  <TableHead className="min-w-[180px]">渠道</TableHead>
                )}
                {showModelColumn && (
                  <TableHead className="min-w-[150px]">模型</TableHead>
                )}
                <TableHead className="w-[90px]">请求数</TableHead>
                <TableHead className="w-[110px]">成功率</TableHead>
                <TableHead className="w-[100px]">平均 TTFT</TableHead>
                <TableHead className="w-[100px]">平均延迟</TableHead>
                <TableHead className="w-[90px]">TPOT</TableHead>
                <TableHead className="w-[80px]">RPM</TableHead>
                <TableHead className="w-[80px]">TPM</TableHead>
                <TableHead className="min-w-[160px]">Tokens（入/出/总）</TableHead>
                <TableHead className="min-w-[110px]">成本</TableHead>
                <TableHead className="min-w-[160px]">最近窗口</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell
                    colSpan={columnCount}
                    className="h-32 text-center text-muted-foreground"
                  >
                    <span className="inline-flex items-center gap-2">
                      <Loader2 className="size-4 animate-spin" />
                      加载中...
                    </span>
                  </TableCell>
                </TableRow>
              ) : summaries.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={columnCount}
                    className="h-32 text-center text-muted-foreground"
                  >
                    暂无指标数据，聚合任务将按分钟生成
                  </TableCell>
                </TableRow>
              ) : (
                summaries.map((row) => (
                  <TableRow
                    key={`${row.channel_id ?? 'all'}-${row.model ?? 'all'}`}
                  >
                    {showChannelColumn && (
                      <TableCell>
                        <div className="flex flex-col gap-1">
                          <span className="font-medium">
                            {channelText(row)}
                          </span>
                          <span className="flex items-center gap-2 text-xs text-muted-foreground">
                            {row.channel_provider ?? '-'}
                            {row.channel_status ? (
                              <DotBadge
                                tone={
                                  row.channel_status === 'active'
                                    ? 'success'
                                    : row.channel_status === 'circuit_open'
                                      ? 'danger'
                                      : 'muted'
                                }
                              >
                                {row.channel_status === 'active'
                                  ? '正常'
                                  : row.channel_status === 'circuit_open'
                                    ? '熔断'
                                    : '禁用'}
                              </DotBadge>
                            ) : null}
                          </span>
                        </div>
                      </TableCell>
                    )}
                    {showModelColumn && (
                      <TableCell>{row.model ?? '-'}</TableCell>
                    )}
                    <TableCell className="tabular-nums">
                      {numberText(row.request_count)}
                    </TableCell>
                    <TableCell>
                      <DotBadge tone={successTone(row.success_rate)}>
                        {percent(row.success_rate)}
                      </DotBadge>
                    </TableCell>
                    <TableCell className="tabular-nums">
                      {row.avg_ttft_ms > 0 ? formatTime(row.avg_ttft_ms) : '-'}
                    </TableCell>
                    <TableCell className="tabular-nums">
                      {row.avg_latency_ms > 0
                        ? formatTime(row.avg_latency_ms)
                        : '-'}
                    </TableCell>
                    <TableCell className="tabular-nums">
                      {row.tpot > 0 ? `${row.tpot.toFixed(1)}ms` : '-'}
                    </TableCell>
                    <TableCell className="tabular-nums">
                      {row.rpm.toFixed(1)}
                    </TableCell>
                    <TableCell className="tabular-nums">
                      {numberText(Math.round(row.tpm))}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground tabular-nums">
                      {numberText(row.total_input_tokens)} /{' '}
                      {numberText(row.total_output_tokens)} /{' '}
                      {numberText(row.total_tokens)}
                    </TableCell>
                    <TableCell className="tabular-nums">
                      {row.total_cost} {row.currency}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {row.last_window_start
                        ? formatDate(row.last_window_start)
                        : '-'}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}
