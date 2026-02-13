<script setup lang="ts">
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  LockOutlined,
  UnlockOutlined,
  DatabaseOutlined,
  FileTextOutlined,
  ThunderboltOutlined,
  SettingOutlined,
  RobotOutlined,
  CheckOutlined,
  UploadOutlined,
  EyeOutlined,
  EyeInvisibleOutlined,
  ControlOutlined,
  ClockCircleOutlined,
  MailOutlined,
  DashboardOutlined,
  FolderOpenOutlined,
  AppstoreOutlined,
  AuditOutlined,
  SafetyCertificateOutlined,
  TeamOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type { ProcessAuditConfig, ProcessField, AuditRule, CronTaskTypeConfig, ArchiveReviewConfig, FlowRuleConfig } from '~/composables/useMockData'

definePageMeta({ middleware: 'auth', layout: 'default' })

const { mockProcessAuditConfigs, mockCronTaskTypeConfigs, mockArchiveReviewConfigs } = useMockData()

// ===== Top-level tab: 审核工作台 vs 定时任务配置 vs 归档复盘 =====
const topTab = ref<'audit' | 'cron' | 'archive'>('audit')

// ===== Cron task type configs =====
const cronConfigs = ref<CronTaskTypeConfig[]>(JSON.parse(JSON.stringify(mockCronTaskTypeConfigs)))
const selectedCronType = ref<string>(cronConfigs.value[0]?.task_type || '')

const selectedCronConfig = computed(() =>
  cronConfigs.value.find(c => c.task_type === selectedCronType.value)
)

const cronAiProviders = [
  { value: '本地部署', label: '本地部署' },
  { value: '云端API', label: '云端 API' },
]

const cronModelOptions: Record<string, string[]> = {
  '本地部署': ['Qwen2.5-72B', 'Qwen2.5-32B', 'ChatGLM4-9B', 'DeepSeek-V3'],
  '云端API': ['GPT-4o', 'GPT-4o-mini', 'Claude-3.5-Sonnet', 'Gemini-2.0-Flash'],
}

const pushFormatOptions = [
  { value: 'html', label: 'HTML 邮件' },
  { value: 'markdown', label: 'Markdown' },
  { value: 'plain', label: '纯文本' },
]

const cronPermissionLabels: Record<string, { label: string; desc: string }> = {
  allow_modify_email: { label: '修改推送邮箱', desc: '允许用户设置个人推送邮箱地址' },
  allow_modify_schedule: { label: '修改执行计划', desc: '允许用户调整定时任务的 Cron 表达式' },
  allow_modify_prompt: { label: '修改提示词', desc: '允许用户查看和修改 AI 提示词内容' },
  allow_modify_template: { label: '修改内容模板', desc: '允许用户自定义推送内容的模板结构' },
}

const cronActiveTab = ref('template')

const handleSaveCronConfig = () => {
  message.success('定时任务配置已保存')
}

const processConfigs = ref<ProcessAuditConfig[]>(JSON.parse(JSON.stringify(mockProcessAuditConfigs)))
const selectedProcessId = ref(processConfigs.value[0]?.id || '')

// ===== Add new process =====
const showAddProcess = ref(false)
const newProcessForm = ref({ process_type: '', flow_path: '' })

const handleAddProcess = () => {
  if (!newProcessForm.value.process_type.trim()) {
    message.warning('请输入流程名称')
    return
  }
  const newConfig: ProcessAuditConfig = {
    id: `PAC-${Date.now()}`,
    process_type: newProcessForm.value.process_type.trim(),
    flow_path: newProcessForm.value.flow_path.trim() || '待配置',
    field_mode: 'selected',
    fields: [],
    rules: [],
    kb_mode: 'rules_only',
    ai_config: {
      ai_provider: '本地部署',
      model_name: 'Qwen2.5-72B',
      audit_strictness: 'standard',
      system_prompt: '',
      context_window: 8192,
      temperature: 0.3,
    },
    user_permissions: {
      allow_custom_fields: false,
      allow_custom_rules: false,
      allow_modify_strictness: false,
    },
  }
  processConfigs.value.push(newConfig)
  selectedProcessId.value = newConfig.id
  showAddProcess.value = false
  newProcessForm.value = { process_type: '', flow_path: '' }
  message.success('流程已添加')
}
const activeTab = ref('fields')

const selectedConfig = computed(() =>
  processConfigs.value.find(c => c.id === selectedProcessId.value)
)

// ===== Field config =====
const fieldTypeLabels: Record<string, string> = {
  text: '文本', number: '数字', date: '日期', select: '下拉选择', textarea: '多行文本', file: '文件',
}

const toggleFieldSelection = (field: ProcessField) => {
  if (selectedConfig.value?.field_mode === 'all') return
  field.selected = !field.selected
}

const selectedFieldCount = computed(() =>
  selectedConfig.value?.fields.filter(f => f.selected).length || 0
)

// ===== Rules config =====
const scopeConfig: Record<string, { label: string; color: string; bg: string; icon: any }> = {
  mandatory: { label: '强制执行', color: 'var(--color-danger)', bg: 'var(--color-danger-bg)', icon: LockOutlined },
  default_on: { label: '默认开启', color: 'var(--color-primary)', bg: 'var(--color-primary-bg)', icon: UnlockOutlined },
  default_off: { label: '默认关闭', color: 'var(--color-text-tertiary)', bg: 'var(--color-bg-hover)', icon: UnlockOutlined },
}

const showRuleEditor = ref(false)
const editingRule = ref<AuditRule | null>(null)

const openRuleEditor = (rule?: AuditRule) => {
  editingRule.value = rule || null
  showRuleEditor.value = true
}

const handleSaveRule = (rule: any) => {
  if (!selectedConfig.value) return
  if (editingRule.value) {
    const idx = selectedConfig.value.rules.findIndex(r => r.id === editingRule.value!.id)
    if (idx >= 0) selectedConfig.value.rules[idx] = { ...editingRule.value, ...rule }
  } else {
    selectedConfig.value.rules.push({
      id: `R${Date.now()}`, process_type: selectedConfig.value.process_type,
      ...rule, enabled: true, source: 'manual' as const,
    })
  }
  showRuleEditor.value = false
  editingRule.value = null
  message.success('规则已保存')
}

const deleteRule = (id: string) => {
  if (!selectedConfig.value) return
  selectedConfig.value.rules = selectedConfig.value.rules.filter(r => r.id !== id)
  message.success('已删除')
}

const handleImportRules = () => {
  message.info('文件识别导入功能开发中，将支持从 PDF/Word/Excel 中提取规则')
}

const kbModes = [
  { key: 'rules_only', icon: FileTextOutlined, title: '仅规则库', desc: '结构化 Checklist 审核', available: true },
  { key: 'rag_only', icon: DatabaseOutlined, title: '仅制度库 (RAG)', desc: 'PDF/Word 文档检索增强', available: false },
  { key: 'hybrid', icon: ThunderboltOutlined, title: '混合模式', desc: '规则库 + 制度库联合审核', available: false },
]

// ===== AI config =====
const strictnessOptions = [
  { value: 'strict', label: '严格', desc: '所有规则严格执行，零容忍' },
  { value: 'standard', label: '标准', desc: '按规则默认配置执行' },
  { value: 'loose', label: '宽松', desc: '仅校验强制规则，其余仅提示' },
]

const aiProviders = [
  { value: '本地部署', label: '本地部署' },
  { value: '云端API', label: '云端 API' },
]

