<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useGroupStore } from '@/stores/groups'
import { useChannelStore } from '@/stores/channels'
import { useApiKeyStore } from '@/stores/apikeys'
import { Plus, Edit, Delete, CopyDocument, Check, ArrowLeft } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const route = useRoute()
const router = useRouter()
const groupStore = useGroupStore()
const channelStore = useChannelStore()
const apiKeyStore = useApiKeyStore()

const groupId = computed(() => Number(route.params.id))
const group = computed(() => groupStore.groups.find((g: any) => g.id === groupId.value))
const activeTab = ref<'channels' | 'apikeys'>('channels')
const loading = ref(true)

// New key display
const newKeyValue = ref('')
const showNewKeyDialog = ref(false)
const keyCopied = ref(false)

// Channel CRUD
const channelDialogVisible = ref(false)
const channelDialogType = ref<'create' | 'edit'>('create')
const channelFormRef = ref()
const channelSubmitting = ref(false)
const channelForm = ref({
  id: 0,
  name: '',
  model_mapping: '',
  base_url: '',
  api_key: '',
  weight: 1,
  status: 1
})

// API Key
const apiKeySubmitting = ref(false)

onMounted(async () => {
  try {
    await Promise.all([
      groupStore.fetchAll(),
      channelStore.fetchByGroup(groupId.value),
      apiKeyStore.fetchAll(groupId.value),
    ])
  } catch (e: any) {
    if (e?.message && e.message !== 'Response data is empty') {
      ElMessage.error('加载数据失败')
    }
  } finally {
    loading.value = false
  }
})

// === Channel Handlers ===
const channelRules = {
  name: [{ required: true, message: '请输入渠道名称', trigger: 'blur' }, { min: 2, message: '至少 2 个字符', trigger: 'blur' }],
  model_mapping: [{ required: true, message: '请输入模型映射', trigger: 'blur' }],
  base_url: [{ required: true, message: '请输入 Base URL', trigger: 'blur' }],
  api_key: [{ required: true, message: '请输入 API Key', trigger: 'blur' }],
  weight: [{ required: true, message: '请输入权重', trigger: 'blur' }]
}

const openCreateChannel = () => {
  channelDialogType.value = 'create'
  channelForm.value = { id: 0, name: '', model_mapping: '', base_url: '', api_key: '', weight: 1, status: 1 }
  channelDialogVisible.value = true
}

const openEditChannel = (row: any) => {
  channelDialogType.value = 'edit'
  channelForm.value = {
    id: row.id,
    name: row.name,
    model_mapping: row.model_mapping,
    base_url: row.base_url,
    api_key: row.api_key,
    weight: row.weight,
    status: row.status
  }
  channelDialogVisible.value = true
}

const handleSaveChannel = async () => {
  const valid = await channelFormRef.value?.validate().catch(() => false)
  if (!valid) return

  channelSubmitting.value = true
  try {
    if (channelDialogType.value === 'edit') {
      await channelStore.update(channelForm.value.id, {
        group_id: groupId.value,
        name: channelForm.value.name,
        model_mapping: channelForm.value.model_mapping,
        base_url: channelForm.value.base_url,
        api_key: channelForm.value.api_key,
        weight: channelForm.value.weight,
        status: channelForm.value.status
      })
      ElMessage.success('渠道更新成功')
    } else {
      await channelStore.create({
        group_id: groupId.value,
        name: channelForm.value.name,
        model_mapping: channelForm.value.model_mapping,
        base_url: channelForm.value.base_url,
        api_key: channelForm.value.api_key,
        weight: channelForm.value.weight
      })
      ElMessage.success('渠道创建成功')
    }
    channelDialogVisible.value = false
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  } finally {
    channelSubmitting.value = false
  }
}

