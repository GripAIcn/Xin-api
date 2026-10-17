// 通用单序列趋势图（纯 SVG，无第三方图表依赖），供仪表盘的请求量 / Token / 费用趋势复用

export interface TrendDatum {
  // X 轴标签（调用方按时间范围格式化）
  label: string
  value: number
  // 悬停提示
  title: string
}

interface DashboardTrendChartProps {
  data: TrendDatum[]
  color: string
  formatValue: (value: number) => string
  ariaLabel: string
}

const WIDTH = 760
const HEIGHT = 220
const PAD = { top: 16, right: 20, bottom: 28, left: 56 }

const CHART_WIDTH = WIDTH - PAD.left - PAD.right
const CHART_HEIGHT = HEIGHT - PAD.top - PAD.bottom

// 将最大值向上取整到 1 / 2 / 2.5 / 5 × 10^n，保证刻度整洁
function niceMax(value: number): number {
  if (value <= 0) return 1
  const magnitude = 10 ** Math.floor(Math.log10(value))
  const normalized = value / magnitude
  const step = [1, 2, 2.5, 5, 10].find((s) => normalized <= s) ?? 10
  return step * magnitude
}

export default function DashboardTrendChart({
  data,
  color,
  formatValue,
  ariaLabel,
}: DashboardTrendChartProps) {
  if (data.length === 0) return null

  const yMax = niceMax(Math.max(...data.map((d) => d.value), 0))

  const x = (index: number) =>
    PAD.left +
    (data.length === 1
      ? CHART_WIDTH / 2
      : (index / (data.length - 1)) * CHART_WIDTH)
  const y = (value: number) =>
    PAD.top + CHART_HEIGHT - (value / yMax) * CHART_HEIGHT

  const linePath = data
    .map((d, i) => `${i === 0 ? 'M' : 'L'}${x(i).toFixed(1)},${y(d.value).toFixed(1)}`)
    .join(' ')

  // 面积路径：折线 + 回落到基线闭合
  const areaPath = `${linePath} L${x(data.length - 1).toFixed(1)},${(PAD.top + CHART_HEIGHT).toFixed(1)} L${x(0).toFixed(1)},${(PAD.top + CHART_HEIGHT).toFixed(1)} Z`

  // Y 轴 4 等分
  const yTicks = [0, 0.25, 0.5, 0.75, 1].map((ratio) => ({
    value: yMax * ratio,
    y: PAD.top + CHART_HEIGHT - ratio * CHART_HEIGHT,
  }))

  // X 轴最多 6 个刻度（含首尾）
  const labelStep = Math.max(1, Math.ceil((data.length - 1) / 5))
  const xTicks = data
    .map((d, i) => ({ index: i, label: d.label }))
    .filter(({ index }) => index % labelStep === 0 || index === data.length - 1)

  return (
    <svg
      viewBox={`0 0 ${WIDTH} ${HEIGHT}`}
      className="h-56 w-full"
      role="img"
      aria-label={ariaLabel}
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
            {formatValue(tick.value)}
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
              : tick.index === data.length - 1
                ? 'end'
                : 'middle'
          }
          fontSize="11"
          fill="var(--color-muted-foreground)"
        >
          {tick.label}
        </text>
      ))}

      {/* 面积 + 折线 */}
      <path d={areaPath} fill={color} fillOpacity="0.12" stroke="none" />
      <path
        d={linePath}
        fill="none"
        stroke={color}
        strokeWidth="2"
        strokeLinejoin="round"
        strokeLinecap="round"
      />

      {/* 数据点，悬停显示具体数值 */}
      {data.map((d, i) => (
        <circle key={`${d.label}-${i}`} cx={x(i)} cy={y(d.value)} r="2.5" fill={color}>
          <title>{d.title}</title>
        </circle>
      ))}
    </svg>
  )
}
