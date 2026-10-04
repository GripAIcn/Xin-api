import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'
import {
  ArrowLeft,
  Check,
  CircleCheck,
  CircleX,
  Copy,
  Loader2,
  Pencil,
  Play,
  Plus,
  Trash2,
  TriangleAlert,
} from 'lucide-react'

import ChannelTestResult from '@/components/ChannelTestResult'
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
import { Alert, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
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
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { formatDate, maskKey } from '@/lib/format'
import {
  channelStatusLabel,
  channelStatusTone,
  groupStatusLabel,
  summarizeTestResult,
} from '@/lib/status'
import { useApiKeyStore } from '@/stores/apikeys'
import { useChannelStore } from '@/stores/channels'
import { useGroupStore } from '@/stores/groups'
import type { ApiKey, Channel, ChannelTestResult as ChannelTestResultData } from '@/types/api'

const channelSchema = z.object({
  name: z.string().min(1, '请输入渠道名称').min(2, '至少 2 个字符'),
  model_mapping: z.string().min(1, '请输入模型映射'),
  base_url: z.string().min(1, '请输入 Base URL'),
  api_key: z.string().min(1, '请输入 API Key'),
  weight: z.number().min(1, '请输入权重').max(100, '权重需在 1-100 之间'),
})

type ChannelFormValues = z.infer<typeof channelSchema>

const emptyChannelForm: ChannelFormValues = {
  name: '',
  model_mapping: '',
  base_url: '',
  api_key: '',
  weight: 1,
}

function errMessage(e: unknown): string {
  if (e instanceof Error) return e.message
  if (typeof e === 'string') return e
  return ''
}

function StatusBadge({ status }: { status: number }) {
  const tone = channelStatusTone(status)
  const label = channelStatusLabel(status)
  if (tone === 'success') {
    return (
      <Badge className="border-transparent bg-green-600 text-white">{label}</Badge>
    )
  }
  if (tone === 'danger') {
    return <Badge variant="destructive">{label}</Badge>
  }
  return <Badge variant="secondary">{label}</Badge>
}

export default function GroupDetailView() {
  const navigate = useNavigate()
  const { id } = useParams()
  const groupId = Number(id)

  const { groups, fetchAll: fetchGroups } = useGroupStore()
  const {
    channels,
    fetchByGroup,
    create: createChannel,
    update: updateChannel,
    remove: removeChannel,
    testChannel,
    testGroupChannels,
  } = useChannelStore()
  const {
    apiKeys,
    fetchAll: fetchApiKeys,
    create: createApiKey,
    remove: removeApiKey,
  } = useApiKeyStore()

  const group = groups.find((g) => g.id === groupId)

  const [loading, setLoading] = useState(true)

  // 测试结果
  const [testResults, setTestResults] = useState<
    Record<number, ChannelTestResultData>
  >({})
  const [testingIds, setTestingIds] = useState<Set<number>>(new Set())
  const [testingAll, setTestingAll] = useState(false)
  const [selectedResult, setSelectedResult] =
    useState<ChannelTestResultData | null>(null)

  // 渠道弹窗
  const [channelDialogOpen, setChannelDialogOpen] = useState(false)
  const [channelDialogType, setChannelDialogType] = useState<'create' | 'edit'>(
    'create',
  )
  const [editingChannelId, setEditingChannelId] = useState<number | null>(null)
  const [editStatus, setEditStatus] = useState(1)
  const [channelSubmitting, setChannelSubmitting] = useState(false)
  const [channelToDelete, setChannelToDelete] = useState<Channel | null>(null)

  // API Key
  const [apiKeySubmitting, setApiKeySubmitting] = useState(false)
  const [newKeyValue, setNewKeyValue] = useState('')
  const [newKeyDialogOpen, setNewKeyDialogOpen] = useState(false)
  const [keyCopied, setKeyCopied] = useState(false)
  const [keyToDelete, setKeyToDelete] = useState<ApiKey | null>(null)

  const form = useForm<ChannelFormValues>({
    resolver: zodResolver(channelSchema),
    defaultValues: emptyChannelForm,
  })

  useEffect(() => {
    let active = true
    const load = async () => {
      try {
        await Promise.all([
          fetchGroups(),
          fetchByGroup(groupId),
          fetchApiKeys(groupId),
        ])
      } catch (e) {
        const msg = errMessage(e)
        if (msg && msg !== 'Response data is empty') {
          toast.error('加载数据失败')
        }
      } finally {
        if (active) setLoading(false)
      }
    }
    void load()
    return () => {
      active = false
    }
  }, [groupId, fetchGroups, fetchByGroup, fetchApiKeys])

  // === 渠道 CRUD ===
  const openCreateChannel = () => {
    setChannelDialogType('create')
    setEditingChannelId(null)
    form.reset(emptyChannelForm)
    setChannelDialogOpen(true)
  }

  const openEditChannel = (channel: Channel) => {
    setChannelDialogType('edit')
    setEditingChannelId(channel.id)
    setEditStatus(channel.status)
    form.reset({
      name: channel.name,
      model_mapping: channel.model_mapping,
      base_url: channel.base_url,
      api_key: channel.api_key,
      weight: channel.weight,
    })
    setChannelDialogOpen(true)
  }

  const onSubmitChannel = async (values: ChannelFormValues) => {
    setChannelSubmitting(true)
    try {
      if (channelDialogType === 'edit' && editingChannelId != null) {
        await updateChannel(editingChannelId, {
          group_id: groupId,
          ...values,
          status: editStatus,
        })
        toast.success('渠道更新成功')
      } else {
        await createChannel({ group_id: groupId, ...values })
        toast.success('渠道创建成功')
      }
      setChannelDialogOpen(false)
    } catch (e) {
      toast.error(errMessage(e) || '操作失败')
    } finally {
      setChannelSubmitting(false)
    }
  }

  const confirmDeleteChannel = async () => {
    if (!channelToDelete) return
    const target = channelToDelete
    setChannelToDelete(null)
    try {
      await removeChannel(target.id)
      setTestResults((prev) => {
        const next = { ...prev }
        delete next[target.id]
        return next
      })
      toast.success('渠道已删除')
    } catch (e) {
      toast.error(errMessage(e) || '删除失败')
    }
  }

  // === 渠道测试 ===
  const handleTestChannel = async (channel: Channel) => {
    setTestingIds((prev) => new Set(prev).add(channel.id))
    try {
      const result = await testChannel(channel.id)
      setTestResults((prev) => ({ ...prev, [channel.id]: result }))
      toast.success('测试完成')
    } catch (e) {
      toast.error(errMessage(e) || '测试失败')
    } finally {
      setTestingIds((prev) => {
        const next = new Set(prev)
        next.delete(channel.id)
        return next
      })
    }
  }

  const handleTestAllChannels = async () => {
    setTestingAll(true)
    try {
      const results = await testGroupChannels(groupId)
      setTestResults((prev) => {
        const next = { ...prev }
        results.forEach((r) => {
          next[r.channel_id] = r
        })
        return next
      })
      toast.success(`已完成 ${results.length} 个渠道的测试`)
    } catch (e) {
      toast.error(errMessage(e) || '批量测试失败')
    } finally {
      setTestingAll(false)
    }
  }

  // === API Key ===
  const handleCreateApiKey = async () => {
    setApiKeySubmitting(true)
    try {
      const result = await createApiKey(groupId)
      setNewKeyValue(result.key)
      setKeyCopied(false)
      setNewKeyDialogOpen(true)
    } catch (e) {
      toast.error(errMessage(e) || '创建失败')
    } finally {
      setApiKeySubmitting(false)
    }
  }

  const copyKey = async () => {
    try {
      await navigator.clipboard.writeText(newKeyValue)
      setKeyCopied(true)
      setTimeout(() => setKeyCopied(false), 2000)
    } catch {
      toast.error('复制失败')
    }
  }

  const confirmDeleteKey = async () => {
    if (!keyToDelete) return
    const target = keyToDelete
    setKeyToDelete(null)
    try {
      await removeApiKey(target.key)
      toast.success('API Key 已删除')
    } catch (e) {
      toast.error(errMessage(e) || '删除失败')
    }
  }

  return (
    <div className="mx-auto w-full max-w-6xl p-6">
      {/* 头部 */}
      <div className="mb-5 flex items-center gap-4">
        <Button variant="ghost" size="sm" onClick={() => navigate('/groups')}>
          <ArrowLeft />
          返回
        </Button>
        {group ? (
          <div className="flex-1">
            <h2 className="flex items-center gap-2 text-xl font-semibold">
              {group.name}
              <Badge
                className={
                  group.status === 1
                    ? 'border-transparent bg-green-600 text-white'
                    : undefined
                }
                variant={group.status === 1 ? undefined : 'secondary'}
              >
                {groupStatusLabel(group.status)}
              </Badge>
            </h2>
            <p className="text-sm text-muted-foreground">
              项目组 ID: {group.id}
            </p>
          </div>
        ) : (
          <Skeleton className="h-10 w-[200px]" />
        )}
      </div>

      <Tabs defaultValue="channels">
        <TabsList>
          <TabsTrigger value="channels">渠道</TabsTrigger>
          <TabsTrigger value="apikeys">{`API Key (${apiKeys.length})`}</TabsTrigger>
        </TabsList>

        {/* 渠道面板 */}
        <TabsContent value="channels">
          <div className="mb-4 flex items-center justify-between">
            <p className="text-sm text-muted-foreground">
              管理此项目组下的上游渠道
            </p>
            <div className="flex gap-2">
              <Button
                variant="secondary"
                size="sm"
                disabled={channels.length === 0 || testingAll}
                onClick={handleTestAllChannels}
              >
                {testingAll ? (
                  <Loader2 className="animate-spin" />
                ) : (
                  <Play />
                )}
                测试所有渠道
              </Button>
              <Button size="sm" onClick={openCreateChannel}>
                <Plus />
                添加渠道
              </Button>
            </div>
          </div>

          <Card className="py-0">
            <CardContent className="px-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="min-w-[150px]">名称</TableHead>
                    <TableHead className="min-w-[150px]">模型映射</TableHead>
                    <TableHead className="min-w-[200px]">Base URL</TableHead>
                    <TableHead className="w-[80px]">权重</TableHead>
                    <TableHead className="w-[100px]">状态</TableHead>
                    <TableHead className="w-[150px]">测试结果</TableHead>
                    <TableHead className="w-[100px]">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {loading ? (
                    Array.from({ length: 3 }).map((_, i) => (
                      <TableRow key={i}>
                        <TableCell colSpan={7}>
                          <Skeleton className="h-6 w-full" />
                        </TableCell>
                      </TableRow>
                    ))
                  ) : channels.length === 0 ? (
                    <TableRow>
                      <TableCell
                        colSpan={7}
                        className="py-8 text-center text-muted-foreground"
                      >
                        暂无渠道
                      </TableCell>
                    </TableRow>
                  ) : (
                    channels.map((channel) => {
                      const result = testResults[channel.id]
                      const testing = testingIds.has(channel.id)
                      const summary = result
                        ? summarizeTestResult(result)
                        : null
                      return (
                        <TableRow key={channel.id}>
                          <TableCell className="font-medium">
                            {channel.name}
                          </TableCell>
                          <TableCell>{channel.model_mapping}</TableCell>
                          <TableCell
                            className="max-w-[200px] truncate"
                            title={channel.base_url}
                          >
                            {channel.base_url}
                          </TableCell>
                          <TableCell>{channel.weight}</TableCell>
                          <TableCell>
                            <StatusBadge status={channel.status} />
                          </TableCell>
                          <TableCell>
                            {summary ? (
                              <button
                                type="button"
                                className="flex cursor-pointer items-center gap-1 rounded px-2 py-1 transition-colors hover:bg-muted"
                                onClick={() =>
                                  setSelectedResult(testResults[channel.id] ?? null)
                                }
                              >
                                {summary.tone === 'success' && (
                                  <CircleCheck className="size-4 text-green-600" />
                                )}
                                {summary.tone === 'danger' && (
                                  <CircleX className="size-4 text-destructive" />
                                )}
                                {summary.tone === 'warning' && (
                                  <TriangleAlert className="size-4 text-amber-500" />
                                )}
                                <span
                                  className={
                                    summary.tone === 'success'
                                      ? 'text-green-600'
                                      : summary.tone === 'danger'
                                        ? 'text-destructive'
                                        : 'text-amber-500'
                                  }
                                >
                                  {summary.text}
                                </span>
                                {summary.isMulti && (
                                  <Badge variant="secondary">详情</Badge>
                                )}
                              </button>
                            ) : (
                              <Button
                                variant="link"
                                size="sm"
                                className="px-0"
                                disabled={channel.status !== 1 || testing}
                                onClick={() => handleTestChannel(channel)}
                              >
                                {testing ? (
                                  <Loader2 className="animate-spin" />
                                ) : (
                                  <Play />
                                )}
                                测试
                              </Button>
                            )}
                          </TableCell>
                          <TableCell>
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              onClick={() => openEditChannel(channel)}
                            >
                              <Pencil />
                            </Button>
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              className="text-destructive"
                              onClick={() => setChannelToDelete(channel)}
                            >
                              <Trash2 />
                            </Button>
                          </TableCell>
                        </TableRow>
                      )
                    })
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        {/* API Key 面板 */}
        <TabsContent value="apikeys">
          <div className="mb-4 flex items-center justify-between">
            <p className="text-sm text-muted-foreground">
              管理此项目组的 API Key
            </p>
            <Button
              size="sm"
              disabled={apiKeySubmitting}
              onClick={handleCreateApiKey}
            >
              {apiKeySubmitting ? (
                <Loader2 className="animate-spin" />
              ) : (
                <Plus />
              )}
              创建 Key
            </Button>
          </div>

          <Card className="py-0">
            <CardContent className="px-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="min-w-[300px]">API Key</TableHead>
                    <TableHead className="w-[180px]">创建时间</TableHead>
                    <TableHead className="w-[100px]">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {loading ? (
                    Array.from({ length: 3 }).map((_, i) => (
                      <TableRow key={i}>
                        <TableCell colSpan={3}>
                          <Skeleton className="h-6 w-full" />
                        </TableCell>
                      </TableRow>
                    ))
                  ) : apiKeys.length === 0 ? (
                    <TableRow>
                      <TableCell
                        colSpan={3}
                        className="py-8 text-center text-muted-foreground"
                      >
                        暂无 API Key
                      </TableCell>
                    </TableRow>
                  ) : (
                    apiKeys.map((item) => (
                      <TableRow key={item.key}>
                        <TableCell>
                          <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">
                            {maskKey(item.key)}
                          </code>
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {formatDate(item.created_at)}
                        </TableCell>
                        <TableCell>
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            className="text-destructive"
                            onClick={() => setKeyToDelete(item)}
                          >
                            <Trash2 />
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* 渠道表单弹窗 */}
      <Dialog open={channelDialogOpen} onOpenChange={setChannelDialogOpen}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>
              {channelDialogType === 'create' ? '添加渠道' : '编辑渠道'}
            </DialogTitle>
          </DialogHeader>
          <Form {...form}>
            <form
              onSubmit={form.handleSubmit(onSubmitChannel)}
              className="space-y-4"
            >
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>渠道名称</FormLabel>
                    <FormControl>
                      <Input
                        placeholder="例如：OpenAI 主通道"
                        {...field}
                      />
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
                        placeholder="例如：gpt-4o,gpt-4o-mini"
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
                        {...field}
                        onChange={(e) =>
                          field.onChange(
                            e.target.value === '' ? 0 : Number(e.target.value),
                          )
                        }
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
                  onClick={() => setChannelDialogOpen(false)}
                >
                  取消
                </Button>
                <Button type="submit" disabled={channelSubmitting}>
                  {channelSubmitting ? '保存中...' : '保存'}
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>

      {/* 新 Key 弹窗 */}
      <Dialog open={newKeyDialogOpen} onOpenChange={setNewKeyDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>API Key 已创建</DialogTitle>
          </DialogHeader>
          <Alert className="border-amber-500/40 bg-amber-500/10 text-amber-700">
            <TriangleAlert />
            <AlertTitle className="line-clamp-none text-amber-700">
              请立即复制并保存此 Key，关闭后将无法再次查看完整 Key。
            </AlertTitle>
          </Alert>
          <div className="flex items-center gap-3 rounded-lg bg-muted p-4">
            <code className="flex-1 font-mono text-xs break-all">
              {newKeyValue}
            </code>
            <Button variant="outline" size="sm" onClick={copyKey}>
              {keyCopied ? <Check /> : <Copy />}
              {keyCopied ? '已复制' : '复制'}
            </Button>
          </div>
          <DialogFooter>
            <Button onClick={() => setNewKeyDialogOpen(false)}>我已保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 测试结果详情弹窗 */}
      <Dialog
        open={!!selectedResult}
        onOpenChange={(open) => {
          if (!open) setSelectedResult(null)
        }}
      >
        <DialogContent className="sm:max-w-[700px]">
          <DialogHeader>
            <DialogTitle>渠道测试结果</DialogTitle>
          </DialogHeader>
          {selectedResult && <ChannelTestResult result={selectedResult} />}
        </DialogContent>
      </Dialog>

      {/* 删除渠道确认 */}
      <AlertDialog
        open={!!channelToDelete}
        onOpenChange={(open) => {
          if (!open) setChannelToDelete(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除</AlertDialogTitle>
            <AlertDialogDescription>
              {`确定要删除渠道「${channelToDelete?.name ?? ''}」吗？`}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              onClick={confirmDeleteChannel}
            >
              确认删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 删除 API Key 确认 */}
      <AlertDialog
        open={!!keyToDelete}
        onOpenChange={(open) => {
          if (!open) setKeyToDelete(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除 API Key</AlertDialogTitle>
            <AlertDialogDescription>
              <span className="block text-amber-600">
                删除后，使用此 Key 的应用将立即无法访问网关。
              </span>
              <span className="mt-2 flex items-center gap-2">
                API Key：
                <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">
                  {keyToDelete ? maskKey(keyToDelete.key) : ''}
                </code>
              </span>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction variant="destructive" onClick={confirmDeleteKey}>
              确认删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}