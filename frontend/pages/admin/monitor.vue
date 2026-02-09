<script setup lang="ts">
import { CheckCircleFilled, CloseCircleFilled, ClockCircleOutlined, ApiOutlined } from '@ant-design/icons-vue'

definePageMeta({ middleware: 'auth' })

// Placeholder metrics
const metrics = ref({
  system_health: 'healthy',
  api_success_rate: 99.2,
  avg_model_response_ms: 1250,
  active_tenants: 3,
  total_audits_today: 42,
})
</script>

<template>
  <div>
    <a-page-header title="全局监控" sub-title="系统健康度与关键指标" />

    <a-row :gutter="16" style="margin-bottom: 24px;">
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="系统状态">
            <template #formatter>
              <CheckCircleFilled style="color: #52c41a; margin-right: 8px;" />
              <span style="color: #52c41a;">健康</span>
            </template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="API 成功率" :value="metrics.api_success_rate" suffix="%" :precision="1" />
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="模型平均响应" :value="metrics.avg_model_response_ms" suffix="ms" />
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="今日审核数" :value="metrics.total_audits_today" />
        </a-card>
      </a-col>
    </a-row>

    <a-card title="告警记录" size="small">
      <a-empty description="暂无告警" />
    </a-card>
  </div>
</template>
