import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ArrowRight, Folder, KeyRound, Waypoints } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useApiKeyStore } from '@/stores/apikeys'
import { useAuthStore } from '@/stores/auth'
import { useChannelStore } from '@/stores/channels'
import { useGroupStore } from '@/stores/groups'

interface StatCard {
  title: string
  value: number
  icon: LucideIcon
  path: string
  color: string
  bg: string
}

const quickActions: { label: string; path: string }[] = [
  { label: '创建项目组', path: '/groups' },
  { label: '查看渠道', path: '/channels' },
  { label: '创建 API Key', path: '/apikeys' },
]

export default function DashboardView() {
  const navigate = useNavigate()
  const username = useAuthStore((s) => s.username)
  const fetchGroups = useGroupStore((s) => s.fetchAll)
  const fetchChannels = useChannelStore((s) => s.fetchByGroup)
  const fetchApiKeys = useApiKeyStore((s) => s.fetchAll)

  const [loading, setLoading] = useState(true)
  const [totalGroups, setTotalGroups] = useState(0)
  const [totalChannels, setTotalChannels] = useState(0)
  const [totalApiKeys, setTotalApiKeys] = useState(0)

  useEffect(() => {
    let active = true

    const load = async () => {
      try {
        // 1. 先获取所有项目组
        const groups = await fetchGroups()
        if (!active) return
        setTotalGroups(groups?.length ?? 0)

        // 2. 获取所有渠道（不传 groupId 获取全部），失败则逐组累加
        try {
          const channels = await fetchChannels()
          if (active) setTotalChannels(channels?.length ?? 0)
        } catch {
          if (groups && groups.length > 0) {
            let count = 0
            for (const g of groups) {
              try {
                const data = await fetchChannels(g.id)
                count += data?.length ?? 0
              } catch {
                /* ignore */
              }
            }
            if (active) setTotalChannels(count)
          }
        }

        // 3. 获取所有 API Key（不传 groupId 获取全部），失败则逐组累加
        try {
          const keys = await fetchApiKeys()
          if (active) setTotalApiKeys(keys?.length ?? 0)
        } catch {
          if (groups && groups.length > 0) {
            let count = 0
            for (const g of groups) {
              try {
                const data = await fetchApiKeys(g.id)
                count += data?.length ?? 0
              } catch {
                /* ignore */
              }
            }
            if (active) setTotalApiKeys(count)
          }
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

  const statsCards: StatCard[] = [
    {
      title: '项目组',
      value: totalGroups,
      icon: Folder,
      path: '/groups',
      color: '#409EFF',
      bg: '#ecf5ff',
    },
    {
      title: '渠道',
      value: totalChannels,
      icon: Waypoints,
      path: '/channels',
      color: '#67C23A',
      bg: '#f0f9eb',
    },
    {
      title: 'API Key',
      value: totalApiKeys,
      icon: KeyRound,
      path: '/apikeys',
      color: '#E6A23C',
      bg: '#fdf6ec',
    },
  ]

  return (
    <div className="mx-auto w-full max-w-6xl">
      {/* 欢迎区域 */}
      <div className="mb-6">
        <h1 className="mb-2 text-[22px] font-semibold text-foreground">
          欢迎回来，{username || '管理员'}
        </h1>
        <p className="text-sm text-muted-foreground">
          这是 Xin-api 网关控制面板，在这里管理你的项目组、渠道和 API Key。
        </p>
      </div>

      {/* 统计卡片 */}
      <div className="mb-6 grid gap-5 md:grid-cols-3">
        {statsCards.map((card) => {
          const Icon = card.icon
          return (
            <Card
              key={card.title}
              onClick={() => navigate(card.path)}
              className="cursor-pointer transition-shadow hover:shadow-md"
            >
              <CardContent>
                {loading ? (
                  <div className="space-y-3">
                    <Skeleton className="h-4 w-16" />
                    <Skeleton className="h-8 w-24" />
                  </div>
                ) : (
                  <div className="flex items-center justify-between">
                    <div className="flex-1">
                      <p className="mb-2 text-sm text-muted-foreground">
                        {card.title}
                      </p>
                      <p
                        className="text-[32px] font-semibold leading-none"
                        style={{ color: card.color }}
                      >
                        {card.value}
                      </p>
                    </div>
                    <div
                      className="flex size-14 items-center justify-center rounded-xl"
                      style={{ background: card.bg, color: card.color }}
                    >
                      <Icon className="size-6" />
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          )
        })}
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