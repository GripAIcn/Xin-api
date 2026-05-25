<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useChannelStore } from '@/stores/channels'
import { useGroupStore } from '@/stores/groups'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose } from '@/components/ui/dialog'
import { toast } from 'vue-sonner'
import { Plus, Pencil, Trash2 } from 'lucide-vue-next'
import ErrorAlertDialog from '@/components/ErrorAlertDialog.vue'

const channelStore = useChannelStore()
const groupStore = useGroupStore()

const loading = ref(true)
const selectedGroupId = ref<string>('')
const channelDialogOpen = ref(false)
const deleteDialogOpen = ref(false)
const editingChannel = ref<any>(null)
const deletingChannel = ref<any>(null)
const channelForm = ref({ group_id: '', name: '', model_mapping: '', base_url: '', api_key: '', weight: 1 })
const submitting = ref(false)

// 错误弹窗状态
const errorDialog = ref({ open: false, title: '提示', message: '' })
const showError = (message: string, title = '提示') => {
  errorDialog.value = { open: true, title, message }
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
      showError('请先创建项目组')
      return
    }

    await channelStore.fetchByGroup(groupId)
  } catch (e: any) {
    showError(e?.message || '加载渠道列表失败')
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
  editingChannel.value = null
  channelForm.value = { group_id: selectedGroupId.value || '', name: '', model_mapping: '', base_url: '', api_key: '', weight: 1 }
  channelDialogOpen.value = true
}

const openEdit = (ch: any) => {
  editingChannel.value = ch
  channelForm.value = {
    group_id: String(ch.group_id),
    name: ch.name,
    model_mapping: ch.model_mapping,
    base_url: ch.base_url,
    api_key: ch.api_key,
    weight: ch.weight,
  }
  channelDialogOpen.value = true
}

const validateForm = () => {
  const f = channelForm.value
  const groupId = Number(f.group_id)
  if (!groupId) return '请选择项目组'
  if (!f.name || f.name.length < 2) return '渠道名称至少 2 个字符'
  if (!f.model_mapping) return '请输入模型映射'
  if (!f.base_url) return '请输入 Base URL'
  if (!f.api_key) return '请输入 API Key'
  if (f.weight < 1 || f.weight > 100) return '权重范围 1-100'
  return null
}

