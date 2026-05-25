<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useApiKeyStore } from '@/stores/apikeys'
import { useGroupStore } from '@/stores/groups'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/table'
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/select'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose } from '@/components/ui/dialog'
import { toast } from 'vue-sonner'
import { Plus, Trash2, Copy, Check } from 'lucide-vue-next'
import ErrorAlertDialog from '@/components/ErrorAlertDialog.vue'

const apiKeyStore = useApiKeyStore()
const groupStore = useGroupStore()

const loading = ref(true)
const selectedGroupId = ref<string>('')

const newKeyValue = ref('')
const showNewKey = ref(false)
const keyCopied = ref(false)
const creating = ref(false)

const deleteDialogOpen = ref(false)
const deletingKey = ref<any>(null)
const deleting = ref(false)

// 错误弹窗状态
const errorDialog = ref({ open: false, title: '提示', message: '' })
const showError = (message: string, title = '提示') => {
  errorDialog.value = { open: true, title, message }
}

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
      showError('请先创建项目组')
      return
    }

    await apiKeyStore.fetchAll(groupId)
  } catch (e: any) {
    showError(e?.message || '加载 API Key 列表失败')
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
    showError('请先在筛选器中选择一个项目组')
    return
  }
  creating.value = true
  try {
    const result = await apiKeyStore.create(Number(selectedGroupId.value))
    newKeyValue.value = result.key
    showNewKey.value = true
  } catch (e: any) {
    showError(e.message || '创建失败')
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
    showError('复制失败，请手动复制')
  }
}

const openDelete = (key: any) => {
  deletingKey.value = key
  deleteDialogOpen.value = true
}

const handleDelete = async () => {
  deleting.value = true
  try {
    await apiKeyStore.remove(deletingKey.value.key)
    toast.success('API Key 已删除')
    deleteDialogOpen.value = false
  } catch (e: any) {
    showError(e.message || '删除失败')
  } finally {
    deleting.value = false
  }
}

const maskKey = (key: string) => {
  if (key.length > 12) return key.slice(0, 8) + '****' + key.slice(-4)
  return key
}

const groupName = (groupId: number) => {
  return groupStore.groups?.find((g: any) => g.id === groupId)?.name || `ID:${groupId}`
}
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold tracking-tight">API Key 管理</h1>
        <p class="text-muted-foreground">管理项目组的 API 访问密钥</p>
      </div>
      <Button @click="handleCreate" :disabled="creating || !selectedGroupId">
        <Plus class="h-4 w-4 mr-1" />{{ creating ? '创建中...' : '创建 API Key' }}
      </Button>
    </div>

    <!-- Filter -->
    <div class="w-64">
      <Select v-model="selectedGroupId">
        <SelectTrigger>
          <SelectValue placeholder="选择项目组" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">全部项目组</SelectItem>
          <SelectItem v-for="g in groupStore.groups" :key="g.id" :value="String(g.id)">{{ g.name }}</SelectItem>
        </SelectContent>
      </Select>
    </div>

    <Card>
      <CardContent class="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>API Key</TableHead>
              <TableHead>所属项目组</TableHead>
              <TableHead>创建时间</TableHead>
              <TableHead class="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="!apiKeyStore.apiKeys?.length">
              <TableCell colspan="4" class="text-center text-muted-foreground py-8">
                {{ loading ? '加载中...' : '暂无 API Key' }}
              </TableCell>
            </TableRow>
            <TableRow v-for="ak in apiKeyStore.apiKeys" :key="ak.key">
              <TableCell class="font-mono text-sm">{{ maskKey(ak.key) }}</TableCell>
              <TableCell>{{ groupName(ak.group_id) }}</TableCell>
              <TableCell class="text-muted-foreground">{{ ak.created_at?.slice(0, 19).replace('T', ' ') }}</TableCell>
              <TableCell class="text-right">
                <Button variant="ghost" size="icon" @click="openDelete(ak)">
                  <Trash2 class="h-4 w-4 text-destructive" />
                </Button>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </CardContent>
    </Card>

    <!-- New Key Dialog -->
    <Dialog v-model:open="showNewKey">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>API Key 已创建</DialogTitle>
          <DialogDescription>
            ⚠️ 请立即复制并保存此 Key。关闭对话框后将无法再次查看完整 Key。
          </DialogDescription>
        </DialogHeader>
        <div class="flex items-center gap-2 rounded-md border bg-muted p-3">
          <code class="flex-1 text-sm break-all font-mono">{{ newKeyValue }}</code>
          <Button variant="outline" size="icon" @click="copyKey">
            <Check v-if="keyCopied" class="h-4 w-4 text-green-500" />
            <Copy v-else class="h-4 w-4" />
          </Button>
        </div>
        <DialogFooter>
          <DialogClose as-child>
            <Button>我已保存</Button>
          </DialogClose>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Delete Dialog -->
    <Dialog v-model:open="deleteDialogOpen">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>确认删除</DialogTitle>
          <DialogDescription>
            确定要删除此 API Key 吗？使用此 Key 的应用将立即无法访问网关。
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <DialogClose as-child>
            <Button variant="outline">取消</Button>
          </DialogClose>
          <Button variant="destructive" @click="handleDelete" :disabled="deleting">
            {{ deleting ? '删除中...' : '确认删除' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Error Alert Dialog -->
    <ErrorAlertDialog v-model:open="errorDialog.open" :title="errorDialog.title" :message="errorDialog.message" />
  </div>
</template>
