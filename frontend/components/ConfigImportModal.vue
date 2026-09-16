<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import {
  AlertOutlined,
  AuditOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ExclamationCircleOutlined,
  FileTextOutlined,
  InboxOutlined,
  InfoCircleOutlined,
  RobotOutlined,
  UploadOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { useI18n } from '~/composables/useI18n'
import type {
  ConfigExportContents,
  ConfigExportModule,
  ImportValidationReport,
  RuleConflictStrategy,
} from '~/types/config-export-import'
import {
  applyRuleConflictStrategy,
  validateImportBundle,
} from '~/utils/configExportImportHelper'

const props = withDefaults(
  defineProps<{
    open: boolean
    module: ConfigExportModule
    processType: string
    processTypeLabel?: string
    existingRules?: any[]
    existingSummaryBlocks?: any[]
    initialSection?: 'all' | 'rules' | 'ai'
  }>(),
  {
    processTypeLabel: '',
    existingRules: () => [],
    existingSummaryBlocks: () => [],
    initialSection: 'all',
  }
)

const emit = defineEmits<{
  'update:open': [val: boolean]
  close: []
  confirm: [
    payload: {
      rules?: any[]
      ai?: any
      summary_blocks?: any[]
      strategy: RuleConflictStrategy
      importRules: boolean
      importAi: boolean
      importSummaryBlocks: boolean
    },
  ]
}>()

const { t } = useI18n()

const fileName = ref('')
const rawJson = ref('')
const report = ref<ImportValidationReport | null>(null)
const conflictStrategy = ref<RuleConflictStrategy>('skip')

// 用户是否勾选导入该模块
const shouldImportRules = ref(true)
const shouldImportAi = ref(true)
const shouldImportSummaryBlocks = ref(true)

watch(
  () => props.open,
  (isOpen) => {
    if (isOpen) {
      fileName.value = ''
      rawJson.value = ''
      report.value = null
      conflictStrategy.value = 'skip'
      shouldImportRules.value = props.initialSection !== 'ai'
      shouldImportAi.value = props.initialSection !== 'rules'
      shouldImportSummaryBlocks.value = true
    }
  },
  { immediate: true }
)

const isSummary = computed(() => props.module === 'summary')

// 处理文件读取
function handleFileChange(e: Event) {
  const target = e.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return

  if (!file.name.endsWith('.json')) {
    message.error(t('admin.ruleConfig.importOnlyJson', '仅支持上传 .json 格式的配置文件'))
    target.value = ''
    return
  }

  fileName.value = file.name
  const reader = new FileReader()
  reader.onload = (event) => {
    try {
      const content = String(event.target?.result || '')
      rawJson.value = content
      runValidation(content)
    } catch (err: any) {
      message.error(t('admin.ruleConfig.readFileError', '读取文件失败: ') + err.message)
    } finally {
      target.value = ''
    }
  }
  reader.readAsText(file)
}

function handleDrop(e: DragEvent) {
  e.preventDefault()
  const file = e.dataTransfer?.files?.[0]
  if (!file) return

  if (!file.name.endsWith('.json')) {
    message.error(t('admin.ruleConfig.importOnlyJson', '仅支持上传 .json 格式的配置文件'))
    return
  }

  fileName.value = file.name
  const reader = new FileReader()
  reader.onload = (event) => {
    const content = String(event.target?.result || '')
    rawJson.value = content
    runValidation(content)
  }
  reader.readAsText(file)
}

function runValidation(content: string) {
  const result = validateImportBundle(
    content,
    props.module,
    props.existingRules,
    props.existingSummaryBlocks
  )
  report.value = result

  // 根据当前初始范围修正勾选项
  if (props.initialSection === 'rules') {
    shouldImportRules.value = result.has_rules
    shouldImportAi.value = false
  } else if (props.initialSection === 'ai') {
    shouldImportRules.value = false
    shouldImportAi.value = result.has_ai
    shouldImportSummaryBlocks.value = result.has_summary_blocks
  } else {
    shouldImportRules.value = result.has_rules
    shouldImportAi.value = result.has_ai
    shouldImportSummaryBlocks.value = result.has_summary_blocks
  }
}

const canConfirm = computed(() => {
  if (!report.value || !report.value.valid) return false
  if (isSummary.value) {
    return shouldImportSummaryBlocks.value && report.value.has_summary_blocks
  }
  const rulesOk = shouldImportRules.value && report.value.has_rules
  const aiOk = shouldImportAi.value && report.value.has_ai
  return rulesOk || aiOk
})

function handleConfirm() {
  if (!canConfirm.value || !report.value?.parsed_data) return

  const parsed = report.value.parsed_data

  emit('confirm', {
    rules: parsed.rules,
    ai: parsed.ai,
    summary_blocks: parsed.summary_blocks,
    strategy: conflictStrategy.value,
    importRules: shouldImportRules.value,
    importAi: shouldImportAi.value,
    importSummaryBlocks: shouldImportSummaryBlocks.value,
  })

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
    :title="t('admin.ruleConfig.importModalTitle', '导入流程配置 (JSON)')"
    width="640px"
    centered
    destroy-on-close
    @ok="handleConfirm"
    @cancel="handleCancel"
  >
    <template #okText>
      <CheckCircleOutlined /> {{ t('admin.ruleConfig.confirmImport', '确认导入') }}
    </template>
    <template #cancelText>
      {{ t('admin.ruleConfig.cancel', '取消') }}
    </template>
    <template #okButtonProps>
      <a-button type="primary" :disabled="!canConfirm">
        <CheckCircleOutlined /> {{ t('admin.ruleConfig.confirmImport', '确认导入') }}
      </a-button>
    </template>

    <div class="import-modal-body">
      <!-- 目标流程提示 -->
      <div class="target-process-box">
        <div class="target-process-info">
          <FileTextOutlined class="process-icon" />
          <span>{{ t('admin.ruleConfig.importTargetLabel', '导入目标') }}:</span>
          <strong>{{ processType }}</strong>
          <a-tag v-if="processTypeLabel" color="blue">{{ processTypeLabel }}</a-tag>
        </div>
        <div class="target-draft-hint">
          <InfoCircleOutlined />
          {{ t('admin.ruleConfig.importDraftNotice', '导入数据将作为未保存草稿载入页面，您可以复核后再点击保存。') }}
        </div>
      </div>

      <!-- 上传区域 -->
      <div
        class="upload-dropzone"
        @dragover.prevent
        @drop="handleDrop"
      >
        <input
          id="config-json-upload-input"
          type="file"
          accept=".json,application/json"
          style="display: none;"
          @change="handleFileChange"
        >
        <label for="config-json-upload-input" class="dropzone-label">
          <InboxOutlined class="dropzone-icon" />
          <p class="dropzone-text">
            {{ fileName ? fileName : t('admin.ruleConfig.dropJsonFileHere', '点击或将 .json 配置文件拖拽至此') }}
          </p>
          <p class="dropzone-sub">
            {{ t('admin.ruleConfig.dropJsonFileSub', '支持 AuraOA 导出的完整流程配置包或纯规则/AI JSON') }}
          </p>
        </label>
      </div>

      <!-- 校验结果与预检报告 -->
      <div v-if="report" class="validation-report-section">
        <!-- 错误拦截 -->
        <a-alert
          v-if="report.errors.length > 0"
          type="error"
          show-icon
          style="margin-bottom: 12px;"
        >
          <template #message>{{ t('admin.ruleConfig.validationErrorTitle', '校验未通过') }}</template>
          <template #description>
            <ul class="alert-list">
              <li v-for="(err, idx) in report.errors" :key="idx">{{ err }}</li>
            </ul>
          </template>
        </a-alert>

        <!-- 警告提示 -->
        <a-alert
          v-if="report.warnings.length > 0"
          type="warning"
          show-icon
          style="margin-bottom: 12px;"
        >
          <template #message>{{ t('admin.ruleConfig.validationWarningTitle', '校验提示与建议') }}</template>
          <template #description>
            <ul class="alert-list">
              <li v-for="(warn, idx) in report.warnings" :key="idx">{{ warn }}</li>
            </ul>
          </template>
        </a-alert>

        <!-- 校验通过项卡片 -->
        <div v-if="report.valid" class="report-cards">
          <!-- 规则校验卡片 -->
          <div v-if="report.has_rules && !isSummary" class="report-card">
            <div class="report-card-header">
              <div class="header-left">
                <a-checkbox v-model:checked="shouldImportRules" />
                <AuditOutlined class="card-icon" />
                <span class="card-title">{{ t('admin.ruleConfig.rulesTab', '规则配置') }}</span>
              </div>
              <div class="header-right">
                <a-tag color="green">
                  {{ t('admin.ruleConfig.validRulesCount', `有效 ${report.rule_stats?.valid} 条`) }}
                </a-tag>
              </div>
            </div>

            <div class="report-card-body">
              <div class="rule-stats-row">
                <span>
                  {{ t('admin.ruleConfig.newRulesCount', `新增: ${report.rule_stats?.new_rules} 条`) }}
                </span>
                <span :class="{ 'highlight-duplicate': (report.rule_stats?.duplicate || 0) > 0 }">
                  {{ t('admin.ruleConfig.dupRulesCount', `与当前重复: ${report.rule_stats?.duplicate} 条`) }}
                </span>
              </div>

              <!-- 冲突处理策略 -->
              <div v-if="(report.rule_stats?.duplicate || 0) > 0" class="conflict-strategy-box">
                <div class="conflict-label">
                  {{ t('admin.ruleConfig.conflictStrategyLabel', '检测到重复规则，请选择处理策略：') }}
                </div>
                <a-radio-group v-model:value="conflictStrategy" size="small">
                  <a-radio value="skip">
                    {{ t('admin.ruleConfig.strategySkip', '跳过重复项（保留现有）') }}
                  </a-radio>
                  <a-radio value="overwrite">
                    {{ t('admin.ruleConfig.strategyOverwrite', '覆盖已有规则') }}
                  </a-radio>
                  <a-radio value="append">
                    {{ t('admin.ruleConfig.strategyAppend', '全部追加为新规则') }}
                  </a-radio>
                </a-radio-group>
              </div>
            </div>
          </div>

          <!-- AI 配置校验卡片 -->
          <div v-if="report.has_ai && !isSummary" class="report-card">
            <div class="report-card-header">
              <div class="header-left">
                <a-checkbox v-model:checked="shouldImportAi" />
                <RobotOutlined class="card-icon" />
                <span class="card-title">{{ t('admin.ruleConfig.aiConfigTab', 'AI 提示词与模型配置') }}</span>
              </div>
              <div class="header-right">
                <a-tag color="blue">
                  {{ t('admin.ruleConfig.strictness', '审核尺度') }}: {{ report.ai_stats?.strictness || 'standard' }}
                </a-tag>
              </div>
            </div>
            <div class="report-card-body">
              <div class="ai-stats-info">
                <span>
                  {{ t('admin.ruleConfig.thinkingMode', '深度思考') }}:
                  <strong>{{ report.ai_stats?.enable_thinking ? t('admin.ruleConfig.enabled', '开启') : t('admin.ruleConfig.disabled', '关闭') }}</strong>
                </span>
                <span v-if="report.ai_stats?.unknown_variables?.length === 0" style="color: #52c41a;">
                  ✓ {{ t('admin.ruleConfig.variablesAllValid', '提示词系统变量全部合规') }}
                </span>
              </div>
            </div>
          </div>

          <!-- 流程总结卡片 -->
          <div v-if="report.has_summary_blocks && isSummary" class="report-card">
            <div class="report-card-header">
              <div class="header-left">
                <a-checkbox v-model:checked="shouldImportSummaryBlocks" />
                <RobotOutlined class="card-icon" />
                <span class="card-title">{{ t('admin.ruleConfig.summaryBlocksExport', 'AI 总结块配置') }}</span>
              </div>
              <div class="header-right">
                <a-tag color="green">
                  {{ t('admin.ruleConfig.summaryBlocksCount', `共 ${report.summary_stats?.total_blocks} 个总结块`) }}
                </a-tag>
              </div>
            </div>
            <div class="report-card-body">
              <div class="summary-blocks-list">
                <a-tag
                  v-for="(title, idx) in report.summary_stats?.block_titles"
                  :key="idx"
                  style="margin-bottom: 4px;"
                >
                  {{ title }}
                </a-tag>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </a-modal>
</template>

<style scoped>
.import-modal-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 4px 0;
}

.target-process-box {
  background: var(--color-bg-card, #fafafa);
  border: 1px solid var(--color-border-light, #f0f0f0);
  border-radius: 8px;
  padding: 10px 14px;
}

.target-process-info {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--color-text-title, #1f2937);
}

.process-icon {
  color: var(--color-primary, #1890ff);
}

.target-draft-hint {
  margin-top: 6px;
  font-size: 12px;
  color: var(--color-text-secondary, #6b7280);
  display: flex;
  align-items: center;
  gap: 6px;
}

.upload-dropzone {
  border: 2px dashed var(--color-border-light, #d9d9d9);
  border-radius: 8px;
  background: #fafafa;
  transition: all 0.2s ease;
  cursor: pointer;
  text-align: center;
  padding: 20px 16px;
}

.upload-dropzone:hover {
  border-color: var(--color-primary, #1890ff);
  background: #f0f7ff;
}

.dropzone-label {
  cursor: pointer;
  display: block;
}

.dropzone-icon {
  font-size: 32px;
  color: var(--color-primary, #1890ff);
  margin-bottom: 8px;
}

.dropzone-text {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-title, #1f2937);
  margin-bottom: 4px;
}

.dropzone-sub {
  font-size: 12px;
  color: var(--color-text-secondary, #8c8c8c);
  margin: 0;
}

.alert-list {
  margin: 0;
  padding-left: 18px;
  font-size: 13px;
}

.alert-list li {
  margin-bottom: 2px;
}

.report-cards {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.report-card {
  border: 1px solid var(--color-border-light, #f0f0f0);
  border-radius: 8px;
  background: #fff;
  overflow: hidden;
}

.report-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  background: #fafafa;
  border-bottom: 1px solid var(--color-border-light, #f0f0f0);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.card-icon {
  color: var(--color-primary, #1890ff);
}

.card-title {
  font-weight: 600;
  font-size: 14px;
}

.report-card-body {
  padding: 12px 14px;
}

.rule-stats-row {
  display: flex;
  gap: 16px;
  font-size: 13px;
  margin-bottom: 10px;
}

.highlight-duplicate {
  color: #faad14;
  font-weight: 500;
}

.conflict-strategy-box {
  background: #fffbe6;
  border: 1px solid #ffe58f;
  border-radius: 6px;
  padding: 8px 12px;
}

.conflict-label {
  font-size: 12px;
  color: #d46b08;
  margin-bottom: 6px;
  font-weight: 500;
}

.ai-stats-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 13px;
}

.summary-blocks-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
</style>