const modelOptions: Record<string, string[]> = {
  '本地部署': ['Qwen2.5-72B', 'Qwen2.5-32B', 'ChatGLM4-9B', 'DeepSeek-V3'],
  '云端API': ['GPT-4o', 'GPT-4o-mini', 'Claude-3.5-Sonnet', 'Gemini-2.0-Flash'],
}

// ===== User permissions =====
// ===== Archive review configs =====
const archiveConfigs = ref<ArchiveReviewConfig[]>(JSON.parse(JSON.stringify(mockArchiveReviewConfigs)))
const selectedArchiveId = ref(archiveConfigs.value[0]?.id || '')
const archiveActiveTab = ref('fields')

const selectedArchiveConfig = computed(() =>
  archiveConfigs.value.find(c => c.id === selectedArchiveId.value)
)

const archiveFieldCount = computed(() =>
  selectedArchiveConfig.value?.fields.filter(f => f.selected).length || 0
)

const toggleArchiveField = (field: ProcessField) => {
  if (selectedArchiveConfig.value?.field_mode === 'all') return
  field.selected = !field.selected
}

const showArchiveRuleEditor = ref(false)
const editingArchiveRule = ref<AuditRule | null>(null)

const openArchiveRuleEditor = (rule?: AuditRule) => {
  editingArchiveRule.value = rule || null
  showArchiveRuleEditor.value = true
}

const handleSaveArchiveRule = (rule: any) => {
  if (!selectedArchiveConfig.value) return
  if (editingArchiveRule.value) {
    const idx = selectedArchiveConfig.value.rules.findIndex(r => r.id === editingArchiveRule.value!.id)
    if (idx >= 0) selectedArchiveConfig.value.rules[idx] = { ...editingArchiveRule.value, ...rule }
  } else {
    selectedArchiveConfig.value.rules.push({
      id: `AR${Date.now()}`, process_type: selectedArchiveConfig.value.process_type,
      ...rule, enabled: true, source: 'manual' as const,
    })
  }
  showArchiveRuleEditor.value = false
  editingArchiveRule.value = null
  message.success('规则已保存')
}

const deleteArchiveRule = (id: string) => {
  if (!selectedArchiveConfig.value) return
  selectedArchiveConfig.value.rules = selectedArchiveConfig.value.rules.filter(r => r.id !== id)
  message.success('已删除')
}

// Flow rules
const showFlowRuleEditor = ref(false)
const editingFlowRule = ref<FlowRuleConfig | null>(null)
const flowRuleForm = ref({ rule_content: '', rule_scope: 'default_on' as string, priority: 50 })

const openFlowRuleEditor = (rule?: FlowRuleConfig) => {
  editingFlowRule.value = rule || null
  flowRuleForm.value = rule
    ? { rule_content: rule.rule_content, rule_scope: rule.rule_scope, priority: rule.priority }
    : { rule_content: '', rule_scope: 'default_on', priority: 50 }
  showFlowRuleEditor.value = true
}

const handleSaveFlowRule = () => {
  if (!selectedArchiveConfig.value || !flowRuleForm.value.rule_content.trim()) return
  if (editingFlowRule.value) {
    const idx = selectedArchiveConfig.value.flow_rules.findIndex(r => r.id === editingFlowRule.value!.id)
    if (idx >= 0) {
      selectedArchiveConfig.value.flow_rules[idx] = {
        ...editingFlowRule.value,
        ...flowRuleForm.value,
      }
    }
  } else {
    selectedArchiveConfig.value.flow_rules.push({
      id: `FR${Date.now()}`,
      ...flowRuleForm.value,
      rule_scope: flowRuleForm.value.rule_scope as any,
      enabled: true,
      source: 'manual',
    })
  }
  showFlowRuleEditor.value = false
  editingFlowRule.value = null
  message.success('审批流规则已保存')
}

const deleteFlowRule = (id: string) => {
  if (!selectedArchiveConfig.value) return
  selectedArchiveConfig.value.flow_rules = selectedArchiveConfig.value.flow_rules.filter(r => r.id !== id)
  message.success('已删除')
}

const archivePermissionLabels: Record<string, { label: string; desc: string }> = {
  allow_custom_fields: { label: '自定义复核字段', desc: '允许用户新增或切换参与复核的字段' },
  allow_custom_rules: { label: '自定义复核规则', desc: '允许用户新增、修改个人复核规则' },
  allow_custom_flow_rules: { label: '自定义审批流规则', desc: '允许用户新增个人审批流合规规则' },
  allow_modify_strictness: { label: '调整复核尺度', desc: '允许用户调整 AI 复核的严格/宽松程度' },
}

const handleSaveArchiveConfig = () => {
  message.success('归档复盘配置已保存')
}

const permissionLabels: Record<string, { label: string; desc: string }> = {
  allow_custom_fields: { label: '自定义审核字段', desc: '允许用户新增或切换参与审核的字段' },
  allow_custom_rules: { label: '自定义审核规则', desc: '允许用户新增、修改个人审核规则' },
  allow_modify_strictness: { label: '调整审核尺度', desc: '允许用户调整 AI 审核的严格/宽松程度' },
}

const handleSave = () => {
  message.success('配置已保存')
}
</script>

