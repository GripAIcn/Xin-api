import {
  Activity,
  BarChart3,
  Calculator,
  Gauge,
  KeyRound,
  Network,
  PlugZap,
  Scale,
  ScanLine,
  ScrollText,
  ShieldCheck,
  Users,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

export interface FeatureItem {
  icon: LucideIcon
  title: string
  desc: string
}

export const features: FeatureItem[] = [
  {
    icon: PlugZap,
    title: '统一 API 接入',
    desc: '标准化 OpenAI、Anthropic Claude、DeepSeek 等主流大模型接口，一个 /v1/chat/completions 端点完成统一接入。',
  },
  {
    icon: Scale,
    title: '智能负载均衡',
    desc: '基于权重的负载均衡策略，自动将请求分发到最佳渠道，提升整体吞吐与稳定性。',
  },
  {
    icon: ShieldCheck,
    title: '分布式熔断保护',
    desc: '基于 Redis 的分布式熔断器自动隔离故障渠道，防止故障蔓延与雪崩效应。',
  },
  {
    icon: Gauge,
    title: '多维度限流',
    desc: '基于 Redis 的分布式限流，支持 RPM / TPM 多维度流量控制，保护后端模型服务。',
  },
  {
    icon: Users,
    title: '多租户管理',
    desc: '用户、项目组、渠道多层级管理，资源按项目组隔离，团队协作清晰有序。',
  },
  {
    icon: KeyRound,
    title: 'API Key 与健康检测',
    desc: '灵活的 API Key 分发与吊销；支持单渠道与批量多模型连通性一键测试。',
  },
  {
    icon: Network,
    title: '多维感知智能路由',
    desc: '可插拔路由架构，按渠道健康度、成功率、延迟 / TTFT / TPS、成本与 RPM / TPM 等多维指标打分，加权智能选路。',
  },
  {
    icon: ScanLine,
    title: '全链路请求追溯',
    desc: '每次请求携带 xin_req_* Request ID 并关联上游供应商 Request ID，客户端到供应商双向追溯，故障定位一查到底。',
  },
]

export const flowSteps: string[] = [
  '客户端请求',
  '认证鉴权',
  '请求上下文 · Request ID',
  '路由选路',
  '渠道',
  '模型供应商',
  '用量提取',
  '计费与日志',
  '控制台 Dashboard',
]

export const flowTraceNote =
  '每一次请求都携带 xin_req_* Request ID，并关联上游供应商返回的 Request ID，从客户端到供应商全链路双向可追溯，故障定位一查到底。'

export interface QuickStep {
  title: string
  desc: string
}

export const quickSteps: QuickStep[] = [
  {
    title: '创建项目组',
    desc: '按业务或团队建立项目组，完成资源隔离与成员规划。',
  },
  {
    title: '添加渠道并测试',
    desc: '配置模型渠道，一键测试单渠道或批量多模型连通性。',
  },
  {
    title: '生成 API Key 接入',
    desc: '在项目组下生成 API Key，使用统一端点即刻接入应用。',
  },
]

export const v2Features: FeatureItem[] = [
  {
    icon: ScrollText,
    title: '请求审计日志',
    desc: '按 Request ID、模型、渠道、供应商、状态、类型与时间多维筛选；默认仅记录元数据，避免存储膨胀。',
  },
  {
    icon: Activity,
    title: 'Token 用量与性能',
    desc: '区分输入、输出与总 Token，记录 TTFT、延迟与 TPS，为模型选型与路由决策提供性能依据。',
  },
  {
    icon: Calculator,
    title: '模型定价与计费',
    desc: '输入 / 输出分别计价并保留价格快照，decimal 精度原子扣费；不预扣、不接支付，账单清晰可对账。',
  },
  {
    icon: BarChart3,
    title: '运营看板与渠道指标',
    desc: '请求量、成功率、Token、费用与延迟趋势，以及模型 / 渠道调用排行，按渠道维度沉淀指标。',
  },
]

export interface FaqItem {
  q: string
  a: string
}

export const faqItems: FaqItem[] = [
  {
    q: 'Xin API 是什么？',
    a: 'Xin API 是一个高性能的大语言模型（LLM）API 网关，统一管理多家大模型渠道，提供标准化接入、负载均衡、熔断限流与多租户管理能力。',
  },
  {
    q: '支持哪些模型与协议？',
    a: '无缝兼容 OpenAI、Anthropic Claude、DeepSeek 等主流大模型，客户端通过统一的 /v1/chat/completions 端点接入，无需为不同供应商维护多套代码。',
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
