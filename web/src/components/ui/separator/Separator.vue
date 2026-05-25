<script setup lang="ts">
import { Separator, type SeparatorProps, useForwardProps } from 'radix-vue'
import { cn } from '@/lib/utils'

const props = defineProps<SeparatorProps & { class?: string }>()
const forwarded = useForwardProps(props)
</script>

<template>
  <Separator
    v-bind="forwarded"
    :class="cn(
      'shrink-0 bg-border',
      props.orientation === 'horizontal' ? 'h-[1px] w-full' : 'h-full w-[1px]',
      props.class,
    )"
  />
</template>
