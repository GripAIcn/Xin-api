import { inject, provide, type ComputedRef, type InjectionKey, type Ref } from 'vue'

export interface FormFieldContext {
  name: Readonly<Ref<string>>
  formItemId: string
  formDescriptionId: string
  formMessageId: string
  isInvalid: ComputedRef<boolean>
  error: ComputedRef<{ message: string } | undefined>
}

const FORM_ITEM_INJECTION_KEY = Symbol('FormFieldContext') as InjectionKey<FormFieldContext>

export function provideFormField(context: FormFieldContext) {
  provide(FORM_ITEM_INJECTION_KEY, context)
}

export function useFormField() {
  const context = inject(FORM_ITEM_INJECTION_KEY)
  if (!context) {
    throw new Error('useFormField must be used within a FormField')
  }
  return context
}
