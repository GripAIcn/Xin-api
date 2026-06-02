<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useGroupStore } from '@/stores/groups'
import { Plus, Edit, Delete } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const router = useRouter()
const groupStore = useGroupStore()

const loading = ref(true)
const dialogVisible = ref(false)
const dialogType = ref<'create' | 'edit'>('create')
const formRef = ref()
const submitting = ref(false)

const form = ref({
  id: 0,
  name: ''
})

const rules = {
  name: [
    { required: true, message: '请输入项目组名称', trigger: 'blur' },
    { min: 2, max: 100, message: '长度需在 2-100 个字符之间', trigger: 'blur' }
  ]
}

onMounted(async () => {
  try {
    await groupStore.fetchAll()
  } catch (e: any) {
    ElMessage.error(e?.message || '加载项目组列表失败')
  } finally {
    loading.value = false
  }
})

const openCreate = () => {
  dialogType.value = 'create'
  form.value = { id: 0, name: '' }
  dialogVisible.value = true
}

const openEdit = (row: any) => {
  dialogType.value = 'edit'
  form.value = { id: row.id, name: row.name }
  dialogVisible.value = true
}

const handleSubmit = async () => {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    if (dialogType.value === 'create') {
      await groupStore.create(form.value.name)
      ElMessage.success('项目组创建成功')
    } else {
      await groupStore.update(form.value.id, form.value.name)
      ElMessage.success('项目组更新成功')
    }
    dialogVisible.value = false
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const handleToggleStatus = async (row: any) => {
  try {
    await groupStore.toggleStatus(row.id, row.status)
    ElMessage.success(row.status === 1 ? '项目组已停用' : '项目组已启用')
  } catch (e: any) {
    ElMessage.error(e.message || '操作失败')
  }
}

const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除项目组「${row.name}」吗？此操作不可撤销，但不会影响已创建的 API Key。`,
      '确认删除',
      {
        confirmButtonText: '确认删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )
    await groupStore.remove(row.id)
    ElMessage.success('项目组已删除')
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e.message || '删除失败')
    }
  }
}

const formatDate = (date: string) => {
  if (!date) return '-'
  return date.slice(0, 19).replace('T', ' ')
}
</script>

<template>
  <div style="max-width: 1200px;">
    <!-- 页面标题 -->
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px;">
      <div>
        <h2 style="font-size: 20px; font-weight: 600; color: #303133; margin-bottom: 8px;">项目组</h2>
        <p style="font-size: 14px; color: #909399;">管理 API 项目组及其状态</p>
      </div>
      <el-button type="primary" :icon="Plus" @click="openCreate">
        创建项目组
      </el-button>
    </div>

    <!-- 数据表格 -->
    <el-card shadow="never">
      <el-table
        v-loading="loading"
        :data="groupStore.groups"
        stripe
        style="width: 100%"
      >
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="名称" min-width="200">
          <template #default="{ row }">
            <el-link
              type="primary"
              @click="router.push(`/groups/${row.id}`)"
            >
              {{ row.name }}
            </el-link>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">
              {{ row.status === 1 ? '启用' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-switch
              :model-value="row.status === 1"
              @change="handleToggleStatus(row)"
              style="margin-right: 12px;"
            />
            <el-button
              link
              type="primary"
              :icon="Edit"
              @click="openEdit(row)"
            />
            <el-button
              link
              type="danger"
              :icon="Delete"
              @click="handleDelete(row)"
            />
          </template>
        </el-table-column>
        <template #empty>
          <el-empty description="暂无项目组，点击上方按钮创建" />
        </template>
      </el-table>
    </el-card>

    <!-- 创建/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogType === 'create' ? '创建项目组' : '编辑项目组'"
      width="500px"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="100px"
      >
        <el-form-item label="项目组名称" prop="name">
          <el-input
            v-model="form.name"
            placeholder="输入名称"
            maxlength="100"
            show-word-limit
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="submitting"
          @click="handleSubmit"
        >
          {{ submitting ? '保存中...' : '保存' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>
