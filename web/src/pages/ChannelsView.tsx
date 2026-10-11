import { useEffect, useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import {
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
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
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
  model_mapping: z.string().min(1, '请输入模型映射'),
  base_url: z.string().min(1, '请输入 Base URL'),
  api_key: z.string().min(1, '请输入 API Key'),
  weight: z.number().min(1, '请输入权重').max(100, '权重不能超过 100'),
})

type ChannelFormValues = z.infer<typeof channelFormSchema>

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

  const form = useForm<ChannelFormValues>({
    resolver: zodResolver(channelFormSchema),
    defaultValues: {
      name: '',
      model_mapping: '',
      base_url: '',
      api_key: '',
      weight: 1,
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
      model_mapping: '',
      base_url: '',
      api_key: '',
      weight: 1,
    })
    setDialogOpen(true)
  }

  const openEdit = (row: Channel) => {
    setEditingChannel(row)
    form.reset({
      name: row.name,
      model_mapping: row.model_mapping,
      base_url: row.base_url,
      api_key: row.api_key,
      weight: row.weight,
    })
    setDialogOpen(true)
  }

  const onSubmit = async (values: ChannelFormValues) => {
    setSubmitting(true)
    try {
      if (editingChannel) {
        await updateChannel(editingChannel.id, {
          name: values.name,
          model_mapping: values.model_mapping,
          base_url: values.base_url,
          api_key: values.api_key,
          weight: values.weight,
          status: editingChannel.status,
        })
        toast.success('渠道更新成功')
      } else {
        await createChannel({
          name: values.name,
          model_mapping: values.model_mapping,
          base_url: values.base_url,
          api_key: values.api_key,
          weight: values.weight,
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
      <button
        type="button"
        onClick={() => openTestDetail(result)}
        className={cn(
          'flex items-center gap-1 rounded px-1 py-1 text-sm',
          toneClass,
        )}
      >
        <Icon className="size-4" />
        <span>{summary.text}</span>
        {summary.isMulti && <Badge variant="secondary">详情</Badge>}
      </button>
    )
  }

  const isEdit = editingChannel !== null

  return (
    <div className="mx-auto w-full max-w-6xl space-y-5">
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
      <Card className="py-0">
        <CardContent className="px-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[70px]">ID</TableHead>
                <TableHead className="min-w-[140px]">名称</TableHead>
                <TableHead className="min-w-[140px]">模型映射</TableHead>
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
                        <span
                          className="block max-w-[180px] truncate"
                          title={row.model_mapping}
                        >
                          {row.model_mapping}
                        </span>
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
      <Dialog
        open={dialogOpen}
        onOpenChange={(open) => {
          setDialogOpen(open)
          if (!open) setEditingChannel(null)
        }}
      >
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle>{isEdit ? '编辑渠道' : '添加渠道'}</DialogTitle>
            <DialogDescription className="sr-only">
              填写渠道信息
            </DialogDescription>
          </DialogHeader>
          <Form {...form}>
            <form
              onSubmit={(event) => {
                void form.handleSubmit(onSubmit)(event)
              }}
              className="space-y-4"
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
                name="model_mapping"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>模型映射</FormLabel>
                    <FormControl>
                      <Input
                        placeholder="例如：gpt-4o,deepseek-chat"
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

              <DialogFooter>
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
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>

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
    </div>
  )
}