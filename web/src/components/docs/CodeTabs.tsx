import { useEffect, useRef, useState } from 'react'
import { Check, Copy } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

export interface CodeSample {
  value: string
  label: string
  code: string
}

/** 优先使用剪贴板 API，不可用时降级到 execCommand */
async function copyText(text: string): Promise<boolean> {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
      return true
    }
  } catch {
    // 权限被拒绝或非安全上下文时继续走降级方案
  }

  try {
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.setAttribute('readonly', '')
    textarea.className = 'sr-only'
    document.body.appendChild(textarea)
    textarea.select()
    const copied = document.execCommand('copy')
    textarea.remove()
    return copied
  } catch {
    return false
  }
}

interface CodeTabsProps {
  samples: CodeSample[]
  ariaLabel?: string
}

export default function CodeTabs({
  samples,
  ariaLabel = '调用示例',
}: CodeTabsProps) {
  const [value, setValue] = useState(samples[0]?.value ?? '')
  const [copied, setCopied] = useState(false)
  const timerRef = useRef<number | null>(null)

  useEffect(
    () => () => {
      if (timerRef.current !== null) window.clearTimeout(timerRef.current)
    },
    [],
  )

  const active = samples.find((sample) => sample.value === value) ?? samples[0]

  const handleCopy = async () => {
    if (!active) return
    const success = await copyText(active.code)
    if (!success) return

    setCopied(true)
    if (timerRef.current !== null) window.clearTimeout(timerRef.current)
    timerRef.current = window.setTimeout(() => setCopied(false), 2000)
  }

  return (
    <Tabs
      value={value}
      onValueChange={(next) => {
        setValue(next)
        setCopied(false)
      }}
      className="gap-0 overflow-hidden rounded-2xl border bg-marketing-panel"
    >
      <div className="flex items-center gap-2 border-b px-2 py-1.5">
        <div className="min-w-0 flex-1 overflow-x-auto [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
          <TabsList
            variant="line"
            aria-label={ariaLabel}
            className="h-9 w-max min-w-full justify-start gap-1"
          >
            {samples.map((sample) => (
              <TabsTrigger
                key={sample.value}
                value={sample.value}
                className="h-9 flex-none px-3 text-xs after:inset-x-0 after:bottom-0 after:h-0.5 after:bg-primary"
              >
                {sample.label}
              </TabsTrigger>
            ))}
          </TabsList>
        </div>

        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={handleCopy}
          aria-label={
            copied ? '已复制代码' : `复制 ${active?.label ?? ''} 代码`
          }
          className="shrink-0 px-2 text-xs text-muted-foreground hover:text-foreground"
        >
          {copied ? (
            <Check className="size-3.5 text-success" aria-hidden="true" />
          ) : (
            <Copy className="size-3.5" aria-hidden="true" />
          )}
          <span className="hidden sm:inline">{copied ? '已复制' : '复制'}</span>
        </Button>
      </div>

      {samples.map((sample) => (
        <TabsContent key={sample.value} value={sample.value} className="mt-0">
          <pre className="overflow-x-auto px-4 py-4 font-mono text-[13px] leading-6 text-foreground">
            <code>{sample.code}</code>
          </pre>
        </TabsContent>
      ))}
    </Tabs>
  )
}
