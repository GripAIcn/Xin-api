import { useEffect, useState } from 'react'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { createPrice, updatePrice } from '@/api/modules/prices'
import type { ModelPrice } from '@/types/api'

// 单价字段：非空且为不小于 0 的数字
const priceField = (message: string) =>
  z
    .string()
    .min(1, message)
    .refine(
      (v) => Number.isFinite(Number(v)) && Number(v) >= 0,
      '请输入不小于 0 的数字',
    )

const priceFormSchema = z.object({
  name: z.string().min(1, '请输入方案名'),
  input_price: priceField('请输入输入单价'),
  output_price: priceField('请输入输出单价'),
  cached_input_price: priceField('请输入缓存命中单价'),
  cache_creation_price: priceField('请输入缓存创建单价'),
  currency: z.enum(['USD', 'CNY'], { message: '请选择币种' }),
  status: z.enum(['active', 'disabled']),
})

type PriceFormValues = z.infer<typeof priceFormSchema>

const EMPTY_FORM: PriceFormValues = {
  name: '',
  input_price: '0',
  output_price: '0',
  cached_input_price: '0',
  cache_creation_price: '0',
  currency: 'USD',
  status: 'active',
}

// 编辑态币种统一转大写，非法值兜底 USD
const normalizeCurrency = (value: string): PriceFormValues['currency'] => {
  const upper = value.toUpperCase()
  return upper === 'CNY' ? 'CNY' : 'USD'
}

interface PriceFormSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** 编辑态：被编辑的价格方案；新增态传 null */
  editing: ModelPrice | null
  /** 保存成功后回调（用于刷新列表） */
  onSuccess: () => void
}

export default function PriceFormSheet({
  open,
  onOpenChange,
  editing,
  onSuccess,
}: PriceFormSheetProps) {
  const [submitting, setSubmitting] = useState(false)
  const isEdit = editing !== null

  const form = useForm<PriceFormValues>({
    resolver: zodResolver(priceFormSchema),
    defaultValues: EMPTY_FORM,
  })

  // 打开时按编辑态填充表单初值
  useEffect(() => {
    if (!open) return
    if (editing) {
      form.reset({
        name: editing.name,
        input_price: String(editing.input_price),
        output_price: String(editing.output_price),
        cached_input_price: String(editing.cached_input_price),
        cache_creation_price: String(editing.cache_creation_price),
        currency: normalizeCurrency(editing.currency),
        status: editing.status,
      })
    } else {
      form.reset(EMPTY_FORM)
    }
  }, [open, editing, form])

  const onSubmit = async (values: PriceFormValues) => {
    setSubmitting(true)
    try {
      const base = {
        name: values.name.trim(),
        input_price: Number(values.input_price),
        output_price: Number(values.output_price),
        cached_input_price: Number(values.cached_input_price),
        cache_creation_price: Number(values.cache_creation_price),
        currency: values.currency,
      }
      if (editing) {
        await updatePrice(editing.id, { ...base, status: values.status })
        toast.success('价格方案更新成功')
      } else {
        await createPrice(base)
        toast.success('价格方案创建成功')
      }
      onOpenChange(false)
      onSuccess()
    } catch (e) {
      toast.error(e instanceof Error && e.message ? e.message : '保存失败')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Sheet
      open={open}
      onOpenChange={(value) => {
        onOpenChange(value)
      }}
    >
      <SheetContent className="overflow-y-auto sm:max-w-xl">
        <SheetHeader>
          <SheetTitle>{isEdit ? '编辑价格方案' : '新增价格方案'}</SheetTitle>
          <SheetDescription className="sr-only">
            填写价格方案信息
          </SheetDescription>
        </SheetHeader>

        <Form {...form}>
          <form
            onSubmit={(event) => {
              void form.handleSubmit(onSubmit)(event)
            }}
            className="space-y-4 px-4 pb-4"
          >
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>方案名</FormLabel>
                  <FormControl>
                    <Input placeholder="例如：GPT-4o 标准价" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div className="grid grid-cols-2 gap-4">
              <FormField
                control={form.control}
                name="input_price"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>输入单价</FormLabel>
                    <FormControl>
                      <Input inputMode="decimal" placeholder="≥ 0" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="output_price"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>输出单价</FormLabel>
                    <FormControl>
                      <Input inputMode="decimal" placeholder="≥ 0" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="cached_input_price"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>缓存命中单价</FormLabel>
                    <FormControl>
                      <Input inputMode="decimal" placeholder="≥ 0" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="cache_creation_price"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>缓存创建单价</FormLabel>
                    <FormControl>
                      <Input inputMode="decimal" placeholder="≥ 0" {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <FormField
              control={form.control}
              name="currency"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>币种</FormLabel>
                  <Select value={field.value} onValueChange={field.onChange}>
                    <FormControl>
                      <SelectTrigger className="w-full">
                        <SelectValue placeholder="请选择币种" />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="USD">USD</SelectItem>
                      <SelectItem value="CNY">CNY</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormMessage />
                </FormItem>
              )}
            />

            {isEdit && (
              <FormField
                control={form.control}
                name="status"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>状态</FormLabel>
                    <Select value={field.value} onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger className="w-full">
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="active">启用</SelectItem>
                        <SelectItem value="disabled">禁用</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}

            <p className="text-xs text-muted-foreground">
              单价单位为「币种 / 1M tokens」；USD 定价的消费会按管理端配置的汇率折算为 CNY 扣减余额。
            </p>

            <SheetFooter className="p-0 sm:flex-row sm:justify-end">
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
              >
                取消
              </Button>
              <Button type="submit" disabled={submitting}>
                {submitting ? '保存中...' : '保存'}
              </Button>
            </SheetFooter>
          </form>
        </Form>
      </SheetContent>
    </Sheet>
  )
}
