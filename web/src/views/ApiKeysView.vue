<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useApiKeyStore } from '@/stores/apikeys'
import { useGroupStore } from '@/stores/groups'
import { Plus, Delete, CopyDocument, Check } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

const apiKeyStore = useApiKeyStore()
const groupStore = useGroupStore()

const loading = ref(true)
const selectedGroupId = ref<string>('')
const creating = ref(false)

const newKeyValue = ref('')
const showNewKeyDialog = ref(false)
const keyCopied = ref(false)

// 删除确认弹窗
const showDeleteDialog = ref(false)
const deleteKeyInfo = ref<any>(null)

const loadKeys = async () => {
  loading.value = true
  try {
    const gid = selectedGroupId.value
    let groupId: number | undefined = undefined

    if (gid && gid !== 'all' && gid !== '') {
      const parsed = Number(gid)
      if (!isNaN(parsed) && parsed > 0) {
        groupId = parsed
      }
    }

    if (groupId === undefined && groupStore.groups?.length) {
      groupId = groupStore.groups[0]!.id
      selectedGroupId.value = String(groupId)
    }

    if (groupId === undefined) {
      ElMessage.warning('请先创建项目组')
      return
    }

    await apiKeyStore.fetchAll(groupId)
  } catch (e: any) {
    ElMessage.error(e?.message || '加载 API Key 列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  try {
    await groupStore.fetchAll()
  } catch { }
  await loadKeys()
})

watch(selectedGroupId, () => loadKeys())

const handleCreate = async () => {
  if (!selectedGroupId.value) {
    ElMessage.warning('请先在筛选器中选择一个项目组')
    return
  }
  creating.value = true
  try {
    const result = await apiKeyStore.create(Number(selectedGroupId.value))
    newKeyValue.value = result.key
    showNewKeyDialog.value = true
  } catch (e: any) {
    ElMessage.error(e.message || '创建失败')
  } finally {
    creating.value = false
  }
}

const copyKey = async () => {
  try {
    await navigator.clipboard.writeText(newKeyValue.value)
    keyCopied.value = true
    setTimeout(() => { keyCopied.value = false }, 2000)
  } catch {
    ElMessage.error('复制失败，请手动复制')
  }
}

const openDeleteDialog = (row: any) => {
  deleteKeyInfo.value = row
  showDeleteDialog.value = true
}

const handleDelete = async () => {
  if (!deleteKeyInfo.value) return
  try {
    await apiKeyStore.remove(deleteKeyInfo.value.key)
    ElMessage.success('API Key 已删除')
    showDeleteDialog.value = false
    deleteKeyInfo.value = null
  } catch (e: any) {
    ElMessage.error(e.message || '删除失败')
  }
}

const maskKey = (key: string) => {
  if (key.length > 12) return key.slice(0, 8) + '****' + key.slice(-4)
  return key
}

const groupName = (groupId: number) => {
  return groupStore.groups?.find((g: any) => g.id === groupId)?.name || `ID:${groupId}`
}

const formatDate = (date: string) => {
  if (!date) return '-'
  return date.slice(0, 19).replace('T', ' ')
}
</script>

<template>
  <div class="apikeys-page">
    <!-- 页面标题 -->
    <div class="page-header">
      <div>
        <h2 class="page-title">API Key 管理</h2>
        <p class="page-desc">管理项目组的 API 访问密钥</p>
      </div>
      <el-button
        type="primary"
        :icon="Plus"
        @click="handleCreate"
        :disabled="creating || !selectedGroupId"
        :loading="creating"
      >
        创建 API Key
      </el-button>
    </div>

    <!-- 筛选器 -->
    <el-card shadow="never" class="filter-card">
      <el-select v-model="selectedGroupId" placeholder="选择项目组" style="width: 240px">
        <el-option label="全部项目组" value="all" />
        <el-option
          v-for="g in groupStore.groups"
          :key="g.id"
          :label="g.name"
          :value="String(g.id)"
        />
      </el-select>
    </el-card>

    <!-- 数据表格 -->
    <el-card shadow="never">
      <el-table
        v-loading="loading"
        :data="apiKeyStore.apiKeys"
        stripe
        style="width: 100%"
      >
        <el-table-column prop="key" label="API Key" min-width="350">
          <template #default="{ row }">
            <code>{{ maskKey(row.key) }}</code>
          </template>
        </el-table-column>
        <el-table-column label="所属项目组" min-width="150">
          <template #default="{ row }">
            {{ groupName(row.group_id) }}
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button link type="danger" :icon="Delete" @click="openDeleteDialog(row)" />
          </template>
        </el-table-column>
        <template #empty>
          <el-empty description="暂无 API Key" />
        </template>
      </el-table>
    </el-card>

    <!-- New Key Dialog -->
    <el-dialog
      v-model="showNewKeyDialog"
      title="API Key 已创建"
      width="500px"
      :close-on-click-modal="false"
    >
      <el-alert
        title="⚠️ 请立即复制并保存此 Key。关闭对话框后将无法再次查看完整 Key。"
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

    <!-- Delete Confirm Dialog -->
    <el-dialog
      v-model="showDeleteDialog"
      title="确认删除 API Key"
      width="500px"
      :close-on-click-modal="false"
    >
      <el-alert
        title="删除后，使用此 Key 的应用将立即无法访问网关。"
        type="warning"
        :closable="false"
        style="margin-bottom: 16px"
      />
      <div class="delete-key-display" v-if="deleteKeyInfo">
        <span class="delete-key-label">API Key：</span>
        <code class="delete-key-value">{{ maskKey(deleteKeyInfo.key) }}</code>
      </div>
      <template #footer>
        <el-button @click="showDeleteDialog = false">取消</el-button>
        <el-button type="danger" @click="handleDelete">确认删除</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.apikeys-page {
  max-width: 1200px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  margin: 0 0 8px 0;
}

.page-desc {
  font-size: 14px;
  color: #909399;
  margin: 0;
}

.filter-card {
  margin-bottom: 20px;
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

.delete-key-display {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: #f5f7fa;
  border-radius: 8px;
  border-left: 3px solid #409eff;
}

.delete-key-label {
  font-size: 13px;
  color: #909399;
  flex-shrink: 0;
}

.delete-key-value {
  font-family: monospace;
  font-size: 14px;
  color: #606266;
  word-break: break-all;
}

code {
  font-family: monospace;
  background: #f5f7fa;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 13px;
}
</style>
