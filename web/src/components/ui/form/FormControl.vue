<script setup lang="ts">
import { Slot } from 'radix-vue'
import { useFormField } from './useFormField'

const { error, formItemId, formDescriptionId, formMessageId, isInvalid } = useFormField()
</script>

<template>
  <Slot
    :id="formItemId"
    :aria-describedby="!error ? formDescriptionId : `${formDescriptionId} ${formMessageId}`"
    :aria-invalid="isInvalid"
  >
    <slot />
  </Slot>
</template>
