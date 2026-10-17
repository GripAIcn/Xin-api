import { useCallback, useEffect, useState } from 'react'
import { AlertTriangle, Check, Copy, Plus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'

import { Alert, AlertTitle } from '@/components/ui/alert'
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
import PageHeader from '@/components/PageHeader'
import { DotBadge } from '@/components/StatusBadge'
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
import { Skeleton } from '@/components/ui/skeleton'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatDate } from '@/lib/format'
import { apiKeyStatusLabel, apiKeyStatusTone } from '@/lib/status'
import { useApiKeyStore } from '@/stores/apikeys'
import type { ApiKey } from '@/types/api'

export default function ApiKeysView() {
  const apiKeys = useApiKeyStore((s) => s.apiKeys)
  const fetchKeys = useApiKeyStore((s) => s.fetchAll)
  const create = useApiKeyStore((s) => s.create)
  const remove = useApiKeyStore((s) => s.remove)

  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState(false)
  const [newKeyValue, setNewKeyValue] = useState('')
  const [keyDialogOpen, setKeyDialogOpen] = useState(false)
  const [keyCopied, setKeyCopied] = useState(false)
  const [createSheetOpen, setCreateSheetOpen] = useState(false)
  const [keyName, setKeyName] = useState('未命名')
  const [deleteTarget, setDeleteTarget] = useState<ApiKey | null>(null)

  const loadKeys = useCallback(async () => {
    setLoading(true)
    try {
      await fetchKeys()
    } catch (e) {
      toast.error((e as Error)?.message || '加载 API Key 列表失败')
    } finally {
      setLoading(false)
    }
  }, [fetchKeys])

  useEffect(() => {
    void loadKeys()
  }, [loadKeys])

  // 创建凭证：先填写名称，再在弹窗中一次性展示明文 key
  const handleCreate = async () => {
    setCreating(true)
    try {
      const result = await create(keyName.trim() || '未命名')
      // 明文 key 仅在创建响应中出现一次
      setNewKeyValue(result.key)
      setCreateSheetOpen(false)
      setKeyDialogOpen(true)
    } catch (e) {
      toast.error((e as Error)?.message || '创建失败')
    } finally {
      setCreating(false)
    }
  }

  // 打开创建表单，名称默认「未命名」
  const openCreateSheet = () => {
    setKeyName('未命名')
    setCreateSheetOpen(true)
  }

  const copyKey = async () => {
    try {
      await navigator.clipboard.writeText(newKeyValue)
      setKeyCopied(true)
      setTimeout(() => setKeyCopied(false), 2000)
    } catch {
      toast.error('复制失败，请手动复制')
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await remove(deleteTarget.id)
      toast.success('API Key 已删除')
      setDeleteTarget(null)
    } catch (e) {
      toast.error((e as Error)?.message || '删除失败')
    }
  }

  return (
    <div className="w-full">
      <PageHeader
        className="mb-5"
        title="API Key"
        description="管理 API 访问密钥"
        actions={
          <Button onClick={openCreateSheet}>
            <Plus />
            创建 API Key
          </Button>
        }
      />

      <Card className="overflow-hidden py-0">
        <CardContent className="px-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="min-w-[200px]">名称</TableHead>
                <TableHead className="w-28">状态</TableHead>
                <TableHead className="w-48">创建时间</TableHead>
                <TableHead className="w-24 text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                Array.from({ length: 3 }).map((_, index) => (
                  <TableRow key={index}>
                    <TableCell colSpan={4}>
                      <Skeleton className="h-6 w-full" />
                    </TableCell>
                  </TableRow>
                ))
              ) : apiKeys.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={4} className="py-10 text-center text-muted-foreground">
                    暂无 API Key
                  </TableCell>
                </TableRow>
              ) : (
                apiKeys.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell className="font-medium">{row.name || '未命名'}</TableCell>
                    <TableCell>
                      <DotBadge tone={apiKeyStatusTone(row.status)}>
                        {apiKeyStatusLabel(row.status)}
                      </DotBadge>
                    </TableCell>
                    <TableCell>{formatDate(row.created_at)}</TableCell>
                    <TableCell className="text-right">
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        className="text-destructive hover:text-destructive"
                        onClick={() => setDeleteTarget(row)}
                      >
                        <Trash2 />
                        <span className="sr-only">删除</span>
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* 创建凭证（居中弹窗） */}
      <Dialog open={createSheetOpen} onOpenChange={setCreateSheetOpen}>
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle>创建 API Key</DialogTitle>
            <DialogDescription>
              为新的访问密钥设置名称以便识别，留空则使用「未命名」。
            </DialogDescription>
          </DialogHeader>
          <form
            className="space-y-4"
            onSubmit={(event) => {
              event.preventDefault()
              void handleCreate()
            }}
          >
            <div className="space-y-2">
              <Label htmlFor="api-key-name">名称</Label>
              <Input
                id="api-key-name"
                value={keyName}
                maxLength={100}
                placeholder="未命名"
                onChange={(event) => setKeyName(event.target.value)}
              />
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setCreateSheetOpen(false)}
              >
                取消
              </Button>
              <Button type="submit" disabled={creating}>
                {creating ? '创建中...' : '创建'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={keyDialogOpen} onOpenChange={setKeyDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>API Key 已创建</DialogTitle>
          </DialogHeader>
          <Alert>
            <AlertTriangle className="size-4" />
            <AlertTitle>请立即复制并保存此 Key，关闭后将无法再次查看完整 Key。</AlertTitle>
          </Alert>
          <div className="flex items-center gap-3 rounded-lg bg-muted p-4">
            <code className="flex-1 break-all font-mono text-[13px]">{newKeyValue}</code>
            <Button variant="outline" size="sm" onClick={() => void copyKey()}>
              {keyCopied ? <Check /> : <Copy />}
              {keyCopied ? '已复制' : '复制'}
            </Button>
          </div>
          <DialogFooter>
            <Button onClick={() => setKeyDialogOpen(false)}>我已保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除 API Key</AlertDialogTitle>
            <AlertDialogDescription>
              删除后，使用此 Key 的应用将立即无法访问网关。
            </AlertDialogDescription>
          </AlertDialogHeader>
          {deleteTarget && (
            <div className="flex items-center gap-2 rounded-lg border-l-[3px] border-primary bg-muted px-4 py-3">
              <span className="shrink-0 text-sm text-muted-foreground">API Key：</span>
              <span className="break-all font-medium text-foreground/80">
                {deleteTarget.name || `#${deleteTarget.id}`}
              </span>
            </div>
          )}
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
    </div>
  )
}
