import type { ChannelTestResult } from '@/types/api'
import { formatTime } from '@/lib/format'

export function channelStatusLabel(status: number): '正常' | '熔断' | '禁用' {
  if (status === 1) return '正常'
  if (status === 2) return '熔断'
  return '禁用'
}

export function groupStatusLabel(status: number): '启用' | '停用' {
  if (status === 1) return '启用'
  return '停用'
}

export function channelStatusTone(
  status: number,
): 'success' | 'danger' | 'muted' {
  if (status === 1) return 'success'
  if (status === 2) return 'danger'
  return 'muted'
}

export interface TestResultSummary {
  tone: 'success' | 'danger' | 'warning'
  text: string
  isMulti: boolean
}

export function summarizeTestResult(
  result: ChannelTestResult,
): TestResultSummary {
  const successCount = result.results.filter((r) => r.success).length
  const totalCount = result.results.length

  if (totalCount === 1) {
    const singleResult = result.results[0]
    if (singleResult?.success) {
      return {
        tone: 'success',
        text: formatTime(singleResult.response_time_ms),
        isMulti: false,
      }
    }
    return { tone: 'danger', text: '失败', isMulti: false }
  }

  if (successCount === totalCount) {
    return { tone: 'success', text: `${totalCount}个模型`, isMulti: true }
  }
  if (successCount === 0) {
    return { tone: 'danger', text: `${totalCount}个模型`, isMulti: true }
  }
  return {
    tone: 'warning',
    text: `${successCount}/${totalCount}个`,
    isMulti: true,
  }
}