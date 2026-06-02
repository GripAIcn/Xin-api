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
  <div style="display: flex; height: 100vh; overflow: hidden; width: 100%;">
    <!-- 移动端顶部导航 -->
    <div v-if="isMobile" style="position: fixed; top: 0; left: 0; right: 0; height: 56px; display: flex; align-items: center; justify-content: space-between; padding: 0 16px; z-index: 100; background: #304156;">
      <el-button text @click="toggleSidebar" style="color: #bfcbd9;">
        <el-icon :size="20"><Menu /></el-icon>
      </el-button>
      <span style="color: white; font-size: 18px; font-weight: 600;">Xin-api</span>
      <div style="width: 40px;"></div>
    </div>

    <!-- 侧边栏 - 桌面端 -->
    <aside
      v-if="!isMobile"
      style="display: flex; flex-direction: column; position: relative; transition: all 0.3s; overflow: hidden; flex-shrink: 0; background: #304156; height: 100vh;"
      :style="{ width: isCollapse ? '64px' : '200px' }"
    >
      <div style="height: 60px; display: flex; align-items: center; justify-content: center; border-bottom: 1px solid #1f2d3d;">
        <div style="display: flex; align-items: center; gap: 12px;">
          <div style="width: 36px; height: 36px; border-radius: 8px; display: flex; align-items: center; justify-content: center; color: white; font-weight: bold; font-size: 18px; background: #409eff;">
            X
          </div>
          <span v-if="!isCollapse" style="color: white; font-size: 18px; font-weight: 600;">Xin-api</span>
        </div>
      </div>

      <el-menu
        :default-active="activeMenu"
        :collapse="isCollapse"
        :collapse-transition="false"
        router
        style="flex: 1; border-right: none; overflow: hidden; background: #304156;"
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

      <div style="padding: 12px 16px; flex-shrink: 0; border-top: 1px solid #1f2d3d;">
        <div style="display: flex; align-items: center; gap: 10px; margin-bottom: 8px;">
          <el-avatar :size="32" style="background: #409eff; color: white; font-weight: 500;">
            {{ authStore.username?.charAt(0).toUpperCase() || 'U' }}
          </el-avatar>
          <span v-if="!isCollapse" style="font-size: 14px; color: #bfcbd9; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">{{ authStore.username }}</span>
        </div>
        <el-button
          v-if="!isCollapse"
          text
          type="danger"
          style="width: 100%; justify-content: start; gap: 6px; color: #f56c6c;"
          @click="handleLogout"
        >
          <el-icon><SwitchButton /></el-icon>
          <span>退出</span>
        </el-button>
        <el-button
          v-else
          text
          type="danger"
          style="width: 100%; justify-content: center; color: #f56c6c;"
          @click="handleLogout"
        >
          <el-icon><SwitchButton /></el-icon>
        </el-button>
      </div>

      <!-- 折叠按钮 -->
      <div 
        style="position: absolute; right: -12px; top: 50%; transform: translateY(-50%); width: 24px; height: 24px; border-radius: 50%; display: flex; align-items: center; justify-content: center; cursor: pointer; color: white; font-size: 12px; z-index: 10; background: #409eff;"
        @click="isCollapse = !isCollapse"
      >
        <el-icon v-if="isCollapse"><ArrowRight /></el-icon>
        <el-icon v-else><ArrowLeft /></el-icon>
      </div>
    </aside>

    <!-- 侧边栏 - 移动端抽屉 -->
    <el-drawer
      v-model="drawerVisible"
      direction="ltr"
      :with-header="false"
      size="220px"
      class="mobile-drawer"
    >
      <div style="height: 60px; display: flex; align-items: center; justify-content: center; border-bottom: 1px solid #1f2d3d;">
        <div style="display: flex; align-items: center; gap: 12px;">
          <div style="width: 36px; height: 36px; border-radius: 8px; display: flex; align-items: center; justify-content: center; color: white; font-weight: bold; font-size: 18px; background: #409eff;">
            X
          </div>
          <span style="color: white; font-size: 18px; font-weight: 600;">Xin-api</span>
        </div>
      </div>

      <el-menu
        :default-active="activeMenu"
        router
        style="flex: 1; border-right: none; background: #304156;"
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

      <div style="padding: 12px 16px; border-top: 1px solid #1f2d3d;">
        <div style="display: flex; align-items: center; gap: 10px; margin-bottom: 8px;">
          <el-avatar :size="32" style="background: #409eff; color: white; font-weight: 500;">
            {{ authStore.username?.charAt(0).toUpperCase() || 'U' }}
          </el-avatar>
          <span style="font-size: 14px; color: #bfcbd9;">{{ authStore.username }}</span>
        </div>
        <el-button text type="danger" style="width: 100%; justify-content: start; gap: 6px; color: #f56c6c;" @click="handleLogout">
          <el-icon><SwitchButton /></el-icon>
          <span>退出</span>
        </el-button>
      </div>
    </el-drawer>

    <!-- 主内容区 -->
    <main style="flex: 1; overflow: hidden; background: #f0f2f5; display: flex; flex-direction: column;">
      <div 
        style="flex: 1; overflow-y: auto; padding: 20px;" 
        :style="{ paddingTop: isMobile ? '72px' : '20px' }"
      >
        <router-view />
      </div>
    </main>
  </div>
</template>

<style scoped>
/* 移动端抽屉样式 */
:deep(.mobile-drawer .el-drawer__body) {
  padding: 0;
  background: #304156;
  display: flex;
  flex-direction: column;
}
</style>
