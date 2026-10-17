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
import { formatDate } from '@/lib/format'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth'
import { useBillingStore } from '@/stores/billing'
import type { BillingLogItem, BillingLogQuery, BillingLogSource } from '@/types/api'

// 类型中文映射（与后端 source 枚举一致）
const SOURCE_OPTIONS: { value: BillingLogSource; label: string }[] = [
  { value: 'consumption', label: '消费' },
  { value: 'recharge', label: '充值' },
  { value: 'refund', label: '退款' },
]

const SOURCE_LABEL: Record<string, string> = Object.fromEntries(
  SOURCE_OPTIONS.map((o) => [o.value, o.label]),
)

// 消费行状态中文映射（billing_records.status）
const CONSUMPTION_STATUS_LABEL: Record<string, string> = {
  settled: '已结算',
  success: '成功',
  failed: '失败',
  refunded: '已退款',
}

const PAGE_SIZE_OPTIONS = [10, 20, 50, 100]

const sourceLabel = (source: string): string => SOURCE_LABEL[source] ?? source

const billingStatusText = (row: BillingLogItem): string => {
  // 充值 / 退款为入账事件，恒为成功
  if (row.source !== 'consumption') return '成功'
  return CONSUMPTION_STATUS_LABEL[row.status] ?? row.status
}

const billingStatusTone = (row: BillingLogItem): 'success' | 'danger' =>
  row.source === 'consumption' && row.status === 'failed' ? 'danger' : 'success'

// 空串转 undefined，非空转数字（非法值同样置空）
const toOptionalNumber = (value: string): number | undefined => {
  const trimmed = value.trim()
  if (!trimmed) return undefined
  const num = Number(trimmed)
  return Number.isFinite(num) ? num : undefined
}

// datetime-local（YYYY-MM-DDTHH:mm）→ 后端可解析的 "YYYY-MM-DD HH:mm:ss"
const toBackendTime = (value: string): string | undefined => {
  const trimmed = value.trim()
  if (!trimmed) return undefined
  const normalized = trimmed.includes('T') ? trimmed.replace('T', ' ') : trimmed
  if (/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}$/.test(normalized)) {
    return `${normalized}:00`
  }
  return normalized
}

const dash = (value: string | number | null | undefined): ReactNode =>
  value === undefined || value === null || value === '' ? '-' : String(value)

const money = (value: number, currency?: string): string =>
  currency ? `${value} ${currency}` : String(value)

// 带符号金额：消费为负（-）、充值 / 退款为正（+）
const formatAmount = (amount: number, currency?: string): string => {
  const sign = amount < 0 ? '-' : '+'
  const text = `${sign}${Math.abs(amount).toFixed(6)}`
  return currency ? `${text} ${currency}` : text
}

const amountToneClass = (amount: number): string =>
  amount < 0 ? 'text-destructive' : 'text-success'

