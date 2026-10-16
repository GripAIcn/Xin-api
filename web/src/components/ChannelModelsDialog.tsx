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
import type { ChannelModel } from '@/types/api'

const modelFormSchema = z.object({
  name: z.string().min(1, '请输入对外模型名'),
  upstream_model: z.string().min(1, '请输入上游模型名'),
  context_length: z.string(),
  max_output_tokens: z.string(),
  capabilities: z.string(),
  status: z.enum(['active', 'disabled']),
  input_price: z.string(),
  output_price: z.string(),
  cached_input_price: z.string(),
  cache_creation_price: z.string(),
  currency: z.string(),
})

type ModelFormValues = z.infer<typeof modelFormSchema>

const EMPTY_FORM: ModelFormValues = {
  name: '',
  upstream_model: '',
  context_length: '',
  max_output_tokens: '',
  capabilities: '',
  status: 'active',
  input_price: '',
  output_price: '',
  cached_input_price: '',
  cache_creation_price: '',
  currency: '',
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

const toOptionalString = (value: string): string | undefined => {
  const trimmed = value.trim()
  return trimmed ? trimmed : undefined
}

const priceText = (value?: number): string =>
  value == null ? '-' : String(value)

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
      input_price: row.input_price == null ? '' : String(row.input_price),
      output_price: row.output_price == null ? '' : String(row.output_price),
      cached_input_price:
        row.cached_input_price == null ? '' : String(row.cached_input_price),
      cache_creation_price:
        row.cache_creation_price == null
          ? ''
          : String(row.cache_creation_price),
      currency: row.currency ?? '',
    })
    setFormOpen(true)
  }

  const onSubmit = async (values: ModelFormValues) => {
    if (channelId == null) return
    setSubmitting(true)
    try {
      const payload = {
        name: values.name.trim(),
        upstream_model: values.upstream_model.trim(),
        context_length: toOptionalNumber(values.context_length),
        max_output_tokens: toOptionalNumber(values.max_output_tokens),
        capabilities: splitList(values.capabilities),
        status: values.status,
        input_price: toOptionalNumber(values.input_price),
        output_price: toOptionalNumber(values.output_price),
        cached_input_price: toOptionalNumber(values.cached_input_price),
        cache_creation_price: toOptionalNumber(values.cache_creation_price),
        currency: toOptionalString(values.currency),
      }
      if (editingModel) {
        await updateModel(channelId, editingModel.id, payload)
        toast.success('模型更新成功')
      } else {
        await createModel(channelId, payload)
        toast.success('模型新增成功')
      }
      setFormOpen(false)
      setEditingModel(null)
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

  const renderPrice = (row: ChannelModel) => {
    const hasAny =
      row.input_price != null ||
      row.output_price != null ||
      row.cached_input_price != null ||
      row.cache_creation_price != null
    if (!hasAny) return <span className="text-muted-foreground">-</span>
    return (
      <div className="text-xs text-muted-foreground">
        <div>
          入 {priceText(row.input_price)} / 出 {priceText(row.output_price)}
        </div>
        <div>
          缓存入 {priceText(row.cached_input_price)} / 缓存建{' '}
          {priceText(row.cache_creation_price)}
        </div>
        {row.currency && <div>币种 {row.currency}</div>}
      </div>
    )
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
                  <TableHead className="min-w-[120px]">能力</TableHead>
                  <TableHead className="min-w-[180px]">价格</TableHead>
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
                        <TableCell>{renderPrice(row)}</TableCell>
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

              <div className="space-y-3">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium">价格（可选）</span>
                  <Badge variant="secondary">待接入计费</Badge>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <FormField
                    control={form.control}
                    name="input_price"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>输入单价</FormLabel>
                        <FormControl>
                          <Input
                            inputMode="decimal"
                            placeholder="可空"
                            {...field}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="output_price"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>输出单价</FormLabel>
                        <FormControl>
                          <Input
                            inputMode="decimal"
                            placeholder="可空"
                            {...field}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="cached_input_price"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>缓存命中输入单价</FormLabel>
                        <FormControl>
                          <Input
                            inputMode="decimal"
                            placeholder="可空"
                            {...field}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="cache_creation_price"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>缓存创建单价</FormLabel>
                        <FormControl>
                          <Input
                            inputMode="decimal"
                            placeholder="可空"
                            {...field}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="currency"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>币种</FormLabel>
                        <FormControl>
                          <Input placeholder="例如：USD / CNY" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </div>
              </div>

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
