<script setup lang="ts">
import { SafetyCertificateOutlined, DownOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import type { AIRecheckRecord } from '~/types/ai-recheck'

const props = defineProps<{ record: AIRecheckRecord; output?: string; retrying?: boolean }>()
const emit = defineEmits<{ retry: [] }>()
const { t } = useI18n()
const expanded = ref(false)
const panelId = useId()
const incomplete = computed(() => props.record.status === 'failed' || props.record.status === 'cancelled')
const badge = computed(() => t(props.retrying ? 'recheck.status.running' : incomplete.value ? 'recheck.status.failed' : props.record.changed ? 'recheck.panel.adjusted' : 'recheck.panel.reused'))
</script>

<template>
  <section class="result-recheck-panel" :class="{ 'result-recheck-panel--incomplete': incomplete }">
    <button type="button" class="result-recheck-panel__toggle" :aria-expanded="expanded" :aria-controls="panelId" @click="expanded = !expanded">
      <span class="result-recheck-panel__title"><SafetyCertificateOutlined aria-hidden="true" />{{ t('recheck.title') }}<span class="result-recheck-panel__badge">{{ badge }}</span></span>
      <span class="result-recheck-panel__action">{{ t(expanded ? 'resultDetails.collapse' : 'resultDetails.expand') }}<DownOutlined aria-hidden="true" :class="{ 'result-recheck-panel__arrow--open': expanded }" /></span>
    </button>
    <div v-if="expanded" :id="panelId" class="result-recheck-panel__body">
      <template v-if="incomplete">
        <p class="result-recheck-panel__explanation">{{ t('recheck.panel.fallback') }}</p>
        <p v-if="record.error" class="result-recheck-panel__error">{{ record.error }}</p>
        <a-button type="primary" :loading="retrying" @click="emit('retry')"><template #icon><ReloadOutlined aria-hidden="true" /></template>{{ t('recheck.panel.retry') }}</a-button>
      </template>
      <template v-else-if="record.changed">
        <AIRecheckResult :record="record" task-status="completed" />
      </template>
      <p v-else class="result-recheck-panel__explanation">{{ t('recheck.panel.unchanged') }}</p>
      <details v-if="output" class="result-recheck-panel__output">
        <summary>{{ t('recheck.panel.output') }}</summary>
        <AiMarkdownStream :text="output" format="json" page-flow />
      </details>
    </div>
  </section>
</template>

<style scoped>
.result-recheck-panel { margin-bottom: 10px; border: 1px solid var(--color-border-light); border-radius: 12px; background: var(--color-bg-card); }
.result-recheck-panel__toggle { display: flex; width: 100%; min-height: 46px; justify-content: space-between; align-items: center; gap: 12px; padding: 11px 14px; border: 0; border-radius: 12px; background: var(--color-bg-card); color: var(--color-text-primary); font: inherit; text-align: left; cursor: pointer; }
.result-recheck-panel__toggle:hover { background: var(--color-bg-hover); }
.result-recheck-panel__title { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; font-size: 13.5px; font-weight: 600; }
.result-recheck-panel__title > .anticon { color: var(--color-primary); font-size: 16px; }
.result-recheck-panel__badge { padding: 2px 8px; border-radius: 20px; font-size: 11px; font-weight: 500; color: var(--color-primary); background: var(--color-primary-bg); }
.result-recheck-panel--incomplete .result-recheck-panel__badge { color: var(--color-warning); background: var(--color-warning-bg); }
.result-recheck-panel__action { display: flex; flex-shrink: 0; align-items: center; gap: 6px; color: var(--color-text-tertiary); font-size: 12px; }
.result-recheck-panel__arrow--open { transform: rotate(180deg); }
.result-recheck-panel__body { padding: 14px; border-top: 1px solid var(--color-border-light); color: var(--color-text-secondary); font-size: 13px; line-height: 1.7; overflow-wrap: anywhere; }
.result-recheck-panel__explanation { margin: 0 0 10px; }
.result-recheck-panel__error { margin: 0 0 14px; padding: 9px 12px; border-radius: 8px; background: var(--color-bg-hover); color: var(--color-text-secondary); }
.result-recheck-panel__output { margin-top: 12px; }
.result-recheck-panel__output summary { color: var(--color-text-tertiary); cursor: pointer; font-size: 12px; }
.result-recheck-panel__body :deep(.recheck-result) { margin: 0; }
.result-recheck-panel__toggle:focus-visible { outline: 2px solid var(--color-primary); outline-offset: -2px; }
</style>
