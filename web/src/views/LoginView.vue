<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardHeader, CardTitle, CardContent, CardFooter } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { toast } from 'vue-sonner'

const router = useRouter()
const authStore = useAuthStore()

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

// 如果已登录，自动跳转到仪表盘
onMounted(() => {
  if (authStore.isAuthenticated) {
    router.replace('/dashboard')
  }
})

const handleLogin = async () => {
  error.value = ''

  const currentUsername = username.value.trim()
  const currentPassword = password.value.trim()

  // 校验逻辑
  if (!currentUsername) {
    error.value = '请输入用户名'
    toast.warning('请输入用户名')
    return
  }
  if (currentUsername.length < 5 || currentUsername.length > 10) {
    error.value = '用户名长度需在 5-10 个字符之间'
    toast.warning('用户名长度需在 5-10 个字符之间')
    return
  }

  if (!currentPassword) {
    error.value = '请输入密码'
    toast.warning('请输入密码')
    return
  }
  if (currentPassword.length < 8 || currentPassword.length > 20) {
    error.value = '密码长度需在 8-20 个字符之间'
    toast.warning('密码长度需在 8-20 个字符之间')
    return
  }

  loading.value = true
  try {
    await authStore.login(currentUsername, currentPassword)
    // 登录成功，直接跳转，不显示提示
    router.replace('/dashboard')
  } catch (e: any) {
    error.value = e.message || '登录失败，请重试'
    toast.error(error.value)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-muted/30 p-4">
    <Card class="w-full max-w-sm">
      <CardHeader class="text-center">
        <div class="mx-auto mb-2 flex h-12 w-12 items-center justify-center rounded-xl bg-primary text-lg font-bold text-primary-foreground">
          X
        </div>
        <CardTitle class="text-xl">Xin-api 控制面板</CardTitle>
      </CardHeader>

      <form @submit.prevent="handleLogin">
        <CardContent class="space-y-4">
          <div class="space-y-2">
            <Label for="username">用户名</Label>
            <Input
              id="username"
              v-model="username"
              placeholder="请输入用户名"
              :disabled="loading"
              autocomplete="username"
            />
          </div>
          <div class="space-y-2">
            <Label for="password">密码</Label>
            <Input
              id="password"
              v-model="password"
              type="password"
              placeholder="请输入密码"
              :disabled="loading"
              autocomplete="current-password"
            />
          </div>
          <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
        </CardContent>

        <CardFooter>
          <Button class="w-full" type="submit" :disabled="loading">
            {{ loading ? '登录中...' : '登录' }}
          </Button>
        </CardFooter>
      </form>
    </Card>
  </div>
</template>
