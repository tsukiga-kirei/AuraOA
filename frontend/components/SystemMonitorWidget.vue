<script setup lang="ts">
/**
 * SystemMonitorWidget — 系统运行监控小部件。
 *
 * 仅系统管理员可见，展示 CPU / 内存 / 磁盘使用率、关键服务状态及系统告警。
 * 数据通过 fetchSystemMonitorData() 从后端获取。
 */
import { ReloadOutlined, WarningOutlined, CloseCircleOutlined } from '@ant-design/icons-vue'
import type { SystemMonitorData, SystemAlert } from '~/types/dashboard-overview'
import { MONITOR_THRESHOLDS, getMetricColor, getServiceStatusColor } from '~/composables/useThemeColors'
import GaugeChart from '~/components/charts/GaugeChart.vue'

const { t } = useI18n()
const { fetchSystemMonitorData } = useDashboardOverviewApi()

// 组件内部状态
const monitorData = ref<SystemMonitorData | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)

/**
 * 将秒数格式化为人类可读的运行时间（天、小时、分钟）。
 */
function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const parts: string[] = []
  if (days > 0) parts.push(`${days} ${t('overview.monitor.days')}`)
  if (hours > 0) parts.push(`${hours} ${t('overview.monitor.hours')}`)
  if (minutes > 0 || parts.length === 0) parts.push(`${minutes} ${t('overview.monitor.minutes')}`)
  return parts.join(' ')
}

/**
 * 获取系统监控数据，更新组件状态。
 * 请求失败时记录错误日志并设置错误提示。
 */
async function loadMonitorData() {
  loading.value = true
  error.value = null
  try {
    monitorData.value = await fetchSystemMonitorData()
  }
  catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e)
    console.error('[SystemMonitorWidget] 获取系统监控数据失败:', msg)
    error.value = msg || t('overview.monitor.loadFailed')
    monitorData.value = null
  }
  finally {
    loading.value = false
  }
}

/**
 * 将服务状态映射为国际化标签。
 */
function serviceStatusLabel(status: 'online' | 'offline' | 'degraded'): string {
  return t(`overview.monitor.${status}`)
}

/**
 * 将告警 message（i18n key）翻译为本地化文本，并替换 value 占位符。
 */
function formatAlertMessage(alert: SystemAlert): string {
  const key = `overview.monitor.${alert.message}`
  return t(key, alert.value)
}

/**
 * 告警列表（按级别排序：critical 优先）。
 */
const sortedAlerts = computed(() => {
  if (!monitorData.value?.alerts) return []
  return [...monitorData.value.alerts].sort((a, b) => {
    if (a.level === 'critical' && b.level !== 'critical') return -1
    if (a.level !== 'critical' && b.level === 'critical') return 1
    return 0
  })
})

// 组件挂载时自动加载数据
onMounted(() => {
  void loadMonitorData()
})
</script>

<template>
  <div class="monitor-widget">
    <!-- 顶部：刷新按钮 + 系统运行时间 -->
    <div class="monitor-header">
      <div class="monitor-uptime" v-if="monitorData">
        <span class="monitor-uptime-label">{{ t('overview.monitor.uptimeLabel') }}:</span>
        <span class="monitor-uptime-value">{{ formatUptime(monitorData.uptime_seconds) }}</span>
      </div>
      <div v-else class="monitor-uptime" />
      <a-button
        size="small"
        :loading="loading"
        @click="loadMonitorData"
      >
        <template #icon><ReloadOutlined /></template>
        {{ t('overview.monitor.refresh') }}
      </a-button>
    </div>

    <!-- 加载状态 -->
    <a-spin v-if="loading && !monitorData" style="display: flex; justify-content: center; padding: 40px 0;" />

    <!-- 错误状态 -->
    <div v-else-if="error" class="monitor-error">
      <p class="monitor-error-text">{{ t('overview.monitor.loadFailed') }}</p>
      <a-button type="primary" size="small" @click="loadMonitorData">
        {{ t('overview.monitor.retry') }}
      </a-button>
    </div>

    <!-- 数据展示 -->
    <template v-else-if="monitorData">
      <!-- 中部：CPU / 内存 Gauge 并排 -->
      <div class="monitor-gauges">
        <div class="monitor-gauge-item">
          <GaugeChart
            :value="monitorData.cpu_usage"
            :label="t('overview.monitor.cpuUsage')"
            :thresholds="MONITOR_THRESHOLDS.cpu"
            height="160px"
          />
        </div>
        <div class="monitor-gauge-item">
          <GaugeChart
            :value="monitorData.memory_usage"
            :label="t('overview.monitor.memoryUsage')"
            :thresholds="MONITOR_THRESHOLDS.memory"
            height="160px"
          />
        </div>
      </div>

      <!-- 磁盘使用率进度条 -->
      <div class="monitor-disk">
        <div class="monitor-disk-header">
          <span class="monitor-disk-label">{{ t('overview.monitor.diskUsage') }}</span>
          <span
            class="monitor-disk-value"
            :style="{ color: getMetricColor(monitorData.disk_usage, MONITOR_THRESHOLDS.disk) }"
          >
            {{ monitorData.disk_usage }}%
          </span>
        </div>
        <a-progress
          :percent="monitorData.disk_usage"
          :show-info="false"
          :stroke-color="getMetricColor(monitorData.disk_usage, MONITOR_THRESHOLDS.disk)"
          :trail-color="'var(--color-bg-page)'"
          size="small"
        />
      </div>

      <!-- 底部：关键服务状态列表 -->
      <div class="monitor-services">
        <div class="monitor-services-title">{{ t('overview.monitor.serviceStatus') }}</div>
        <div class="monitor-service-list">
          <div
            v-for="svc in monitorData.services"
            :key="svc.name"
            class="monitor-service-item"
          >
            <span
              class="monitor-service-dot"
              :style="{ background: getServiceStatusColor(svc.status) }"
            />
            <span class="monitor-service-name">{{ svc.name }}</span>
            <span class="monitor-service-status" :style="{ color: getServiceStatusColor(svc.status) }">
              {{ serviceStatusLabel(svc.status) }}
            </span>
            <span class="monitor-service-rt">
              {{ svc.response_time_ms }}ms
            </span>
          </div>
        </div>
      </div>

      <!-- 系统告警列表 -->
      <div class="monitor-alerts">
        <div class="monitor-alerts-title">{{ t('overview.monitor.recentAlerts') }}</div>
        <div v-if="sortedAlerts.length === 0" class="monitor-alerts-empty">
          {{ t('overview.monitor.noAlerts') }}
        </div>
        <div v-else class="monitor-alert-list">
          <div
            v-for="(alert, idx) in sortedAlerts"
            :key="idx"
            class="monitor-alert-item"
            :class="`monitor-alert-item--${alert.level}`"
          >
            <CloseCircleOutlined v-if="alert.level === 'critical'" class="monitor-alert-icon monitor-alert-icon--critical" />
            <WarningOutlined v-else class="monitor-alert-icon monitor-alert-icon--warning" />
            <span class="monitor-alert-msg">{{ formatAlertMessage(alert) }}</span>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
