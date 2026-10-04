import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Lock, User } from 'lucide-react'
import { toast } from 'sonner'

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

const usernameSchema = z.object({
  username: z
    .string()
    .min(1, '请输入用户名')
    .min(5, '用户名长度需在 5-10 个字符之间')
    .max(10, '用户名长度需在 5-10 个字符之间'),
})

type UsernameForm = z.infer<typeof usernameSchema>

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

export default function SettingsView() {
  const username = useAuthStore((s) => s.username)
  const updateAccount = useAuthStore((s) => s.updateAccount)
  const updatePassword = useAuthStore((s) => s.updatePassword)

  const usernameForm = useForm<UsernameForm>({
    resolver: zodResolver(usernameSchema),
    defaultValues: { username: username ?? '' },
  })

  const passwordForm = useForm<PasswordForm>({
    resolver: zodResolver(passwordSchema),
    defaultValues: { old_password: '', new_password: '', confirm_password: '' },
  })

  const onSubmitUsername = async (values: UsernameForm) => {
    try {
      await updateAccount({ username: values.username })
      toast.success('用户名修改成功')
    } catch (e) {
      toast.error((e as Error)?.message || '修改失败')
    }
  }

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

  return (
    <div className="mx-auto w-full max-w-6xl">
      <div className="mb-6">
        <h2 className="mb-2 text-[22px] font-semibold text-foreground">账户设置</h2>
        <p className="text-sm text-muted-foreground">管理你的账户信息和登录密码</p>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        <Card className="h-full">
          <CardHeader>
            <div className="flex items-center gap-3">
              <span className="flex size-9 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <User className="size-[18px]" />
              </span>
              <div className="flex flex-col gap-1">
                <CardTitle>修改用户名</CardTitle>
                <CardDescription>当前：{username}</CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <Form {...usernameForm}>
              <form
                onSubmit={usernameForm.handleSubmit(onSubmitUsername)}
                className="space-y-4"
              >
                <FormField
                  control={usernameForm.control}
                  name="username"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>新用户名</FormLabel>
                      <FormControl>
                        <Input
                          placeholder="输入新用户名"
                          maxLength={10}
                          disabled={usernameForm.formState.isSubmitting}
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
                  disabled={usernameForm.formState.isSubmitting}
                >
                  {usernameForm.formState.isSubmitting ? '保存中...' : '保存修改'}
                </Button>
              </form>
            </Form>
          </CardContent>
        </Card>

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
      </div>
    </div>
  )
}