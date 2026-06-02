<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useGroupStore } from '@/stores/groups'
import { useChannelStore } from '@/stores/channels'
import { useApiKeyStore } from '@/stores/apikeys'
import {
  Folder,
  Connection,
  Key,
  ArrowRight,
} from '@element-plus/icons-vue'

const router = useRouter()
const authStore = useAuthStore()
const groupStore = useGroupStore()
const channelStore = useChannelStore()
const apiKeyStore = useApiKeyStore()

const loading = ref(true)

// 统计数据
const totalGroups = ref(0)
const totalChannels = ref(0)
const totalApiKeys = ref(0)

onMounted(async () => {
  try {
    // 1. 先获取所有项目组
    const groups = await groupStore.fetchAll()
    totalGroups.value = groups?.length ?? 0

    // 2. 获取所有渠道（不传 group_id 获取全部）
    try {
      const channels = await channelStore.fetchByGroup()
      totalChannels.value = channels?.length ?? 0
    } catch {
      // 如果失败，尝试逐个获取
      if (groups && groups.length > 0) {
        let count = 0
        for (const g of groups) {
          try {
            const data = await channelStore.fetchByGroup(g.id)
            count += data?.length ?? 0
          } catch { /* ignore */ }
        }
        totalChannels.value = count
      }
    }

    // 3. 获取所有 API Keys（不传 group_id 获取全部）
    try {
      const keys = await apiKeyStore.fetchAll()
      totalApiKeys.value = keys?.length ?? 0
    } catch {
      // 如果失败，尝试逐个获取
      if (groups && groups.length > 0) {
        let count = 0
        for (const g of groups) {
          try {
            const data = await apiKeyStore.fetchAll(g.id)
            count += data?.length ?? 0
          } catch { /* ignore */ }
        }
        totalApiKeys.value = count
      }
    }
  } catch {
    // silently fail
  } finally {
    loading.value = false
  }
})

// 使用计算属性确保响应式更新
const statsCards = computed(() => [
  {
    title: '项目组',
    value: totalGroups.value,
    icon: Folder,
    path: '/groups',
    color: '#409EFF',
    bg: '#ecf5ff',
  },
  {
    title: '渠道',
    value: totalChannels.value,
    icon: Connection,
    path: '/channels',
    color: '#67C23A',
    bg: '#f0f9eb',
  },
  {
    title: 'API Key',
    value: totalApiKeys.value,
    icon: Key,
    path: '/apikeys',
    color: '#E6A23C',
    bg: '#fdf6ec',
  },
])

const quickActions = [
  { label: '创建项目组', path: '/groups' },
  { label: '查看渠道', path: '/channels' },
  { label: '创建 API Key', path: '/apikeys' },
]
</script>

<template>
  <div style="max-width: 1200px;">
    <!-- 欢迎区域 -->
    <div style="margin-bottom: 24px;">
      <h1 style="font-size: 22px; font-weight: 600; color: #303133; margin-bottom: 8px;">
        欢迎回来，{{ authStore.username || '管理员' }}
      </h1>
      <p style="font-size: 14px; color: #909399;">
        这是 Xin-api 网关控制面板，在这里管理你的项目组、渠道和 API Key。
      </p>
    </div>

    <!-- 统计卡片 -->
    <div style="display: flex; gap: 20px; margin-bottom: 24px;">
      <el-card
        v-for="card in statsCards"
        :key="card.title"
        style="flex: 1; cursor:pointer;"
        shadow="hover"
        @click="router.push(card.path)"
      >
        <el-skeleton :rows="2" animated v-if="loading" />
        <div v-else style="display: flex; justify-content: space-between; align-items: center;">
          <div style="flex: 1;">
            <p style="font-size: 14px; color: #909399; margin-bottom: 8px;">{{ card.title }}</p>
            <p style="font-size: 32px; font-weight: 600; line-height: 1;" :style="{ color: card.color }">
              {{ card.value }}
            </p>
          </div>
          <div
            style="width: 56px; height: 56px; border-radius: 12px; display: flex; align-items: center; justify-content: center;"
            :style="{ background: card.bg, color: card.color }"
          >
            <el-icon :size="24">
              <component :is="card.icon" />
            </el-icon>
          </div>
        </div>
      </el-card>
    </div>

    <!-- 快速操作 -->
    <el-card shadow="never">
      <template #header>
        <div style="display: flex; align-items: center; gap: 12px;">
          <span style="font-size: 16px; font-weight: 600; color: #303133;">快速操作</span>
          <span style="font-size: 13px; color: #909399; font-weight: normal;">常用管理入口</span>
        </div>
      </template>
      <div style="display: flex; gap: 12px; flex-wrap: wrap;">
        <el-button
          v-for="action in quickActions"
          :key="action.path"
          @click="router.push(action.path)"
        >
          {{ action.label }}
          <el-icon style="font-size: 12px;"><ArrowRight /></el-icon>
        </el-button>
      </div>
    </el-card>
  </div>
</template>
