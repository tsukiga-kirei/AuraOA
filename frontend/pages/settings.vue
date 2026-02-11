<script setup lang="ts">
import {
  UserOutlined,
  MailOutlined,
  PhoneOutlined,
  SaveOutlined,
  PlusOutlined,
  DeleteOutlined,
  SettingOutlined,
  LockOutlined,
  CheckOutlined,
  EditOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type { ProcessAuditConfig, ProcessField, AuditRule } from '~/composables/useMockData'

definePageMeta({ middleware: 'auth' })

const { userRole } = useAuth()
const { mockProcessAuditConfigs } = useMockData()

const activeTab = ref('profile')

// ===== Profile tab =====
const profile = ref({
  nickname: '张明',
  email: 'zhangming@example.com',
  phone: '138****8888',
  department: '研发部',
  position: '高级工程师',
})

const roleLabels: Record<string, string> = {
  business: '业务用户',
  tenant_admin: '租户管理员',
  system_admin: '系统管理员',
}

// ===== Audit workbench tab =====
// Deep clone tenant configs as user's working copy
const userProcessConfigs = ref<ProcessAuditConfig[]>(
  JSON.parse(JSON.stringify(mockProcessAuditConfigs))
)

// User's custom rules per process (separate from tenant rules)
const userCustomRules = ref<Record<string, { id: string; content: string; enabled: boolean }[]>>({
  'PAC-001': [{ id: 'UCR-001', content: '供应商必须在合格名录中', enabled: true }],
  'PAC-002': [],
  'PAC-003': [{ id: 'UCR-002', content: '合同期限超过2年需额外审批', enabled: true }],
  'PAC-004': [],
})

// User's custom field overrides (additional selected fields)
const userFieldOverrides = ref<Record<string, string[]>>({
  'PAC-004': ['salary_range'],
})

const selectedProcessId = ref(userProcessConfigs.value[0]?.id || '')

const selectedConfig = computed(() =>
  userProcessConfigs.value.find(c => c.id === selectedProcessId.value)
)

const permissions = computed(() => selectedConfig.value?.user_permissions)

// Field type labels
const fieldTypeLabels: Record<string, string> = {
  text: '文本', number: '数字', date: '日期', select: '下拉选择', textarea: '多行文本', file: '文件',
}

// Scope config
const scopeConfig: Record<string, { label: string; color: string }> = {
  mandatory: { label: '强制', color: 'var(--color-danger)' },
  default_on: { label: '默认开启', color: 'var(--color-primary)' },
  default_off: { label: '默认关闭', color: 'var(--color-text-tertiary)' },
}

// Strictness
const strictnessOptions = [
  { value: 'strict', label: '严格', desc: '所有规则严格执行，零容忍' },
  { value: 'standard', label: '标准', desc: '按规则默认配置执行' },
  { value: 'loose', label: '宽松', desc: '仅校验强制规则，其余提示' },
]

// Toggle user field override
const toggleUserField = (field: ProcessField) => {
  if (!selectedConfig.value || !permissions.value?.allow_custom_fields) return
  if (selectedConfig.value.field_mode === 'all') return
  field.selected = !field.selected
}

// Custom rules
const newRuleContent = ref('')

const addCustomRule = () => {
  if (!newRuleContent.value.trim() || !selectedConfig.value) return
  const pid = selectedConfig.value.id
  if (!userCustomRules.value[pid]) userCustomRules.value[pid] = []
  userCustomRules.value[pid].push({
    id: `UCR-${Date.now()}`,
    content: newRuleContent.value.trim(),
    enabled: true,
  })
  newRuleContent.value = ''
  message.success('自定义规则已添加')
}

const removeCustomRule = (ruleId: string) => {
  if (!selectedConfig.value) return
  const pid = selectedConfig.value.id
  userCustomRules.value[pid] = (userCustomRules.value[pid] || []).filter(r => r.id !== ruleId)
  message.success('已删除')
}

const currentCustomRules = computed(() =>
  userCustomRules.value[selectedConfig.value?.id || ''] || []
)

const saving = ref(false)
const handleSave = async () => {
  saving.value = true
  await new Promise(r => setTimeout(r, 800))
  saving.value = false
  message.success('设置已保存')
}

// Active workbench sub-section
const workbenchSection = ref('fields')
</script>

<template>
  <div class="settings-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">个人设置</h1>
        <p class="page-subtitle">管理您的账户信息与审核偏好</p>
      </div>
    </div>

    <!-- Tab navigation -->
    <div class="tab-nav">
      <button
        v-for="tab in [
          { key: 'profile', label: '基本信息' },
          { key: 'workbench', label: '审核工作台' },
        ]"
        :key="tab.key"
        class="tab-btn"
        :class="{ 'tab-btn--active': activeTab === tab.key }"
        @click="activeTab = tab.key"
      >
        {{ tab.label }}
      </button>
    </div>

    <!-- Profile tab -->
    <div v-if="activeTab === 'profile'" class="tab-content">
      <div class="settings-card">
        <div class="profile-avatar-section">
          <a-avatar :size="72" class="profile-avatar">
            <template #icon><UserOutlined /></template>
          </a-avatar>
          <div class="profile-avatar-info">
            <div class="profile-name">{{ profile.nickname }}</div>
            <div class="profile-role">
              <span class="role-badge">{{ roleLabels[userRole] || '业务用户' }}</span>
            </div>
          </div>
        </div>

        <a-form layout="vertical" class="settings-form">
          <div class="form-row">
            <a-form-item label="昵称" class="form-col">
              <a-input v-model:value="profile.nickname" size="large">
                <template #prefix><UserOutlined class="input-icon" /></template>
              </a-input>
            </a-form-item>
            <a-form-item label="邮箱" class="form-col">
              <a-input v-model:value="profile.email" size="large">
                <template #prefix><MailOutlined class="input-icon" /></template>
              </a-input>
            </a-form-item>
          </div>
          <div class="form-row">
            <a-form-item label="手机号" class="form-col">
              <a-input v-model:value="profile.phone" size="large">
                <template #prefix><PhoneOutlined class="input-icon" /></template>
              </a-input>
            </a-form-item>
            <a-form-item label="部门" class="form-col">
              <a-input v-model:value="profile.department" size="large" disabled />
            </a-form-item>
          </div>
          <a-form-item label="职位">
            <a-input v-model:value="profile.position" size="large" disabled />
          </a-form-item>
        </a-form>

        <div class="settings-actions">
          <a-button type="primary" size="large" :loading="saving" @click="handleSave">
            <SaveOutlined /> 保存
          </a-button>
        </div>
      </div>
    </div>
    <!-- Audit workbench tab -->
    <div v-if="activeTab === 'workbench'" class="tab-content">
      <div class="workbench-layout">
        <!-- Left: process list -->
        <div class="process-list-panel">
          <div class="process-list-header">
            <SettingOutlined />
            <span>我的审核流程</span>
          </div>
          <div
            v-for="proc in userProcessConfigs"
            :key="proc.id"
            class="process-list-item"
            :class="{ 'process-list-item--active': selectedProcessId === proc.id }"
            @click="selectedProcessId = proc.id"
          >
            <div class="process-list-item-name">{{ proc.process_type }}</div>
            <div class="process-list-item-path">{{ proc.flow_path }}</div>
          </div>
        </div>

        <!-- Right: config detail -->
        <div v-if="selectedConfig" class="process-config-panel">
          <h3 class="config-title">{{ selectedConfig.process_type }} - 个人审核配置</h3>
          <p class="config-subtitle">流程路径：{{ selectedConfig.flow_path }}</p>

          <!-- Sub-section nav -->
          <div class="section-nav">
            <button
              v-for="sec in [
                { key: 'fields', label: '审核字段' },
                { key: 'rules', label: '审核规则' },
                { key: 'ai', label: '审核尺度' },
              ]"
              :key="sec.key"
              class="section-nav-btn"
              :class="{ 'section-nav-btn--active': workbenchSection === sec.key }"
              @click="workbenchSection = sec.key"
            >
              {{ sec.label }}
            </button>
          </div>

          <!-- ===== Fields section ===== -->
          <div v-if="workbenchSection === 'fields'" class="config-section">
            <div class="section-header-row">
              <h4 class="config-section-title">传输 AI 的字段</h4>
              <span v-if="!permissions?.allow_custom_fields" class="locked-tag">
                <LockOutlined /> 管理员已锁定
              </span>
            </div>
            <p class="config-section-desc">
              {{ selectedConfig.field_mode === 'all' ? '当前为全部字段模式' : '以下为参与 AI 审核的字段配置' }}
              <template v-if="permissions?.allow_custom_fields && selectedConfig.field_mode === 'selected'">
                ，您可以切换字段的选中状态
              </template>
            </p>

            <div class="field-grid">
              <div
                v-for="field in selectedConfig.fields"
                :key="field.field_key"
                class="field-card"
                :class="{
                  'field-card--selected': field.selected || selectedConfig.field_mode === 'all',
                  'field-card--readonly': !permissions?.allow_custom_fields || selectedConfig.field_mode === 'all',
                }"
                @click="toggleUserField(field)"
              >
                <div class="field-card-check">
                  <CheckOutlined v-if="field.selected || selectedConfig.field_mode === 'all'" />
                </div>
                <div class="field-card-info">
                  <div class="field-card-name">{{ field.field_name }}</div>
                  <span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span>
                </div>
              </div>
            </div>
          </div>

          <!-- ===== Rules section ===== -->
          <div v-if="workbenchSection === 'rules'" class="config-section">
            <!-- System rules (from tenant config) -->
            <div class="section-header-row">
              <h4 class="config-section-title">通用审核规则（租户配置）</h4>
            </div>
            <div class="rule-config-list">
              <div v-for="rule in selectedConfig.rules" :key="rule.id" class="rule-config-item">
                <div class="rule-config-content">
                  <span class="rule-config-text">{{ rule.rule_content }}</span>
                  <span
                    class="rule-scope-tag"
                    :class="{
                      'rule-scope-tag--mandatory': rule.rule_scope === 'mandatory',
                      'rule-scope-tag--on': rule.rule_scope === 'default_on',
                      'rule-scope-tag--off': rule.rule_scope === 'default_off',
                    }"
                  >{{ scopeConfig[rule.rule_scope]?.label }}</span>
                </div>
                <a-switch
                  v-model:checked="rule.enabled"
                  size="small"
                  :disabled="rule.rule_scope === 'mandatory'"
                />
              </div>
            </div>

            <!-- Custom rules (user private) -->
            <div class="section-header-row" style="margin-top: 20px;">
              <h4 class="config-section-title">个人自定义规则</h4>
              <span v-if="!permissions?.allow_custom_rules" class="locked-tag">
                <LockOutlined /> 管理员已锁定
              </span>
            </div>
            <p class="config-section-desc">
              {{ permissions?.allow_custom_rules ? '您可以为此流程添加个人审核规则，优先级低于租户强制规则' : '当前流程不允许添加个人规则' }}
            </p>

            <div class="rule-config-list" v-if="currentCustomRules.length > 0">
              <div v-for="rule in currentCustomRules" :key="rule.id" class="rule-config-item">
                <div class="rule-config-content">
                  <span class="rule-config-text">{{ rule.content }}</span>
                  <span class="rule-scope-tag rule-scope-tag--custom">个人</span>
                </div>
                <div class="rule-config-actions">
                  <a-switch v-model:checked="rule.enabled" size="small" />
                  <a-popconfirm v-if="permissions?.allow_custom_rules" title="确认删除？" @confirm="removeCustomRule(rule.id)">
                    <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
                  </a-popconfirm>
                </div>
              </div>
            </div>

            <div v-if="permissions?.allow_custom_rules" class="add-rule-row">
              <a-input
                v-model:value="newRuleContent"
                placeholder="输入自定义规则内容..."
                @pressEnter="addCustomRule"
              />
              <a-button type="primary" :disabled="!newRuleContent.trim()" @click="addCustomRule">
                <PlusOutlined /> 添加
              </a-button>
            </div>
          </div>

          <!-- ===== AI strictness section ===== -->
          <div v-if="workbenchSection === 'ai'" class="config-section">
            <div class="section-header-row">
              <h4 class="config-section-title">审核尺度</h4>
              <span v-if="!permissions?.allow_modify_strictness" class="locked-tag">
                <LockOutlined /> 管理员已锁定
              </span>
            </div>
            <p class="config-section-desc">
              当前 AI 模型：{{ selectedConfig.ai_config.model_name }}（{{ selectedConfig.ai_config.ai_provider }}）
            </p>
            <div class="strictness-options">
              <div
                v-for="opt in strictnessOptions"
                :key="opt.value"
                class="strictness-option"
                :class="{
                  'strictness-option--active': selectedConfig.ai_config.audit_strictness === opt.value,
                  'strictness-option--disabled': !permissions?.allow_modify_strictness,
                }"
                @click="permissions?.allow_modify_strictness && (selectedConfig.ai_config.audit_strictness = opt.value as any)"
              >
                <div class="strictness-option-radio" />
                <div>
                  <div class="strictness-option-label">{{ opt.label }}</div>
                  <div class="strictness-option-desc">{{ opt.desc }}</div>
                </div>
              </div>
            </div>

            <!-- Knowledge base mode (read-only display) -->
            <div style="margin-top: 20px;">
              <h4 class="config-section-title">知识库模式</h4>
              <p class="config-section-desc">
                当前模式：<span style="font-weight: 600;">
                  {{ selectedConfig.kb_mode === 'rules_only' ? '仅规则库' : selectedConfig.kb_mode === 'rag_only' ? '仅制度库' : '混合模式' }}
                </span>
                （由管理员配置）
              </p>
            </div>
          </div>

          <div class="settings-actions">
            <a-button type="primary" size="large" :loading="saving" @click="handleSave">
              <SaveOutlined /> 保存配置
            </a-button>
          </div>
        </div>

        <div v-else class="process-config-empty">
          <a-empty description="请选择左侧流程查看配置" />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page-header { margin-bottom: 24px; }
