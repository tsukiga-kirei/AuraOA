<script setup lang="ts">
import { use } from 'echarts/core'
import { BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'

use([BarChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer])

interface Props {
  categories: string[]
  series: { name: string; data: number[]; color: string }[]
  height?: string
}

const props = withDefaults(defineProps<Props>(), { height: '240px' })

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
