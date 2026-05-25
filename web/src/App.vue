<script setup lang="ts">
import { RouterView } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { Sonner } from '@/components/ui/sonner'
import { computed } from 'vue'

const authStore = useAuthStore()

// 检查认证状态是否已确定（用于避免闪烁）
const isAuthReady = computed(() => {
  // 如果有 token 或明确未登录，都认为状态已确定
  return authStore.isAuthenticated || !localStorage.getItem('token')
})
</script>

<template>
  <Sonner position="top-right" rich-colors />

  <!-- 认证状态确定后显示内容 -->
  <RouterView v-if="isAuthReady" />

  <!-- 加载状态 -->
  <div
    v-else
    class="flex h-screen w-full items-center justify-center bg-background"
  >
    <div class="flex flex-col items-center gap-4">
      <div class="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent"></div>
      <p class="text-sm text-muted-foreground">加载中...</p>
    </div>
  </div>
</template>