.page-title { font-size: 24px; font-weight: 700; color: var(--color-text-primary); margin: 0; }
.page-subtitle { font-size: 14px; color: var(--color-text-tertiary); margin: 4px 0 0; }

/* Tabs */
.tab-nav {
  display: flex; gap: 4px; background: var(--color-bg-hover); padding: 4px;
  border-radius: var(--radius-lg); margin-bottom: 24px; width: fit-content;
}
.tab-btn {
  padding: 8px 20px; border: none; background: transparent; border-radius: var(--radius-md);
  font-size: 14px; font-weight: 500; color: var(--color-text-secondary); cursor: pointer;
  transition: all var(--transition-fast);
}
.tab-btn:hover { color: var(--color-text-primary); }
.tab-btn--active { background: var(--color-bg-card); color: var(--color-primary); box-shadow: var(--shadow-xs); }

/* Settings card */
.settings-card {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); padding: 24px; max-width: 700px;
}
.profile-avatar-section { display: flex; align-items: center; gap: 16px; margin-bottom: 24px; }
.profile-avatar { background: linear-gradient(135deg, #4f46e5, #7c3aed) !important; flex-shrink: 0; }
.profile-name { font-size: 18px; font-weight: 600; color: var(--color-text-primary); }
.role-badge {
  font-size: 12px; font-weight: 500; padding: 2px 10px; border-radius: var(--radius-full);
  background: var(--color-primary-bg); color: var(--color-primary);
}
.settings-form :deep(.ant-form-item) { margin-bottom: 16px; }
.form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.input-icon { color: var(--color-text-tertiary); }
.settings-actions { margin-top: 24px; display: flex; justify-content: flex-end; }

/* Workbench layout */
.workbench-layout { display: grid; grid-template-columns: 240px 1fr; gap: 20px; align-items: start; }

.process-list-panel {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); overflow: hidden;
}
.process-list-header {
  padding: 14px 16px; border-bottom: 1px solid var(--color-border-light);
  font-size: 14px; font-weight: 600; color: var(--color-text-primary);
  display: flex; align-items: center; gap: 8px;
}
.process-list-item {
  padding: 12px 16px; cursor: pointer; transition: all var(--transition-fast);
  border-bottom: 1px solid var(--color-border-light);
}
.process-list-item:last-child { border-bottom: none; }
.process-list-item:hover { background: var(--color-bg-hover); }
.process-list-item--active { background: var(--color-primary-bg); border-left: 3px solid var(--color-primary); }
.process-list-item-name { font-size: 14px; font-weight: 500; color: var(--color-text-primary); margin-bottom: 2px; }
.process-list-item-path { font-size: 12px; color: var(--color-text-tertiary); }

.process-config-panel {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); padding: 24px;
}
.process-config-empty {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); padding: 48px;
}

