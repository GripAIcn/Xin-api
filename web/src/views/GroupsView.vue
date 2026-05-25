<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useGroupStore } from '@/stores/groups'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Card, CardContent } from '@/components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose } from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { toast } from 'vue-sonner'
import { Plus, Pencil, Trash2 } from 'lucide-vue-next'
import ErrorAlertDialog from '@/components/ErrorAlertDialog.vue'

const router = useRouter()
const groupStore = useGroupStore()

const loading = ref(true)
const createDialogOpen = ref(false)
const editDialogOpen = ref(false)
const deleteDialogOpen = ref(false)
const editingGroup = ref<any>(null)
const deletingGroup = ref<any>(null)
const formName = ref('')
const submitting = ref(false)

// 错误弹窗状态
const errorDialog = ref({ open: false, title: '提示', message: '' })
const showError = (message: string, title = '提示') => {
  errorDialog.value = { open: true, title, message }
}

onMounted(async () => {
  try {
    await groupStore.fetchAll()
  } catch (e: any) {
    showError(e?.message || '加载项目组列表失败')
  } finally {
    loading.value = false
  }
})

const openCreate = () => {
  formName.value = ''
  createDialogOpen.value = true
}

const validateName = () => {
  if (!formName.value || formName.value.length < 2 || formName.value.length > 100) {
    return '项目组名称长度需在 2-100 个字符之间'
  }
  return null
}

const handleCreate = async () => {
  const error = validateName()
  if (error) {
    showError(error)
    return
  }
  submitting.value = true
  try {
    await groupStore.create(formName.value)
    toast.success('项目组创建成功')
    createDialogOpen.value = false
  } catch (e: any) {
    showError(e.message || '创建失败')
  } finally {
    submitting.value = false
  }
}

const openEdit = (group: any) => {
  editingGroup.value = group
  formName.value = group.name
  editDialogOpen.value = true
}

const handleEdit = async () => {
  const error = validateName()
  if (error) {
    showError(error)
    return
  }
  submitting.value = true
  try {
    await groupStore.update(editingGroup.value.id, formName.value)
    toast.success('项目组更新成功')
    editDialogOpen.value = false
  } catch (e: any) {
    showError(e.message || '更新失败')
  } finally {
    submitting.value = false
  }
}

const handleToggleStatus = async (group: any) => {
  try {
    await groupStore.toggleStatus(group.id, group.status)
    toast.success(group.status === 1 ? '项目组已停用' : '项目组已启用')
  } catch (e: any) {
    showError(e.message || '操作失败')
  }
}

const openDelete = (group: any) => {
  deletingGroup.value = group
  deleteDialogOpen.value = true
}

const handleDelete = async () => {
  submitting.value = true
  try {
    await groupStore.remove(deletingGroup.value.id)
    toast.success('项目组已删除')
    deleteDialogOpen.value = false
  } catch (e: any) {
    showError(e.message || '删除失败')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold tracking-tight">项目组</h1>
        <p class="text-muted-foreground">管理 API 项目组及其状态</p>
      </div>
      <Button @click="openCreate">
        <Plus class="h-4 w-4 mr-1" />创建项目组
      </Button>
    </div>

    <Card>
      <CardContent class="p-0">
        <Table v-if="!loading">
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>名称</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>创建时间</TableHead>
              <TableHead class="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="!groupStore.groups?.length">
              <TableCell colspan="5" class="text-center text-muted-foreground py-8">
                暂无项目组，点击上方按钮创建
              </TableCell>
            </TableRow>
            <TableRow v-for="group in groupStore.groups" :key="group.id" class="cursor-pointer" @click="router.push(`/groups/${group.id}`)">
              <TableCell class="font-mono text-sm">{{ group.id }}</TableCell>
              <TableCell class="font-medium">{{ group.name }}</TableCell>
              <TableCell>
                <Badge :variant="group.status === 1 ? 'default' : 'secondary'">
                  {{ group.status === 1 ? '启用' : '停用' }}
                </Badge>
              </TableCell>
              <TableCell class="text-muted-foreground">{{ group.created_at?.slice(0, 19).replace('T', ' ') }}</TableCell>
              <TableCell class="text-right" @click.stop>
                <div class="flex items-center justify-end gap-2">
                  <Switch :checked="group.status === 1" @update:checked="handleToggleStatus(group)" />
                  <Button variant="ghost" size="icon" @click="openEdit(group)">
                    <Pencil class="h-4 w-4" />
                  </Button>
                  <Button variant="ghost" size="icon" @click="openDelete(group)">
                    <Trash2 class="h-4 w-4 text-destructive" />
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
        <div v-else class="p-6 space-y-3">
          <Skeleton v-for="i in 3" :key="i" class="h-10 w-full" />
        </div>
      </CardContent>
    </Card>

    <!-- Create Dialog -->
    <Dialog v-model:open="createDialogOpen">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>创建项目组</DialogTitle>
          <DialogDescription>输入项目组名称，长度 2-100 个字符</DialogDescription>
        </DialogHeader>
        <form @submit.prevent="handleCreate">
          <div class="space-y-4 py-4">
            <div class="space-y-2">
              <Label for="create-name">项目组名称</Label>
              <Input id="create-name" v-model="formName" placeholder="输入名称" :maxlength="100" />
            </div>
          </div>
          <DialogFooter>
            <DialogClose as-child>
              <Button variant="outline" type="button">取消</Button>
            </DialogClose>
            <Button type="submit" :disabled="submitting">{{ submitting ? '创建中...' : '创建' }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <!-- Edit Dialog -->
    <Dialog v-model:open="editDialogOpen">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>编辑项目组</DialogTitle>
          <DialogDescription>修改项目组名称</DialogDescription>
        </DialogHeader>
        <form @submit.prevent="handleEdit">
          <div class="space-y-4 py-4">
            <div class="space-y-2">
              <Label for="edit-name">项目组名称</Label>
              <Input id="edit-name" v-model="formName" placeholder="输入名称" :maxlength="100" />
            </div>
          </div>
          <DialogFooter>
            <DialogClose as-child>
              <Button variant="outline" type="button">取消</Button>
            </DialogClose>
            <Button type="submit" :disabled="submitting">{{ submitting ? '保存中...' : '保存' }}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <!-- Delete Dialog -->
    <Dialog v-model:open="deleteDialogOpen">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>确认删除</DialogTitle>
          <DialogDescription>
            确定要删除项目组「{{ deletingGroup?.name }}」吗？此操作不可撤销，但不会影响已创建的 API Key。
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <DialogClose as-child>
            <Button variant="outline" type="button">取消</Button>
          </DialogClose>
          <Button variant="destructive" @click="handleDelete" :disabled="submitting">
            {{ submitting ? '删除中...' : '确认删除' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Error Alert Dialog -->
    <ErrorAlertDialog v-model:open="errorDialog.open" :title="errorDialog.title" :message="errorDialog.message" />
  </div>
</template>