/* 系统监控小部件 — 使用 CSS 自定义属性定义样式 */

.monitor-widget {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* 顶部：刷新按钮 + 运行时间 */
.monitor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.monitor-uptime {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--color-text-secondary);
}

.monitor-uptime-label {
  color: var(--color-text-tertiary);
}

.monitor-uptime-value {
  font-weight: 500;
  color: var(--color-text-primary);
}

/* 错误状态 */
.monitor-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 32px 0;
}

.monitor-error-text {
  font-size: 13px;
  color: var(--color-danger);
  margin: 0;
}

/* Gauge 图表并排 */
.monitor-gauges {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.monitor-gauge-item {
  min-width: 0;
  overflow: hidden;
}

/* 磁盘使用率 */
.monitor-disk {
  padding: 12px 16px;
  background: var(--color-bg-page);
  border-radius: var(--radius-md);
}

.monitor-disk-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.monitor-disk-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-secondary);
}

.monitor-disk-value {
  font-size: 14px;
  font-weight: 600;
}

/* 服务状态列表 */
.monitor-services {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.monitor-services-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-secondary);
}

.monitor-service-list {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.monitor-service-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 0;
  border-bottom: 1px solid var(--color-border-light);
  font-size: 13px;
}

.monitor-service-item:last-child {
  border-bottom: none;
}

.monitor-service-dot {
  width: 8px;
  height: 8px;
  border-radius: var(--radius-full);
  flex-shrink: 0;
}

.monitor-service-name {
  flex: 1;
  color: var(--color-text-primary);
  font-weight: 500;
}

.monitor-service-status {
  font-size: 12px;
  font-weight: 500;
}

.monitor-service-rt {
  font-size: 12px;
  color: var(--color-text-tertiary);
  font-family: var(--font-mono, monospace);
  min-width: 50px;
  text-align: right;
}

/* 响应式：小屏幕下 Gauge 图表改为单列 */
@media (max-width: 480px) {
  .monitor-gauges {
    grid-template-columns: 1fr;
  }
}

/* 系统告警列表 */
.monitor-alerts {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.monitor-alerts-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-secondary);
}

.monitor-alerts-empty {
  font-size: 13px;
  color: var(--color-text-tertiary);
  padding: 8px 0;
}

.monitor-alert-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.monitor-alert-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: var(--radius-md);
  font-size: 13px;
}

.monitor-alert-item--critical {
  background: color-mix(in srgb, var(--color-danger) 8%, transparent);
  border: 1px solid color-mix(in srgb, var(--color-danger) 20%, transparent);
  color: var(--color-danger);
}

.monitor-alert-item--warning {
  background: color-mix(in srgb, var(--color-warning) 8%, transparent);
  border: 1px solid color-mix(in srgb, var(--color-warning) 20%, transparent);
  color: var(--color-warning);
}

.monitor-alert-icon {
  flex-shrink: 0;
  font-size: 14px;
}

.monitor-alert-icon--critical {
  color: var(--color-danger);
}

.monitor-alert-icon--warning {
  color: var(--color-warning);
}

.monitor-alert-msg {
  flex: 1;
  line-height: 1.4;
}
</style>
