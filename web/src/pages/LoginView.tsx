import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Eye, EyeOff, LoaderCircle, Lock, User } from 'lucide-react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { useAuthStore } from '@/stores/auth'

const loginSchema = z.object({
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
})

type LoginForm = z.infer<typeof loginSchema>

const features = ['统一渠道与密钥管理', '智能负载均衡与熔断', '实时流量监控与统计']

export default function LoginView() {
  const navigate = useNavigate()
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated)
  const login = useAuthStore((s) => s.login)
  const [showPassword, setShowPassword] = useState(false)

  const form = useForm<LoginForm>({
    resolver: zodResolver(loginSchema),
    defaultValues: { username: '', password: '' },
  })

  useEffect(() => {
    if (isAuthenticated) {
      navigate('/dashboard', { replace: true })
    }
  }, [isAuthenticated, navigate])

  const onSubmit = async (values: LoginForm) => {
    try {
      await login(values.username, values.password)
      navigate('/dashboard', { replace: true })
    } catch (e) {
      toast.error((e as Error).message || '登录失败，请重试')
    }
  }

  return (
    <div className="flex min-h-screen w-full">
      {/* 左侧装饰区域 */}
      <div className="relative hidden flex-col justify-center overflow-hidden bg-primary p-16 text-primary-foreground lg:flex lg:flex-1">
        <div className="pointer-events-none absolute -top-1/2 -right-1/5 h-[200%] w-4/5 bg-[radial-gradient(ellipse,rgb(255_255_255/0.1)_0%,transparent_70%)]" />

        <div className="relative z-10">
          <div className="mb-10 flex items-center gap-4">
            <div className="flex size-14 items-center justify-center rounded-xl bg-white/15 text-2xl font-bold text-white shadow-lg shadow-black/25">
              X
            </div>
            <span className="text-2xl font-semibold tracking-wide">Xin-api</span>
          </div>
          <h1 className="mb-5 text-4xl font-semibold leading-tight">
            API 网关管理控制台
          </h1>
          <p className="mb-16 max-w-[400px] text-base leading-relaxed opacity-80">
            统一管理上游渠道和 API Key，
            <br />
            轻松构建高性能的 AI 服务接入层
          </p>
        </div>

        <div className="relative z-10 flex flex-col gap-5">
          {features.map((feature) => (
            <div
              key={feature}
              className="flex items-center gap-4 text-[15px] opacity-90"
            >
              <div className="size-2 rounded-full bg-white/90 shadow-[0_0_12px_rgb(255_255_255/0.45)]" />
              <span>{feature}</span>
            </div>
          ))}
        </div>
      </div>

      {/* 右侧登录表单 */}
      <div className="flex w-full flex-1 items-center justify-center bg-background p-10 lg:w-[480px] lg:flex-none">
        <div className="w-full max-w-[360px]">
          <div className="mb-10">
            <h2 className="mb-2 text-[28px] font-semibold text-foreground">
              欢迎登录
            </h2>
            <p className="text-sm text-muted-foreground">请输入账号密码登录系统</p>
          </div>

          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
              <FormField
                control={form.control}
                name="username"
                render={({ field }) => (
                  <FormItem>
                    <div className="relative">
                      <User className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                      <FormControl>
                        <Input
                          placeholder="请输入用户名"
                          autoComplete="username"
                          className="h-10 pl-10"
                          {...field}
                        />
                      </FormControl>
                    </div>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="password"
                render={({ field }) => (
                  <FormItem>
                    <div className="relative">
                      <Lock className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                      <FormControl>
                        <Input
                          type={showPassword ? 'text' : 'password'}
                          placeholder="请输入密码"
                          autoComplete="current-password"
                          className="h-10 px-10"
                          {...field}
                        />
                      </FormControl>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        onClick={() => setShowPassword((v) => !v)}
                        className="absolute right-3 top-1/2 size-auto -translate-y-1/2 text-muted-foreground hover:bg-transparent hover:text-foreground"
                        aria-label={showPassword ? '隐藏密码' : '显示密码'}
                      >
                        {showPassword ? (
                          <EyeOff className="size-4" />
                        ) : (
                          <Eye className="size-4" />
                        )}
                      </Button>
                    </div>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <Button
                type="submit"
                size="lg"
                disabled={form.formState.isSubmitting}
                className="mt-2 h-11 w-full text-base"
              >
                {form.formState.isSubmitting && (
                  <LoaderCircle className="size-4 animate-spin" />
                )}
                {form.formState.isSubmitting ? '登录中...' : '登录'}
              </Button>
            </form>
          </Form>

          <div className="mt-10 text-center text-xs text-muted-foreground/70">
            <p>© 2026 Xin-api. All rights reserved.</p>
          </div>
        </div>
      </div>
    </div>
  )
}