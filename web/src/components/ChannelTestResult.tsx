import { CircleCheck, CircleX, TriangleAlert } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { formatTime } from '@/lib/format'
import { summarizeTestResult } from '@/lib/status'
import type { ChannelTestResult } from '@/types/api'

interface ChannelTestResultProps {
  result: ChannelTestResult
}

export default function ChannelTestResultCard({
  result,
}: ChannelTestResultProps) {
  const { tone } = summarizeTestResult(result)
  const successCount = result.results.filter((r) => r.success).length
  const totalCount = result.results.length

  return (
    <div className="space-y-4 py-2">
      {/* 渠道信息 */}
      <Card className="border-l-4 border-l-primary">
        <CardHeader>
          <CardTitle>{result.channel_name}</CardTitle>
          <CardDescription className="break-all">
            {result.base_url}
          </CardDescription>
        </CardHeader>
      </Card>

      {/* 汇总信息 */}
      <div>
        {tone === 'success' && (
          <Badge className="border-transparent bg-green-600 text-white">
            <CircleCheck />
            全部通过 ({successCount}/{totalCount})
          </Badge>
        )}
        {tone === 'danger' && (
          <Badge variant="destructive">
            <CircleX />
            全部失败 (0/{totalCount})
          </Badge>
        )}
        {tone === 'warning' && (
          <Badge className="border-transparent bg-amber-500 text-white">
            <TriangleAlert />
            部分通过 ({successCount}/{totalCount})
          </Badge>
        )}
      </div>

      {/* 模型测试结果表格 */}
      <Card>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="min-w-[120px]">模型</TableHead>
                <TableHead className="w-[100px]">状态</TableHead>
                <TableHead className="w-[100px]">响应时间</TableHead>
                <TableHead className="min-w-[150px]">错误信息</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {result.results.map((row) => (
                <TableRow key={row.model}>
                  <TableCell className="font-medium">{row.model}</TableCell>
                  <TableCell>
                    {row.success ? (
                      <Badge className="border-transparent bg-green-600 text-white">
                        <CircleCheck />
                        成功
                      </Badge>
                    ) : (
                      <Badge variant="destructive">
                        <CircleX />
                        失败
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell
                    className={
                      row.success ? 'text-green-600' : 'text-destructive'
                    }
                  >
                    {formatTime(row.response_time_ms)}
                  </TableCell>
                  <TableCell className="text-sm">
                    {row.error_msg ? (
                      <span className="text-destructive">{row.error_msg}</span>
                    ) : (
                      <span className="text-muted-foreground">-</span>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  )
}