<template>
  <div class="tenant-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">规则配置</h1>
        <p class="page-subtitle">以流程为维度，配置审核规则、定时任务参数与归档复盘</p>
      </div>
    </div>

    <!-- Top-level tab: 审核工作台 / 定时任务配置 / 归档复盘 -->
    <div class="top-tab-nav">
      <button
        v-for="tab in [
          { key: 'audit', label: '审核工作台', icon: DashboardOutlined },
          { key: 'cron', label: '定时任务配置', icon: ClockCircleOutlined },
          { key: 'archive', label: '归档复盘', icon: FolderOpenOutlined },
        ]"
        :key="tab.key"
        class="top-tab-btn"
        :class="{ 'top-tab-btn--active': topTab === tab.key }"
        @click="topTab = tab.key as any"
      >
        <component :is="tab.icon" />
        {{ tab.label }}
      </button>
    </div>

    <!-- ==================== 审核工作台配置 ==================== -->
    <div v-if="topTab === 'audit'" class="main-layout">
      <!-- Left: process list -->
      <div class="process-nav">
        <div class="process-nav-header">
          <SettingOutlined />
          <span>审核流程</span>
          <button class="add-process-btn" @click="showAddProcess = true" title="新增流程">
            <PlusOutlined />
          </button>
        </div>
        <div
          v-for="cfg in processConfigs"
          :key="cfg.id"
          class="process-nav-item"
          :class="{ 'process-nav-item--active': selectedProcessId === cfg.id }"
          @click="selectedProcessId = cfg.id"
        >
          <div class="process-nav-name">{{ cfg.process_type }}</div>
          <div class="process-nav-path">{{ cfg.flow_path }}</div>
        </div>
      </div>

      <!-- Right: config panel -->
      <div v-if="selectedConfig" class="config-panel">
        <div class="config-panel-header">
          <h2 class="config-panel-title">{{ selectedConfig.process_type }}</h2>
          <p class="config-panel-subtitle">{{ selectedConfig.flow_path }}</p>
        </div>

        <!-- Sub tabs -->
        <div class="tab-nav">
          <button
            v-for="tab in [
              { key: 'fields', label: '字段配置', icon: AppstoreOutlined },
              { key: 'rules', label: '审核规则', icon: AuditOutlined },
              { key: 'ai', label: 'AI 配置', icon: RobotOutlined },
              { key: 'permissions', label: '用户权限', icon: SafetyCertificateOutlined },
            ]"
            :key="tab.key"
            class="tab-btn"
            :class="{ 'tab-btn--active': activeTab === tab.key }"
            @click="activeTab = tab.key"
          >
            <component :is="tab.icon" />
            {{ tab.label }}
          </button>
        </div>

        <!-- ========== Fields tab ========== -->
        <div v-if="activeTab === 'fields'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">传输 AI 的字段</h4>
              <p class="section-desc">选择参与 AI 审核的字段。全部字段模式下提示效果不如专门字段精准。</p>
            </div>
          </div>

          <div class="field-mode-switch">
            <div
              class="field-mode-option"
              :class="{ 'field-mode-option--active': selectedConfig.field_mode === 'selected' }"
              @click="selectedConfig.field_mode = 'selected'"
            >
              <div class="field-mode-radio" />
              <div>
                <div class="field-mode-label">选择字段</div>
                <div class="field-mode-desc">手动选择参与审核的字段（推荐）</div>
              </div>
            </div>
            <div
              class="field-mode-option"
              :class="{ 'field-mode-option--active': selectedConfig.field_mode === 'all' }"
              @click="selectedConfig.field_mode = 'all'"
            >
              <div class="field-mode-radio" />
              <div>
                <div class="field-mode-label">全部字段</div>
                <div class="field-mode-desc">所有字段均传输给 AI（信息量大，精准度可能下降）</div>
              </div>
            </div>
          </div>

          <div class="field-count" v-if="selectedConfig.field_mode === 'selected'">
            已选 {{ selectedFieldCount }} / {{ selectedConfig.fields.length }} 个字段
          </div>

          <div class="field-grid">
            <div
              v-for="field in selectedConfig.fields"
              :key="field.field_key"
              class="field-card"
              :class="{
                'field-card--selected': field.selected || selectedConfig.field_mode === 'all',
                'field-card--disabled': selectedConfig.field_mode === 'all',
              }"
              @click="toggleFieldSelection(field)"
            >
              <div class="field-card-check">
                <CheckOutlined v-if="field.selected || selectedConfig.field_mode === 'all'" />
              </div>
              <div class="field-card-info">
                <div class="field-card-name">{{ field.field_name }}</div>
                <div class="field-card-meta">
                  <span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span>
                  <span class="field-key">{{ field.field_key }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- ========== Rules tab ========== -->
        <div v-if="activeTab === 'rules'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">审核规则</h4>
              <p class="section-desc">为当前流程配置审核规则，支持手工添加或文件识别导入</p>
            </div>
          </div>

          <!-- KB mode selector -->
          <div class="kb-modes">
            <div
              v-for="mode in kbModes"
              :key="mode.key"
              class="kb-mode-card"
              :class="{
                'kb-mode-card--active': selectedConfig.kb_mode === mode.key,
                'kb-mode-card--disabled': !mode.available,
              }"
              @click="mode.available && (selectedConfig.kb_mode = mode.key as any)"
            >
              <div class="kb-mode-icon"><component :is="mode.icon" /></div>
              <div class="kb-mode-info">
                <div class="kb-mode-title">{{ mode.title }}</div>
                <div class="kb-mode-desc">{{ mode.desc }}</div>
              </div>
              <div v-if="selectedConfig.kb_mode === mode.key" class="kb-mode-check">✓</div>
              <div v-if="!mode.available" class="kb-mode-badge">即将推出</div>
            </div>
          </div>

          <div class="rules-toolbar">
            <span class="rules-count">共 {{ selectedConfig.rules.length }} 条规则</span>
            <div class="rules-toolbar-actions">
              <a-button @click="handleImportRules">
                <UploadOutlined /> 文件识别导入
              </a-button>
              <a-button type="primary" @click="openRuleEditor()">
                <PlusOutlined /> 手工添加
              </a-button>
            </div>
          </div>

          <div class="rules-list">
            <div v-for="rule in selectedConfig.rules" :key="rule.id" class="rule-card">
              <div class="rule-card-left">
                <div class="rule-scope-badge" :style="{ color: scopeConfig[rule.rule_scope]?.color, background: scopeConfig[rule.rule_scope]?.bg }">
                  <component :is="scopeConfig[rule.rule_scope]?.icon" />
                  {{ scopeConfig[rule.rule_scope]?.label }}
                </div>
                <div class="rule-card-body">
                  <div class="rule-card-content">{{ rule.rule_content }}</div>
                  <div class="rule-card-meta">
                    <span v-if="rule.source === 'file_import'" class="rule-source-tag">文件导入</span>
                    <span v-else class="rule-source-tag rule-source-tag--manual">手工添加</span>
                    <span>优先级: {{ rule.priority }}</span>
                  </div>
                </div>
              </div>
              <div class="rule-card-actions">
                <a-switch v-model:checked="rule.enabled" size="small" />
                <button class="icon-btn" @click="openRuleEditor(rule)"><EditOutlined /></button>
                <a-popconfirm title="确认删除此规则？" @confirm="deleteRule(rule.id)">
                  <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
                </a-popconfirm>
              </div>
            </div>
          </div>
        </div>

        <!-- ========== AI tab ========== -->
        <div v-if="activeTab === 'ai'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">AI 审核配置</h4>
              <p class="section-desc">配置对接的 AI 系统、审核尺度及提示词模板</p>
            </div>
          </div>

          <div class="ai-form">
            <div class="ai-form-row">
              <div class="ai-form-group">
                <label class="ai-form-label">AI 服务商</label>
                <a-select v-model:value="selectedConfig.ai_config.ai_provider" style="width: 100%;" size="large" placeholder="选择服务商">
                  <a-select-option v-for="p in aiProviders" :key="p.value" :value="p.value">{{ p.label }}</a-select-option>
                </a-select>
              </div>
              <div class="ai-form-group">
                <label class="ai-form-label">模型</label>
                <a-select v-model:value="selectedConfig.ai_config.model_name" style="width: 100%;" size="large" placeholder="选择模型">
                  <a-select-option
                    v-for="m in (modelOptions[selectedConfig.ai_config.ai_provider] || [])"
                    :key="m" :value="m"
                  >{{ m }}</a-select-option>
                </a-select>
              </div>
            </div>

            <div class="ai-form-group">
              <label class="ai-form-label">审核尺度</label>
              <div class="strictness-options">
                <div
                  v-for="opt in strictnessOptions"
                  :key="opt.value"
                  class="strictness-option"
                  :class="{ 'strictness-option--active': selectedConfig.ai_config.audit_strictness === opt.value }"
                  @click="selectedConfig.ai_config.audit_strictness = opt.value as any"
                >
                  <div class="strictness-option-radio" />
                  <div>
                    <div class="strictness-option-label">{{ opt.label }}</div>
                    <div class="strictness-option-desc">{{ opt.desc }}</div>
                  </div>
                </div>
              </div>
            </div>

            <div class="ai-form-group">
              <label class="ai-form-label">系统提示词（System Prompt）</label>
              <a-textarea
                v-model:value="selectedConfig.ai_config.system_prompt"
                :rows="6"
                placeholder="输入 AI 审核的系统提示词..."
              />
            </div>

            <div class="ai-form-row">
              <div class="ai-form-group">
                <label class="ai-form-label">上下文窗口</label>
                <a-input-number
                  v-model:value="selectedConfig.ai_config.context_window"
                  :min="1024" :max="131072" :step="1024"
                  style="width: 100%;" size="large"
                  :formatter="(v: number) => `${v} tokens`"
                />
              </div>
              <div class="ai-form-group">
                <label class="ai-form-label">Temperature</label>
                <a-slider
                  v-model:value="selectedConfig.ai_config.temperature"
                  :min="0" :max="1" :step="0.1"
                />
                <div class="slider-labels">
                  <span>精确 (0)</span>
                  <span>当前: {{ selectedConfig.ai_config.temperature }}</span>
                  <span>创意 (1)</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- ========== Permissions tab ========== -->
        <div v-if="activeTab === 'permissions'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">用户自定义权限</h4>
              <p class="section-desc">控制业务用户在个人设置中可以自定义的内容范围，以流程为维度独立管控</p>
            </div>
          </div>

          <div class="permissions-list">
            <div
              v-for="(perm, key) in permissionLabels"
              :key="key"
              class="permission-item"
            >
              <div class="permission-info">
                <div class="permission-label">{{ perm.label }}</div>
                <div class="permission-desc">{{ perm.desc }}</div>
              </div>
              <a-switch
                v-model:checked="(selectedConfig.user_permissions as any)[key]"
                :checked-children="'允许'"
                :un-checked-children="'禁止'"
              />
            </div>
          </div>
        </div>

        <div class="config-actions">
          <a-button type="primary" size="large" @click="handleSave">保存配置</a-button>
        </div>
      </div>

      <div v-else class="config-empty">
        <a-empty description="请选择左侧流程查看配置" />
      </div>
    </div>

    <!-- Rule editor modal -->
    <RuleEditor
      :open="showRuleEditor"
      :rule="editingRule"
      @close="showRuleEditor = false; editingRule = null"
      @save="handleSaveRule"
    />

    <!-- Add process modal -->
    <a-modal
      v-model:open="showAddProcess"
      title="新增审核流程"
      @ok="handleAddProcess"
      ok-text="确认"
      cancel-text="取消"
    >
      <a-form layout="vertical" style="margin-top: 16px;">
        <a-form-item label="流程名称" required>
          <a-input v-model:value="newProcessForm.process_type" placeholder="如：采购审批、费用报销" />
        </a-form-item>
        <a-form-item label="审批路径">
          <a-input v-model:value="newProcessForm.flow_path" placeholder="如：部门经理 → 财务总监 → 总经理" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- ==================== 定时任务配置 ==================== -->
    <div v-if="topTab === 'cron'" class="main-layout">
      <!-- Left: task type list -->
      <div class="process-nav">
        <div class="process-nav-header">
          <ClockCircleOutlined />
          <span>任务类型</span>
        </div>
        <div
          v-for="cfg in cronConfigs"
          :key="cfg.task_type"
          class="process-nav-item"
          :class="{ 'process-nav-item--active': selectedCronType === cfg.task_type }"
          @click="selectedCronType = cfg.task_type"
        >
          <div class="process-nav-name">{{ cfg.label }}</div>
          <div class="process-nav-path">
            <span :class="cfg.enabled ? 'status-dot status-dot--active' : 'status-dot'" />
            {{ cfg.enabled ? '已启用' : '已禁用' }}
          </div>
        </div>
      </div>

      <!-- Right: cron config panel -->
      <div v-if="selectedCronConfig" class="config-panel">
        <div class="config-panel-header" style="display: flex; justify-content: space-between; align-items: flex-start;">
          <div>
            <h2 class="config-panel-title">{{ selectedCronConfig.label }}</h2>
            <p class="config-panel-subtitle">配置该类型定时任务的内容模板、AI 参数及用户权限</p>
          </div>
          <a-switch
            v-model:checked="selectedCronConfig.enabled"
            :checked-children="'启用'"
            :un-checked-children="'禁用'"
          />
        </div>

        <!-- Sub tabs -->
        <div class="tab-nav">
          <button
            v-for="tab in [
              { key: 'template', label: '内容模板', icon: MailOutlined },
              { key: 'ai', label: 'AI 配置', icon: RobotOutlined },
              { key: 'permissions', label: '用户权限', icon: SafetyCertificateOutlined },
            ]"
            :key="tab.key"
            class="tab-btn"
            :class="{ 'tab-btn--active': cronActiveTab === tab.key }"
            @click="cronActiveTab = tab.key"
          >
            <component :is="tab.icon" />
            {{ tab.label }}
          </button>
        </div>

        <!-- ========== Content template tab ========== -->
        <div v-if="cronActiveTab === 'template'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">推送内容模板</h4>
              <p class="section-desc">配置推送邮件/消息的内容结构，支持变量占位符（如 <code>{<!-- -->{date}}</code>、<code>{<!-- -->{total}}</code>）</p>
            </div>
          </div>

          <!-- Push format -->
          <div class="ai-form-group" style="margin-bottom: 20px;">
            <label class="ai-form-label">内容格式</label>
            <div class="push-format-options">
              <div
                v-for="fmt in pushFormatOptions"
                :key="fmt.value"
                class="push-format-option"
                :class="{ 'push-format-option--active': selectedCronConfig.push_format === fmt.value }"
                @click="selectedCronConfig.push_format = fmt.value as any"
              >
                <div class="push-format-radio" />
                <span>{{ fmt.label }}</span>
              </div>
            </div>
          </div>

          <div class="ai-form">
            <div class="ai-form-group">
              <label class="ai-form-label">邮件主题</label>
              <a-input v-model:value="selectedCronConfig.content_template.subject" size="large" placeholder="如：【OA智审】批量审核结果通知 - {{date}}" />
            </div>
            <div class="ai-form-group">
              <label class="ai-form-label">头部内容</label>
              <a-input v-model:value="selectedCronConfig.content_template.header" size="large" placeholder="邮件开头的引导文字" />
            </div>
            <div class="ai-form-group">
              <label class="ai-form-label">正文模板</label>
              <a-textarea
                v-model:value="selectedCronConfig.content_template.body_template"
                :rows="4"
                placeholder="正文内容模板，支持变量占位符..."
              />
            </div>
            <div class="ai-form-group">
              <label class="ai-form-label">底部内容</label>
              <a-input v-model:value="selectedCronConfig.content_template.footer" size="large" placeholder="邮件底部的附加说明" />
            </div>
          </div>

          <!-- Content modules toggle -->
          <div style="margin-top: 20px;">
            <label class="ai-form-label" style="margin-bottom: 10px; display: block;">包含内容模块</label>
            <div class="permissions-list">
              <div class="permission-item">
                <div class="permission-info">
                  <div class="permission-label">AI 智能摘要</div>
                  <div class="permission-desc">在推送内容中包含 AI 生成的分析摘要</div>
                </div>
                <a-switch v-model:checked="selectedCronConfig.content_template.include_ai_summary" :checked-children="'包含'" :un-checked-children="'不含'" />
              </div>
              <div class="permission-item">
                <div class="permission-info">
                  <div class="permission-label">统计数据</div>
                  <div class="permission-desc">在推送内容中包含审核数量、通过率等统计信息</div>
                </div>
                <a-switch v-model:checked="selectedCronConfig.content_template.include_statistics" :checked-children="'包含'" :un-checked-children="'不含'" />
              </div>
              <div class="permission-item">
                <div class="permission-info">
                  <div class="permission-label">明细列表</div>
                  <div class="permission-desc">在推送内容中包含每条流程的审核明细</div>
                </div>
                <a-switch v-model:checked="selectedCronConfig.content_template.include_detail_list" :checked-children="'包含'" :un-checked-children="'不含'" />
              </div>
            </div>
          </div>
        </div>

        <!-- ========== AI config tab ========== -->
        <div v-if="cronActiveTab === 'ai'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">AI 审核配置</h4>
              <p class="section-desc">配置该任务类型使用的 AI 模型和提示词</p>
            </div>
          </div>
          <div class="ai-form">
            <div class="ai-form-row">
              <div class="ai-form-group">
                <label class="ai-form-label">AI 服务商</label>
                <a-select v-model:value="selectedCronConfig.ai_config.ai_provider" style="width: 100%;" size="large" placeholder="选择服务商">
                  <a-select-option v-for="p in cronAiProviders" :key="p.value" :value="p.value">{{ p.label }}</a-select-option>
                </a-select>
              </div>
              <div class="ai-form-group">
                <label class="ai-form-label">模型</label>
                <a-select v-model:value="selectedCronConfig.ai_config.model_name" style="width: 100%;" size="large" placeholder="选择模型">
                  <a-select-option
                    v-for="m in (cronModelOptions[selectedCronConfig.ai_config.ai_provider] || [])"
                    :key="m" :value="m"
                  >{{ m }}</a-select-option>
                </a-select>
              </div>
            </div>
            <div class="ai-form-group">
              <label class="ai-form-label">系统提示词（System Prompt）</label>
              <a-textarea
                v-model:value="selectedCronConfig.ai_config.system_prompt"
                :rows="5"
                placeholder="输入该任务类型的 AI 提示词..."
              />
            </div>
          </div>
        </div>

        <!-- ========== Permissions tab ========== -->
        <div v-if="cronActiveTab === 'permissions'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">用户自定义权限</h4>
              <p class="section-desc">控制业务用户在个人设置中可以自定义的定时任务配置范围</p>
            </div>
          </div>
          <div class="permissions-list">
            <div
              v-for="(perm, key) in cronPermissionLabels"
              :key="key"
              class="permission-item"
            >
              <div class="permission-info">
                <div class="permission-label">{{ perm.label }}</div>
                <div class="permission-desc">{{ perm.desc }}</div>
              </div>
              <a-switch
                v-model:checked="(selectedCronConfig.user_permissions as any)[key]"
                :checked-children="'允许'"
                :un-checked-children="'禁止'"
              />
            </div>
          </div>
        </div>

        <div class="config-actions">
          <a-button type="primary" size="large" @click="handleSaveCronConfig">保存配置</a-button>
        </div>
      </div>

      <div v-else class="config-empty">
        <a-empty description="请选择左侧任务类型查看配置" />
      </div>
    </div>

    <!-- ==================== 归档复盘配置 ==================== -->
    <div v-if="topTab === 'archive'" class="main-layout">
      <!-- Left: process list -->
      <div class="process-nav">
        <div class="process-nav-header">
          <SettingOutlined />
          <span>复核流程</span>
        </div>
        <div
          v-for="cfg in archiveConfigs"
          :key="cfg.id"
          class="process-nav-item"
          :class="{ 'process-nav-item--active': selectedArchiveId === cfg.id }"
          @click="selectedArchiveId = cfg.id"
        >
          <div class="process-nav-name">{{ cfg.process_type }}</div>
          <div class="process-nav-path">{{ cfg.flow_path }}</div>
        </div>
      </div>

      <!-- Right: archive config panel -->
      <div v-if="selectedArchiveConfig" class="config-panel">
        <div class="config-panel-header">
          <h2 class="config-panel-title">{{ selectedArchiveConfig.process_type }} - 归档复盘配置</h2>
          <p class="config-panel-subtitle">{{ selectedArchiveConfig.flow_path }}</p>
        </div>

        <!-- Sub tabs -->
        <div class="tab-nav">
          <button
            v-for="tab in [
              { key: 'fields', label: '字段配置', icon: AppstoreOutlined },
              { key: 'rules', label: '审核规则', icon: AuditOutlined },
              { key: 'flow_rules', label: '审批流规则', icon: ControlOutlined },
              { key: 'ai', label: 'AI 配置', icon: RobotOutlined },
              { key: 'permissions', label: '用户权限', icon: SafetyCertificateOutlined },
            ]"
            :key="tab.key"
            class="tab-btn"
            :class="{ 'tab-btn--active': archiveActiveTab === tab.key }"
            @click="archiveActiveTab = tab.key"
          >
            <component :is="tab.icon" />
            {{ tab.label }}
          </button>
        </div>

        <!-- ========== Fields tab ========== -->
        <div v-if="archiveActiveTab === 'fields'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">复核字段</h4>
              <p class="section-desc">选择参与归档合规复核的字段，用于字段校验环节</p>
            </div>
          </div>

          <div class="field-mode-switch">
            <div
              class="field-mode-option"
              :class="{ 'field-mode-option--active': selectedArchiveConfig.field_mode === 'selected' }"
              @click="selectedArchiveConfig.field_mode = 'selected'"
            >
              <div class="field-mode-radio" />
              <div>
                <div class="field-mode-label">选择字段</div>
                <div class="field-mode-desc">手动选择参与复核的字段（推荐）</div>
              </div>
            </div>
            <div
              class="field-mode-option"
              :class="{ 'field-mode-option--active': selectedArchiveConfig.field_mode === 'all' }"
              @click="selectedArchiveConfig.field_mode = 'all'"
            >
              <div class="field-mode-radio" />
              <div>
                <div class="field-mode-label">全部字段</div>
                <div class="field-mode-desc">所有字段均参与复核</div>
              </div>
            </div>
          </div>

          <div class="field-count" v-if="selectedArchiveConfig.field_mode === 'selected'">
            已选 {{ archiveFieldCount }} / {{ selectedArchiveConfig.fields.length }} 个字段
          </div>

          <div class="field-grid">
            <div
              v-for="field in selectedArchiveConfig.fields"
              :key="field.field_key"
              class="field-card"
              :class="{
                'field-card--selected': field.selected || selectedArchiveConfig.field_mode === 'all',
                'field-card--disabled': selectedArchiveConfig.field_mode === 'all',
              }"
              @click="toggleArchiveField(field)"
            >
              <div class="field-card-check">
                <CheckOutlined v-if="field.selected || selectedArchiveConfig.field_mode === 'all'" />
              </div>
              <div class="field-card-info">
                <div class="field-card-name">{{ field.field_name }}</div>
                <div class="field-card-meta">
                  <span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span>
                  <span class="field-key">{{ field.field_key }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- ========== Rules tab ========== -->
        <div v-if="archiveActiveTab === 'rules'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">复核规则</h4>
              <p class="section-desc">配置归档合规复核的字段校验和业务规则</p>
            </div>
          </div>

          <!-- KB mode selector -->
          <div class="kb-modes">
            <div
              v-for="mode in kbModes"
              :key="mode.key"
              class="kb-mode-card"
              :class="{
                'kb-mode-card--active': selectedArchiveConfig.kb_mode === mode.key,
                'kb-mode-card--disabled': !mode.available,
              }"
              @click="mode.available && (selectedArchiveConfig.kb_mode = mode.key as any)"
            >
              <div class="kb-mode-icon"><component :is="mode.icon" /></div>
              <div class="kb-mode-info">
                <div class="kb-mode-title">{{ mode.title }}</div>
                <div class="kb-mode-desc">{{ mode.desc }}</div>
              </div>
              <div v-if="selectedArchiveConfig.kb_mode === mode.key" class="kb-mode-check">✓</div>
              <div v-if="!mode.available" class="kb-mode-badge">即将推出</div>
            </div>
          </div>

          <div class="rules-toolbar">
            <span class="rules-count">共 {{ selectedArchiveConfig.rules.length }} 条规则</span>
            <div class="rules-toolbar-actions">
              <a-button type="primary" @click="openArchiveRuleEditor()">
                <PlusOutlined /> 手工添加
              </a-button>
            </div>
          </div>

          <div class="rules-list">
            <div v-for="rule in selectedArchiveConfig.rules" :key="rule.id" class="rule-card">
              <div class="rule-card-left">
                <div class="rule-scope-badge" :style="{ color: scopeConfig[rule.rule_scope]?.color, background: scopeConfig[rule.rule_scope]?.bg }">
                  <component :is="scopeConfig[rule.rule_scope]?.icon" />
                  {{ scopeConfig[rule.rule_scope]?.label }}
                </div>
                <div class="rule-card-body">
                  <div class="rule-card-content">{{ rule.rule_content }}</div>
                  <div class="rule-card-meta">
                    <span v-if="rule.source === 'file_import'" class="rule-source-tag">文件导入</span>
                    <span v-else class="rule-source-tag rule-source-tag--manual">手工添加</span>
                    <span>优先级: {{ rule.priority }}</span>
                  </div>
                </div>
              </div>
              <div class="rule-card-actions">
                <a-switch v-model:checked="rule.enabled" size="small" />
                <button class="icon-btn" @click="openArchiveRuleEditor(rule)"><EditOutlined /></button>
                <a-popconfirm title="确认删除此规则？" @confirm="deleteArchiveRule(rule.id)">
                  <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
                </a-popconfirm>
              </div>
            </div>
          </div>
        </div>

        <!-- ========== Flow rules tab ========== -->
        <div v-if="archiveActiveTab === 'flow_rules'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">审批流规则</h4>
              <p class="section-desc">配置整个审批流程是否符合要求的合规规则，如审批链完整性、节点顺序、时效等</p>
            </div>
          </div>

          <div class="rules-toolbar">
            <span class="rules-count">共 {{ selectedArchiveConfig.flow_rules.length }} 条审批流规则</span>
            <div class="rules-toolbar-actions">
              <a-button type="primary" @click="openFlowRuleEditor()">
                <PlusOutlined /> 添加审批流规则
              </a-button>
            </div>
          </div>

          <div class="rules-list">
            <div v-for="rule in selectedArchiveConfig.flow_rules" :key="rule.id" class="rule-card">
              <div class="rule-card-left">
                <div class="rule-scope-badge" :style="{ color: scopeConfig[rule.rule_scope]?.color, background: scopeConfig[rule.rule_scope]?.bg }">
                  <component :is="scopeConfig[rule.rule_scope]?.icon" />
                  {{ scopeConfig[rule.rule_scope]?.label }}
                </div>
                <div class="rule-card-body">
                  <div class="rule-card-content">{{ rule.rule_content }}</div>
                  <div class="rule-card-meta">
                    <span v-if="rule.source === 'file_import'" class="rule-source-tag">文件导入</span>
                    <span v-else class="rule-source-tag rule-source-tag--manual">手工添加</span>
                    <span>优先级: {{ rule.priority }}</span>
                  </div>
                </div>
              </div>
              <div class="rule-card-actions">
                <a-switch v-model:checked="rule.enabled" size="small" />
                <button class="icon-btn" @click="openFlowRuleEditor(rule)"><EditOutlined /></button>
                <a-popconfirm title="确认删除此规则？" @confirm="deleteFlowRule(rule.id)">
                  <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
                </a-popconfirm>
              </div>
            </div>
          </div>
        </div>

        <!-- ========== AI tab ========== -->
        <div v-if="archiveActiveTab === 'ai'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">AI 复核配置</h4>
              <p class="section-desc">配置归档合规复核使用的 AI 系统、审核尺度及提示词模板</p>
            </div>
          </div>

          <div class="ai-form">
            <div class="ai-form-row">
              <div class="ai-form-group">
                <label class="ai-form-label">AI 服务商</label>
                <a-select v-model:value="selectedArchiveConfig.ai_config.ai_provider" style="width: 100%;" size="large" placeholder="选择服务商">
                  <a-select-option v-for="p in aiProviders" :key="p.value" :value="p.value">{{ p.label }}</a-select-option>
                </a-select>
              </div>
              <div class="ai-form-group">
                <label class="ai-form-label">模型</label>
                <a-select v-model:value="selectedArchiveConfig.ai_config.model_name" style="width: 100%;" size="large" placeholder="选择模型">
                  <a-select-option
                    v-for="m in (modelOptions[selectedArchiveConfig.ai_config.ai_provider] || [])"
                    :key="m" :value="m"
                  >{{ m }}</a-select-option>
                </a-select>
              </div>
            </div>

            <div class="ai-form-group">
              <label class="ai-form-label">复核尺度</label>
              <div class="strictness-options">
                <div
                  v-for="opt in strictnessOptions"
                  :key="opt.value"
                  class="strictness-option"
                  :class="{ 'strictness-option--active': selectedArchiveConfig.ai_config.audit_strictness === opt.value }"
                  @click="selectedArchiveConfig.ai_config.audit_strictness = opt.value as any"
                >
                  <div class="strictness-option-radio" />
                  <div>
                    <div class="strictness-option-label">{{ opt.label }}</div>
                    <div class="strictness-option-desc">{{ opt.desc }}</div>
                  </div>
                </div>
              </div>
            </div>

            <div class="ai-form-group">
              <label class="ai-form-label">系统提示词（System Prompt）</label>
              <a-textarea
                v-model:value="selectedArchiveConfig.ai_config.system_prompt"
                :rows="6"
                placeholder="输入 AI 归档复核的系统提示词..."
              />
            </div>

            <div class="ai-form-row">
              <div class="ai-form-group">
                <label class="ai-form-label">上下文窗口</label>
                <a-input-number
                  v-model:value="selectedArchiveConfig.ai_config.context_window"
                  :min="1024" :max="131072" :step="1024"
                  style="width: 100%;" size="large"
                  :formatter="(v: number) => `${v} tokens`"
                />
              </div>
              <div class="ai-form-group">
                <label class="ai-form-label">Temperature</label>
                <a-slider
                  v-model:value="selectedArchiveConfig.ai_config.temperature"
                  :min="0" :max="1" :step="0.1"
                />
                <div class="slider-labels">
                  <span>精确 (0)</span>
                  <span>当前: {{ selectedArchiveConfig.ai_config.temperature }}</span>
                  <span>创意 (1)</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- ========== Permissions tab ========== -->
        <div v-if="archiveActiveTab === 'permissions'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">用户自定义权限</h4>
              <p class="section-desc">控制业务用户在个人设置中可以自定义的归档复盘配置范围</p>
            </div>
          </div>

          <div class="permissions-list">
            <div
              v-for="(perm, key) in archivePermissionLabels"
              :key="key"
              class="permission-item"
            >
              <div class="permission-info">
                <div class="permission-label">{{ perm.label }}</div>
                <div class="permission-desc">{{ perm.desc }}</div>
              </div>
              <a-switch
                v-model:checked="(selectedArchiveConfig.user_permissions as any)[key]"
                :checked-children="'允许'"
                :un-checked-children="'禁止'"
              />
            </div>
          </div>
        </div>

        <div class="config-actions">
          <a-button type="primary" size="large" @click="handleSaveArchiveConfig">保存配置</a-button>
        </div>
      </div>

      <div v-else class="config-empty">
        <a-empty description="请选择左侧流程查看归档复盘配置" />
      </div>
    </div>

    <!-- Archive rule editor modal -->
    <RuleEditor
      :open="showArchiveRuleEditor"
      :rule="editingArchiveRule"
      @close="showArchiveRuleEditor = false; editingArchiveRule = null"
      @save="handleSaveArchiveRule"
    />

    <!-- Flow rule editor modal -->
    <a-modal
      v-model:open="showFlowRuleEditor"
      :title="editingFlowRule ? '编辑审批流规则' : '新增审批流规则'"
      @ok="handleSaveFlowRule"
      ok-text="保存"
      cancel-text="取消"
      :width="520"
    >
      <a-form layout="vertical" style="margin-top: 16px;">
        <a-form-item label="规则内容">
          <a-textarea
            v-model:value="flowRuleForm.rule_content"
            :rows="3"
            placeholder="如：审批链须完整，不得跳过任何必要节点"
          />
        </a-form-item>
        <a-form-item label="规则级别">
          <a-radio-group v-model:value="flowRuleForm.rule_scope" button-style="solid">
            <a-radio-button value="mandatory">强制执行</a-radio-button>
            <a-radio-button value="default_on">默认开启</a-radio-button>
            <a-radio-button value="default_off">默认关闭</a-radio-button>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="优先级">
          <a-slider v-model:value="flowRuleForm.priority" :min="0" :max="100" />
          <div style="display: flex; justify-content: space-between; font-size: 12px; color: var(--color-text-tertiary);">
            <span>低</span>
            <span>当前: {{ flowRuleForm.priority }}</span>
            <span>高</span>
          </div>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.page-header { margin-bottom: 24px; }
