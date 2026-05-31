<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import {
  Grid,
  Folder,
  Connection,
  Key,
  Setting,
  SwitchButton,
  Menu,
  ArrowRight,
  ArrowLeft,
} from '@element-plus/icons-vue'

const authStore = useAuthStore()
const route = useRoute()
const router = useRouter()

const isCollapse = ref(false)
const isMobile = ref(false)
const drawerVisible = ref(false)

// 检测是否为移动端
const checkMobile = () => {
  isMobile.value = window.innerWidth < 768
  if (isMobile.value) {
    isCollapse.value = true
  }
}

// 初始化检测
if (typeof window !== 'undefined') {
  checkMobile()
  window.addEventListener('resize', checkMobile)
}

const navigation = [
  { name: '仪表盘', path: '/dashboard', icon: Grid },
  { name: '项目组', path: '/groups', icon: Folder },
  { name: '渠道', path: '/channels', icon: Connection },
  { name: 'API Key', path: '/apikeys', icon: Key },
  { name: '账户设置', path: '/settings', icon: Setting },
]

// 精确匹配激活状态
const isActive = (path: string) => {
  if (path === '/dashboard') {
    return route.path === '/dashboard'
  }
  return route.path === path || route.path.startsWith(path + '/')
}

const activeMenu = computed(() => {
  const item = navigation.find(item => isActive(item.path))
  return item?.path || '/dashboard'
})

const handleSelect = (path: string) => {
  router.push(path)
  if (isMobile.value) {
    drawerVisible.value = false
  }
}

const handleLogout = () => {
  authStore.logout()
}

const toggleSidebar = () => {
  if (isMobile.value) {
    drawerVisible.value = !drawerVisible.value
  } else {
    isCollapse.value = !isCollapse.value
  }
}
</script>

<template>
  <div class="app-layout">
    <!-- 移动端顶部导航 -->
    <div v-if="isMobile" class="mobile-header">
      <el-button text @click="toggleSidebar">
        <el-icon :size="20"><Menu /></el-icon>
      </el-button>
      <span class="mobile-title">Xin-api</span>
      <div style="width: 40px"></div>
    </div>

    <!-- 侧边栏 - 桌面端 -->
    <el-aside
      v-if="!isMobile"
      class="sidebar"
      :width="isCollapse ? '64px' : '220px'"
    >
      <div class="sidebar-header">
        <div class="logo">
          <span class="logo-icon">X</span>
          <span v-if="!isCollapse" class="logo-text">Xin-api</span>
        </div>
      </div>

      <el-menu
        :default-active="activeMenu"
        :collapse="isCollapse"
        :collapse-transition="false"
        router
        class="sidebar-menu"
        background-color="#304156"
        text-color="#bfcbd9"
        active-text-color="#409EFF"
      >
        <el-menu-item
          v-for="item in navigation"
          :key="item.path"
          :index="item.path"
          @click="handleSelect(item.path)"
        >
          <el-icon>
            <component :is="item.icon" />
          </el-icon>
          <template #title>{{ item.name }}</template>
        </el-menu-item>
      </el-menu>

      <div class="sidebar-footer">
        <div class="user-info">
          <el-avatar :size="32" class="user-avatar">
            {{ authStore.username?.charAt(0).toUpperCase() || 'U' }}
          </el-avatar>
          <span v-if="!isCollapse" class="username">{{ authStore.username }}</span>
        </div>
        <el-button
          v-if="!isCollapse"
          text
          type="danger"
          class="logout-btn"
          @click="handleLogout"
        >
          <el-icon><SwitchButton /></el-icon>
          <span>退出</span>
        </el-button>
        <el-button
          v-else
          text
          type="danger"
          class="logout-btn-icon"
          @click="handleLogout"
        >
          <el-icon><SwitchButton /></el-icon>
        </el-button>
      </div>

      <!-- 折叠按钮 -->
      <div class="collapse-btn" @click="isCollapse = !isCollapse">
        <el-icon v-if="isCollapse"><ArrowRight /></el-icon>
        <el-icon v-else><ArrowLeft /></el-icon>
      </div>
    </el-aside>

    <!-- 侧边栏 - 移动端抽屉 -->
    <el-drawer
      v-model="drawerVisible"
      direction="ltr"
      :with-header="false"
      size="220px"
      class="mobile-drawer"
    >
      <div class="sidebar-header">
        <div class="logo">
          <span class="logo-icon">X</span>
          <span class="logo-text">Xin-api</span>
        </div>
      </div>

      <el-menu
        :default-active="activeMenu"
        router
        class="sidebar-menu"
        background-color="#304156"
        text-color="#bfcbd9"
        active-text-color="#409EFF"
      >
        <el-menu-item
          v-for="item in navigation"
          :key="item.path"
          :index="item.path"
          @click="handleSelect(item.path)"
        >
          <el-icon>
            <component :is="item.icon" />
          </el-icon>
          <template #title>{{ item.name }}</template>
        </el-menu-item>
      </el-menu>

      <div class="sidebar-footer">
        <div class="user-info">
          <el-avatar :size="32" class="user-avatar">
            {{ authStore.username?.charAt(0).toUpperCase() || 'U' }}
          </el-avatar>
          <span class="username">{{ authStore.username }}</span>
        </div>
        <el-button text type="danger" class="logout-btn" @click="handleLogout">
          <el-icon><SwitchButton /></el-icon>
          <span>退出</span>
        </el-button>
      </div>
    </el-drawer>

    <!-- 主内容区 -->
    <el-container class="main-container">
      <el-main class="main-content">
        <router-view />
      </el-main>
    </el-container>
  </div>
