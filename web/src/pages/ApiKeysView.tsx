import { useCallback, useEffect, useState } from 'react'
import { AlertTriangle, Check, Copy, Plus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'

import { Alert, AlertTitle } from '@/components/ui/alert'
import PageHeader from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatDate, maskKey } from '@/lib/format'
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

  const handleCreate = async () => {
    setCreating(true)
    try {
      const result = await create()
      setNewKeyValue(result.key)
      setKeyDialogOpen(true)
    } catch (e) {
      toast.error((e as Error)?.message || '创建失败')
    } finally {
      setCreating(false)
    }
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
      await remove(deleteTarget.key)
      toast.success('API Key 已删除')
      setDeleteTarget(null)
    } catch (e) {
      toast.error((e as Error)?.message || '删除失败')
    }
  }

  return (
    <div className="mx-auto w-full max-w-6xl">
      <PageHeader
        className="mb-5"
        title="API Key"
        description="管理 API 访问密钥"
        actions={
          <Button onClick={handleCreate} disabled={creating}>
            <Plus />
            {creating ? '创建中...' : '创建 API Key'}
          </Button>
        }
      />

      <Card>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>API Key</TableHead>
                <TableHead className="w-48">创建时间</TableHead>
                <TableHead className="w-24 text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                Array.from({ length: 3 }).map((_, index) => (
                  <TableRow key={index}>
                    <TableCell colSpan={3}>
                      <Skeleton className="h-6 w-full" />
                    </TableCell>
                  </TableRow>
                ))
              ) : apiKeys.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={3} className="py-10 text-center text-muted-foreground">
                    暂无 API Key
                  </TableCell>
                </TableRow>
              ) : (
                apiKeys.map((row) => (
                  <TableRow key={row.key}>
                    <TableCell>
                      <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-[13px]">
                        {maskKey(row.key)}
                      </code>
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

      <Dialog
        open={deleteTarget !== null}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null)
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除 API Key</DialogTitle>
          </DialogHeader>
          <Alert>
            <AlertTriangle className="size-4" />
            <AlertTitle>删除后，使用此 Key 的应用将立即无法访问网关。</AlertTitle>
          </Alert>
          {deleteTarget && (
            <div className="flex items-center gap-2 rounded-lg border-l-[3px] border-primary bg-muted px-4 py-3">
              <span className="shrink-0 text-sm text-muted-foreground">API Key：</span>
              <code className="break-all font-mono text-[13px] text-foreground/80">
                {maskKey(deleteTarget.key)}
              </code>
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>
              取消
            </Button>
            <Button variant="destructive" onClick={() => void handleDelete()}>
              确认删除
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}