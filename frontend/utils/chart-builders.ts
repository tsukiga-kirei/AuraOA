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

// ── 堆叠柱状图选项构建 ──

/** 部门分布图系列配色映射 */
const SERIES_COLORS = {
  audit: '#4f46e5',
  cron: '#06b6d4',
  archive: '#10b981',
} as const

/**
 * 构建纵向堆叠柱状图的 ECharts option。
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
    // tooltip：圆角、阴影、主题适配背景色
    tooltip: {
      trigger: 'axis' as const,
      axisPointer: { type: 'shadow' as const },
      backgroundColor: colors.isDark ? '#1e293b' : '#ffffff',
      borderColor: colors.border || '#e2e8f0',
      borderWidth: 1,
      borderRadius: 8,
      padding: [8, 12],
      textStyle: {
        color: colors.textPrimary || '#0f172a',
        fontSize: 13,
      },
      extraCssText: 'box-shadow: 0 4px 12px rgba(0,0,0,0.12);',
    },

    // 图例：使用主题文字色
    legend: {
      bottom: 0,
      textStyle: {
        fontSize: 12,
        color: colors.textSecondary || '#475569',
      },
    },

    grid: { left: 40, right: 16, top: 16, bottom: 40 },

    // X 轴：分类轴，颜色从 chartColors 读取
    xAxis: {
      type: 'category' as const,
      data: categories,
      axisLine: { lineStyle: { color: colors.border || '#e2e8f0' } },
      axisLabel: { color: colors.textSecondary || '#475569' },
      axisTick: { lineStyle: { color: colors.border || '#e2e8f0' } },
    },

    // Y 轴：数值轴，颜色从 chartColors 读取
    yAxis: {
      type: 'value' as const,
      minInterval: 1,
      axisLine: { show: false },
      axisLabel: { color: colors.textSecondary || '#475569' },
      splitLine: { lineStyle: { color: colors.border || '#e2e8f0', type: 'dashed' as const } },
    },

    // 全局动画配置
    animationDuration: 600,
    animationEasing: 'cubicOut' as const,

    // 为每个系列创建渐变色填充 + 圆角柱体
    series: series.map(s => ({
      name: s.name,
      type: 'bar' as const,
      stack: 'total',
      data: s.data,
      barMaxWidth: 32,
      itemStyle: {
        // 使用 LinearGradient 实现从上到下的渐变色填充
        color: new LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: s.color },
          { offset: 1, color: hexToRgba(s.color, 0.6) },
        ]),
        // 圆角柱体：仅顶部两个角为圆角
        borderRadius: [4, 4, 0, 0] as [number, number, number, number],
      },
    })),
  }
}

// ── 部门分布图选项构建 ──

/**
 * 构建横向堆叠柱状图（部门分布）的 ECharts option。
 *
 * @param data - 部门分布数据数组
 * @param labels - 图例标签（审核 / 定时任务 / 归档）
 * @param themeColors - 当前主题色配置
 * @returns ECharts option 对象
 */
