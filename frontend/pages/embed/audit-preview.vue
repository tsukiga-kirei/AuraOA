<script setup lang="ts">
import { CloseOutlined, ThunderboltOutlined, DesktopOutlined, MobileOutlined, ExpandOutlined } from '@ant-design/icons-vue'
import { embedAuditPreviewKey } from '~/utils/embedAuditPreview'
import { createEmbedAuditPreview, embedAuditPreviewScenarios, type EmbedAuditPreviewScenario } from '~/utils/embedAuditFixtures'

definePageMeta({ layout: false })

const route = useRoute()
const { t } = useI18n()
const canvas = route.query.canvas === '1'
const initialScenario = String(route.query.scenario || 'return')
const scenario = ref<EmbedAuditPreviewScenario>(embedAuditPreviewScenarios.includes(initialScenario as EmbedAuditPreviewScenario) ? initialScenario as EmbedAuditPreviewScenario : 'return')
const width = ref(900)
const closed = ref(false)
const frameUrl = computed(() => `/embed/audit-preview?canvas=1&scenario=${scenario.value}`)
const options = computed(() => embedAuditPreviewScenarios.map(value => ({ value, label: t(`embed.preview.scenario.${value}`) })))

if (canvas) {
  const fixture = createEmbedAuditPreview(scenario.value, {
    failure: t('embed.preview.mockFailure'), parseError: t('embed.preview.mockParseError'), unsupported: t('embed.preview.mockUnsupported'),
    stages: [t('embed.auditProgress.assembling'), t('embed.auditProgress.reasoning'), t('embed.auditProgress.extracting')],
  })
  provide(embedAuditPreviewKey, fixture.preview)
  onBeforeUnmount(fixture.dispose)
}

watch(scenario, () => { closed.value = false })
useHead({ title: t('embed.preview.title') })
</script>

<template>
  <div v-if="canvas" class="preview-canvas"><EmbedAuditPage /></div>
  <div v-else class="audit-preview-workspace">
    <header class="preview-toolbar">
      <div class="preview-brand"><span class="preview-brand-icon"><ThunderboltOutlined /></span><div><h1>{{ t('embed.preview.title') }}</h1><p>{{ t('embed.preview.subtitle') }}</p></div></div>
      <span class="preview-data-tag">{{ t('embed.preview.mockData') }}</span>
    </header>
    <div class="preview-controls">
      <div class="preview-size-controls" :aria-label="t('embed.preview.viewport')">
        <button type="button" :class="{ active: width === 900 }" @click="width = 900"><DesktopOutlined />{{ t('embed.preview.desktop') }}</button>
        <button type="button" :class="{ active: width === 560 }" @click="width = 560"><ExpandOutlined />{{ t('embed.preview.sidebar') }}</button>
        <button type="button" :class="{ active: width === 390 }" @click="width = 390"><MobileOutlined />{{ t('embed.preview.mobile') }}</button>
      </div>
      <label class="preview-width"><input v-model.number="width" type="range" min="320" max="1200" step="10" :aria-label="t('embed.preview.viewport')" /><span>{{ width }} px</span></label>
      <label class="preview-scenario"><span>{{ t('embed.preview.scenarioLabel') }}</span><select v-model="scenario"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select></label>
    </div>
    <main class="preview-stage">
      <div class="preview-dialog" :style="{ width: `${width}px` }">
        <div class="preview-dialog-header"><strong>{{ t('embed.preview.dialogTitle') }}</strong><span>{{ t('embed.preview.advisory') }}</span><button type="button" :aria-label="t('embed.preview.close')" @click="closed = true"><CloseOutlined /></button></div>
        <div v-if="closed" class="preview-closed"><ThunderboltOutlined /><p>{{ t('embed.preview.closed') }}</p><button type="button" @click="closed = false">{{ t('embed.preview.reopen') }}</button></div>
        <iframe v-else :key="scenario" :src="frameUrl" :title="t('embed.preview.frameTitle')" />
      </div>
    </main>
    <footer class="preview-footer"><span>{{ t('embed.preview.interactions') }}</span><a :href="frameUrl" target="_blank" rel="noopener">{{ t('embed.preview.openStandalone') }} ↗</a></footer>
  </div>
</template>

