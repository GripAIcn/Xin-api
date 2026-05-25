<script setup lang="ts">
import {
  DropdownMenuContent,
  DropdownMenuPortal,
  useForwardProps,
  type DropdownMenuContentProps,
} from 'radix-vue'
import { cn } from '@/lib/utils'

const props = defineProps<DropdownMenuContentProps & { class?: string }>()
const forwarded = useForwardProps(props)
</script>

<template>
  <DropdownMenuPortal>
    <DropdownMenuContent
      v-bind="forwarded"
      :class="cn(
        'z-50 min-w-32 overflow-hidden rounded-md border bg-popover p-1 text-popover-foreground shadow-md',
        props.class,
      )"
    >
      <slot />
    </DropdownMenuContent>
  </DropdownMenuPortal>
</template>
