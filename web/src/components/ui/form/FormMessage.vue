<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/lib/utils'
import { useFormField } from './useFormField'

const props = defineProps<{
  class?: string
}>()

const { error, formMessageId } = useFormField()

const message = computed(() => error?.value?.message ?? '')
</script>

<template>
  <p
    v-if="message"
    :id="formMessageId"
    :class="cn('text-sm text-destructive', props.class)"
  >
    {{ message }}
  </p>
</template>
