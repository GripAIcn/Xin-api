import { useEffect, useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import {
  Boxes,
  CircleCheck,
  CirclePlay,
  CircleX,
  Loader2,
  Pencil,
  Plus,
  Trash2,
  TriangleAlert,
} from 'lucide-react'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'

import ChannelModelsDialog from '@/components/ChannelModelsDialog'
import ChannelTestResultCard from '@/components/ChannelTestResult'
import { DotBadge } from '@/components/StatusBadge'
import PageHeader from '@/components/PageHeader'
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
import { Card, CardContent } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { cn } from '@/lib/utils'
import {
  channelStatusLabel,
  channelStatusTone,
  summarizeTestResult,
} from '@/lib/status'
import { useChannelStore } from '@/stores/channels'
import type { Channel, ChannelTestResult } from '@/types/api'

const channelFormSchema = z.object({
  name: z.string().min(1, '请输入渠道名称').min(2, '至少 2 个字符'),
  provider: z.string().min(1, '请输入供应商，例如 openai'),
  protocol: z
    .string()
    .min(1, '请输入协议')
    .refine((v) => v.split(',').map((s) => s.trim()).filter(Boolean).length > 0, {
      message: '至少填写一个协议',
    }),
  base_url: z.string().min(1, '请输入 Base URL'),
  api_key: z.string().min(1, '请输入 API Key'),
  weight: z.number().min(1, '请输入权重').max(100, '权重不能超过 100'),
  priority: z.number().min(0, '优先级不能小于 0'),
  tags: z.string(),
  timeout: z.number().min(1, '超时至少 1 秒'),
  max_retries: z.number().min(0, '重试次数不能小于 0'),
})

type ChannelFormValues = z.infer<typeof channelFormSchema>

// 将逗号分隔文本拆成去空数组（用于 protocol / tags）
const splitList = (value: string): string[] =>
  value.split(',').map((s) => s.trim()).filter(Boolean)

export default function ChannelsView() {
  const channels = useChannelStore((s) => s.channels)
  const fetchAllChannels = useChannelStore((s) => s.fetchAll)
  const createChannel = useChannelStore((s) => s.create)
  const updateChannel = useChannelStore((s) => s.update)
  const removeChannel = useChannelStore((s) => s.remove)
  const testChannel = useChannelStore((s) => s.testChannel)

  const [loading, setLoading] = useState(true)

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingChannel, setEditingChannel] = useState<Channel | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const [testResults, setTestResults] = useState<Map<number, ChannelTestResult>>(
    new Map(),
  )
  const [testing, setTesting] = useState<Set<number>>(new Set())
  const [testDialogOpen, setTestDialogOpen] = useState(false)
  const [selectedTestResult, setSelectedTestResult] =
    useState<ChannelTestResult | null>(null)

  const [deleteTarget, setDeleteTarget] = useState<Channel | null>(null)

  const [modelsChannel, setModelsChannel] = useState<Channel | null>(null)

  const form = useForm<ChannelFormValues>({
    resolver: zodResolver(channelFormSchema),
    defaultValues: {
      name: '',
      provider: '',
      protocol: '',
      base_url: '',
      api_key: '',
      weight: 1,
      priority: 0,
      tags: '',
      timeout: 60,
      max_retries: 0,
    },
  })

  useEffect(() => {
    let cancelled = false

    const load = async () => {
      setLoading(true)
      try {
        await fetchAllChannels()
        if (!cancelled) setTestResults(new Map())
      } catch (e) {
        toast.error(
          e instanceof Error && e.message ? e.message : '加载渠道列表失败',
        )
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    void load()
    return () => {
      cancelled = true
    }
  }, [fetchAllChannels])

  const openCreate = () => {
    setEditingChannel(null)
    form.reset({
      name: '',
      provider: '',
      protocol: '',
      base_url: '',
      api_key: '',
      weight: 1,
      priority: 0,
      tags: '',
      timeout: 60,
      max_retries: 0,
    })
    setDialogOpen(true)
  }

  const openEdit = (row: Channel) => {
    setEditingChannel(row)
    form.reset({
      name: row.name,
      provider: row.provider,
      protocol: (row.protocol ?? []).join(','),
      base_url: row.base_url,
      api_key: row.api_key,
      weight: row.weight,
      priority: row.priority,
      tags: (row.tags ?? []).join(','),
      timeout: row.timeout,
      max_retries: row.max_retries,
    })
    setDialogOpen(true)
  }

  const onSubmit = async (values: ChannelFormValues) => {
    setSubmitting(true)
    try {
      if (editingChannel) {
        await updateChannel(editingChannel.id, {
          name: values.name,
          provider: values.provider,
          protocol: splitList(values.protocol),
          base_url: values.base_url,
          api_key: values.api_key,
          weight: values.weight,
          priority: values.priority,
          tags: splitList(values.tags),
          timeout: values.timeout,
          max_retries: values.max_retries,
          status: editingChannel.status,
        })
        toast.success('渠道更新成功')
      } else {
        await createChannel({
          name: values.name,
          provider: values.provider,
          protocol: splitList(values.protocol),
          base_url: values.base_url,
          api_key: values.api_key,
          weight: values.weight,
          priority: values.priority,
          tags: splitList(values.tags),
          timeout: values.timeout,
          max_retries: values.max_retries,
        })
        toast.success('渠道创建成功')
      }
      setDialogOpen(false)
      setEditingChannel(null)
    } catch (e) {
      toast.error(e instanceof Error && e.message ? e.message : '操作失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await removeChannel(deleteTarget.id)
      toast.success('渠道已删除')
    } catch (e) {
      toast.error(e instanceof Error && e.message ? e.message : '删除失败')
    } finally {
      setDeleteTarget(null)
    }
  }

  const handleTestChannel = async (row: Channel) => {
    setTesting((prev) => new Set(prev).add(row.id))
    try {
      const result = await testChannel(row.id)
      setTestResults((prev) => {
        const next = new Map(prev)
        next.set(row.id, result)
        return next
      })
      toast.success('测试完成')
    } catch (e) {
      toast.error(e instanceof Error && e.message ? e.message : '测试失败')
    } finally {
      setTesting((prev) => {
        const next = new Set(prev)
        next.delete(row.id)
        return next
      })
    }
  }

  const openTestDetail = (result: ChannelTestResult) => {
    setSelectedTestResult(result)
    setTestDialogOpen(true)
  }

  const renderTestResult = (row: Channel) => {
    const result = testResults.get(row.id)
    if (!result) {
      const isTesting = testing.has(row.id)
      return (
        <Button
          variant="link"
          size="sm"
          className="h-auto p-0"
          disabled={row.status !== 'active' || isTesting}
          onClick={() => void handleTestChannel(row)}
        >
          {isTesting ? (
            <Loader2 className="animate-spin" />
          ) : (
            <CirclePlay />
          )}
          测试
        </Button>
      )
    }

    const summary = summarizeTestResult(result)
    const Icon =
      summary.tone === 'success'
        ? CircleCheck
        : summary.tone === 'danger'
          ? CircleX
          : TriangleAlert
    const toneClass =
      summary.tone === 'success'
        ? 'text-green-600'
        : summary.tone === 'danger'
          ? 'text-red-600'
          : 'text-amber-500'

    return (
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={() => openTestDetail(result)}
        className={cn('h-auto gap-1 px-1 py-1', toneClass)}
      >
        <Icon className="size-4" />
        <span>{summary.text}</span>
        {summary.isMulti && <Badge variant="secondary">详情</Badge>}
      </Button>
    )
  }

  const isEdit = editingChannel !== null

  return (
    <div className="w-full space-y-5">
      <PageHeader
        className="mb-5"
        title="渠道管理"
        description="管理所有上游供应商渠道"
        actions={
          <Button onClick={openCreate}>
            <Plus />
            添加渠道
          </Button>
        }
      />

      {/* 数据表格 */}
      <Card className="overflow-hidden py-0">
        <CardContent className="px-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[70px]">ID</TableHead>
                <TableHead className="min-w-[140px]">名称</TableHead>
                <TableHead className="min-w-[180px]">供应商 / 协议</TableHead>
                <TableHead className="min-w-[180px]">Base URL</TableHead>
                <TableHead className="w-[70px]">权重</TableHead>
                <TableHead className="w-[90px]">状态</TableHead>
                <TableHead className="w-[150px]">测试结果</TableHead>
                <TableHead className="w-[100px]">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {channels.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={8}
                    className="h-32 text-center text-muted-foreground"
                  >
                    {loading ? '加载中...' : '暂无渠道'}
                  </TableCell>
                </TableRow>
              ) : (
                channels.map((row) => {
                  const tone = channelStatusTone(row.status)
                  return (
                    <TableRow key={row.id}>
                      <TableCell>{row.id}</TableCell>
                      <TableCell className="font-medium">{row.name}</TableCell>
                      <TableCell>
                        <div className="flex flex-wrap items-center gap-1">
                          <Badge variant="secondary">{row.provider}</Badge>
                          {(row.protocol ?? []).map((p) => (
                            <Badge key={p} variant="outline">
                              {p}
                            </Badge>
                          ))}
                        </div>
                      </TableCell>
                      <TableCell>
                        <span
                          className="block max-w-[220px] truncate"
                          title={row.base_url}
                        >
                          {row.base_url}
                        </span>
                      </TableCell>
                      <TableCell>{row.weight}</TableCell>
                      <TableCell>
                        <DotBadge tone={tone}>
                          {channelStatusLabel(row.status)}
                        </DotBadge>
                      </TableCell>
                      <TableCell>{renderTestResult(row)}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            aria-label="模型管理"
                            onClick={() => setModelsChannel(row)}
                          >
                            <Boxes />
                          </Button>
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
        </CardContent>
      </Card>

      {/* 创建/编辑对话框 */}
      <Sheet
        open={dialogOpen}
        onOpenChange={(open) => {
          setDialogOpen(open)
          if (!open) setEditingChannel(null)
        }}
      >
        <SheetContent className="overflow-y-auto sm:max-w-xl">
          <SheetHeader>
            <SheetTitle>{isEdit ? '编辑渠道' : '添加渠道'}</SheetTitle>
            <SheetDescription className="sr-only">
              填写渠道信息
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
                    <FormLabel>渠道名称</FormLabel>
                    <FormControl>
                      <Input placeholder="例如：OpenAI 主通道" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="provider"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>供应商</FormLabel>
                    <FormControl>
                      <Input placeholder="例如：openai" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="protocol"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>协议</FormLabel>
                    <FormControl>
                      <Input
                        placeholder="多个用逗号分隔，例如：chat,responses"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="tags"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>标签</FormLabel>
                    <FormControl>
                      <Input
                        placeholder="多个用逗号分隔，例如：fast,cheap"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="base_url"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Base URL</FormLabel>
                    <FormControl>
                      <Input
                        placeholder="例如：https://api.openai.com"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="api_key"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>API Key</FormLabel>
                    <FormControl>
                      <Input
                        type="password"
                        placeholder="上游供应商 API Key"
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="weight"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>权重</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        min={1}
                        max={100}
                        name={field.name}
                        ref={field.ref}
                        value={field.value}
                        onBlur={field.onBlur}
                        onChange={(e) => field.onChange(Number(e.target.value))}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="priority"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>优先级</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        min={0}
                        name={field.name}
                        ref={field.ref}
                        value={field.value}
                        onBlur={field.onBlur}
                        onChange={(e) => field.onChange(Number(e.target.value))}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="timeout"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>超时（秒）</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        min={1}
                        name={field.name}
                        ref={field.ref}
                        value={field.value}
                        onBlur={field.onBlur}
                        onChange={(e) => field.onChange(Number(e.target.value))}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="max_retries"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>失败重试次数</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        min={0}
                        name={field.name}
                        ref={field.ref}
                        value={field.value}
                        onBlur={field.onBlur}
                        onChange={(e) => field.onChange(Number(e.target.value))}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <SheetFooter className="p-0 sm:flex-row sm:justify-end">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setDialogOpen(false)}
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
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              确定要删除渠道「{deleteTarget?.name}」吗？
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

      {/* 测试结果详情 */}
      <Dialog open={testDialogOpen} onOpenChange={setTestDialogOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>渠道测试结果</DialogTitle>
            <DialogDescription className="sr-only">
              渠道测试结果详情
            </DialogDescription>
          </DialogHeader>
          {selectedTestResult && (
            <ChannelTestResultCard result={selectedTestResult} />
          )}
        </DialogContent>
      </Dialog>

      {/* 渠道模型管理 */}
      <ChannelModelsDialog
        channelId={modelsChannel?.id ?? null}
        channelName={modelsChannel?.name}
        open={modelsChannel !== null}
        onOpenChange={(open) => {
          if (!open) setModelsChannel(null)
        }}
      />
    </div>
  )
}