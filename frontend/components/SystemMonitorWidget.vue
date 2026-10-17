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
    <!-- 顶部状态指示栏：运行状态药丸 + 连续运行时长 + 刷新 -->
    <div class="monitor-header">
      <div class="monitor-status-pill" v-if="monitorData">
        <span class="status-pulse-dot" />
        <span class="status-pill-text">{{ t('overview.monitor.statusNormal') }}</span>
        <span class="status-pill-divider">·</span>
        <span class="monitor-uptime-label">{{ t('overview.monitor.uptimeLabel') }}:</span>
        <span class="monitor-uptime-value">{{ formatUptime(monitorData.uptime_seconds) }}</span>
      </div>
      <div v-else class="monitor-status-pill" />
      <a-button
        size="small"
        :loading="loading"
        class="refresh-btn"
        @click="loadMonitorData"
      >
        <template #icon><ReloadOutlined /></template>
        {{ t('overview.monitor.refresh') }}
      </a-button>
    </div>

    <!-- 加载状态 -->
    <a-spin v-if="loading && !monitorData" style="display: flex; justify-content: center; padding: 48px 0;" />

    <!-- 错误状态 -->
    <div v-else-if="error" class="monitor-error">
      <p class="monitor-error-text">{{ t('overview.monitor.loadFailed') }}</p>
      <a-button type="primary" size="small" @click="loadMonitorData">
        {{ t('overview.monitor.retry') }}
      </a-button>
    </div>

    <!-- 数据展示 -->
    <template v-else-if="monitorData">
      <!-- 核心硬件资源三联卡片：CPU / 内存 / 磁盘 平权统一展示 -->
      <div class="monitor-metrics-grid">
        <!-- CPU 使用率卡片 -->
        <div class="metric-card">
          <div class="metric-card-header">
            <span class="metric-card-title">{{ t('overview.monitor.cpuUsage') }}</span>
            <span
              class="metric-badge"
              :style="{
                color: getMetricColor(monitorData.cpu_usage, MONITOR_THRESHOLDS.cpu),
                background: `color-mix(in srgb, ${getMetricColor(monitorData.cpu_usage, MONITOR_THRESHOLDS.cpu)} 12%, transparent)`,
              }"
            >
              {{ monitorData.cpu_usage > MONITOR_THRESHOLDS.cpu.warning ? 'High' : 'Normal' }}
            </span>
          </div>
          <GaugeChart
            :value="monitorData.cpu_usage"
            :label="t('overview.monitor.cpuUsage')"
            :thresholds="MONITOR_THRESHOLDS.cpu"
            height="130px"
          />
        </div>

        <!-- 内存使用率卡片 -->
        <div class="metric-card">
          <div class="metric-card-header">
            <span class="metric-card-title">{{ t('overview.monitor.memoryUsage') }}</span>
            <span
              class="metric-badge"
              :style="{
                color: getMetricColor(monitorData.memory_usage, MONITOR_THRESHOLDS.memory),
                background: `color-mix(in srgb, ${getMetricColor(monitorData.memory_usage, MONITOR_THRESHOLDS.memory)} 12%, transparent)`,
              }"
            >
              {{ monitorData.memory_usage > MONITOR_THRESHOLDS.memory.warning ? 'High' : 'Normal' }}
            </span>
          </div>
          <GaugeChart
            :value="monitorData.memory_usage"
            :label="t('overview.monitor.memoryUsage')"
            :thresholds="MONITOR_THRESHOLDS.memory"
            height="130px"
          />
        </div>

        <!-- 磁盘使用率卡片 -->
        <div class="metric-card">
          <div class="metric-card-header">
            <span class="metric-card-title">{{ t('overview.monitor.diskUsage') }}</span>
            <span
              class="metric-badge"
              :style="{
                color: getMetricColor(monitorData.disk_usage, MONITOR_THRESHOLDS.disk),
                background: `color-mix(in srgb, ${getMetricColor(monitorData.disk_usage, MONITOR_THRESHOLDS.disk)} 12%, transparent)`,
              }"
            >
              {{ monitorData.disk_usage > MONITOR_THRESHOLDS.disk.warning ? 'High' : 'Normal' }}
            </span>
          </div>
          <GaugeChart
            :value="monitorData.disk_usage"
            :label="t('overview.monitor.diskUsage')"
            :thresholds="MONITOR_THRESHOLDS.disk"
            height="130px"
          />
        </div>
      </div>

      <!-- 关键服务状态横向网格卡片 -->
      <div class="monitor-services">
        <div class="monitor-section-title">{{ t('overview.monitor.serviceStatus') }}</div>
        <div class="monitor-services-grid">
          <div
            v-for="svc in monitorData.services"
            :key="svc.name"
            class="service-pill"
          >
            <span
              class="service-status-dot"
              :style="{ background: getServiceStatusColor(svc.status) }"
            />
            <span class="service-name" :title="svc.name">{{ svc.name }}</span>
            <span class="service-rt">{{ svc.response_time_ms }}ms</span>
          </div>
        </div>
      </div>

      <!-- 系统告警流 -->
      <div class="monitor-alerts">
        <div class="monitor-section-title">{{ t('overview.monitor.recentAlerts') }}</div>
        <div v-if="sortedAlerts.length === 0" class="monitor-alerts-clean">
          <span class="clean-dot">✓</span>
          <span>{{ t('overview.monitor.noAlerts') }}</span>
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
.monitor-widget {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* 顶部状态指示栏 */
.monitor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.monitor-status-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  background: var(--color-bg-page);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-full);
  font-size: 12px;
  color: var(--color-text-secondary);
}

