import { useEffect, useState } from 'react'
import { toast } from 'sonner'

import { DotBadge } from '@/components/StatusBadge'
import { Badge } from '@/components/ui/badge'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { fetchPriceModels } from '@/api/modules/prices'
import { channelStatusLabel, channelStatusTone } from '@/lib/status'
import type { ModelPrice, PriceBoundModelEntry } from '@/types/api'

interface PriceModelsDialogProps {
  price: ModelPrice | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export default function PriceModelsDialog({
  price,
  open,
  onOpenChange,
}: PriceModelsDialogProps) {
  const [entries, setEntries] = useState<PriceBoundModelEntry[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (!open || !price) return
    let cancelled = false
    setEntries([])

    const load = async () => {
      setLoading(true)
      try {
        const data = await fetchPriceModels(price.id)
        if (!cancelled) setEntries(data || [])
      } catch (e) {
        if (!cancelled) {
          toast.error(
            e instanceof Error && e.message ? e.message : '加载绑定模型失败',
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
  }, [open, price])

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-2xl">
        <SheetHeader>
          <SheetTitle>{price?.name ?? '绑定模型'}</SheetTitle>
          <SheetDescription>
            {loading ? '加载中...' : `共 ${entries.length} 个绑定模型`}
          </SheetDescription>
        </SheetHeader>

        <div className="min-h-0 flex-1 overflow-auto px-4">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="min-w-[140px]">对外模型名</TableHead>
                <TableHead className="min-w-[140px]">渠道</TableHead>
                <TableHead className="min-w-[140px]">上游模型</TableHead>
                <TableHead className="w-[110px]">模型状态</TableHead>
                <TableHead className="w-[100px]">渠道状态</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {entries.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={5}
                    className="h-24 text-center text-muted-foreground"
                  >
                    {loading ? '加载中...' : '暂无绑定模型'}
                  </TableCell>
                </TableRow>
              ) : (
                entries.map((entry) => {
                  const active = entry.model_status === 'active'
                  return (
                    <TableRow key={entry.binding_id}>
                      <TableCell className="font-medium">
                        {entry.model_name}
                      </TableCell>
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
                        <DotBadge tone={active ? 'success' : 'muted'}>
                          {active ? '启用' : '禁用'}
                        </DotBadge>
                      </TableCell>
                      <TableCell>
                        <DotBadge
                          tone={channelStatusTone(entry.channel_status)}
                        >
                          {channelStatusLabel(entry.channel_status)}
                        </DotBadge>
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
  )
}