.page-title { font-size: 24px; font-weight: 700; color: var(--color-text-primary); margin: 0; }
.page-subtitle { font-size: 14px; color: var(--color-text-tertiary); margin: 4px 0 0; }

/* Top-level tabs */
.top-tab-nav {
  display: flex; gap: 4px; background: var(--color-bg-hover); padding: 4px;
  border-radius: var(--radius-lg); margin-bottom: 24px; width: fit-content;
}
.top-tab-btn {
  display: flex; align-items: center; gap: 6px;
  padding: 8px 24px; border: none; background: transparent; border-radius: var(--radius-md);
  font-size: 14px; font-weight: 500; color: var(--color-text-secondary); cursor: pointer;
  transition: all var(--transition-fast);
}
.top-tab-btn:hover { color: var(--color-text-primary); }
.top-tab-btn--active { background: var(--color-bg-card); color: var(--color-primary); box-shadow: var(--shadow-xs); }

/* Main layout */
.main-layout { display: grid; grid-template-columns: 240px 1fr; gap: 20px; align-items: start; }

/* Process nav */
.process-nav {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); overflow: hidden; position: sticky; top: 20px;
}
.process-nav-header {
  padding: 14px 16px; border-bottom: 1px solid var(--color-border-light);
  font-size: 14px; font-weight: 600; color: var(--color-text-primary);
  display: flex; align-items: center; gap: 8px;
}
.add-process-btn {
  margin-left: auto; width: 26px; height: 26px; border-radius: var(--radius-md);
  border: 1px dashed var(--color-border); background: transparent; cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  color: var(--color-text-tertiary); font-size: 12px; transition: all var(--transition-fast);
}
.add-process-btn:hover { border-color: var(--color-primary); color: var(--color-primary); background: var(--color-primary-bg); }
.process-nav-item {
  padding: 12px 16px; cursor: pointer; transition: all var(--transition-fast);
  border-bottom: 1px solid var(--color-border-light);
}
.process-nav-item:last-child { border-bottom: none; }
.process-nav-item:hover { background: var(--color-bg-hover); }
.process-nav-item--active { background: var(--color-primary-bg); border-left: 3px solid var(--color-primary); }
.process-nav-name { font-size: 14px; font-weight: 500; color: var(--color-text-primary); margin-bottom: 2px; }
.process-nav-path { font-size: 12px; color: var(--color-text-tertiary); }