.config-title { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 4px; }
.config-subtitle { font-size: 13px; color: var(--color-text-tertiary); margin: 0 0 16px; }

/* Section nav */
.section-nav {
  display: flex; gap: 4px; background: var(--color-bg-hover); padding: 3px;
  border-radius: var(--radius-md); margin-bottom: 20px; width: fit-content;
}
.section-nav-btn {
  padding: 6px 16px; border: none; background: transparent; border-radius: var(--radius-sm);
  font-size: 13px; font-weight: 500; color: var(--color-text-secondary); cursor: pointer;
  transition: all var(--transition-fast);
}
.section-nav-btn:hover { color: var(--color-text-primary); }
.section-nav-btn--active { background: var(--color-bg-card); color: var(--color-primary); box-shadow: var(--shadow-xs); }

.config-section { margin-bottom: 20px; }
.section-header-row { display: flex; align-items: center; gap: 10px; margin-bottom: 8px; }
.config-section-title { font-size: 14px; font-weight: 600; color: var(--color-text-primary); margin: 0; }
.config-section-desc { font-size: 12px; color: var(--color-text-tertiary); margin: 0 0 12px; }

.locked-tag {
  display: inline-flex; align-items: center; gap: 4px; font-size: 11px; font-weight: 500;
  padding: 2px 8px; border-radius: var(--radius-full);
  background: var(--color-warning-bg); color: var(--color-warning);
}

