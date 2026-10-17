import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Contact, Lock, Wallet } from 'lucide-react'
import { toast } from 'sonner'

import PageHeader from '@/components/PageHeader'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { useAuthStore } from '@/stores/auth'

const passwordSchema = z
  .object({
    old_password: z
      .string()
      .min(1, '请输入旧密码')
      .min(8, '密码长度需在 8-20 个字符之间')
      .max(20, '密码长度需在 8-20 个字符之间'),
    new_password: z
      .string()
      .min(1, '请输入新密码')
      .min(8, '密码长度需在 8-20 个字符之间')
      .max(20, '密码长度需在 8-20 个字符之间'),
    confirm_password: z.string().min(1, '请确认新密码'),
  })
  .refine((data) => data.confirm_password === data.new_password, {
    message: '两次输入的密码不一致',
    path: ['confirm_password'],
  })

type PasswordForm = z.infer<typeof passwordSchema>

const profileSchema = z
  .object({
    email: z
      .string()
      .email('请输入有效的邮箱地址')
      .optional()
      .or(z.literal('')),
    phone: z.string().optional(),
    old_password: z
      .string()
      .min(1, '请输入当前密码')
      .min(8, '密码长度需在 8-20 个字符之间')
      .max(20, '密码长度需在 8-20 个字符之间'),
  })
  .refine((data) => !!data.email || !!data.phone, {
    message: '请至少填写邮箱或手机号',
    path: ['email'],
  })

type ProfileForm = z.infer<typeof profileSchema>

export default function SettingsView() {
  const updatePassword = useAuthStore((s) => s.updatePassword)
  const updateProfile = useAuthStore((s) => s.updateProfile)
  const profile = useAuthStore((s) => s.profile)
  const fetchProfile = useAuthStore((s) => s.fetchProfile)

  useEffect(() => {
    void fetchProfile()
  }, [fetchProfile])

  const passwordForm = useForm<PasswordForm>({
    resolver: zodResolver(passwordSchema),
    defaultValues: { old_password: '', new_password: '', confirm_password: '' },
  })

  const profileForm = useForm<ProfileForm>({
    resolver: zodResolver(profileSchema),
    defaultValues: { email: '', phone: '', old_password: '' },
  })

  const onSubmitPassword = async (values: PasswordForm) => {
    try {
      await updatePassword({
        old_password: values.old_password,
        new_password: values.new_password,
      })
      toast.success('密码修改成功')
      passwordForm.reset()
    } catch (e) {
      toast.error((e as Error)?.message || '修改失败')
    }
  }

  const onSubmitProfile = async (values: ProfileForm) => {
    try {
      await updateProfile({
        old_password: values.old_password,
        email: values.email || undefined,
        phone: values.phone || undefined,
      })
      toast.success('资料更新成功')
      profileForm.reset()
    } catch (e) {
      toast.error((e as Error)?.message || '更新失败')
    }
  }

  return (
    <div className="mx-auto w-full max-w-6xl">
      <PageHeader
        className="mb-6"
        title="账户与余额"
        description="查看余额并管理账户信息与登录密码"
      />

      {/* 余额概览（统一以 CNY 结算） */}
      <Card className="mb-6">
        <CardHeader>
          <div className="flex items-center gap-3">
            <span className="flex size-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
              <Wallet className="size-[18px]" />
            </span>
            <div className="flex flex-col gap-1">
              <CardTitle>余额概览</CardTitle>
              <CardDescription>账户余额统一以 CNY 结算</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-3">
            <div className="flex flex-col gap-1">
              <span className="text-xs text-muted-foreground">可用余额</span>
              <span className="text-xl font-semibold tabular-nums">
                {profile
                  ? `${(profile.balance ?? 0).toFixed(4)} CNY`
                  : '--'}
              </span>
            </div>
            <div className="flex flex-col gap-1">
              <span className="text-xs text-muted-foreground">累计已用额度</span>
              <span className="text-xl font-semibold tabular-nums">
                {profile
                  ? `${(profile.used_quota ?? 0).toFixed(4)} CNY`
                  : '--'}
              </span>
            </div>
            <div className="flex flex-col gap-1">
              <span className="text-xs text-muted-foreground">额度上限</span>
              <span className="text-xl font-semibold tabular-nums">
                {profile
                  ? profile.quota == null
                    ? '不限'
                    : `${profile.quota.toFixed(4)} CNY`
                  : '--'}
              </span>
            </div>
          </div>
        </CardContent>
      </Card>

      <div className="grid gap-6 md:grid-cols-2">
        <Card className="h-full">
          <CardHeader>
            <div className="flex items-center gap-3">
              <span className="flex size-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <Lock className="size-[18px]" />
              </span>
              <div className="flex flex-col gap-1">
                <CardTitle>修改密码</CardTitle>
                <CardDescription>密码长度 8-20 位</CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <Form {...passwordForm}>
              <form
                onSubmit={passwordForm.handleSubmit(onSubmitPassword)}
                className="space-y-4"
              >
                <FormField
                  control={passwordForm.control}
                  name="old_password"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>旧密码</FormLabel>
                      <FormControl>
                        <Input
                          type="password"
                          placeholder="输入旧密码"
                          disabled={passwordForm.formState.isSubmitting}
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={passwordForm.control}
                  name="new_password"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>新密码</FormLabel>
                      <FormControl>
                        <Input
                          type="password"
                          placeholder="输入新密码"
                          disabled={passwordForm.formState.isSubmitting}
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={passwordForm.control}
                  name="confirm_password"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>确认密码</FormLabel>
                      <FormControl>
                        <Input
                          type="password"
                          placeholder="再次输入新密码"
                          disabled={passwordForm.formState.isSubmitting}
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <Button
                  type="submit"
                  className="w-full"
                  disabled={passwordForm.formState.isSubmitting}
                >
                  {passwordForm.formState.isSubmitting ? '保存中...' : '修改密码'}
                </Button>
              </form>
            </Form>
          </CardContent>
        </Card>

        <Card className="h-full">
          <CardHeader>
            <div className="flex items-center gap-3">
              <span className="flex size-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <Contact className="size-[18px]" />
              </span>
              <div className="flex flex-col gap-1">
                <CardTitle>修改资料</CardTitle>
                <CardDescription>修改邮箱/手机号需验证当前密码</CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <Form {...profileForm}>
              <form
                onSubmit={profileForm.handleSubmit(onSubmitProfile)}
                className="space-y-4"
              >
                <FormField
                  control={profileForm.control}
                  name="email"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>邮箱</FormLabel>
                      <FormControl>
                        <Input
                          type="email"
                          placeholder="输入邮箱（选填）"
                          disabled={profileForm.formState.isSubmitting}
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={profileForm.control}
                  name="phone"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>手机号</FormLabel>
                      <FormControl>
                        <Input
                          placeholder="输入手机号（选填）"
                          disabled={profileForm.formState.isSubmitting}
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={profileForm.control}
                  name="old_password"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>当前密码</FormLabel>
                      <FormControl>
                        <Input
                          type="password"
                          placeholder="输入当前密码以验证身份"
                          disabled={profileForm.formState.isSubmitting}
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <Button
                  type="submit"
                  className="w-full"
                  disabled={profileForm.formState.isSubmitting}
                >
                  {profileForm.formState.isSubmitting ? '保存中...' : '保存修改'}
                </Button>
              </form>
            </Form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}