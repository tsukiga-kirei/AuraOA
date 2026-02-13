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
  ClockCircleOutlined,
  SafetyCertificateOutlined,
  NodeIndexOutlined,
  DashboardOutlined,
  FolderOpenOutlined,
  AppstoreOutlined,
  FileTextOutlined,
  RobotOutlined,
  ControlOutlined,
  SendOutlined,
  AuditOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type { ProcessAuditConfig, ProcessField, AuditRule, CronTaskTypeConfig, ArchiveReviewConfig } from '~/composables/useMockData'

// All layouts now use the same unified sidebar, so just use default.
definePageMeta({
  middleware: 'auth',
  layout: 'default',
})

const { userRole } = useAuth()
const { mockProcessAuditConfigs, mockCronTaskTypeConfigs, mockArchiveReviewConfigs, mockOrgRoles, mockOrgMembers } = useMockData()

const activeTab = ref('profile')

// ===== Profile tab =====
// Find current user's org member record to show role-based permissions
const currentMember = computed(() => {
  const { currentUser } = useAuth()
  return mockOrgMembers.find(m => m.username === currentUser.value?.username) || null
})
const currentOrgRole = computed(() => {
  if (!currentMember.value) return null
  return mockOrgRoles.find(r => r.id === currentMember.value!.role_id) || null
})

const allPageLabels: Record<string, string> = {
  '/dashboard': '审核工作台',
  '/cron': '定时任务',
  '/archive': '归档复盘',
  '/settings': '个人设置',
  '/admin/tenant': '规则配置',
  '/admin/tenant/org': '组织人员',
  '/admin/tenant/data': '数据信息',
  '/admin/system': '全局监控',
  '/admin/system/tenants': '租户管理',
  '/admin/system/settings': '系统设置',
}

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

// ===== Cron personal settings =====
const userCronConfigs = ref<CronTaskTypeConfig[]>(
  JSON.parse(JSON.stringify(mockCronTaskTypeConfigs))
)
const selectedCronType = ref<string>(userCronConfigs.value[0]?.task_type || '')
const selectedCronConfig = computed(() =>
  userCronConfigs.value.find(c => c.task_type === selectedCronType.value)
)
const cronPermissions = computed(() => selectedCronConfig.value?.user_permissions)

// User's default push email for cron tasks
const cronDefaultEmail = ref('zhangming@example.com')

const cronTaskTypeLabels: Record<string, string> = {
  batch_audit: '批量审核',
  daily_report: '日报推送',
  weekly_report: '周报推送',
}

const cronSection = ref('push')

// ===== Archive review personal settings =====
const userArchiveConfigs = ref<ArchiveReviewConfig[]>(
  JSON.parse(JSON.stringify(mockArchiveReviewConfigs))
)
const selectedArchiveId = ref<string>(userArchiveConfigs.value[0]?.id || '')
const selectedArchiveConfig = computed(() =>
  userArchiveConfigs.value.find(c => c.id === selectedArchiveId.value)
)
const archivePermissions = computed(() => selectedArchiveConfig.value?.user_permissions)
const archiveSection = ref('fields')

// User's custom archive rules
const userArchiveCustomRules = ref<Record<string, { id: string; content: string; enabled: boolean }[]>>({
  'ARC-001': [{ id: 'UACR-001', content: '付款条件须与公司标准一致', enabled: true }],
  'ARC-002': [],
  'ARC-003': [],
  'ARC-004': [{ id: 'UACR-002', content: 'HR总监审批须在用人部门确认之后', enabled: true }],
})

// User's custom flow rules
const userArchiveFlowRules = ref<Record<string, { id: string; content: string; enabled: boolean }[]>>({
  'ARC-001': [],
  'ARC-002': [],
  'ARC-003': [],
  'ARC-004': [{ id: 'UAFR-001', content: '入职审批须在招聘计划审批之后', enabled: true }],
})

const newArchiveRuleContent = ref('')
const newArchiveFlowRuleContent = ref('')