const handleSave = async () => {
  const error = validateForm()
  if (error) {
    showError(error)
    return
  }

  const f = channelForm.value
  const groupId = Number(f.group_id)

  submitting.value = true
  try {
    if (editingChannel.value) {
      await channelStore.update(editingChannel.value.id, { ...f, group_id: groupId, status: editingChannel.value.status })
      toast.success('渠道更新成功')
    } else {
      await channelStore.create({ ...f, group_id: groupId })
      toast.success('渠道创建成功')
    }
    channelDialogOpen.value = false
  } catch (e: any) {
    showError(e.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const openDelete = (ch: any) => {
  deletingChannel.value = ch
  deleteDialogOpen.value = true
}

const handleDelete = async () => {
  submitting.value = true
  try {
    await channelStore.remove(deletingChannel.value.id)
    toast.success('渠道已删除')
    deleteDialogOpen.value = false
  } catch (e: any) {
    showError(e.message || '删除失败')
  } finally {
    submitting.value = false
  }
}

const statusBadge = (status: number) => {
  if (status === 1) return { label: '正常', variant: 'default' as const }
  if (status === 2) return { label: '熔断', variant: 'destructive' as const }
  return { label: '禁用', variant: 'secondary' as const }
}

const groupName = (groupId: number) => {
  return groupStore.groups?.find((g: any) => g.id === groupId)?.name || `ID:${groupId}`
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold tracking-tight">渠道管理</h1>
        <p class="text-muted-foreground">管理所有上游供应商渠道</p>
      </div>
      <Button @click="openCreate" :disabled="!groupStore.groups?.length">
        <Plus class="h-4 w-4 mr-1" />添加渠道
      </Button>
    </div>

    <!-- Filter -->
    <div class="flex items-center gap-4">
      <div class="w-64">
        <Select v-model="selectedGroupId">
          <SelectTrigger>
            <SelectValue placeholder="全部项目组" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">全部项目组</SelectItem>
            <SelectItem v-for="g in groupStore.groups" :key="g.id" :value="String(g.id)">
              {{ g.name }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div>

    <Card>
      <CardContent class="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>ID</TableHead>
              <TableHead>所属项目组</TableHead>
              <TableHead>名称</TableHead>
              <TableHead>模型映射</TableHead>
              <TableHead>Base URL</TableHead>
              <TableHead>权重</TableHead>
              <TableHead>状态</TableHead>
              <TableHead class="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="!channelStore.channels?.length">
              <TableCell colspan="8" class="text-center text-muted-foreground py-8">
                {{ loading ? '加载中...' : '暂无渠道' }}
              </TableCell>
            </TableRow>
            <TableRow v-for="ch in channelStore.channels" :key="ch.id">
              <TableCell class="font-mono text-sm">{{ ch.id }}</TableCell>
              <TableCell>{{ groupName(ch.group_id) }}</TableCell>
              <TableCell class="font-medium">{{ ch.name }}</TableCell>
              <TableCell class="font-mono text-sm max-w-[150px] truncate">{{ ch.model_mapping }}</TableCell>
              <TableCell class="font-mono text-sm max-w-[200px] truncate">{{ ch.base_url }}</TableCell>
              <TableCell>{{ ch.weight }}</TableCell>
              <TableCell>
                <Badge :variant="statusBadge(ch.status).variant">{{ statusBadge(ch.status).label }}</Badge>
              </TableCell>
              <TableCell class="text-right">
                <div class="flex justify-end gap-1">
                  <Button variant="ghost" size="icon" @click="openEdit(ch)">
                    <Pencil class="h-4 w-4" />
                  </Button>
                  <Button variant="ghost" size="icon" @click="openDelete(ch)">
                    <Trash2 class="h-4 w-4 text-destructive" />
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </CardContent>
    </Card>

    <!-- Channel Form Dialog -->
    <Dialog v-model:open="channelDialogOpen">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ editingChannel ? '编辑渠道' : '添加渠道' }}</DialogTitle>
          <DialogDescription>配置上游渠道的连接信息</DialogDescription>
        </DialogHeader>
        <form @submit.prevent="handleSave">
          <div class="space-y-3 py-2">
            <div class="space-y-1.5">
              <Label>所属项目组</Label>
              <Select v-if="!editingChannel" v-model="channelForm.group_id">
                <SelectTrigger>
                  <SelectValue placeholder="选择项目组" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="g in groupStore.groups" :key="g.id" :value="String(g.id)">{{ g.name }}</SelectItem>
                </SelectContent>
              </Select>
              <Input v-else :value="groupName(Number(channelForm.group_id))" disabled />
            </div>
            <div class="space-y-1.5">
              <Label>渠道名称</Label>
              <Input v-model="channelForm.name" placeholder="例如：OpenAI 主通道" />
            </div>
            <div class="space-y-1.5">
              <Label>模型映射</Label>
              <Input v-model="channelForm.model_mapping" placeholder="例如：gpt-4o,deepseek-chat" />
            </div>
            <div class="space-y-1.5">
              <Label>Base URL</Label>
              <Input v-model="channelForm.base_url" placeholder="例如：https://api.openai.com" />
            </div>
            <div class="space-y-1.5">
              <Label>API Key</Label>
              <Input v-model="channelForm.api_key" placeholder="上游供应商 API Key" type="password" />
            </div>
            <div class="space-y-1.5">
              <Label>权重 (1-100)</Label>
              <Input v-model.number="channelForm.weight" type="number" min="1" max="100" />
            </div>
          </div>
          <DialogFooter class="mt-4">
            <DialogClose as-child>
              <Button variant="outline" type="button">取消</Button>
            </DialogClose>
            <Button type="submit" :disabled="submitting">
              {{ submitting ? '保存中...' : '保存' }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <!-- Delete Dialog -->
    <Dialog v-model:open="deleteDialogOpen">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>确认删除</DialogTitle>
          <DialogDescription>确定要删除渠道「{{ deletingChannel?.name }}」吗？</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <DialogClose as-child>
            <Button variant="outline">取消</Button>
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