/* Config panel */
.config-panel {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); padding: 24px;
}
.config-panel-header { margin-bottom: 20px; }
.config-panel-title { font-size: 18px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 4px; }
.config-panel-subtitle { font-size: 13px; color: var(--color-text-tertiary); margin: 0; }
.config-empty {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); padding: 48px;
}

/* Tabs */
.tab-nav {
  display: flex; gap: 4px; background: var(--color-bg-hover); padding: 4px;
  border-radius: var(--radius-lg); margin-bottom: 24px; width: fit-content;
}
.tab-btn {
  display: flex; align-items: center; gap: 6px;
  padding: 8px 20px; border: none; background: transparent; border-radius: var(--radius-md);
  font-size: 14px; font-weight: 500; color: var(--color-text-secondary); cursor: pointer;
  transition: all var(--transition-fast);
}
.tab-btn:hover { color: var(--color-text-primary); }
.tab-btn--active { background: var(--color-bg-card); color: var(--color-primary); box-shadow: var(--shadow-xs); }

/* Section */
.section-header { margin-bottom: 16px; }
.section-title { font-size: 15px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 4px; }
.section-desc { font-size: 13px; color: var(--color-text-tertiary); margin: 0; }

/* Field mode switch */
.field-mode-switch { display: flex; gap: 12px; margin-bottom: 16px; }
.field-mode-option {
  display: flex; align-items: center; gap: 12px; padding: 12px 16px; flex: 1;
  border: 2px solid var(--color-border-light); border-radius: var(--radius-md);
  cursor: pointer; transition: all var(--transition-fast);
}
.field-mode-option:hover { border-color: var(--color-primary-lighter); }
.field-mode-option--active { border-color: var(--color-primary); background: var(--color-primary-bg); }
.field-mode-radio {
  width: 18px; height: 18px; border-radius: 50%; border: 2px solid var(--color-border);
  flex-shrink: 0; transition: all var(--transition-fast);
}
.field-mode-option--active .field-mode-radio { border-color: var(--color-primary); border-width: 5px; }
.field-mode-label { font-size: 14px; font-weight: 500; color: var(--color-text-primary); }
.field-mode-desc { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }

.field-count { font-size: 13px; color: var(--color-text-tertiary); margin-bottom: 12px; }

/* Field grid */
.field-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 10px; }
.field-card {
  display: flex; align-items: center; gap: 10px; padding: 12px 14px;
  border: 1px solid var(--color-border-light); border-radius: var(--radius-md);
  cursor: pointer; transition: all var(--transition-fast);
}
.field-card:hover { border-color: var(--color-primary-lighter); }
.field-card--selected { border-color: var(--color-primary); background: var(--color-primary-bg); }
.field-card--disabled { cursor: default; opacity: 0.7; }
.field-card-check {
  width: 22px; height: 22px; border-radius: 4px; border: 2px solid var(--color-border);
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
  font-size: 12px; color: #fff; transition: all var(--transition-fast);
}
.field-card--selected .field-card-check { background: var(--color-primary); border-color: var(--color-primary); }
.field-card-name { font-size: 13px; font-weight: 500; color: var(--color-text-primary); }
.field-card-meta { display: flex; align-items: center; gap: 6px; margin-top: 2px; }
.field-type-tag {
  font-size: 10px; font-weight: 600; padding: 1px 6px; border-radius: var(--radius-sm);
  background: var(--color-bg-hover); color: var(--color-text-tertiary);
}
.field-key { font-size: 11px; color: var(--color-text-tertiary); font-family: monospace; }

