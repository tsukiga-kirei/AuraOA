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

definePageMeta({ middleware: 'auth' })

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

const alerts = ref([
  { id: 1, level: 'warning', message: '租户"华东分公司" Token 用量已达 70%', time: '10 分钟前' },
  { id: 2, level: 'info', message: '系统自动完成每日数据备份', time: '2 小时前' },
  { id: 3, level: 'info', message: 'AI 模型响应时间恢复正常', time: '5 小时前' },
])

const alertLevelConfig: Record<string, { color: string; bg: string }> = {
  warning: { color: '#f59e0b', bg: '#fffbeb' },
  error: { color: '#ef4444', bg: '#fef2f2' },
  info: { color: '#3b82f6', bg: '#eff6ff' },
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
      <div class="metric-card">
        <div class="metric-icon metric-icon--success">
          <ApiOutlined />
        </div>
        <div class="metric-info">
          <div class="metric-value">{{ metrics.api_success_rate }}<span class="metric-unit">%</span></div>
          <div class="metric-label">API 成功率</div>
        </div>
      </div>

      <div class="metric-card">
        <div class="metric-icon metric-icon--primary">
          <ClockCircleOutlined />
        </div>
        <div class="metric-info">
          <div class="metric-value">{{ metrics.avg_model_response_ms }}<span class="metric-unit">ms</span></div>
          <div class="metric-label">模型平均响应</div>
        </div>
      </div>

      <div class="metric-card">
        <div class="metric-icon metric-icon--warning">
          <ThunderboltOutlined />
        </div>
        <div class="metric-info">
          <div class="metric-value">{{ metrics.p95_latency }}<span class="metric-unit">ms</span></div>
          <div class="metric-label">P95 延迟</div>
        </div>
      </div>

      <div class="metric-card">
        <div class="metric-icon metric-icon--info">
          <RiseOutlined />
        </div>
        <div class="metric-info">
          <div class="metric-value">{{ metrics.total_requests_24h }}</div>
          <div class="metric-label">24h 请求数</div>
        </div>
      </div>

      <div class="metric-card">
        <div class="metric-icon metric-icon--success">
          <TeamOutlined />
        </div>
        <div class="metric-info">
          <div class="metric-value">{{ metrics.active_tenants }}</div>
          <div class="metric-label">活跃租户</div>
        </div>
      </div>

      <div class="metric-card">
        <div class="metric-icon metric-icon--primary">
          <CheckCircleOutlined />
        </div>
        <div class="metric-info">
          <div class="metric-value">{{ metrics.uptime }}</div>
          <div class="metric-label">系统可用率</div>
        </div>
      </div>
    </div>

    <div class="monitor-grid">
      <!-- Weekly trend -->
      <div class="chart-card">
        <h3 class="card-title">近 7 日审核趋势</h3>
        <div class="mini-chart">
          <div class="chart-bars">
            <div
              v-for="item in weeklyTrend"
              :key="item.date"
              class="chart-bar-wrapper"
            >
              <div class="chart-bar-value">{{ item.count }}</div>
              <div class="chart-bar-track">
                <div
                  class="chart-bar-fill"
                  :style="{ height: (item.count / 50) * 100 + '%' }"
                />
              </div>
              <div class="chart-bar-label">{{ item.date }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- Alerts -->
      <div class="alerts-card">
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
            <div
              class="alert-dot"
              :style="{ background: alertLevelConfig[alert.level]?.color }"
            />
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
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.page-title {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin: 0;
}

.page-subtitle {
  font-size: 14px;
  color: var(--color-text-tertiary);
  margin: 4px 0 0;
}

.health-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border-radius: var(--radius-full);
  background: var(--color-success-bg);
  color: var(--color-success);
  font-size: 13px;
  font-weight: 600;
}

/* Metrics grid */
.metrics-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.metric-card {
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light);
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  transition: all var(--transition-base);
}

.metric-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.metric-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  flex-shrink: 0;
}

.metric-icon--primary {
  background: var(--color-primary-bg);
  color: var(--color-primary);
}

.metric-icon--success {
  background: var(--color-success-bg);
  color: var(--color-success);
}

.metric-icon--warning {
  background: var(--color-warning-bg);
  color: var(--color-warning);
}

.metric-icon--info {
  background: var(--color-info-bg);
  color: var(--color-info);
}

.metric-value {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-text-primary);
  line-height: 1.2;
}

.metric-unit {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-tertiary);
  margin-left: 2px;
}

.metric-label {
  font-size: 13px;
  color: var(--color-text-tertiary);
  margin-top: 2px;
}

/* Monitor grid */
.monitor-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
}

.chart-card,
.alerts-card {
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light);
  padding: 20px;
}

.card-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0 0 20px;
  display: flex;
  align-items: center;
  gap: 8px;
}

/* Mini chart */
.chart-bars {
  display: flex;
  align-items: flex-end;
  gap: 12px;
  height: 180px;
  padding-top: 20px;
}

.chart-bar-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  height: 100%;
}

.chart-bar-value {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-secondary);
  margin-bottom: 6px;
}

.chart-bar-track {
  flex: 1;
  width: 100%;
  max-width: 40px;
  background: var(--color-bg-hover);
  border-radius: var(--radius-sm);
  display: flex;
  align-items: flex-end;
  overflow: hidden;
}

.chart-bar-fill {
  width: 100%;
  background: linear-gradient(180deg, var(--color-primary), var(--color-primary-lighter));
  border-radius: var(--radius-sm);
  transition: height 0.5s ease;
  min-height: 4px;
}

.chart-bar-label {
  font-size: 11px;
  color: var(--color-text-tertiary);
  margin-top: 8px;
}

/* Alerts */
.alerts-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.alert-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 14px;
  border-radius: var(--radius-md);
  background: var(--color-bg-page);
  border-left: 3px solid;
}

.alert-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  margin-top: 5px;
}

.alert-message {
  font-size: 13px;
  color: var(--color-text-primary);
  line-height: 1.4;
}

.alert-time {
  font-size: 11px;
  color: var(--color-text-tertiary);
  margin-top: 4px;
}

@media (max-width: 1024px) {
  .metrics-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .monitor-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .metrics-grid {
    grid-template-columns: 1fr;
  }

  .page-header {
    flex-direction: column;
    gap: 12px;
  }
}
</style>
