import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Pencil, Plus, Trash2 } from 'lucide-react'
import { toast } from 'sonner'

import { useGroupStore } from '@/stores/groups'
import type { Group } from '@/types/api'
import { formatDate } from '@/lib/format'
import { groupStatusLabel } from '@/lib/status'
import { DotBadge } from '@/components/StatusBadge'
import PageHeader from '@/components/PageHeader'
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'

const groupFormSchema = z.object({
  name: z.string().superRefine((value, ctx) => {
    if (!value) {
      ctx.addIssue({ code: 'custom', message: '请输入项目组名称' })
      return
    }
    if (value.length < 2 || value.length > 100) {
      ctx.addIssue({ code: 'custom', message: '长度需在 2-100 个字符之间' })
    }
  }),
})

type GroupFormValues = z.infer<typeof groupFormSchema>

export default function GroupsView() {
  const navigate = useNavigate()
  const groups = useGroupStore((s) => s.groups)
  const fetchAll = useGroupStore((s) => s.fetchAll)
  const create = useGroupStore((s) => s.create)
  const update = useGroupStore((s) => s.update)
  const toggleStatus = useGroupStore((s) => s.toggleStatus)
  const remove = useGroupStore((s) => s.remove)

  const [loading, setLoading] = useState(true)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [dialogType, setDialogType] = useState<'create' | 'edit'>('create')
  const [editingId, setEditingId] = useState(0)
  const [submitting, setSubmitting] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<Group | null>(null)

  const form = useForm<GroupFormValues>({
    resolver: zodResolver(groupFormSchema),
    defaultValues: { name: '' },
  })

  useEffect(() => {
    fetchAll()
      .catch((e: unknown) => {
        toast.error((e as Error)?.message || '加载项目组列表失败')
      })
      .finally(() => setLoading(false))
  }, [fetchAll])

  const openCreate = () => {
    setDialogType('create')
    setEditingId(0)
    form.reset({ name: '' })
    setDialogOpen(true)
  }

  const openEdit = (row: Group) => {
    setDialogType('edit')
    setEditingId(row.id)
    form.reset({ name: row.name })
    setDialogOpen(true)
  }

  const onSubmit = async (values: GroupFormValues) => {
    setSubmitting(true)
    try {
      if (dialogType === 'create') {
        await create(values.name)
        toast.success('项目组创建成功')
      } else {
        await update(editingId, values.name)
        toast.success('项目组更新成功')
      }
      setDialogOpen(false)
    } catch (e: unknown) {
      toast.error((e as Error)?.message || '操作失败')
    } finally {
      setSubmitting(false)
    }
  }

  const handleToggleStatus = async (row: Group) => {
    try {
      await toggleStatus(row.id, row.status)
      toast.success(row.status === 1 ? '项目组已停用' : '项目组已启用')
    } catch (e: unknown) {
      toast.error((e as Error)?.message || '操作失败')
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await remove(deleteTarget.id)
      toast.success('项目组已删除')
    } catch (e: unknown) {
      toast.error((e as Error)?.message || '删除失败')
    } finally {
      setDeleteTarget(null)
    }
  }

  return (
    <div className="mx-auto w-full max-w-6xl">
      <PageHeader
        className="mb-5"
        title="项目组"
        description="管理 API 项目组及其状态"
        actions={
          <Button onClick={openCreate}>
            <Plus />
            创建项目组
          </Button>
        }
      />

      <Card>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-20">ID</TableHead>
                <TableHead>名称</TableHead>
                <TableHead className="w-24">状态</TableHead>
                <TableHead className="w-48">创建时间</TableHead>
                <TableHead className="w-48 text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                Array.from({ length: 3 }).map((_, index) => (
                  <TableRow key={index}>
                    <TableCell colSpan={5}>
                      <Skeleton className="h-6 w-full" />
                    </TableCell>
                  </TableRow>
                ))
              ) : groups.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="py-10 text-center text-muted-foreground">
                    暂无项目组，点击上方按钮创建
                  </TableCell>
                </TableRow>
              ) : (
                groups.map((row) => (
                  <TableRow key={row.id}>
                    <TableCell>{row.id}</TableCell>
                    <TableCell>
                      <Button
                        variant="link"
                        className="h-auto p-0"
                        onClick={() => navigate(`/groups/${row.id}`)}
                      >
                        {row.name}
                      </Button>
                    </TableCell>
                    <TableCell>
                      <DotBadge tone={row.status === 1 ? 'success' : 'muted'}>
                        {groupStatusLabel(row.status)}
                      </DotBadge>
                    </TableCell>
                    <TableCell>{formatDate(row.created_at)}</TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-3">
                        <Switch
                          checked={row.status === 1}
                          onCheckedChange={() => handleToggleStatus(row)}
                        />
                        <Button variant="ghost" size="icon-sm" onClick={() => openEdit(row)}>
                          <Pencil />
                          <span className="sr-only">编辑</span>
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          className="text-destructive hover:text-destructive"
                          onClick={() => setDeleteTarget(row)}
                        >
                          <Trash2 />
                          <span className="sr-only">删除</span>
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{dialogType === 'create' ? '创建项目组' : '编辑项目组'}</DialogTitle>
          </DialogHeader>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>项目组名称</FormLabel>
                    <FormControl>
                      <Input placeholder="输入名称" maxLength={100} disabled={submitting} {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <DialogFooter>
                <Button
                  type="button"
                  variant="outline"
                  disabled={submitting}
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
              确定要删除项目组「{deleteTarget?.name}」吗？此操作不可撤销，但不会影响已创建的 API Key。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={(event) => {
                event.preventDefault()
                void handleDelete()
              }}
            >
              确认删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}