/* Rules */
.rules-toolbar {
  display: flex; justify-content: space-between; align-items: center; margin-bottom: 14px;
}
.rules-count { font-size: 13px; color: var(--color-text-tertiary); }
.rules-toolbar-actions { display: flex; gap: 8px; }

.rules-list { display: flex; flex-direction: column; gap: 10px; }
.rule-card {
  display: flex; align-items: center; justify-content: space-between;
  padding: 14px 18px; background: var(--color-bg-page); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); transition: all var(--transition-fast); gap: 16px;
}
.rule-card:hover { box-shadow: var(--shadow-sm); }
.rule-card-left { display: flex; align-items: flex-start; gap: 12px; flex: 1; min-width: 0; }
.rule-scope-badge {
  display: inline-flex; align-items: center; gap: 4px; font-size: 11px; font-weight: 600;
  padding: 4px 10px; border-radius: var(--radius-full); white-space: nowrap; flex-shrink: 0;
}
.rule-card-content { font-size: 14px; font-weight: 500; color: var(--color-text-primary); margin-bottom: 4px; }
.rule-card-meta { display: flex; align-items: center; gap: 8px; font-size: 12px; color: var(--color-text-tertiary); }
.rule-source-tag {
  font-size: 10px; font-weight: 500; padding: 1px 6px; border-radius: var(--radius-sm);
  background: var(--color-info-bg); color: var(--color-info);
}
.rule-source-tag--manual { background: var(--color-bg-hover); color: var(--color-text-tertiary); }
.rule-card-actions { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }

