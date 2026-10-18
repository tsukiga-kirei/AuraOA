<script setup lang="ts">
import { use } from 'echarts/core'
import { BarChart, PieChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import { PieChartOutlined, BarChartOutlined } from '@ant-design/icons-vue'
import type { DeptDistributionData } from '~/types/dashboard-overview'
import { buildDeptChartOption, buildDeptDonutOption, SERIES_COLORS } from '~/utils/chart-builders'

use([BarChart, PieChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer])

// 引入主题色工具，获取当前主题下的配色方案
const { chartColors } = useThemeColors()
const { t } = useI18n()

// props：部门分布数据 / 图例标签 / 图表高度
interface Props {
  data: DeptDistributionData[]
  labels: { audit: string; cron: string; archive: string; summary: string; chat?: string }
  height?: string
}

const props = withDefaults(defineProps<Props>(), { height: '260px' })

// 视图切换：'dual' (左侧环图+右侧榜单，默认极美视图) | 'bar' (传统横向柱状图)
const viewMode = ref<'dual' | 'bar'>('dual')

// 横向堆叠柱状图 option
const barOption = computed(() =>
  buildDeptChartOption(props.data, props.labels, chartColors.value),
)

// 环形构成图 option
const donutOption = computed(() =>
  buildDeptDonutOption(props.data, props.labels, chartColors.value),
)

// 全租户总业务量
const grandTotal = computed(() => {
  return props.data.reduce(
    (acc, cur) =>
      acc + (cur.total ?? (cur.audit_count + cur.cron_count + cur.archive_count + cur.summary_count + (cur.chat_count ?? 0))),
    0,
  )
})

// 业务类型汇总明细（供左侧环图下方徽标展示）
const categoryBreakdown = computed(() => {
  const audit = props.data.reduce((a, c) => a + (c.audit_count || 0), 0)
  const summary = props.data.reduce((a, c) => a + (c.summary_count || 0), 0)
  const archive = props.data.reduce((a, c) => a + (c.archive_count || 0), 0)
  const cron = props.data.reduce((a, c) => a + (c.cron_count || 0), 0)
  const chat = props.data.reduce((a, c) => a + (c.chat_count || 0), 0)
  const total = grandTotal.value || 1

  const list = [
    { key: 'summary', name: props.labels.summary, count: summary, color: SERIES_COLORS.summary },
    { key: 'audit', name: props.labels.audit, count: audit, color: SERIES_COLORS.audit },
    { key: 'chat', name: props.labels.chat || '智能体', count: chat, color: SERIES_COLORS.chat },
    { key: 'cron', name: props.labels.cron, count: cron, color: SERIES_COLORS.cron },
    { key: 'archive', name: props.labels.archive, count: archive, color: SERIES_COLORS.archive },
  ].filter(item => item.count > 0)

  return list.map(item => ({
    ...item,
    pct: ((item.count / total) * 100).toFixed(0),
  }))
})

// 按总业务量降序排序的部门数据
const sortedDepts = computed(() => {
  return [...props.data]
    .map(d => ({
      ...d,
      totalVal: d.total ?? (d.audit_count + d.cron_count + d.archive_count + d.summary_count + (d.chat_count ?? 0)),
    }))
    .filter(d => d.totalVal > 0)
    .sort((a, b) => b.totalVal - a.totalVal)
})

// 取 Top 5 活跃部门（卡片纵向排布高度刚好饱满）
const topDepts = computed(() => sortedDepts.value.slice(0, 5))

// 部门业务量基准值（以 Top 1 为 100% 相对长度基准）
const maxDeptTotal = computed(() => {
  if (topDepts.value.length === 0) return 1
  return topDepts.value[0].totalVal || 1
})

// 计算单部门的分段进度条数据
function getDeptSegments(dept: typeof sortedDepts.value[0]) {
  const total = dept.totalVal || 1
  const segs = [
    { count: dept.summary_count, color: SERIES_COLORS.summary, name: props.labels.summary },
    { count: dept.audit_count, color: SERIES_COLORS.audit, name: props.labels.audit },
    { count: dept.chat_count ?? 0, color: SERIES_COLORS.chat, name: props.labels.chat || '智能体' },
    { count: dept.cron_count, color: SERIES_COLORS.cron, name: props.labels.cron },
    { count: dept.archive_count, color: SERIES_COLORS.archive, name: props.labels.archive },
  ].filter(s => s.count > 0)

  return segs.map(s => ({
    ...s,
    width: `${(s.count / total) * 100}%`,
  }))
}

// 部门分段 Tooltip 内容
function getDeptTooltipText(dept: typeof sortedDepts.value[0]) {
  const lines = [
    `${dept.department} · 共 ${dept.totalVal} 次`,
  ]
  if (dept.summary_count > 0) lines.push(`● ${props.labels.summary}: ${dept.summary_count} 次`)
  if (dept.audit_count > 0) lines.push(`● ${props.labels.audit}: ${dept.audit_count} 次`)
  if ((dept.chat_count ?? 0) > 0) lines.push(`● ${props.labels.chat || '智能体'}: ${dept.chat_count} 次`)
  if (dept.cron_count > 0) lines.push(`● ${props.labels.cron}: ${dept.cron_count} 次`)
  if (dept.archive_count > 0) lines.push(`● ${props.labels.archive}: ${dept.archive_count} 次`)
  return lines.join('\n')
}
</script>

<template>
  <div class="dept-distribution-wrapper">
    <!-- 顶部视图切换工具栏 -->
    <div class="dept-chart-toolbar">
      <div class="chart-type-capsule">
        <button
          type="button"
          class="capsule-btn"
          :class="{ 'capsule-btn--active': viewMode === 'dual' }"
          :title="t('overview.chartType.dual')"
          @click="viewMode = 'dual'"
        >
          <PieChartOutlined />
        </button>
        <button
          type="button"
          class="capsule-btn"
          :class="{ 'capsule-btn--active': viewMode === 'bar' }"
          :title="t('overview.chartType.bar')"
          @click="viewMode = 'bar'"
        >
          <BarChartOutlined />
        </button>
      </div>
    </div>

    <!-- 1. 默认高颜值架构：左侧业务构成 Donut 环图 + 右侧 Top 活跃部门分段进度榜单 -->
    <div v-if="viewMode === 'dual'" class="dual-view-container" :style="{ minHeight: height }">
      <!-- 左侧：全租户业务类型构成环图 -->
      <div class="donut-section">
        <div class="donut-chart-box">
          <VChart :option="donutOption" class="donut-vchart" autoresize />
          <div class="donut-center-metric">
            <span class="donut-center-label">{{ t('overview.deptTotalProcess') }}</span>
            <span class="donut-center-value">{{ grandTotal }}</span>
          </div>
        </div>
        <!-- 业务构成图例微胶囊 -->
        <div class="donut-legend-chips">
          <div
            v-for="cat in categoryBreakdown"
            :key="cat.key"
            class="donut-chip"
            :title="`${cat.name}: ${cat.count}次 (${cat.pct}%)`"
          >
            <span class="chip-dot" :style="{ background: cat.color }" />
            <span class="chip-name">{{ cat.name }}</span>
            <span class="chip-pct">{{ cat.pct }}%</span>
          </div>
        </div>
      </div>

      <!-- 右侧：Top 活跃部门分段进度排行榜 -->
      <div class="leaderboard-section">
        <div class="leaderboard-title">
          <span>{{ t('overview.deptRankTitle') }}</span>
          <span class="leaderboard-sub">TOP {{ topDepts.length }}</span>
        </div>

        <div v-if="topDepts.length === 0" class="empty-hint">
          {{ t('overview.noData') }}
        </div>

        <div v-else class="dept-rank-list">
          <div
            v-for="(dept, idx) in topDepts"
            :key="dept.department"
            class="dept-rank-row"
          >
            <!-- 排名胶囊 + 部门名 + 总次数 -->
            <div class="dept-rank-meta">
              <span class="rank-badge" :class="`rank-badge--${idx + 1}`">
                0{{ idx + 1 }}
              </span>
              <span class="dept-name" :title="dept.department">{{ dept.department }}</span>
              <span class="dept-total-count">
                <strong>{{ dept.totalVal }}</strong> <span class="unit-text">{{ t('overview.times') }}</span>
              </span>
            </div>

            <!-- 分段彩色进度条（带悬浮详细 Tooltip） -->
            <a-tooltip :title="getDeptTooltipText(dept)" placement="top">
              <div class="dept-progress-track">
                <div
                  class="dept-progress-bar"
                  :style="{ width: `${Math.max(8, (dept.totalVal / maxDeptTotal) * 100)}%` }"
                >
                  <span
                    v-for="(seg, sIdx) in getDeptSegments(dept)"
                    :key="sIdx"
                    class="seg-item"
                    :style="{ width: seg.width, background: seg.color }"
                  />
                </div>
              </div>
            </a-tooltip>
          </div>
        </div>
      </div>
    </div>

    <!-- 2. 经典横向柱状图视图 -->
    <div v-else :style="{ height, width: '100%' }">
      <VChart :option="barOption" :style="{ height, width: '100%' }" autoresize />
    </div>
  </div>
</template>

<style scoped>
.dept-distribution-wrapper {
  position: relative;
  width: 100%;
}

.dept-chart-toolbar {
  position: absolute;
  top: -36px;
  right: 0;
  z-index: 10;
}

.chart-type-capsule {
  display: inline-flex;
  align-items: center;
  background: var(--color-bg-page);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-sm);
  padding: 2px;
  gap: 2px;
}

