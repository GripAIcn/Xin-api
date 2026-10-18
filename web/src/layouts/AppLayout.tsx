import { useEffect, useMemo, useState } from 'react'
import type { LucideIcon } from 'lucide-react'
import {
  Activity,
  Boxes,
  ChevronsUpDown,
  CircleDollarSign,
  KeyRound,
  LayoutDashboard,
  Lock,
  LogOut,
  Menu,
  ReceiptText,
  ScrollText,
  Settings,
  UserCog,
  Users,
  Waypoints,
} from 'lucide-react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { toast } from 'sonner'
import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'

import ThemeToggle from '@/components/ThemeToggle'
import BrandLogo from '@/components/brand/BrandLogo'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Sheet, SheetContent } from '@/components/ui/sheet'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth'

const APP_VERSION = 'v0.1.0'

interface NavItem {
  name: string
  path: string
  icon: LucideIcon
}

interface NavGroup {
  label: string
  items: NavItem[]
}

function buildNavGroups(isAdmin: boolean): NavGroup[] {
  return [
    {
      label: '概览',
      items: [{ name: '仪表盘', path: '/dashboard', icon: LayoutDashboard }],
    },
    {
      label: '资源',
      items: [
        ...(isAdmin
          ? [
              { name: '渠道管理', path: '/channels', icon: Waypoints },
              { name: '渠道监控', path: '/monitoring', icon: Activity },
              { name: '模型管理', path: '/models', icon: Boxes },
              { name: '定价管理', path: '/pricing', icon: CircleDollarSign },
            ]
          : []),
        { name: 'API Key', path: '/apikeys', icon: KeyRound },
        { name: '使用日志', path: '/logs', icon: ScrollText },
        { name: '账单日志', path: '/billing', icon: ReceiptText },
      ],
    },
    {
      label: '系统',
      items: [
        { name: '账户与余额', path: '/settings', icon: Settings },
        ...(isAdmin
          ? [{ name: '用户管理', path: '/users', icon: Users }]
          : []),
      ],
    },
  ]
}

// 标题解析使用包含管理员项的完整导航，保证任意角色下均可解析路径标题
const allNavItems = buildNavGroups(true).flatMap((group) => group.items)

function pageTitle(pathname: string): string {
  const exact =
    pathname.length > 1 && pathname.endsWith('/')
      ? pathname.slice(0, -1)
      : pathname
  return allNavItems.find((item) => item.path === exact)?.name ?? '控制台'
}

function Logo({ collapsed }: { collapsed: boolean }) {
  return (
    <div
      className={cn(
        'flex h-16 shrink-0 items-center overflow-hidden border-b border-sidebar-border',
        collapsed ? 'justify-center px-0' : 'px-4',
      )}
    >
      {collapsed ? (
        <BrandLogo variant="mark" size="md" />
      ) : (
        <BrandLogo variant="lockup" size="md" />
      )}
    </div>
  )
}

function NavGroupList({
  collapsed,
  groups,
  onNavigate,
}: {
  collapsed: boolean
  groups: NavGroup[]
  onNavigate?: () => void
}) {
  return (
    <nav className="flex-1 space-y-4 overflow-y-auto px-2 py-4">
      {groups.map((group) => (
        <div key={group.label}>
          {!collapsed && (
            <p className="mb-1.5 px-3 text-[11px] font-medium tracking-wider text-muted-foreground/80 uppercase">
              {group.label}
            </p>
          )}
          {collapsed && <div className="mx-2 mb-2 h-px bg-sidebar-border" />}
          <div className="space-y-0.5">
            {group.items.map((item) => {
              const Icon = item.icon
              return (
                <NavLink
                  key={item.path}
                  to={item.path}
                  onClick={onNavigate}
                  className={({ isActive }) =>
                    cn(
                      'relative flex items-center gap-3 rounded-lg py-2 text-sm font-medium transition-colors',
                      collapsed ? 'justify-center px-0' : 'px-3',
                      isActive
                        ? 'bg-sidebar-accent text-sidebar-accent-foreground before:absolute before:top-1/2 before:left-0 before:h-5 before:w-[3px] before:-translate-y-1/2 before:rounded-r-full before:bg-sidebar-primary'
                        : 'text-sidebar-foreground/70 hover:bg-sidebar-accent/60 hover:text-sidebar-accent-foreground',
                    )
                  }
                >
                  <Icon className="size-5 shrink-0" />
                  {!collapsed && <span className="truncate">{item.name}</span>}
                </NavLink>
              )
            })}
          </div>
        </div>
      ))}
    </nav>
  )
}

