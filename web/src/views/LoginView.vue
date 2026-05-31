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
  <div class="login-page">
    <!-- 左侧装饰区域 -->
    <div class="login-left">
      <div class="brand-section">
        <div class="brand-logo">
          <span class="logo-icon">X</span>
          <span class="logo-text">Xin-api</span>
        </div>
        <h1 class="brand-title">API 网关管理控制台</h1>
        <p class="brand-desc">
          统一管理项目组、上游渠道和 API Key，<br>
          轻松构建高性能的 AI 服务接入层
        </p>
      </div>
      <div class="feature-list">
        <div class="feature-item">
          <div class="feature-dot"></div>
          <span>多项目组隔离管理</span>
        </div>
        <div class="feature-item">
          <div class="feature-dot"></div>
          <span>智能负载均衡与熔断</span>
        </div>
        <div class="feature-item">
          <div class="feature-dot"></div>
          <span>实时流量监控与统计</span>
        </div>
      </div>
    </div>

    <!-- 右侧登录表单 -->
    <div class="login-right">
      <el-card class="login-card" shadow="never">
        <div class="login-header">
          <h2 class="title">欢迎登录</h2>
          <p class="subtitle">请使用管理员账号登录系统</p>
        </div>

        <el-form
          ref="loginFormRef"
          :model="loginForm"
          :rules="rules"
          class="login-form"
          @keyup.enter="handleLogin"
        >
          <el-form-item prop="username">
            <el-input
              v-model="loginForm.username"
              placeholder="请输入用户名"
              :prefix-icon="User"
              size="large"
              clearable
            />
          </el-form-item>

          <el-form-item prop="password">
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
              class="login-button"
              :loading="loading"
              @click="handleLogin"
            >
              {{ loading ? '登录中...' : '登录' }}
            </el-button>
          </el-form-item>
        </el-form>

        <div class="login-footer">
          <p>© 2026 Xin-api. All rights reserved.</p>
        </div>
      </el-card>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  background: #f5f7fa;
}

/* 左侧区域 */
.login-left {
  flex: 1;
  background: linear-gradient(145deg, #1a2a3a 0%, #2d4a5e 100%);
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 60px 80px;
  color: white;
  position: relative;
  overflow: hidden;
}

.login-left::before {
  content: '';
  position: absolute;
  top: -50%;
  right: -20%;
  width: 80%;
  height: 200%;
  background: radial-gradient(ellipse, rgba(64, 158, 255, 0.08) 0%, transparent 70%);
  pointer-events: none;
}

.brand-section {
  position: relative;
  z-index: 1;
}

.brand-logo {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 40px;
}

.logo-icon {
  width: 56px;
  height: 56px;
  background: #409eff;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  font-weight: bold;
  box-shadow: 0 8px 24px rgba(64, 158, 255, 0.3);
}

.logo-text {
  font-size: 28px;
  font-weight: 600;
  letter-spacing: 1px;
}

.brand-title {
  font-size: 36px;
  font-weight: 600;
  margin: 0 0 20px 0;
  line-height: 1.3;
}

.brand-desc {
  font-size: 16px;
  line-height: 1.8;
  opacity: 0.8;
  margin: 0 0 60px 0;
  max-width: 480px;
}

.feature-list {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.feature-item {
  display: flex;
  align-items: center;
  gap: 16px;
  font-size: 15px;
  opacity: 0.9;
}

.feature-dot {
  width: 8px;
  height: 8px;
  background: #409eff;
  border-radius: 50%;
  box-shadow: 0 0 12px rgba(64, 158, 255, 0.5);
}

/* 右侧区域 */
.login-right {
  width: 480px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
  background: white;
}

.login-card {
  width: 100%;
  max-width: 360px;
  border: none;
  box-shadow: none;
}

.login-header {
  margin-bottom: 40px;
}

.title {
  font-size: 28px;
  font-weight: 600;
  color: #303133;
  margin: 0 0 8px 0;
}

.subtitle {
  font-size: 14px;
  color: #909399;
  margin: 0;
}

.login-form {
  margin-bottom: 40px;
}

.login-form :deep(.el-form-item) {
  margin-bottom: 24px;
}

.login-button {
  width: 100%;
  height: 44px;
  font-size: 16px;
  margin-top: 8px;
}

.login-footer {
  text-align: center;
  font-size: 12px;
  color: #c0c4cc;
}

.login-footer p {
  margin: 0;
}

/* 响应式 */
@media (max-width: 992px) {
  .login-left {
    display: none;
  }

  .login-right {
    width: 100%;
    background: linear-gradient(145deg, #1a2a3a 0%, #2d4a5e 100%);
  }

  .login-card {
    background: white;
    padding: 40px 32px;
    border-radius: 12px;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
  }
}

@media (max-width: 576px) {
  .login-right {
    padding: 20px;
  }

  .login-card {
    padding: 32px 24px;
  }
}
</style>
