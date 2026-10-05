/**
 * 构建时预渲染脚本：在 vite build 之后执行
 *
 * 1. 将 dist/index.html 的空 <div id="root"></div> 替换为 LandingView 的 SSR 正文
 * 2. 替换字面量 __SITE_URL__ 占位符为 SITE_URL
 * 3. 注入 WebSite / SoftwareApplication / FAQPage 三段 JSON-LD
 * 4. 生成 robots.txt 与 sitemap.xml
 *
 * 幂等：可对已预渲染的 dist/index.html 重复执行（例如带不同 SITE_URL 重跑）。
 * 由 bun 直接执行：`bun scripts/prerender.ts`
 */
import { mkdirSync } from 'node:fs'
import { readFile, writeFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { createElement } from 'react'
import { renderToString } from 'react-dom/server'
// react-router-dom v7 的 package.json exports 不再提供 ./server 子路径，
// StaticRouter 由 react-router 主导出提供（react-router-dom 亦 re-export 之）
import { StaticRouter } from 'react-router'

import { features, faqItems } from '../src/components/landing/content'
import LandingView from '../src/pages/LandingView'

const SITE_URL = (process.env.SITE_URL || 'http://localhost:8080').replace(
  /\/+$/,
  '',
)

// 与 index.html 中 meta[name=description] / og:description 保持一致
const SITE_DESCRIPTION =
  'Xin API 是高性能大模型 API 网关：统一接入 OpenAI、Claude、DeepSeek，提供权重负载均衡、分布式熔断、RPM/TPM 限流、多租户与 API Key 管理，并内置请求审计、Token 用量计费与多维感知智能路由。'

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url))
// 默认处理 dist/；可通过 PRERENDER_DIST_DIR 覆盖（便于以非生产构建做调试）
const DIST_DIR = process.env.PRERENDER_DIST_DIR
  ? resolve(SCRIPT_DIR, '..', process.env.PRERENDER_DIST_DIR)
  : resolve(SCRIPT_DIR, '..', 'dist')
const INDEX_HTML_PATH = resolve(DIST_DIR, 'index.html')
const ROBOTS_PATH = resolve(DIST_DIR, 'robots.txt')
const SITEMAP_PATH = resolve(DIST_DIR, 'sitemap.xml')

const PLACEHOLDER = '__SITE_URL__'

/**
 * 渲染首页为 HTML 字符串。
 * 使用 createElement 而非 JSX，以便本脚本以 .ts 扩展名被 bun 直接执行。
 * 不渲染 <App/>：其自带 RouterProvider 与 Toaster，只需要纯静态首页。
 */
function renderLanding(): string {
  return renderToString(
    createElement(
      StaticRouter,
      { location: '/' },
      createElement(LandingView),
    ),
  )
}

/**
 * 让产物可被重复处理：去掉上一次注入的 JSON-LD，并把上一次的绝对站点 URL
 * 还原为 __SITE_URL__ 占位符（从上一版 canonical 中提取旧 URL）。
 */
function resetPrerenderedHtml(html: string): string {
  let output = html.replace(
    /[ \t]*<script type="application\/ld\+json">[\s\S]*?<\/script>\n?/g,
    '',
  )

  if (!output.includes(PLACEHOLDER)) {
    const canonical = output.match(
      /<link rel="canonical" href="([^"]+?)\/">/,
    )
    if (!canonical || !canonical[1]) {
      throw new Error(
        'dist/index.html 既无 __SITE_URL__ 占位符也无法识别 canonical，无法重建预渲染产物',
      )
    }
    const previousSiteUrl = canonical[1]
    output = output.split(previousSiteUrl).join(PLACEHOLDER)
  }

  return output
}

/**
 * 替换 root 容器：兼容空容器（含内部空白）与已被预渲染填充的容器。
 * 利用后置 </body> 作为锚点（vite 将 bundle script 注入在 head 中），
 * 避免非贪婪匹配在正文内部提前结束。
 */
function injectRootHtml(html: string, appHtml: string): string {
  const rootPattern = /<div id="root">[\s\S]*?<\/div>\s*(?=<\/body>)/i
  if (!rootPattern.test(html)) {
    throw new Error('未在 dist/index.html 中找到 <div id="root"> 容器')
  }
  return html.replace(rootPattern, `<div id="root">${appHtml}</div>\n  `)
}

