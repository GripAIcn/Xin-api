import { useEffect, useState } from 'react'
import { Boxes, Plus } from 'lucide-react'
import { toast } from 'sonner'

import ModelChannelsDialog from '@/components/ModelChannelsDialog'
import ModelFormSheet from '@/components/ModelFormSheet'
import { DotBadge } from '@/components/StatusBadge'
import PageHeader from '@/components/PageHeader'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { summarizeGroup } from '@/lib/models'
import { useChannelStore } from '@/stores/channels'
import { useModelStore } from '@/stores/models'

export default function ModelsView() {
  const groups = useModelStore((s) => s.groups)
  const fetchAllModels = useModelStore((s) => s.fetchAll)

  const [loading, setLoading] = useState(true)

  // 仅记录选中的逻辑模型名，明细渲染时从最新的 groups 中查找，保证明细始终为最新
  const [selectedGroupName, setSelectedGroupName] = useState<string | null>(null)
  const [createOpen, setCreateOpen] = useState(false)

  const selectedGroup = selectedGroupName
    ? (groups.find((g) => g.name === selectedGroupName) ?? null)
    : null

  useEffect(() => {
    let cancelled = false

    const load = async () => {
      setLoading(true)
      try {
        await Promise.all([
          fetchAllModels(),
          // 渠道列表仅供新增表单的多选渠道选项，失败单独容错、不阻塞模型列表渲染
          useChannelStore
            .getState()
            .fetchAll()
            .catch(() => {
              if (!cancelled) toast.error('加载渠道列表失败')
            }),
        ])
      } catch (e) {
        if (!cancelled) {
          toast.error(
            e instanceof Error && e.message ? e.message : '加载模型列表失败',
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
  }, [fetchAllModels])

  const refresh = async () => {
    try {
      await fetchAllModels()
    } catch (e) {
      toast.error(
        e instanceof Error && e.message ? e.message : '加载模型列表失败',
      )
    }
  }

  const openCreate = () => {
    setCreateOpen(true)
  }

  return (
    <div className="mx-auto w-full max-w-6xl space-y-5">
      <PageHeader
        className="mb-5"
        title="模型管理"
        description="按逻辑模型名查看与管理各渠道的上游模型"
        actions={
          <Button onClick={openCreate}>
            <Plus />
            新增模型
          </Button>
        }
      />

      {/* 数据表格 */}
      <Card className="py-0">
        <CardContent className="px-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="min-w-[160px]">模型名</TableHead>
                <TableHead className="w-[120px]">渠道数</TableHead>
                <TableHead className="w-[150px]">状态</TableHead>
                <TableHead className="w-[100px]">上下文</TableHead>
                <TableHead className="min-w-[180px]">能力</TableHead>
                <TableHead className="w-[100px]">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {groups.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={6}
                    className="h-32 text-center text-muted-foreground"
                  >
                    {loading ? '加载中...' : '暂无模型'}
                  </TableCell>
                </TableRow>
              ) : (
                groups.map((group) => {
                  const summary = summarizeGroup(group)
                  return (
                    <TableRow key={group.name}>
                      <TableCell className="font-medium">{group.name}</TableCell>
                      <TableCell>
                        <Badge variant="secondary">
                          {group.channel_count} 个渠道
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <DotBadge tone={summary.status.tone}>
                          {summary.status.label}
                        </DotBadge>
                      </TableCell>
                      <TableCell>{summary.contextLabel}</TableCell>
                      <TableCell>
                        {summary.capabilities.length === 0 ? (
                          <span className="text-muted-foreground">-</span>
                        ) : (
                          <div className="flex flex-wrap gap-1">
                            {summary.capabilities.map((cap) => (
                              <Badge key={cap} variant="secondary">
                                {cap}
                              </Badge>
                            ))}
                          </div>
                        )}
                      </TableCell>
                      <TableCell>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          aria-label="查看渠道"
                          onClick={() => setSelectedGroupName(group.name)}
                        >
                          <Boxes />
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

      {/* 渠道明细 */}
      <ModelChannelsDialog
        group={selectedGroup}
        open={selectedGroup !== null}
        onOpenChange={(o) => {
          if (!o) setSelectedGroupName(null)
        }}
        onChanged={() => void refresh()}
      />

      {/* 新增模型 */}
      <ModelFormSheet
        open={createOpen}
        onOpenChange={setCreateOpen}
        editing={null}
        onSuccess={() => void refresh()}
      />
    </div>
  )
}
