<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  AuditOutlined,
  CloudDownloadOutlined,
  FileTextOutlined,
  InfoCircleOutlined,
  RobotOutlined,
  SafetyCertificateOutlined,
  AppstoreOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { useI18n } from '~/composables/useI18n'
import type { ConfigExportContents, ConfigExportModule, ExportedRuleItem } from '~/types/config-export-import'
import { buildExportBundle, downloadJsonFile, formatExportTimestamp } from '~/utils/configExportImportHelper'

const props = withDefaults(
  defineProps<{
    open: boolean
    module: ConfigExportModule
    processType: string
    processTypeLabel?: string
    mainTableName?: string
    rules?: any[]
    aiConfig?: any
    summaryBlocks?: any[]
    preselectedRuleIds?: string[]
    initialSection?: 'all' | 'rules' | 'ai'
  }>(),
  {
    processTypeLabel: '',
    mainTableName: '',
    rules: () => [],
    summaryBlocks: () => [],
    preselectedRuleIds: () => [],
    initialSection: 'all',
  }
)

const emit = defineEmits<{
  'update:open': [val: boolean]
  close: []
}>()

const { t } = useI18n()

// 选中的导出模块项
const exportRules = ref(true)
const exportAi = ref(true)
const exportSummaryBlocks = ref(true)
const customFileName = ref('')

