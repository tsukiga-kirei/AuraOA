/**
 * chart-builders.ts — 图表选项构建纯函数。
 *
 * 从 StackedBarChart.vue 和 DeptDistributionChart.vue 中提取的
 * ECharts option 构建逻辑，便于单元测试和属性测试。
 */

// @ts-expect-error echarts 内部模块无类型声明，但运行时可用
import { LinearGradient } from 'echarts/lib/util/graphic'
import type { DeptDistributionData } from '~/types/dashboard-overview'

// ── 公共类型 ──

/** 图表主题色配置（与 useThemeColors 的 chartColors 形状一致） */
export interface ChartThemeColors {
  primary: string
  primaryLight: string
  accent: string
  success: string
  warning: string
  danger: string
  textPrimary: string
  textSecondary: string
  textTertiary: string
  border: string
  bgCard: string
  isDark: boolean
}

/** 堆叠柱状图系列数据 */
export interface StackedBarSeriesItem {
  name: string
  data: number[]
  color: string
}

/** 部门分布图图例标签 */
export interface DeptChartLabels {
  audit: string
  cron: string
  archive: string
  summary: string
  chat?: string
}

// ── 工具函数 ──

/**
 * 将十六进制颜色转换为 RGBA 字符串。
 * 用于生成渐变色的起止色值（带透明度）。
 */
export function hexToRgba(hex: string, alpha: number): string {
  const r = parseInt(hex.slice(1, 3), 16)
  const g = parseInt(hex.slice(3, 5), 16)
  const b = parseInt(hex.slice(5, 7), 16)
  return `rgba(${r},${g},${b},${alpha})`
}

// ── 堆叠柱状图 & 平滑面积折线图选项构建 ──

/** 部门分布图系列配色映射 */
const SERIES_COLORS = {
  audit: '#4f46e5',
  cron: '#06b6d4',
  archive: '#10b981',
  summary: '#f59e0b',
  chat: '#8b5cf6',
} as const

/** 通用 Tooltip 配置，具有现代微投影与圆角风格 */
function createModernTooltip(colors: ChartThemeColors, isAxis = true) {
  return {
    trigger: isAxis ? ('axis' as const) : ('item' as const),
    axisPointer: {
      type: 'cross' as const,
      lineStyle: {
        color: colors.isDark ? '#475569' : '#cbd5e1',
        type: 'dashed' as const,
        width: 1,
      },
      crossStyle: {
        color: colors.isDark ? '#475569' : '#cbd5e1',
      },
      shadowStyle: {
        color: colors.isDark ? 'rgba(255, 255, 255, 0.04)' : 'rgba(0, 0, 0, 0.03)',
      },
    },
    backgroundColor: colors.isDark ? '#1e293b' : '#ffffff',
    borderColor: colors.border || (colors.isDark ? '#334155' : '#e2e8f0'),
    borderWidth: 1,
    borderRadius: 8,
    padding: [10, 14],
    textStyle: {
      color: colors.textPrimary || '#0f172a',
      fontSize: 12,
    },
    extraCssText: 'box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.1), 0 8px 10px -6px rgba(0, 0, 0, 0.1); border-radius: 8px;',
  }
}

/**
 * 构建平滑渐变面积折线图的 ECharts option（趋势图默认高级形态）。
 *
 * @param categories - X 轴分类标签
 * @param series - 系列数据（含名称、数值、颜色）
 * @param themeColors - 当前主题色配置
 * @returns ECharts option 对象
 */
