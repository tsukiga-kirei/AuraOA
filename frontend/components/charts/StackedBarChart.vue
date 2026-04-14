<script setup lang="ts">
import { use } from 'echarts/core'
import { BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'

use([BarChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer])

// props：X 轴分类标签 / 系列数据（含名称、数值、颜色）/ 图表高度
interface Props {
  categories: string[]
  series: { name: string; data: number[]; color: string }[]
  height?: string
}

const props = withDefaults(defineProps<Props>(), { height: '240px' })

// 根据传入数据构建 ECharts 纵向堆叠柱状图配置
const option = computed(() => ({
  tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
  legend: { bottom: 0, textStyle: { fontSize: 12 } },
  grid: { left: 40, right: 16, top: 16, bottom: 40 },
  xAxis: { type: 'category', data: props.categories },
  yAxis: { type: 'value', minInterval: 1 },
  series: props.series.map(s => ({
    name: s.name,
    type: 'bar',
    stack: 'total',
    data: s.data,
    itemStyle: { color: s.color },
    barMaxWidth: 32,
  })),
}))
</script>

<template>
  <VChart :option="option" :style="{ height, width: '100%' }" autoresize />
</template>
