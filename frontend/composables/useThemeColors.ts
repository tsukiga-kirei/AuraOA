/**
 * useThemeColors — 从 CSS 自定义属性读取当前主题色值。
 *
 * ECharts 组件通过此工具获取与 Theme_System 一致的配色，
 * 确保图表在亮色/暗色主题切换时自动适配。
 * SSR 环境下返回 fallback 默认色值，避免服务端渲染报错。
 */
export const useThemeColors = () => {
  const { isDark } = useTheme()

  /**
   * 读取指定 CSS 自定义属性的当前计算值。
   * SSR 环境下（无 window 对象）返回空字符串。
   */
  function getCssVar(name: string): string {
    if (typeof window === 'undefined') return ''
    return getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  }

  /** 当前主题下的图表配色方案（响应式，主题切换时自动更新） */
  const chartColors = computed(() => ({
    primary: getCssVar('--color-primary') || '#4f46e5',
    primaryLight: getCssVar('--color-primary-light') || '#6366f1',
    accent: getCssVar('--color-accent') || '#06b6d4',
    success: getCssVar('--color-success') || '#10b981',
    warning: getCssVar('--color-warning') || '#f59e0b',
    danger: getCssVar('--color-danger') || '#ef4444',
    textPrimary: getCssVar('--color-text-primary'),
    textSecondary: getCssVar('--color-text-secondary'),
    textTertiary: getCssVar('--color-text-tertiary'),
    border: getCssVar('--color-border'),
    bgCard: getCssVar('--color-bg-card'),
    isDark: isDark.value,
  }))

  return { chartColors, getCssVar, isDark }
}

// ── 系统监控阈值常量与颜色判断纯函数 ──

/** 阈值配置类型 */
export interface MetricThresholds {
  readonly warning: number
  readonly danger: number
}

/** 系统监控指标阈值（CPU / 内存 / 磁盘） */
export const MONITOR_THRESHOLDS = {
  cpu: { warning: 80, danger: 95 },
  memory: { warning: 85, danger: 95 },
  disk: { warning: 90, danger: 95 },
} as const

/** 默认语义色值（与 CSS 自定义属性 fallback 一致） */
const SEMANTIC_COLORS = {
  success: '#10b981',
  warning: '#f59e0b',
  danger: '#ef4444',
} as const

/**
 * 根据指标值和阈值返回对应的语义颜色。
 * - value ≤ warning  → 正常色（success / 绿色）
 * - value > warning 且 ≤ danger → 警告色（warning / 黄色）
 * - value > danger   → 危险色（danger / 红色）
 */
export function getMetricColor(
  value: number,
  thresholds: MetricThresholds,
): string {
  if (value > thresholds.danger) return SEMANTIC_COLORS.danger
  if (value > thresholds.warning) return SEMANTIC_COLORS.warning
  return SEMANTIC_COLORS.success
}

/**
 * 根据服务运行状态返回对应的语义颜色。
 * - 'online'   → 成功色（绿色）
 * - 'offline'  → 危险色（红色）
 * - 'degraded' → 警告色（黄色）
 */
export function getServiceStatusColor(
  status: 'online' | 'offline' | 'degraded',
): string {
  switch (status) {
    case 'online':
      return SEMANTIC_COLORS.success
    case 'offline':
      return SEMANTIC_COLORS.danger
    case 'degraded':
      return SEMANTIC_COLORS.warning
  }
}
