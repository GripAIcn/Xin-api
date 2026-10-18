import { useEffect, useState, type ReactNode } from 'react'
import { Loader2, RotateCcw, Search } from 'lucide-react'
import { toast } from 'sonner'

import PageHeader from '@/components/PageHeader'
import { DotBadge } from '@/components/StatusBadge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { fetchLogDetail } from '@/api/modules/logs'
import { formatDate, formatTime } from '@/lib/format'
import { useAuthStore } from '@/stores/auth'
import { useLogsStore } from '@/stores/logs'
import type { LogDetail, LogItem, LogListQuery, LogType } from '@/types/api'

// 类型中文映射（与后端 log_type 枚举一致）
const LOG_TYPE_OPTIONS: { value: LogType; label: string }[] = [
  { value: 'consumption', label: '消费' },
  { value: 'error', label: '错误' },
  { value: 'system', label: '系统' },
  { value: 'management', label: '管理' },
  { value: 'login', label: '登录' },
  { value: 'refund', label: '退款' },
  { value: 'recharge', label: '充值' },
]

const LOG_TYPE_LABEL: Record<string, string> = Object.fromEntries(
  LOG_TYPE_OPTIONS.map((o) => [o.value, o.label]),
)

const PAGE_SIZE_OPTIONS = [10, 20, 50, 100]

const logTypeLabel = (type: string): string => LOG_TYPE_LABEL[type] ?? type

// 空串转 undefined，非空转数字（非法值同样置空）
const toOptionalNumber = (value: string): number | undefined => {
  const trimmed = value.trim()
  if (!trimmed) return undefined
  const num = Number(trimmed)
  return Number.isFinite(num) ? num : undefined
}

// 时间范围预设（不支持自定义时间）
type TimeRange = 'today' | '1d' | '7d' | 'week' | '30d' | 'month'

const TIME_RANGE_OPTIONS: { value: TimeRange; label: string }[] = [
  { value: 'today', label: '当天' },
  { value: '1d', label: '1天' },
  { value: '7d', label: '7天' },
  { value: 'week', label: '本周' },
  { value: '30d', label: '30天' },
  { value: 'month', label: '本月' },
]

const DAY_MS = 24 * 60 * 60 * 1000

const pad = (n: number): string => String(n).padStart(2, '0')

// Date → 后端可解析的 "YYYY-MM-DD HH:mm:ss"
const formatTimeParam = (d: Date): string =>
  `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`

// 按预设计算查询区间：起始时间 → 当前时间
const buildTimeRange = (range: TimeRange): { start_time: string; end_time: string } => {
  const end = new Date()
  const start = new Date(end)
  start.setHours(0, 0, 0, 0)

  switch (range) {
    case 'today':
      break
    case '1d':
      start.setTime(end.getTime() - DAY_MS)
      break
    case '7d':
      start.setTime(end.getTime() - 7 * DAY_MS)
      break
    case 'week': {
      // 周一作为一周起始（getDay：0=周日）
      const day = start.getDay()
      start.setDate(start.getDate() - (day === 0 ? 6 : day - 1))
      break
    }
    case '30d':
      start.setTime(end.getTime() - 30 * DAY_MS)
      break
    case 'month':
      start.setDate(1)
      break
  }

  return { start_time: formatTimeParam(start), end_time: formatTimeParam(end) }
}

const dash = (value: string | number | null | undefined): ReactNode =>
  value === undefined || value === null || value === '' ? '-' : String(value)

// 展示渠道：优先名称，其次 id
const channelText = (row: LogItem): ReactNode => {
  if (row.channel_name) return row.channel_name
  if (row.channel_id != null) return `#${row.channel_id}`
  return '-'
}

const userText = (row: LogItem): ReactNode => {
  if (row.user_name) return row.user_name
  if (row.user_id != null) return `#${row.user_id}`
  return '-'
}

/** 筛选栏单个字段：小号标题 + 控件 */
function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-xs text-muted-foreground">{label}</span>
      {children}
    </div>
  )
}

