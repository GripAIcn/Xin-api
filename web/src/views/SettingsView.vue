<script setup lang="ts">
import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from '@/components/ui/card'
import { toast } from 'vue-sonner'
import ErrorAlertDialog from '@/components/ErrorAlertDialog.vue'

const authStore = useAuthStore()

// 错误弹窗状态
const errorDialog = ref({ open: false, title: '提示', message: '' })
const showError = (message: string, title = '提示') => {
  errorDialog.value = { open: true, title, message }
}

// Change Username
const username = ref(authStore.username || '')
const usernameLoading = ref(false)

const validateUsername = () => {
  if (!username.value || username.value.length < 5 || username.value.length > 10) {
    return '用户名长度需在 5-10 个字符之间'
  }
  return null
}

const handleUpdateUsername = async () => {
  const error = validateUsername()
  if (error) {
    showError(error)
    return
  }
  usernameLoading.value = true
  try {
    await authStore.updateAccount({ username: username.value })
    toast.success('用户名修改成功')
  } catch (e: any) {
    showError(e.message || '修改失败')
  } finally {
    usernameLoading.value = false
  }
}

// Change Password
const oldPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const passwordLoading = ref(false)

const validatePassword = () => {
  if (!oldPassword.value || oldPassword.value.length < 8 || oldPassword.value.length > 20) {
    return '旧密码长度需在 8-20 个字符之间'
  }
  if (!newPassword.value || newPassword.value.length < 8 || newPassword.value.length > 20) {
    return '新密码长度需在 8-20 个字符之间'
  }
  if (newPassword.value !== confirmPassword.value) {
    return '两次输入的新密码不一致'
  }
  return null
}

const handleUpdatePassword = async () => {
  const error = validatePassword()
  if (error) {
    showError(error)
    return
  }

  passwordLoading.value = true
  try {
    await authStore.updatePassword({ old_password: oldPassword.value, new_password: newPassword.value })
    toast.success('密码修改成功')
    oldPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
  } catch (e: any) {
    showError(e.message || '修改失败')
  } finally {
    passwordLoading.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-2xl font-bold tracking-tight">账户设置</h1>
      <p class="text-muted-foreground">管理你的账户信息和登录密码</p>
    </div>

    <!-- Change Username -->
    <Card>
      <CardHeader>
        <CardTitle class="text-base">修改用户名</CardTitle>
        <CardDescription>当前用户名：{{ authStore.username }}</CardDescription>
      </CardHeader>
      <form @submit.prevent="handleUpdateUsername">
        <CardContent class="space-y-3">
          <div class="space-y-2">
            <Label for="new-username">新用户名</Label>
            <Input id="new-username" v-model="username" placeholder="输入新用户名" :maxlength="10" :disabled="usernameLoading" />
          </div>
        </CardContent>
        <CardFooter>
          <Button type="submit" :disabled="usernameLoading">
            {{ usernameLoading ? '保存中...' : '保存修改' }}
          </Button>
        </CardFooter>
      </form>
    </Card>

    <!-- Change Password -->
    <Card>
      <CardHeader>
        <CardTitle class="text-base">修改密码</CardTitle>
        <CardDescription>密码长度需在 8-20 个字符之间</CardDescription>
      </CardHeader>
      <form @submit.prevent="handleUpdatePassword">
        <CardContent class="space-y-3">
          <div class="space-y-2">
            <Label for="old-password">旧密码</Label>
            <Input id="old-password" v-model="oldPassword" type="password" placeholder="输入旧密码" :maxlength="20" :disabled="passwordLoading" />
          </div>
          <div class="space-y-2">
            <Label for="new-password">新密码</Label>
            <Input id="new-password" v-model="newPassword" type="password" placeholder="输入新密码" :maxlength="20" :disabled="passwordLoading" />
          </div>
          <div class="space-y-2">
            <Label for="confirm-password">确认新密码</Label>
            <Input id="confirm-password" v-model="confirmPassword" type="password" placeholder="再次输入新密码" :maxlength="20" :disabled="passwordLoading" />
          </div>
        </CardContent>
        <CardFooter>
          <Button type="submit" :disabled="passwordLoading">
            {{ passwordLoading ? '保存中...' : '修改密码' }}
          </Button>
        </CardFooter>
      </form>
    </Card>

    <!-- Error Alert Dialog -->
    <ErrorAlertDialog v-model:open="errorDialog.open" :title="errorDialog.title" :message="errorDialog.message" />
  </div>
</template>