/* Field grid */
.field-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 8px; }
.field-card {
  display: flex; align-items: center; gap: 10px; padding: 10px 12px;
  border: 1px solid var(--color-border-light); border-radius: var(--radius-md);
  cursor: pointer; transition: all var(--transition-fast);
}
.field-card:hover:not(.field-card--readonly) { border-color: var(--color-primary-lighter); }
.field-card--selected { border-color: var(--color-primary); background: var(--color-primary-bg); }
.field-card--readonly { cursor: default; opacity: 0.8; }
.field-card-check {
  width: 20px; height: 20px; border-radius: 4px; border: 2px solid var(--color-border);
  display: flex; align-items: center; justify-content: center; flex-shrink: 0;
  font-size: 11px; color: #fff; transition: all var(--transition-fast);
}
.field-card--selected .field-card-check { background: var(--color-primary); border-color: var(--color-primary); }
.field-card-name { font-size: 13px; font-weight: 500; color: var(--color-text-primary); }
.field-type-tag {
  font-size: 10px; font-weight: 600; padding: 1px 6px; border-radius: var(--radius-sm);
  background: var(--color-bg-hover); color: var(--color-text-tertiary);
}

/* Rule config list */
.rule-config-list { display: flex; flex-direction: column; gap: 8px; margin-bottom: 12px; }
.rule-config-item {
  display: flex; align-items: center; justify-content: space-between; gap: 12px;
  padding: 10px 14px; background: var(--color-bg-page); border-radius: var(--radius-md);
}
.rule-config-content { display: flex; align-items: center; gap: 8px; flex: 1; min-width: 0; }
.rule-config-text { font-size: 13px; color: var(--color-text-primary); }
.rule-config-actions { display: flex; align-items: center; gap: 8px; }

