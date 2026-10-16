import { useEffect, useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { Loader2, Pencil, Plus, RefreshCw, Trash2 } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'

import { DotBadge } from '@/components/StatusBadge'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useChannelStore } from '@/stores/channels'
import { usePriceStore } from '@/stores/prices'
import type { ChannelModel } from '@/types/api'

const modelFormSchema = z.object({
  name: z.string().min(1, '请输入对外模型名'),
  upstream_model: z.string().min(1, '请输入上游模型名'),
  context_length: z.string(),
  max_output_tokens: z.string(),
  capabilities: z.string(),
  status: z.enum(['active', 'disabled']),
  // 价格方案 id 字符串（'0' 表示不绑定）
  model_price_id: z.string(),
})

type ModelFormValues = z.infer<typeof modelFormSchema>

const EMPTY_FORM: ModelFormValues = {
  name: '',
  upstream_model: '',
  context_length: '',
  max_output_tokens: '',
  capabilities: '',
  status: 'active',
  model_price_id: '0',
}

// 将逗号分隔文本拆成去空数组（用于 capabilities）
const splitList = (value: string): string[] =>
  value.split(',').map((s) => s.trim()).filter(Boolean)

// 空串转为 undefined，否则转数字（非法值同样置空）
const toOptionalNumber = (value: string): number | undefined => {
  const trimmed = value.trim()
  if (!trimmed) return undefined
  const num = Number(trimmed)
  return Number.isFinite(num) ? num : undefined
}

// 表单值 → 渠道模型请求体（新增/编辑共用）
// 价格方案语义：新增态「不绑定」不提交该字段；编辑态始终提交（0=解绑 / >0=绑定）
const buildPayload = (values: ModelFormValues, isEdit: boolean) => {
  const priceId = toOptionalNumber(values.model_price_id)
  const payload: {
    name: string
    upstream_model: string
    context_length?: number
    max_output_tokens?: number
    capabilities: string[]
    status: 'active' | 'disabled'
    model_price_id?: number
  } = {
    name: values.name.trim(),
    upstream_model: values.upstream_model.trim(),
    context_length: toOptionalNumber(values.context_length),
    max_output_tokens: toOptionalNumber(values.max_output_tokens),
    capabilities: splitList(values.capabilities),
    status: values.status,
  }
  if (isEdit) {
    payload.model_price_id = priceId ?? 0
  } else if (priceId != null && priceId > 0) {
    payload.model_price_id = priceId
  }
  return payload
}

