<script setup lang="ts">
import { computed, ref, useId } from 'vue'
import { provideFormField } from './useFormField'

const props = defineProps<{
  name?: string
}>()

const inputId = useId()
const name = ref(props.name ?? '')

const error = computed(() => undefined)

const formItemId = `${inputId}-form-item`
const formDescriptionId = `${inputId}-form-item-description`
const formMessageId = `${inputId}-form-item-message`

provideFormField({
  name,
  formItemId,
  formDescriptionId,
  formMessageId,
  isInvalid: computed(() => !!error.value),
  error: computed(() => error.value),
})
</script>

<template>
  <slot />
</template>