/** 详情分区标题 */
function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="space-y-3 rounded-lg border p-4">
      <h4 className="text-sm font-semibold">{title}</h4>
      {children}
    </div>
  )
}

function InfoGrid({ children }: { children: ReactNode }) {
  return (
    <div className="grid grid-cols-2 gap-x-4 gap-y-3 sm:grid-cols-3">
      {children}
    </div>
  )
}

function InfoItem({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div className="flex flex-col gap-0.5">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="text-sm break-all">{value}</span>
    </div>
  )
}

/** 日志详情内容（含计价表明细） */
function LogDetailContent({
  detail,
  isAdmin,
}: {
  detail: LogDetail
  isAdmin: boolean
}) {
  const billing = detail.billing
  const currency = billing?.price.currency ?? detail.currency
  // 余额以结算币种（CNY）记账，计价金额仍按模型定价币种展示
  const settleCurrency = billing?.settle_currency || 'CNY'
  const hasError = Boolean(
    detail.error_stage ||
      detail.error_type ||
      detail.error_code ||
      detail.error_message,
  )

  return (
    <div className="space-y-4">
      <Section title="基础信息">
        <InfoGrid>
          <InfoItem label="请求 ID" value={dash(detail.request_id)} />
          {isAdmin && (
            <InfoItem
              label="上游请求 ID"
              value={dash(detail.upstream_req_id)}
            />
          )}
          <InfoItem label="类型" value={logTypeLabel(detail.log_type)} />
          <InfoItem
            label="状态"
            value={detail.status === 'success' ? '成功' : '失败'}
          />
          <InfoItem label="模型" value={dash(detail.model)} />
          {isAdmin && (
            <>
              <InfoItem label="上游模型" value={dash(detail.upstream_model)} />
              <InfoItem label="供应商" value={dash(detail.provider)} />
            </>
          )}
          <InfoItem label="协议" value={dash(detail.protocol)} />
          {isAdmin && <InfoItem label="渠道" value={channelText(detail)} />}
          <InfoItem label="用户" value={userText(detail)} />
          <InfoItem label="API Key ID" value={dash(detail.api_key_id)} />
          <InfoItem label="客户端 IP" value={dash(detail.client_ip)} />
          {isAdmin && (
            <InfoItem label="网关节点" value={dash(detail.gateway_node)} />
          )}
          <InfoItem label="创建时间" value={formatDate(detail.created_at)} />
          <InfoItem
            label="开始时间"
            value={detail.started_at ? formatDate(detail.started_at) : '-'}
          />
          <InfoItem
            label="完成时间"
            value={detail.completed_at ? formatDate(detail.completed_at) : '-'}
          />
        </InfoGrid>
      </Section>

      <Section title="计价表明细">
        {billing ? (
          <div className="space-y-4">
            <div>
              <p className="mb-2 text-xs font-medium text-muted-foreground">
                单价
              </p>
              <InfoGrid>
                <InfoItem
                  label="输入单价"
                  value={dash(billing.price.input_price)}
                />
                <InfoItem
                  label="输出单价"
                  value={dash(billing.price.output_price)}
                />
                <InfoItem
                  label="缓存输入单价"
                  value={dash(billing.price.cached_input_price)}
                />
                <InfoItem
                  label="缓存创建单价"
                  value={dash(billing.price.cache_creation_price)}
                />
                <InfoItem label="倍率" value={dash(billing.price.ratio)} />
                <InfoItem label="币种" value={dash(billing.price.currency)} />
              </InfoGrid>
            </div>

            <div>
              <p className="mb-2 text-xs font-medium text-muted-foreground">
                Tokens
              </p>
              <InfoGrid>
                <InfoItem
                  label="输入 tokens"
                  value={dash(billing.input_tokens)}
                />
                <InfoItem
                  label="输出 tokens"
                  value={dash(billing.output_tokens)}
                />
                <InfoItem
                  label="总计 tokens"
                  value={dash(billing.total_tokens)}
                />
                <InfoItem
                  label="缓存输入 tokens"
                  value={dash(billing.cached_input_tokens)}
                />
                <InfoItem
                  label="缓存创建 tokens"
                  value={dash(billing.cache_creation_tokens)}
                />
                <InfoItem
                  label="推理 tokens"
                  value={dash(detail.reasoning_tokens)}
                />
              </InfoGrid>
            </div>

            <div>
              <p className="mb-2 text-xs font-medium text-muted-foreground">
                成本
              </p>
              <InfoGrid>
                <InfoItem
                  label="输入成本"
                  value={`${billing.input_cost} ${currency}`}
                />
                <InfoItem
                  label="输出成本"
                  value={`${billing.output_cost} ${currency}`}
                />
                <InfoItem
                  label="缓存输入成本"
                  value={`${billing.cached_input_cost} ${currency}`}
                />
                <InfoItem
                  label="缓存创建成本"
                  value={`${billing.cache_creation_cost} ${currency}`}
                />
                <InfoItem
                  label="总成本"
                  value={`${billing.total_cost} ${currency}`}
                />
                <InfoItem
                  label="结算金额"
                  value={`${billing.settle_cost} ${settleCurrency}`}
                />
                <InfoItem
                  label="结算汇率"
                  value={
                    billing.exchange_rate != null
                      ? `1 ${currency} = ${billing.exchange_rate} ${settleCurrency}`
                      : '-'
                  }
                />
              </InfoGrid>
            </div>

            <div>
              <p className="mb-2 text-xs font-medium text-muted-foreground">
                余额变动
              </p>
              <p className="text-sm">
                {billing.balance_before != null
                  ? `${billing.balance_before} ${settleCurrency}`
                  : '-'}
                <span className="mx-1.5 text-muted-foreground">→</span>
                {billing.balance_after != null
                  ? `${billing.balance_after} ${settleCurrency}`
                  : '-'}
              </p>
            </div>
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">无计费记录</p>
        )}
      </Section>

      <Section title="运行信息">
        <InfoGrid>
          <InfoItem label="用量来源" value={dash(detail.usage_source)} />
          {isAdmin && (
            <>
              <InfoItem label="路由策略" value={dash(detail.route_strategy)} />
              <InfoItem label="路由权重" value={dash(detail.route_weight)} />
            </>
          )}
          <InfoItem label="重试次数" value={dash(detail.retry_count)} />
          {isAdmin && (
            <InfoItem
              label="重试渠道"
              value={
                detail.retry_channels?.length
                  ? detail.retry_channels.join(', ')
                  : '-'
              }
            />
          )}
          <InfoItem label="是否流式" value={detail.is_stream ? '是' : '否'} />
          <InfoItem label="是否取消" value={detail.is_cancelled ? '是' : '否'} />
          <InfoItem label="取消方" value={dash(detail.cancelled_by)} />
          <InfoItem
            label="首字耗时"
            value={detail.ttft_ms != null ? formatTime(detail.ttft_ms) : '-'}
          />
          <InfoItem
            label="生成耗时"
            value={
              detail.generation_ms != null
                ? formatTime(detail.generation_ms)
                : '-'
            }
          />
          <InfoItem label="结束原因" value={dash(detail.finish_reason)} />
          <InfoItem label="状态码" value={dash(detail.status_code)} />
        </InfoGrid>
      </Section>

      {hasError && (
        <Section title="错误信息">
          <InfoGrid>
            <InfoItem label="错误阶段" value={dash(detail.error_stage)} />
            <InfoItem label="错误类型" value={dash(detail.error_type)} />
            <InfoItem label="错误码" value={dash(detail.error_code)} />
            <InfoItem label="错误信息" value={dash(detail.error_message)} />
          </InfoGrid>
        </Section>
      )}
    </div>
  )
}

