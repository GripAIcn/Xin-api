import type { ReactNode } from 'react'
import { Info } from 'lucide-react'

import CodeTabs, { type CodeSample } from '@/components/docs/CodeTabs'
import LandingFooter from '@/components/landing/LandingFooter'
import LandingHeader from '@/components/landing/LandingHeader'

const DOC_SECTIONS = [
  { id: 'get-api-key', title: '获取 API Key' },
  { id: 'first-request', title: '发起第一个请求' },
  { id: 'response-example', title: '响应示例' },
  { id: 'errors', title: '常见错误' },
]

const CODE_SAMPLES: CodeSample[] = [
  {
    value: 'curl',
    label: 'cURL',
    code: `curl https://api.gripai.cn/v1/chat/completions \\
  -H "Authorization: Bearer sk-..." \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "你好，介绍一下你自己"}]
  }'`,
  },
  {
    value: 'python',
    label: 'Python',
    code: `from openai import OpenAI

client = OpenAI(api_key="sk-...", base_url="https://api.gripai.cn/v1")

resp = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "你好，介绍一下你自己"}],
)
print(resp.choices[0].message.content)`,
  },
  {
    value: 'node',
    label: 'Node.js',
    code: `import OpenAI from 'openai'

const client = new OpenAI({ apiKey: 'sk-...', baseURL: 'https://api.gripai.cn/v1' })

const resp = await client.chat.completions.create({
  model: 'gpt-4o',
  messages: [{ role: 'user', content: '你好，介绍一下你自己' }],
})
console.log(resp.choices[0].message.content)`,
  },
  {
    value: 'go',
    label: 'Go',
    code: `package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

func main() {
    body, _ := json.Marshal(map[string]any{
        "model": "gpt-4o",
        "messages": []map[string]string{
            {"role": "user", "content": "你好，介绍一下你自己"},
        },
    })

    req, _ := http.NewRequest("POST", "https://api.gripai.cn/v1/chat/completions", bytes.NewReader(body))
    req.Header.Set("Authorization", "Bearer sk-...")
    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    data, _ := io.ReadAll(resp.Body)
    fmt.Println(string(data))
}`,
  },
  {
    value: 'java',
    label: 'Java',
    code: `import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;

public class Main {
    public static void main(String[] args) throws Exception {
        String body = """
            {
              "model": "gpt-4o",
              "messages": [{"role": "user", "content": "你好，介绍一下你自己"}]
            }
            """;

        HttpRequest request = HttpRequest.newBuilder()
            .uri(URI.create("https://api.gripai.cn/v1/chat/completions"))
            .header("Authorization", "Bearer sk-...")
            .header("Content-Type", "application/json")
            .POST(HttpRequest.BodyPublishers.ofString(body))
            .build();

        HttpResponse<String> response = HttpClient.newHttpClient()
            .send(request, HttpResponse.BodyHandlers.ofString());
        System.out.println(response.body());
    }
}`,
  },
]

const RESPONSE_EXAMPLE = `{
  "id": "chatcmpl-9x2kQ7f1",
  "object": "chat.completion",
  "created": 1760000000,
  "model": "gpt-4o",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "你好，我是通过 Xin API 接入的模型，很高兴为你服务。"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 18,
    "completion_tokens": 42,
    "total_tokens": 60
  }
}`

const ERROR_ITEMS = [
  {
    code: '401',
    title: '密钥无效',
    detail:
      'Authorization 头缺少 Bearer 前缀、密钥已被删除或复制不完整。请到控制台重新创建密钥，并确认请求头格式为 Authorization: Bearer sk-...。',
  },
  {
    code: '404',
    title: 'Base URL 少了 /v1',
    detail:
      '请求地址应为 https://api.gripai.cn/v1/chat/completions，注意保留路径中的 /v1。',
  },
  {
    code: '429',
    title: 'RPM / TPM 超限',
    detail:
      '已达到密钥或模型的请求数 / Token 速率限制，降低并发、缩短上下文或稍后重试。',
  },
  {
    code: '4xx',
    title: '模型不存在',
    detail:
      '请求的模型名未在控制台登记为对外模型，请使用控制台「模型管理」中的对外模型名。',
  },
]

function DocSection({
  id,
  title,
  children,
}: {
  id: string
  title: string
  children: ReactNode
}) {
  return (
    <section id={id} className="scroll-mt-24">
      <h2 className="text-xl font-semibold tracking-tight md:text-2xl">
        {title}
      </h2>
      <div className="mt-5 space-y-4 text-sm leading-7 text-muted-foreground">
        {children}
      </div>
    </section>
  )
}

