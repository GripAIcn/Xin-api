import { useEffect, useState } from 'react'
import { Boxes, Pencil, Plus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'

import PriceFormSheet from '@/components/PriceFormSheet'
import PriceModelsDialog from '@/components/PriceModelsDialog'
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
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { deletePrice, updatePriceStatus } from '@/api/modules/prices'
import { usePriceStore } from '@/stores/prices'
import type { ModelPriceWithCount } from '@/types/api'

export default function PricingView() {
  const prices = usePriceStore((s) => s.prices)
  const fetchAll = usePriceStore((s) => s.fetchAll)
  const setPrices = usePriceStore((s) => s.setPrices)

  const [loading, setLoading] = useState(true)
  const [statusPending, setStatusPending] = useState<Set<number>>(new Set())

  const [formOpen, setFormOpen] = useState(false)
  const [editingPrice, setEditingPrice] = useState<ModelPriceWithCount | null>(
    null,
  )
  const [deleteTarget, setDeleteTarget] = useState<ModelPriceWithCount | null>(
    null,
  )
  const [viewPrice, setViewPrice] = useState<ModelPriceWithCount | null>(null)

  useEffect(() => {
    let cancelled = false

    const load = async () => {
      setLoading(true)
      try {
        await fetchAll()
      } catch (e) {
        if (!cancelled) {
          toast.error(
            e instanceof Error && e.message ? e.message : '加载价格方案失败',
          )
        }
      } finally {
        if (!cancelled) setLoading(false)
      }
    }

    void load()
    return () => {
      cancelled = true
    }
  }, [fetchAll])

  const refresh = async () => {
    try {
      await fetchAll()
    } catch (e) {
      toast.error(
        e instanceof Error && e.message ? e.message : '加载价格方案失败',
      )
    }
  }

  const openCreate = () => {
    setEditingPrice(null)
    setFormOpen(true)
  }

  const openEdit = (row: ModelPriceWithCount) => {
    setEditingPrice(row)
    setFormOpen(true)
  }

  const handleToggleStatus = async (
    row: ModelPriceWithCount,
    checked: boolean,
  ) => {
    const next = checked ? 'active' : 'disabled'
    setStatusPending((prev) => new Set(prev).add(row.id))
    try {
      await updatePriceStatus(row.id, { status: next })
      setPrices(
        usePriceStore
          .getState()
          .prices.map((p) => (p.id === row.id ? { ...p, status: next } : p)),
      )
      toast.success(next === 'active' ? '方案已启用' : '方案已禁用')
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

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await deletePrice(deleteTarget.id)
      toast.success('价格方案已删除')
      await refresh()
    } catch (e) {
      toast.error(e instanceof Error && e.message ? e.message : '删除失败')
    } finally {
      setDeleteTarget(null)
    }
  }

  return (
    <div className="mx-auto w-full max-w-6xl space-y-5">
      <PageHeader
        className="mb-5"
        title="定价管理"
        description="管理可复用的模型价格方案，并在渠道模型中绑定"
        actions={
          <Button onClick={openCreate}>
            <Plus />
            新增方案
          </Button>
        }
      />

      {/* 数据表格 */}
      <Card className="py-0">
        <CardContent className="px-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="min-w-[140px]">方案名</TableHead>
                <TableHead className="w-[110px]">输入单价</TableHead>
                <TableHead className="w-[110px]">输出单价</TableHead>
                <TableHead className="w-[130px]">缓存命中单价</TableHead>
                <TableHead className="w-[130px]">缓存创建单价</TableHead>
                <TableHead className="w-[80px]">币种</TableHead>
                <TableHead className="w-[120px]">绑定模型数</TableHead>
                <TableHead className="w-[130px]">状态</TableHead>
                <TableHead className="w-[120px]">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {prices.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={9}
                    className="h-32 text-center text-muted-foreground"
                  >
                    {loading ? '加载中...' : '暂无价格方案'}
                  </TableCell>
                </TableRow>
              ) : (
                prices.map((row) => {
                  const pending = statusPending.has(row.id)
                  const active = row.status === 'active'
                  return (
                    <TableRow key={row.id}>
                      <TableCell className="font-medium">{row.name}</TableCell>
                      <TableCell>{row.input_price}</TableCell>
                      <TableCell>{row.output_price}</TableCell>
                      <TableCell>{row.cached_input_price}</TableCell>
                      <TableCell>{row.cache_creation_price}</TableCell>
                      <TableCell>{row.currency}</TableCell>
                      <TableCell>
                        <Badge variant="secondary">
                          {row.bound_count} 个模型
                        </Badge>
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
                            aria-label="查看绑定模型"
                            onClick={() => setViewPrice(row)}
                          >
                            <Boxes />
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

      {/* 新增/编辑价格方案 */}
      <PriceFormSheet
        open={formOpen}
        onOpenChange={(value) => {
          setFormOpen(value)
          if (!value) setEditingPrice(null)
        }}
        editing={editingPrice}
        onSuccess={() => void refresh()}
      />

      {/* 查看绑定模型 */}
      <PriceModelsDialog
        price={viewPrice}
        open={viewPrice !== null}
        onOpenChange={(value) => {
          if (!value) setViewPrice(null)
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
              确定要删除价格方案「{deleteTarget?.name}」吗？该方案下的绑定关系将一并解除。
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
    </div>
  )
}