export function buildTrendLineOption(
  categories: string[],
  series: StackedBarSeriesItem[],
  themeColors: ChartThemeColors,
) {
  const colors = themeColors

  return {
    tooltip: createModernTooltip(colors, true),
    legend: {
      top: 0,
      right: 0,
      icon: 'circle',
      itemWidth: 8,
      itemHeight: 8,
      itemGap: 14,
      textStyle: {
        fontSize: 12,
        color: colors.textSecondary || '#64748b',
      },
    },
    grid: {
      left: 10,
      right: 16,
      top: 36,
      bottom: 12,
      containLabel: true,
    },
    xAxis: {
      type: 'category' as const,
      data: categories,
      boundaryGap: false,
      axisLine: { lineStyle: { color: colors.border || (colors.isDark ? '#334155' : '#e2e8f0') } },
      axisLabel: { color: colors.textTertiary || '#94a3b8', fontSize: 11, margin: 10 },
      axisTick: { show: false },
    },
    yAxis: {
      type: 'value' as const,
      minInterval: 1,
      axisLine: { show: false },
      axisLabel: { color: colors.textTertiary || '#94a3b8', fontSize: 11 },
      splitLine: {
        lineStyle: {
          color: colors.isDark ? '#334155' : '#f1f5f9',
          type: 'dashed' as const,
        },
      },
    },
    animationDuration: 800,
    animationEasing: 'cubicOut' as const,
    series: series.map(s => ({
      name: s.name,
      type: 'line' as const,
      smooth: 0.38,
      showSymbol: false,
      symbol: 'circle',
      symbolSize: 6,
      lineStyle: { width: 2.2, color: s.color },
      itemStyle: { color: s.color, borderWidth: 2, borderColor: colors.bgCard || '#ffffff' },
      areaStyle: {
        color: new LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: hexToRgba(s.color, 0.28) },
          { offset: 1, color: hexToRgba(s.color, 0.01) },
        ]),
      },
      emphasis: { focus: 'series' as const },
      data: s.data,
    })),
  }
}

/**
 * 构建纵向堆叠柱状图的 ECharts option。
 * 优化排版：图例置于顶部右侧，彻底解决图例与底部 X 轴日期重叠的问题。
 *
 * @param categories - X 轴分类标签
 * @param series - 系列数据（含名称、数值、颜色）
 * @param themeColors - 当前主题色配置
 * @returns ECharts option 对象
 */
export function buildStackedBarOption(
  categories: string[],
  series: StackedBarSeriesItem[],
  themeColors: ChartThemeColors,
) {
  const colors = themeColors

  return {
    tooltip: createModernTooltip(colors, true),
    legend: {
      top: 0,
      right: 0,
      icon: 'circle',
      itemWidth: 8,
      itemHeight: 8,
      itemGap: 14,
      textStyle: {
        fontSize: 12,
        color: colors.textSecondary || '#64748b',
      },
    },
    grid: {
      left: 10,
      right: 16,
      top: 36,
      bottom: 12,
      containLabel: true,
    },
    xAxis: {
      type: 'category' as const,
      data: categories,
      axisLine: { lineStyle: { color: colors.border || (colors.isDark ? '#334155' : '#e2e8f0') } },
      axisLabel: { color: colors.textTertiary || '#94a3b8', fontSize: 11, margin: 10 },
      axisTick: { show: false },
    },
    yAxis: {
      type: 'value' as const,
      minInterval: 1,
      axisLine: { show: false },
      axisLabel: { color: colors.textTertiary || '#94a3b8', fontSize: 11 },
      splitLine: {
        lineStyle: {
          color: colors.isDark ? '#334155' : '#f1f5f9',
          type: 'dashed' as const,
        },
      },
    },
    animationDuration: 600,
    animationEasing: 'cubicOut' as const,
    series: series.map((s, idx) => ({
      name: s.name,
      type: 'bar' as const,
      stack: 'total',
      data: s.data,
      barMaxWidth: 18,
      itemStyle: {
        color: new LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: s.color },
          { offset: 1, color: hexToRgba(s.color, 0.75) },
        ]),
        borderRadius: idx === series.length - 1 ? [4, 4, 0, 0] : [2, 2, 0, 0],
      },
    })),
  }
}

// ── 部门分布图选项构建 ──

/**
 * 构建横向堆叠柱状图（部门分布）的 ECharts option。
 * 优化排版与信息密度：
 * 1. 过滤无效/长尾全为 0 的部门，按总业务量降序提取 Top 8 活跃部门，避免大片空白；
 * 2. 图例置于顶部右侧，彻底解决与底部 X 轴标签重叠拥挤；
 * 3. 柱条附带底色槽与圆角，更显精致。
 *
 * @param data - 部门分布数据数组
 * @param labels - 图例标签
 * @param themeColors - 当前主题色配置
 * @returns ECharts option 对象
 */