const addArchiveCustomRule = () => {
  if (!newArchiveRuleContent.value.trim() || !selectedArchiveConfig.value) return
  const pid = selectedArchiveConfig.value.id
  if (!userArchiveCustomRules.value[pid]) userArchiveCustomRules.value[pid] = []
  userArchiveCustomRules.value[pid].push({
    id: `UACR-${Date.now()}`,
    content: newArchiveRuleContent.value.trim(),
    enabled: true,
  })
  newArchiveRuleContent.value = ''
  message.success('自定义复核规则已添加')
}

const removeArchiveCustomRule = (ruleId: string) => {
  if (!selectedArchiveConfig.value) return
  const pid = selectedArchiveConfig.value.id
  userArchiveCustomRules.value[pid] = (userArchiveCustomRules.value[pid] || []).filter(r => r.id !== ruleId)
  message.success('已删除')
}

const currentArchiveCustomRules = computed(() =>
  userArchiveCustomRules.value[selectedArchiveConfig.value?.id || ''] || []
)

const addArchiveFlowRule = () => {
  if (!newArchiveFlowRuleContent.value.trim() || !selectedArchiveConfig.value) return
  const pid = selectedArchiveConfig.value.id
  if (!userArchiveFlowRules.value[pid]) userArchiveFlowRules.value[pid] = []
  userArchiveFlowRules.value[pid].push({
    id: `UAFR-${Date.now()}`,
    content: newArchiveFlowRuleContent.value.trim(),
    enabled: true,
  })
  newArchiveFlowRuleContent.value = ''
  message.success('自定义审批流规则已添加')
}

const removeArchiveFlowRule = (ruleId: string) => {
  if (!selectedArchiveConfig.value) return
  const pid = selectedArchiveConfig.value.id
  userArchiveFlowRules.value[pid] = (userArchiveFlowRules.value[pid] || []).filter(r => r.id !== ruleId)
  message.success('已删除')
}

const currentArchiveFlowRules = computed(() =>
  userArchiveFlowRules.value[selectedArchiveConfig.value?.id || ''] || []
)

