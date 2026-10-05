import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  ArrowRight,
  Folder,
  FolderKanban,
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
import { useGroupStore } from '@/stores/groups'
import type { ApiKey, Channel, Group } from '@/types/api'

interface StatCard {
  title: string
  value: number
  icon: LucideIcon
  path: string
  iconClass: string
}

const quickActions: { label: string; path: string }[] = [
  { label: '创建项目组', path: '/groups' },
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
  const fetchGroups = useGroupStore((s) => s.fetchAll)
  const fetchChannels = useChannelStore((s) => s.fetchByGroup)
  const fetchApiKeys = useApiKeyStore((s) => s.fetchAll)

  const [loading, setLoading] = useState(true)
  const [groups, setGroups] = useState<Group[]>([])
  const [channels, setChannels] = useState<Channel[]>([])
  const [apiKeys, setApiKeys] = useState<ApiKey[]>([])

  useEffect(() => {
    let active = true

    const load = async () => {
      try {
        // 1. 先获取所有项目组
        const groupList = (await fetchGroups()) ?? []
        if (!active) return
        setGroups(groupList)

        // 2. 获取所有渠道（不传 groupId 获取全部），失败则逐组累加
        try {
          const list = (await fetchChannels()) ?? []
          if (active) setChannels(list)
        } catch {
          const collected: Channel[] = []
          for (const g of groupList) {
            try {
              const data = await fetchChannels(g.id)
              collected.push(...(data ?? []))
            } catch {
              /* ignore */
            }
          }
          if (active) setChannels(collected)
        }

        // 3. 获取所有 API Key（不传 groupId 获取全部），失败则逐组累加
        try {
          const list = (await fetchApiKeys()) ?? []
          if (active) setApiKeys(list)
        } catch {
          const collected: ApiKey[] = []
          for (const g of groupList) {
            try {
              const data = await fetchApiKeys(g.id)
              collected.push(...(data ?? []))
            } catch {
              /* ignore */
            }
          }
          if (active) setApiKeys(collected)
        }
      } catch {
        /* silently fail */
      } finally {
        if (active) setLoading(false)
      }
    }

    void load()

    return () => {
      active = false
    }
  }, [fetchGroups, fetchChannels, fetchApiKeys])

  const totalGroups = groups.length
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

  // 各项目组渠道数量与占比（客户端按 groupId 聚合）
  const groupOverview = useMemo(() => {
    const counts = new Map<number, number>()
    for (const c of channels) {
      counts.set(c.group_id, (counts.get(c.group_id) ?? 0) + 1)
    }
    return groups.map((g) => ({
      group: g,
      channelCount: counts.get(g.id) ?? 0,
      percent: totalChannels > 0 ? ((counts.get(g.id) ?? 0) / totalChannels) * 100 : 0,
    }))
  }, [groups, channels, totalChannels])

  const statsCards: StatCard[] = [
    {
      title: '项目组',
      value: totalGroups,
      icon: FolderKanban,
      path: '/groups',
      iconClass: 'bg-primary/10 text-primary',
    },
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
          这是 Xin-api 网关控制台，在这里管理你的项目组、渠道和 API Key。
        </p>
      </div>

      {/* 统计卡片 */}
      <div className="grid gap-4 md:grid-cols-3">
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

      {/* 分布与概览 */}
      <div className="grid gap-4 lg:grid-cols-2">
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

        {/* 项目组概览 */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">项目组概览</CardTitle>
            <CardDescription>各项目组渠道数量与占比</CardDescription>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="space-y-4">
                <Skeleton className="h-6 w-full" />
                <Skeleton className="h-6 w-full" />
                <Skeleton className="h-6 w-full" />
              </div>
            ) : groupOverview.length === 0 ? (
              <div className="py-4 text-center">
                <p className="text-sm text-muted-foreground">
                  暂无项目组，从创建第一个项目组开始
                </p>
                <Button
                  variant="link"
                  size="sm"
                  className="mt-1"
                  onClick={() => navigate('/groups')}
                >
                  前往创建
                  <ArrowRight className="size-3.5" />
                </Button>
              </div>
            ) : (
              <div className="space-y-4">
                {groupOverview.map(({ group, channelCount, percent }) => (
                  <button
                    key={group.id}
                    type="button"
                    onClick={() => navigate(`/groups/${group.id}`)}
                    className="block w-full text-left"
                  >
                    <div className="mb-1.5 flex items-center justify-between gap-2">
                      <span className="flex min-w-0 items-center gap-2 text-sm font-medium text-foreground">
                        <Folder className="size-4 shrink-0 text-muted-foreground" />
                        <span className="truncate">{group.name}</span>
                      </span>
                      <span className="shrink-0 text-sm text-muted-foreground tabular-nums">
                        {channelCount} 个 · {Math.round(percent)}%
                      </span>
                    </div>
                    <Progress value={percent} />
                  </button>
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

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
