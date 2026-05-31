<script setup lang="ts">
import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { ElMessage } from 'element-plus'
import { User, Lock } from '@element-plus/icons-vue'

const authStore = useAuthStore()

// 修改用户名
const username = ref(authStore.username || '')
const usernameLoading = ref(false)
const usernameFormRef = ref()

const usernameRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 5, max: 10, message: '用户名长度需在 5-10 个字符之间', trigger: 'blur' }
  ]
}

const handleUpdateUsername = async () => {
  const valid = await usernameFormRef.value?.validate().catch(() => false)
  if (!valid) return

  usernameLoading.value = true
  try {
    await authStore.updateAccount({ username: username.value })
    ElMessage.success('用户名修改成功')
  } catch (e: any) {
    ElMessage.error(e.message || '修改失败')
  } finally {
    usernameLoading.value = false
  }
}

// 修改密码
const passwordForm = ref({
  old_password: '',
  new_password: '',
  confirm_password: ''
})
const passwordLoading = ref(false)
const passwordFormRef = ref()

const validateConfirmPassword = (rule: any, value: string, callback: any) => {
  if (value !== passwordForm.value.new_password) {
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}

const passwordRules = {
  old_password: [
    { required: true, message: '请输入旧密码', trigger: 'blur' },
    { min: 8, max: 20, message: '密码长度需在 8-20 个字符之间', trigger: 'blur' }
  ],
  new_password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 8, max: 20, message: '密码长度需在 8-20 个字符之间', trigger: 'blur' }
  ],
  confirm_password: [
    { required: true, message: '请确认新密码', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ]
}

const handleUpdatePassword = async () => {
  const valid = await passwordFormRef.value?.validate().catch(() => false)
  if (!valid) return

  passwordLoading.value = true
  try {
    await authStore.updatePassword({
      old_password: passwordForm.value.old_password,
      new_password: passwordForm.value.new_password
    })
    ElMessage.success('密码修改成功')
    passwordForm.value = { old_password: '', new_password: '', confirm_password: '' }
  } catch (e: any) {
    ElMessage.error(e.message || '修改失败')
  } finally {
    passwordLoading.value = false
  }
}
</script>

<template>
  <div class="settings-page">
    <!-- 页面标题 -->
    <div class="page-header">
      <h2 class="page-title">账户设置</h2>
      <p class="page-desc">管理你的账户信息和登录密码</p>
    </div>

    <!-- 两列布局 -->
    <el-row :gutter="24">
      <!-- 修改用户名 -->
      <el-col :xs="24" :md="12">
        <el-card shadow="hover" class="setting-card">
          <template #header>
            <div class="card-header">
              <el-icon :size="18" class="header-icon"><User /></el-icon>
              <div class="header-title">
                <span class="title-text">修改用户名</span>
                <span class="header-desc">当前：{{ authStore.username }}</span>
              </div>
            </div>
          </template>
          <el-form
            ref="usernameFormRef"
            :model="{ username }"
            :rules="usernameRules"
            label-position="top"
          >
            <el-form-item label="新用户名" prop="username">
              <el-input
                v-model="username"
                placeholder="输入新用户名"
                maxlength="10"
                show-word-limit
                size="large"
              />
            </el-form-item>
            <el-form-item>
              <el-button
                type="primary"
                size="large"
                :loading="usernameLoading"
                @click="handleUpdateUsername"
                class="submit-btn"
              >
                {{ usernameLoading ? '保存中...' : '保存修改' }}
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <!-- 修改密码 -->
      <el-col :xs="24" :md="12">
        <el-card shadow="hover" class="setting-card">
          <template #header>
            <div class="card-header">
              <el-icon :size="18" class="header-icon"><Lock /></el-icon>
              <div class="header-title">
                <span class="title-text">修改密码</span>
                <span class="header-desc">密码长度 8-20 位</span>
              </div>
            </div>
          </template>
          <el-form
            ref="passwordFormRef"
            :model="passwordForm"
            :rules="passwordRules"
            label-position="top"
          >
            <el-form-item label="旧密码" prop="old_password">
              <el-input
                v-model="passwordForm.old_password"
                type="password"
                placeholder="输入旧密码"
                show-password
                size="large"
              />
            </el-form-item>
            <el-form-item label="新密码" prop="new_password">
              <el-input
                v-model="passwordForm.new_password"
                type="password"
                placeholder="输入新密码"
                show-password
                size="large"
              />
            </el-form-item>
            <el-form-item label="确认密码" prop="confirm_password">
              <el-input
                v-model="passwordForm.confirm_password"
                type="password"
                placeholder="再次输入新密码"
                show-password
                size="large"
              />
            </el-form-item>
            <el-form-item>
              <el-button
                type="primary"
                size="large"
                :loading="passwordLoading"
                @click="handleUpdatePassword"
                class="submit-btn"
              >
                {{ passwordLoading ? '保存中...' : '修改密码' }}
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<style scoped>
.settings-page {
  max-width: 100%;
}

.page-header {
  margin-bottom: 24px;
}

.page-title {
  font-size: 22px;
  font-weight: 600;
  color: #303133;
  margin: 0 0 8px 0;
}

.page-desc {
  font-size: 14px;
  color: #909399;
  margin: 0;
}

.setting-card {
  height: 100%;
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-icon {
  color: #409eff;
  background: #ecf5ff;
  padding: 8px;
  border-radius: 8px;
}

.header-title {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.title-text {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.header-desc {
  font-size: 13px;
  color: #909399;
  font-weight: normal;
}

.submit-btn {
  width: 100%;
}

:deep(.el-form-item__label) {
  font-weight: 500;
  color: #606266;
}

:deep(.el-input__wrapper) {
  padding: 4px 11px;
}

:deep(.el-input__inner) {
  height: 40px;
}
</style>
