import {
  BarChart3,
  BookOpen,
  KeyRound,
  LayoutDashboard,
  LifeBuoy,
  Mail,
  Network,
  PlugZap,
  ScanLine,
  ScrollText,
  ShieldCheck,
  Users,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

/** 顶部导航与页脚共用的站点菜单：均指向可被搜索引擎收录的独立页面 */
export const siteNavItems = [
  { label: '首页', href: '/' },
  { label: '功能介绍', href: '/features' },
  { label: '使用文档', href: '/docs' },
  { label: '常见问题', href: '/faq' },
  { label: '联系我们', href: '/contact' },
] as const

export interface FeatureItem {
  icon: LucideIcon
  title: string
  desc: string
}

export interface CapabilityItem extends FeatureItem {
  index: string
  keywords: string[]
}

/** 核心能力：四张编号大卡片 */
export const coreCapabilities: CapabilityItem[] = [
  {
    index: '01',
    icon: PlugZap,
    title: '统一接入',
    desc: '标准化 DeepSeek、Kimi、GLM、Qwen 等主流模型接口，兼容 /v1/chat/completions、/v1/responses 与 Gemini generateContent 三种入站协议。',
    keywords: ['OpenAI 兼容', 'DeepSeek', 'Kimi', 'GLM', 'Qwen'],
  },
  {
    index: '02',
    icon: Network,
    title: '智能路由',
    desc: '基于权重的负载均衡与可插拔路由架构，按渠道健康度、成功率、延迟 / TTFT / TPS、成本与 RPM / TPM 等多维指标加权选路。',
    keywords: ['权重分发', '多维打分', '自动切换'],
  },
  {
    index: '03',
    icon: ShieldCheck,
    title: '高可用防护',
    desc: '基于 Redis 的分布式熔断自动隔离故障渠道，RPM / TPM 多维限流保护上游，配合单渠道与批量多模型健康检测快速止损。',
    keywords: ['熔断隔离', 'RPM / TPM', '健康检测'],
  },
  {
    index: '04',
    icon: ScanLine,
    title: '全链路可观测',
    desc: '每次请求携带 xin_req_* Request ID 并关联上游供应商 Request ID，Token、延迟、费用与审计日志全程可查、可对账。',
    keywords: ['Request ID', 'Token 计量', '审计日志'],
  },
]

/** 更多能力：四张小卡片 */
export const moreCapabilities: FeatureItem[] = [
  {
    icon: Users,
    title: '统一渠道管理',
    desc: '集中管理多家上游供应商渠道，配置、权重与状态一目了然。',
  },
  {
    icon: KeyRound,
    title: 'API Key 管理',
    desc: '灵活的密钥分发与吊销，按需控制访问范围。',
  },
  {
    icon: ScrollText,
    title: '请求审计日志',
    desc: '按 Request ID、模型、渠道、状态与时间多维筛选，默认仅记录元数据。',
  },
  {
    icon: BarChart3,
    title: '运营看板',
    desc: '请求量、成功率、Token、费用与延迟趋势，以及模型与渠道调用排行。',
  },
]

export interface FlowStage {
  title: string
  desc: string
  items: string[]
}

/** 工作原理：四个阶段 */
export const flowStages: FlowStage[] = [
  {
    title: '接入鉴权',
    desc: '校验密钥并建立请求上下文',
    items: ['客户端请求', '认证鉴权', '请求上下文 · Request ID'],
  },
  {
    title: '路由选路',
    desc: '按多维指标加权决策',
    items: ['模型映射', '健康度与延迟评分', '成本与限流评分'],
  },
  {
    title: '渠道与供应商',
    desc: '转发到最优上游',
    items: ['渠道权重选择', 'DeepSeek / Kimi / GLM / Qwen', '熔断与限流保护'],
  },
  {
    title: '计量与观测',
    desc: '用量落库并进入看板',
    items: ['用量提取', '计费与日志', '控制台 Dashboard'],
  },
]

export const flowTraceNote =
  '每一次请求都携带 xin_req_* Request ID，并关联上游供应商返回的 Request ID，从客户端到供应商双向可追溯，故障定位一查到底。'

export interface MetricItem {
  label: string
  desc: string
}

/** 可观测指标说明 */
export const observabilityMetrics: MetricItem[] = [
  { label: '请求量', desc: '按模型、渠道与状态维度统计调用次数。' },
  { label: '成功率', desc: '区分网关与上游失败，快速定位异常渠道。' },
  { label: 'Token 用量', desc: '输入 / 输出分别计量，支持成本核算。' },
  { label: '费用', desc: '按价格快照原子扣费，账单清晰可对账。' },
  { label: '延迟', desc: '记录 TTFT、总延迟与 TPS，指导选路与选型。' },
]

export interface FaqItem {
  q: string
  a: string
}

export const faqItems: FaqItem[] = [
  {
    q: 'Xin API 是什么？',
    a: 'Xin API 是一个高性能的大语言模型（LLM）API 网关，统一管理多家大模型渠道，提供标准化接入、负载均衡、熔断限流与统一渠道管理能力。',
  },
  {
    q: '支持哪些模型与协议？',
    a: '无缝兼容 DeepSeek、Kimi、GLM、Qwen 等主流大模型，客户端可通过 /v1/chat/completions、/v1/responses 或 Gemini 原生 generateContent 端点接入，无需为不同供应商维护多套代码。',
  },
  {
    q: '如何保障渠道的高可用？',
    a: 'Xin API 通过基于权重的负载均衡在多个渠道间分发请求，并使用基于 Redis 的分布式熔断器自动隔离故障渠道，配合 RPM / TPM 多维限流与一键渠道健康检测，降低故障影响。',
  },
  {
    q: '是否支持用量统计与计费？',
    a: '支持。Xin API 提供请求审计日志、输入 / 输出 Token 分别计量与价格快照，按用量原子扣费，账单清晰可对账；多维感知智能路由基于这些运行数据持续优化选路决策。',
  },
]

export interface ContactChannel {
  icon: LucideIcon
  title: string
  desc: string
  action: { label: string; href: string; external?: boolean }
}

/** 对外邮箱：技术支持、商务合作与其它事宜的统一入口 */
export const CONTACT_EMAIL = '3107698270@qq.com'

/** 联系我们页的沟通渠道：只列可验证的入口 */
export const contactChannels: ContactChannel[] = [
  {
    icon: Mail,
    title: '邮件联系',
    desc: '商务合作、渠道接入或不便公开讨论的问题，直接发邮件给我们。',
    action: { label: CONTACT_EMAIL, href: `mailto:${CONTACT_EMAIL}` },
  },
  {
    icon: LifeBuoy,
    title: '技术支持',
    desc: '使用中遇到问题、疑似缺陷或需要排查协助，在 GitHub Issues 提交即可。',
    action: {
      label: '提交 Issue',
      href: 'https://github.com/GripAIcn/Xin-api/issues',
      external: true,
    },
  },
  {
    icon: BookOpen,
    title: '使用文档',
    desc: '从获取 API Key 到发起第一个请求的完整指引，含五种语言示例。',
    action: { label: '查看文档', href: '/docs' },
  },
  {
    icon: LayoutDashboard,
    title: '控制台',
    desc: '登录后配置渠道与密钥、查看用量与账单，日常问题多数可自助解决。',
    action: { label: '登录控制台', href: '/login' },
  },
]