export default function DocsView() {
  return (
    <>
      <LandingHeader />
      <main className="min-h-screen bg-background text-foreground">
        <div className="border-b">
          <div className="mx-auto w-full max-w-[1440px] px-6 py-14 lg:px-10 lg:py-20">
            <p className="text-sm font-semibold text-primary">使用文档</p>
            <h1 className="mt-3 text-3xl font-semibold tracking-tight md:text-4xl">
              快速开始
            </h1>
            <p className="mt-4 max-w-2xl text-base leading-7 text-muted-foreground">
              几分钟内完成第一次调用：创建 API Key、配置 Base URL，然后发送一个 OpenAI
              兼容的 Chat Completions 请求。
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1440px] px-6 lg:px-10">
          <div className="grid gap-10 lg:grid-cols-[220px_minmax(0,1fr)] lg:gap-16">
            <aside className="hidden lg:block">
              <nav aria-label="本页目录" className="sticky top-20">
                <p className="text-xs font-semibold tracking-wider text-muted-foreground">
                  本页内容
                </p>
                <ul className="mt-4 border-l">
                  {DOC_SECTIONS.map((section) => (
                    <li key={section.id}>
                      <a
                        href={`#${section.id}`}
                        className="-ml-px flex min-h-9 items-center border-l border-transparent pl-4 text-sm text-muted-foreground transition-colors hover:border-primary hover:text-foreground"
                      >
                        {section.title}
                      </a>
                    </li>
                  ))}
                </ul>
              </nav>
            </aside>

            <div className="min-w-0 max-w-3xl py-12 lg:py-16">
              <nav
                aria-label="本页目录"
                className="-mx-6 overflow-x-auto px-6 [scrollbar-width:none] lg:hidden [&::-webkit-scrollbar]:hidden"
              >
                <ul className="flex w-max gap-2">
                  {DOC_SECTIONS.map((section) => (
                    <li key={section.id}>
                      <a
                        href={`#${section.id}`}
                        className="inline-flex h-8 items-center rounded-full border bg-card px-3 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground"
                      >
                        {section.title}
                      </a>
                    </li>
                  ))}
                </ul>
              </nav>

              <div className="mt-10 space-y-12 lg:mt-0 lg:space-y-14">
                <DocSection id="get-api-key" title="获取 API Key">
                  <p>
                    先登录控制台，创建一个仅用于当前应用的密钥。
                  </p>
                  <ol className="list-decimal space-y-1 pl-5">
                    <li>打开控制台，进入「API Keys」页面。</li>
                    <li>点击「新建密钥」，填写用途名称后保存。</li>
                    <li>复制生成的密钥，并保存到安全的位置。</li>
                  </ol>
                  <p>
                    密钥仅在创建时完整展示一次，离开页面后无法再次查看；如遗失请重新创建。
                  </p>
                  <div className="rounded-2xl border bg-marketing-panel px-4 py-3">
                    <p className="text-xs font-medium">Base URL</p>
                    <code className="mt-1 block font-mono text-sm text-foreground">
                      https://api.gripai.cn/v1
                    </code>
                    <p className="mt-1 text-xs">
                      所有接口都挂在该域名下，路径中的 /v1 不能省略。
                    </p>
                  </div>
                </DocSection>

                <DocSection id="first-request" title="发起第一个请求">
                  <p>
                    所有请求都发往 POST /v1/chat/completions，请求体与 OpenAI Chat
                    Completions 接口保持一致。下面的示例统一使用占位密钥 sk-... 与模型
                    gpt-4o。
                  </p>
                  <div className="mt-6">
                    <CodeTabs samples={CODE_SAMPLES} ariaLabel="调用示例语言" />
                  </div>
                  <p className="text-xs">
                    Python 与 Node.js 示例使用官方 OpenAI SDK，将 base_url 指向 /v1
                    即可；Go 与 Java 使用标准库，无需额外依赖。
                  </p>
                  <p className="text-xs">
                    实际可用的模型名以控制台「模型管理」中登记的对外模型名为准，示例中的
                    gpt-4o 仅作演示。
                  </p>
                </DocSection>

                <DocSection id="response-example" title="响应示例">
                  <p>
                    调用成功后返回标准 Chat Completions 结构，usage 中给出本次请求的
                    Token 用量：
                  </p>
                  <pre className="overflow-x-auto rounded-2xl border bg-marketing-panel px-4 py-4 font-mono text-[13px] leading-6 text-foreground">
                    <code>{RESPONSE_EXAMPLE}</code>
                  </pre>
                  <div className="flex gap-3 rounded-2xl border bg-marketing-panel px-4 py-3">
                    <Info
                      aria-hidden="true"
                      className="mt-1 size-4 shrink-0 text-primary"
                    />
                    <div>
                      <p className="font-medium text-foreground">X-Request-Id</p>
                      <p>
                        响应头中的 X-Request-Id 形如 xin_req_9f2c...，用于对账与排查。提交问题时附上该
                        ID，可以定位到完整的请求链路。
                      </p>
                    </div>
                  </div>
                </DocSection>

                <DocSection id="errors" title="常见错误">
                  <p>
                    接入初期最常见的问题集中在密钥、Base URL、限流与模型名四处：
                  </p>
                  <dl className="divide-y divide-border overflow-hidden rounded-2xl border bg-card">
                    {ERROR_ITEMS.map((item) => (
                      <div
                        key={item.title}
                        className="flex flex-col gap-2 px-4 py-4 sm:flex-row sm:gap-6"
                      >
                        <dt className="w-12 shrink-0 font-mono text-sm font-semibold text-primary">
                          {item.code}
                        </dt>
                        <dd>
                          <p className="font-medium text-foreground">
                            {item.title}
                          </p>
                          <p className="mt-1 text-sm leading-6">{item.detail}</p>
                        </dd>
                      </div>
                    ))}
                  </dl>
                </DocSection>
              </div>
            </div>
          </div>
        </div>
      </main>
      <LandingFooter />
    </>
  )
}
