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
  ControlOutlined,
  AuditOutlined,
  PieChartOutlined,
  EyeOutlined,
  EyeInvisibleOutlined,
  GlobalOutlined,
  KeyOutlined,
  SearchOutlined,
  SwapRightOutlined,
  CloseOutlined,
  LoadingOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type { ProcessAuditConfig, ProcessField, AuditRule, ArchiveReviewConfig, OverviewWidgetId } from '~/composables/useMockData'
import { OVERVIEW_WIDGETS } from '~/composables/useMockData'
import type { Locale } from '~/composables/useI18n'

definePageMeta({
  middleware: 'auth',
  layout: 'default',
})

const { userRole, userPermissions, currentUser } = useAuth()
const { mockProcessAuditConfigs, mockArchiveReviewConfigs, mockOrgRoles, mockOrgMembers, mockUserDashboardPrefs, mockUserSecurityInfo, mockUserLocalePrefs } = useMockData()
const { t, locale, setLocale, availableLocales } = useI18n()

const activeTab = ref('profile')

// ===== Language & Region tab =====
const userDateFormat = ref('YYYY-MM-DD')
onMounted(() => {
  const uname = currentUser.value?.username || ''
  const prefs = mockUserLocalePrefs[uname]
  if (prefs) {
    userDateFormat.value = prefs.dateFormat
  }
})

const handleLocaleChange = (newLocale: Locale) => {
  setLocale(newLocale)
  message.success(t('settings.language.switchSuccess'))
}

const dateFormatOptions = [
  { value: 'YYYY-MM-DD', label: 'YYYY-MM-DD (2026-02-19)' },
  { value: 'MM/DD/YYYY', label: 'MM/DD/YYYY (02/19/2026)' },
  { value: 'DD/MM/YYYY', label: 'DD/MM/YYYY (19/02/2026)' },
]

// ===== Security / Password tab =====
const passwordForm = ref({
  currentPassword: '',
  newPassword: '',
  confirmPassword: '',
})
const passwordChanging = ref(false)
const showCurrentPassword = ref(false)
const showNewPassword = ref(false)