const toggleArchiveField = (field: ProcessField) => {
  if (!selectedArchiveConfig.value || !archivePermissions.value?.allow_custom_fields) return
  if (selectedArchiveConfig.value.field_mode === 'all') return
  field.selected = !field.selected
}
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
          { key: 'profile', label: '基本信息', icon: UserOutlined },
          { key: 'workbench', label: '审核工作台', icon: DashboardOutlined },
          { key: 'cron', label: '定时任务', icon: ClockCircleOutlined },
          { key: 'archive', label: '归档复盘', icon: FolderOpenOutlined },
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

      <!-- Role & Permissions card -->
      <div class="settings-card" style="margin-top: 20px;">
        <h4 class="perm-card-title">
          <SafetyCertificateOutlined style="color: var(--color-primary);" />
          角色与权限
        </h4>
        <div class="perm-info-row">
          <span class="perm-info-label">当前角色</span>
          <span class="perm-role-badge">{{ currentOrgRole?.name || roleLabels[userRole] || '业务用户' }}</span>
        </div>
        <div v-if="currentOrgRole?.description" class="perm-info-row">
          <span class="perm-info-label">角色说明</span>
          <span class="perm-info-value">{{ currentOrgRole.description }}</span>
        </div>
        <div v-if="currentMember" class="perm-info-row">
          <span class="perm-info-label">所属部门</span>
          <span class="perm-info-value">{{ currentMember.department_name }}</span>
        </div>
        <div class="perm-pages-section">
          <span class="perm-info-label">可访问页面</span>
          <div class="perm-page-tags">
            <span
              v-for="p in (currentOrgRole?.page_permissions || ['/dashboard', '/cron', '/settings'])"
              :key="p"
              class="perm-page-tag"
            >
              {{ allPageLabels[p] || p }}
            </span>
          </div>
        </div>
        <p class="perm-hint-text">权限由管理员在「组织人员」中配置，如需调整请联系租户管理员</p>
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
                { key: 'fields', label: '审核字段', icon: AppstoreOutlined },
                { key: 'rules', label: '审核规则', icon: NodeIndexOutlined },
                { key: 'ai', label: '审核尺度', icon: ControlOutlined },
              ]"
              :key="sec.key"
              class="section-nav-btn"
              :class="{ 'section-nav-btn--active': workbenchSection === sec.key }"
              @click="workbenchSection = sec.key"
            >
              <component :is="sec.icon" />
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

    <!-- Cron personal settings tab -->
    <div v-if="activeTab === 'cron'" class="tab-content">
      <div class="settings-card" style="max-width: 700px; margin-bottom: 20px;">
        <h4 class="config-section-title" style="margin-bottom: 12px;">默认推送邮箱</h4>
        <p class="config-section-desc">所有定时任务的推送结果将发送至此邮箱</p>
        <a-input v-model:value="cronDefaultEmail" placeholder="输入默认推送邮箱，多个邮箱使用英文逗号分隔" size="large">
          <template #prefix><MailOutlined class="input-icon" /></template>
        </a-input>
        <p class="config-section-desc" style="margin-top: 4px; margin-bottom: 0;">多个邮箱请使用英文逗号（,）分隔</p>
      </div>

      <div class="workbench-layout">
        <!-- Left: cron task type list -->
        <div class="process-list-panel">
          <div class="process-list-header">
            <ClockCircleOutlined />
            <span>定时任务类型</span>
          </div>
          <div
            v-for="cfg in userCronConfigs"
            :key="cfg.task_type"
            class="process-list-item"
            :class="{ 'process-list-item--active': selectedCronType === cfg.task_type }"
            @click="selectedCronType = cfg.task_type"
          >
            <div class="process-list-item-name">{{ cfg.label }}</div>
            <div class="process-list-item-path">
              {{ cfg.enabled ? '已启用' : '已禁用' }}
            </div>
          </div>
        </div>

        <!-- Right: cron config detail -->
        <div v-if="selectedCronConfig" class="process-config-panel">
          <h3 class="config-title">{{ selectedCronConfig.label }} - 个人配置</h3>
          <p class="config-subtitle">管理员允许的配置项</p>

          <!-- Sub-section nav (tab style like workbench) -->
          <div class="section-nav">
            <button
              v-for="sec in [
                { key: 'push', label: '推送设置', icon: SendOutlined },
                { key: 'template', label: '内容模板', icon: FileTextOutlined },
                { key: 'ai', label: 'AI 配置', icon: RobotOutlined },
              ]"
              :key="sec.key"
              class="section-nav-btn"
              :class="{ 'section-nav-btn--active': cronSection === sec.key }"
              @click="cronSection = sec.key"
            >
              <component :is="sec.icon" />
              {{ sec.label }}
            </button>
          </div>

          <!-- ===== Push settings section ===== -->
          <div v-if="cronSection === 'push'" class="config-section">
            <!-- Email override -->
            <div v-if="cronPermissions?.allow_modify_email">
              <div class="section-header-row">
                <h4 class="config-section-title">推送邮箱</h4>
              </div>
              <p class="config-section-desc">为该任务类型单独设置推送邮箱（留空则使用默认邮箱）</p>
              <a-input placeholder="留空使用默认邮箱" size="large">
                <template #prefix><MailOutlined class="input-icon" /></template>
              </a-input>
            </div>
            <div v-else>
              <div class="section-header-row">
                <h4 class="config-section-title">推送邮箱</h4>
                <span class="locked-tag"><LockOutlined /> 管理员已锁定</span>
              </div>
              <p class="config-section-desc">使用默认推送邮箱：{{ cronDefaultEmail }}</p>
            </div>

            <!-- Schedule -->
            <div style="margin-top: 20px;">
              <div v-if="cronPermissions?.allow_modify_schedule">
                <div class="section-header-row">
                  <h4 class="config-section-title">执行计划</h4>
                </div>
                <p class="config-section-desc">您可以在定时任务中心调整该任务的执行时间</p>
              </div>
              <div v-else>
                <div class="section-header-row">
                  <h4 class="config-section-title">执行计划</h4>
                  <span class="locked-tag"><LockOutlined /> 管理员已锁定</span>
                </div>
                <p class="config-section-desc">执行计划由管理员统一配置</p>
              </div>
            </div>
          </div>

          <!-- ===== Content template section ===== -->
          <div v-if="cronSection === 'template'" class="config-section">
            <div v-if="cronPermissions?.allow_modify_template">
              <div class="section-header-row">
                <h4 class="config-section-title">内容模板</h4>
              </div>
              <p class="config-section-desc">自定义推送内容的模板结构，支持变量占位符</p>

              <div style="display: flex; flex-direction: column; gap: 14px; margin-top: 8px;">
                <div>
                  <label class="template-label">邮件主题</label>
                  <a-input v-model:value="selectedCronConfig.content_template.subject" size="large" />
                </div>
                <div>
                  <label class="template-label">头部内容</label>
                  <a-input v-model:value="selectedCronConfig.content_template.header" size="large" />
                </div>
                <div>
                  <label class="template-label">正文模板</label>
                  <a-textarea v-model:value="selectedCronConfig.content_template.body_template" :rows="3" />
                </div>
                <div>
                  <label class="template-label">底部内容</label>
                  <a-input v-model:value="selectedCronConfig.content_template.footer" size="large" />
                </div>
              </div>

              <div style="margin-top: 16px;">
                <label class="template-label" style="margin-bottom: 8px; display: block;">包含内容模块</label>
                <div class="rule-config-list">
                  <div class="rule-config-item">
                    <span class="rule-config-text">AI 智能摘要</span>
                    <a-switch v-model:checked="selectedCronConfig.content_template.include_ai_summary" size="small" />
                  </div>
                  <div class="rule-config-item">
                    <span class="rule-config-text">统计数据</span>
                    <a-switch v-model:checked="selectedCronConfig.content_template.include_statistics" size="small" />
                  </div>
                  <div class="rule-config-item">
                    <span class="rule-config-text">明细列表</span>
                    <a-switch v-model:checked="selectedCronConfig.content_template.include_detail_list" size="small" />
                  </div>
                </div>
              </div>
            </div>
            <div v-else>
              <div class="section-header-row">
                <h4 class="config-section-title">内容模板</h4>
                <span class="locked-tag"><LockOutlined /> 管理员已锁定</span>
              </div>
              <p class="config-section-desc">内容模板由管理员统一配置，当前内容格式：<span style="font-weight: 600;">{{ selectedCronConfig.push_format === 'html' ? 'HTML 邮件' : selectedCronConfig.push_format === 'markdown' ? 'Markdown' : '纯文本' }}</span></p>
              <div class="rule-config-list" style="margin-top: 8px;">
                <div class="rule-config-item">
                  <span class="rule-config-text">AI 智能摘要</span>
                  <span :style="{ color: selectedCronConfig.content_template.include_ai_summary ? 'var(--color-success)' : 'var(--color-text-tertiary)' }">{{ selectedCronConfig.content_template.include_ai_summary ? '已包含' : '未包含' }}</span>
                </div>
                <div class="rule-config-item">
                  <span class="rule-config-text">统计数据</span>
                  <span :style="{ color: selectedCronConfig.content_template.include_statistics ? 'var(--color-success)' : 'var(--color-text-tertiary)' }">{{ selectedCronConfig.content_template.include_statistics ? '已包含' : '未包含' }}</span>
                </div>
                <div class="rule-config-item">
                  <span class="rule-config-text">明细列表</span>
                  <span :style="{ color: selectedCronConfig.content_template.include_detail_list ? 'var(--color-success)' : 'var(--color-text-tertiary)' }">{{ selectedCronConfig.content_template.include_detail_list ? '已包含' : '未包含' }}</span>
                </div>
              </div>
            </div>
          </div>

          <!-- ===== AI config section ===== -->
          <div v-if="cronSection === 'ai'" class="config-section">
            <div class="section-header-row">
              <h4 class="config-section-title">AI 模型</h4>
            </div>
            <p class="config-section-desc">
              当前模型：<span style="font-weight: 600;">{{ selectedCronConfig.ai_config.model_name }}</span>
              （{{ selectedCronConfig.ai_config.ai_provider }}）— 由管理员配置
            </p>

            <div style="margin-top: 16px;">
              <div class="section-header-row">
                <h4 class="config-section-title">AI 提示词</h4>
                <span v-if="!cronPermissions?.allow_modify_prompt" class="locked-tag">
                  <LockOutlined /> 不可见
                </span>
              </div>
              <p class="config-section-desc">
                {{ cronPermissions?.allow_modify_prompt ? '您可以查看和修改该任务的 AI 提示词' : '提示词内容由管理员配置，用户不可见' }}
              </p>
              <a-textarea
                v-if="cronPermissions?.allow_modify_prompt"
                v-model:value="selectedCronConfig.ai_config.system_prompt"
                :rows="4"
                placeholder="AI 提示词..."
              />
            </div>
          </div>

          <div class="settings-actions">
            <a-button type="primary" size="large" :loading="saving" @click="handleSave">
              <SaveOutlined /> 保存配置
            </a-button>
          </div>
        </div>

        <div v-else class="process-config-empty">
          <a-empty description="请选择左侧任务类型查看配置" />
        </div>
      </div>
    </div>

    <!-- Archive review personal settings tab -->
    <div v-if="activeTab === 'archive'" class="tab-content">
      <div class="workbench-layout">
        <!-- Left: process list -->
        <div class="process-list-panel">
          <div class="process-list-header">
            <SafetyCertificateOutlined />
            <span>复核流程</span>
          </div>
          <div
            v-for="cfg in userArchiveConfigs"
            :key="cfg.id"
            class="process-list-item"
            :class="{ 'process-list-item--active': selectedArchiveId === cfg.id }"
            @click="selectedArchiveId = cfg.id"
          >
            <div class="process-list-item-name">{{ cfg.process_type }}</div>
            <div class="process-list-item-path">{{ cfg.flow_path }}</div>
          </div>
        </div>

        <!-- Right: config detail -->
        <div v-if="selectedArchiveConfig" class="process-config-panel">
          <h3 class="config-title">{{ selectedArchiveConfig.process_type }} - 个人复核配置</h3>
          <p class="config-subtitle">流程路径：{{ selectedArchiveConfig.flow_path }}</p>

          <!-- Sub-section nav -->
          <div class="section-nav">
            <button
              v-for="sec in [
                { key: 'fields', label: '复核字段', icon: AppstoreOutlined },
                { key: 'rules', label: '复核规则', icon: AuditOutlined },
                { key: 'flow_rules', label: '审批流规则', icon: NodeIndexOutlined },
                { key: 'ai', label: '复核尺度', icon: ControlOutlined },
              ]"
              :key="sec.key"
              class="section-nav-btn"
              :class="{ 'section-nav-btn--active': archiveSection === sec.key }"
              @click="archiveSection = sec.key"
            >
              <component :is="sec.icon" />
              {{ sec.label }}
            </button>
          </div>

          <!-- ===== Fields section ===== -->
          <div v-if="archiveSection === 'fields'" class="config-section">
            <div class="section-header-row">
              <h4 class="config-section-title">复核字段</h4>
              <span v-if="!archivePermissions?.allow_custom_fields" class="locked-tag">
                <LockOutlined /> 管理员已锁定
              </span>
            </div>
            <p class="config-section-desc">
              {{ selectedArchiveConfig.field_mode === 'all' ? '当前为全部字段模式' : '以下为参与归档复核的字段配置' }}
              <template v-if="archivePermissions?.allow_custom_fields && selectedArchiveConfig.field_mode === 'selected'">
                ，您可以切换字段的选中状态
              </template>
            </p>

            <div class="field-grid">
              <div
                v-for="field in selectedArchiveConfig.fields"
                :key="field.field_key"
                class="field-card"
                :class="{
                  'field-card--selected': field.selected || selectedArchiveConfig.field_mode === 'all',
                  'field-card--readonly': !archivePermissions?.allow_custom_fields || selectedArchiveConfig.field_mode === 'all',
                }"
                @click="toggleArchiveField(field)"
              >
                <div class="field-card-check">
                  <CheckOutlined v-if="field.selected || selectedArchiveConfig.field_mode === 'all'" />
                </div>
                <div class="field-card-info">
                  <div class="field-card-name">{{ field.field_name }}</div>
                  <span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span>
                </div>
              </div>
            </div>
          </div>

          <!-- ===== Rules section ===== -->
          <div v-if="archiveSection === 'rules'" class="config-section">
            <!-- System rules -->
            <div class="section-header-row">
              <h4 class="config-section-title">通用复核规则（租户配置）</h4>
            </div>
            <div class="rule-config-list">
              <div v-for="rule in selectedArchiveConfig.rules" :key="rule.id" class="rule-config-item">
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

            <!-- Custom rules -->
            <div class="section-header-row" style="margin-top: 20px;">
              <h4 class="config-section-title">个人自定义复核规则</h4>
              <span v-if="!archivePermissions?.allow_custom_rules" class="locked-tag">
                <LockOutlined /> 管理员已锁定
              </span>
            </div>
            <p class="config-section-desc">
              {{ archivePermissions?.allow_custom_rules ? '您可以为此流程添加个人复核规则' : '当前流程不允许添加个人规则' }}
            </p>

            <div class="rule-config-list" v-if="currentArchiveCustomRules.length > 0">
              <div v-for="rule in currentArchiveCustomRules" :key="rule.id" class="rule-config-item">
                <div class="rule-config-content">
                  <span class="rule-config-text">{{ rule.content }}</span>
                  <span class="rule-scope-tag rule-scope-tag--custom">个人</span>
                </div>
                <div class="rule-config-actions">
                  <a-switch v-model:checked="rule.enabled" size="small" />
                  <a-popconfirm v-if="archivePermissions?.allow_custom_rules" title="确认删除？" @confirm="removeArchiveCustomRule(rule.id)">
                    <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
                  </a-popconfirm>
                </div>
              </div>
            </div>

            <div v-if="archivePermissions?.allow_custom_rules" class="add-rule-row">
              <a-input
                v-model:value="newArchiveRuleContent"
                placeholder="输入自定义复核规则内容..."
                @pressEnter="addArchiveCustomRule"
              />
              <a-button type="primary" :disabled="!newArchiveRuleContent.trim()" @click="addArchiveCustomRule">
                <PlusOutlined /> 添加
              </a-button>
            </div>
          </div>

          <!-- ===== Flow rules section ===== -->
          <div v-if="archiveSection === 'flow_rules'" class="config-section">
            <!-- System flow rules -->
            <div class="section-header-row">
              <h4 class="config-section-title">通用审批流规则（租户配置）</h4>
            </div>
            <p class="config-section-desc">审批流程合规性校验规则，如审批链完整性、节点顺序等</p>
            <div class="rule-config-list">
              <div v-for="rule in selectedArchiveConfig.flow_rules" :key="rule.id" class="rule-config-item">
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

            <!-- Custom flow rules -->
            <div class="section-header-row" style="margin-top: 20px;">
              <h4 class="config-section-title">个人自定义审批流规则</h4>
              <span v-if="!archivePermissions?.allow_custom_flow_rules" class="locked-tag">
                <LockOutlined /> 管理员已锁定
              </span>
            </div>
            <p class="config-section-desc">
              {{ archivePermissions?.allow_custom_flow_rules ? '您可以为此流程添加个人审批流合规规则' : '当前流程不允许添加个人审批流规则' }}
            </p>

            <div class="rule-config-list" v-if="currentArchiveFlowRules.length > 0">
              <div v-for="rule in currentArchiveFlowRules" :key="rule.id" class="rule-config-item">
                <div class="rule-config-content">
                  <span class="rule-config-text">{{ rule.content }}</span>
                  <span class="rule-scope-tag rule-scope-tag--custom">个人</span>
                </div>
                <div class="rule-config-actions">
                  <a-switch v-model:checked="rule.enabled" size="small" />
                  <a-popconfirm v-if="archivePermissions?.allow_custom_flow_rules" title="确认删除？" @confirm="removeArchiveFlowRule(rule.id)">
                    <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
                  </a-popconfirm>
                </div>
              </div>
            </div>

            <div v-if="archivePermissions?.allow_custom_flow_rules" class="add-rule-row">
              <a-input
                v-model:value="newArchiveFlowRuleContent"
                placeholder="输入自定义审批流规则内容..."
                @pressEnter="addArchiveFlowRule"
              />
              <a-button type="primary" :disabled="!newArchiveFlowRuleContent.trim()" @click="addArchiveFlowRule">
                <PlusOutlined /> 添加
              </a-button>
            </div>
          </div>

          <!-- ===== AI strictness section ===== -->
          <div v-if="archiveSection === 'ai'" class="config-section">
            <div class="section-header-row">
              <h4 class="config-section-title">复核尺度</h4>
              <span v-if="!archivePermissions?.allow_modify_strictness" class="locked-tag">
                <LockOutlined /> 管理员已锁定
              </span>
            </div>
            <p class="config-section-desc">
              当前 AI 模型：{{ selectedArchiveConfig.ai_config.model_name }}（{{ selectedArchiveConfig.ai_config.ai_provider }}）
            </p>
            <div class="strictness-options">
              <div
                v-for="opt in strictnessOptions"
                :key="opt.value"
                class="strictness-option"
                :class="{
                  'strictness-option--active': selectedArchiveConfig.ai_config.audit_strictness === opt.value,
                  'strictness-option--disabled': !archivePermissions?.allow_modify_strictness,
                }"
                @click="archivePermissions?.allow_modify_strictness && (selectedArchiveConfig.ai_config.audit_strictness = opt.value as any)"
              >
                <div class="strictness-option-radio" />
                <div>
                  <div class="strictness-option-label">{{ opt.label }}</div>
                  <div class="strictness-option-desc">{{ opt.desc }}</div>
                </div>
              </div>
            </div>

            <div style="margin-top: 20px;">
              <h4 class="config-section-title">知识库模式</h4>
              <p class="config-section-desc">
                当前模式：<span style="font-weight: 600;">
                  {{ selectedArchiveConfig.kb_mode === 'rules_only' ? '仅规则库' : selectedArchiveConfig.kb_mode === 'rag_only' ? '仅制度库' : '混合模式' }}
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
          <a-empty description="请选择左侧流程查看复核配置" />
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
  display: flex; align-items: center; gap: 6px;
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
  display: flex; align-items: center; gap: 5px;
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