export function buildDeptChartOption(
  data: DeptDistributionData[],
  labels: DeptChartLabels,
  themeColors: ChartThemeColors,
) {
  const colors = themeColors
  const depts = data.map(d => d.department)

  return {
    // tooltip：圆角、阴影、主题适配背景色（与趋势图一致）
    tooltip: {
      trigger: 'axis' as const,
      axisPointer: { type: 'shadow' as const },
      backgroundColor: colors.isDark ? '#1e293b' : '#ffffff',
      borderColor: colors.border || '#e2e8f0',
      borderWidth: 1,
      borderRadius: 8,
      padding: [8, 12],
      textStyle: {
        color: colors.textPrimary || '#0f172a',
        fontSize: 13,
      },
      extraCssText: 'box-shadow: 0 4px 12px rgba(0,0,0,0.12);',
    },

    // 图例：使用主题文字色
    legend: {
      bottom: 0,
      textStyle: {
        fontSize: 12,
        color: colors.textSecondary || '#475569',
      },
    },

    grid: { left: 100, right: 16, top: 16, bottom: 40 },

    // Y 轴：分类轴（横向柱状图的分类在 Y 轴），颜色从 chartColors 读取
    yAxis: {
      type: 'category' as const,
      data: depts,
      inverse: true,
      axisLine: { lineStyle: { color: colors.border || '#e2e8f0' } },
      axisLabel: { color: colors.textSecondary || '#475569' },
      axisTick: { lineStyle: { color: colors.border || '#e2e8f0' } },
    },

    // X 轴：数值轴，颜色从 chartColors 读取
    xAxis: {
      type: 'value' as const,
      minInterval: 1,
      axisLine: { show: false },
      axisLabel: { color: colors.textSecondary || '#475569' },
      splitLine: { lineStyle: { color: colors.border || '#e2e8f0', type: 'dashed' as const } },
    },

    // 全局动画配置
    animationDuration: 600,
    animationEasing: 'cubicOut' as const,

    // 为每个系列创建渐变色填充 + 圆角柱体（横向方向）
    series: [
      {
        name: labels.audit,
        type: 'bar' as const,
        stack: 'total',
        data: data.map(d => d.audit_count),
        barMaxWidth: 24,
        itemStyle: {
          // 使用 LinearGradient 实现从左到右的渐变色填充（横向柱状图方向）
          color: new LinearGradient(0, 0, 1, 0, [
            { offset: 0, color: hexToRgba(SERIES_COLORS.audit, 0.6) },
            { offset: 1, color: SERIES_COLORS.audit },
          ]),
          // 横向柱体圆角：右侧两个角为圆角 [左上, 右上, 右下, 左下]
          borderRadius: [0, 4, 4, 0] as [number, number, number, number],
        },
      },
      {
        name: labels.cron,
        type: 'bar' as const,
        stack: 'total',
        data: data.map(d => d.cron_count),
        barMaxWidth: 24,
        itemStyle: {
          color: new LinearGradient(0, 0, 1, 0, [
            { offset: 0, color: hexToRgba(SERIES_COLORS.cron, 0.6) },
            { offset: 1, color: SERIES_COLORS.cron },
          ]),
          borderRadius: [0, 4, 4, 0] as [number, number, number, number],
        },
      },
      {
        name: labels.archive,
        type: 'bar' as const,
        stack: 'total',
        data: data.map(d => d.archive_count),
        barMaxWidth: 24,
        itemStyle: {
          color: new LinearGradient(0, 0, 1, 0, [
            { offset: 0, color: hexToRgba(SERIES_COLORS.archive, 0.6) },
            { offset: 1, color: SERIES_COLORS.archive },
          ]),
          borderRadius: [0, 4, 4, 0] as [number, number, number, number],
        },
      },
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
        startAngle: 220,
        endAngle: -40,
        center: ['50%', '60%'],
        radius: '85%',
        min: 0,
        max: 100,
        // 粗圆弧进度条
        progress: {
          show: true,
          width: 16,
          roundCap: true,
          itemStyle: { color },
        },
        // 轨道背景
        axisLine: {
          roundCap: true,
          lineStyle: { width: 16, color: [[1, trackColor]] },
        },
        pointer: { show: false },
        axisTick: { show: false },
        splitLine: { show: false },
        axisLabel: { show: false },
        // 中心百分比数字
        detail: {
          valueAnimation: true,
          offsetCenter: [0, '-5%'],
          fontSize: 26,
          fontWeight: 'bold' as const,
          color,
          formatter: '{value}%',
        },
        // 指标名称
        title: {
          show: true,
          offsetCenter: [0, '25%'],
          fontSize: 13,
          fontWeight: 'normal' as const,
          color: themeColors.textSecondary || '#475569',
        },
        data: [{ value, name: label }],
        animationDuration: 800,
        animationEasing: 'cubicOut' as const,
      },
    ],
  }
}