const userText = (row: BillingLogItem): ReactNode => {
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

/** 余额变动：before → after，任一缺失显示 - */
function BalanceChange({ row }: { row: BillingLogItem }) {
  if (row.balance_before == null || row.balance_after == null) return <>-</>
  return (
    <span className="whitespace-nowrap">
      {money(row.balance_before, row.currency)}
      <span className="mx-1.5 text-muted-foreground">→</span>
      {money(row.balance_after, row.currency)}
    </span>
  )
}

/** 账单详情内容（纯展示，直接用列表行数据渲染） */
function BillingDetailContent({ row }: { row: BillingLogItem }) {
  const isConsumption = row.source === 'consumption'
  // 单价 / 倍率 / tokens 仅消费行有值，充值 / 退款行显示 -
  const numValue = (value: number): ReactNode => (isConsumption ? value : '-')
  const costValue = (value: number): ReactNode =>
    isConsumption ? money(value, row.currency) : '-'

  return (
    <div className="space-y-4">
      <Section title="资金信息">
        <InfoGrid>
          <InfoItem label="请求 ID" value={dash(row.request_id)} />
          <InfoItem label="类型" value={sourceLabel(row.source)} />
          <InfoItem label="状态" value={billingStatusText(row)} />
          <InfoItem label="模型" value={dash(row.model)} />
          <InfoItem label="币种" value={dash(row.currency)} />
          <InfoItem
            label="金额"
            value={
              <span className={amountToneClass(row.amount)}>
                {formatAmount(row.amount, row.currency)}
              </span>
            }
          />
          <InfoItem label="余额变动" value={<BalanceChange row={row} />} />
          <InfoItem label="用户" value={userText(row)} />
          <InfoItem label="创建时间" value={formatDate(row.created_at)} />
        </InfoGrid>
      </Section>

      <Section title="计价明细">
        <div className="space-y-4">
          <div>
            <p className="mb-2 text-xs font-medium text-muted-foreground">单价</p>
            <InfoGrid>
              <InfoItem label="输入单价" value={numValue(row.input_price)} />
              <InfoItem label="输出单价" value={numValue(row.output_price)} />
              <InfoItem
                label="缓存命中单价"
                value={numValue(row.cached_input_price)}
              />
              <InfoItem
                label="缓存创建单价"
                value={numValue(row.cache_creation_price)}
              />
              <InfoItem label="倍率" value={numValue(row.ratio)} />
            </InfoGrid>
          </div>

          <div>
            <p className="mb-2 text-xs font-medium text-muted-foreground">成本</p>
            <InfoGrid>
              <InfoItem label="输入成本" value={costValue(row.input_cost)} />
              <InfoItem label="输出成本" value={costValue(row.output_cost)} />
              <InfoItem
                label="缓存命中成本"
                value={costValue(row.cached_input_cost)}
              />
              <InfoItem
                label="缓存创建成本"
                value={costValue(row.cache_creation_cost)}
              />
            </InfoGrid>
          </div>

          <div>
            <p className="mb-2 text-xs font-medium text-muted-foreground">Tokens</p>
            <InfoGrid>
              <InfoItem
                label="输入 tokens"
                value={numValue(row.input_tokens)}
              />
              <InfoItem
                label="输出 tokens"
                value={numValue(row.output_tokens)}
              />
              <InfoItem
                label="总计 tokens"
                value={numValue(row.total_tokens)}
              />
              <InfoItem
                label="缓存命中 tokens"
                value={numValue(row.cached_input_tokens)}
              />
              <InfoItem
                label="缓存创建 tokens"
                value={numValue(row.cache_creation_tokens)}
              />
            </InfoGrid>
          </div>
        </div>
      </Section>
    </div>
  )
}

export default function BillingView() {
  const list = useBillingStore((s) => s.list)
  const total = useBillingStore((s) => s.total)
  const page = useBillingStore((s) => s.page)
  const pageSize = useBillingStore((s) => s.pageSize)
  const loading = useBillingStore((s) => s.loading)
  const fetchList = useBillingStore((s) => s.fetchList)
  const setFilter = useBillingStore((s) => s.setFilter)
  const resetFilters = useBillingStore((s) => s.resetFilters)
  const setPage = useBillingStore((s) => s.setPage)
  const setPageSize = useBillingStore((s) => s.setPageSize)
  const isAdmin = useAuthStore((s) => s.isAdmin)

  // 筛选栏本地态（点击「查询」时才写入 store）
  const [source, setSource] = useState('all')
  const [userId, setUserId] = useState('')
  const [model, setModel] = useState('')
  const [requestId, setRequestId] = useState('')
  const [startTime, setStartTime] = useState('')
  const [endTime, setEndTime] = useState('')

  const [detailOpen, setDetailOpen] = useState(false)
  const [detail, setDetail] = useState<BillingLogItem | null>(null)

  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  const columnCount = isAdmin ? 9 : 8

  useEffect(() => {
    void fetchList().catch((e) => {
      toast.error((e as Error)?.message || '加载账单列表失败')
    })
  }, [fetchList])

  const buildFilters = (): BillingLogQuery => ({
    source: source === 'all' ? undefined : (source as BillingLogSource),
    user_id: isAdmin ? toOptionalNumber(userId) : undefined,
    model: model.trim() || undefined,
    request_id: requestId.trim() || undefined,
    start_time: toBackendTime(startTime),
    end_time: toBackendTime(endTime),
  })

  const handleSearch = async () => {
    setFilter(buildFilters())
    try {
      await fetchList()
    } catch (e) {
      toast.error((e as Error)?.message || '加载账单列表失败')
    }
  }

  const handleReset = async () => {
    setSource('all')
    setUserId('')
    setModel('')
    setRequestId('')
    setStartTime('')
    setEndTime('')
    resetFilters()
    try {
      await fetchList()
    } catch (e) {
      toast.error((e as Error)?.message || '加载账单列表失败')
    }
  }

  const handlePrev = async () => {
    if (page <= 1) return
    setPage(page - 1)
    try {
      await fetchList()
    } catch (e) {
      toast.error((e as Error)?.message || '加载账单列表失败')
    }
  }

  const handleNext = async () => {
    if (page >= totalPages) return
    setPage(page + 1)
    try {
      await fetchList()
    } catch (e) {
      toast.error((e as Error)?.message || '加载账单列表失败')
    }
  }

  const handlePageSizeChange = async (value: string) => {
    setPageSize(Number(value))
    try {
      await fetchList()
    } catch (e) {
      toast.error((e as Error)?.message || '加载账单列表失败')
    }
  }

  const openDetail = (row: BillingLogItem) => {
    setDetail(row)
    setDetailOpen(true)
  }

  return (
    <div className="mx-auto w-full max-w-7xl space-y-5">
      <PageHeader
        className="mb-5"
        title="账单日志"
        description="查询账户资金流水：消费扣费与充值记录"
      />

      {/* 筛选栏 */}
      <Card className="py-0">
        <CardContent className="px-4 py-4">
          <div className="flex flex-wrap items-end gap-3">
            <Field label="类型">
              <Select value={source} onValueChange={setSource}>
                <SelectTrigger className="w-32">
                  <SelectValue placeholder="全部类型" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部类型</SelectItem>
                  {SOURCE_OPTIONS.map((o) => (
                    <SelectItem key={o.value} value={o.value}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>

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

            <Field label="模型">
              <Input
                className="w-40"
                placeholder="模型名"
                value={model}
                onChange={(e) => setModel(e.target.value)}
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

            <Field label="开始时间">
              <Input
                className="w-52"
                type="datetime-local"
                value={startTime}
                onChange={(e) => setStartTime(e.target.value)}
              />
            </Field>

            <Field label="结束时间">
              <Input
                className="w-52"
                type="datetime-local"
                value={endTime}
                onChange={(e) => setEndTime(e.target.value)}
              />
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
      <Card className="py-0">
        <CardContent className="px-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="min-w-[150px]">时间</TableHead>
                <TableHead className="w-[80px]">类型</TableHead>
                <TableHead className="min-w-[150px]">金额</TableHead>
                <TableHead className="w-[80px]">币种</TableHead>
                <TableHead className="min-w-[200px]">余额变动</TableHead>
                <TableHead className="min-w-[140px]">模型</TableHead>
                {isAdmin && <TableHead className="min-w-[120px]">用户</TableHead>}
                <TableHead className="w-[90px]">状态</TableHead>
                <TableHead className="w-[80px]">操作</TableHead>
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
              ) : list.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={columnCount}
                    className="h-32 text-center text-muted-foreground"
                  >
                    暂无账单流水
                  </TableCell>
                </TableRow>
              ) : (
                list.map((row) => (
                  <TableRow key={`${row.source}-${row.id}`}>
                    <TableCell>{formatDate(row.created_at)}</TableCell>
                    <TableCell>{sourceLabel(row.source)}</TableCell>
                    <TableCell>
                      <span
                        className={cn(
                          'font-medium tabular-nums whitespace-nowrap',
                          amountToneClass(row.amount),
                        )}
                      >
                        {formatAmount(row.amount, row.currency)}
                      </span>
                    </TableCell>
                    <TableCell>{dash(row.currency)}</TableCell>
                    <TableCell>
                      <BalanceChange row={row} />
                    </TableCell>
                    <TableCell>{dash(row.model)}</TableCell>
                    {isAdmin && <TableCell>{userText(row)}</TableCell>}
                    <TableCell>
                      <DotBadge tone={billingStatusTone(row)}>
                        {billingStatusText(row)}
                      </DotBadge>
                    </TableCell>
                    <TableCell>
                      <Button
                        variant="link"
                        size="sm"
                        className="h-auto p-0"
                        onClick={() => openDetail(row)}
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
            <DialogTitle>账单详情</DialogTitle>
            <DialogDescription className="sr-only">
              账单流水与计价明细
            </DialogDescription>
          </DialogHeader>
          {detail ? (
            <BillingDetailContent row={detail} />
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