export function buildDeptChartOption(
  data: DeptDistributionData[],
  labels: DeptChartLabels,
  themeColors: ChartThemeColors,
) {
  const colors = themeColors

  // 1. 部门数据按总业务量（total）降序排序
  const sorted = [...data].sort((a, b) => (b.total ?? (b.audit_count + b.cron_count + b.archive_count + b.summary_count + (b.chat_count ?? 0))) - (a.total ?? (a.audit_count + a.cron_count + a.archive_count + a.summary_count + (a.chat_count ?? 0))))

  // 2. 取 Top 7 部门，若有更多，将余下非零部门归并为“其他”；若无数据则兜底展示空或原始项
  let displayData: DeptDistributionData[] = []
  if (sorted.length <= 8) {
    displayData = sorted
  } else {
    const top7 = sorted.slice(0, 7)
    const rest = sorted.slice(7)
    const restTotal = rest.reduce((acc, cur) => acc + (cur.total ?? 0), 0)
    if (restTotal > 0) {
      const otherItem: DeptDistributionData = {
        department: '其他部门',
        total: restTotal,
        audit_count: rest.reduce((acc, cur) => acc + cur.audit_count, 0),
        cron_count: rest.reduce((acc, cur) => acc + cur.cron_count, 0),
        archive_count: rest.reduce((acc, cur) => acc + cur.archive_count, 0),
        summary_count: rest.reduce((acc, cur) => acc + cur.summary_count, 0),
        chat_count: rest.reduce((acc, cur) => acc + (cur.chat_count ?? 0), 0),
      }
      displayData = [...top7, otherItem]
    } else {
      displayData = top7
    }
  }

  // 如果全部部门业务都为 0，只保留前 5 项避免无限空白
  if (displayData.every(d => (d.total ?? 0) === 0) && displayData.length > 5) {
    displayData = displayData.slice(0, 5)
  }

  const depts = displayData.map(d => d.department)

  return {
    tooltip: createModernTooltip(colors, true),
    legend: {
      top: 0,
      right: 0,
      icon: 'circle',
      itemWidth: 8,
      itemHeight: 8,
      itemGap: 14,
      textStyle: {
        fontSize: 12,
        color: colors.textSecondary || '#64748b',
      },
    },
    grid: {
      left: 10,
      right: 28,
      top: 36,
      bottom: 10,
      containLabel: true,
    },
    yAxis: {
      type: 'category' as const,
      data: depts,
      inverse: true,
      axisLine: { lineStyle: { color: colors.border || (colors.isDark ? '#334155' : '#e2e8f0') } },
      axisLabel: {
        color: colors.textSecondary || '#475569',
        fontSize: 12,
        formatter: (val: string) => (val.length > 7 ? val.slice(0, 6) + '…' : val),
      },
      axisTick: { show: false },
    },
    xAxis: {
      type: 'value' as const,
      minInterval: 1,
      axisLine: { show: false },
      axisLabel: { color: colors.textTertiary || '#94a3b8', fontSize: 11 },
      splitLine: {
        lineStyle: {
          color: colors.isDark ? '#334155' : '#f1f5f9',
          type: 'dashed' as const,
        },
      },
    },
    animationDuration: 600,
    animationEasing: 'cubicOut' as const,
    series: [
      {
        name: labels.audit,
        type: 'bar' as const,
        stack: 'total',
        data: displayData.map(d => d.audit_count),
        barMaxWidth: 14,
        itemStyle: {
          color: new LinearGradient(0, 0, 1, 0, [
            { offset: 0, color: hexToRgba(SERIES_COLORS.audit, 0.75) },
            { offset: 1, color: SERIES_COLORS.audit },
          ]),
          borderRadius: [0, 4, 4, 0],
        },
      },
      {
        name: labels.cron,
        type: 'bar' as const,
        stack: 'total',
        data: displayData.map(d => d.cron_count),
        barMaxWidth: 14,
        itemStyle: {
          color: new LinearGradient(0, 0, 1, 0, [
            { offset: 0, color: hexToRgba(SERIES_COLORS.cron, 0.75) },
            { offset: 1, color: SERIES_COLORS.cron },
          ]),
          borderRadius: [0, 4, 4, 0],
        },
      },
      {
        name: labels.archive,
        type: 'bar' as const,
        stack: 'total',
        data: displayData.map(d => d.archive_count),
        barMaxWidth: 14,
        itemStyle: {
          color: new LinearGradient(0, 0, 1, 0, [
            { offset: 0, color: hexToRgba(SERIES_COLORS.archive, 0.75) },
            { offset: 1, color: SERIES_COLORS.archive },
          ]),
          borderRadius: [0, 4, 4, 0],
        },
      },
      {
        name: labels.summary,
        type: 'bar' as const,
        stack: 'total',
        data: displayData.map(d => d.summary_count),
        barMaxWidth: 14,
        itemStyle: {
          color: new LinearGradient(0, 0, 1, 0, [
            { offset: 0, color: hexToRgba(SERIES_COLORS.summary, 0.75) },
            { offset: 1, color: SERIES_COLORS.summary },
          ]),
          borderRadius: [0, 4, 4, 0],
        },
      },
      ...(labels.chat ? [{
        name: labels.chat,
        type: 'bar' as const,
        stack: 'total',
        data: displayData.map(d => d.chat_count ?? 0),
        barMaxWidth: 14,
        itemStyle: {
          color: new LinearGradient(0, 0, 1, 0, [
            { offset: 0, color: hexToRgba(SERIES_COLORS.chat, 0.75) },
            { offset: 1, color: SERIES_COLORS.chat },
          ]),
          borderRadius: [0, 4, 4, 0],
        },
      }] : []),
    ],
  }
}

