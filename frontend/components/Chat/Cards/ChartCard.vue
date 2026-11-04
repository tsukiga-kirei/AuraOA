<script setup lang="ts">
import { ref, computed } from 'vue'
import { use } from 'echarts/core'
import { BarChart, LineChart, PieChart, RadarChart } from 'echarts/charts'
import {
  GridComponent,
  TooltipComponent,
  LegendComponent,
  RadarComponent,
  TitleComponent,
} from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import {
  BarChartOutlined,
  LineChartOutlined,
  PieChartOutlined,
  RadarChartOutlined,
  TableOutlined,
  CameraOutlined,
  DownloadOutlined,
  DotChartOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { normalizeChartPayload, type ChartPayload } from '~/utils/chatUiProtocol'

use([
  BarChart,
  LineChart,
  PieChart,
  RadarChart,
  GridComponent,
  TooltipComponent,
  LegendComponent,
  RadarComponent,
  TitleComponent,
  CanvasRenderer,
])

const props = withDefaults(
  defineProps<{
    // 解析器已归一化；这里仍做一次幂等归一化，兼容历史消息与外部直传数据
    data: Record<string, any>
    height?: string
  }>(),
  {
    height: '280px',
  }
)

const { t } = useI18n()
const { chartColors, isDark } = useThemeColors()
const vchartRef = ref<any>(null)

const payload = computed<ChartPayload>(() => normalizeChartPayload(props.data))

// 视图模式：'chart' | 'table'
const viewMode = ref<'chart' | 'table'>('chart')

// 调色盘
const PALETTE = [
  '#4f46e5',
  '#06b6d4',
  '#10b981',
  '#f59e0b',
  '#8b5cf6',
  '#ec4899',
  '#3b82f6',
  '#14b8a6',
]

// 维度与指标数据（归一化后已保证 categories / series / indicators 结构一致）
const resolvedCategories = computed<string[]>(() => payload.value.categories)
const resolvedSeries = computed(() => payload.value.series)

// 当前图表类型，支持用户点击快速在 bar / line / pie 间切换
const currentChartType = ref<'bar' | 'line' | 'pie' | 'radar'>(payload.value.type)

// 没有可绘制数据时展示空状态，而不是留一块空白画布
const hasChartData = computed(() => {
  if (!resolvedSeries.value.length) return false
  if (currentChartType.value === 'radar') return payload.value.indicators.length > 0
  return resolvedCategories.value.length > 0
})

// 无有效分类但带了表格数据时默认切为表格视图
if (!resolvedCategories.value.length && payload.value.rows.length) {
  viewMode.value = 'table'
}

// ── 构建 ECharts Option ──
const chartOption = computed(() => {
  const cats = resolvedCategories.value
  const seriesItems = resolvedSeries.value
  const type = currentChartType.value
  const textColor = chartColors.value.textPrimary || (isDark.value ? '#e5e7eb' : '#374151')
  const textSubColor = chartColors.value.textSecondary || (isDark.value ? '#9ca3af' : '#6b7280')
  const borderColor = chartColors.value.border || (isDark.value ? '#374151' : '#e5e7eb')

  const baseOption: any = {
    color: PALETTE,
    backgroundColor: 'transparent',
    tooltip: {
      trigger: type === 'pie' ? 'item' : 'axis',
      backgroundColor: isDark.value ? 'rgba(31, 41, 55, 0.95)' : 'rgba(255, 255, 255, 0.95)',
      borderColor,
      borderWidth: 1,
      textStyle: {
        color: textColor,
        fontSize: 12,
      },
      padding: [8, 12],
    },
    legend: {
      show: seriesItems.length > 1 || type === 'pie',
      top: 0,
      right: 0,
      textStyle: { color: textSubColor, fontSize: 11 },
      icon: 'circle',
      itemWidth: 8,
      itemHeight: 8,
    },
  }

  // 1. 柱状图
  if (type === 'bar') {
    return {
      ...baseOption,
      grid: {
        top: seriesItems.length > 1 ? 32 : 16,
        left: '3%',
        right: '4%',
        bottom: '8%',
        containLabel: true,
      },
      xAxis: {
        type: 'category',
        data: cats,
        axisLine: { lineStyle: { color: borderColor } },
        axisLabel: {
          color: textSubColor,
          fontSize: 11,
          interval: 0,
          rotate: cats.length > 5 ? 25 : 0,
        },
      },
      yAxis: {
        type: 'value',
        splitLine: { lineStyle: { color: borderColor, type: 'dashed' } },
        axisLabel: { color: textSubColor, fontSize: 11 },
      },
      series: seriesItems.map((s, idx) => ({
        name: s.name,
        type: 'bar',
        barMaxWidth: 28,
        itemStyle: {
          borderRadius: [4, 4, 0, 0],
          color: s.color || PALETTE[idx % PALETTE.length],
        },
        data: s.data,
      })),
    }
  }

  // 2. 折线图
  if (type === 'line') {
    return {
      ...baseOption,
      grid: {
        top: seriesItems.length > 1 ? 32 : 16,
        left: '3%',
        right: '4%',
        bottom: '8%',
        containLabel: true,
      },
      xAxis: {
        type: 'category',
        data: cats,
        boundaryGap: false,
        axisLine: { lineStyle: { color: borderColor } },
        axisLabel: {
          color: textSubColor,
          fontSize: 11,
          interval: 0,
          rotate: cats.length > 5 ? 25 : 0,
        },
      },
      yAxis: {
        type: 'value',
        splitLine: { lineStyle: { color: borderColor, type: 'dashed' } },
        axisLabel: { color: textSubColor, fontSize: 11 },
      },
      series: seriesItems.map((s, idx) => ({
        name: s.name,
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        itemStyle: { color: s.color || PALETTE[idx % PALETTE.length] },
        areaStyle: {
          opacity: 0.15,
          color: s.color || PALETTE[idx % PALETTE.length],
        },
        data: s.data,
      })),
    }
  }

  // 3. 饼图 / 环形图
  if (type === 'pie') {
    const pieData = cats.map((cat, idx) => {
      const val = seriesItems[0]?.data[idx]
      const numVal = typeof val === 'object' && val !== null ? val.value : Number(val) || 0
      return {
        name: cat,
        value: numVal,
        itemStyle: { color: PALETTE[idx % PALETTE.length] },
      }
    })

    return {
      ...baseOption,
      tooltip: {
        ...baseOption.tooltip,
        formatter: '{b}: <b>{c}</b> ({d}%)',
      },
      legend: {
        orient: 'vertical',
        right: '2%',
        top: 'middle',
        textStyle: { color: textSubColor, fontSize: 11 },
      },
      series: [
        {
          name: payload.value.title || '数据占比',
          type: 'pie',
          radius: ['40%', '68%'],
          center: ['40%', '50%'],
          avoidLabelOverlap: true,
          itemStyle: {
            borderRadius: 4,
            borderColor: chartColors.value.bgCard || '#ffffff',
            borderWidth: 2,
          },
          label: {
            show: false,
          },
          labelLine: {
            show: false,
          },
          data: pieData,
        },
      ],
    }
  }

  // 4. 雷达图
  if (type === 'radar') {
    const indicators = payload.value.indicators.length
      ? payload.value.indicators
      : cats.map(name => ({ name, max: 100 }))
    return {
      ...baseOption,
      radar: {
        indicator: indicators,
        shape: 'polygon',
        splitNumber: 4,
        axisName: {
          color: textSubColor,
          fontSize: 11,
        },
        splitLine: {
          lineStyle: { color: borderColor },
        },
        splitArea: {
          show: true,
          areaStyle: {
            color: isDark.value
              ? ['rgba(255,255,255,0.02)', 'rgba(255,255,255,0.05)']
              : ['rgba(0,0,0,0.01)', 'rgba(0,0,0,0.03)'],
          },
        },
        axisLine: {
          lineStyle: { color: borderColor },
        },
      },
      series: [
        {
          name: payload.value.title || '多维评估',
          type: 'radar',
          data: seriesItems.map((s, idx) => ({
            name: s.name,
            value: s.data,
            itemStyle: { color: s.color || PALETTE[idx % PALETTE.length] },
            areaStyle: { opacity: 0.25 },
          })),
        },
      ],
    }
  }

  return baseOption
})

// ── 表格视图数据计算 ──
const tableColumns = computed(() => {
  if (payload.value.columns.length) {
    return payload.value.columns
  }

  // 逆向自 categories & series 生成表格列
  const cols: any[] = [
    {
      title: t('chat.chart.dimensionColumn', '维度 / 项目'),
      dataIndex: 'category',
      key: 'category',
    },
  ]
  resolvedSeries.value.forEach(s => {
    cols.push({
      title: s.name,
      dataIndex: s.name,
      key: s.name,
      sorter: (a: any, b: any) => Number(a[s.name] || 0) - Number(b[s.name] || 0),
    })
  })
  return cols
})

const tableRows = computed(() => {
  if (payload.value.rows.length) {
    return payload.value.rows
  }
  const cats = resolvedCategories.value
  const seriesItems = resolvedSeries.value
  return cats.map((cat, idx) => {
    const row: Record<string, any> = { category: cat }
    seriesItems.forEach(s => {
      row[s.name] = s.data[idx] ?? 0
    })
    return row
  })
})

// ── 导出图片与 CSV ──
const exportImage = () => {
  if (!vchartRef.value) {
    message.warning('图表正在加载中，请稍后再试')
    return
  }
  try {
    const url = vchartRef.value.getDataURL({
      type: 'png',
      pixelRatio: 2,
      backgroundColor: isDark.value ? '#1e1e24' : '#ffffff',
    })
    const link = document.createElement('a')
    link.href = url
    link.download = `${payload.value.title || 'analysis_chart'}_${Date.now()}.png`
    link.click()
    message.success(t('chat.chart.exportedImage', '已成功导出高清图表图片'))
  } catch {
    message.error('导出图表图片失败')
  }
}

const exportCsv = () => {
  const rows = tableRows.value
  const cols = tableColumns.value
  if (!rows.length) {
    message.warning(t('common.noData', '暂无可导出的数据'))
    return
  }
  const colHeaders: string[] = cols.map(c => (typeof c === 'string' ? c : c.title || c.dataIndex))
  const colKeys: string[] = cols.map(c => (typeof c === 'string' ? c : c.dataIndex || c.key))

  const escapeCsv = (val: any) => {
    if (val === null || val === undefined) return ''
    const s = String(val).replace(/"/g, '""')
    return `"${s}"`
  }

  const csvContent =
    '\uFEFF' +
    [colHeaders.map(escapeCsv).join(','), ...rows.map(row => colKeys.map(k => escapeCsv(row[k])).join(','))].join(
      '\n'
    )

  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `${payload.value.title || 'analysis_data'}_${Date.now()}.csv`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
  message.success(t('chat.formConfirm.exported', '已成功导出 CSV'))
}
</script>

<template>
  <div class="analysis-chart-card">
    <!-- 头部工具栏 -->
    <div class="chart-header">
      <div class="header-left">
        <DotChartOutlined class="header-icon" />
        <span class="header-title">{{ payload.title || t('chat.chart.defaultTitle', '数据分析与统计') }}</span>
        <span v-if="payload.subtitle" class="header-subtitle">{{ payload.subtitle }}</span>
      </div>

      <div class="header-actions">
        <!-- 图表类型切换（仅当处于图表视图且非雷达图时提供快速切换） -->
        <div v-if="viewMode === 'chart' && currentChartType !== 'radar'" class="type-capsule">
          <button
            type="button"
            class="capsule-item"
            :class="{ active: currentChartType === 'bar' }"
            :title="t('chat.chart.bar', '柱状图')"
            @click="currentChartType = 'bar'"
          >
            <BarChartOutlined />
          </button>
          <button
            type="button"
            class="capsule-item"
            :class="{ active: currentChartType === 'line' }"
            :title="t('chat.chart.line', '折线图')"
            @click="currentChartType = 'line'"
          >
            <LineChartOutlined />
          </button>
          <button
            type="button"
            class="capsule-item"
            :class="{ active: currentChartType === 'pie' }"
            :title="t('chat.chart.pie', '饼图')"
            @click="currentChartType = 'pie'"
          >
            <PieChartOutlined />
          </button>
        </div>

        <!-- 视图模式切换：图表 ⇋ 表格 -->
        <div class="mode-capsule">
          <button
            type="button"
            class="mode-item"
            :class="{ active: viewMode === 'chart' }"
            @click="viewMode = 'chart'"
          >
            <BarChartOutlined class="mode-icon" />
            <span>{{ t('chat.chart.viewChart', '图表') }}</span>
          </button>
          <button
            type="button"
            class="mode-item"
            :class="{ active: viewMode === 'table' }"
            @click="viewMode = 'table'"
          >
            <TableOutlined class="mode-icon" />
            <span>{{ t('chat.chart.viewTable', '表格') }}</span>
          </button>
        </div>

        <!-- 导出动作 -->
        <a-dropdown placement="bottomRight" :trigger="['click']">
          <a-button size="small" class="export-dropdown-btn">
            <DownloadOutlined />
          </a-button>
          <template #overlay>
            <a-menu>
              <a-menu-item key="img" @click="exportImage">
                <CameraOutlined style="margin-right: 6px;" />
                {{ t('chat.chart.exportImage', '导出高清图表 (PNG)') }}
              </a-menu-item>
              <a-menu-item key="csv" @click="exportCsv">
                <DownloadOutlined style="margin-right: 6px;" />
                {{ t('chat.chart.exportCsv', '导出数据明细 (CSV)') }}
              </a-menu-item>
            </a-menu>
          </template>
        </a-dropdown>
      </div>
    </div>

    <!-- 1. 图表主体渲染 -->
    <div v-show="viewMode === 'chart'" class="chart-content">
      <VChart
        v-if="hasChartData"
        ref="vchartRef"
        :option="chartOption"
        :style="{ height, width: '100%' }"
        autoresize
      />
      <div v-else class="chart-empty" :style="{ height }">
        {{ t('chat.chart.noData', '暂无可绘制的数据，可切换到表格查看明细') }}
      </div>
    </div>

    <!-- 2. 二维表格渲染 -->
    <div v-show="viewMode === 'table'" class="table-content">
      <a-table
        :data-source="tableRows"
        :columns="tableColumns"
        size="small"
        :pagination="{ pageSize: 8, size: 'small', showTotal: total => `共 ${total} 项` }"
        :scroll="{ x: 'max-content' }"
        bordered
      />
    </div>
  </div>
</template>

<style scoped>
.analysis-chart-card {
  margin: 12px 0;
  border-radius: 12px;
  background: var(--color-bg-container, #ffffff);
  border: 1px solid var(--color-border-light, #ebe5df);
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.03);
  padding: 14px 16px;
  overflow: hidden;
}

.chart-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--color-border-subtle, #f5f2ed);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.header-icon {
  font-size: 16px;
  color: var(--color-primary, #5b5bd6);
}

.header-title {
  font-size: 13.5px;
  font-weight: 600;
  color: var(--color-text-primary, #1f1d1a);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.header-subtitle {
  font-size: 12px;
  color: var(--color-text-tertiary, #8c8680);
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.type-capsule,
.mode-capsule {
  display: inline-flex;
  align-items: center;
  background: var(--color-bg-page, #f5f3f0);
  border: 1px solid var(--color-border-light, #ebe5df);
  border-radius: 6px;
  padding: 2px;
  gap: 2px;
}

.capsule-item {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 22px;
  border: none;
  background: transparent;
  color: var(--color-text-tertiary, #8c8680);
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  transition: all 0.2s;
}

.capsule-item:hover {
  color: var(--color-text-primary, #1f1d1a);
}

.capsule-item.active {
  background: var(--color-bg-container, #ffffff);
  color: var(--color-primary, #5b5bd6);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

.mode-item {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  height: 22px;
  border: none;
  background: transparent;
  color: var(--color-text-tertiary, #8c8680);
  border-radius: 4px;
  cursor: pointer;
  font-size: 11.5px;
  transition: all 0.2s;
}

.mode-item:hover {
  color: var(--color-text-primary, #1f1d1a);
}

.mode-item.active {
  background: var(--color-bg-container, #ffffff);
  color: var(--color-primary, #5b5bd6);
  font-weight: 500;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}

.mode-icon {
  font-size: 12px;
}

.export-dropdown-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  padding: 0 6px;
  height: 26px;
}

.chart-content {
  width: 100%;
}

.chart-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  color: var(--color-text-tertiary, #8c8680);
  background: var(--color-bg-subtle, #faf8f5);
  border: 1px dashed var(--color-border-subtle, #ebe5df);
  border-radius: 8px;
}

.table-content {
  margin-top: 4px;
}
</style>