const handleDeleteChannel = async (row: any) => {
  try {
    await ElMessageBox.confirm(`确定要删除渠道「${row.name}」吗？`, '确认删除', {
      confirmButtonText: '确认删除',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await channelStore.remove(row.id)
    ElMessage.success('渠道已删除')
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e.message || '删除失败')
    }
  }
}

// === API Key Handlers ===
const handleCreateApiKey = async () => {
  apiKeySubmitting.value = true
  try {
    const result = await apiKeyStore.create(groupId.value)
    newKeyValue.value = result.key
    showNewKeyDialog.value = true
  } catch (e: any) {
    ElMessage.error(e.message || '创建失败')
  } finally {
    apiKeySubmitting.value = false
  }
}

const copyKey = async () => {
  try {
    await navigator.clipboard.writeText(newKeyValue.value)
    keyCopied.value = true
    setTimeout(() => { keyCopied.value = false }, 2000)
  } catch {
    ElMessage.error('复制失败')
  }
}

const handleDeleteKey = async (row: any) => {
  try {
    await ElMessageBox.confirm(
      `<div class="delete-confirm-content">
        <p class="delete-warning">删除后，使用此 Key 的应用将<strong>立即无法访问</strong>网关。</p>
        <div class="delete-key-info">
          <span class="delete-key-label">API Key：</span>
          <code class="delete-key-value">${maskKey(row.key)}</code>
        </div>
      </div>`,
      '确认删除 API Key',
      {
        confirmButtonText: '确认删除',
        cancelButtonText: '取消',
        type: 'warning',
        dangerouslyUseHTMLString: true,
        customClass: 'delete-confirm-dialog'
      }
    )
    await apiKeyStore.remove(row.key)
    ElMessage.success('API Key 已删除')
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e.message || '删除失败')
    }
  }
}

const statusType = (status: number) => {
  if (status === 1) return 'success'
  if (status === 2) return 'danger'
  return 'info'
}

const statusLabel = (status: number) => {
  if (status === 1) return '正常'
  if (status === 2) return '熔断'
  return '禁用'
}

const maskKey = (key: string) => {
  if (key.length > 12) return key.slice(0, 8) + '****' + key.slice(-4)
  return key
}

const formatDate = (date: string) => {
  if (!date) return '-'
  return date.slice(0, 19).replace('T', ' ')
}
</script>