.icon-btn {
  width: 32px; height: 32px; border: 1px solid var(--color-border); background: transparent;
  border-radius: var(--radius-md); cursor: pointer; display: flex; align-items: center;
  justify-content: center; color: var(--color-text-tertiary); transition: all var(--transition-fast);
}
.icon-btn:hover { border-color: var(--color-primary); color: var(--color-primary); }
.icon-btn--danger:hover { border-color: var(--color-danger); color: var(--color-danger); }

/* KB modes */
.kb-modes { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; margin-bottom: 20px; }
.kb-mode-card {
  display: flex; align-items: center; gap: 12px; padding: 14px;
  background: var(--color-bg-page); border-radius: var(--radius-md);
  border: 2px solid var(--color-border-light); cursor: pointer;
  transition: all var(--transition-fast); position: relative;
}
.kb-mode-card:hover:not(.kb-mode-card--disabled) { border-color: var(--color-primary-lighter); }
.kb-mode-card--active { border-color: var(--color-primary); background: var(--color-primary-bg); }
.kb-mode-card--disabled { opacity: 0.5; cursor: not-allowed; }
.kb-mode-icon {
  width: 36px; height: 36px; border-radius: var(--radius-md); background: var(--color-bg-card);
  display: flex; align-items: center; justify-content: center; font-size: 16px;
  color: var(--color-primary); flex-shrink: 0;
}
.kb-mode-title { font-size: 13px; font-weight: 600; color: var(--color-text-primary); }
.kb-mode-desc { font-size: 11px; color: var(--color-text-tertiary); margin-top: 1px; }
.kb-mode-check {
  position: absolute; top: 8px; right: 8px; width: 20px; height: 20px; border-radius: 50%;
  background: var(--color-primary); color: #fff; font-size: 11px;
  display: flex; align-items: center; justify-content: center;
}
.kb-mode-badge {
  position: absolute; top: 8px; right: 8px; font-size: 10px; font-weight: 600;
  padding: 2px 6px; border-radius: var(--radius-full);
  background: var(--color-bg-hover); color: var(--color-text-tertiary);
}

/* AI form */
.ai-form { display: flex; flex-direction: column; gap: 20px; }
.ai-form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.ai-form-group { display: flex; flex-direction: column; gap: 6px; }
.ai-form-label { font-size: 13px; font-weight: 600; color: var(--color-text-primary); }
.slider-labels { display: flex; justify-content: space-between; font-size: 12px; color: var(--color-text-tertiary); }

/* Strictness */
.strictness-options { display: flex; gap: 10px; }
.strictness-option {
  display: flex; align-items: center; gap: 10px; padding: 10px 14px; flex: 1;
  border: 2px solid var(--color-border-light); border-radius: var(--radius-md);
  cursor: pointer; transition: all var(--transition-fast);
}
.strictness-option:hover { border-color: var(--color-primary-lighter); }
.strictness-option--active { border-color: var(--color-primary); background: var(--color-primary-bg); }
.strictness-option-radio {
  width: 16px; height: 16px; border-radius: 50%; border: 2px solid var(--color-border);
  flex-shrink: 0; transition: all var(--transition-fast);
}
.strictness-option--active .strictness-option-radio { border-color: var(--color-primary); border-width: 5px; }
.strictness-option-label { font-size: 13px; font-weight: 500; color: var(--color-text-primary); }
.strictness-option-desc { font-size: 11px; color: var(--color-text-tertiary); margin-top: 1px; }

/* Permissions */
.permissions-list { display: flex; flex-direction: column; gap: 12px; }
.permission-item {
  display: flex; align-items: center; justify-content: space-between; gap: 16px;
  padding: 16px 20px; background: var(--color-bg-page); border-radius: var(--radius-md);
  border: 1px solid var(--color-border-light);
}
.permission-label { font-size: 14px; font-weight: 500; color: var(--color-text-primary); }
.permission-desc { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }

.config-actions { margin-top: 24px; display: flex; justify-content: flex-end; }

@media (max-width: 768px) {
  .main-layout { grid-template-columns: 1fr; }
  .field-mode-switch { flex-direction: column; }
  .field-grid { grid-template-columns: 1fr; }
  .kb-modes { grid-template-columns: 1fr; }
  .ai-form-row { grid-template-columns: 1fr; }
  .strictness-options { flex-direction: column; }
  .tab-nav { width: 100%; overflow-x: auto; }
  .tab-btn { flex-shrink: 0; }
  .push-format-options { flex-direction: column; }
}

/* Cron config sections */
.cron-config-section { margin-bottom: 24px; }

.status-dot {
  display: inline-block; width: 6px; height: 6px; border-radius: 50%;
  background: var(--color-text-tertiary); margin-right: 4px;
}
.status-dot--active { background: var(--color-success); }

.push-format-options { display: flex; gap: 10px; }
.push-format-option {
  display: flex; align-items: center; gap: 10px; padding: 10px 16px; flex: 1;
  border: 2px solid var(--color-border-light); border-radius: var(--radius-md);
  cursor: pointer; transition: all var(--transition-fast);
  font-size: 13px; font-weight: 500; color: var(--color-text-primary);
}
.push-format-option:hover { border-color: var(--color-primary-lighter); }
.push-format-option--active { border-color: var(--color-primary); background: var(--color-primary-bg); }
.push-format-radio {
  width: 16px; height: 16px; border-radius: 50%; border: 2px solid var(--color-border);
  flex-shrink: 0; transition: all var(--transition-fast);
}
.push-format-option--active .push-format-radio { border-color: var(--color-primary); border-width: 5px; }
</style>
