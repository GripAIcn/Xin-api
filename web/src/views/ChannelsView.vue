<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useChannelStore } from '@/stores/channels'
import { useGroupStore } from '@/stores/groups'
import { Plus, Edit, Delete } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const channelStore = useChannelStore()
const groupStore = useGroupStore()

const loading = ref(true)
const selectedGroupId = ref<string>('')
const dialogVisible = ref(false)
const dialogType = ref<'create' | 'edit'>('create')
const formRef = ref()
const submitting = ref(false)

const form = ref({
  id: 0,
  group_id: '',
  name: '',
  model_mapping: '',
  base_url: '',
  api_key: '',
  weight: 1,
  status: 1
})

const rules = {
  group_id: [{ required: true, message: '请选择项目组', trigger: 'change' }],
  name: [{ required: true, message: '请输入渠道名称', trigger: 'blur' }, { min: 2, message: '至少 2 个字符', trigger: 'blur' }],
  model_mapping: [{ required: true, message: '请输入模型映射', trigger: 'blur' }],
  base_url: [{ required: true, message: '请输入 Base URL', trigger: 'blur' }],
  api_key: [{ required: true, message: '请输入 API Key', trigger: 'blur' }],
  weight: [{ required: true, message: '请输入权重', trigger: 'blur' }]
}

const loadChannels = async () => {
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

    await channelStore.fetchByGroup(groupId)
  } catch (e: any) {
    ElMessage.error(e?.message || '加载渠道列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  try {
    await groupStore.fetchAll()
  } catch { }
  await loadChannels()
})

watch(selectedGroupId, () => loadChannels())

const openCreate = () => {
  dialogType.value = 'create'
  form.value = {
    id: 0,
    group_id: selectedGroupId.value || '',
    name: '',
    model_mapping: '',
    base_url: '',
    api_key: '',
    weight: 1,
    status: 1
  }
  dialogVisible.value = true
}

const openEdit = (row: any) => {
  dialogType.value = 'edit'
  form.value = {
    id: row.id,
    group_id: String(row.group_id),
    name: row.name,
    model_mapping: row.model_mapping,
    base_url: row.base_url,
    api_key: row.api_key,
    weight: row.weight,
    status: row.status
  }
  dialogVisible.value = true
}

const handleSubmit = async () => {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    const groupId = Number(form.value.group_id)
    if (dialogType.value === 'edit') {
      await channelStore.update(form.value.id, {
        group_id: groupId,
        name: form.value.name,
        model_mapping: form.value.model_mapping,
        base_url: form.value.base_url,
        api_key: form.value.api_key,
        weight: form.value.weight,
        status: form.value.status
      })
      ElMessage.success('渠道更新成功')
    } else {
      await channelStore.create({
        group_id: groupId,
        name: form.value.name,
        model_mapping: form.value.model_mapping,
        base_url: form.value.base_url,
        api_key: form.value.api_key,
        weight: form.value.weight
      })
      ElMessage.success('渠道创建成功')
    }
    dialogVisible.value = false
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (row: any) => {
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

const groupName = (groupId: number) => {
  return groupStore.groups?.find((g: any) => g.id === groupId)?.name || `ID:${groupId}`
}
</script>

<template>
  <div class="channels-page">
    <!-- 页面标题 -->
    <div class="page-header">
      <div>
        <h2 class="page-title">渠道管理</h2>
        <p class="page-desc">管理所有上游供应商渠道</p>
      </div>
      <el-button type="primary" :icon="Plus" @click="openCreate" :disabled="!groupStore.groups?.length">
        添加渠道
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
        :data="channelStore.channels"
        stripe
        style="width: 100%"
      >
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="所属项目组" min-width="150">
          <template #default="{ row }">
            {{ groupName(row.group_id) }}
          </template>
        </el-table-column>
        <el-table-column prop="name" label="名称" min-width="150" />
        <el-table-column prop="model_mapping" label="模型映射" min-width="150" show-overflow-tooltip />
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
            <el-button link type="primary" :icon="Edit" @click="openEdit(row)" />
            <el-button link type="danger" :icon="Delete" @click="handleDelete(row)" />
          </template>
        </el-table-column>
        <template #empty>
          <el-empty description="暂无渠道" />
        </template>
      </el-table>
    </el-card>

    <!-- 创建/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogType === 'create' ? '添加渠道' : '编辑渠道'"
      width="600px"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="所属项目组" prop="group_id">
          <el-select v-model="form.group_id" placeholder="选择项目组" style="width: 100%" :disabled="dialogType === 'edit'">
            <el-option
              v-for="g in groupStore.groups"
              :key="g.id"
              :label="g.name"
              :value="String(g.id)"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="渠道名称" prop="name">
          <el-input v-model="form.name" placeholder="例如：OpenAI 主通道" />
        </el-form-item>
        <el-form-item label="模型映射" prop="model_mapping">
          <el-input v-model="form.model_mapping" placeholder="例如：gpt-4o,deepseek-chat" />
        </el-form-item>
        <el-form-item label="Base URL" prop="base_url">
          <el-input v-model="form.base_url" placeholder="例如：https://api.openai.com" />
        </el-form-item>
        <el-form-item label="API Key" prop="api_key">
          <el-input v-model="form.api_key" type="password" placeholder="上游供应商 API Key" show-password />
        </el-form-item>
        <el-form-item label="权重" prop="weight">
          <el-input-number v-model="form.weight" :min="1" :max="100" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          {{ submitting ? '保存中...' : '保存' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.channels-page {
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
</style>