const passwordStrength = computed(() => {
  const pwd = passwordForm.value.newPassword
  if (!pwd) return null
  if (pwd.length < 6) return 'weak'
  const hasUpper = /[A-Z]/.test(pwd)
  const hasLower = /[a-z]/.test(pwd)
  const hasNum = /[0-9]/.test(pwd)
  const hasSpecial = /[!@#$%^&*(),.?":{}|<>]/.test(pwd)
  const score = [hasUpper, hasLower, hasNum, hasSpecial].filter(Boolean).length
  if (pwd.length >= 10 && score >= 3) return 'strong'
  if (pwd.length >= 6 && score >= 2) return 'medium'
  return 'weak'
})

const strengthConfig: Record<string, { color: string; percent: number }> = {
  weak: { color: 'var(--color-danger)', percent: 33 },
  medium: { color: 'var(--color-warning)', percent: 66 },
  strong: { color: 'var(--color-success)', percent: 100 },
}

const securityInfo = computed(() => {
  const uname = currentUser.value?.username || ''
  return mockUserSecurityInfo[uname] || { password_last_changed: '-', login_history: [] }
})

const handleChangePassword = async () => {
  const { currentPassword, newPassword, confirmPassword } = passwordForm.value
  if (!currentPassword || !newPassword || !confirmPassword) {
    message.error(t('settings.security.changeError.empty')); return
  }
  if (newPassword.length < 6) {
    message.error(t('settings.security.changeError.tooShort')); return
  }
  if (newPassword !== confirmPassword) {
    message.error(t('settings.security.changeError.mismatch')); return
  }
  // Mock: verify current password
  const { MOCK_USERS } = useAuth()
  const user = MOCK_USERS.find(u => u.username === currentUser.value?.username)
  if (user && user.password !== currentPassword) {
    message.error(t('settings.security.changeError.wrongCurrent')); return
  }
  passwordChanging.value = true
  await new Promise(r => setTimeout(r, 1000))
  passwordChanging.value = false
  passwordForm.value = { currentPassword: '', newPassword: '', confirmPassword: '' }
  message.success(t('settings.security.changeSuccess'))
}



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

const getPageLabel = (path: string) => t(`page.${path}`, path)

const profile = ref({
  nickname: '张明',
  email: 'zhangming@example.com',
  phone: '138****8888',
  department: '研发部',
  position: '高级工程师',
})

const getRoleLabel = (role: string) => {
  const map: Record<string, string> = { business: 'role.business', tenant_admin: 'role.tenantAdmin', system_admin: 'role.systemAdmin' }
  return t(map[role] || 'role.business')
}

// ===== Audit workbench tab =====
// Deep clone tenant configs as user's working copy
const userProcessConfigs = ref<ProcessAuditConfig[]>(
  JSON.parse(JSON.stringify(mockProcessAuditConfigs))
)

// User's custom rules per process (separate from tenant rules)
const userCustomRules = ref<Record<string, { id: string; content: string; enabled: boolean; related_flow: boolean }[]>>({
  'PAC-001': [{ id: 'UCR-001', content: '供应商必须在合格名录中', enabled: true, related_flow: false }],
  'PAC-002': [],
  'PAC-003': [{ id: 'UCR-002', content: '合同期限超过2年需额外审批', enabled: true, related_flow: true }],
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
const fieldTypeLabels = computed<Record<string, string>>(() => ({
  text: t('field.type.text'),
  number: t('field.type.number'),
  date: t('field.type.date'),
  money: t('field.type.money'),
  select: t('field.type.select'),
  user: t('field.type.user'),
  dept: t('field.type.dept'),
  rich_text: t('field.type.richText'),
}))

// Scope config
const scopeConfig = computed<Record<string, { label: string; color?: string }>>(() => ({
  mandatory: { label: t('rule.scope.mandatory') },
  default_on: { label: t('rule.scope.defaultOn') },
  default_off: { label: t('rule.scope.defaultOff') },
}))

// Strictness
const strictnessOptions = computed(() => [
  { value: 'strict', label: t('settings.workbench.strict'), desc: t('settings.workbench.strictDesc') },
  { value: 'standard', label: t('settings.workbench.standard'), desc: t('settings.workbench.standardDesc') },
  { value: 'loose', label: t('settings.workbench.loose'), desc: t('settings.workbench.looseDesc') },
])

// Toggle user field override
const toggleUserField = (field: ProcessField) => {
  if (!selectedConfig.value || !permissions.value?.allow_custom_fields) return
  if (selectedConfig.value.field_mode === 'all') return
  field.selected = !field.selected
}

// ===== Field picker modal (settings) =====
const showFieldPicker = ref(false)
const fieldSearchQuery = ref('')

interface SettingsPickerField {
  field_key: string; field_name: string; field_type: string; selected: boolean
  source: string; sourceLabel: string
}
interface SettingsFieldGroup {
  source: string; sourceLabel: string; fields: SettingsPickerField[]
}

const settingsGroupedFields = computed<SettingsFieldGroup[]>(() => {
  if (!selectedConfig.value) return []
  const groups: SettingsFieldGroup[] = []
  const mainFields = selectedConfig.value.main_fields || selectedConfig.value.fields
  groups.push({
    source: 'main',
    sourceLabel: t('settings.workbench.mainTableFields'),
    fields: mainFields.map(f => ({ ...f, source: 'main', sourceLabel: t('settings.workbench.mainTableFields') })),
  })
  if (selectedConfig.value.detail_tables) {
    selectedConfig.value.detail_tables.forEach((dt, idx) => {
      groups.push({
        source: dt.table_name,
        sourceLabel: `${t('settings.workbench.detailTableLabel')} ${idx + 1}`,
        fields: dt.fields.map(f => ({ ...f, source: dt.table_name, sourceLabel: `${t('settings.workbench.detailTableLabel')} ${idx + 1}` })),
      })
    })
  }
  return groups
})

const settingsAllFields = computed<SettingsPickerField[]>(() =>
  settingsGroupedFields.value.flatMap(g => g.fields)
)

const settingsSelectedCount = computed(() =>
  settingsAllFields.value.filter(f => f.selected).length
)

const settingsGroupedUnselected = computed<SettingsFieldGroup[]>(() => {
  const q = fieldSearchQuery.value.toLowerCase().trim()
  return settingsGroupedFields.value
    .map(g => ({
      ...g,
      fields: g.fields.filter(f => {
        if (f.selected) return false
        if (!q) return true
        return f.field_name.toLowerCase().includes(q) || f.field_key.toLowerCase().includes(q)
      }),
    }))
    .filter(g => g.fields.length > 0)
})

const settingsGroupedSelected = computed<SettingsFieldGroup[]>(() =>
  settingsGroupedFields.value
    .map(g => ({ ...g, fields: g.fields.filter(f => f.selected) }))
    .filter(g => g.fields.length > 0)
)

const openSettingsFieldPicker = () => {
  fieldSearchQuery.value = ''
  showFieldPicker.value = true
}

const settingsPickField = (field: { field_key: string; source: string }) => {
  if (!selectedConfig.value) return
  const mainFields = selectedConfig.value.main_fields || selectedConfig.value.fields
  const mf = mainFields.find(f => f.field_key === field.field_key)
  if (mf && field.source === 'main') { mf.selected = true; return }
  if (selectedConfig.value.detail_tables) {
    for (const dt of selectedConfig.value.detail_tables) {
      if (dt.table_name === field.source) {
        const df = dt.fields.find(f => f.field_key === field.field_key)
        if (df) { df.selected = true; return }
      }
    }
  }
}

const settingsUnpickField = (field: { field_key: string; source: string }) => {
  if (!selectedConfig.value) return
  const mainFields = selectedConfig.value.main_fields || selectedConfig.value.fields
  const mf = mainFields.find(f => f.field_key === field.field_key)
  if (mf && field.source === 'main') { mf.selected = false; return }
  if (selectedConfig.value.detail_tables) {
    for (const dt of selectedConfig.value.detail_tables) {
      if (dt.table_name === field.source) {
        const df = dt.fields.find(f => f.field_key === field.field_key)
        if (df) { df.selected = false; return }
      }
    }
  }
}

// Custom rules
const newRuleContent = ref('')
const newRuleRelatedFlow = ref(false)

const addCustomRule = () => {
  if (!newRuleContent.value.trim() || !selectedConfig.value) return
  const pid = selectedConfig.value.id
  if (!userCustomRules.value[pid]) userCustomRules.value[pid] = []
  userCustomRules.value[pid].push({
    id: `UCR-${Date.now()}`,
    content: newRuleContent.value.trim(),
    enabled: true,
    related_flow: newRuleRelatedFlow.value,
  })
  newRuleContent.value = ''
  newRuleRelatedFlow.value = false
  message.success(t('settings.workbench.ruleAdded'))
}

const removeCustomRule = (ruleId: string) => {
  if (!selectedConfig.value) return
  const pid = selectedConfig.value.id
  userCustomRules.value[pid] = (userCustomRules.value[pid] || []).filter(r => r.id !== ruleId)
  message.success(t('settings.workbench.deleted'))
}

const currentCustomRules = computed(() =>
  userCustomRules.value[selectedConfig.value?.id || ''] || []
)

const saving = ref(false)
const handleSave = async () => {
  saving.value = true
  await new Promise(r => setTimeout(r, 800))
  saving.value = false
  message.success(t('settings.profile.saveSuccess'))
}

// Active workbench sub-section
const workbenchSection = ref('fields')

// ===== Cron personal settings =====
const cronDefaultEmail = ref('zhangming@example.com')

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
  message.success(t('settings.archive.ruleAdded'))
}

const removeArchiveCustomRule = (ruleId: string) => {
  if (!selectedArchiveConfig.value) return
  const pid = selectedArchiveConfig.value.id
  userArchiveCustomRules.value[pid] = (userArchiveCustomRules.value[pid] || []).filter(r => r.id !== ruleId)
  message.success(t('settings.workbench.deleted'))
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
  message.success(t('settings.archive.flowRuleAdded'))
}

const removeArchiveFlowRule = (ruleId: string) => {
  if (!selectedArchiveConfig.value) return
  const pid = selectedArchiveConfig.value.id
  userArchiveFlowRules.value[pid] = (userArchiveFlowRules.value[pid] || []).filter(r => r.id !== ruleId)
  message.success(t('settings.workbench.deleted'))
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
        <h1 class="page-title">{{ t('settings.title') }}</h1>
        <p class="page-subtitle">{{ t('settings.subtitle') }}</p>
      </div>
    </div>

    <!-- Tab navigation -->
    <div class="tab-nav">
      <button
        v-for="tab in [
          { key: 'profile', label: t('settings.tab.profile'), icon: UserOutlined },
          { key: 'workbench', label: t('settings.tab.workbench'), icon: DashboardOutlined },
          { key: 'cron', label: t('settings.tab.cron'), icon: ClockCircleOutlined },
          { key: 'archive', label: t('settings.tab.archive'), icon: FolderOpenOutlined },
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
              <span class="role-badge">{{ getRoleLabel(userRole) }}</span>
            </div>
          </div>
        </div>

        <a-form layout="vertical" class="settings-form">
          <div class="form-row">
            <a-form-item :label="t('settings.profile.nickname')" class="form-col">
              <a-input v-model:value="profile.nickname" size="large" :placeholder="t('settings.profile.nicknamePlaceholder')">
                <template #prefix><UserOutlined class="input-icon" /></template>
              </a-input>
            </a-form-item>
            <a-form-item :label="t('settings.profile.email')" class="form-col">
              <a-input v-model:value="profile.email" size="large" :placeholder="t('settings.profile.emailPlaceholder')">
                <template #prefix><MailOutlined class="input-icon" /></template>
              </a-input>
            </a-form-item>
          </div>
          <div class="form-row">
            <a-form-item :label="t('settings.profile.phone')" class="form-col">
              <a-input v-model:value="profile.phone" size="large" :placeholder="t('settings.profile.phonePlaceholder')">
                <template #prefix><PhoneOutlined class="input-icon" /></template>
              </a-input>
            </a-form-item>
            <a-form-item :label="t('settings.profile.department')" class="form-col">
              <a-input v-model:value="profile.department" size="large" disabled />
            </a-form-item>
          </div>
          <a-form-item :label="t('settings.profile.position')">
            <a-input v-model:value="profile.position" size="large" disabled />
          </a-form-item>
        </a-form>

        <div class="settings-actions">
          <a-button type="primary" size="large" :disabled="saving" @click="handleSave">
            <LoadingOutlined v-if="saving" spin />
            <SaveOutlined v-else />
            {{ t('settings.profile.save') }}
          </a-button>
        </div>
      </div>

      <!-- Role & Permissions card -->
      <div class="settings-card" style="margin-top: 20px;">
        <h4 class="perm-card-title">
          <SafetyCertificateOutlined style="color: var(--color-primary);" />
          {{ t('settings.profile.roleAndPermissions') }}
        </h4>
        <div class="perm-info-row">
          <span class="perm-info-label">{{ t('settings.profile.currentRole') }}</span>
          <span class="perm-role-badge">{{ currentOrgRole?.name || getRoleLabel(userRole) }}</span>
        </div>
        <div v-if="currentOrgRole?.description" class="perm-info-row">
          <span class="perm-info-label">{{ t('settings.profile.roleDescription') }}</span>
          <span class="perm-info-value">{{ currentOrgRole.description }}</span>
        </div>
        <div v-if="currentMember" class="perm-info-row">
          <span class="perm-info-label">{{ t('settings.profile.belongDepartment') }}</span>
          <span class="perm-info-value">{{ currentMember.department_name }}</span>
        </div>
        <div class="perm-pages-section">
          <span class="perm-info-label">{{ t('settings.profile.accessiblePages') }}</span>
          <div class="perm-page-tags">
            <span
              v-for="p in (currentOrgRole?.page_permissions || ['/dashboard', '/cron', '/settings'])"
              :key="p"
              class="perm-page-tag"
            >
              {{ getPageLabel(p) }}
            </span>
          </div>
        </div>
        <p class="perm-hint-text">{{ t('settings.profile.permissionHint') }}</p>
      </div>

      <!-- Language setting (integrated into profile) -->
      <div class="settings-card" style="margin-top: 20px;">
        <h4 class="perm-card-title">
          <GlobalOutlined style="color: var(--color-primary);" />
          {{ t('settings.language.title') }}
        </h4>
        <p class="config-section-desc" style="margin-bottom: 16px;">{{ t('settings.language.subtitle') }}</p>
        <div class="language-options">
          <div
            v-for="loc in availableLocales"
            :key="loc.value"
            class="language-option"
            :class="{ 'language-option--active': locale === loc.value }"
            @click="handleLocaleChange(loc.value)"
          >
            <span class="language-flag">{{ loc.flag }}</span>
            <span class="language-label">{{ loc.label }}</span>
            <CheckOutlined v-if="locale === loc.value" class="language-check" />
          </div>
        </div>
      </div>

      <!-- Security / Password (integrated into profile) -->
      <div class="settings-card" style="margin-top: 20px;">
        <h4 class="perm-card-title">
          <LockOutlined style="color: var(--color-primary);" />
          {{ t('settings.security.title') }}
        </h4>
        <p class="config-section-desc" style="margin-bottom: 16px;">{{ t('settings.security.subtitle') }}</p>

        <a-form layout="vertical" class="settings-form" style="max-width: 480px;">
          <a-form-item :label="t('settings.security.currentPassword')">
            <a-input-password
              v-model:value="passwordForm.currentPassword"
              size="large"
              :placeholder="t('settings.security.currentPasswordPlaceholder')"
              :visibility-toggle="true"
            />
          </a-form-item>
          <a-form-item :label="t('settings.security.newPassword')">
            <a-input-password
              v-model:value="passwordForm.newPassword"
              size="large"
              :placeholder="t('settings.security.newPasswordPlaceholder')"
              :visibility-toggle="true"
            />
            <div v-if="passwordStrength" class="password-strength">
              <div class="strength-bar">
                <div class="strength-fill" :style="{ width: strengthConfig[passwordStrength].percent + '%', background: strengthConfig[passwordStrength].color }" />
              </div>
              <span class="strength-label" :style="{ color: strengthConfig[passwordStrength].color }">
                {{ t('settings.security.passwordStrength') }}: {{ t(`settings.security.strength.${passwordStrength}`) }}
              </span>
            </div>
          </a-form-item>
          <a-form-item :label="t('settings.security.confirmPassword')">
            <a-input-password
              v-model:value="passwordForm.confirmPassword"
              size="large"
              :placeholder="t('settings.security.confirmPasswordPlaceholder')"
              :visibility-toggle="true"
            />
          </a-form-item>
          <a-button type="primary" size="large" :disabled="passwordChanging" @click="handleChangePassword">
            <LoadingOutlined v-if="passwordChanging" spin />
            <LockOutlined v-else />
            {{ t('settings.security.changePassword') }}
          </a-button>
        </a-form>

        <div class="settings-divider" />

        <div class="security-info">
          <div class="security-info-row">
            <span class="perm-info-label">{{ t('settings.security.lastChanged') }}</span>
            <span class="perm-info-value">{{ securityInfo.password_last_changed }}</span>
          </div>
        </div>

        <div v-if="securityInfo.login_history.length" class="login-history">
          <h4 class="config-section-title" style="margin-top: 24px;">{{ t('settings.security.loginHistory') }}</h4>
          <div class="login-history-list">
            <div v-for="(entry, idx) in securityInfo.login_history" :key="idx" class="login-history-item">
              <div class="login-history-time">{{ entry.time }}</div>
              <div class="login-history-details">
                <span>{{ entry.device }}</span>
                <span class="login-history-sep">·</span>
                <span>{{ entry.ip }}</span>
                <span class="login-history-sep">·</span>
                <span>{{ entry.location }}</span>
              </div>
            </div>
          </div>
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
            <div class="process-list-item-path">{{ proc.main_table_name || '待配置' }}</div>
          </div>
        </div>

        <!-- Right: config detail -->
        <div v-if="selectedConfig" class="process-config-panel">
          <h3 class="config-title">{{ selectedConfig.process_type }} - 个人审核配置</h3>
          <p class="config-subtitle">{{ selectedConfig.main_table_name || '待配置' }}</p>

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
                ，您可以通过弹框切换字段的选中状态
              </template>
            </p>

            <template v-if="selectedConfig.field_mode === 'selected'">
              <div class="field-picker-toolbar">
                <span class="field-count">已选 {{ settingsSelectedCount }} / {{ settingsAllFields.length }} 个字段</span>
                <a-button
                  v-if="permissions?.allow_custom_fields"
                  type="primary"
                  size="small"
                  @click="openSettingsFieldPicker"
                >
                  <AppstoreOutlined /> 选择字段
                </a-button>
              </div>

              <!-- Selected fields grouped by table -->
              <template v-if="settingsGroupedSelected.length">
                <div v-for="group in settingsGroupedSelected" :key="group.source" class="selected-field-group">
                  <div class="field-group-label">{{ group.sourceLabel }}</div>
                  <div class="selected-fields-display">
                    <div
                      v-for="field in group.fields"
                      :key="field.field_key + field.source"
                      class="selected-field-tag"
                    >
                      <span class="selected-field-name">{{ field.field_name }}</span>
                      <span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span>
                    </div>
                  </div>
                </div>
              </template>
              <div v-else class="field-empty-hint">
                暂未选择字段
              </div>
            </template>

            <template v-else>
              <div class="field-count" style="margin-top: 8px;">
                全部字段模式：所有主表及明细表字段均传输给 AI
              </div>
            </template>
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
                  <span v-if="(rule as any).related_flow" class="rule-flow-tag">
                    <NodeIndexOutlined /> {{ t('settings.workbench.relatedFlow') }}
                  </span>
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
                  <span v-if="rule.related_flow" class="rule-flow-tag">
                    <NodeIndexOutlined /> {{ t('settings.workbench.relatedFlow') }}
                  </span>
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
              <a-tooltip title="关联审批流：该规则需要审批流节点信息才能校验">
                <button
                  class="icon-btn"
                  :class="{ 'icon-btn--active': newRuleRelatedFlow }"
                  @click="newRuleRelatedFlow = !newRuleRelatedFlow"
                >
                  <NodeIndexOutlined />
                </button>
              </a-tooltip>
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
              {{ t('settings.workbench.strictnessDesc', '审核尺度影响 AI 建议倾向，由管理员或个人设置') }}
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
            <a-button type="primary" size="large" :disabled="saving" @click="handleSave">
              <LoadingOutlined v-if="saving" spin />
              <SaveOutlined v-else />
              保存配置
            </a-button>
          </div>
        </div>

        <div v-else class="process-config-empty">
          <a-empty description="请选择左侧流程查看配置" />
        </div>
      </div>
    </div>

    <!-- Settings field picker modal -->
    <a-modal
      v-model:open="showFieldPicker"
      title="选择字段"
      :width="720"
      :footer="null"
      @cancel="showFieldPicker = false"
    >
      <div class="field-picker-modal">
        <div class="field-picker-left">
          <div class="field-picker-panel-header">
            <span>可选字段</span>
          </div>
          <div class="field-picker-search">
            <a-input
              v-model:value="fieldSearchQuery"
              placeholder="搜索字段名称或字段键..."
              allow-clear
              size="small"
            >
              <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
            </a-input>
          </div>
          <div class="field-picker-list">
            <template v-for="group in settingsGroupedUnselected" :key="group.source">
              <div class="field-picker-group-label">{{ group.sourceLabel }}</div>
              <div
                v-for="field in group.fields"
                :key="field.field_key + field.source"
                class="field-picker-item"
                @click="settingsPickField(field)"
              >
                <div class="field-picker-item-info">
                  <div class="field-picker-item-name">{{ field.field_name }}</div>
                  <div class="field-picker-item-meta">
                    <span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span>
                    <span class="field-key">{{ field.field_key }}</span>
                  </div>
                </div>
                <SwapRightOutlined class="field-picker-arrow" />
              </div>
            </template>
            <div v-if="!settingsGroupedUnselected.length" class="field-picker-empty">
              {{ fieldSearchQuery ? '无匹配字段' : '所有字段已添加' }}
            </div>
          </div>
        </div>
        <div class="field-picker-right">
          <div class="field-picker-panel-header">
            <span>已选字段</span>
            <span class="field-picker-count">{{ settingsSelectedCount }}</span>
          </div>
          <div class="field-picker-list">
            <template v-for="group in settingsGroupedSelected" :key="group.source">
              <div class="field-picker-group-label">{{ group.sourceLabel }}</div>
              <div
                v-for="field in group.fields"
                :key="field.field_key + field.source"
                class="field-picker-item field-picker-item--selected"
              >
                <div class="field-picker-item-info">
                  <div class="field-picker-item-name">{{ field.field_name }}</div>
                  <div class="field-picker-item-meta">
                    <span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span>
                    <span class="field-key">{{ field.field_key }}</span>
                  </div>
                </div>
                <button class="field-picker-remove" @click="settingsUnpickField(field)">
                  <CloseOutlined />
                </button>
              </div>
            </template>
            <div v-if="!settingsGroupedSelected.length" class="field-picker-empty">
              暂未选择字段
            </div>
          </div>
        </div>
      </div>
    </a-modal>

    <!-- Cron personal settings tab -->
    <div v-if="activeTab === 'cron'" class="tab-content">
      <div class="settings-card" style="max-width: 700px;">
        <h4 class="config-section-title" style="margin-bottom: 12px;">默认推送邮箱</h4>
        <p class="config-section-desc">日报推送和周报推送的结果将发送至此邮箱，批量审核任务不涉及邮件推送</p>
        <a-input v-model:value="cronDefaultEmail" placeholder="输入默认推送邮箱，多个邮箱使用英文逗号分隔" size="large">
          <template #prefix><MailOutlined class="input-icon" /></template>
        </a-input>
        <p class="config-section-desc" style="margin-top: 4px; margin-bottom: 0;">多个邮箱请使用英文逗号（,）分隔</p>
        <div class="settings-actions" style="margin-top: 20px;">
          <a-button type="primary" size="large" :disabled="saving" @click="handleSave">
            <LoadingOutlined v-if="saving" spin />
            <SaveOutlined v-else />
            保存配置
          </a-button>
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
            <a-button type="primary" size="large" :disabled="saving" @click="handleSave">
              <LoadingOutlined v-if="saving" spin />
              <SaveOutlined v-else />
              保存配置
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
  display: flex; flex-direction: row; flex-wrap: nowrap; gap: 4px;
  background: var(--color-bg-hover); padding: 4px;
  border-radius: var(--radius-lg); margin-bottom: 24px; width: fit-content;
}
.tab-btn {
  display: inline-flex; align-items: center; gap: 6px; white-space: nowrap;
  padding: 8px 20px; border: none; background: transparent; border-radius: var(--radius-md);
  font-size: 14px; font-weight: 500; color: var(--color-text-secondary); cursor: pointer;
  transition: all var(--transition-fast); flex-shrink: 0;
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
  display: flex; flex-direction: row; flex-wrap: nowrap; gap: 4px;
  background: var(--color-bg-hover); padding: 3px;
  border-radius: var(--radius-md); margin-bottom: 20px; width: fit-content;
}
.section-nav-btn {
  display: inline-flex; align-items: center; gap: 5px; white-space: nowrap;
  padding: 6px 16px; border: none; background: transparent; border-radius: var(--radius-sm);
  font-size: 13px; font-weight: 500; color: var(--color-text-secondary); cursor: pointer;
  transition: all var(--transition-fast); flex-shrink: 0;
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
.icon-btn--active { border-color: var(--color-primary); color: var(--color-primary); background: var(--color-primary-bg); }

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
  .tab-nav {
    width: 100%;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
    scrollbar-width: none;
    flex-wrap: nowrap;
  }
  .tab-nav::-webkit-scrollbar { display: none; }
  .tab-btn { flex-shrink: 0; padding: 8px 14px; font-size: 13px; }
  .section-nav {
    width: 100%;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
    scrollbar-width: none;
    flex-wrap: nowrap;
  }
  .section-nav::-webkit-scrollbar { display: none; }
  .section-nav-btn { flex-shrink: 0; white-space: nowrap; }
  .settings-card { padding: 16px; }
  .process-config-panel { padding: 16px; }
  .strictness-options { gap: 6px; }
  .strictness-option { padding: 10px 12px; }
  .add-rule-row { flex-direction: column; }
  .add-rule-row .ant-btn { width: 100%; }
  .rule-config-item { flex-wrap: wrap; gap: 8px; padding: 8px 10px; }
  .perm-info-row { flex-direction: column; align-items: flex-start; gap: 4px; }
  .perm-pages-section { flex-direction: column; gap: 8px; }
}
@media (max-width: 480px) {
  .page-title { font-size: 20px; }
  .tab-btn { padding: 6px 10px; font-size: 12px; gap: 4px; }
  .field-grid { grid-template-columns: 1fr; }
  .profile-avatar-section { flex-direction: column; text-align: center; }
  .settings-card { padding: 14px; }
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

/* Dashboard widget prefs */
.dash-widget-list { display: flex; flex-direction: column; gap: 8px; }
.dash-widget-item {
  display: flex; align-items: center; gap: 14px;
  padding: 14px 16px; border-radius: var(--radius-md);
  border: 1px solid var(--color-border-light); background: var(--color-bg-page);
  cursor: pointer; transition: all var(--transition-fast);
}
.dash-widget-item:hover { border-color: var(--color-primary); }
.dash-widget-item--active { background: var(--color-primary-bg); border-color: var(--color-primary); }
.dash-widget-check {
  width: 28px; height: 28px; border-radius: var(--radius-sm);
  border: 2px solid var(--color-border); display: flex; align-items: center; justify-content: center;
  color: transparent; flex-shrink: 0; transition: all var(--transition-fast);
}
.dash-widget-item--active .dash-widget-check {
  background: var(--color-primary); border-color: var(--color-primary); color: #fff;
}
.dash-widget-info { flex: 1; min-width: 0; }
.dash-widget-name { font-size: 14px; font-weight: 500; color: var(--color-text-primary); }
.dash-widget-desc { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }
.dash-widget-perms { display: flex; gap: 4px; flex-shrink: 0; }
.dash-perm-tag {
  font-size: 11px; padding: 2px 8px; border-radius: var(--radius-full);
  background: var(--color-bg-hover); color: var(--color-text-tertiary);
}
/* Language & Region tab */
.language-options { display: flex; gap: 12px; flex-wrap: wrap; }
.language-option {
  display: flex; align-items: center; gap: 12px;
  padding: 16px 24px; border-radius: var(--radius-lg);
  border: 2px solid var(--color-border);
  cursor: pointer; transition: all 0.2s ease;
  min-width: 180px; position: relative;
}
.language-option:hover { border-color: var(--color-primary-light); background: var(--color-bg-hover); }
.language-option--active {
  border-color: var(--color-primary); background: var(--color-primary-bg);
  box-shadow: 0 0 0 3px rgba(79, 70, 229, 0.1);
}
.language-flag { font-size: 24px; }
.language-label { font-size: 15px; font-weight: 600; color: var(--color-text-primary); }
.language-check { color: var(--color-primary); font-size: 16px; margin-left: auto; }

.settings-divider {
  height: 1px; background: var(--color-border-light);
  margin: 24px 0;
}
.timezone-display {
  font-size: 14px; font-weight: 500; color: var(--color-text-primary);
  padding: 10px 16px; background: var(--color-bg-hover);
  border-radius: var(--radius-md); display: inline-block; margin-top: 8px;
}

/* Security tab */
.password-strength { margin-top: 8px; }
.strength-bar {
  height: 4px; background: var(--color-bg-hover);
  border-radius: 2px; overflow: hidden; margin-bottom: 4px;
}
.strength-fill {
  height: 100%; border-radius: 2px;
  transition: width 0.3s ease, background 0.3s ease;
}
.strength-label { font-size: 12px; font-weight: 500; }
.security-info { margin-top: 16px; }
.security-info-row {
  display: flex; align-items: center; gap: 12px;
  padding: 8px 0;
}
.login-history-list { display: flex; flex-direction: column; gap: 8px; margin-top: 12px; }
.login-history-item {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 14px; border-radius: var(--radius-md);
  background: var(--color-bg-hover); transition: background 0.2s ease;
}
.login-history-item:hover { background: var(--color-bg-active); }
.login-history-time { font-size: 13px; font-weight: 500; color: var(--color-text-primary); }
.login-history-details { font-size: 12px; color: var(--color-text-tertiary); display: flex; align-items: center; gap: 6px; }
.login-history-sep { opacity: 0.4; }

.field-group-label {
  font-size: 13px; font-weight: 600; color: var(--color-text-secondary);
  margin: 12px 0 8px; padding-left: 4px;
  border-left: 3px solid var(--color-primary);
}
.rule-flow-tag {
  display: inline-flex; align-items: center; gap: 4px;
  font-size: 11px; font-weight: 500; padding: 1px 8px;
  border-radius: var(--radius-full);
  background: var(--color-info-bg); color: var(--color-info);
}

/* Field picker toolbar */
.field-picker-toolbar {
  display: flex; justify-content: space-between; align-items: center; margin-bottom: 14px;
}

/* Selected fields display */
.selected-fields-display { display: flex; flex-wrap: wrap; gap: 8px; }
.selected-field-tag {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 6px 12px; border-radius: var(--radius-md);
  background: var(--color-primary-bg); border: 1px solid var(--color-primary-lighter);
  font-size: 13px; color: var(--color-text-primary);
}
.selected-field-name { font-weight: 500; }
.selected-field-group { margin-bottom: 12px; }
.field-empty-hint {
  padding: 24px; text-align: center; color: var(--color-text-tertiary);
  font-size: 13px; background: var(--color-bg-hover); border-radius: var(--radius-md);
}

/* Field picker modal */
.field-picker-modal {
  display: grid; grid-template-columns: 1fr 1fr; gap: 16px;
  min-height: 400px; margin-top: 12px;
}
.field-picker-left, .field-picker-right {
  border: 1px solid var(--color-border-light); border-radius: var(--radius-md);
  display: flex; flex-direction: column; overflow: hidden;
}
.field-picker-panel-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 14px; background: var(--color-bg-hover);
  font-size: 13px; font-weight: 600; color: var(--color-text-primary);
  border-bottom: 1px solid var(--color-border-light);
}
.field-picker-count {
  font-size: 11px; font-weight: 500; padding: 1px 8px;
  border-radius: var(--radius-full); background: var(--color-primary-bg); color: var(--color-primary);
}
.field-picker-search { padding: 8px 10px; border-bottom: 1px solid var(--color-border-light); }
.field-picker-list { flex: 1; overflow-y: auto; padding: 4px; }
.field-picker-item {
  display: flex; align-items: center; justify-content: space-between;
  padding: 8px 10px; border-radius: var(--radius-sm); cursor: pointer;
  transition: all var(--transition-fast); gap: 8px;
}
.field-picker-item:hover { background: var(--color-bg-hover); }
.field-picker-item--selected { cursor: default; }
.field-picker-item--selected:hover { background: transparent; }
.field-picker-item-name { font-size: 13px; font-weight: 500; color: var(--color-text-primary); }
.field-picker-item-meta { display: flex; align-items: center; gap: 6px; margin-top: 2px; }
.field-picker-group-label {
  font-size: 12px; font-weight: 600; color: var(--color-text-secondary);
  padding: 6px 10px 2px; margin-top: 4px;
  border-left: 3px solid var(--color-primary);
}
.field-picker-arrow { color: var(--color-primary); font-size: 14px; flex-shrink: 0; }
.field-picker-remove {
  width: 22px; height: 22px; border: none; background: transparent;
  border-radius: var(--radius-sm); cursor: pointer; display: flex;
  align-items: center; justify-content: center;
  color: var(--color-text-tertiary); font-size: 11px;
  transition: all var(--transition-fast); flex-shrink: 0;
}
.field-picker-remove:hover { background: var(--color-danger-bg); color: var(--color-danger); }
.field-picker-empty {
  padding: 32px 16px; text-align: center; color: var(--color-text-tertiary); font-size: 13px;
}
</style>