.rule-scope-tag {
  font-size: 10px; font-weight: 600; padding: 2px 8px; border-radius: var(--radius-full);
  white-space: nowrap; flex-shrink: 0;
}
.rule-scope-tag--mandatory { background: var(--color-danger-bg); color: var(--color-danger); }
.rule-scope-tag--on { background: var(--color-primary-bg); color: var(--color-primary); }
.rule-scope-tag--off { background: var(--color-bg-hover); color: var(--color-text-tertiary); }
.rule-scope-tag--custom { background: var(--color-info-bg); color: var(--color-info); }

.icon-btn {
  width: 28px; height: 28px; border: 1px solid var(--color-border); background: transparent;
  border-radius: var(--radius-sm); cursor: pointer; display: flex; align-items: center;
  justify-content: center; color: var(--color-text-tertiary); transition: all var(--transition-fast);
}
.icon-btn--danger:hover { border-color: var(--color-danger); color: var(--color-danger); }

.add-rule-row { display: flex; gap: 8px; }
.add-rule-row :deep(.ant-btn-primary) { font-weight: 600; min-width: 80px; }
.add-rule-row :deep(.ant-btn-primary[disabled]) { background: var(--color-primary); opacity: 0.5; color: #fff; }

/* Strictness options */
.strictness-options { display: flex; flex-direction: column; gap: 8px; }
.strictness-option {
  display: flex; align-items: center; gap: 14px; padding: 12px 16px;
  border: 2px solid var(--color-border-light); border-radius: var(--radius-md);
  cursor: pointer; transition: all var(--transition-fast);
}
.strictness-option:hover:not(.strictness-option--disabled) { border-color: var(--color-primary-lighter); }
.strictness-option--active { border-color: var(--color-primary); background: var(--color-primary-bg); }
.strictness-option--disabled { cursor: not-allowed; opacity: 0.6; }
.strictness-option-radio {
  width: 18px; height: 18px; border-radius: 50%; border: 2px solid var(--color-border);
  flex-shrink: 0; transition: all var(--transition-fast);
}
.strictness-option--active .strictness-option-radio { border-color: var(--color-primary); border-width: 5px; }
.strictness-option-label { font-size: 14px; font-weight: 500; color: var(--color-text-primary); }
.strictness-option-desc { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }

@media (max-width: 768px) {
  .form-row { grid-template-columns: 1fr; }
  .workbench-layout { grid-template-columns: 1fr; }
  .field-grid { grid-template-columns: 1fr 1fr; }
}
</style>
