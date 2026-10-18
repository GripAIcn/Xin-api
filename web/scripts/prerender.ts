/**
 * 构建时预渲染脚本：在 vite build 之后执行
 *
 * 1. 将 dist/index.html 的空 <div id="root"></div> 替换为 LandingView 的 SSR 正文
 * 2. 替换字面量 __SITE_URL__ 占位符为 SITE_URL
 * 3. 注入 WebSite / SoftwareApplication / FAQPage 三段 JSON-LD
 * 4. 生成 robots.txt 与 sitemap.xml
 *
 * 幂等：可对已预渲染的 dist/index.html 重复执行（例如带不同 SITE_URL 重跑）。
 * 由 tsx 执行：`npm run build`（或 `npx tsx scripts/prerender.ts`）
 */
import { mkdirSync } from 'node:fs'
import { readFile, writeFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { StrictMode, createElement } from 'react'
import { renderToString } from 'react-dom/server'
import { createMemoryRouter } from 'react-router-dom'
// 与客户端 main.tsx 复用同一个 ThemeProvider；它只维护主题状态、不输出任何 DOM 节点，
// 因此 SSR 树与客户端树天然一致（不再需要在树内渲染主题引导 <script>）
import { ThemeProvider } from '../src/components/theme-provider'

import {
  coreCapabilities,
  faqItems,
  moreCapabilities,
} from '../src/components/landing/content'
import App from '../src/App'
import { publicRoutes } from '../src/routes/public'

const RAW_SITE_URL = process.env.SITE_URL
// 缺省值即正式站点地址：canonical / og:url / og:image / sitemap / robots 全部由它拼接
const SITE_URL = (RAW_SITE_URL || 'https://api.gripai.cn').replace(/\/+$/, '')

// 防呆：一旦 SITE_URL 是 localhost，整站 canonical 会指向本机地址（线上已发生过一次）。
// 显式传入 localhost 视为误配置直接拦截；本地调试确需 localhost 时设置 ALLOW_LOCALHOST_SITE_URL=1。
if (
  /^(https?:\/\/)?(localhost|127\.0\.0\.1)/i.test(SITE_URL) &&
  process.env.ALLOW_LOCALHOST_SITE_URL !== '1'
) {
  throw new Error(
    `prerender 终止：SITE_URL 不能为 ${SITE_URL}。请在构建时注入正式地址，` +
      '例如 SITE_URL=https://api.gripai.cn；本地调试确需 localhost 时设置 ALLOW_LOCALHOST_SITE_URL=1。',
  )
}
if (!RAW_SITE_URL) {
  console.warn(
    `[prerender] 未注入 SITE_URL，已回落到默认站点地址 ${SITE_URL}；如需覆盖请设置环境变量 SITE_URL。`,
  )
}

// 与 index.html 中 meta[name=description] / og:description 保持一致
const SITE_DESCRIPTION =
  'Xin API 是高性能大模型 API 网关：统一接入 DeepSeek、Kimi、GLM、Qwen，提供权重负载均衡、分布式熔断、RPM/TPM 限流与 API Key 管理，并内置请求审计、Token 用量计费与智能路由。'

const SITE_NAME = 'Xin API'

const SCRIPT_DIR = dirname(fileURLToPath(import.meta.url))
// 默认处理 dist/；可通过 PRERENDER_DIST_DIR 覆盖（便于以非生产构建做调试）
const DIST_DIR = process.env.PRERENDER_DIST_DIR
  ? resolve(SCRIPT_DIR, '..', process.env.PRERENDER_DIST_DIR)
  : resolve(SCRIPT_DIR, '..', 'dist')
const INDEX_HTML_PATH = resolve(DIST_DIR, 'index.html')
const ROBOTS_PATH = resolve(DIST_DIR, 'robots.txt')
const SITEMAP_PATH = resolve(DIST_DIR, 'sitemap.xml')

// 与 /docs 页面内容对应，供静态产物与客户端 useEffect 设置保持一致
const DOCS_TITLE = '使用文档 - Xin API'
const DOCS_DESCRIPTION =
  'Xin API 使用文档：获取 API Key、发起第一个 OpenAI 兼容请求，并提供 cURL、Python、Node.js、Go、Java 五种语言的调用示例与常见错误排查。'

const FEATURES_TITLE = '功能介绍 - Xin API'
const FEATURES_DESCRIPTION =
  'Xin API 功能一览：统一接入 DeepSeek、Kimi、GLM、Qwen 等主流模型与三种入站协议，基于权重与多维指标的智能路由、Redis 分布式熔断、RPM/TPM 限流，以及 Token 计量、请求审计与运营看板。'

const FAQ_TITLE = '常见问题 - Xin API'
const FAQ_DESCRIPTION =
  'Xin API 常见问题：产品定位、支持的模型与协议、渠道高可用保障方式，以及用量统计与计费机制。'

const CONTACT_TITLE = '联系我们 - Xin API'
const CONTACT_DESCRIPTION =
  '联系 Xin API：技术支持与功能建议、GitHub 开源仓库、使用文档与控制台入口，以及商务合作的沟通方式。'

/** 公开子页面：与 src/routes/public.tsx 的公开路由一一对应 */
interface PublicPageMeta {
  /** 站点路径，同时决定产物目录 dist/<path>/index.html */
  path: string
  title: string
  description: string
  /** 面包屑与结构化数据里使用的页面名 */
  breadcrumb: string
  /** schema.org 页面类型，缺省 WebPage */
  pageType?: 'WebPage' | 'FAQPage' | 'TechArticle' | 'ContactPage'
  /** sitemap 中的 priority */
  priority: string
}

const PUBLIC_PAGES: PublicPageMeta[] = [
  {
    path: '/features',
    title: FEATURES_TITLE,
    description: FEATURES_DESCRIPTION,
    breadcrumb: '功能介绍',
    priority: '0.8',
  },
  {
    path: '/docs',
    title: DOCS_TITLE,
    description: DOCS_DESCRIPTION,
    breadcrumb: '使用文档',
    pageType: 'TechArticle',
    priority: '0.8',
  },
  {
    path: '/faq',
    title: FAQ_TITLE,
    description: FAQ_DESCRIPTION,
    breadcrumb: '常见问题',
    pageType: 'FAQPage',
    priority: '0.7',
  },
  {
    path: '/contact',
    title: CONTACT_TITLE,
    description: CONTACT_DESCRIPTION,
    breadcrumb: '联系我们',
    pageType: 'ContactPage',
    priority: '0.6',
  },
]

const PLACEHOLDER = '__SITE_URL__'

/**
 * 渲染公开路由为 HTML 字符串。
 * 使用 createElement 而非 JSX，以便本脚本以 .ts 扩展名被 tsx 直接执行。
 * 与客户端 main.tsx 使用完全相同的组件树（StrictMode > ThemeProvider > App > RouterProvider），
 * 仅把浏览器路由换成内存路由，保证 useId 与路由上下文在两端一致，避免水合不匹配。
 */
function renderRoute(path: string): string {
  const router = createMemoryRouter(publicRoutes, { initialEntries: [path] })
  return renderToString(
    createElement(
      StrictMode,
      null,
      createElement(
        ThemeProvider,
        null,
        createElement(App, { router }),
      ),
    ),
  )
}

/**
 * 将页面模板中的 title / description / canonical / OG / Twitter 元信息
 * 改写成该页面的版本；路由与站点 URL 占位符留给后续统一替换。
 */
function applyPageMeta(html: string, page: PublicPageMeta): string {
  return html
    .replace(/<title>[^<]*<\/title>/, `<title>${page.title}</title>`)
    .replace(
      /(<meta name="description" content=")[^"]*(")/,
      `$1${page.description}$2`,
    )
    .replace(
      /(<meta property="og:title" content=")[^"]*(")/,
      `$1${page.title}$2`,
    )
    .replace(
      /(<meta property="og:description" content=")[^"]*(")/,
      `$1${page.description}$2`,
    )
    .replace(
      /(<meta property="og:url" content=")[^"]*(")/,
      `$1${PLACEHOLDER}${page.path}$2`,
    )
    .replace(
      /(<meta name="twitter:title" content=")[^"]*(")/,
      `$1${page.title}$2`,
    )
    .replace(
      /(<meta name="twitter:description" content=")[^"]*(")/,
      `$1${page.description}$2`,
    )
    .replace(
      /(<link rel="canonical" href=")[^"]*(")/,
      `$1${PLACEHOLDER}${page.path}$2`,
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

function buildJsonLdBlocks(): unknown[] {
  const websiteLd = {
    '@context': 'https://schema.org',
    '@type': 'WebSite',
    name: SITE_NAME,
    url: `${SITE_URL}/`,
    description: SITE_DESCRIPTION,
    inLanguage: 'zh-CN',
  }

  const softwareAppLd = {
    '@context': 'https://schema.org',
    '@type': 'SoftwareApplication',
    name: SITE_NAME,
    applicationCategory: 'DeveloperApplication',
    operatingSystem: 'Web',
    url: `${SITE_URL}/`,
    description: SITE_DESCRIPTION,
    inLanguage: 'zh-CN',
    featureList: [...coreCapabilities, ...moreCapabilities].map(
      (feature) => feature.title,
    ),
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
}

/** 面包屑：首页 → 目标页 */
function buildBreadcrumbList(name: string, path: string): unknown {
  return {
    '@context': 'https://schema.org',
    '@type': 'BreadcrumbList',
    itemListElement: [
      {
        '@type': 'ListItem',
        position: 1,
        name: '首页',
        item: `${SITE_URL}/`,
      },
      {
        '@type': 'ListItem',
        position: 2,
        name,
        item: `${SITE_URL}${path}`,
      },
    ],
  }
}

/**
 * 公开子页面的结构化数据：面包屑 + 与该页类型匹配的主实体。
 * 首页的 JSON-LD 只描述站点与首页，子页面需要自己的层级与类型声明才便于被识别。
 */
function buildPageJsonLd(page: PublicPageMeta): unknown[] {
  const url = `${SITE_URL}${page.path}`

  switch (page.pageType) {
    case 'TechArticle':
      return [
        buildBreadcrumbList(page.breadcrumb, page.path),
        {
          '@context': 'https://schema.org',
          '@type': 'TechArticle',
          headline: page.title,
          description: page.description,
          url,
          inLanguage: 'zh-CN',
          author: { '@type': 'Organization', name: SITE_NAME },
          publisher: { '@type': 'Organization', name: SITE_NAME },
          mainEntityOfPage: { '@type': 'WebPage', '@id': url },
        },
      ]
    case 'FAQPage':
      return [
        buildBreadcrumbList(page.breadcrumb, page.path),
        {
          '@context': 'https://schema.org',
          '@type': 'FAQPage',
          name: page.title,
          description: page.description,
          url,
          inLanguage: 'zh-CN',
          mainEntity: faqItems.map((item) => ({
            '@type': 'Question',
            name: item.q,
            acceptedAnswer: {
              '@type': 'Answer',
              text: item.a,
            },
          })),
        },
      ]
    default:
      return [
        buildBreadcrumbList(page.breadcrumb, page.path),
        {
          '@context': 'https://schema.org',
          '@type': page.pageType ?? 'WebPage',
          name: page.title,
          description: page.description,
          url,
          inLanguage: 'zh-CN',
        },
      ]
  }
}

/** 序列化并插入到 </head> 之前 */
function injectJsonLd(html: string, blocks: unknown[], label: string): string {
  if (!html.includes('</head>')) {
    throw new Error(`${label} 中未找到 </head>，无法注入 JSON-LD`)
  }
  const serialized = blocks
    .map(
      (structured) =>
        `<script type="application/ld+json">${JSON.stringify(
          structured,
          null,
          2,
        )}</script>`,
    )
    .join('\n')
  return html.replace('</head>', `${serialized}\n  </head>`)
}

/** 替换站点 URL 占位符；残留占位符视为构建失败 */
function finalizeHtml(html: string, label: string): string {
  const output = html.split(PLACEHOLDER).join(SITE_URL)
  if (output.includes(PLACEHOLDER)) {
    console.error(`预渲染失败：${label} 中仍存在 __SITE_URL__ 占位符`)
    process.exit(1)
  }
  return output
}

function buildRobotsTxt(): string {
  return [
    'User-agent: *',
    'Allow: /',
    // 控制台页面禁止收录：这些路由会被 SPA 兜底成 app.html（该外壳已声明 noindex,nofollow），
    // 这里再显式 Disallow，避免爬虫反复抓取并生成大量重复空页
    'Disallow: /login',
    'Disallow: /dashboard',
    'Disallow: /apikeys',
    'Disallow: /logs',
    'Disallow: /billing',
    'Disallow: /settings',
    // 管理员路由
    'Disallow: /channels',
    'Disallow: /models',
    'Disallow: /monitoring',
    'Disallow: /pricing',
    'Disallow: /users',
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

  const entries = [
    { loc: `${SITE_URL}/`, priority: '1.0' },
    ...PUBLIC_PAGES.map((page) => ({
      loc: `${SITE_URL}${page.path}`,
      priority: page.priority,
    })),
  ]

  return [
    '<?xml version="1.0" encoding="UTF-8"?>',
    '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">',
    ...entries.map(
      (entry) =>
        `  <url><loc>${escapeXml(
          entry.loc,
        )}</loc><lastmod>${lastmod}</lastmod><changefreq>weekly</changefreq><priority>${entry.priority}</priority></url>`,
    ),
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

  // 首页：SSR 正文 + 站点级 JSON-LD
  const homeAppHtml = renderRoute('/')
  const homeHtml = finalizeHtml(
    injectRootHtml(templateHtml, homeAppHtml),
    'dist/index.html',
  )
  const homeJsonLd = buildJsonLdBlocks()
  await writeFile(
    INDEX_HTML_PATH,
    injectJsonLd(homeHtml, homeJsonLd, 'dist/index.html'),
    'utf8',
  )

  // 公开子页面：各自产出 dist/<path>/index.html，
  // 避免 SPA fallback 返回首页或控制台外壳（前者水合不匹配，后者带 noindex）
  const pageSizes: string[] = []
  for (const page of PUBLIC_PAGES) {
    const outPath = resolve(DIST_DIR, page.path.replace(/^\//, ''), 'index.html')
    mkdirSync(dirname(outPath), { recursive: true })

    const appHtml = renderRoute(page.path)
    const pageHtml = finalizeHtml(
      injectRootHtml(applyPageMeta(templateHtml, page), appHtml),
      outPath,
    )
    await writeFile(
      outPath,
      injectJsonLd(pageHtml, buildPageJsonLd(page), outPath),
      'utf8',
    )
    pageSizes.push(`${page.path} ${appHtml.length} 字符`)
  }

  await writeFile(ROBOTS_PATH, buildRobotsTxt(), 'utf8')
  await writeFile(SITEMAP_PATH, buildSitemapXml(), 'utf8')

  console.log(
    `[prerender] 完成：首页 SSR 正文 ${homeAppHtml.length} 字符、首页 JSON-LD ${JSON.stringify(
      homeJsonLd,
    ).length} 字符；` +
      `公开子页面 ${PUBLIC_PAGES.length} 个（${pageSizes.join('、')}）；` +
      `SITE_URL=${SITE_URL}；产物：${INDEX_HTML_PATH}、${ROBOTS_PATH}、${SITEMAP_PATH}`,
  )
}

main().catch((error: unknown) => {
  console.error('[prerender] 失败：', error)
  process.exit(1)
})