function VersionFooter({ collapsed }: { collapsed: boolean }) {
  return (
    <div
      className={cn(
        'shrink-0 border-t border-sidebar-border px-4 py-3 text-xs text-muted-foreground',
        collapsed && 'px-0 text-center',
      )}
    >
      {collapsed ? (
        <span className="text-[10px]">{APP_VERSION}</span>
      ) : (
        <p>
          Xin API 控制台
          <span className="ml-1.5">{APP_VERSION}</span>
        </p>
      )}
    </div>
  )
}

function UserMenu({
  username,
  onLogout,
}: {
  username: string | null
  onLogout: () => void
}) {
  const navigate = useNavigate()

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          className="h-auto gap-2 rounded-md px-1.5 py-1"
          aria-label="用户菜单"
        >
          <span className="flex size-8 shrink-0 items-center justify-center rounded-full bg-primary text-sm font-medium text-primary-foreground">
            {username?.charAt(0).toUpperCase() || 'U'}
          </span>
          <span className="hidden max-w-[120px] truncate text-sm font-medium sm:inline-block">
            {username || '管理员'}
          </span>
          <ChevronsUpDown className="hidden size-3.5 text-muted-foreground sm:block" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-48">
        <DropdownMenuLabel className="text-muted-foreground">
          {username || '管理员'}
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <DropdownMenuItem onClick={() => navigate('/settings')}>
            <UserCog />
            账户与余额
          </DropdownMenuItem>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          variant="destructive"
          onClick={onLogout}
        >
          <LogOut />
          退出登录
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

const forcePasswordSchema = z
  .object({
    old_password: z
      .string()
      .min(1, '请输入当前密码')
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
  .refine((data) => data.new_password !== data.old_password, {
    message: '新密码不能与当前密码相同',
    path: ['new_password'],
  })

type ForcePasswordForm = z.infer<typeof forcePasswordSchema>

// ForcePasswordChange 首次登录/密码被重置后的强制改密界面，不提供跳过入口
function ForcePasswordChange({
  username,
  onLogout,
}: {
  username: string | null
  onLogout: () => void
}) {
  const updatePassword = useAuthStore((s) => s.updatePassword)
  const form = useForm<ForcePasswordForm>({
    resolver: zodResolver(forcePasswordSchema),
    defaultValues: { old_password: '', new_password: '', confirm_password: '' },
  })

  const onSubmit = async (values: ForcePasswordForm) => {
    try {
      await updatePassword({
        old_password: values.old_password,
        new_password: values.new_password,
      })
      toast.success('密码修改成功')
      form.reset()
    } catch (e) {
      toast.error((e as Error)?.message || '修改失败')
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-muted/40 p-4">
      <Card className="w-full max-w-md">
        <CardHeader>
          <div className="mb-2 flex justify-center">
            <BrandLogo variant="lockup" size="md" />
          </div>
          <div className="flex items-center justify-center gap-2">
            <Lock className="size-4 text-primary" />
            <CardTitle>请先修改密码</CardTitle>
          </div>
          <CardDescription className="text-center">
            {username ? `${username}，` : ''}
            首次登录或密码被重置后，需要设置新密码才能继续使用控制台。
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Form {...form}>
            <form
              onSubmit={form.handleSubmit(onSubmit)}
              className="space-y-4"
            >
              <FormField
                control={form.control}
                name="old_password"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>当前密码</FormLabel>
                    <FormControl>
                      <Input
                        type="password"
                        placeholder="输入当前密码"
                        disabled={form.formState.isSubmitting}
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="new_password"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>新密码</FormLabel>
                    <FormControl>
                      <Input
                        type="password"
                        placeholder="输入新密码（8-20 位）"
                        disabled={form.formState.isSubmitting}
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="confirm_password"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>确认新密码</FormLabel>
                    <FormControl>
                      <Input
                        type="password"
                        placeholder="再次输入新密码"
                        disabled={form.formState.isSubmitting}
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
                disabled={form.formState.isSubmitting}
              >
                {form.formState.isSubmitting ? '保存中...' : '修改密码并继续'}
              </Button>
              <Button
                type="button"
                variant="ghost"
                className="w-full"
                onClick={onLogout}
              >
                退出登录
              </Button>
            </form>
          </Form>
        </CardContent>
      </Card>
    </div>
  )
}

export default function AppLayout() {
  const username = useAuthStore((s) => s.username)
  const isAdmin = useAuthStore((s) => s.isAdmin)
  const mustChangePassword = useAuthStore((s) => s.mustChangePassword)
  const logout = useAuthStore((s) => s.logout)
  const location = useLocation()
  const [collapsed, setCollapsed] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)

  const navGroups = useMemo(() => buildNavGroups(isAdmin), [isAdmin])

  // 挂载时若存在 token 则验证其有效性
  useEffect(() => {
    void useAuthStore.getState().validateToken()
  }, [])

  const title = pageTitle(location.pathname)

  // 强制改密：拦截整个控制台，仅保留改密与退出登录
  if (mustChangePassword) {
    return <ForcePasswordChange username={username} onLogout={logout} />
  }

  return (
    <div className="flex h-screen w-full overflow-hidden">
      {/* 侧边栏 - 桌面端 */}
      <aside
        className={cn(
          'relative hidden shrink-0 flex-col border-r border-sidebar-border bg-sidebar transition-[width] duration-300 lg:flex',
          collapsed ? 'w-16' : 'w-60',
        )}
      >
        <Logo collapsed={collapsed} />
        <NavGroupList collapsed={collapsed} groups={navGroups} />
        <VersionFooter collapsed={collapsed} />

        {/* 折叠按钮 */}
        <Button
          type="button"
          variant="outline"
          size="icon-xs"
          onClick={() => setCollapsed((c) => !c)}
          className="absolute top-20 -right-3 z-10 rounded-full border-sidebar-border bg-background text-muted-foreground shadow-sm hover:bg-background hover:text-foreground"
          aria-label={collapsed ? '展开侧边栏' : '折叠侧边栏'}
        >
          <Menu className="size-3" />
        </Button>
      </aside>

      <div className="flex flex-1 flex-col overflow-hidden">
        {/* 顶部栏 */}
        <header className="flex h-14 shrink-0 items-center gap-2 border-b border-sidebar-border bg-background px-4">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setMobileOpen(true)}
            className="lg:hidden"
            aria-label="打开导航"
          >
            <Menu className="size-5" />
          </Button>
          <h2 className="text-base font-semibold text-foreground">{title}</h2>

          <div className="ml-auto flex items-center gap-2">
            <ThemeToggle />
            <UserMenu username={username} onLogout={logout} />
          </div>
        </header>

        {/* 侧边栏 - 移动端抽屉 */}
        <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
          <SheetContent
            side="left"
            className="w-[240px] gap-0 bg-sidebar p-0"
            showCloseButton={false}
          >
            <Logo collapsed={false} />
            <NavGroupList
              collapsed={false}
              groups={navGroups}
              onNavigate={() => setMobileOpen(false)}
            />
            <VersionFooter collapsed={false} />
          </SheetContent>
        </Sheet>

        {/* 主内容区 */}
        <main className="flex-1 overflow-y-auto bg-muted/40 p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
