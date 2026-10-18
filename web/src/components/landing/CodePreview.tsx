import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

const metrics = [
  { label: '142 ms' },
  { label: '27 tokens' },
  { label: '$0.0002' },
]

/** 首屏右侧的请求 / 响应示例面板，数据为静态示例 */
export default function CodePreview() {
  return (
    <div className="flex h-full min-h-[460px] flex-col overflow-hidden rounded-2xl border bg-marketing-panel shadow-sm">
      <div className="flex items-center gap-3 border-b px-4 py-3">
        <span className="rounded-md bg-primary/10 px-2 py-1 font-mono text-[11px] font-semibold text-primary">
          POST
        </span>
        <span className="truncate font-mono text-xs text-muted-foreground">
          api.gripai.cn/v1/chat/completions
        </span>
        <span className="ml-auto flex shrink-0 items-center gap-1.5 text-xs font-medium text-success">
          <span aria-hidden="true" className="size-1.5 rounded-full bg-success" />
          200 OK
        </span>
      </div>

      <Tabs defaultValue="request" className="flex-1 gap-0">
        <TabsList
          variant="line"
          aria-label="请求与响应示例"
          className="h-10 w-full justify-start gap-6 border-b px-4"
        >
          <TabsTrigger
            value="request"
            className="h-full flex-none px-0 text-xs after:inset-x-0 after:bottom-0 after:h-0.5 after:bg-primary"
          >
            Request
          </TabsTrigger>
          <TabsTrigger
            value="response"
            className="h-full flex-none px-0 text-xs after:inset-x-0 after:bottom-0 after:h-0.5 after:bg-primary"
          >
            Response
          </TabsTrigger>
        </TabsList>

        <TabsContent value="request" className="min-h-0">
          <pre className="overflow-x-auto px-4 py-4 font-mono text-xs leading-6">
            <code>
              <span className="text-muted-foreground">$ </span>
              <span className="text-foreground">curl</span>{' '}
              <span className="text-foreground">
                https://api.gripai.cn/v1/chat/completions
              </span>{' '}
              <span className="text-muted-foreground">\</span>
              {'\n  '}
              <span className="text-primary">-H</span>{' '}
              <span className="text-foreground">
                &quot;Authorization: Bearer sk-••••&quot;
              </span>{' '}
              <span className="text-muted-foreground">\</span>
              {'\n  '}
              <span className="text-primary">-H</span>{' '}
              <span className="text-foreground">
                &quot;Content-Type: application/json&quot;
              </span>{' '}
              <span className="text-muted-foreground">\</span>
              {'\n  '}
              <span className="text-primary">-d</span>{' '}
              <span className="text-foreground">&apos;{'{'}</span>
              {'\n    '}
              <span className="text-primary">&quot;model&quot;</span>
              <span className="text-muted-foreground">: </span>
              <span className="text-foreground">&quot;gpt-4o-mini&quot;</span>
              <span className="text-muted-foreground">,</span>
              {'\n    '}
              <span className="text-primary">&quot;messages&quot;</span>
              <span className="text-muted-foreground">: [</span>
              {'\n      '}
              <span className="text-muted-foreground">{'{'}</span>{' '}
              <span className="text-primary">&quot;role&quot;</span>
              <span className="text-muted-foreground">: </span>
              <span className="text-foreground">&quot;user&quot;</span>
              <span className="text-muted-foreground">, </span>
              <span className="text-primary">&quot;content&quot;</span>
              <span className="text-muted-foreground">: </span>
              <span className="text-foreground">
                &quot;你好，介绍一下 Xin API&quot;
              </span>{' '}
              <span className="text-muted-foreground">{'}'}</span>
              {'\n    '}
              <span className="text-muted-foreground">]</span>
              <span className="text-muted-foreground">,</span>
              {'\n    '}
              <span className="text-primary">&quot;stream&quot;</span>
              <span className="text-muted-foreground">: </span>
              <span className="text-foreground">true</span>
              {'\n  '}
              <span className="text-foreground">{'}'}&apos;</span>
            </code>
          </pre>
        </TabsContent>

        <TabsContent value="response" className="min-h-0">
          <pre className="overflow-x-auto px-4 py-4 font-mono text-xs leading-6">
            <code>
              <span className="text-muted-foreground">{'{'}</span>
              {'\n  '}
              <span className="text-primary">&quot;id&quot;</span>
              <span className="text-muted-foreground">: </span>
              <span className="text-foreground">&quot;chatcmpl-9x2k&quot;</span>
              <span className="text-muted-foreground">,</span>
              {'\n  '}
              <span className="text-primary">&quot;object&quot;</span>
              <span className="text-muted-foreground">: </span>
              <span className="text-foreground">&quot;chat.completion&quot;</span>
              <span className="text-muted-foreground">,</span>
              {'\n  '}
              <span className="text-primary">&quot;model&quot;</span>
              <span className="text-muted-foreground">: </span>
              <span className="text-foreground">&quot;gpt-4o-mini&quot;</span>
              <span className="text-muted-foreground">,</span>
              {'\n  '}
              <span className="text-primary">&quot;choices&quot;</span>
              <span className="text-muted-foreground">: [</span>
              {'\n    '}
              <span className="text-muted-foreground">{'{'}</span>
              {'\n      '}
              <span className="text-primary">&quot;index&quot;</span>
              <span className="text-muted-foreground">: </span>
              <span className="text-foreground">0</span>
              <span className="text-muted-foreground">,</span>
              {'\n      '}
              <span className="text-primary">&quot;message&quot;</span>
              <span className="text-muted-foreground">: {'{'}</span>
              {'\n        '}
              <span className="text-primary">&quot;role&quot;</span>
              <span className="text-muted-foreground">: </span>
              <span className="text-foreground">&quot;assistant&quot;</span>
              <span className="text-muted-foreground">,</span>
              {'\n        '}
              <span className="text-primary">&quot;content&quot;</span>
              <span className="text-muted-foreground">: </span>
              <span className="text-foreground">
                &quot;Xin API 是统一的大模型 API 网关……&quot;
              </span>
              {'\n      '}
              <span className="text-muted-foreground">{'}'},</span>
              {'\n      '}
              <span className="text-primary">&quot;finish_reason&quot;</span>
              <span className="text-muted-foreground">: </span>
              <span className="text-foreground">&quot;stop&quot;</span>
              {'\n    '}
              <span className="text-muted-foreground">{'}'}</span>
              {'\n  '}
              <span className="text-muted-foreground">],</span>
              {'\n  '}
              <span className="text-primary">&quot;usage&quot;</span>
              <span className="text-muted-foreground">: {'{'}</span>
              {'\n    '}
              <span className="text-primary">&quot;prompt_tokens&quot;</span>
              <span className="text-muted-foreground">: </span>
              <span className="text-foreground">12</span>
              <span className="text-muted-foreground">,</span>
              {'\n    '}
              <span className="text-primary">&quot;completion_tokens&quot;</span>
              <span className="text-muted-foreground">: </span>
              <span className="text-foreground">15</span>
              <span className="text-muted-foreground">,</span>
              {'\n    '}
              <span className="text-primary">&quot;total_tokens&quot;</span>
              <span className="text-muted-foreground">: </span>
              <span className="text-foreground">27</span>
              {'\n  '}
              <span className="text-muted-foreground">{'}'}</span>
              {'\n'}
              <span className="text-muted-foreground">{'}'}</span>
            </code>
          </pre>
        </TabsContent>
      </Tabs>

      <div className="mt-auto flex flex-wrap items-center gap-2 border-t px-4 py-3">
        {metrics.map((metric) => (
          <span
            key={metric.label}
            className="rounded-full border bg-background px-2.5 py-1 font-mono text-[11px] tabular-nums text-muted-foreground"
          >
            {metric.label}
          </span>
        ))}
        <span className="ml-auto text-[11px] text-muted-foreground">
          示例数据
        </span>
      </div>
    </div>
  )
}