<style scoped>
.preview-canvas { min-height: 100vh; background: var(--color-bg-page); }
.audit-preview-workspace { height: 100dvh; min-height: 0; overflow: hidden; display: flex; flex-direction: column; color: var(--color-text-primary); background: #edf0f5; }
.preview-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 14px; padding: 17px 26px; background: var(--color-bg-card); border-bottom: 1px solid var(--color-border-light); }
.preview-brand { display: flex; align-items: center; gap: 11px; }
.preview-brand-icon { display: grid; place-items: center; width: 36px; height: 36px; border: 1px solid var(--color-border-light); border-radius: 11px; color: var(--color-primary); background: var(--color-primary-bg); font-size: 20px; }
.preview-brand h1 { margin: 0; font-size: 15px; font-family: var(--font-sans); }
.preview-brand p { margin: 2px 0 0; font-size: 11px; color: var(--color-text-tertiary); }
.preview-data-tag { flex-shrink: 0; padding: 4px 9px; color: var(--color-primary); background: var(--color-primary-bg); border-radius: 6px; font-size: 11px; }
.preview-controls { display: flex; flex-wrap: wrap; gap: 12px 20px; align-items: center; padding: 12px 26px; background: var(--color-bg-card); }
.preview-size-controls { display: flex; gap: 3px; padding: 3px; border-radius: 8px; background: var(--color-bg-hover); }
.preview-size-controls button { display: inline-flex; align-items: center; gap: 6px; border: 0; padding: 5px 11px; background: transparent; color: var(--color-text-secondary); border-radius: 6px; font: inherit; font-size: 12px; cursor: pointer; }
.preview-size-controls button.active { background: var(--color-bg-card); color: var(--color-primary); box-shadow: 0 1px 3px rgb(15 23 42 / 8%); }
.preview-width { display: flex; align-items: center; gap: 10px; }
.preview-width input { width: 110px; accent-color: var(--color-primary); }
.preview-width span { min-width: 55px; color: var(--color-text-tertiary); font-size: 11px; font-variant-numeric: tabular-nums; }
.preview-scenario { display: flex; align-items: center; gap: 8px; margin-left: auto; font-size: 12px; color: var(--color-text-secondary); }
.preview-scenario select { max-width: 100%; padding: 6px 28px 6px 9px; border: 1px solid var(--color-border-light); border-radius: 7px; background: var(--color-bg-card); color: var(--color-text-primary); font: inherit; }
.preview-stage { min-height: 0; flex: 1; display: flex; justify-content: center; padding: 20px 20px 12px; }
.preview-dialog { max-width: 100%; min-width: 0; height: 100%; display: flex; flex-direction: column; overflow: hidden; border: 1px solid #dfe4ec; border-radius: 16px; box-shadow: 0 10px 36px rgb(31 41 55 / 8%), 0 1px 3px rgb(31 41 55 / 3%); background: var(--color-bg-page); transition: width 240ms cubic-bezier(.2,.8,.2,1); }
.preview-dialog-header { display: flex; align-items: center; gap: 10px; flex-shrink: 0; padding: 12px 16px; border-bottom: 1px solid var(--color-border-light); background: var(--color-bg-card); }
.preview-dialog-header strong { font-size: 14px; }
.preview-dialog-header > span { font-size: 10px; padding: 2px 7px; color: var(--color-text-tertiary); background: var(--color-bg-hover); border-radius: 20px; }
.preview-dialog-header button { display: grid; place-items: center; margin-left: auto; width: 30px; height: 30px; flex-shrink: 0; border: 0; border-radius: 50%; background: var(--color-bg-hover); color: var(--color-text-secondary); cursor: pointer; }
.preview-dialog iframe { display: block; width: 100%; flex: 1; min-height: 0; border: 0; background: var(--color-bg-page); }
.preview-closed { display: flex; flex: 1; flex-direction: column; align-items: center; justify-content: center; gap: 14px; color: var(--color-text-tertiary); }
.preview-closed > .anticon { color: var(--color-primary); font-size: 32px; }
.preview-closed button { border: 0; border-radius: 7px; padding: 7px 16px; background: var(--color-primary-bg); color: var(--color-primary); cursor: pointer; }
.preview-footer { display: flex; justify-content: space-between; flex-wrap: wrap; gap: 6px 20px; padding: 0 26px 12px; font-size: 11px; color: var(--color-text-tertiary); }
.preview-footer a { color: var(--color-primary); }
button:focus-visible, select:focus-visible, input:focus-visible { outline: 2px solid var(--color-primary); outline-offset: 3px; }
@media (max-width: 680px) {
  .preview-toolbar { padding: 12px; }
  .preview-brand p { max-width: 240px; }
  .preview-data-tag { display: none; }
  .preview-controls { padding: 10px 12px; gap: 8px 12px; }
  .preview-width { margin-left: auto; }
  .preview-width input { width: 70px; }
  .preview-scenario { width: 100%; margin-left: 0; }
  .preview-scenario select { flex: 1; }
  .preview-stage { padding: 10px 8px; }
  .preview-footer { padding: 0 12px 10px; }
  .preview-footer > span { display: none; }
  .preview-dialog-header { gap: 7px; padding: 10px 12px; }
}
@media (max-width: 370px) { .preview-width input { display: none; } .preview-size-controls button { padding: 5px 8px; } }
@media (prefers-reduced-motion: reduce) { .preview-dialog { transition: none; } }
</style>