export default function LogsView() {
  const list = useLogsStore((s) => s.list)
  const total = useLogsStore((s) => s.total)
  const page = useLogsStore((s) => s.page)
  const pageSize = useLogsStore((s) => s.pageSize)
  const loading = useLogsStore((s) => s.loading)
  const fetchList = useLogsStore((s) => s.fetchList)
  const setFilter = useLogsStore((s) => s.setFilter)
  const resetFilters = useLogsStore((s) => s.resetFilters)
  const setPage = useLogsStore((s) => s.setPage)
  const setPageSize = useLogsStore((s) => s.setPageSize)
  const isAdmin = useAuthStore((s) => s.isAdmin)

  // 筛选栏本地态（点击「查询」时才写入 store）
  const [logType, setLogType] = useState('all')
  const [model, setModel] = useState('')
  const [channelId, setChannelId] = useState('')
  const [userId, setUserId] = useState('')
  const [apiKeyId, setApiKeyId] = useState('')
  const [requestId, setRequestId] = useState('')
  const [provider, setProvider] = useState('')
  const [protocol, setProtocol] = useState('')
  const [upstreamReqId, setUpstreamReqId] = useState('')
  const [isStream, setIsStream] = useState('all')
  const [status, setStatus] = useState('all')
  const [timeRange, setTimeRange] = useState<TimeRange>('today')

  const [detailOpen, setDetailOpen] = useState(false)
  const [detail, setDetail] = useState<LogDetail | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  useEffect(() => {
    // 首次进入按默认时间范围（当天）查询
    setFilter(buildTimeRange(timeRange))
    void fetchList().catch((e) => {
      toast.error((e as Error)?.message || '加载日志列表失败')
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fetchList])

  const buildFilters = (): LogListQuery => ({
    log_type: logType === 'all' ? undefined : (logType as LogType),
    model: model.trim() || undefined,
    channel_id: toOptionalNumber(channelId),
    user_id: toOptionalNumber(userId),
    api_key_id: toOptionalNumber(apiKeyId),
    request_id: requestId.trim() || undefined,
    provider: provider.trim() || undefined,
    protocol: protocol.trim() || undefined,
    upstream_req_id: upstreamReqId.trim() || undefined,
    is_stream: isStream === 'all' ? undefined : isStream === 'true',
    status: status === 'all' ? undefined : (status as LogListQuery['status']),
    ...buildTimeRange(timeRange),
  })

  const handleSearch = async () => {
    setFilter(buildFilters())
    try {
      await fetchList()
    } catch (e) {
      toast.error((e as Error)?.message || '加载日志列表失败')
    }
  }

  const handleReset = async () => {
    setLogType('all')
    setModel('')
    setChannelId('')
    setUserId('')
    setApiKeyId('')
    setRequestId('')
    setProvider('')
    setProtocol('')
    setUpstreamReqId('')
    setIsStream('all')
    setStatus('all')
    setTimeRange('today')
    resetFilters()
    setFilter(buildTimeRange('today'))
    try {
      await fetchList()
    } catch (e) {
      toast.error((e as Error)?.message || '加载日志列表失败')
    }
  }

  const handlePrev = async () => {
    if (page <= 1) return
    setPage(page - 1)
    try {
      await fetchList()
    } catch (e) {
      toast.error((e as Error)?.message || '加载日志列表失败')
    }
  }

  const handleNext = async () => {
    if (page >= totalPages) return
    setPage(page + 1)
    try {
      await fetchList()
    } catch (e) {
      toast.error((e as Error)?.message || '加载日志列表失败')
    }
  }

  const handlePageSizeChange = async (value: string) => {
    setPageSize(Number(value))
    try {
      await fetchList()
    } catch (e) {
      toast.error((e as Error)?.message || '加载日志列表失败')
    }
  }

  const openDetail = async (row: LogItem) => {
    setDetailOpen(true)
    setDetail(null)
    setDetailLoading(true)
    try {
      const data = await fetchLogDetail(row.id)
      setDetail(data)
    } catch (e) {
      toast.error((e as Error)?.message || '加载日志详情失败')
    } finally {
      setDetailLoading(false)
    }
  }

  return (
    <div className="w-full space-y-5">
      <PageHeader
        className="mb-5"
        title="日志"
        description="查询使用日志与消费记录"
      />

      {/* 筛选栏 */}
      <Card className="py-0">
        <CardContent className="px-4 py-4">
          <div className="flex flex-wrap items-end gap-3">
            <Field label="类型">
              <Select value={logType} onValueChange={setLogType}>
                <SelectTrigger className="w-32">
                  <SelectValue placeholder="全部类型" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部类型</SelectItem>
                  {LOG_TYPE_OPTIONS.map((o) => (
                    <SelectItem key={o.value} value={o.value}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>

            <Field label="模型">
              <Input
                className="w-40"
                placeholder="模型名"
                value={model}
                onChange={(e) => setModel(e.target.value)}
              />
            </Field>

            {isAdmin && (
              <Field label="渠道 ID">
                <Input
                  className="w-28"
                  inputMode="numeric"
                  placeholder="渠道 ID"
                  value={channelId}
                  onChange={(e) => setChannelId(e.target.value)}
                />
              </Field>
            )}

            {isAdmin && (
              <Field label="用户 ID">
                <Input
                  className="w-28"
                  inputMode="numeric"
                  placeholder="用户 ID"
                  value={userId}
                  onChange={(e) => setUserId(e.target.value)}
                />
              </Field>
            )}

            <Field label="API Key ID">
              <Input
                className="w-28"
                inputMode="numeric"
                placeholder="Key ID"
                value={apiKeyId}
                onChange={(e) => setApiKeyId(e.target.value)}
              />
            </Field>

            <Field label="请求 ID">
              <Input
                className="w-48"
                placeholder="request_id"
                value={requestId}
                onChange={(e) => setRequestId(e.target.value)}
              />
            </Field>

            {isAdmin && (
              <Field label="上游请求 ID">
                <Input
                  className="w-48"
                  placeholder="upstream_req_id"
                  value={upstreamReqId}
                  onChange={(e) => setUpstreamReqId(e.target.value)}
                />
              </Field>
            )}

            {isAdmin && (
              <Field label="供应商">
                <Input
                  className="w-32"
                  placeholder="provider"
                  value={provider}
                  onChange={(e) => setProvider(e.target.value)}
                />
              </Field>
            )}

            {isAdmin && (
              <Field label="协议">
                <Input
                  className="w-32"
                  placeholder="protocol"
                  value={protocol}
                  onChange={(e) => setProtocol(e.target.value)}
                />
              </Field>
            )}

            <Field label="是否流式">
              <Select value={isStream} onValueChange={setIsStream}>
                <SelectTrigger className="w-24">
                  <SelectValue placeholder="全部" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部</SelectItem>
                  <SelectItem value="true">是</SelectItem>
                  <SelectItem value="false">否</SelectItem>
                </SelectContent>
              </Select>
            </Field>

            <Field label="状态">
              <Select value={status} onValueChange={setStatus}>
                <SelectTrigger className="w-28">
                  <SelectValue placeholder="全部状态" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部状态</SelectItem>
                  <SelectItem value="success">成功</SelectItem>
                  <SelectItem value="error">失败</SelectItem>
                </SelectContent>
              </Select>
            </Field>

            <Field label="时间范围">
              <Select
                value={timeRange}
                onValueChange={(v) => setTimeRange(v as TimeRange)}
              >
                <SelectTrigger className="w-32">
                  <SelectValue placeholder="时间范围" />
                </SelectTrigger>
                <SelectContent>
                  {TIME_RANGE_OPTIONS.map((o) => (
                    <SelectItem key={o.value} value={o.value}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>

            <div className="flex items-center gap-2">
              <Button onClick={() => void handleSearch()} disabled={loading}>
                <Search />
                查询
              </Button>
              <Button
                variant="outline"
                onClick={() => void handleReset()}
                disabled={loading}
              >
                <RotateCcw />
                重置
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* 数据表格 */}
      <Card className="overflow-hidden py-0">
        <CardContent className="px-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="min-w-[150px]">时间</TableHead>
                <TableHead className="w-[80px]">类型</TableHead>
                <TableHead className="min-w-[200px]">请求 ID</TableHead>
                {isAdmin && (
                  <TableHead className="min-w-[160px]">
                    上游请求 ID
                  </TableHead>
                )}
                <TableHead className="min-w-[140px]">模型</TableHead>
                {isAdmin && (
                  <TableHead className="min-w-[120px]">渠道</TableHead>
                )}
                <TableHead className="w-[80px]">状态</TableHead>
                <TableHead className="min-w-[160px]">Tokens（入/出/总）</TableHead>
                <TableHead className="w-[90px]">耗时</TableHead>
                <TableHead className="min-w-[120px]">费用</TableHead>
                <TableHead className="w-[80px]">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell
                    colSpan={isAdmin ? 11 : 9}
                    className="h-32 text-center text-muted-foreground"
                  >
                    <span className="inline-flex items-center gap-2">
                      <Loader2 className="size-4 animate-spin" />
                      加载中...
                    </span>
                  </TableCell>
                </TableRow>
              ) : list.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={isAdmin ? 11 : 9}
                    className="h-32 text-center text-muted-foreground"
                  >
                    暂无日志
                  </TableCell>
                </TableRow>
              ) : (
                list.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell>{formatDate(row.created_at)}</TableCell>
                    <TableCell>{logTypeLabel(row.log_type)}</TableCell>
                    <TableCell>
                      <span
                        className="block max-w-[220px] truncate"
                        title={row.request_id}
                      >
                        {row.request_id}
                      </span>
                    </TableCell>
                    {isAdmin && (
                      <TableCell>
                        {row.upstream_req_id ? (
                          <span
                            className="block max-w-[180px] truncate"
                            title={row.upstream_req_id}
                          >
                            {row.upstream_req_id}
                          </span>
                        ) : (
                          '-'
                        )}
                      </TableCell>
                    )}
                    <TableCell>{dash(row.model)}</TableCell>
                    {isAdmin && <TableCell>{channelText(row)}</TableCell>}
                    <TableCell>
                      <DotBadge
                        tone={row.status === 'success' ? 'success' : 'danger'}
                      >
                        {row.status === 'success' ? '成功' : '失败'}
                      </DotBadge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {row.input_tokens} / {row.output_tokens} /{' '}
                      {row.total_tokens}
                    </TableCell>
                    <TableCell>
                      {row.latency_ms != null ? formatTime(row.latency_ms) : '-'}
                    </TableCell>
                    <TableCell>
                      {row.total_cost} {row.currency}
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="link"
                        size="sm"
                        className="h-auto p-0"
                        onClick={() => void openDetail(row)}
                      >
                        详情
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* 分页 */}
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-muted-foreground">
          共 {total} 条，第 {page} / {totalPages} 页
        </p>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <span className="text-sm text-muted-foreground">每页</span>
            <Select
              value={String(pageSize)}
              onValueChange={(v) => void handlePageSizeChange(v)}
            >
              <SelectTrigger className="w-20" size="sm">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {PAGE_SIZE_OPTIONS.map((size) => (
                  <SelectItem key={size} value={String(size)}>
                    {size}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={loading || page <= 1}
              onClick={() => void handlePrev()}
            >
              上一页
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={loading || page >= totalPages}
              onClick={() => void handleNext()}
            >
              下一页
            </Button>
          </div>
        </div>
      </div>

      {/* 详情弹窗 */}
      <Dialog open={detailOpen} onOpenChange={setDetailOpen}>
        <DialogContent className="sm:max-w-3xl">
          <DialogHeader>
            <DialogTitle>日志详情</DialogTitle>
            <DialogDescription className="sr-only">
              日志与消费记录详情
            </DialogDescription>
          </DialogHeader>
          {detailLoading ? (
            <div className="flex h-40 items-center justify-center text-muted-foreground">
              <span className="inline-flex items-center gap-2">
                <Loader2 className="size-4 animate-spin" />
                加载中...
              </span>
            </div>
          ) : detail ? (
            <LogDetailContent detail={detail} isAdmin={isAdmin} />
          ) : (
            <p className="py-6 text-center text-sm text-muted-foreground">
              暂无详情
            </p>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
