<script setup lang="ts">
import { use } from 'echarts/core'
import { BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import type { DeptDistributionData } from '~/types/dashboard-overview'
import { buildDeptChartOption } from '~/utils/chart-builders'

use([BarChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer])

// 引入主题色工具，获取当前主题下的配色方案
const { chartColors } = useThemeColors()

// props：部门分布数据 / 图例标签 / 图表高度
interface Props {
  data: DeptDistributionData[]
  labels: { audit: string; cron: string; archive: string }
  height?: string
}

const props = withDefaults(defineProps<Props>(), { height: '300px' })

// 根据传入数据和当前主题色构建 ECharts 横向堆叠柱状图配置
const option = computed(() =>
  buildDeptChartOption(props.data, props.labels, chartColors.value),
)
</script>

<template>
  <VChart :option="option" :style="{ height, width: '100%' }" autoresize />
</template>