/* Template label */
.template-label {
  font-size: 13px; font-weight: 600; color: var(--color-text-primary);
  margin-bottom: 4px; display: block;
}

@media (max-width: 768px) {
  .form-row { grid-template-columns: 1fr; }
  .workbench-layout { grid-template-columns: 1fr; }
  .field-grid { grid-template-columns: 1fr 1fr; }
}

/* Permission card in profile */
.perm-card-title {
  font-size: 15px; font-weight: 600; color: var(--color-text-primary);
  margin: 0 0 16px; display: flex; align-items: center; gap: 8px;
}
.perm-info-row {
  display: flex; align-items: center; gap: 12px; margin-bottom: 10px;
}
.perm-info-label {
  font-size: 13px; font-weight: 500; color: var(--color-text-secondary); min-width: 72px;
}
.perm-info-value { font-size: 13px; color: var(--color-text-primary); }
.perm-role-badge {
  font-size: 12px; font-weight: 600; padding: 2px 12px; border-radius: var(--radius-full);
  background: var(--color-primary-bg); color: var(--color-primary);
}
.perm-pages-section {
  display: flex; align-items: flex-start; gap: 12px; margin-top: 12px;
}
.perm-page-tags { display: flex; flex-wrap: wrap; gap: 6px; }
.perm-page-tag {
  font-size: 11px; padding: 2px 10px; border-radius: var(--radius-sm);
  background: var(--color-bg-hover); color: var(--color-text-secondary); font-weight: 500;
}
.perm-hint-text {
  font-size: 12px; color: var(--color-text-tertiary); margin: 14px 0 0;
  padding-top: 12px; border-top: 1px solid var(--color-border-light);
}
</style>
