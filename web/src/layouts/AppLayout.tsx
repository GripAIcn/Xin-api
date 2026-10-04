import { useEffect, useState } from 'react'
import type { LucideIcon } from 'lucide-react'
import {
  ChevronLeft,
  ChevronRight,
  FolderKanban,
  KeyRound,
  LayoutDashboard,
  LogOut,
  Menu,
  Settings,
  Waypoints,
} from 'lucide-react'
import { NavLink, Outlet } from 'react-router-dom'

import { Button } from '@/components/ui/button'
import { Sheet, SheetContent } from '@/components/ui/sheet'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth'

interface NavItem {
  name: string
  path: string
  icon: LucideIcon
}

const navigation: NavItem[] = [
  { name: '仪表盘', path: '/dashboard', icon: LayoutDashboard },
  { name: '项目组', path: '/groups', icon: FolderKanban },
  { name: '渠道', path: '/channels', icon: Waypoints },
  { name: 'API Key', path: '/apikeys', icon: KeyRound },
  { name: '账户设置', path: '/settings', icon: Settings },
]

function Logo({ collapsed }: { collapsed: boolean }) {
  return (
    <div className="flex h-[60px] shrink-0 items-center justify-center border-b border-sidebar-border">
      <div className="flex items-center gap-3">
        <div className="flex size-9 items-center justify-center rounded-lg bg-primary text-lg font-bold text-primary-foreground">
          X
        </div>
        {!collapsed && (
          <span className="text-lg font-semibold text-sidebar-foreground">
            Xin-api
          </span>
        )}
      </div>
    </div>
  )
}

function NavItems({
  collapsed,
  onNavigate,
}: {
  collapsed: boolean
  onNavigate?: () => void
}) {
  return (
    <nav className="flex flex-1 flex-col gap-1 overflow-y-auto p-2">
      {navigation.map((item) => {
        const Icon = item.icon
        return (
          <NavLink
            key={item.path}
            to={item.path}
            onClick={onNavigate}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                collapsed && 'justify-center px-0',
                isActive
                  ? 'bg-sidebar-primary text-sidebar-primary-foreground'
                  : 'text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground',
              )
            }
          >
            <Icon className="size-5 shrink-0" />
            {!collapsed && <span className="truncate">{item.name}</span>}
          </NavLink>
        )
      })}
    </nav>
  )
}

function UserArea({
  collapsed,
  username,
  onLogout,
}: {
  collapsed: boolean
  username: string | null
  onLogout: () => void
}) {
  return (
    <div className="shrink-0 border-t border-sidebar-border p-2">
      <div
        className={cn(
          'mb-2 flex items-center gap-3 px-1',
          collapsed && 'justify-center px-0',
        )}
      >
        <div className="flex size-8 shrink-0 items-center justify-center rounded-full bg-primary text-sm font-medium text-primary-foreground">
          {username?.charAt(0).toUpperCase() || 'U'}
        </div>
        {!collapsed && (
          <span className="truncate text-sm text-sidebar-foreground">
            {username}
          </span>
        )}
      </div>
      <Button
        variant="ghost"
        onClick={onLogout}
        className={cn(
          'w-full justify-start gap-2 text-destructive hover:bg-sidebar-accent hover:text-destructive',
          collapsed && 'justify-center px-0',
        )}
      >
        <LogOut className="size-4" />
        {!collapsed && <span>退出</span>}
      </Button>
    </div>
  )
}

export default function AppLayout() {
  const username = useAuthStore((s) => s.username)
  const logout = useAuthStore((s) => s.logout)
  const [collapsed, setCollapsed] = useState(false)
  const [mobileOpen, setMobileOpen] = useState(false)

  // 挂载时若存在 token 则验证其有效性
  useEffect(() => {
    void useAuthStore.getState().validateToken()
  }, [])

  return (
    <div className="flex h-screen w-full overflow-hidden">
      {/* 侧边栏 - 桌面端 */}
      <aside
        className={cn(
          'relative hidden shrink-0 flex-col border-r border-sidebar-border bg-sidebar transition-[width] duration-300 lg:flex',
          collapsed ? 'w-16' : 'w-[200px]',
        )}
      >
        <Logo collapsed={collapsed} />
        <NavItems collapsed={collapsed} />
        <UserArea collapsed={collapsed} username={username} onLogout={logout} />

        {/* 折叠按钮 */}
        <button
          type="button"
          onClick={() => setCollapsed((c) => !c)}
          className="absolute top-1/2 -right-3 z-10 flex size-6 -translate-y-1/2 items-center justify-center rounded-full bg-primary text-primary-foreground"
          aria-label={collapsed ? '展开侧边栏' : '折叠侧边栏'}
        >
          {collapsed ? (
            <ChevronRight className="size-3" />
          ) : (
            <ChevronLeft className="size-3" />
          )}
        </button>
      </aside>

      <div className="flex flex-1 flex-col overflow-hidden">
        {/* 顶部栏 - 移动端 */}
        <header className="flex h-14 shrink-0 items-center gap-3 border-b border-sidebar-border bg-sidebar px-4 lg:hidden">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setMobileOpen(true)}
            className="text-sidebar-foreground hover:bg-sidebar-accent"
          >
            <Menu className="size-5" />
          </Button>
          <span className="text-lg font-semibold text-sidebar-foreground">
            Xin-api
          </span>
        </header>

        {/* 侧边栏 - 移动端抽屉 */}
        <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
          <SheetContent
            side="left"
            className="w-[220px] gap-0 bg-sidebar p-0"
            showCloseButton={false}
          >
            <Logo collapsed={false} />
            <NavItems
              collapsed={false}
              onNavigate={() => setMobileOpen(false)}
            />
            <UserArea
              collapsed={false}
              username={username}
              onLogout={logout}
            />
          </SheetContent>
        </Sheet>

        {/* 主内容区 */}
        <main className="flex-1 overflow-y-auto bg-muted/40 p-5">
          <Outlet />
        </main>
      </div>
    </div>
  )
}