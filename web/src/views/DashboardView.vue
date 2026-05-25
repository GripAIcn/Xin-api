<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useGroupStore } from '@/stores/groups'
import { useChannelStore } from '@/stores/channels'
import { useApiKeyStore } from '@/stores/apikeys'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import { FolderKanban, Cable, Key, ArrowRight } from 'lucide-vue-next'

const router = useRouter()
const authStore = useAuthStore()
const groupStore = useGroupStore()
const channelStore = useChannelStore()
const apiKeyStore = useApiKeyStore()

const loading = ref(true)

onMounted(async () => {
  try {
    await Promise.all([
      groupStore.fetchAll(),
      channelStore.fetchByGroup(),
      apiKeyStore.fetchAll(),
    ])
  } catch {
    // silently fail
  } finally {
    loading.value = false
  }
})

// 使用计算属性确保响应式更新，添加空数组保护
const statsCards = computed(() => [
  {
    title: '项目组',
    value: groupStore.groups?.length ?? 0,
    icon: FolderKanban,
    path: '/groups',
    color: 'text-blue-600',
    bg: 'bg-blue-100',
  },
  {
    title: '渠道',
    value: channelStore.channels?.length ?? 0,
    icon: Cable,
    path: '/channels',
    color: 'text-emerald-600',
    bg: 'bg-emerald-100',
  },
  {
    title: 'API Key',
    value: apiKeyStore.apiKeys?.length ?? 0,
    icon: Key,
    path: '/apikeys',
    color: 'text-purple-600',
    bg: 'bg-purple-100',
  },
])

const quickActions = [
  { label: '创建项目组', path: '/groups' },
  { label: '查看渠道', path: '/channels' },
  { label: '创建 API Key', path: '/apikeys' },
]
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-2xl font-bold tracking-tight">
        欢迎回来，{{ authStore.username || '管理员' }}
      </h1>
      <p class="text-muted-foreground">这是 Xin-api 网关控制面板，在这里管理你的项目组、渠道和 API Key。</p>
    </div>

    <!-- Stats -->
    <div class="grid gap-4 sm:grid-cols-3">
      <div v-for="card in statsCards" :key="card.title">
        <Card
          class="cursor-pointer transition-colors hover:bg-accent/50"
          @click="router.push(card.path)"
        >
          <CardContent class="p-6">
            <div v-if="loading">
              <Skeleton class="h-4 w-20 mb-2" />
              <Skeleton class="h-8 w-12" />
            </div>
            <template v-else>
              <div class="flex items-center justify-between">
                <div>
                  <p class="text-sm text-muted-foreground">{{ card.title }}</p>
                  <p class="text-2xl font-bold mt-1">{{ card.value }}</p>
                </div>
                <div :class="cn('flex h-10 w-10 items-center justify-center rounded-full', card.bg)">
                  <component :is="card.icon" :class="cn('h-5 w-5', card.color)" />
                </div>
              </div>
            </template>
          </CardContent>
        </Card>
      </div>
    </div>

    <!-- Quick Actions -->
    <Card>
      <CardHeader>
        <CardTitle class="text-base">快速操作</CardTitle>
        <CardDescription>常用管理入口</CardDescription>
      </CardHeader>
      <CardContent>
        <div class="flex flex-wrap gap-3">
          <div
            v-for="action in quickActions"
            :key="action.path"
            @click="router.push(action.path)"
            class="flex cursor-pointer items-center gap-2 rounded-lg border bg-card px-4 py-2 text-sm font-medium transition-colors hover:bg-accent"
          >
            {{ action.label }}
            <ArrowRight class="h-3.5 w-3.5" />
          </div>
        </div>
      </CardContent>
    </Card>
  </div>
</template>
