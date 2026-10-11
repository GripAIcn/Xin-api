import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  ArrowRight,
  KeyRound,
  Waypoints,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

import { DotBadge, type StatusTone } from '@/components/StatusBadge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Skeleton } from '@/components/ui/skeleton'
import { useApiKeyStore } from '@/stores/apikeys'
import { useAuthStore } from '@/stores/auth'
import { useChannelStore } from '@/stores/channels'
import type { ApiKey, Channel } from '@/types/api'

interface StatCard {
  title: string
  value: number
  icon: LucideIcon
  path: string
  iconClass: string
}

const quickActions: { label: string; path: string }[] = [
  { label: '查看渠道', path: '/channels' },
  { label: '创建 API Key', path: '/apikeys' },
]

const distributionRows: {
  label: string
  tone: StatusTone
  barClass: string
  match: (status: number) => boolean
}[] = [
  {
    label: '正常',
    tone: 'success',
    barClass: 'bg-success',
    match: (status) => status === 1,
  },
  {
    label: '熔断',
    tone: 'danger',
    barClass: 'bg-destructive',
    match: (status) => status === 2,
  },
  {
    label: '禁用',
    tone: 'muted',
    barClass: 'bg-muted-foreground/40',
    match: (status) => status !== 1 && status !== 2,
  },
]

export default function DashboardView() {
  const navigate = useNavigate()
  const username = useAuthStore((s) => s.username)
  const fetchChannels = useChannelStore((s) => s.fetchAll)
  const fetchApiKeys = useApiKeyStore((s) => s.fetchAll)

  const [loading, setLoading] = useState(true)
  const [channels, setChannels] = useState<Channel[]>([])
  const [apiKeys, setApiKeys] = useState<ApiKey[]>([])

  useEffect(() => {
    let active = true

    const load = async () => {
      try {
        try {
          const list = (await fetchChannels()) ?? []
          if (active) setChannels(list)
        } catch {
          /* silently fail */
        }

        try {
          const list = (await fetchApiKeys()) ?? []
          if (active) setApiKeys(list)
        } catch {
          /* silently fail */
        }
      } finally {
        if (active) setLoading(false)
      }
    }

    void load()

    return () => {
      active = false
    }
  }, [fetchChannels, fetchApiKeys])

  const totalChannels = channels.length
  const totalApiKeys = apiKeys.length

  // 渠道状态分布（客户端聚合）
  const distribution = useMemo(
    () =>
      distributionRows.map((row) => ({
        ...row,
        count: channels.filter((c) => row.match(c.status)).length,
      })),
    [channels],
  )

  const statsCards: StatCard[] = [
    {
      title: '渠道',
      value: totalChannels,
      icon: Waypoints,
      path: '/channels',
      iconClass: 'bg-success/10 text-success',
    },
    {
      title: 'API Key',
      value: totalApiKeys,
      icon: KeyRound,
      path: '/apikeys',
      iconClass: 'bg-warning/10 text-warning',
    },
  ]

  return (
    <div className="mx-auto w-full max-w-6xl space-y-6">
      {/* 欢迎区域 */}
      <div>
        <h1 className="mb-1 text-2xl font-semibold tracking-tight text-foreground">
          欢迎回来，{username || '管理员'}
        </h1>
        <p className="text-sm text-muted-foreground">
          这是 Xin-api 网关控制台，在这里管理你的渠道和 API Key。
        </p>
      </div>

      {/* 统计卡片 */}
      <div className="grid gap-4 md:grid-cols-2">
        {statsCards.map((card) => {
          const Icon = card.icon
          return (
            <Card
              key={card.title}
              onClick={() => navigate(card.path)}
              className="cursor-pointer gap-2 py-5 transition-all hover:-translate-y-0.5 hover:shadow-md"
            >
              <CardContent className="flex items-center justify-between px-5">
                {loading ? (
                  <div className="space-y-3">
                    <Skeleton className="h-4 w-16" />
                    <Skeleton className="h-10 w-20" />
                  </div>
                ) : (
                  <>
                    <div>
                      <p className="mb-1.5 text-sm text-muted-foreground">
                        {card.title}
                      </p>
                      <p className="text-4xl font-semibold leading-none tracking-tight text-foreground tabular-nums">
                        {card.value}
                      </p>
                    </div>
                    <div
                      className={`flex size-12 items-center justify-center rounded-xl ${card.iconClass}`}
                    >
                      <Icon className="size-6" />
                    </div>
                  </>
                )}
              </CardContent>
            </Card>
          )
        })}
      </div>

      {/* 渠道状态分布 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">渠道状态分布</CardTitle>
          <CardDescription>全部渠道按状态统计</CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="space-y-4">
              <Skeleton className="h-6 w-full" />
              <Skeleton className="h-6 w-full" />
              <Skeleton className="h-6 w-full" />
            </div>
          ) : totalChannels === 0 ? (
            <p className="py-4 text-center text-sm text-muted-foreground">
              暂无渠道数据
            </p>
          ) : (
            <div className="space-y-4">
              {distribution.map((row) => (
                <div key={row.label}>
                  <div className="mb-1.5 flex items-center justify-between">
                    <DotBadge tone={row.tone}>{row.label}</DotBadge>
                    <span className="text-sm text-muted-foreground tabular-nums">
                      {row.count} 个 ·{' '}
                      {totalChannels > 0
                        ? Math.round((row.count / totalChannels) * 100)
                        : 0}
                      %
                    </span>
                  </div>
                  <Progress
                    value={
                      totalChannels > 0
                        ? (row.count / totalChannels) * 100
                        : 0
                    }
                    indicatorClassName={row.barClass}
                  />
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* 快速操作 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">快速操作</CardTitle>
          <CardDescription>常用管理入口</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-wrap gap-3">
          {quickActions.map((action) => (
            <Button
              key={action.path}
              variant="outline"
              onClick={() => navigate(action.path)}
            >
              {action.label}
              <ArrowRight className="size-3" />
            </Button>
          ))}
        </CardContent>
      </Card>
    </div>
  )
}
