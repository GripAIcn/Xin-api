<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import { useRoute, useRouter } from 'vue-router'
import { cn } from '@/lib/utils'
import { computed } from 'vue'
import {
  LayoutDashboard,
  FolderKanban,
  Cable,
  Key,
  Settings,
  LogOut,
  ChevronLeft,
  Menu,
} from 'lucide-vue-next'

const authStore = useAuthStore()
const route = useRoute()

const collapsed = defineModel<boolean>('collapsed', { default: false })

const navigation = [
  { name: '仪表盘', path: '/dashboard', icon: LayoutDashboard },
  { name: '项目组', path: '/groups', icon: FolderKanban },
  { name: '渠道', path: '/channels', icon: Cable },
  { name: 'API Key', path: '/apikeys', icon: Key },
  { name: '账户设置', path: '/settings', icon: Settings },
]

// 精确匹配激活状态
const isActive = (path: string) => {
  // 仪表盘需要精确匹配
  if (path === '/dashboard') {
    return route.path === '/dashboard'
  }
  // 其他路由检查是否以该路径开头
  return route.path === path || route.path.startsWith(path + '/')
}

const handleLogout = () => {
  authStore.logout()
}
</script>

<template>
  <div class="flex h-screen overflow-hidden bg-background">
    <!-- Mobile Overlay -->
    <div
      v-if="!collapsed"
      class="fixed inset-0 z-40 bg-black/50 md:hidden"
      @click="collapsed = true"
    />

    <!-- Sidebar -->
    <aside
      :class="cn(
        'fixed inset-y-0 left-0 z-50 flex flex-col border-r bg-card transition-all duration-300 md:static',
        collapsed ? '-translate-x-full md:translate-x-0 md:w-16' : 'translate-x-0 w-64'
      )"
    >
      <!-- Logo -->
      <div class="flex h-14 items-center border-b px-4">
        <router-link
          v-if="!collapsed"
          to="/dashboard"
          class="flex items-center gap-2 overflow-hidden"
        >
          <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-sm font-bold text-primary-foreground">
            X
          </div>
          <span class="text-base font-semibold whitespace-nowrap">Xin-api</span>
        </router-link>
        <router-link
          v-else
          to="/dashboard"
          class="flex w-full justify-center"
        >
          <div class="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-sm font-bold text-primary-foreground">
            X
          </div>
        </router-link>
      </div>

      <!-- Navigation -->
      <nav class="flex-1 space-y-1 p-3">
        <router-link
          v-for="item in navigation"
          :key="item.path"
          :to="item.path"
          :class="cn(
            'flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors',
            isActive(item.path)
              ? 'bg-primary/10 text-primary'
              : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
          )"
        >
          <component :is="item.icon" class="h-4 w-4 shrink-0" />
          <span v-if="!collapsed" class="truncate">{{ item.name }}</span>
        </router-link>
      </nav>

      <!-- User & Logout -->
      <div class="border-t p-3">
        <div class="flex items-center gap-3">
          <div class="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10 text-xs font-medium text-primary shrink-0">
            {{ authStore.username?.charAt(0).toUpperCase() || 'U' }}
          </div>
          <div v-if="!collapsed" class="flex-1 min-w-0">
            <p class="text-sm font-medium truncate">{{ authStore.username }}</p>
          </div>
          <button
            @click="handleLogout"
            class="flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors shrink-0"
            :title="'退出登录'"
          >
            <LogOut class="h-4 w-4" />
          </button>
        </div>
      </div>

      <!-- Collapse Toggle (desktop) -->
      <button
        @click="collapsed = !collapsed"
        class="hidden md:flex absolute -right-3 top-20 h-6 w-6 items-center justify-center rounded-full border bg-background shadow-sm"
      >
        <ChevronLeft :class="cn('h-3 w-3 transition-transform', collapsed && 'rotate-180')" />
      </button>
    </aside>

    <!-- Main Content -->
    <main class="flex-1 overflow-auto">
      <!-- Top Bar (mobile) -->
      <div class="flex h-14 items-center gap-4 border-b px-4 md:hidden">
        <button @click="collapsed = false" class="text-muted-foreground">
          <Menu class="h-5 w-5" />
        </button>
        <span class="font-semibold">Xin-api</span>
      </div>

      <!-- Page Content -->
      <div class="p-6">
        <RouterView />
      </div>
    </main>
  </div>
</template>
