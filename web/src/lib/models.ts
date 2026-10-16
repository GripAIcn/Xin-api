import type { ModelChannelEntry, ModelGroup } from '@/types/api'

// 一级行「状态汇总」结果
export interface GroupStatusSummary {
  tone: 'success' | 'warning' | 'muted'
  label: string
}

// 汇总一个模型分组：渠道数、状态汇总、上下文一致性、能力并集
export interface GroupSummary {
  channelCount: number
  activeCount: number
  status: GroupStatusSummary
  contextLabel: string // 各渠道 context_length 一致 → 显示该值（如 "128K" 或原始数字），否则 "多值"
  capabilities: string[] // 各渠道 capabilities 的并集（去重，保持首次出现顺序）
}

// 格式化单个上下文长度为展示字符串：>= 1000 用 K 表示（如 128000 → "128K"），否则原样
function formatContextLength(value: number): string {
  if (value >= 1000) {
    // 向下取整保留一位小数，去掉多余的 .0
    const k = Math.floor((value / 1000) * 10) / 10
    return `${k}K`
  }
  return String(value)
}

// 汇总模型启用状态：全部启用 / 全部禁用 / 混合 / 无渠道
export function summarizeStatus(
  channels: ModelChannelEntry[],
): GroupStatusSummary {
  const total = channels.length
  if (total === 0) {
    return { tone: 'muted', label: '无渠道' }
  }
  const active = channels.filter((c) => c.status === 'active').length
  if (active === total) {
    return { tone: 'success', label: '全部启用' }
  }
  if (active === 0) {
    return { tone: 'muted', label: '全部禁用' }
  }
  return { tone: 'warning', label: `${active} 启用 / ${total - active} 禁用` }
}

// 汇总上下文长度：一致则显示格式化值，多值则 "多值"，无有效值则 "-"
export function summarizeContext(channels: ModelChannelEntry[]): string {
  const values = new Set<number>()
  for (const c of channels) {
    if (c.context_length != null) {
      values.add(c.context_length)
    }
  }
  if (values.size === 0) {
    return '-'
  }
  if (values.size === 1) {
    return formatContextLength(values.values().next().value as number)
  }
  return '多值'
}

// 汇总各渠道能力并集（去重，保持首次出现顺序）
export function summarizeCapabilities(
  channels: ModelChannelEntry[],
): string[] {
  const result: string[] = []
  const seen = new Set<string>()
  for (const c of channels) {
    for (const cap of c.capabilities) {
      if (!seen.has(cap)) {
        seen.add(cap)
        result.push(cap)
      }
    }
  }
  return result
}

// 汇总一个模型分组的一级展示信息
export function summarizeGroup(group: ModelGroup): GroupSummary {
  return {
    channelCount: group.channel_count,
    activeCount: group.active_count,
    status: summarizeStatus(group.channels),
    contextLabel: summarizeContext(group.channels),
    capabilities: summarizeCapabilities(group.channels),
  }
}