// ── 仪表盘（Gauge）图表选项构建 ──

/** Gauge 图表阈值配置 */
export interface GaugeThresholds {
  warning: number
  danger: number
}

/**
 * 根据指标值和阈值返回对应的语义颜色。
 * - value ≤ warning  → 正常色（success / 绿色）
 * - value > warning 且 ≤ danger → 警告色（warning / 黄色）
 * - value > danger   → 危险色（danger / 红色）
 *
 * 与 useThemeColors.ts 中的 getMetricColor 逻辑一致，
 * 此处使用 themeColors 中的色值以适配当前主题。
 */
function resolveGaugeColor(
  value: number,
  thresholds: GaugeThresholds,
  themeColors: ChartThemeColors,
): string {
  if (value > thresholds.danger) return themeColors.danger || '#ef4444'
  if (value > thresholds.warning) return themeColors.warning || '#f59e0b'
  return themeColors.success || '#10b981'
}

/** 默认阈值（CPU / 内存通用） */
const DEFAULT_GAUGE_THRESHOLDS: GaugeThresholds = { warning: 80, danger: 95 }

/**
 * 构建 ECharts gauge 仪表盘图表的 option。
 * 采用简洁的圆弧进度条风格，无指针、无刻度标签。
 *
 * @param value      - 指标值（0-100 百分比）
 * @param label      - 指标名称（如 "CPU 使用率"）
 * @param thresholds - 阈值配置（warning / danger）
 * @param themeColors - 当前主题色配置
 * @returns ECharts option 对象
 */
export function buildGaugeOption(
  value: number,
  label: string,
  thresholds: GaugeThresholds | undefined,
  themeColors: ChartThemeColors,
) {
  const t = thresholds ?? DEFAULT_GAUGE_THRESHOLDS
  const color = resolveGaugeColor(value, t, themeColors)
  const trackColor = themeColors.isDark ? '#334155' : '#f1f5f9'

  return {
    series: [
      {
        type: 'gauge' as const,
        startAngle: 215,
        endAngle: -35,
        center: ['50%', '52%'],
        radius: '92%',
        min: 0,
        max: 100,
        // 精致圆弧进度条
        progress: {
          show: true,
          width: 11,
          roundCap: true,
          itemStyle: {
            color,
            shadowBlur: 8,
            shadowColor: hexToRgba(color, 0.35),
          },
        },
        // 轨道底槽
        axisLine: {
          roundCap: true,
          lineStyle: { width: 11, color: [[1, trackColor]] },
        },
        pointer: { show: false },
        axisTick: { show: false },
        splitLine: { show: false },
        axisLabel: { show: false },
        // 中心百分比数字
        detail: {
          valueAnimation: true,
          offsetCenter: [0, '-4%'],
          fontSize: 22,
          fontWeight: '700' as const,
          color: themeColors.textPrimary || '#0f172a',
          formatter: '{value}%',
        },
        // 指标名称
        title: {
          show: true,
          offsetCenter: [0, '36%'],
          fontSize: 12,
          fontWeight: '500' as const,
          color: themeColors.textSecondary || '#64748b',
        },
        data: [{ value, name: label }],
        animationDuration: 800,
        animationEasing: 'cubicOut' as const,
      },
    ],
  }
}