<template>
  <div class="group-detail">
    <!-- Header -->
    <div class="page-header">
      <div class="header-left">
        <el-button text :icon="ArrowLeft" @click="router.push('/groups')">
          返回
        </el-button>
        <div v-if="group" class="group-info">
          <h2 class="group-name">
            {{ group.name }}
            <el-tag :type="group.status === 1 ? 'success' : 'info'" size="small">
              {{ group.status === 1 ? '启用' : '停用' }}
            </el-tag>
          </h2>
          <p class="group-id">项目组 ID: {{ group.id }}</p>
        </div>
        <el-skeleton v-else :rows="1" style="width: 200px" />
      </div>
    </div>

    <!-- Tabs -->
    <el-tabs v-model="activeTab" type="border-card">
      <!-- Channels Tab -->
      <el-tab-pane label="渠道" name="channels">
        <div class="tab-header">
          <p class="tab-desc">管理此项目组下的上游渠道</p>
          <el-button type="primary" size="small" :icon="Plus" @click="openCreateChannel">
            添加渠道
          </el-button>
        </div>

        <el-table :data="channelStore.channels" stripe v-loading="loading">
          <el-table-column prop="name" label="名称" min-width="150" />
          <el-table-column prop="model_mapping" label="模型映射" min-width="150" />
          <el-table-column prop="base_url" label="Base URL" min-width="200" show-overflow-tooltip />
          <el-table-column prop="weight" label="权重" width="80" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="statusType(row.status)" size="small">
                {{ statusLabel(row.status) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" :icon="Edit" @click="openEditChannel(row)" />
              <el-button link type="danger" :icon="Delete" @click="handleDeleteChannel(row)" />
            </template>
          </el-table-column>
          <template #empty>
            <el-empty description="暂无渠道" />
          </template>
        </el-table>
      </el-tab-pane>

      <!-- API Keys Tab -->
      <el-tab-pane :label="`API Key (${apiKeyStore.apiKeys.length})`" name="apikeys">
        <div class="tab-header">
          <p class="tab-desc">管理此项目组的 API Key</p>
          <el-button type="primary" size="small" :icon="Plus" @click="handleCreateApiKey" :loading="apiKeySubmitting">
            创建 Key
          </el-button>
        </div>

        <el-table :data="apiKeyStore.apiKeys" stripe>
          <el-table-column prop="key" label="API Key" min-width="300">
            <template #default="{ row }">
              <code>{{ maskKey(row.key) }}</code>
            </template>
          </el-table-column>
          <el-table-column label="创建时间" width="180">
            <template #default="{ row }">
              {{ formatDate(row.created_at) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="100" fixed="right">
            <template #default="{ row }">
              <el-button link type="danger" :icon="Delete" @click="handleDeleteKey(row)" />
            </template>
          </el-table-column>
          <template #empty>
            <el-empty description="暂无 API Key" />
          </template>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <!-- Channel Form Dialog -->
    <el-dialog
      v-model="channelDialogVisible"
      :title="channelDialogType === 'create' ? '添加渠道' : '编辑渠道'"
      width="600px"
    >
      <el-form ref="channelFormRef" :model="channelForm" :rules="channelRules" label-width="100px">
        <el-form-item label="渠道名称" prop="name">
          <el-input v-model="channelForm.name" placeholder="例如：OpenAI 主通道" />
        </el-form-item>
        <el-form-item label="模型映射" prop="model_mapping">
          <el-input v-model="channelForm.model_mapping" placeholder="例如：gpt-4o,gpt-4o-mini" />
        </el-form-item>
        <el-form-item label="Base URL" prop="base_url">
          <el-input v-model="channelForm.base_url" placeholder="例如：https://api.openai.com" />
        </el-form-item>
        <el-form-item label="API Key" prop="api_key">
          <el-input v-model="channelForm.api_key" type="password" placeholder="上游供应商 API Key" show-password />
        </el-form-item>
        <el-form-item label="权重" prop="weight">
          <el-input-number v-model="channelForm.weight" :min="1" :max="100" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="channelDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="channelSubmitting" @click="handleSaveChannel">
          {{ channelSubmitting ? '保存中...' : '保存' }}
        </el-button>
      </template>
    </el-dialog>

    <!-- New Key Dialog -->
    <el-dialog v-model="showNewKeyDialog" title="API Key 已创建" width="500px" :close-on-click-modal="false">
      <el-alert
        title="请立即复制并保存此 Key，关闭后将无法再次查看完整 Key。"
        type="warning"
        :closable="false"
        style="margin-bottom: 16px"
      />
      <div class="key-display">
        <code class="key-value">{{ newKeyValue }}</code>
        <el-button :icon="keyCopied ? Check : CopyDocument" @click="copyKey">
          {{ keyCopied ? '已复制' : '复制' }}
        </el-button>
      </div>
      <template #footer>
        <el-button type="primary" @click="showNewKeyDialog = false">我已保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.group-detail {
  max-width: 1200px;
}

.page-header {
  margin-bottom: 20px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.group-info {
  flex: 1;
}

.group-name {
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  margin: 0 0 4px 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.group-id {
  font-size: 13px;
  color: #909399;
  margin: 0;
}

.tab-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.tab-desc {
  font-size: 14px;
  color: #606266;
  margin: 0;
}

.key-display {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  background: #f5f7fa;
  border-radius: 8px;
}

.key-value {
  flex: 1;
  font-family: monospace;
  font-size: 14px;
  word-break: break-all;
}

code {
  font-family: monospace;
  background: #f5f7fa;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 13px;
}
</style>
