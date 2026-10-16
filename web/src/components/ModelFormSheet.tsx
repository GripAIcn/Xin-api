import { useEffect, useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { Loader2, RefreshCw } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
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
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import {
  createChannelModel,
  fetchUpstreamModels,
  updateChannelModel,
} from '@/api/modules/channels'
import { useChannelStore } from '@/stores/channels'
import { usePriceStore } from '@/stores/prices'
import type { Channel, ModelChannelEntry } from '@/types/api'

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

interface ModelFormSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** 编辑态：被编辑的明细行与其所属逻辑模型名；新增态传 null */
  editing: { groupName: string; entry: ModelChannelEntry } | null
  /** 保存成功后回调（用于刷新聚合列表） */
  onSuccess: () => void
}

export default function ModelFormSheet({
  open,
  onOpenChange,
  editing,
  onSuccess,
}: ModelFormSheetProps) {
  const channels = useChannelStore((s) => s.channels)
  const prices = usePriceStore((s) => s.prices)

  const [selectedChannelIds, setSelectedChannelIds] = useState<number[]>([])
  const [channelError, setChannelError] = useState('')
  const [upstreamOptions, setUpstreamOptions] = useState<string[]>([])
  const [loadingUpstream, setLoadingUpstream] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  const form = useForm<ModelFormValues>({
    resolver: zodResolver(modelFormSchema),
    defaultValues: EMPTY_FORM,
  })

  const isEdit = editing !== null

  // 打开时重置临时状态并按编辑态填充表单初值
  useEffect(() => {
    if (!open) return
    setSelectedChannelIds([])
    setChannelError('')
    setUpstreamOptions([])
    if (editing) {
      const { entry } = editing
      form.reset({
        name: editing.groupName,
        upstream_model: entry.upstream_model,
        context_length:
          entry.context_length == null ? '' : String(entry.context_length),
        max_output_tokens:
          entry.max_output_tokens == null ? '' : String(entry.max_output_tokens),
        capabilities: (entry.capabilities ?? []).join(','),
        status: entry.status,
        model_price_id:
          entry.model_price_id == null ? '0' : String(entry.model_price_id),
      })
    } else {
      form.reset(EMPTY_FORM)
    }
  }, [open, editing, form])

  // 打开且渠道列表为空时，只读拉取一次渠道列表作为多选选项
  useEffect(() => {
    if (!open) return
    if (useChannelStore.getState().channels.length === 0) {
      void useChannelStore
        .getState()
        .fetchAll()
        .catch(() => {
          /* 静默失败：无渠道选项时仍可提交并看到错误提示 */
        })
    }
  }, [open])

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

  const toggleChannel = (id: number, checked: boolean) => {
    setChannelError('')
    setSelectedChannelIds((prev) =>
      checked ? [...prev, id] : prev.filter((item) => item !== id),
    )
  }

  const handleFetchUpstream = async () => {
    if (!editing) return
    setLoadingUpstream(true)
    try {
      const list = await fetchUpstreamModels(editing.entry.channel_id)
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

  const onClose = () => {
    setSelectedChannelIds([])
    setChannelError('')
    setUpstreamOptions([])
  }

  // 新增：对每个选中渠道依次创建，逐条收集成败
  const handleCreate = async (values: ModelFormValues) => {
    if (selectedChannelIds.length === 0) {
      setChannelError('至少选择一个渠道')
      return
    }
    const payload = buildPayload(values, false)
    const nameById = new Map<number, string>(
      channels.map((c) => [c.id, c.name] as const),
    )
    const failures: { name: string; error: string }[] = []
    let success = 0

    setSubmitting(true)
    for (const channelId of selectedChannelIds) {
      const errorMsg = (e: unknown) =>
        e instanceof Error && e.message ? e.message : '未知错误'
      try {
        await createChannelModel(channelId, payload)
        success += 1
      } catch (e) {
        failures.push({
          name: nameById.get(channelId) ?? `#${channelId}`,
          error: errorMsg(e),
        })
      }
    }
    setSubmitting(false)

    if (failures.length === 0) {
      toast.success(`已在 ${success} 个渠道新增模型`)
      onOpenChange(false)
      onSuccess()
      return
    }

    const description = failures
      .map((f) => `${f.name}：${f.error}`)
      .join('；')
    const first = failures[0]
    if (success > 0 && first) {
      toast.error(`渠道 ${first.name} 失败（${first.error}），其余已创建`, {
        description,
      })
      onSuccess()
    } else {
      toast.error('新增失败', { description })
    }
  }

  // 编辑：仅更新当前渠道下的该条记录
  const handleUpdate = async (values: ModelFormValues) => {
    if (!editing) return
    setSubmitting(true)
    try {
      await updateChannelModel(
        editing.entry.channel_id,
        editing.entry.model_id,
        buildPayload(values, true),
      )
      toast.success('模型更新成功')
      onOpenChange(false)
      onSuccess()
    } catch (e) {
      toast.error(e instanceof Error && e.message ? e.message : '保存失败')
    } finally {
      setSubmitting(false)
    }
  }

  const onSubmit = async (values: ModelFormValues) => {
    if (editing) {
      await handleUpdate(values)
    } else {
      await handleCreate(values)
    }
  }

  return (
    <Sheet
      open={open}
      onOpenChange={(value) => {
        onOpenChange(value)
        if (!value) onClose()
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
            {isEdit && editing ? (
              <div className="rounded-md border p-3 text-sm">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-medium">{editing.entry.channel_name}</span>
                  <Badge variant="secondary">
                    {editing.entry.channel_provider}
                  </Badge>
                </div>
                <div className="mt-1 text-muted-foreground">
                  逻辑模型：{editing.groupName}
                </div>
                <div className="mt-1 text-xs text-muted-foreground">
                  编辑将仅作用于该渠道
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  className="mt-2"
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
              </div>
            ) : (
              <div className="space-y-2">
                <div className="text-sm font-medium">选择渠道</div>
                {channels.length === 0 ? (
                  <div className="text-xs text-muted-foreground">暂无可用渠道</div>
                ) : (
                  <div className="max-h-48 space-y-1 overflow-y-auto rounded-md border p-2">
                    {channels.map((ch: Channel) => (
                      <label
                        key={ch.id}
                        className="flex items-center gap-2 text-sm"
                      >
                        <Checkbox
                          checked={selectedChannelIds.includes(ch.id)}
                          onCheckedChange={(checked) =>
                            toggleChannel(ch.id, checked === true)
                          }
                        />
                        <span>{ch.name}</span>
                        <Badge variant="secondary">{ch.provider}</Badge>
                      </label>
                    ))}
                  </div>
                )}
                {channelError && (
                  <p className="text-sm text-destructive">{channelError}</p>
                )}
                <p className="text-xs text-muted-foreground">
                  不同渠道的上游模型名可能不同，此处将统一使用同一上游名；如需差异，请新增后到明细行逐个修改
                </p>
              </div>
            )}

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
                      value={upstreamOptions.includes(field.value) ? field.value : ''}
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
                      <Input inputMode="numeric" placeholder="可空" {...field} />
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
                  <Select value={field.value} onValueChange={field.onChange}>
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
                onClick={() => onOpenChange(false)}
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
  )
}