interface ChannelModelsDialogProps {
  channelId: number | null
  channelName?: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

export default function ChannelModelsDialog({
  channelId,
  channelName,
  open,
  onOpenChange,
}: ChannelModelsDialogProps) {
  const models = useChannelStore((s) => s.models)
  const fetchModels = useChannelStore((s) => s.fetchModels)
  const createModel = useChannelStore((s) => s.createModel)
  const updateModel = useChannelStore((s) => s.updateModel)
  const removeModel = useChannelStore((s) => s.removeModel)
  const updateModelStatus = useChannelStore((s) => s.updateModelStatus)
  const fetchUpstreamModels = useChannelStore((s) => s.fetchUpstreamModels)
  const prices = usePriceStore((s) => s.prices)

  const [loading, setLoading] = useState(false)
  const [statusPending, setStatusPending] = useState<Set<number>>(new Set())

  const [upstreamOptions, setUpstreamOptions] = useState<string[]>([])
  const [loadingUpstream, setLoadingUpstream] = useState(false)

  const [formOpen, setFormOpen] = useState(false)
  const [editingModel, setEditingModel] = useState<ChannelModel | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<ChannelModel | null>(null)

  const form = useForm<ModelFormValues>({
    resolver: zodResolver(modelFormSchema),
    defaultValues: EMPTY_FORM,
  })

  useEffect(() => {
    if (!open || channelId == null) return
    let cancelled = false
    setFormOpen(false)
    setEditingModel(null)
    setUpstreamOptions([])

    const load = async () => {
      setLoading(true)
      try {
        await fetchModels(channelId)
      } catch (e) {
        toast.error(
          e instanceof Error && e.message ? e.message : '加载模型列表失败',
        )
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    void load()
    return () => {
      cancelled = true
    }
  }, [open, channelId, fetchModels])

  // 打开且价格方案列表为空时，只读拉取一次作为下拉选项
  useEffect(() => {
    if (!open) return
    if (usePriceStore.getState().prices.length === 0) {
      void usePriceStore
        .getState()
        .fetchAll()
        .catch(() => {
          /* 静默失败：无方案选项时仍可提交（视为不绑定） */
        })
    }
  }, [open])

  const openCreate = () => {
    setEditingModel(null)
    form.reset(EMPTY_FORM)
    setFormOpen(true)
  }

  const openEdit = (row: ChannelModel) => {
    setEditingModel(row)
    form.reset({
      name: row.name,
      upstream_model: row.upstream_model,
      context_length:
        row.context_length == null ? '' : String(row.context_length),
      max_output_tokens:
        row.max_output_tokens == null ? '' : String(row.max_output_tokens),
      capabilities: (row.capabilities ?? []).join(','),
      status: row.status,
      model_price_id:
        row.model_price_id == null ? '0' : String(row.model_price_id),
    })
    setFormOpen(true)
  }

  const onSubmit = async (values: ModelFormValues) => {
    if (channelId == null) return
    setSubmitting(true)
    try {
      if (editingModel) {
        await updateModel(channelId, editingModel.id, buildPayload(values, true))
        toast.success('模型更新成功')
      } else {
        await createModel(channelId, buildPayload(values, false))
        toast.success('模型新增成功')
      }
      setFormOpen(false)
      setEditingModel(null)
      // 重新拉取列表，刷新「价格方案」列（绑定信息由列表接口带出）
      await fetchModels(channelId)
    } catch (e) {
      toast.error(e instanceof Error && e.message ? e.message : '保存失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async () => {
    if (channelId == null || !deleteTarget) return
    try {
      await removeModel(channelId, deleteTarget.id)
      toast.success('模型已删除')
    } catch (e) {
      toast.error(e instanceof Error && e.message ? e.message : '删除失败')
    } finally {
      setDeleteTarget(null)
    }
  }

  const handleToggleStatus = async (row: ChannelModel, checked: boolean) => {
    if (channelId == null) return
    const next = checked ? 'active' : 'disabled'
    setStatusPending((prev) => new Set(prev).add(row.id))
    try {
      await updateModelStatus(channelId, row.id, next)
      toast.success(next === 'active' ? '模型已启用' : '模型已禁用')
    } catch (e) {
      toast.error(e instanceof Error && e.message ? e.message : '状态更新失败')
    } finally {
      setStatusPending((prev) => {
        const set = new Set(prev)
        set.delete(row.id)
        return set
      })
    }
  }

  const handleFetchUpstream = async () => {
    if (channelId == null) return
    setLoadingUpstream(true)
    try {
      const list = await fetchUpstreamModels(channelId)
      setUpstreamOptions(list ?? [])
      if (!list || list.length === 0) {
        toast.info('未获取到上游模型')
      } else {
        toast.success(`已获取 ${list.length} 个上游模型`)
      }
    } catch (e) {
      toast.error(e instanceof Error && e.message ? e.message : '获取上游模型失败')
    } finally {
      setLoadingUpstream(false)
    }
  }

  const isEdit = editingModel !== null

  return (
    <>
      <Sheet open={open} onOpenChange={onOpenChange}>
        <SheetContent className="sm:max-w-2xl">
          <SheetHeader>
            <SheetTitle>模型管理</SheetTitle>
            <SheetDescription>
              {channelName ? `${channelName} 的可用模型` : '管理渠道下的模型'}
            </SheetDescription>
          </SheetHeader>

          <div className="flex flex-wrap items-center justify-between gap-2 px-4">
            <Button
              variant="outline"
              size="sm"
              disabled={loadingUpstream}
              onClick={() => void handleFetchUpstream()}
            >
              {loadingUpstream ? (
                <Loader2 className="animate-spin" />
              ) : (
                <RefreshCw />
              )}
              从上游获取模型
            </Button>
            <Button size="sm" onClick={openCreate}>
              <Plus />
              新增模型
            </Button>
          </div>

          {upstreamOptions.length > 0 && (
            <div className="mx-4 rounded-md border p-2 text-xs">
              <div className="mb-1 text-muted-foreground">
                已获取 {upstreamOptions.length} 个上游模型，可在新增/编辑时选择
              </div>
              <div className="flex flex-wrap gap-1">
                {upstreamOptions.map((m) => (
                  <Badge key={m} variant="outline">
                    {m}
                  </Badge>
                ))}
              </div>
            </div>
          )}

          <div className="min-h-0 flex-1 overflow-auto px-4">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="min-w-[120px]">对外模型名</TableHead>
                  <TableHead className="min-w-[120px]">上游模型名</TableHead>
                  <TableHead className="w-[110px]">状态</TableHead>
                  <TableHead className="min-w-[120px]">价格方案</TableHead>
                  <TableHead className="min-w-[120px]">能力</TableHead>
                  <TableHead className="w-[90px]">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {models.length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={6}
                      className="h-24 text-center text-muted-foreground"
                    >
                      {loading ? '加载中...' : '暂无模型'}
                    </TableCell>
                  </TableRow>
                ) : (
                  models.map((row) => {
                    const pending = statusPending.has(row.id)
                    const active = row.status === 'active'
                    return (
                      <TableRow key={row.id}>
                        <TableCell className="font-medium">{row.name}</TableCell>
                        <TableCell>
                          <span className="text-sm text-muted-foreground">
                            {row.upstream_model}
                          </span>
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            <Switch
                              size="sm"
                              checked={active}
                              disabled={pending}
                              onCheckedChange={(checked) =>
                                void handleToggleStatus(row, checked)
                              }
                            />
                            <DotBadge tone={active ? 'success' : 'muted'}>
                              {active ? '启用' : '禁用'}
                            </DotBadge>
                          </div>
                        </TableCell>
                        <TableCell>
                          {row.price_name ? (
                            <span className="text-sm">{row.price_name}</span>
                          ) : (
                            <span className="text-muted-foreground">-</span>
                          )}
                        </TableCell>
                        <TableCell>
                          {(row.capabilities ?? []).length === 0 ? (
                            <span className="text-muted-foreground">-</span>
                          ) : (
                            <div className="flex flex-wrap gap-1">
                              {row.capabilities.map((cap) => (
                                <Badge key={cap} variant="secondary">
                                  {cap}
                                </Badge>
                              ))}
                            </div>
                          )}
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-1">
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              aria-label="编辑"
                              onClick={() => openEdit(row)}
                            >
                              <Pencil />
                            </Button>
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              aria-label="删除"
                              className="text-destructive hover:text-destructive"
                              onClick={() => setDeleteTarget(row)}
                            >
                              <Trash2 />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    )
                  })
                )}
              </TableBody>
            </Table>
          </div>
        </SheetContent>
      </Sheet>

      {/* 新增/编辑模型 */}
      <Sheet
        open={formOpen}
        onOpenChange={(value) => {
          setFormOpen(value)
          if (!value) setEditingModel(null)
        }}
      >
        <SheetContent className="overflow-y-auto sm:max-w-xl">
          <SheetHeader>
            <SheetTitle>{isEdit ? '编辑模型' : '新增模型'}</SheetTitle>
            <SheetDescription className="sr-only">
              填写模型信息
            </SheetDescription>
          </SheetHeader>
          <Form {...form}>
            <form
              onSubmit={(event) => {
                void form.handleSubmit(onSubmit)(event)
              }}
              className="space-y-4 px-4 pb-4"
            >
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>对外模型名</FormLabel>
                    <FormControl>
                      <Input placeholder="例如：gpt-4o" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="upstream_model"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>上游模型名</FormLabel>
                    <FormControl>
                      <Input placeholder="例如：gpt-4o-2024-08-06" {...field} />
                    </FormControl>
                    {upstreamOptions.length > 0 && (
                      <Select
                        value={
                          upstreamOptions.includes(field.value)
                            ? field.value
                            : ''
                        }
                        onValueChange={field.onChange}
                      >
                        <SelectTrigger className="w-full">
                          <SelectValue placeholder="从上游列表选择" />
                        </SelectTrigger>
                        <SelectContent>
                          {upstreamOptions.map((m) => (
                            <SelectItem key={m} value={m}>
                              {m}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    )}
                    <FormMessage />
                  </FormItem>
                )}
              />

              <div className="grid grid-cols-2 gap-4">
                <FormField
                  control={form.control}
                  name="context_length"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>上下文长度</FormLabel>
                      <FormControl>
                        <Input
                          inputMode="numeric"
                          placeholder="tokens，可空"
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="max_output_tokens"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>最大输出 tokens</FormLabel>
                      <FormControl>
                        <Input
                          inputMode="numeric"
                          placeholder="可空"
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>

              <FormField
                control={form.control}
                name="capabilities"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>能力标签</FormLabel>
                    <FormControl>
                      <Input
                        placeholder="多个用逗号分隔，例如：vision,tools,reasoning"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="status"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>状态</FormLabel>
                    <Select
                      value={field.value}
                      onValueChange={field.onChange}
                    >
                      <FormControl>
                        <SelectTrigger className="w-full">
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="active">启用</SelectItem>
                        <SelectItem value="disabled">禁用</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="model_price_id"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>价格方案</FormLabel>
                    <Select value={field.value} onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger className="w-full">
                          <SelectValue placeholder="不绑定" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="0">不绑定</SelectItem>
                        {prices.map((p) => (
                          <SelectItem key={p.id} value={String(p.id)}>
                            {p.name}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    {prices.length === 0 && (
                      <p className="text-xs text-muted-foreground">
                        暂无可用价格方案，可先到「定价管理」创建
                      </p>
                    )}
                    <FormMessage />
                  </FormItem>
                )}
              />

              <SheetFooter className="p-0 sm:flex-row sm:justify-end">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setFormOpen(false)}
                >
                  取消
                </Button>
                <Button type="submit" disabled={submitting}>
                  {submitting ? '保存中...' : '保存'}
                </Button>
              </SheetFooter>
            </form>
          </Form>
        </SheetContent>
      </Sheet>

      {/* 删除确认 */}
      <AlertDialog
        open={deleteTarget !== null}
        onOpenChange={(value) => {
          if (!value) setDeleteTarget(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              确定要删除模型「{deleteTarget?.name}」吗？
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={() => void handleDelete()}
            >
              确认删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
