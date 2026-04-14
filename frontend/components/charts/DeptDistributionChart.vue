<script setup lang="ts">
import { use } from 'echarts/core'
import { BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import type { DeptDistributionData } from '~/types/dashboard-overview'

use([BarChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer])

// props：部门分布数据 / 图例标签 / 图表高度
interface Props {
  data: DeptDistributionData[]
  labels: { audit: string; cron: string; archive: string }
  height?: string
}

const props = withDefaults(defineProps<Props>(), { height: '300px' })

// 根据传入数据构建 ECharts 横向堆叠柱状图配置
const option = computed(() => {
  const depts = props.data.map(d => d.department)
  return {
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    legend: { bottom: 0, textStyle: { fontSize: 12 } },
    grid: { left: 100, right: 16, top: 16, bottom: 40 },
    yAxis: { type: 'category', data: depts, inverse: true },
    xAxis: { type: 'value', minInterval: 1 },
    series: [
      { name: props.labels.audit, type: 'bar', stack: 'total', data: props.data.map(d => d.audit_count), itemStyle: { color: '#4f46e5' }, barMaxWidth: 24 },
      { name: props.labels.cron, type: 'bar', stack: 'total', data: props.data.map(d => d.cron_count), itemStyle: { color: '#06b6d4' }, barMaxWidth: 24 },
      { name: props.labels.archive, type: 'bar', stack: 'total', data: props.data.map(d => d.archive_count), itemStyle: { color: '#10b981' }, barMaxWidth: 24 },
    ],
  }
})
</script>

<template>
  <VChart :option="option" :style="{ height, width: '100%' }" autoresize />
</template>