function buildJsonLdBlocks(): string {
  const websiteLd = {
    '@context': 'https://schema.org',
    '@type': 'WebSite',
    name: 'Xin API',
    url: `${SITE_URL}/`,
    description: SITE_DESCRIPTION,
    inLanguage: 'zh-CN',
  }

  const softwareAppLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: 'Xin API',
    applicationCategory: 'DeveloperApplication',
    operatingSystem: 'Web',
    url: `${SITE_URL}/`,
    description: SITE_DESCRIPTION,
    inLanguage: 'zh-CN',
    featureList: features.map((feature) => feature.title),
  }

  const faqLd = {
    '@context': 'https://schema.org',
    '@type': 'FAQPage',
    mainEntity: faqItems.map((item) => ({
      '@type': 'Question',
      name: item.q,
      acceptedAnswer: {
        '@type': 'Answer',
        text: item.a,
      },
    })),
  }

  return [websiteLd, softwareAppLd, faqLd]
    .map(
      (structured) =>
        `<script type="application/ld+json">${JSON.stringify(
          structured,
          null,
          2,
        )}</script>`,
    )
    .join('\n')
}

function buildRobotsTxt(): string {
  return [
    'User-agent: *',
    'Allow: /',
    'Disallow: /dashboard',
    'Disallow: /groups',
    'Disallow: /channels',
    'Disallow: /apikeys',
    'Disallow: /settings',
    'Disallow: /login',
    '',
    `Sitemap: ${SITE_URL}/sitemap.xml`,
    '', // 末尾换行
  ].join('\n')
}

function escapeXml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;')
}

function buildSitemapXml(): string {
  const now = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  const lastmod = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(
    now.getDate(),
  )}`

  return [
    '<?xml version="1.0" encoding="UTF-8"?>',
    '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">',
    `  <url><loc>${escapeXml(
      SITE_URL,
    )}/</loc><lastmod>${lastmod}</lastmod><changefreq>weekly</changefreq><priority>1.0</priority></url>`,
    '</urlset>',
    '', // 末尾换行
  ].join('\n')
}

async function main(): Promise<void> {
  mkdirSync(DIST_DIR, { recursive: true })

  const rawHtml = await readFile(INDEX_HTML_PATH, 'utf8')
  // 还原为「带占位符的空壳模板」状态，使脚本可幂等重跑
  const templateHtml = resetPrerenderedHtml(rawHtml)

  if (!templateHtml.includes(PLACEHOLDER)) {
    throw new Error('占位符还原失败：dist/index.html 中不存在 __SITE_URL__')
  }

  const appHtml = renderLanding()

  let html = injectRootHtml(templateHtml, appHtml)

  // split/join 做字面量替换，避免 $ 等特殊替换模式
  html = html.split(PLACEHOLDER).join(SITE_URL)

  if (html.includes(PLACEHOLDER)) {
    console.error('预渲染失败：产物中仍存在 __SITE_URL__ 占位符')
    process.exit(1)
  }

  const jsonLdBlocks = buildJsonLdBlocks()
  if (!html.includes('</head>')) {
    throw new Error('dist/index.html 中未找到 </head>，无法注入 JSON-LD')
  }
  html = html.replace('</head>', `${jsonLdBlocks}\n  </head>`)

  await writeFile(INDEX_HTML_PATH, html, 'utf8')
  await writeFile(ROBOTS_PATH, buildRobotsTxt(), 'utf8')
  await writeFile(SITEMAP_PATH, buildSitemapXml(), 'utf8')

  console.log(
    `[prerender] 完成：注入 SSR 正文 ${appHtml.length} 字符、JSON-LD ${jsonLdBlocks.length} 字符；` +
      `SITE_URL=${SITE_URL}；产物：${INDEX_HTML_PATH}、${ROBOTS_PATH}、${SITEMAP_PATH}`,
  )
}

main().catch((error: unknown) => {
  console.error('[prerender] 失败：', error)
  process.exit(1)
})
