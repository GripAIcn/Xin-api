import { useEffect, useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import {
  CircleOff,
  CirclePlay,
  KeyRound,
  Loader2,
  UserPlus,
  Wallet,
} from 'lucide-react'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'

import PageHeader from '@/components/PageHeader'
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
import { formatDate } from '@/lib/format'
import { useUsersStore } from '@/stores/users'
import type { User } from '@/types/api'

const createUserSchema = z.object({
  username: z
    .string()
    .min(1, '请输入用户名')
    .min(5, '用户名长度需在 5-10 个字符之间')
    .max(10, '用户名长度需在 5-10 个字符之间'),
  password: z
    .string()
    .min(1, '请输入密码')
    .min(8, '密码长度需在 8-20 个字符之间')
    .max(20, '密码长度需在 8-20 个字符之间'),
  email: z.union([z.literal(''), z.email('请输入有效的邮箱')]),
  phone: z.string(),
})

type CreateUserValues = z.infer<typeof createUserSchema>

const rechargeSchema = z.object({
  amount: z.number().positive('金额必须大于 0'),
})

type RechargeValues = z.infer<typeof rechargeSchema>

const resetPasswordSchema = z.object({
  new_password: z
    .string()
    .min(1, '请输入新密码')
    .min(8, '密码长度需在 8-20 个字符之间')
    .max(20, '密码长度需在 8-20 个字符之间'),
})

type ResetPasswordValues = z.infer<typeof resetPasswordSchema>

// 可选字段：空字符串转 undefined，避免提交空值
const optionalText = (value: string): string | undefined => {
  const trimmed = value.trim()
  return trimmed ? trimmed : undefined
}

export default function UsersView() {
  const users = useUsersStore((s) => s.users)
  const loading = useUsersStore((s) => s.loading)
  const fetchAll = useUsersStore((s) => s.fetchAll)
  const create = useUsersStore((s) => s.create)
  const updateStatus = useUsersStore((s) => s.updateStatus)
  const resetPassword = useUsersStore((s) => s.resetPassword)
  const recharge = useUsersStore((s) => s.recharge)

  const [createOpen, setCreateOpen] = useState(false)
  const [creating, setCreating] = useState(false)

  const [statusTarget, setStatusTarget] = useState<User | null>(null)
  const [statusSubmitting, setStatusSubmitting] = useState(false)

  const [rechargeTarget, setRechargeTarget] = useState<User | null>(null)
  const [rechargeSubmitting, setRechargeSubmitting] = useState(false)

  const [resetTarget, setResetTarget] = useState<User | null>(null)
  const [resetSubmitting, setResetSubmitting] = useState(false)

  const createForm = useForm<CreateUserValues>({
    resolver: zodResolver(createUserSchema),
    defaultValues: { username: '', password: '', email: '', phone: '' },
  })

  const rechargeForm = useForm<RechargeValues>({
    resolver: zodResolver(rechargeSchema),
    defaultValues: { amount: 0 },
  })

  const resetForm = useForm<ResetPasswordValues>({
    resolver: zodResolver(resetPasswordSchema),
    defaultValues: { new_password: '' },
  })

  useEffect(() => {
    const load = async () => {
      try {
        await fetchAll()
      } catch (e) {
        toast.error((e as Error)?.message || '加载用户列表失败')
      }
    }

    void load()
  }, [fetchAll])

  const openCreate = () => {
    createForm.reset({ username: '', password: '', email: '', phone: '' })
    setCreateOpen(true)
  }

  const onSubmitCreate = async (values: CreateUserValues) => {
    setCreating(true)
    try {
      await create({
        username: values.username,
        password: values.password,
        email: optionalText(values.email),
        phone: optionalText(values.phone),
      })
      toast.success('用户创建成功')
      setCreateOpen(false)
    } catch (e) {
      toast.error((e as Error)?.message || '创建失败')
    } finally {
      setCreating(false)
    }
  }

  const openRecharge = (row: User) => {
    rechargeForm.reset({ amount: 0 })
    setRechargeTarget(row)
  }

  const onSubmitRecharge = async (values: RechargeValues) => {
    if (!rechargeTarget) return
    setRechargeSubmitting(true)
    try {
      await recharge(rechargeTarget.id, { amount: values.amount })
      toast.success('充值成功')
      setRechargeTarget(null)
    } catch (e) {
      toast.error((e as Error)?.message || '充值失败')
    } finally {
      setRechargeSubmitting(false)
    }
  }

  const openReset = (row: User) => {
    resetForm.reset({ new_password: '' })
    setResetTarget(row)
  }

  const onSubmitReset = async (values: ResetPasswordValues) => {
    if (!resetTarget) return
    setResetSubmitting(true)
    try {
      await resetPassword(resetTarget.id, { new_password: values.new_password })
      toast.success('密码重置成功')
      setResetTarget(null)
    } catch (e) {
      toast.error((e as Error)?.message || '重置密码失败')
    } finally {
      setResetSubmitting(false)
    }
  }

  const handleToggleStatus = async () => {
    if (!statusTarget) return
    const nextStatus = statusTarget.status === 'active' ? 'disabled' : 'active'
    setStatusSubmitting(true)
    try {
      await updateStatus(statusTarget.id, { status: nextStatus })
      toast.success(nextStatus === 'active' ? '用户已启用' : '用户已禁用')
      setStatusTarget(null)
    } catch (e) {
      toast.error((e as Error)?.message || '操作失败')
    } finally {
      setStatusSubmitting(false)
    }
  }

  return (
    <div className="mx-auto w-full max-w-6xl space-y-5">
      <PageHeader
        className="mb-5"
        title="用户管理"
        description="管理平台用户账号，仅管理员可操作"
        actions={
          <Button onClick={openCreate}>
            <UserPlus />
            新建用户
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
                <TableHead className="min-w-[120px]">用户名</TableHead>
                <TableHead className="min-w-[160px]">邮箱</TableHead>
                <TableHead className="min-w-[120px]">手机号</TableHead>
                <TableHead className="w-[90px]">角色</TableHead>
                <TableHead className="w-[90px]">状态</TableHead>
                <TableHead className="w-[120px]">余额</TableHead>
                <TableHead className="w-[170px]">创建时间</TableHead>
                <TableHead className="w-[130px]">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell
                    colSpan={9}
                    className="h-32 text-center text-muted-foreground"
                  >
                    <span className="inline-flex items-center gap-2">
                      <Loader2 className="size-4 animate-spin" />
                      加载中...
                    </span>
                  </TableCell>
                </TableRow>
              ) : users.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={9}
                    className="h-32 text-center text-muted-foreground"
                  >
                    暂无用户
                  </TableCell>
                </TableRow>
              ) : (
                users.map((row) => {
                  const isAdminRow = row.role === 'admin'
                  const isActive = row.status === 'active'
                  // 管理员账号不可禁用，避免误操作自锁
                  const toggleBlocked = isAdminRow && isActive
                  return (
                    <TableRow key={row.id}>
                      <TableCell>{row.id}</TableCell>
                      <TableCell className="font-medium">
                        {row.username}
                      </TableCell>
                      <TableCell>{row.email || '-'}</TableCell>
                      <TableCell>{row.phone || '-'}</TableCell>
                      <TableCell>
                        {isAdminRow ? (
                          <Badge>管理员</Badge>
                        ) : (
                          <Badge variant="secondary">用户</Badge>
                        )}
                      </TableCell>
                      <TableCell>
                        <DotBadge tone={isActive ? 'success' : 'muted'}>
                          {isActive ? '启用' : '禁用'}
                        </DotBadge>
                      </TableCell>
                      <TableCell>
                        {row.balance} {row.currency || 'USD'}
                      </TableCell>
                      <TableCell>{formatDate(row.created_at)}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-1">
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            aria-label={isActive ? '禁用' : '启用'}
                            title={
                              toggleBlocked
                                ? '管理员账号不可禁用'
                                : isActive
                                  ? '禁用用户'
                                  : '启用用户'
                            }
                            disabled={toggleBlocked}
                            onClick={() => setStatusTarget(row)}
                          >
                            {isActive ? <CircleOff /> : <CirclePlay />}
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            aria-label="充值"
                            title="充值"
                            onClick={() => openRecharge(row)}
                          >
                            <Wallet />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            aria-label="重置密码"
                            title={
                              isAdminRow ? '管理员账号不可重置密码' : '重置密码'
                            }
                            disabled={isAdminRow}
                            onClick={() => openReset(row)}
                          >
                            <KeyRound />
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

      {/* 新建用户 */}
      <Dialog
        open={createOpen}
        onOpenChange={(open) => {
          setCreateOpen(open)
        }}
      >
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle>新建用户</DialogTitle>
            <DialogDescription className="sr-only">
              填写用户信息
            </DialogDescription>
          </DialogHeader>
          <Form {...createForm}>
            <form
              onSubmit={(event) => {
                void createForm.handleSubmit(onSubmitCreate)(event)
              }}
              className="space-y-4"
            >
              <FormField
                control={createForm.control}
                name="username"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>用户名</FormLabel>
                    <FormControl>
                      <Input
                        placeholder="5-10 个字符"
                        maxLength={10}
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={createForm.control}
                name="password"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>密码</FormLabel>
                    <FormControl>
                      <Input
                        type="password"
                        placeholder="8-20 个字符"
                        maxLength={20}
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={createForm.control}
                name="email"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>邮箱（可选）</FormLabel>
                    <FormControl>
                      <Input placeholder="例如：user@example.com" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={createForm.control}
                name="phone"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>手机号（可选）</FormLabel>
                    <FormControl>
                      <Input placeholder="例如：13800138000" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <DialogFooter>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setCreateOpen(false)}
                >
                  取消
                </Button>
                <Button type="submit" disabled={creating}>
                  {creating ? '创建中...' : '创建'}
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>

      {/* 启用/禁用确认 */}
      <AlertDialog
        open={statusTarget !== null}
        onOpenChange={(open) => {
          if (!open) setStatusTarget(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {statusTarget?.status === 'active' ? '确认禁用' : '确认启用'}
            </AlertDialogTitle>
            <AlertDialogDescription>
              确定要{statusTarget?.status === 'active' ? '禁用' : '启用'}用户
              「{statusTarget?.username}」吗？
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              variant={
                statusTarget?.status === 'active' ? 'destructive' : 'default'
              }
              disabled={statusSubmitting}
              onClick={() => void handleToggleStatus()}
            >
              {statusSubmitting
                ? '处理中...'
                : statusTarget?.status === 'active'
                  ? '确认禁用'
                  : '确认启用'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 充值 */}
      <Dialog
        open={rechargeTarget !== null}
        onOpenChange={(open) => {
          if (!open) setRechargeTarget(null)
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>充值</DialogTitle>
            <DialogDescription>
              为用户「{rechargeTarget?.username}」增加余额（当前{' '}
              {rechargeTarget?.balance} {rechargeTarget?.currency || 'USD'}）
            </DialogDescription>
          </DialogHeader>
          <Form {...rechargeForm}>
            <form
              onSubmit={(event) => {
                void rechargeForm.handleSubmit(onSubmitRecharge)(event)
              }}
              className="space-y-4"
            >
              <FormField
                control={rechargeForm.control}
                name="amount"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>充值金额</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        min={0}
                        step="0.01"
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
                  onClick={() => setRechargeTarget(null)}
                >
                  取消
                </Button>
                <Button type="submit" disabled={rechargeSubmitting}>
                  {rechargeSubmitting ? '充值中...' : '确认充值'}
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>

      {/* 重置密码 */}
      <Dialog
        open={resetTarget !== null}
        onOpenChange={(open) => {
          if (!open) setResetTarget(null)
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>重置密码</DialogTitle>
            <DialogDescription>
              为用户「{resetTarget?.username}」设置新密码
            </DialogDescription>
          </DialogHeader>
          <Form {...resetForm}>
            <form
              onSubmit={(event) => {
                void resetForm.handleSubmit(onSubmitReset)(event)
              }}
              className="space-y-4"
            >
              <FormField
                control={resetForm.control}
                name="new_password"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>新密码</FormLabel>
                    <FormControl>
                      <Input
                        type="password"
                        placeholder="8-20 个字符"
                        maxLength={20}
                        {...field}
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
                  onClick={() => setResetTarget(null)}
                >
                  取消
                </Button>
                <Button type="submit" disabled={resetSubmitting}>
                  {resetSubmitting ? '重置中...' : '确认重置'}
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
