<script setup lang="ts">
import { use } from 'echarts/core'
import { BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import { buildStackedBarOption } from '~/utils/chart-builders'

use([BarChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer])

// 引入主题色工具，获取当前主题下的配色方案
const { chartColors } = useThemeColors()

// props：X 轴分类标签 / 系列数据（含名称、数值、颜色）/ 图表高度
interface Props {
  categories: string[]
  series: { name: string; data: number[]; color: string }[]
  height?: string
}

const props = withDefaults(defineProps<Props>(), { height: '240px' })

// 根据传入数据和当前主题色构建 ECharts 纵向堆叠柱状图配置
const option = computed(() =>
  buildStackedBarOption(props.categories, props.series, chartColors.value),
)
</script>

<template>
  <VChart :option="option" :style="{ height, width: '100%' }" autoresize />
</template>