</template>

<style scoped>
.app-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

.sidebar {
  background-color: #304156;
  display: flex;
  flex-direction: column;
  position: relative;
  transition: width 0.3s;
}

.sidebar-header {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-bottom: 1px solid #1f2d3d;
}

.logo {
  display: flex;
  align-items: center;
  gap: 12px;
}

.logo-icon {
  width: 36px;
  height: 36px;
  background: #409eff;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: bold;
  font-size: 18px;
}

.logo-text {
  color: white;
  font-size: 18px;
  font-weight: 600;
}

.sidebar-menu {
  flex: 1;
  border-right: none;
}

.sidebar-footer {
  padding: 16px;
  border-top: 1px solid #1f2d3d;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.user-avatar {
  background: #409eff;
  color: white;
  font-weight: 500;
}

.username {
  color: #bfcbd9;
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.logout-btn {
  width: 100%;
  justify-content: flex-start;
  gap: 8px;
  color: #f56c6c;
}

.logout-btn-icon {
  width: 100%;
  justify-content: center;
  color: #f56c6c;
}

.collapse-btn {
  position: absolute;
  right: -12px;
  top: 50%;
  transform: translateY(-50%);
  width: 24px;
  height: 24px;
  background: #409eff;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: white;
  font-size: 12px;
  z-index: 10;
}

.main-container {
  flex: 1;
  background: #f0f2f5;
  overflow: hidden;
}

.main-content {
  padding: 20px;
  overflow-y: auto;
}

/* 移动端样式 */
.mobile-header {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 56px;
  background: #304156;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  z-index: 100;
}

.mobile-header :deep(.el-button) {
  color: #bfcbd9;
}

.mobile-title {
  color: white;
  font-size: 18px;
  font-weight: 600;
}

.mobile-drawer :deep(.el-drawer__body) {
  padding: 0;
  background: #304156;
  display: flex;
  flex-direction: column;
}

/* 适配移动端主内容区 */
@media (max-width: 768px) {
  .main-container {
    padding-top: 56px;
  }

  .main-content {
    padding: 16px;
  }
}
</style>
