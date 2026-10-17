import type { ChannelMetricRange, ChannelMetricTrendPoint } from '@/types/api'

interface MetricTrendChartProps {
  points: ChannelMetricTrendPoint[]
  range: ChannelMetricRange
}

const WIDTH = 760
const HEIGHT = 260
const PAD = { top: 16, right: 20, bottom: 30, left: 56 }

const CHART_WIDTH = WIDTH - PAD.left - PAD.right
const CHART_HEIGHT = HEIGHT - PAD.top - PAD.bottom

const TTFT_COLOR = 'var(--color-chart-2)'
const LATENCY_COLOR = 'var(--color-chart-1)'

// 毫秒展示：>= 1s 转为秒
function formatMs(value: number): string {
  if (value >= 1000) return `${(value / 1000).toFixed(1)}s`
  return `${Math.round(value)}ms`
}

// 将最大值向上取整到 1 / 2 / 2.5 / 5 × 10^n，保证刻度整洁
function niceMax(value: number): number {
  if (value <= 0) return 1
  const magnitude = 10 ** Math.floor(Math.log10(value))
  const normalized = value / magnitude
  const step = [1, 2, 2.5, 5, 10].find((s) => normalized <= s) ?? 10
  return step * magnitude
}

function formatBucket(iso: string, range: ChannelMetricRange): string {
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  const pad = (n: number) => String(n).padStart(2, '0')
  const time = `${pad(date.getHours())}:${pad(date.getMinutes())}`
  if (range === '7d') {
    return `${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${time}`
  }
  return time
}

/**
 * TTFT / 平均延迟双折线图（纯 SVG，无第三方图表依赖）
 */
export default function MetricTrendChart({
  points,
  range,
}: MetricTrendChartProps) {
  if (points.length === 0) return null

  const yMax = niceMax(
    Math.max(...points.map((p) => Math.max(p.avg_ttft_ms, p.avg_latency_ms)), 0),
  )

  const x = (index: number) =>
    PAD.left +
    (points.length === 1
      ? CHART_WIDTH / 2
      : (index / (points.length - 1)) * CHART_WIDTH)
  const y = (value: number) =>
    PAD.top + CHART_HEIGHT - (value / yMax) * CHART_HEIGHT

  const linePath = (pick: (p: ChannelMetricTrendPoint) => number) =>
    points
      .map((p, i) => `${i === 0 ? 'M' : 'L'}${x(i).toFixed(1)},${y(pick(p)).toFixed(1)}`)
      .join(' ')

  // Y 轴 4 等分
  const yTicks = [0, 0.25, 0.5, 0.75, 1].map((ratio) => ({
    value: yMax * ratio,
    y: PAD.top + CHART_HEIGHT - ratio * CHART_HEIGHT,
  }))

  // X 轴最多 6 个刻度（含首尾）
  const labelStep = Math.max(1, Math.ceil((points.length - 1) / 5))
  const xTicks = points
    .map((p, i) => ({ index: i, label: formatBucket(p.bucket_start, range) }))
    .filter(({ index }) => index % labelStep === 0 || index === points.length - 1)

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center gap-4 text-xs text-muted-foreground">
        <span className="inline-flex items-center gap-1.5">
          <span
            aria-hidden
            className="h-0.5 w-4 rounded-full"
            style={{ backgroundColor: TTFT_COLOR }}
          />
          TTFT
        </span>
        <span className="inline-flex items-center gap-1.5">
          <span
            aria-hidden
            className="h-0.5 w-4 rounded-full"
            style={{ backgroundColor: LATENCY_COLOR }}
          />
          平均延迟
        </span>
      </div>

      <svg
        viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
        className="h-64 w-full"
        role="img"
        aria-label="TTFT 与平均延迟趋势"
      >
        {/* 网格与 Y 轴刻度 */}
        {yTicks.map((tick) => (
          <g key={tick.value}>
            <line
              x1={PAD.left}
              x2={PAD.left + CHART_WIDTH}
              y1={tick.y}
              y2={tick.y}
              stroke="var(--color-border)"
              strokeDasharray={tick.value === 0 ? undefined : '3 4'}
            />
            <text
              x={PAD.left - 8}
              y={tick.y}
              textAnchor="end"
              dominantBaseline="middle"
              fontSize="11"
              fill="var(--color-muted-foreground)"
            >
              {formatMs(tick.value)}
            </text>
          </g>
        ))}

        {/* X 轴刻度 */}
        {xTicks.map((tick) => (
          <text
            key={tick.index}
            x={x(tick.index)}
            y={HEIGHT - 8}
            textAnchor={
              tick.index === 0
                ? 'start'
                : tick.index === points.length - 1
                  ? 'end'
                  : 'middle'
            }
            fontSize="11"
            fill="var(--color-muted-foreground)"
          >
            {tick.label}
          </text>
        ))}

        {/* 折线 */}
        <path
          d={linePath((p) => p.avg_latency_ms)}
          fill="none"
          stroke={LATENCY_COLOR}
          strokeWidth="2"
          strokeLinejoin="round"
          strokeLinecap="round"
        />
        <path
          d={linePath((p) => p.avg_ttft_ms)}
          fill="none"
          stroke={TTFT_COLOR}
          strokeWidth="2"
          strokeLinejoin="round"
          strokeLinecap="round"
        />

        {/* 数据点，悬停显示具体数值 */}
        {points.map((p, i) => (
          <g key={p.bucket_start}>
            <circle
              cx={x(i)}
              cy={y(p.avg_latency_ms)}
              r="2.5"
              fill={LATENCY_COLOR}
            >
              <title>
                {`${formatBucket(p.bucket_start, range)} 延迟 ${formatMs(p.avg_latency_ms)}`}
              </title>
            </circle>
            <circle cx={x(i)} cy={y(p.avg_ttft_ms)} r="2.5" fill={TTFT_COLOR}>
              <title>
                {`${formatBucket(p.bucket_start, range)} TTFT ${formatMs(p.avg_ttft_ms)}`}
              </title>
            </circle>
          </g>
        ))}
      </svg>
    </div>
  )
}
