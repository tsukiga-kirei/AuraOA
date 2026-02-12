<script setup lang="ts">
import {
  CheckCircleOutlined,
  ApiOutlined,
  ClockCircleOutlined,
  ThunderboltOutlined,
  RiseOutlined,
  TeamOutlined,
  AlertOutlined,
} from '@ant-design/icons-vue'

const { mockDashboardStats } = useMockData()

const metrics = ref({
  system_health: 'healthy',
  api_success_rate: 99.2,
  avg_model_response_ms: 1250,
  active_tenants: 3,
  total_audits_today: mockDashboardStats.todayAudits,
  uptime: '99.97%',
  p95_latency: 2100,
  total_requests_24h: 1847,
})

const weeklyTrend = ref(mockDashboardStats.weeklyTrend)
const maxCount = computed(() => Math.max(...weeklyTrend.value.map(i => i.count), 1))
const hoveredBar = ref<string | null>(null)

const alerts = ref([
  { id: 1, level: 'warning', message: '租户"华东分公司" Token 用量已达 70%', time: '10 分钟前' },
  { id: 2, level: 'info', message: '系统自动完成每日数据备份', time: '2 小时前' },
  { id: 3, level: 'info', message: 'AI 模型响应时间恢复正常', time: '5 小时前' },
])

const alertLevelConfig: Record<string, { color: string; bg: string }> = {
  warning: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)' },
  error: { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)' },
  info: { color: 'var(--color-info)', bg: 'var(--color-info-bg)' },
}
</script>

<template>
  <div class="monitor-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">全局监控</h1>
        <p class="page-subtitle">系统健康度与关键运行指标</p>
      </div>
      <div class="health-badge">
        <CheckCircleOutlined />
        系统健康
      </div>
    </div>

    <!-- Metrics grid -->
    <div class="metrics-grid">
      <div
        v-for="(m, i) in [
          { icon: ApiOutlined, value: metrics.api_success_rate, unit: '%', label: 'API 成功率', variant: 'success' },
          { icon: ClockCircleOutlined, value: metrics.avg_model_response_ms, unit: 'ms', label: '模型平均响应', variant: 'primary' },
          { icon: ThunderboltOutlined, value: metrics.p95_latency, unit: 'ms', label: 'P95 延迟', variant: 'warning' },
          { icon: RiseOutlined, value: metrics.total_requests_24h, unit: '', label: '24h 请求数', variant: 'info' },
          { icon: TeamOutlined, value: metrics.active_tenants, unit: '', label: '活跃租户', variant: 'success' },
          { icon: CheckCircleOutlined, value: metrics.uptime, unit: '', label: '系统可用率', variant: 'primary' },
        ]"
        :key="i"
        class="metric-card"
      >
        <div class="metric-icon" :class="`metric-icon--${m.variant}`">
          <component :is="m.icon" />
        </div>
        <div class="metric-info">
          <div class="metric-value">{{ m.value }}<span v-if="m.unit" class="metric-unit">{{ m.unit }}</span></div>
          <div class="metric-label">{{ m.label }}</div>
        </div>
      </div>
    </div>

    <div class="monitor-grid">
      <!-- Weekly trend chart -->
      <div class="monitor-card">
        <h3 class="card-title">近 7 日审核趋势</h3>
        <div class="chart-area">
          <!-- Y-axis grid lines -->
          <div class="chart-grid">
            <div v-for="n in 4" :key="n" class="chart-grid-line" :style="{ bottom: (n * 25) + '%' }">
              <span class="chart-grid-label">{{ Math.round(maxCount * n / 4) }}</span>
            </div>
          </div>
          <!-- Bars -->
          <div class="chart-bars">
            <div
              v-for="item in weeklyTrend"
              :key="item.date"
              class="chart-bar-wrapper"
              @mouseenter="hoveredBar = item.date"
              @mouseleave="hoveredBar = null"
            >
              <Transition name="tooltip-fade">
                <div v-if="hoveredBar === item.date" class="chart-tooltip">
                  <span class="chart-tooltip-value">{{ item.count }}</span>
                  <span class="chart-tooltip-label">条审核</span>
                </div>
              </Transition>
              <div class="chart-bar-track">
                <div
                  class="chart-bar-fill"
                  :class="{ 'chart-bar-fill--active': hoveredBar === item.date }"
                  :style="{ height: (item.count / maxCount) * 100 + '%' }"
                />
              </div>
              <div class="chart-bar-label" :class="{ 'chart-bar-label--active': hoveredBar === item.date }">
                {{ item.date }}
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Alerts -->
      <div class="monitor-card">
        <h3 class="card-title">
          <AlertOutlined style="color: var(--color-warning);" />
          最近告警
        </h3>
        <div class="alerts-list">
          <div
            v-for="alert in alerts"
            :key="alert.id"
            class="alert-item"
            :style="{ borderLeftColor: alertLevelConfig[alert.level]?.color }"
          >
            <div class="alert-dot" :style="{ background: alertLevelConfig[alert.level]?.color }" />
            <div class="alert-content">
              <div class="alert-message">{{ alert.message }}</div>
              <div class="alert-time">{{ alert.time }}</div>
            </div>
          </div>
        </div>
        <div v-if="alerts.length === 0" style="padding: 32px; text-align: center;">
          <a-empty description="暂无告警" />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page-header {
  display: flex; justify-content: space-between; align-items: flex-start;
  margin-bottom: 24px;
}
.page-title { font-size: 24px; font-weight: 700; color: var(--color-text-primary); margin: 0; }
.page-subtitle { font-size: 14px; color: var(--color-text-tertiary); margin: 4px 0 0; }

