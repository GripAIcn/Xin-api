<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useGroupStore } from '@/stores/groups'
import { useChannelStore } from '@/stores/channels'
import { useApiKeyStore } from '@/stores/apikeys'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import {
  Card, CardHeader, CardTitle, CardDescription, CardContent,
} from '@/components/ui/card'
import {
  Table, TableHeader, TableBody, TableRow, TableHead, TableCell,
} from '@/components/ui/table'
import {
  Dialog, DialogTrigger, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose,
} from '@/components/ui/dialog'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import { toast } from 'vue-sonner'
import { ArrowLeft, Plus, Pencil, Trash2, Copy, Check, AlertTriangle } from 'lucide-vue-next'

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
const showNewKey = ref(false)
const keyCopied = ref(false)

// Channel CRUD
const channelDialogOpen = ref(false)
const editingChannel = ref<any>(null)
const channelForm = ref({ name: '', model_mapping: '', base_url: '', api_key: '', weight: 1 })
const channelSubmitting = ref(false)
const deleteChannelDialogOpen = ref(false)
const deletingChannel = ref<any>(null)

// API Key CRUD
const apiKeySubmitting = ref(false)
const deleteKeyDialogOpen = ref(false)
const deletingKey = ref<any>(null)

onMounted(async () => {
  try {
    await Promise.all([
      groupStore.fetchAll(),
      channelStore.fetchByGroup(groupId.value),
      apiKeyStore.fetchAll(groupId.value),
    ])
  } catch (e: any) {
    // 只在真正出错时提示，空数据不提示
    if (e?.message && e.message !== 'Response data is empty') {
      toast.error('加载数据失败')
    }
  } finally {
    loading.value = false
  }
})

// === Channel Handlers ===
const openCreateChannel = () => {
  editingChannel.value = null
  channelForm.value = { name: '', model_mapping: '', base_url: '', api_key: '', weight: 1 }
  channelDialogOpen.value = true
}

const openEditChannel = (ch: any) => {
  editingChannel.value = ch
  channelForm.value = {
    name: ch.name,
    model_mapping: ch.model_mapping,
    base_url: ch.base_url,
    api_key: ch.api_key,
    weight: ch.weight,
  }
  channelDialogOpen.value = true
}

const handleSaveChannel = async () => {
  const f = channelForm.value
  if (!f.name || f.name.length < 2) { toast.error('渠道名称至少 2 个字符'); return }
  if (!f.model_mapping) { toast.error('请输入模型映射'); return }
  if (!f.base_url) { toast.error('请输入 Base URL'); return }
  if (!f.api_key) { toast.error('请输入 API Key'); return }
  if (!f.weight || f.weight < 1 || f.weight > 100) { toast.error('权重范围 1-100'); return }

  channelSubmitting.value = true
  try {
    if (editingChannel.value) {
      await channelStore.update(editingChannel.value.id, {
        group_id: groupId.value,
        ...f,
        status: editingChannel.value.status,
      })
      toast.success('渠道更新成功')
    } else {
      await channelStore.create({ group_id: groupId.value, ...f })
      toast.success('渠道创建成功')
    }
    channelDialogOpen.value = false
  } catch (e: any) {
    toast.error(e.message || '操作失败')
  } finally {
    channelSubmitting.value = false
  }
}

const openDeleteChannel = (ch: any) => {
  deletingChannel.value = ch
  deleteChannelDialogOpen.value = true
}

const handleDeleteChannel = async () => {
  channelSubmitting.value = true
  try {
    await channelStore.remove(deletingChannel.value.id)
    toast.success('渠道已删除')
    deleteChannelDialogOpen.value = false
  } catch (e: any) {
    toast.error(e.message || '删除失败')
  } finally {
    channelSubmitting.value = false
  }
}

// === API Key Handlers ===
const handleCreateApiKey = async () => {
  apiKeySubmitting.value = true
  try {
    const result = await apiKeyStore.create(groupId.value)
    newKeyValue.value = result.key
    showNewKey.value = true
  } catch (e: any) {
    toast.error(e.message || '创建失败')
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
    toast.error('复制失败')
  }
}

const openDeleteKey = (key: any) => {
  deletingKey.value = key
  deleteKeyDialogOpen.value = true
}

const handleDeleteKey = async () => {
  apiKeySubmitting.value = true
  try {
    await apiKeyStore.remove(deletingKey.value.key)
    toast.success('API Key 已删除')
    deleteKeyDialogOpen.value = false
  } catch (e: any) {
    toast.error(e.message || '删除失败')
  } finally {
    apiKeySubmitting.value = false
  }
}

const statusBadge = (status: number) => {
  if (status === 1) return { label: '正常', variant: 'default' as const }
  if (status === 2) return { label: '熔断', variant: 'destructive' as const }
  return { label: '禁用', variant: 'secondary' as const }
}

