<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { User, Lock } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

const router = useRouter()
const authStore = useAuthStore()

const loginForm = ref({
  username: '',
  password: ''
})

const loading = ref(false)
const loginFormRef = ref()

// 如果已登录，自动跳转到仪表盘
onMounted(() => {
  if (authStore.isAuthenticated) {
    router.replace('/dashboard')
  }
})

const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 5, max: 10, message: '用户名长度需在 5-10 个字符之间', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 8, max: 20, message: '密码长度需在 8-20 个字符之间', trigger: 'blur' }
  ]
}

const handleLogin = async () => {
  const valid = await loginFormRef.value?.validate().catch(() => false)
  if (!valid) return

  loading.value = true
  try {
    await authStore.login(loginForm.value.username, loginForm.value.password)
    router.replace('/dashboard')
  } catch (e: any) {
    ElMessage.error(e.message || '登录失败，请重试')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div style="display: flex; min-height: 100vh; width: 100%;">
    <!-- 左侧装饰区域 -->
    <div style="flex: 1; display: flex; flex-direction: column; justify-content: center; padding: 64px 80px; color: white; position: relative; overflow: hidden; background: linear-gradient(145deg, #1a2a3a 0%, #2d4a5e 100%);">
      <!-- 背景光晕 -->
      <div style="position: absolute; top: -50%; right: -20%; width: 80%; height: 200%; pointer-events: none; background: radial-gradient(ellipse, rgba(64, 158, 255, 0.08) 0%, transparent 70%);"></div>

      <div style="position: relative; z-index: 10;">
        <div style="display: flex; align-items: center; gap: 16px; margin-bottom: 40px;">
          <div style="width: 56px; height: 56px; border-radius: 12px; display: flex; align-items: center; justify-content: center; font-size: 24px; font-weight: bold; color: white; background: #409eff; box-shadow: 0 8px 24px rgba(64, 158, 255, 0.3);">
            X
          </div>
          <span style="font-size: 24px; font-weight: 600; letter-spacing: 1px;">Xin-api</span>
        </div>
        <h1 style="font-size: 36px; font-weight: 600; margin-bottom: 20px; line-height: 1.3;">
          API 网关管理控制台
        </h1>
        <p style="font-size: 16px; line-height: 1.6; opacity: 0.8; margin-bottom: 64px; max-width: 400px;">
          统一管理项目组、上游渠道和 API Key，<br>
          轻松构建高性能的 AI 服务接入层
        </p>
      </div>

      <div style="position: relative; z-index: 10; display: flex; flex-direction: column; gap: 20px;">
        <div style="display: flex; align-items: center; gap: 16px; font-size: 15px; opacity: 0.9;">
          <div style="width: 8px; height: 8px; border-radius: 50%; background: #409eff; box-shadow: 0 0 12px rgba(64, 158, 255, 0.5);"></div>
          <span>多项目组隔离管理</span>
        </div>
        <div style="display: flex; align-items: center; gap: 16px; font-size: 15px; opacity: 0.9;">
          <div style="width: 8px; height: 8px; border-radius: 50%; background: #409eff; box-shadow: 0 0 12px rgba(64, 158, 255, 0.5);"></div>
          <span>智能负载均衡与熔断</span>
        </div>
        <div style="display: flex; align-items: center; gap: 16px; font-size: 15px; opacity: 0.9;">
          <div style="width: 8px; height: 8px; border-radius: 50%; background: #409eff; box-shadow: 0 0 12px rgba(64, 158, 255, 0.5);"></div>
          <span>实时流量监控与统计</span>
        </div>
      </div>
    </div>

    <!-- 右侧登录表单 -->
    <div style="width: 480px; display: flex; align-items: center; justify-content: center; padding: 40px; background: white;">
      <el-card style="width: 360px; border: none; box-shadow: none;" shadow="never">
        <div style="margin-bottom: 40px;">
          <h2 style="font-size: 28px; font-weight: 600; color: #303133; margin-bottom: 8px;">欢迎登录</h2>
          <p style="font-size: 14px; color: #909399;">请使用管理员账号登录系统</p>
        </div>

        <el-form
          ref="loginFormRef"
          :model="loginForm"
          :rules="rules"
          style="margin-bottom: 40px;"
          @keyup.enter="handleLogin"
        >
          <el-form-item prop="username" style="margin-bottom: 24px;">
            <el-input
              v-model="loginForm.username"
              placeholder="请输入用户名"
              :prefix-icon="User"
              size="large"
              clearable
            />
          </el-form-item>

          <el-form-item prop="password" style="margin-bottom: 24px;">
            <el-input
              v-model="loginForm.password"
              type="password"
              placeholder="请输入密码"
              :prefix-icon="Lock"
              size="large"
              show-password
              clearable
            />
          </el-form-item>

          <el-form-item>
            <el-button
              type="primary"
              size="large"
              style="width: 100%; height: 44px; font-size: 16px; margin-top: 8px;"
              :loading="loading"
              @click="handleLogin"
            >
              {{ loading ? '登录中...' : '登录' }}
            </el-button>
          </el-form-item>
        </el-form>

        <div style="text-align: center; font-size: 12px; color: #c0c4cc;">
          <p>© 2026 Xin-api. All rights reserved.</p>
        </div>
      </el-card>
    </div>
  </div>
</template>

<style scoped>
/* 响应式 - 移动端 */
@media (max-width: 1024px) {
  div > div:first-child {
    display: none;
  }

  div > div:last-child {
    width: 100%;
    background: linear-gradient(145deg, #1a2a3a 0%, #2d4a5e 100%);
  }

  .el-card {
    background: transparent;
  }

  h2, p {
    color: white !important;
  }

  p:last-child {
    color: rgba(255, 255, 255, 0.5) !important;
  }
}
</style>