.health-badge {
  display: flex; align-items: center; gap: 6px;
  padding: 8px 16px; color: var(--color-success);
  font-size: 13px; font-weight: 600;
  background: var(--color-success-bg);
  border-radius: var(--radius-full);
  border: 1px solid rgba(16, 185, 129, 0.2);
}

/* Metrics grid */
.metrics-grid {
  display: grid; grid-template-columns: repeat(3, 1fr);
  gap: 16px; margin-bottom: 24px;
}
.metric-card {
  display: flex; align-items: center; gap: 16px;
  padding: 18px 20px;
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xl);
  transition: all 0.3s ease;
  box-shadow: var(--shadow-xs);
}
.metric-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}
.metric-icon {
  width: 48px; height: 48px; border-radius: var(--radius-lg);
  display: flex; align-items: center; justify-content: center;
  font-size: 22px; flex-shrink: 0;
}
.metric-icon--primary { background: var(--color-primary-bg); color: var(--color-primary); }
.metric-icon--success { background: var(--color-success-bg); color: var(--color-success); }
.metric-icon--warning { background: var(--color-warning-bg); color: var(--color-warning); }
.metric-icon--info { background: var(--color-info-bg); color: var(--color-info); }
.metric-value { font-size: 24px; font-weight: 700; color: var(--color-text-primary); line-height: 1.2; }
.metric-unit { font-size: 14px; font-weight: 500; color: var(--color-text-tertiary); margin-left: 2px; }
.metric-label { font-size: 13px; color: var(--color-text-tertiary); margin-top: 2px; }

/* Monitor grid */
.monitor-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 24px; }

.monitor-card {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xl);
  padding: 20px;
  box-shadow: var(--shadow-xs);
}

.card-title {
  font-size: 15px; font-weight: 600; color: var(--color-text-primary);
  margin: 0 0 20px; display: flex; align-items: center; gap: 8px;
}

/* ===== Enhanced Chart ===== */
.chart-area { position: relative; height: 220px; padding-left: 36px; }

.chart-grid { position: absolute; inset: 0; left: 36px; pointer-events: none; }
.chart-grid-line {
  position: absolute; left: 0; right: 0;
  border-top: 1px dashed var(--color-border-light);
}
.chart-grid-label {
  position: absolute; left: -36px; top: -8px;
  font-size: 10px; color: var(--color-text-tertiary);
  width: 30px; text-align: right;
}

.chart-bars {
  display: flex; align-items: flex-end; gap: 8px;
  height: 100%; position: relative; z-index: 1;
}
.chart-bar-wrapper {
  flex: 1; display: flex; flex-direction: column;
  align-items: center; height: 100%;
  position: relative; cursor: pointer;
}

/* Tooltip */
.chart-tooltip {
  position: absolute; top: -8px; left: 50%; transform: translate(-50%, -100%);
  background: var(--color-bg-sidebar); color: #fff;
  padding: 6px 10px; border-radius: 8px;
  font-size: 12px; white-space: nowrap; z-index: 10;
  box-shadow: 0 4px 12px rgba(0,0,0,0.2);
  display: flex; align-items: baseline; gap: 3px;
  pointer-events: none;
}
.chart-tooltip::after {
  content: ''; position: absolute; bottom: -4px; left: 50%;
  transform: translateX(-50%) rotate(45deg); width: 8px; height: 8px;
  background: var(--color-bg-sidebar);
  border-radius: 1px;
}
.chart-tooltip-value { font-weight: 700; font-size: 14px; }
.chart-tooltip-label { font-size: 10px; opacity: 0.7; }

.tooltip-fade-enter-active { transition: all 0.2s ease; }
.tooltip-fade-leave-active { transition: all 0.15s ease; }
.tooltip-fade-enter-from, .tooltip-fade-leave-to { opacity: 0; transform: translate(-50%, -90%); }

.chart-bar-track {
  flex: 1; width: 100%; max-width: 36px;
  background: var(--color-bg-hover); border-radius: 6px;
  display: flex; align-items: flex-end; overflow: hidden;
  transition: background 0.2s ease;
}
.chart-bar-wrapper:hover .chart-bar-track {
  background: var(--color-bg-active);
}

.chart-bar-fill {
  width: 100%;
  background: linear-gradient(180deg, var(--color-primary), var(--color-primary-lighter));
  border-radius: 6px;
  transition: height 0.6s cubic-bezier(0.4, 0, 0.2, 1), box-shadow 0.2s ease;
  min-height: 4px;
}
.chart-bar-fill--active {
  box-shadow: 0 0 12px rgba(79, 70, 229, 0.4);
  background: linear-gradient(180deg, var(--color-primary-light), var(--color-primary));
}

.chart-bar-label {
  font-size: 11px; color: var(--color-text-tertiary);
  margin-top: 8px; transition: all 0.2s ease;
}
.chart-bar-label--active {
  color: var(--color-primary); font-weight: 600;
}

/* Alerts */
.alerts-list { display: flex; flex-direction: column; gap: 10px; }
.alert-item {
  display: flex; align-items: flex-start; gap: 12px;
  padding: 12px 14px; border-radius: var(--radius-md);
  background: var(--color-bg-hover); border-left: 3px solid;
  transition: background 0.2s ease;
}
.alert-item:hover { background: var(--color-bg-active); }
.alert-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; margin-top: 5px; }
.alert-message { font-size: 13px; color: var(--color-text-primary); line-height: 1.4; }
.alert-time { font-size: 11px; color: var(--color-text-tertiary); margin-top: 4px; }

@media (max-width: 1024px) {
  .metrics-grid { grid-template-columns: repeat(2, 1fr); }
  .monitor-grid { grid-template-columns: 1fr; }
}
@media (max-width: 640px) {
  .metrics-grid { grid-template-columns: 1fr; }
  .page-header { flex-direction: column; gap: 12px; }
}
</style>