const maskKey = (key: string) => {
  if (key.length > 12) return key.slice(0, 8) + '****' + key.slice(-4)
  return key
}
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center gap-4">
      <Button variant="ghost" size="icon" @click="router.push('/groups')">
        <ArrowLeft class="h-4 w-4" />
      </Button>
      <div v-if="group">
        <div class="flex items-center gap-2">
          <h1 class="text-2xl font-bold tracking-tight">{{ group.name }}</h1>
          <Badge :variant="group.status === 1 ? 'default' : 'secondary'">
            {{ group.status === 1 ? '启用' : '停用' }}
          </Badge>
        </div>
        <p class="text-sm text-muted-foreground">项目组 ID: {{ group.id }}</p>
      </div>
      <Skeleton v-else class="h-8 w-48" />
    </div>

    <!-- Tabs -->
    <div class="flex gap-1 border-b">
      <button
        @click="activeTab = 'channels'"
        class="px-4 py-2 text-sm font-medium transition-colors"
        :class="activeTab === 'channels' ? 'border-b-2 border-primary text-primary' : 'text-muted-foreground hover:text-foreground'"
      >渠道 ({{ channelStore.channels.length }})</button>
      <button
        @click="activeTab = 'apikeys'"
        class="px-4 py-2 text-sm font-medium transition-colors"
        :class="activeTab === 'apikeys' ? 'border-b-2 border-primary text-primary' : 'text-muted-foreground hover:text-foreground'"
      >API Key ({{ apiKeyStore.apiKeys.length }})</button>
    </div>

    <!-- Channels Tab -->
    <div v-if="activeTab === 'channels'">
      <div class="flex items-center justify-between mb-4">
        <p class="text-sm text-muted-foreground">管理此项目组下的上游渠道</p>
        <Button size="sm" @click="openCreateChannel">
          <Plus class="h-4 w-4 mr-1" />添加渠道
        </Button>
      </div>

      <Card>
        <CardContent class="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>名称</TableHead>
                <TableHead>模型映射</TableHead>
                <TableHead>Base URL</TableHead>
                <TableHead>权重</TableHead>
                <TableHead>状态</TableHead>
                <TableHead class="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="channelStore.channels.length === 0">
                <TableCell colspan="6" class="text-center text-muted-foreground py-8">暂无渠道</TableCell>
              </TableRow>
              <TableRow v-for="ch in channelStore.channels" :key="ch.id">
                <TableCell class="font-medium">{{ ch.name }}</TableCell>
                <TableCell class="font-mono text-sm">{{ ch.model_mapping }}</TableCell>
                <TableCell class="font-mono text-sm max-w-[200px] truncate">{{ ch.base_url }}</TableCell>
                <TableCell>{{ ch.weight }}</TableCell>
                <TableCell>
                  <Badge :variant="statusBadge(ch.status).variant">{{ statusBadge(ch.status).label }}</Badge>
                </TableCell>
                <TableCell class="text-right">
                  <div class="flex justify-end gap-1">
                    <Button variant="ghost" size="icon" @click="openEditChannel(ch)">
                      <Pencil class="h-4 w-4" />
                    </Button>
                    <Button variant="ghost" size="icon" @click="openDeleteChannel(ch)">
                      <Trash2 class="h-4 w-4 text-destructive" />
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>

    <!-- API Keys Tab -->
    <div v-if="activeTab === 'apikeys'">
      <div class="flex items-center justify-between mb-4">
        <p class="text-sm text-muted-foreground">管理此项目组的 API Key</p>
        <Button size="sm" @click="handleCreateApiKey" :disabled="apiKeySubmitting">
          <Plus class="h-4 w-4 mr-1" />{{ apiKeySubmitting ? '创建中...' : '创建 Key' }}
        </Button>
      </div>

      <Card>
        <CardContent class="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>API Key</TableHead>
                <TableHead>创建时间</TableHead>
                <TableHead class="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-if="apiKeyStore.apiKeys.length === 0">
                <TableCell colspan="3" class="text-center text-muted-foreground py-8">暂无 API Key</TableCell>
              </TableRow>
              <TableRow v-for="ak in apiKeyStore.apiKeys" :key="ak.key">
                <TableCell class="font-mono text-sm">{{ maskKey(ak.key) }}</TableCell>
                <TableCell class="text-muted-foreground">{{ ak.created_at?.slice(0, 19).replace('T', ' ') }}</TableCell>
                <TableCell class="text-right">
                  <Button variant="ghost" size="icon" @click="openDeleteKey(ak)">
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
              请立即复制并保存此 Key，关闭后将无法再次查看完整 Key。
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

      <!-- Delete Key Dialog -->
      <Dialog v-model:open="deleteKeyDialogOpen">
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
            <Button variant="destructive" @click="handleDeleteKey" :disabled="apiKeySubmitting">
              {{ apiKeySubmitting ? '删除中...' : '确认删除' }}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>

    <!-- Channel Form Dialog -->
    <Dialog v-model:open="channelDialogOpen">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ editingChannel ? '编辑渠道' : '添加渠道' }}</DialogTitle>
          <DialogDescription>配置上游渠道的连接信息</DialogDescription>
        </DialogHeader>
        <form @submit.prevent="handleSaveChannel">
          <div class="space-y-3 py-2">
            <div class="space-y-1.5">
              <Label>渠道名称</Label>
              <Input v-model="channelForm.name" placeholder="例如：OpenAI 主通道" />
            </div>
            <div class="space-y-1.5">
              <Label>模型映射</Label>
              <Input v-model="channelForm.model_mapping" placeholder="例如：gpt-4o,gpt-4o-mini" />
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
            <Button type="submit" :disabled="channelSubmitting">
              {{ channelSubmitting ? '保存中...' : '保存' }}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <!-- Delete Channel Dialog -->
    <Dialog v-model:open="deleteChannelDialogOpen">
      <DialogContent>
        <DialogHeader>
          <DialogTitle>确认删除</DialogTitle>
          <DialogDescription>确定要删除渠道「{{ deletingChannel?.name }}」吗？</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <DialogClose as-child>
            <Button variant="outline">取消</Button>
          </DialogClose>
          <Button variant="destructive" @click="handleDeleteChannel" :disabled="channelSubmitting">
            {{ channelSubmitting ? '删除中...' : '确认删除' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