.capsule-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 20px;
  border: none;
  background: transparent;
  color: var(--color-text-tertiary);
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  transition: all var(--transition-fast);
}

.capsule-btn:hover {
  color: var(--color-text-primary);
}

.capsule-btn--active {
  background: var(--color-bg-card);
  color: var(--color-primary);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.06);
}

/* 左右双栏联动容器 */
.dual-view-container {
  display: flex;
  gap: 24px;
  align-items: stretch;
  width: 100%;
  padding-top: 4px;
}

/* 左侧：环形图区域 */
.donut-section {
  flex: 0 0 42%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: space-between;
  border-right: 1px dashed var(--color-border-light);
  padding-right: 16px;
}

.donut-chart-box {
  position: relative;
  width: 100%;
  height: 170px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.donut-vchart {
  width: 100%;
  height: 100%;
}

.donut-center-metric {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  display: flex;
  flex-direction: column;
  align-items: center;
  pointer-events: none;
}

.donut-center-label {
  font-size: 11px;
  color: var(--color-text-tertiary);
  margin-bottom: 2px;
}

.donut-center-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-text-primary);
  font-family: var(--font-mono, monospace);
  line-height: 1;
}

.donut-legend-chips {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 6px;
  width: 100%;
  margin-top: 4px;
}

.donut-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 6px;
  background: var(--color-bg-page);
  border-radius: var(--radius-sm);
  font-size: 11px;
  color: var(--color-text-secondary);
}

