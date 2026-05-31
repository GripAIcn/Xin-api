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
    icon: Folder,
    path: '/groups',
    color: '#409EFF',
    bg: '#ecf5ff',
  },
  {
    title: '渠道',
    value: channelStore.channels?.length ?? 0,
    icon: Connection,
    path: '/channels',
    color: '#67C23A',
    bg: '#f0f9eb',
  },
  {
    title: 'API Key',
    value: apiKeyStore.apiKeys?.length ?? 0,
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
  <div class="dashboard">
    <!-- 欢迎区域 -->
    <div class="welcome-section">
      <h1 class="welcome-title">
        欢迎回来，{{ authStore.username || '管理员' }}
      </h1>
      <p class="welcome-desc">
        这是 Xin-api 网关控制面板，在这里管理你的项目组、渠道和 API Key。
      </p>
    </div>

    <!-- 统计卡片 -->
    <el-row :gutter="20" class="stats-row">
      <el-col :xs="24" :sm="8" v-for="card in statsCards" :key="card.title">
        <el-card
          class="stat-card"
          shadow="hover"
          @click="router.push(card.path)"
        >
          <el-skeleton :rows="2" animated v-if="loading" />
          <div v-else class="stat-content">
            <div class="stat-info">
              <p class="stat-title">{{ card.title }}</p>
              <p class="stat-value" :style="{ color: card.color }">
                {{ card.value }}
              </p>
            </div>
            <div
              class="stat-icon"
              :style="{ background: card.bg, color: card.color }"
            >
              <el-icon :size="24">
                <component :is="card.icon" />
              </el-icon>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 快速操作 -->
    <el-card class="quick-actions" shadow="never">
      <template #header>
        <div class="card-header">
          <span>快速操作</span>
          <span class="header-desc">常用管理入口</span>
        </div>
      </template>
      <div class="actions-list">
        <el-button
          v-for="action in quickActions"
          :key="action.path"
          class="action-btn"
          @click="router.push(action.path)"
        >
          {{ action.label }}
          <el-icon class="action-icon"><ArrowRight /></el-icon>
        </el-button>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.dashboard {
  max-width: 1200px;
}

.welcome-section {
  margin-bottom: 24px;
}

.welcome-title {
  font-size: 24px;
  font-weight: 600;
  color: #303133;
  margin: 0 0 8px 0;
}

.welcome-desc {
  font-size: 14px;
  color: #909399;
  margin: 0;
}

.stats-row {
  margin-bottom: 24px;
}

.stat-card {
  cursor: pointer;
  transition: transform 0.2s;
  margin-bottom: 20px;
}

.stat-card:hover {
  transform: translateY(-2px);
}

.stat-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.stat-info {
  flex: 1;
}

.stat-title {
  font-size: 14px;
  color: #909399;
  margin: 0 0 8px 0;
}

.stat-value {
  font-size: 32px;
  font-weight: 600;
  margin: 0;
  line-height: 1;
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.quick-actions {
  margin-top: 8px;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.card-header span:first-child {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.header-desc {
  font-size: 13px;
  color: #909399;
  font-weight: normal;
}

.actions-list {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 8px;
}

.action-icon {
  font-size: 12px;
}
</style>
