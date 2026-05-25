<script setup lang="ts">
import { ref, watch } from 'vue'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { AlertCircle } from 'lucide-vue-next'

const props = defineProps<{
  open: boolean
  title?: string
  message: string
  duration?: number
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
}>()

const isOpen = ref(props.open)
let timeoutId: ReturnType<typeof setTimeout> | null = null

watch(() => props.open, (newVal) => {
  isOpen.value = newVal
  if (newVal) {
    // 清除之前的定时器
    if (timeoutId) {
      clearTimeout(timeoutId)
    }
    // 设置自动关闭
    const duration = props.duration || 3000
    timeoutId = setTimeout(() => {
      isOpen.value = false
      emit('update:open', false)
    }, duration)
  }
})

watch(isOpen, (newVal) => {
  emit('update:open', newVal)
})
</script>

<template>
  <Dialog v-model:open="isOpen">
    <DialogContent class="sm:max-w-md border-red-200 dark:border-red-800">
      <DialogHeader class="flex flex-col items-center text-center">
        <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-red-100 dark:bg-red-900">
          <AlertCircle class="h-6 w-6 text-red-600 dark:text-red-400" />
        </div>
        <DialogTitle class="text-lg font-semibold text-red-600 dark:text-red-400">
          {{ title || '提示' }}
        </DialogTitle>
        <DialogDescription class="mt-2 text-sm text-gray-600 dark:text-gray-400">
          {{ message }}
        </DialogDescription>
      </DialogHeader>
    </DialogContent>
  </Dialog>
</template>
