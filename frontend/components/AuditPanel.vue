<script setup lang="ts">
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  EditOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons-vue'
import { useI18n } from '~/composables/useI18n'

interface ChecklistResult {
  rule_id: string
  rule_name?: string
  passed: boolean
  reasoning: string
  is_locked?: boolean
}

interface AuditResult {
  trace_id: string
  process_id: string
  recommendation: 'approve' | 'reject' | 'revise'
  score?: number
  details: ChecklistResult[]
  ai_reasoning: string
  duration_ms?: number
}

// props：审核结果数据 / 加载状态
defineProps<{
  result: AuditResult | null
  loading: boolean
}>()

const { t } = useI18n()

// 根据审核建议类型返回对应的颜色、背景色、图标和标签配置
const recommendationConfig = computed(() => ({
  approve: { color: 'var(--color-success)', bg: 'var(--color-success-bg)', icon: CheckCircleOutlined, label: t('dashboard.rec.approve') },
  reject: { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)', icon: CloseCircleOutlined, label: t('dashboard.rec.reject') },
  revise: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', icon: EditOutlined, label: t('dashboard.rec.revise') },
}))
</script>

<template>
  <div class="audit-panel">
    <!--加载中状态-->
    <div v-if="loading" class="panel-loading">
      <div class="loading-pulse" />
      <p class="loading-text">{{ t('auditPanel.auditing') }}</p>
    </div>

    <!--审核结果展示-->
    <template v-else-if="result">
      <div
        class="result-banner"
        :style="{
          background: recommendationConfig[result.recommendation]?.bg,
          borderColor: recommendationConfig[result.recommendation]?.color,
        }"
      >
        <component
          :is="recommendationConfig[result.recommendation]?.icon"
          :style="{ color: recommendationConfig[result.recommendation]?.color, fontSize: '24px' }"
        />
        <div class="result-banner-info">
          <div
            class="result-banner-title"
            :style="{ color: recommendationConfig[result.recommendation]?.color }"
          >
            {{ recommendationConfig[result.recommendation]?.label }}
          </div>
          <div class="result-banner-meta">Trace: {{ result.trace_id }}</div>
        </div>
        <div
          v-if="result.score"
          class="result-score"
          :style="{ color: recommendationConfig[result.recommendation]?.color }"
        >
          {{ result.score }}
        </div>
      </div>

      <div class="section">
        <h4 class="section-title">{{ t('auditPanel.ruleResults') }}</h4>
        <RuleList :rules="result.details" />
      </div>

      <div class="section">
        <h4 class="section-title">{{ t('auditPanel.aiReasoning') }}</h4>
        <div class="reasoning-block">
          <pre>{{ result.ai_reasoning }}</pre>
        </div>
      </div>
    </template>

    <!--空状态：尚未发起审核-->
    <div v-else class="panel-empty">
      <div class="empty-icon">
        <ThunderboltOutlined />
      </div>
      <p>{{ t('auditPanel.startAudit') }}</p>
    </div>
  </div>
</template>

<style scoped>
.panel-loading {
  text-align: center;
  padding: 48px 0;
}

.loading-pulse {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: var(--color-primary);
  margin: 0 auto 12px;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { transform: scale(1); opacity: 0.6; }
  50% { transform: scale(1.15); opacity: 1; }
}

.loading-text {
  color: var(--color-text-tertiary);
  font-size: 14px;
}

.result-banner {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 18px;
  border-radius: var(--radius-lg);
  border-left: 4px solid;
  margin-bottom: 20px;
}

.result-banner-title {
  font-size: 15px;
  font-weight: 700;
}

.result-banner-meta {
  font-size: 11px;
  color: var(--color-text-tertiary);
  font-family: var(--font-mono);
  margin-top: 2px;
}

.result-score {
  font-size: 32px;
  font-weight: 800;
  line-height: 1;
  margin-left: auto;
}

.section {
  margin-bottom: 20px;
}

.section-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-secondary);
  margin: 0 0 10px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.reasoning-block {
  background: var(--color-bg-page);
  border-radius: var(--radius-md);
  padding: 14px;
  border: 1px solid var(--color-border-light);
}

.reasoning-block pre {
  white-space: pre-wrap;
  word-break: break-word;
  font-family: var(--font-sans);
  font-size: 13px;
  line-height: 1.7;
  color: var(--color-text-secondary);
  margin: 0;
}

.panel-empty {
  text-align: center;
  padding: 48px 20px;
  color: var(--color-text-tertiary);
}

.empty-icon {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  background: var(--color-primary-bg);
  color: var(--color-primary);
  font-size: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 12px;
}

.panel-empty p {
  font-size: 14px;
  margin: 0;
}
</style>