// 当弹窗打开时重置状态
watch(
  () => props.open,
  (isOpen) => {
    if (isOpen) {
      if (props.initialSection === 'rules') {
        exportRules.value = true
        exportAi.value = false
        exportSummaryBlocks.value = false
      } else if (props.initialSection === 'ai') {
        exportRules.value = false
        exportAi.value = true
        exportSummaryBlocks.value = true
      } else {
        exportRules.value = true
        exportAi.value = true
        exportSummaryBlocks.value = true
      }

      const timeStamp = formatExportTimestamp()
      const cleanProcess = (props.processType || 'process').replace(/[/\\?%*:|"<>]/g, '_')
      customFileName.value = `${cleanProcess}_${props.module}_config_${timeStamp}`
    }
  },
  { immediate: true }
)

const isSummary = computed(() => props.module === 'summary')

// 规则筛选（是否有部分选中的规则）
const effectiveRules = computed(() => {
  if (!props.rules || props.rules.length === 0) return []
  if (props.preselectedRuleIds && props.preselectedRuleIds.length > 0) {
    const set = new Set(props.preselectedRuleIds)
    return props.rules.filter((r) => set.has(r.id))
  }
  return props.rules
})

const hasSelectedSubset = computed(
  () =>
    props.preselectedRuleIds &&
    props.preselectedRuleIds.length > 0 &&
    props.preselectedRuleIds.length < (props.rules?.length || 0)
)

const canExport = computed(() => {
  if (isSummary.value) {
    return exportSummaryBlocks.value && (props.summaryBlocks?.length || 0) > 0
  }
  return (exportRules.value && effectiveRules.value.length > 0) || exportAi.value
})

function handleExport() {
  if (!canExport.value) {
    message.warning(t('admin.ruleConfig.exportNoContentSelected', '请至少选择一项有效内容进行导出'))
    return
  }

  const contents: ConfigExportContents = {}

  if (!isSummary.value) {
    if (exportRules.value && effectiveRules.value.length > 0) {
      contents.rules = effectiveRules.value.map((r): ExportedRuleItem => ({
        rule_content: r.rule_content,
        rule_scope: r.rule_scope,
        enabled: r.enabled,
        source: r.source,
        related_flow: r.related_flow,
        context_enabled: r.context_enabled,
        context_mounts: r.context_mounts || [],
      }))
    }
    if (exportAi.value && props.aiConfig) {
      contents.ai = {
        audit_strictness: props.aiConfig.audit_strictness,
        enable_thinking: props.aiConfig.enable_thinking,
        system_reasoning_prompt: props.aiConfig.system_reasoning_prompt,
        system_extraction_prompt: props.aiConfig.system_extraction_prompt,
        user_reasoning_prompt: props.aiConfig.user_reasoning_prompt,
        user_extraction_prompt: props.aiConfig.user_extraction_prompt,
      }
    }
  } else {
    if (exportSummaryBlocks.value && props.summaryBlocks && props.summaryBlocks.length > 0) {
      contents.summary_blocks = props.summaryBlocks
    }
  }

  const bundle = buildExportBundle({
    module: props.module,
    processType: props.processType,
    processTypeLabel: props.processTypeLabel,
    mainTableName: props.mainTableName,
    contents,
  })

  const fileName = customFileName.value.trim() || `${props.processType}_${props.module}_config`
  downloadJsonFile(bundle, fileName)
  message.success(t('admin.ruleConfig.exportSuccess', '配置导出成功'))
  emit('update:open', false)
  emit('close')
}

function handleCancel() {
  emit('update:open', false)
  emit('close')
}
</script>

<template>
  <a-modal
    :open="open"
    :title="t('admin.ruleConfig.exportModalTitle', '导出流程配置 (JSON)')"
    width="560px"
    centered
    destroy-on-close
    @ok="handleExport"
    @cancel="handleCancel"
  >
    <template #okText>
      <CloudDownloadOutlined /> {{ t('admin.ruleConfig.confirmExport', '确认导出') }}
    </template>
    <template #cancelText>
      {{ t('admin.ruleConfig.cancel', '取消') }}
    </template>

    <div class="export-modal-body">
      <!-- 流程信息卡片 -->
      <div class="process-summary-box">
        <div class="process-summary-title">
          <FileTextOutlined class="summary-icon" />
          <span>{{ processType }}</span>
          <a-tag v-if="processTypeLabel" color="blue">{{ processTypeLabel }}</a-tag>
        </div>
        <div v-if="mainTableName" class="process-summary-sub">
          {{ t('admin.ruleConfig.mainTableLabel', '主表') }}: <code>{{ mainTableName }}</code>
        </div>
      </div>

      <!-- 导出选项清单 -->
      <div class="export-section">
        <div class="export-section-title">
          {{ t('admin.ruleConfig.selectExportContents', '选择导出内容') }}
        </div>

        <div class="export-options-list">
          <!-- 审核/归档：规则 -->
          <div
            v-if="!isSummary"
            class="export-option-row"
            :class="{ 'export-option-row--disabled': !rules || rules.length === 0 }"
          >
            <a-checkbox
              v-model:checked="exportRules"
              :disabled="!rules || rules.length === 0"
            >
              <span class="option-label">
                <AuditOutlined /> {{ t('admin.ruleConfig.rulesTab', '规则配置') }}
              </span>
            </a-checkbox>
            <span class="option-desc">
              <template v-if="hasSelectedSubset">
                {{ t('admin.ruleConfig.exportSelectedRulesCount', [preselectedRuleIds.length, rules.length]) }}
              </template>
              <template v-else>
                {{ t('admin.ruleConfig.exportAllRulesCount', [rules.length]) }}
              </template>
            </span>
          </div>

          <!-- 审核/归档：AI 提示词 -->
          <div v-if="!isSummary" class="export-option-row">
            <a-checkbox v-model:checked="exportAi">
              <span class="option-label">
                <RobotOutlined /> {{ t('admin.ruleConfig.aiConfigTab', 'AI 提示词与模型配置') }}
              </span>
            </a-checkbox>
            <span class="option-desc">
              {{ t('admin.ruleConfig.exportAiDesc', '包含推理/提取提示词、严格度与深度思考') }}
            </span>
          </div>

          <!-- 流程总结：总结块 -->
          <div v-if="isSummary" class="export-option-row">
            <a-checkbox
              v-model:checked="exportSummaryBlocks"
              :disabled="!summaryBlocks || summaryBlocks.length === 0"
            >
              <span class="option-label">
                <RobotOutlined /> {{ t('admin.ruleConfig.summaryBlocksExport', 'AI 总结块配置') }}
              </span>
            </a-checkbox>
            <span class="option-desc">
              {{ t('admin.ruleConfig.summaryBlocksCount', [summaryBlocks.length]) }}
            </span>
          </div>

          <!-- 字段配置（暂不支持） -->
          <div class="export-option-row export-option-row--disabled">
            <a-checkbox disabled>
              <span class="option-label">
                <AppstoreOutlined /> {{ t('admin.ruleConfig.tabFields', '字段配置') }}
              </span>
            </a-checkbox>
            <a-tag size="small">{{ t('admin.ruleConfig.notSupported', '暂不支持') }}</a-tag>
          </div>

          <!-- 权限配置（暂不支持） -->
          <div class="export-option-row export-option-row--disabled">
            <a-checkbox disabled>
              <span class="option-label">
                <SafetyCertificateOutlined /> {{ t('admin.ruleConfig.tabPerms', '权限配置') }}
              </span>
            </a-checkbox>
            <a-tag size="small">{{ t('admin.ruleConfig.notSupported', '暂不支持') }}</a-tag>
          </div>
        </div>
      </div>

      <!-- 文件名配置 -->
      <div class="export-filename-section">
        <div class="export-section-title">
          {{ t('admin.ruleConfig.exportFileNameLabel', '导出文件名') }}
        </div>
        <a-input
          v-model:value="customFileName"
          addon-after=".json"
          placeholder="config_export"
        />
      </div>

      <div class="export-tip">
        <InfoCircleOutlined />
        <span>{{ t('admin.ruleConfig.exportTipText', '导出的 JSON 格式符合 AuraOA 流程配置标准，可在同类流程中导入复用。') }}</span>
      </div>
    </div>
  </a-modal>
</template>

<style scoped>
.export-modal-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 4px 0;
}

.process-summary-box {
  background: var(--color-bg-card, #fafafa);
  border: 1px solid var(--color-border-light, #f0f0f0);
  border-radius: 8px;
  padding: 12px 16px;
}

.process-summary-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-title, #1f2937);
}

.summary-icon {
  color: var(--color-primary, #1890ff);
}

.process-summary-sub {
  margin-top: 4px;
  font-size: 13px;
  color: var(--color-text-secondary, #6b7280);
}

.process-summary-sub code {
  background: rgba(0, 0, 0, 0.04);
  padding: 2px 6px;
  border-radius: 4px;
  font-family: monospace;
}

.export-section-title {
  font-size: 14px;
  font-weight: 500;
  margin-bottom: 8px;
  color: var(--color-text-primary, #374151);
}

.export-options-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  background: #fff;
  border: 1px solid var(--color-border-light, #f0f0f0);
  border-radius: 8px;
  padding: 8px 12px;
}

.export-option-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 4px;
  border-bottom: 1px solid var(--color-border-light, #f8f9fa);
}

.export-option-row:last-child {
  border-bottom: none;
}

.export-option-row--disabled {
  opacity: 0.65;
}

.option-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
}

.option-desc {
  font-size: 12px;
  color: var(--color-text-secondary, #8c8c8c);
}

.export-tip {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--color-text-secondary, #8c8c8c);
  margin-top: 4px;
}
</style>
