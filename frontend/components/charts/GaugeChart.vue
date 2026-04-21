<script setup lang="ts">
import { use } from 'echarts/core'
import { GaugeChart } from 'echarts/charts'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import { buildGaugeOption } from '~/utils/chart-builders'
import type { GaugeThresholds } from '~/utils/chart-builders'

use([GaugeChart, CanvasRenderer])

// 引入主题色工具，获取当前主题下的配色方案
const { chartColors } = useThemeColors()

// props：指标值（0-100）/ 指标名称 / 阈值配置 / 图表高度
interface Props {
  value: number
  label: string
  thresholds?: GaugeThresholds
  height?: string
}

const props = withDefaults(defineProps<Props>(), { height: '200px' })

// 根据传入值、阈值和当前主题色构建 ECharts gauge 仪表盘配置
const option = computed(() =>
  buildGaugeOption(props.value, props.label, props.thresholds, chartColors.value),
)
</script>

<template>
  <VChart :option="option" :style="{ height, width: '100%' }" autoresize />
</template>
