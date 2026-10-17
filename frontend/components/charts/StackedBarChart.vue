<script setup lang="ts">
import { use } from 'echarts/core'
import { BarChart, LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import { LineChartOutlined, BarChartOutlined } from '@ant-design/icons-vue'
import { buildStackedBarOption, buildTrendLineOption } from '~/utils/chart-builders'

use([BarChart, LineChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer])

// 引入主题色工具，获取当前主题下的配色方案
const { chartColors } = useThemeColors()
const { t } = useI18n()

// props：X 轴分类标签 / 系列数据（含名称、数值、颜色）/ 图表高度 / 初始类型
interface Props {
  categories: string[]
  series: { name: string; data: number[]; color: string }[]
  height?: string
  defaultType?: 'line' | 'bar'
}

const props = withDefaults(defineProps<Props>(), {
  height: '240px',
  defaultType: 'line',
})

const currentType = ref<'line' | 'bar'>(props.defaultType)

// 根据选择的类型动态构建平滑面积折线图或堆叠柱状图
const option = computed(() => {
  if (currentType.value === 'line') {
    return buildTrendLineOption(props.categories, props.series, chartColors.value)
  }
  return buildStackedBarOption(props.categories, props.series, chartColors.value)
})
</script>

<template>
  <div class="trend-chart-wrapper">
    <div class="trend-chart-toolbar">
      <div class="chart-type-capsule">
        <button
          type="button"
          class="capsule-btn"
          :class="{ 'capsule-btn--active': currentType === 'line' }"
          :title="t('overview.chartType.line')"
          @click="currentType = 'line'"
        >
          <LineChartOutlined />
        </button>
        <button
          type="button"
          class="capsule-btn"
          :class="{ 'capsule-btn--active': currentType === 'bar' }"
          :title="t('overview.chartType.bar')"
          @click="currentType = 'bar'"
        >
          <BarChartOutlined />
        </button>
      </div>
    </div>
    <VChart :option="option" :style="{ height, width: '100%' }" autoresize />
  </div>
</template>

<style scoped>
.trend-chart-wrapper {
  position: relative;
  width: 100%;
}

.trend-chart-toolbar {
  position: absolute;
  top: 0;
  left: 0;
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
</style>