.status-pulse-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--color-success);
  box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.4);
  animation: pulse 2s infinite;
}

@keyframes pulse {
  0% {
    transform: scale(0.95);
    box-shadow: 0 0 0 0 rgba(16, 185, 129, 0.7);
  }
  70% {
    transform: scale(1);
    box-shadow: 0 0 0 5px rgba(16, 185, 129, 0);
  }
  100% {
    transform: scale(0.95);
    box-shadow: 0 0 0 0 rgba(16, 185, 129, 0);
  }
}

.status-pill-text {
  font-weight: 500;
  color: var(--color-success);
}

.status-pill-divider {
  color: var(--color-text-tertiary);
}

.monitor-uptime-label {
  color: var(--color-text-tertiary);
}

.monitor-uptime-value {
  font-weight: 500;
  color: var(--color-text-primary);
  font-family: var(--font-mono, monospace);
}

/* 核心硬件 3 列网格卡片 */
.monitor-metrics-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}

.metric-card {
  background: var(--color-bg-page);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  padding: 12px 10px 6px;
  display: flex;
  flex-direction: column;
  transition: all var(--transition-fast);
}

.metric-card:hover {
  border-color: var(--color-border);
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.04);
}

.metric-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 4px;
}

.metric-card-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-secondary);
}

.metric-badge {
  font-size: 10px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 4px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

/* 服务状态横向铺展微卡片 */
.monitor-section-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-tertiary);
  margin-bottom: 8px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.monitor-services-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
  gap: 8px;
}

.service-pill {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  background: var(--color-bg-page);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-sm);
  font-size: 12px;
  transition: all var(--transition-fast);
}

.service-pill:hover {
  background: var(--color-bg-card);
}

.service-status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.service-name {
  flex: 1;
  min-width: 0;
  color: var(--color-text-primary);
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-rt {
  font-size: 11px;
  color: var(--color-text-tertiary);
  font-family: var(--font-mono, monospace);
  flex-shrink: 0;
}

/* 告警状态 */
.monitor-alerts-clean {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: color-mix(in srgb, var(--color-success) 8%, transparent);
  border: 1px solid color-mix(in srgb, var(--color-success) 20%, transparent);
  border-radius: var(--radius-sm);
  font-size: 12px;
  color: var(--color-success);
}

.clean-dot {
  font-weight: 700;
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
  border-radius: var(--radius-sm);
  font-size: 12px;
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
  font-size: 13px;
}

.monitor-alert-msg {
  flex: 1;
  line-height: 1.4;
}

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
</style>

