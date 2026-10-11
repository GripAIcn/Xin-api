import { useEffect, useState } from 'react'
import type { LucideIcon } from 'lucide-react'
import {
  ChevronsUpDown,
  KeyRound,
  LayoutDashboard,
  LogOut,
  Menu,
  Settings,
  UserCog,
  Waypoints,
} from 'lucide-react'
import { NavLink, Outlet, useLocation, useNavigate } from 'react-router-dom'

import ThemeToggle from '@/components/ThemeToggle'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
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

const navGroups: NavGroup[] = [
  {
    label: '概览',
    items: [{ name: '仪表盘', path: '/dashboard', icon: LayoutDashboard }],
  },
  {
    label: '资源',
    items: [
      { name: '渠道', path: '/channels', icon: Waypoints },
      { name: 'API Key', path: '/apikeys', icon: KeyRound },
    ],
  },
  {
    label: '系统',
    items: [{ name: '账户设置', path: '/settings', icon: Settings }],
  },
]

const allNavItems = navGroups.flatMap((group) => group.items)

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
        'flex h-16 shrink-0 items-center border-b border-sidebar-border px-4',
        collapsed ? 'justify-center px-0' : 'gap-3',
      )}
    >
      <div className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-sidebar-primary text-lg font-bold text-sidebar-primary-foreground">
        X
      </div>
      {!collapsed && (
        <span className="text-lg font-semibold tracking-tight text-sidebar-foreground">
          Xin-api
        </span>
      )}
    </div>
  )
}

function NavGroupList({
  collapsed,
  onNavigate,
}: {
  collapsed: boolean
  onNavigate?: () => void
}) {
  return (
    <nav className="flex-1 space-y-4 overflow-y-auto px-2 py-4">
      {navGroups.map((group) => (
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
                      'relative flex items-center gap-3 rounded-md py-2 text-sm font-medium transition-colors',
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
          Xin-api 控制台
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
        <button
          type="button"
          className="flex items-center gap-2 rounded-md px-1.5 py-1 transition-colors hover:bg-accent focus-visible:outline-ring/50 focus-visible:outline-2"
          aria-label="用户菜单"
        >
          <span className="flex size-8 shrink-0 items-center justify-center rounded-full bg-primary text-sm font-medium text-primary-foreground">
            {username?.charAt(0).toUpperCase() || 'U'}
          </span>
          <span className="hidden max-w-[120px] truncate text-sm font-medium sm:inline-block">
            {username || '管理员'}
          </span>
          <ChevronsUpDown className="hidden size-3.5 text-muted-foreground sm:block" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-48">
        <DropdownMenuLabel className="text-muted-foreground">
          {username || '管理员'}
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuGroup>
          <DropdownMenuItem onClick={() => navigate('/settings')}>
            <UserCog />
            账户设置
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

export default function AppLayout() {
  const username = useAuthStore((s) => s.username)
  const logout = useAuthStore((s) => s.logout)
  const location = useLocation()
  const [collapsed, setCollapsed] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)

  // 挂载时若存在 token 则验证其有效性
  useEffect(() => {
    void useAuthStore.getState().validateToken()
  }, [])

  const title = pageTitle(location.pathname)

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
        <NavGroupList collapsed={collapsed} />
        <VersionFooter collapsed={collapsed} />

        {/* 折叠按钮 */}
        <button
          type="button"
          onClick={() => setCollapsed((c) => !c)}
          className="absolute top-20 -right-3 z-10 flex size-6 items-center justify-center rounded-full border border-sidebar-border bg-background text-muted-foreground shadow-sm transition-colors hover:text-foreground"
          aria-label={collapsed ? '展开侧边栏' : '折叠侧边栏'}
        >
          <Menu className="size-3" />
        </button>
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

          <div className="ml-auto flex items-center gap-1.5">
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
              onNavigate={() => setMobileOpen(false)}
            />
            <VersionFooter collapsed={false} />
          </SheetContent>
        </Sheet>

        {/* 主内容区 */}
        <main className="flex-1 overflow-y-auto bg-muted/40 p-5 lg:p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