.chip-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.chip-pct {
  font-weight: 600;
  color: var(--color-text-primary);
}

/* 右侧：部门活跃榜单 */
.leaderboard-section {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

.leaderboard-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-secondary);
  margin-bottom: 8px;
}

.leaderboard-sub {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 4px;
  background: var(--color-bg-page);
  color: var(--color-text-tertiary);
  font-weight: 600;
}

.empty-hint {
  text-align: center;
  color: var(--color-text-tertiary);
  padding: 30px 0;
  font-size: 12px;
}

.dept-rank-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  flex: 1;
  justify-content: space-around;
}

.dept-rank-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.dept-rank-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}

.rank-badge {
  font-size: 10px;
  font-weight: 700;
  padding: 1px 5px;
  border-radius: 4px;
  font-family: var(--font-mono, monospace);
  background: var(--color-bg-page);
  color: var(--color-text-tertiary);
}

.rank-badge--1 {
  background: #fef3c7;
  color: #d97706;
}

.rank-badge--2 {
  background: #f1f5f9;
  color: #475569;
}

.rank-badge--3 {
  background: #ffedd5;
  color: #ea580c;
}

.dept-name {
  flex: 1;
  min-width: 0;
  font-weight: 500;
  color: var(--color-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.dept-total-count {
  font-size: 12px;
  color: var(--color-text-primary);
  flex-shrink: 0;
}

.unit-text {
  font-size: 11px;
  color: var(--color-text-tertiary);
  font-weight: normal;
}

/* 分段彩色进度条 */
.dept-progress-track {
  width: 100%;
  height: 7px;
  background: var(--color-bg-page);
  border-radius: 4px;
  overflow: hidden;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.dept-progress-track:hover {
  height: 9px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

.dept-progress-bar {
  height: 100%;
  display: flex;
  border-radius: 4px;
  overflow: hidden;
  transition: width 0.6s cubic-bezier(0.16, 1, 0.3, 1);
}

.seg-item {
  height: 100%;
  transition: opacity 0.2s;
}

.seg-item:hover {
  opacity: 0.85;
}
</style>

