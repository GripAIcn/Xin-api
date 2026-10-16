import { useState } from 'react'
import { Pencil, Trash2 } from 'lucide-react'
import { toast } from 'sonner'

import ModelFormSheet from '@/components/ModelFormSheet'
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
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  deleteChannelModel,
  updateChannelModelStatus,
} from '@/api/modules/channels'
import { channelStatusLabel, channelStatusTone } from '@/lib/status'
import type { ModelChannelEntry, ModelGroup } from '@/types/api'

const priceText = (value?: number): string =>
  value == null ? '-' : String(value)

interface ModelChannelsDialogProps {
  group: ModelGroup | null
  open: boolean
  onOpenChange: (open: boolean) => void
  /** 明细发生变化后回调（用于刷新聚合列表） */
  onChanged: () => void
}

export default function ModelChannelsDialog({
  group,
  open,
  onOpenChange,
  onChanged,
}: ModelChannelsDialogProps) {
  const [statusPending, setStatusPending] = useState<Set<number>>(new Set())
  const [deleteTarget, setDeleteTarget] = useState<ModelChannelEntry | null>(null)
  const [editTarget, setEditTarget] = useState<ModelChannelEntry | null>(null)

  const entries = group?.channels ?? []

  const handleToggleStatus = async (
    entry: ModelChannelEntry,
    checked: boolean,
  ) => {
    const next = checked ? 'active' : 'disabled'
    setStatusPending((prev) => new Set(prev).add(entry.model_id))
    try {
      await updateChannelModelStatus(entry.channel_id, entry.model_id, {
        status: next,
      })
      toast.success(next === 'active' ? '模型已启用' : '模型已禁用')
      onChanged()
    } catch (e) {
      toast.error(e instanceof Error && e.message ? e.message : '状态更新失败')
    } finally {
      setStatusPending((prev) => {
        const set = new Set(prev)
        set.delete(entry.model_id)
        return set
      })
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteChannelModel(deleteTarget.channel_id, deleteTarget.model_id)
      toast.success('模型已删除')
      onChanged()
    } catch (e) {
      toast.error(e instanceof Error && e.message ? e.message : '删除失败')
    } finally {
      setDeleteTarget(null)
    }
  }

  const renderPrice = (entry: ModelChannelEntry) => {
    const hasAny =
      entry.input_price != null ||
      entry.output_price != null ||
      entry.cached_input_price != null ||
      entry.cache_creation_price != null
    if (!hasAny) return <span className="text-muted-foreground">-</span>
    return (
      <div className="text-xs text-muted-foreground">
        <div>
          入 {priceText(entry.input_price)} / 出 {priceText(entry.output_price)}
        </div>
        <div>
          缓存入 {priceText(entry.cached_input_price)} / 缓存建{' '}
          {priceText(entry.cache_creation_price)}
        </div>
        {entry.currency && <div>币种 {entry.currency}</div>}
      </div>
    )
  }

  return (
    <>
      <Sheet open={open} onOpenChange={onOpenChange}>
        <SheetContent className="sm:max-w-2xl">
          <SheetHeader>
            <SheetTitle>{group?.name ?? '模型详情'}</SheetTitle>
            <SheetDescription>
              {group ? `${group.channel_count} 个渠道` : ''}
            </SheetDescription>
          </SheetHeader>

          <div className="min-h-0 flex-1 overflow-auto px-4">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="min-w-[140px]">渠道</TableHead>
                  <TableHead className="min-w-[120px]">上游模型名</TableHead>
                  <TableHead className="w-[150px]">模型状态</TableHead>
                  <TableHead className="w-[90px]">渠道状态</TableHead>
                  <TableHead className="w-[90px]">上下文</TableHead>
                  <TableHead className="min-w-[120px]">能力</TableHead>
                  <TableHead className="min-w-[180px]">价格</TableHead>
                  <TableHead className="w-[90px]">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {entries.length === 0 ? (
                  <TableRow>
                    <TableCell
                      colSpan={8}
                      className="h-24 text-center text-muted-foreground"
                    >
                      暂无渠道
                    </TableCell>
                  </TableRow>
                ) : (
                  entries.map((entry) => {
                    const pending = statusPending.has(entry.model_id)
                    const active = entry.status === 'active'
                    return (
                      <TableRow key={entry.model_id}>
                        <TableCell>
                          <div className="flex flex-wrap items-center gap-2">
                            <span className="font-medium">
                              {entry.channel_name}
                            </span>
                            <Badge variant="secondary">
                              {entry.channel_provider}
                            </Badge>
                          </div>
                        </TableCell>
                        <TableCell>
                          <span className="text-sm text-muted-foreground">
                            {entry.upstream_model}
                          </span>
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            <Switch
                              size="sm"
                              checked={active}
                              disabled={pending}
                              onCheckedChange={(checked) =>
                                void handleToggleStatus(entry, checked)
                              }
                            />
                            <DotBadge tone={active ? 'success' : 'muted'}>
                              {active ? '启用' : '禁用'}
                            </DotBadge>
                          </div>
                        </TableCell>
                        <TableCell>
                          <DotBadge tone={channelStatusTone(entry.channel_status)}>
                            {channelStatusLabel(entry.channel_status)}
                          </DotBadge>
                        </TableCell>
                        <TableCell>
                          {entry.context_length == null ? (
                            <span className="text-muted-foreground">-</span>
                          ) : (
                            <span>{entry.context_length}</span>
                          )}
                        </TableCell>
                        <TableCell>
                          {(entry.capabilities ?? []).length === 0 ? (
                            <span className="text-muted-foreground">-</span>
                          ) : (
                            <div className="flex flex-wrap gap-1">
                              {entry.capabilities.map((cap) => (
                                <Badge key={cap} variant="secondary">
                                  {cap}
                                </Badge>
                              ))}
                            </div>
                          )}
                        </TableCell>
                        <TableCell>{renderPrice(entry)}</TableCell>
                        <TableCell>
                          <div className="flex items-center gap-1">
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              aria-label="编辑"
                              onClick={() => setEditTarget(entry)}
                            >
                              <Pencil />
                            </Button>
                            <Button
                              variant="ghost"
                              size="icon-sm"
                              aria-label="删除"
                              className="text-destructive hover:text-destructive"
                              onClick={() => setDeleteTarget(entry)}
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

      {/* 编辑明细行 */}
      <ModelFormSheet
        open={editTarget !== null}
        onOpenChange={(value) => {
          if (!value) setEditTarget(null)
        }}
        editing={
          editTarget && group
            ? { groupName: group.name, entry: editTarget }
            : null
        }
        onSuccess={() => {
          onChanged()
          setEditTarget(null)
        }}
      />

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
              确定要删除模型「{group?.name}」在该渠道下的记录吗？
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
