<script setup lang="ts">
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  EditOutlined,
  CloseOutlined,
} from '@ant-design/icons-vue'

import { computed } from 'vue'
import { useI18n } from '~/composables/useI18n'

const { t } = useI18n()

// props：当前选中的快照数据，为 null 时抽屉关闭
defineProps<{
  snapshot: any | null
}>()

// emit：关闭抽屉
const emit = defineEmits<{
  close: []
}>()

// 根据审核建议类型返回对应的颜色、背景色、图标和标签配置
const recommendationConfig = computed<Record<string, { color: string; bg: string; icon: any; label: string }>>(() => ({
  approve: { color: 'var(--color-success)', bg: 'var(--color-success-bg)', icon: CheckCircleOutlined, label: t('dashboard.rec.approve') },
  reject: { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)', icon: CloseCircleOutlined, label: t('dashboard.rec.reject') },
  revise: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', icon: EditOutlined, label: t('dashboard.rec.revise') },
}))
</script>

<template>
  <Teleport to="body">
    <transition name="drawer">
      <div v-if="snapshot" class="drawer-overlay" @click.self="emit('close')">
        <div class="drawer-panel">
          <div class="drawer-header">
            <h3>{{ t('snapshot.title') }}</h3>
            <button class="drawer-close" @click="emit('close')">
              <CloseOutlined />
            </button>
          </div>

          <div class="drawer-body">
            <!--概览横幅-->
            <div class="detail-section">
              <div
                class="detail-banner"
                :style="{
                  background: recommendationConfig[snapshot.recommendation]?.bg,
                  borderColor: recommendationConfig[snapshot.recommendation]?.color,
                }"
              >
                <component
                  :is="recommendationConfig[snapshot.recommendation]?.icon"
                  :style="{ color: recommendationConfig[snapshot.recommendation]?.color, fontSize: '24px' }"
                />
                <div>
                  <div
                    class="detail-banner-title"
                    :style="{ color: recommendationConfig[snapshot.recommendation]?.color }"
                  >
                    {{ recommendationConfig[snapshot.recommendation]?.label }}
                  </div>
                  <div class="detail-banner-score">
                    {{ t('snapshot.score', [snapshot.score]) }}
                  </div>
                </div>
              </div>
            </div>

            <!--基本信息网格-->
            <div class="detail-section">
              <h4 class="section-title">{{ t('snapshot.basicInfo') }}</h4>
              <div class="info-grid">
                <div class="info-item">
                  <span class="info-label">{{ t('snapshot.snapshotId') }}</span>
                  <span class="info-value mono">{{ snapshot.snapshot_id }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">{{ t('snapshot.processId') }}</span>
                  <span class="info-value mono">{{ snapshot.process_id }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">{{ t('snapshot.applicant') }}</span>
                  <span class="info-value">{{ snapshot.applicant }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">{{ t('snapshot.department') }}</span>
                  <span class="info-value">{{ snapshot.department }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">{{ t('snapshot.auditTime') }}</span>
                  <span class="info-value">{{ snapshot.created_at }}</span>
                </div>
                <div class="info-item">
                  <span class="info-label">{{ t('snapshot.userAdoption') }}</span>
                  <span class="info-value">
                    <span
                      v-if="snapshot.adopted !== null"
                      class="adopted-badge"
                      :class="snapshot.adopted ? 'adopted-badge--yes' : 'adopted-badge--no'"
                    >
                      {{ snapshot.adopted ? t('snapshot.adopted') : t('snapshot.notAdopted') }}
                    </span>
                    <span v-else class="text-muted">{{ t('snapshot.noFeedback') }}</span>
                  </span>
                </div>
              </div>
            </div>

            <!--审核理由-->
            <div class="detail-section">
              <h4 class="section-title">{{ t('snapshot.auditReason') }}</h4>
              <div class="reasoning-block">
                {{ t('snapshot.auditDisclaimer') }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </transition>
  </Teleport>
</template>

<style scoped>
.drawer-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  backdrop-filter: blur(4px);
  z-index: 1000;
  display: flex;
  justify-content: flex-end;
}

.drawer-panel {
  width: 520px;
  max-width: 100vw;
  background: var(--color-bg-card);
  height: 100%;
  display: flex;
  flex-direction: column;
  box-shadow: -8px 0 30px rgba(0, 0, 0, 0.12);
}

.drawer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px;
  border-bottom: 1px solid var(--color-border-light);
  flex-shrink: 0;
}

.drawer-header h3 {
  font-size: 16px;
  font-weight: 600;
  margin: 0;
}

.drawer-close {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  border-radius: var(--radius-md);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-tertiary);
  transition: all var(--transition-fast);
}

.drawer-close:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}

.drawer-body {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
}

.detail-section {
  margin-bottom: 24px;
}

.detail-banner {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px 20px;
  border-radius: var(--radius-lg);
  border-left: 4px solid;
}

.detail-banner-title {
  font-size: 16px;
  font-weight: 700;
}

.detail-banner-score {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin-top: 2px;
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0 0 12px;
}

.info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.info-label {
  font-size: 12px;
  color: var(--color-text-tertiary);
}

.info-value {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-primary);
}

.info-value.mono {
  font-family: var(--font-mono);
  font-size: 12px;
}

.adopted-badge {
  font-size: 12px;
  font-weight: 500;
  padding: 2px 10px;
  border-radius: var(--radius-full);
}

.adopted-badge--yes {
  background: var(--color-success-bg);
  color: var(--color-success);
}

.adopted-badge--no {
  background: var(--color-bg-hover);
  color: var(--color-text-tertiary);
}

.text-muted {
  color: var(--color-text-tertiary);
  font-size: 13px;
}

.reasoning-block {
  padding: 16px;
  background: var(--color-bg-page);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border-light);
  font-size: 13px;
  line-height: 1.7;
  color: var(--color-text-secondary);
}

/*抽屉进出场过渡动画*/
.drawer-enter-active {
  transition: opacity 0.2s ease;
}

.drawer-enter-active .drawer-panel {
  transition: transform 0.3s cubic-bezier(0.16, 1, 0.3, 1);
}

.drawer-leave-active {
  transition: opacity 0.2s ease 0.1s;
}

.drawer-leave-active .drawer-panel {
  transition: transform 0.2s ease;
}

.drawer-enter-from {
  opacity: 0;
}

.drawer-enter-from .drawer-panel {
  transform: translateX(100%);
}

.drawer-leave-to {
  opacity: 0;
}

.drawer-leave-to .drawer-panel {
  transform: translateX(100%);
}

@media (max-width: 640px) {
  .drawer-panel {
    width: 100vw;
  }

  .info-grid {
    grid-template-columns: 1fr;
  }
}
</style>
