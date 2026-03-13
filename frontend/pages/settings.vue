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
  ReloadOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type { Locale } from '~/composables/useI18n'
import type {
  ProcessListItem,
  FullAuditProcessConfig,
  TenantField,
  TenantRule,
  CustomRule,
  CronPrefs,
  AccessibleArchiveConfig,
  FullArchiveConfig,
  UpdatePersonalConfigRequest,
} from '~/types/user-config'
import { usePagination } from '~/composables/usePagination'

definePageMeta({
  middleware: 'auth',
  layout: 'default',
})

const { userRole, activeRole, getProfile, updateProfile, updateLocale, setUserLocale } = useAuth()
const settingsApi = useSettingsApi()
const { t, locale, setLocale, availableLocales } = useI18n()

import type { MeResponse, MeOrgRole } from '~/types/auth'
const meData = ref<MeResponse | null>(null)
const meLoading = ref(false)

const fetchMe = async () => {
  meLoading.value = true
  meData.value = await getProfile()
  meLoading.value = false
}

// ===== 页面初始化 =====
onMounted(async () => {
  fetchMe()
  loadProcessList()
  loadCronPrefs()
  loadArchiveList()
})

const currentRoleType = computed(() => activeRole.value?.role || userRole.value)
const activeTab = ref('profile')

// ===== 语言设置 =====
const handleLocaleChange = async (newLocale: Locale) => {
  setLocale(newLocale)
  message.success(t('settings.language.switchSuccess'))
  await updateLocale(newLocale)
}

// ===== 安全/密码 =====
const passwordForm = ref({ currentPassword: '', newPassword: '', confirmPassword: '' })
const passwordChanging = ref(false)

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
  const history = (meData.value?.login_history || []).map(h => ({ time: h.time, ip: h.ip, device: h.device }))
  return {
    password_last_changed: meData.value?.password_changed_at || '-',
    login_history: history,
  }
})

const handleChangePassword = async () => {
  const { currentPassword, newPassword, confirmPassword } = passwordForm.value
  if (!currentPassword || !newPassword || !confirmPassword) { message.error(t('settings.security.changeError.empty')); return }
  if (newPassword.length < 6) { message.error(t('settings.security.changeError.tooShort')); return }
  if (currentPassword === newPassword) { message.error(t('settings.security.changeError.samePassword')); return }
  if (newPassword !== confirmPassword) { message.error(t('settings.security.changeError.mismatch')); return }
  const { changePassword, logout } = useAuth()
  passwordChanging.value = true
  const ok = await changePassword({ current_password: currentPassword, new_password: newPassword })
  passwordChanging.value = false
  if (!ok) { message.error(t('settings.security.changeError.wrongCurrent')); return }
  passwordForm.value = { currentPassword: '', newPassword: '', confirmPassword: '' }
  message.success(t('settings.security.changeSuccess'))
  setTimeout(() => { logout() }, 1500)
}

// ===== 个人资料 =====
const currentOrgRoles = computed<MeOrgRole[]>(() => meData.value?.org_roles || [])

const currentOrgPagePermissions = computed(() => {
  if (meData.value?.page_permissions?.length) return meData.value.page_permissions
  const { menus } = useAuth()
  return menus.value.map((m: any) => m.path).filter(Boolean)
})

const getPageLabel = (path: string) => t(`page.${path}`, path)

const profile = ref({ nickname: '', email: '', phone: '', department: '', position: '' })

watch(meData, (me) => {
  if (!me) return
  profile.value = {
    nickname: me.user.display_name || '',
    email: me.user.email || '',
    phone: me.user.phone || '',
    department: me.department_name || '',
    position: me.position || '',
  }
  if (me.user.locale && (me.user.locale === 'zh-CN' || me.user.locale === 'en-US')) {
    setUserLocale(me.user.locale)
  }
}, { immediate: true })

const getRoleLabel = (role: string) => {
  const map: Record<string, string> = { business: 'role.business', tenant_admin: 'role.tenantAdmin', system_admin: 'role.systemAdmin' }
  return t(map[role] || 'role.business')
}

// ===== Tab 可见性 =====
const visibleTabs = computed(() => {
  const role = currentRoleType.value
  const perms = currentOrgPagePermissions.value
  const isBiz = role === 'business'
  return [
    { key: 'profile', label: t('settings.tab.profile'), icon: UserOutlined, show: true },
    { key: 'workbench', label: t('settings.tab.workbench'), icon: DashboardOutlined, show: isBiz && perms.includes('/dashboard') },
    { key: 'cron', label: t('settings.tab.cron'), icon: ClockCircleOutlined, show: isBiz && perms.includes('/cron') },
    { key: 'archive', label: t('settings.tab.archive'), icon: FolderOpenOutlined, show: isBiz && perms.includes('/archive') },
  ].filter(tab => tab.show)
})

watch(visibleTabs, (tabs) => {
  if (!tabs.some(t => t.key === activeTab.value)) activeTab.value = 'profile'
})

const saving = ref(false)
const handleSaveProfile = async () => {
  if (profile.value.email.trim()) {
    if (!/^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$/.test(profile.value.email.trim())) {
      message.error(t('settings.profile.emailFormatError')); return
    }
  }
  if (profile.value.phone.trim() && !/^\d{11}$/.test(profile.value.phone.trim())) {
    message.error(t('settings.profile.phoneFormatError')); return
  }
  if (!profile.value.nickname.trim()) { message.error(t('settings.profile.nicknameRequired')); return }
  saving.value = true
  const { ok, errorMsg } = await updateProfile({
    display_name: profile.value.nickname.trim(),
    email: profile.value.email.trim(),
    phone: profile.value.phone.trim(),
  })
  saving.value = false
  if (ok) { message.success(t('settings.profile.saveSuccess')); await fetchMe() }
  else message.error(errorMsg || t('settings.profile.saveFailed'))
}

// =====================================================================
// ===== 审核工作台 Tab =====
// =====================================================================
const workbenchLoading = ref(false)
const processList = ref<ProcessListItem[]>([])
const selectedProcessType = ref('')
const fullProcessConfig = ref<FullAuditProcessConfig | null>(null)
const workbenchSection = ref<'fields' | 'rules' | 'ai'>('fields')

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

const scopeConfig = computed<Record<string, { label: string }>>(() => ({
  mandatory: { label: t('rule.scope.mandatory') },
  default_on: { label: t('rule.scope.defaultOn') },
  default_off: { label: t('rule.scope.defaultOff') },
}))

const strictnessOptions = computed(() => [
  { value: 'strict', label: t('settings.workbench.strict'), desc: t('settings.workbench.strictDesc') },
  { value: 'standard', label: t('settings.workbench.standard'), desc: t('settings.workbench.standardDesc') },
  { value: 'loose', label: t('settings.workbench.loose'), desc: t('settings.workbench.looseDesc') },
])

const loadProcessList = async () => {
  workbenchLoading.value = true
  try {
    processList.value = await settingsApi.listProcesses()
    if (processList.value.length > 0) {
      selectedProcessType.value = processList.value[0].process_type
      await loadFullProcessConfig(selectedProcessType.value)
    }
  }
  catch (e) { console.error('[settings] 加载流程列表失败', e) }
  finally { workbenchLoading.value = false }
}

const loadFullProcessConfig = async (processType: string) => {
  workbenchLoading.value = true
  try {
    const data = await settingsApi.getFullProcessConfig(processType)
    // 标记初始选中状态，用于锁定（只能增不能减）
    data.main_fields.forEach(f => { if (f.selected) (f as any).is_original = true })
    data.detail_tables.forEach(t => t.fields.forEach(f => { if (f.selected) (f as any).is_original = true }))
    fullProcessConfig.value = data
    workbenchSection.value = 'fields'
  }
  catch (e) { console.error('[settings] 加载流程配置失败', e) }
  finally { workbenchLoading.value = false }
}

const selectProcess = async (processType: string) => {
  selectedProcessType.value = processType
  await loadFullProcessConfig(processType)
}

// 字段选择器
const showFieldPicker = ref(false)
const fieldSearchQuery = ref('')

interface PickerFieldGroup {
  source: string; sourceLabel: string
  fields: (TenantField & { source: string; sourceLabel: string })[]
}

const groupedFields = computed<PickerFieldGroup[]>(() => {
  if (!fullProcessConfig.value) return []
  const groups: PickerFieldGroup[] = []
  groups.push({
    source: 'main',
    sourceLabel: t('settings.workbench.mainTableFields'),
    fields: fullProcessConfig.value.main_fields.map(f => ({ ...f, source: 'main', sourceLabel: t('settings.workbench.mainTableFields') })),
  })
  for (const dt of (fullProcessConfig.value.detail_tables || [])) {
    groups.push({
      source: dt.table_name,
      sourceLabel: dt.table_label || dt.table_name,
      fields: dt.fields.map(f => ({ ...f, source: dt.table_name, sourceLabel: dt.table_label || dt.table_name })),
    })
  }
  return groups
})

const allFields = computed(() => groupedFields.value.flatMap(g => g.fields))
const unselectedFieldsFlat = computed(() => {
  const q = fieldSearchQuery.value.toLowerCase().trim()
  return allFields.value.filter(f => {
    if (f.selected) return false
    if (!q) return true
    return f.field_name.toLowerCase().includes(q) || f.field_key.toLowerCase().includes(q)
  })
})
const selectedFieldsFlat = computed(() => {
  return allFields.value.filter(f => f.selected)
})

const leftSelectedKeys = ref<string[]>([])
const rightSelectedKeys = ref<string[]>([])
const fieldSelectedSearchQuery = ref('')

const unselectedPagination = usePagination(unselectedFieldsFlat, 6)
const selectedFieldsFiltered = computed(() => {
  const q = fieldSelectedSearchQuery.value.toLowerCase().trim()
  return selectedFieldsFlat.value.filter(f => {
    if (!q) return true
    return f.field_name.toLowerCase().includes(q) || f.field_key.toLowerCase().includes(q)
  })
})
const selectedPagination = usePagination(selectedFieldsFiltered, 6)
const displaySelectedPagination = usePagination(selectedFieldsFlat, 24)

const selectedFieldCount = computed(() => selectedFieldsFlat.value.length)

const groupedUnselected = computed<PickerFieldGroup[]>(() => {
  const q = fieldSearchQuery.value.toLowerCase().trim()
  return groupedFields.value
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

// 批量操作逻辑
const toggleLeftSelectAll = () => {
  if (leftSelectedKeys.value.length === unselectedFieldsFlat.value.length && unselectedFieldsFlat.value.length > 0) {
    leftSelectedKeys.value = []
  } else {
    leftSelectedKeys.value = unselectedFieldsFlat.value.map(f => f.field_key + '_' + f.source)
  }
}

const toggleLeftSelect = (fieldId: string) => {
  const idx = leftSelectedKeys.value.indexOf(fieldId)
  if (idx >= 0) leftSelectedKeys.value.splice(idx, 1)
  else leftSelectedKeys.value.push(fieldId)
}

const batchPick = () => {
  const toPick = unselectedFieldsFlat.value.filter(f => leftSelectedKeys.value.includes(f.field_key + '_' + f.source))
  toPick.forEach(f => pickField(f))
  leftSelectedKeys.value = []
}

const toggleRightSelectAll = () => {
  if (rightSelectedKeys.value.length === selectedFieldsFiltered.value.length && selectedFieldsFiltered.value.length > 0) {
    rightSelectedKeys.value = []
  } else {
    rightSelectedKeys.value = selectedFieldsFiltered.value.map(f => f.field_key + '_' + f.source)
  }
}

const toggleRightSelect = (fieldId: string) => {
  if (isFieldIdLocked(fieldId)) return
  const idx = rightSelectedKeys.value.indexOf(fieldId)
  if (idx >= 0) rightSelectedKeys.value.splice(idx, 1)
  else rightSelectedKeys.value.push(fieldId)
}

const batchUnpick = () => {
  const toUnpick = selectedFieldsFiltered.value.filter(f => {
    const id = f.field_key + '_' + f.source
    return rightSelectedKeys.value.includes(id) && !isFieldLocked(f)
  })
  toUnpick.forEach(f => unpickField(f))
  rightSelectedKeys.value = []
}

const isFieldIdLocked = (fieldId: string) => {
  const [key, source] = fieldId.split('_')
  const field = allFields.value.find(f => f.field_key === key && f.source === source)
  return field ? isFieldLocked(field) : false
}

const groupedSelected = computed<PickerFieldGroup[]>(() =>
  groupedFields.value
    .map(g => ({ ...g, fields: g.fields.filter(f => f.selected) }))
    .filter(g => g.fields.length > 0),
)

const isFieldLocked = (field: any) => {
  // 租户锁定的，或者本次进入前已保存的字段，不可删除
  return field.locked || field.is_original
}

const pickField = (field: { field_key: string; source: string }) => {
  if (!fullProcessConfig.value || !fullProcessConfig.value.user_permissions.allow_custom_fields) return
  const cfg = fullProcessConfig.value
  if (field.source === 'main') {
    const f = cfg.main_fields.find(f => f.field_key === field.field_key)
    if (f) f.selected = true
  }
  else {
    const dt = cfg.detail_tables.find(d => d.table_name === field.source)
    if (dt) { const f = dt.fields.find(f => f.field_key === field.field_key); if (f) f.selected = true }
  }
}

const unpickField = (field: { field_key: string; source: string }) => {
  if (!fullProcessConfig.value || !fullProcessConfig.value.user_permissions.allow_custom_fields) return
  if (isFieldLocked(field as any)) return
  const cfg = fullProcessConfig.value
  if (field.source === 'main') {
    const f = cfg.main_fields.find(f => f.field_key === field.field_key)
    if (f) f.selected = false
  }
  else {
    const dt = cfg.detail_tables.find(d => d.table_name === field.source)
    if (dt) { const f = dt.fields.find(f => f.field_key === field.field_key); if (f) f.selected = false }
  }
}

// 自定义规则
const newRuleContent = ref('')
const newRuleRelatedFlow = ref(false)

const addCustomRule = () => {
  if (!newRuleContent.value.trim() || !fullProcessConfig.value) return
  if (!fullProcessConfig.value.custom_rules) fullProcessConfig.value.custom_rules = []
  fullProcessConfig.value.custom_rules.push({
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
  if (!fullProcessConfig.value) return
  fullProcessConfig.value.custom_rules = fullProcessConfig.value.custom_rules.filter(r => r.id !== ruleId)
  message.success(t('settings.workbench.deleted'))
}

// 保存审核工作台配置
const handleSaveWorkbench = async () => {
  if (!fullProcessConfig.value) return
  const cfg = fullProcessConfig.value
  const selectedKeys = allFields.value.filter(f => f.selected).map(f => f.field_key)
  const ruleToggleOverrides = cfg.tenant_rules
    .filter(r => r.rule_scope !== 'mandatory')
    .map(r => ({ rule_id: r.id, enabled: r.enabled }))
  
  const req: UpdatePersonalConfigRequest = {
    config_id: cfg.config_id,
    field_config: {
      field_mode: cfg.field_mode,
      field_overrides: cfg.field_mode === 'selected' ? selectedKeys : [],
    },
    rule_config: {
      custom_rules: cfg.custom_rules || [],
      rule_toggle_overrides: ruleToggleOverrides,
    },
    ai_config: {
      strictness_override: cfg.audit_strictness,
    }
  }
  
  saving.value = true
  try {
    await settingsApi.updateProcessConfig(cfg.process_type, req)
    message.success(t('settings.workbench.saveSuccess'))
  }
  catch (e) { message.error(t('settings.workbench.saveFailed')) }
  finally { saving.value = false }
}

// =====================================================================
// ===== 定时任务 Tab =====
// =====================================================================
const cronLoading = ref(false)
const cronDefaultEmail = ref('')

const loadCronPrefs = async () => {
  cronLoading.value = true
  try {
    const prefs = await settingsApi.getCronPrefs()
    // 若无配置，用用户自己的邮箱作默认
    if (prefs.default_email) {
      cronDefaultEmail.value = prefs.default_email
    }
  }
  catch (e) { console.error('[settings] 加载 cron 偏好失败', e) }
  finally { cronLoading.value = false }
}

const handleSaveCron = async () => {
  saving.value = true
  try {
    await settingsApi.updateCronPrefs({ default_email: cronDefaultEmail.value })
    message.success(t('settings.cron.saveSuccess'))
  }
  catch (e) { message.error(t('settings.cron.saveFailed')) }
  finally { saving.value = false }
}

// =====================================================================
// ===== 归档复盘 Tab =====
// =====================================================================
const archiveLoading = ref(false)
const archiveList = ref<AccessibleArchiveConfig[]>([])
const selectedArchiveProcessType = ref('')
const fullArchiveConfig = ref<FullArchiveConfig | null>(null)
const archiveSection = ref<'fields' | 'rules' | 'ai'>('fields')

const loadArchiveList = async () => {
  archiveLoading.value = true
  try {
    archiveList.value = await settingsApi.listArchiveConfigs()
    if (archiveList.value.length > 0) {
      selectedArchiveProcessType.value = archiveList.value[0].process_type
      await loadFullArchiveConfig(selectedArchiveProcessType.value)
    }
  }
  catch (e) { console.error('[settings] 加载归档配置列表失败', e) }
  finally { archiveLoading.value = false }
}

const loadFullArchiveConfig = async (processType: string) => {
  archiveLoading.value = true
  try {
    const data = await settingsApi.getFullArchiveConfig(processType)
    // 标记初始选中状态
    data.main_fields.forEach(f => { if (f.selected) (f as any).is_original = true })
    data.detail_tables.forEach(t => t.fields.forEach(f => { if (f.selected) (f as any).is_original = true }))
    fullArchiveConfig.value = data
    archiveSection.value = 'fields'
  }
  catch (e) { console.error('[settings] 加载归档配置失败', e) }
  finally { archiveLoading.value = false }
}

const selectArchiveProcess = async (processType: string) => {
  selectedArchiveProcessType.value = processType
  await loadFullArchiveConfig(processType)
}

// 归档字段选择器
const showArchiveFieldPicker = ref(false)
const archiveFieldSearchQuery = ref('')

const archiveGroupedFields = computed<PickerFieldGroup[]>(() => {
  if (!fullArchiveConfig.value) return []
  const groups: PickerFieldGroup[] = []
  groups.push({
    source: 'main',
    sourceLabel: t('settings.workbench.mainTableFields'),
    fields: fullArchiveConfig.value.main_fields.map(f => ({ ...f, source: 'main', sourceLabel: t('settings.workbench.mainTableFields') })),
  })
  for (const dt of (fullArchiveConfig.value.detail_tables || [])) {
    groups.push({
      source: dt.table_name,
      sourceLabel: dt.table_label || dt.table_name,
      fields: dt.fields.map(f => ({ ...f, source: dt.table_name, sourceLabel: dt.table_label || dt.table_name })),
    })
  }
  return groups
})

const archiveAllFields = computed(() => archiveGroupedFields.value.flatMap(g => g.fields))
const archiveUnselectedFieldsFlat = computed(() => {
  const q = archiveFieldSearchQuery.value.toLowerCase().trim()
  return archiveAllFields.value.filter(f => {
    if (f.selected) return false
    if (!q) return true
    return f.field_name.toLowerCase().includes(q) || f.field_key.toLowerCase().includes(q)
  })
})
const archiveSelectedFieldsFlat = computed(() => {
  return archiveAllFields.value.filter(f => f.selected)
})

const archiveLeftSelectedKeys = ref<string[]>([])
const archiveRightSelectedKeys = ref<string[]>([])
const archiveFieldSelectedSearchQuery = ref('')

const archiveUnselectedPagination = usePagination(archiveUnselectedFieldsFlat, 6)
const archiveSelectedFieldsFiltered = computed(() => {
  const q = archiveFieldSelectedSearchQuery.value.toLowerCase().trim()
  return archiveSelectedFieldsFlat.value.filter(f => {
    if (!q) return true
    return f.field_name.toLowerCase().includes(q) || f.field_key.toLowerCase().includes(q)
  })
})
const archiveSelectedPagination = usePagination(archiveSelectedFieldsFiltered, 6)
const archiveDisplaySelectedPagination = usePagination(archiveSelectedFieldsFlat, 24)

const archiveSelectedCount = computed(() => archiveSelectedFieldsFlat.value.length)

const archiveGroupedUnselected = computed<PickerFieldGroup[]>(() => {
  const q = archiveFieldSearchQuery.value.toLowerCase().trim()
  return archiveGroupedFields.value
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

const toggleArchiveLeftSelectAll = () => {
  if (archiveLeftSelectedKeys.value.length === archiveUnselectedFieldsFlat.value.length && archiveUnselectedFieldsFlat.value.length > 0) {
    archiveLeftSelectedKeys.value = []
  } else {
    archiveLeftSelectedKeys.value = archiveUnselectedFieldsFlat.value.map(f => f.field_key + '_' + f.source)
  }
}

const toggleArchiveLeftSelect = (fieldId: string) => {
  const idx = archiveLeftSelectedKeys.value.indexOf(fieldId)
  if (idx >= 0) archiveLeftSelectedKeys.value.splice(idx, 1)
  else archiveLeftSelectedKeys.value.push(fieldId)
}

const batchArchivePick = () => {
  const toPick = archiveUnselectedFieldsFlat.value.filter(f => archiveLeftSelectedKeys.value.includes(f.field_key + '_' + f.source))
  toPick.forEach(f => archivePickField(f))
  archiveLeftSelectedKeys.value = []
}

const toggleArchiveRightSelectAll = () => {
  if (archiveRightSelectedKeys.value.length === archiveSelectedFieldsFiltered.value.length && archiveSelectedFieldsFiltered.value.length > 0) {
    archiveRightSelectedKeys.value = []
  } else {
    archiveRightSelectedKeys.value = archiveSelectedFieldsFiltered.value.map(f => f.field_key + '_' + f.source)
  }
}

const toggleArchiveRightSelect = (fieldId: string) => {
  if (isArchiveFieldIdLocked(fieldId)) return
  const idx = archiveRightSelectedKeys.value.indexOf(fieldId)
  if (idx >= 0) archiveRightSelectedKeys.value.splice(idx, 1)
  else archiveRightSelectedKeys.value.push(fieldId)
}

const batchArchiveUnpick = () => {
  const toUnpick = archiveSelectedFieldsFiltered.value.filter(f => {
    const id = f.field_key + '_' + f.source
    return archiveRightSelectedKeys.value.includes(id) && !isArchiveFieldLocked(f)
  })
  toUnpick.forEach(f => archiveUnpickField(f))
  archiveRightSelectedKeys.value = []
}

const isArchiveFieldLocked = (field: any) => {
  return field.locked || field.is_original
}

const isArchiveFieldIdLocked = (fieldId: string) => {
  const [key, source] = fieldId.split('_')
  const field = archiveAllFields.value.find(f => f.field_key === key && f.source === source)
  return field ? isArchiveFieldLocked(field) : false
}

const archiveGroupedSelected = computed<PickerFieldGroup[]>(() =>
  archiveGroupedFields.value
    .map(g => ({ ...g, fields: g.fields.filter(f => f.selected) }))
    .filter(g => g.fields.length > 0),
)

const archivePickField = (field: { field_key: string; source: string }) => {
  if (!fullArchiveConfig.value || !fullArchiveConfig.value.user_permissions.allow_custom_fields) return
  const cfg = fullArchiveConfig.value
  if (field.source === 'main') {
    const f = cfg.main_fields.find(f => f.field_key === field.field_key)
    if (f) f.selected = true
  }
  else {
    const dt = cfg.detail_tables.find(d => d.table_name === field.source)
    if (dt) { const f = dt.fields.find(f => f.field_key === field.field_key); if (f) f.selected = true }
  }
}

const archiveUnpickField = (field: { field_key: string; source: string }) => {
  if (!fullArchiveConfig.value || !fullArchiveConfig.value.user_permissions.allow_custom_fields) return
  if (isArchiveFieldLocked(field as any)) return
  const cfg = fullArchiveConfig.value
  if (field.source === 'main') {
    const f = cfg.main_fields.find(f => f.field_key === field.field_key)
    if (f) f.selected = false
  }
  else {
    const dt = cfg.detail_tables.find(d => d.table_name === field.source)
    if (dt) { const f = dt.fields.find(f => f.field_key === field.field_key); if (f) f.selected = false }
  }
}

// 归档自定义规则
const newArchiveRuleContent = ref('')
const newArchiveRuleRelatedFlow = ref(false)

const addArchiveCustomRule = () => {
  if (!newArchiveRuleContent.value.trim() || !fullArchiveConfig.value) return
  if (!fullArchiveConfig.value.custom_rules) fullArchiveConfig.value.custom_rules = []
  fullArchiveConfig.value.custom_rules.push({
    id: `UACR-${Date.now()}`,
    content: newArchiveRuleContent.value.trim(),
    enabled: true,
    related_flow: newArchiveRuleRelatedFlow.value,
  })
  newArchiveRuleContent.value = ''
  newArchiveRuleRelatedFlow.value = false
  message.success(t('settings.archive.ruleAdded'))
}

const removeArchiveCustomRule = (ruleId: string) => {
  if (!fullArchiveConfig.value) return
  fullArchiveConfig.value.custom_rules = fullArchiveConfig.value.custom_rules.filter(r => r.id !== ruleId)
  message.success(t('settings.workbench.deleted'))
}

// 保存归档复盘配置
const handleSaveArchive = async () => {
  if (!fullArchiveConfig.value) return
  const cfg = fullArchiveConfig.value
  const selectedKeys = archiveAllFields.value.filter(f => f.selected).map(f => f.field_key)
  const ruleToggleOverrides = cfg.tenant_rules
    .filter(r => r.rule_scope !== 'mandatory')
    .map(r => ({ rule_id: r.id, enabled: r.enabled }))
  
  const req: UpdatePersonalConfigRequest = {
    config_id: cfg.config_id,
    field_config: {
      field_mode: cfg.field_mode,
      field_overrides: cfg.field_mode === 'selected' ? selectedKeys : [],
    },
    rule_config: {
      custom_rules: cfg.custom_rules || [],
      rule_toggle_overrides: ruleToggleOverrides,
    },
    ai_config: {
      strictness_override: cfg.audit_strictness,
    }
  }
  
  saving.value = true
  try {
    await settingsApi.updateArchiveConfig(cfg.process_type, req)
    message.success(t('settings.archive.saveSuccess'))
  }
  catch (e) { message.error(t('settings.archive.saveFailed')) }
  finally { saving.value = false }
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

    <!-- Tab 导航 -->
    <div class="tab-nav">
      <button
        v-for="tab in visibleTabs"
        :key="tab.key"
        class="tab-btn"
        :class="{ 'tab-btn--active': activeTab === tab.key }"
        @click="activeTab = tab.key"
      >
        <component :is="tab.icon" />
        {{ tab.label }}
      </button>
    </div>

    <!-- ===== 个人资料 Tab ===== -->
    <div v-if="activeTab === 'profile'" class="tab-content">
      <div class="settings-card">
        <div class="profile-avatar-section">
          <a-avatar :size="72" class="profile-avatar">
            <template #icon><UserOutlined /></template>
          </a-avatar>
          <div class="profile-avatar-info">
            <div class="profile-name">{{ profile.nickname }}</div>
            <div class="profile-role">
              <template v-if="currentRoleType === 'system_admin'">
                <span class="role-badge">{{ getRoleLabel('system_admin') }}</span>
              </template>
              <template v-else-if="currentOrgRoles.length">
                <span v-for="r in currentOrgRoles" :key="r.id" class="role-badge" style="margin-right: 4px;">{{ r.name }}</span>
              </template>
              <template v-else>
                <span class="role-badge">{{ getRoleLabel(currentRoleType) }}</span>
              </template>
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
          <a-button type="primary" size="large" :disabled="saving" @click="handleSaveProfile">
            <LoadingOutlined v-if="saving" spin />
            <SaveOutlined v-else />
            {{ t('settings.profile.save') }}
          </a-button>
        </div>
      </div>

      <!-- 角色与权限卡 -->
      <div class="settings-card" style="margin-top: 20px;">
        <h4 class="perm-card-title">
          <SafetyCertificateOutlined style="color: var(--color-primary);" />
          {{ t('settings.profile.roleAndPermissions') }}
        </h4>
        <template v-if="currentRoleType === 'system_admin'">
          <div class="perm-info-row">
            <span class="perm-info-label">{{ t('settings.profile.currentRole') }}</span>
            <span class="perm-role-badge">{{ t('role.systemAdmin') }}</span>
          </div>
          <div class="perm-info-row">
            <span class="perm-info-label">{{ t('settings.profile.roleDescription') }}</span>
            <span class="perm-info-value">{{ t('settings.profile.sysAdminDesc') }}</span>
          </div>
          <div class="perm-pages-section">
            <span class="perm-info-label">{{ t('settings.profile.accessiblePages') }}</span>
            <div class="perm-page-tags">
              <span class="perm-page-tag">{{ t('page./overview') }}</span>
              <span class="perm-page-tag">{{ t('page./admin/system/tenants') }}</span>
              <span class="perm-page-tag">{{ t('page./admin/system/settings') }}</span>
              <span class="perm-page-tag">{{ t('page./settings') }}</span>
            </div>
          </div>
        </template>
        <template v-else>
          <div class="perm-info-row">
            <span class="perm-info-label">{{ t('settings.profile.currentTenant') }}</span>
            <span class="perm-info-value">{{ meData?.tenant_name || activeRole?.tenant_name || '-' }}</span>
          </div>
          <div class="perm-info-row">
            <span class="perm-info-label">{{ t('settings.profile.currentRole') }}</span>
            <div class="perm-role-badges">
              <span v-for="r in currentOrgRoles" :key="r.id" class="perm-role-badge" style="margin-right: 6px;">{{ r.name }}</span>
              <span v-if="!currentOrgRoles.length" class="perm-role-badge">{{ getRoleLabel(currentRoleType) }}</span>
            </div>
          </div>
          <div v-if="currentOrgRoles.length === 1 && currentOrgRoles[0]?.description" class="perm-info-row">
            <span class="perm-info-label">{{ t('settings.profile.roleDescription') }}</span>
            <span class="perm-info-value">{{ currentOrgRoles[0].description }}</span>
          </div>
          <div v-if="meData?.department_name" class="perm-info-row">
            <span class="perm-info-label">{{ t('settings.profile.belongDepartment') }}</span>
            <span class="perm-info-value">{{ meData.department_name }}</span>
          </div>
          <div class="perm-pages-section">
            <span class="perm-info-label">{{ t('settings.profile.accessiblePages') }}</span>
            <div class="perm-page-tags">
              <span v-for="p in currentOrgPagePermissions" :key="p" class="perm-page-tag">{{ getPageLabel(p) }}</span>
            </div>
          </div>
        </template>
        <p class="perm-hint-text">{{ t('settings.profile.permissionHint') }}</p>
      </div>

      <!-- 语言设置 -->
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

      <!-- 安全/密码 -->
      <div class="settings-card" style="margin-top: 20px;">
        <h4 class="perm-card-title">
          <LockOutlined style="color: var(--color-primary);" />
          {{ t('settings.security.title') }}
        </h4>
        <p class="config-section-desc" style="margin-bottom: 16px;">{{ t('settings.security.subtitle') }}</p>
        <a-form layout="vertical" class="settings-form" style="max-width: 480px;">
          <a-form-item :label="t('settings.security.currentPassword')">
            <a-input-password v-model:value="passwordForm.currentPassword" size="large" :placeholder="t('settings.security.currentPasswordPlaceholder')" />
          </a-form-item>
          <a-form-item :label="t('settings.security.newPassword')">
            <a-input-password v-model:value="passwordForm.newPassword" size="large" :placeholder="t('settings.security.newPasswordPlaceholder')" />
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
            <a-input-password v-model:value="passwordForm.confirmPassword" size="large" :placeholder="t('settings.security.confirmPasswordPlaceholder')" />
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
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ===== 审核工作台 Tab ===== -->
    <div v-if="activeTab === 'workbench'" class="tab-content">
      <div v-if="workbenchLoading && !processList.length" class="loading-placeholder">
        <a-spin :tip="t('common.loading')" />
      </div>
      <div v-else-if="!processList.length" class="settings-card">
        <a-empty :description="t('settings.workbench.noProcess')" />
      </div>
      <div v-else class="workbench-layout">
        <!-- 左：流程列表 -->
        <div class="process-list-panel">
          <div class="process-list-header">
            <SettingOutlined />
            <span>{{ t('settings.workbench.myProcesses') }}</span>
          </div>
          <div
            v-for="proc in processList"
            :key="proc.process_type"
            class="process-list-item"
            :class="{ 'process-list-item--active': selectedProcessType === proc.process_type }"
            @click="selectProcess(proc.process_type)"
          >
            <div class="process-list-item-name">{{ proc.process_type_label || proc.process_type }}</div>
            <div v-if="proc.process_type_label" class="process-list-item-path">{{ proc.process_type }}</div>
          </div>
        </div>

        <!-- 右：配置详情 -->
        <div v-if="fullProcessConfig && !workbenchLoading" class="process-config-panel">
          <h3 class="config-title">{{ fullProcessConfig.process_type_label || fullProcessConfig.process_type }} — {{ t('settings.workbench.personalConfig') }}</h3>

          <!-- 子导航 -->
          <div class="section-nav">
            <button
              v-for="sec in [
                { key: 'fields', label: t('settings.workbench.fieldsTab'), icon: AppstoreOutlined },
                { key: 'rules', label: t('settings.workbench.rulesTab'), icon: NodeIndexOutlined },
                { key: 'ai', label: t('settings.workbench.aiTab'), icon: ControlOutlined },
              ]"
              :key="sec.key"
              class="section-nav-btn"
              :class="{ 'section-nav-btn--active': workbenchSection === sec.key }"
              @click="workbenchSection = sec.key as any"
            >
              <component :is="sec.icon" />
              {{ sec.label }}
            </button>
          </div>

          <!-- 字段部分 -->
          <div v-if="workbenchSection === 'fields'" class="config-section">
            <div class="section-header-row">
              <h4 class="config-section-title">{{ t('settings.workbench.fieldsTitle') }}</h4>
              <span v-if="!fullProcessConfig.user_permissions.allow_custom_fields" class="locked-tag">
                <LockOutlined /> {{ t('settings.workbench.lockedByAdmin') }}
              </span>
            </div>
            <p class="config-section-desc">
              {{ fullProcessConfig.field_mode === 'all' ? t('settings.workbench.allFieldsMode') : t('settings.workbench.selectedFieldsMode') }}
              <template v-if="fullProcessConfig.user_permissions.allow_custom_fields && fullProcessConfig.field_mode === 'selected'">
                {{ t('settings.workbench.canToggleFields') }}
              </template>
            </p>

            <template v-if="fullProcessConfig.field_mode === 'selected'">
              <div class="field-picker-toolbar">
                <span class="field-count">{{ t('settings.workbench.fieldSelected', [selectedFieldCount, allFields.length]) }}</span>
                <a-button v-if="fullProcessConfig.user_permissions.allow_custom_fields" type="primary" size="small" @click="showFieldPicker = true; fieldSearchQuery = ''">
                  <AppstoreOutlined /> {{ t('settings.workbench.selectFields') }}
                </a-button>
              </div>
                <!-- 字段列表展示 -->
                <div v-if="selectedFieldsFlat.length > 0">
                  <div v-for="group in groupedSelected" :key="group.source" class="selected-field-group">
                    <div class="field-group-label">{{ group.sourceLabel }}</div>
                    <div class="selected-fields-display">
                      <div
                        v-for="field in group.fields"
                        :key="field.field_key + '_' + field.source"
                        class="selected-field-tag"
                      >
                        <span class="selected-field-name">{{ field.field_name }}</span>
                        <span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span>
                      </div>
                    </div>
                  </div>
                  <!-- 分页控制 (可选，如果字段很多可以整体分页，但通常分组后整体分页较好) -->
                  <div v-if="selectedFieldsFlat.length > 24" class="pagination-wrapper" style="margin-top: 20px; border-top: none;">
                    <a-pagination
                      v-model:current="displaySelectedPagination.current.value"
                      v-model:page-size="displaySelectedPagination.pageSize.value"
                      :total="selectedFieldsFlat.length"
                      size="small"
                      show-size-changer
                      :page-size-options="['24', '48', '96']"
                    />
                  </div>
                </div>
                <div v-else class="text-center py-8 bg-slate-50 rounded-xl border border-dashed border-slate-200">
                  <div class="text-slate-400 text-sm">{{ t('settings.workbench.noFieldsSelected') }}</div>
                </div>
            </template>
            <template v-else>
              <div class="field-count" style="margin-top: 8px;">{{ t('settings.workbench.allFieldsModeDesc') }}</div>
            </template>
          </div>

          <!-- 规则部分 -->
          <div v-if="workbenchSection === 'rules'" class="config-section">
            <!-- 租户规则 -->
            <div class="section-header-row">
              <h4 class="config-section-title">{{ t('settings.workbench.tenantRules') }}</h4>
            </div>
            <div v-if="fullProcessConfig.tenant_rules.length" class="rule-config-list">
              <div v-for="rule in fullProcessConfig.tenant_rules" :key="rule.id" class="rule-config-item">
                <div class="rule-config-content">
                  <span class="rule-config-text">{{ rule.rule_content }}</span>
                  <span v-if="rule.related_flow" class="rule-flow-tag">
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
                <a-switch v-model:checked="rule.enabled" size="small" :disabled="rule.rule_scope === 'mandatory'" />
              </div>
            </div>
            <div v-else class="field-empty-hint">{{ t('settings.workbench.noTenantRules') }}</div>

            <!-- 自定义规则 -->
            <div class="section-header-row" style="margin-top: 20px;">
              <h4 class="config-section-title">{{ t('settings.workbench.personalRules') }}</h4>
              <span v-if="!fullProcessConfig.user_permissions.allow_custom_rules" class="locked-tag">
                <LockOutlined /> {{ t('settings.workbench.lockedByAdmin') }}
              </span>
            </div>
            <p class="config-section-desc">
              {{ fullProcessConfig.user_permissions.allow_custom_rules ? t('settings.workbench.personalRulesAllowed') : t('settings.workbench.personalRulesDenied') }}
            </p>
            <div v-if="fullProcessConfig.custom_rules?.length" class="rule-config-list">
              <div v-for="rule in fullProcessConfig.custom_rules" :key="rule.id" class="rule-config-item">
                <div class="rule-config-content">
                  <span class="rule-config-text">{{ rule.content }}</span>
                  <span v-if="rule.related_flow" class="rule-flow-tag">
                    <NodeIndexOutlined /> {{ t('settings.workbench.relatedFlow') }}
                  </span>
                  <span class="rule-scope-tag rule-scope-tag--custom">{{ t('settings.workbench.personal') }}</span>
                </div>
                <div class="rule-config-actions">
                  <a-switch v-model:checked="rule.enabled" size="small" />
                  <a-popconfirm v-if="fullProcessConfig.user_permissions.allow_custom_rules" :title="t('settings.workbench.confirmDelete')" @confirm="removeCustomRule(rule.id)">
                    <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
                  </a-popconfirm>
                </div>
              </div>
            </div>
            <div v-if="fullProcessConfig.user_permissions.allow_custom_rules" class="add-rule-row">
              <a-input
                v-model:value="newRuleContent"
                :placeholder="t('settings.workbench.addRulePlaceholder')"
                @pressEnter="addCustomRule"
              />
              <a-tooltip :title="t('settings.workbench.relatedFlowTip')">
                <button class="icon-btn" :class="{ 'icon-btn--active': newRuleRelatedFlow }" @click="newRuleRelatedFlow = !newRuleRelatedFlow">
                  <NodeIndexOutlined />
                </button>
              </a-tooltip>
              <a-button type="primary" :disabled="!newRuleContent.trim()" @click="addCustomRule">
                <PlusOutlined /> {{ t('settings.workbench.add') }}
              </a-button>
            </div>
          </div>

          <!-- AI 严格度部分 -->
          <div v-if="workbenchSection === 'ai'" class="config-section">
            <div class="section-header-row">
              <h4 class="config-section-title">{{ t('settings.workbench.strictnessTitle') }}</h4>
              <span v-if="!fullProcessConfig.user_permissions.allow_modify_strictness" class="locked-tag">
                <LockOutlined /> {{ t('settings.workbench.lockedByAdmin') }}
              </span>
            </div>
            <p class="config-section-desc">{{ t('settings.workbench.strictnessDesc') }}</p>
            <div class="strictness-options">
              <div
                v-for="opt in strictnessOptions"
                :key="opt.value"
                class="strictness-option"
                :class="{
                  'strictness-option--active': fullProcessConfig.audit_strictness === opt.value,
                  'strictness-option--disabled': !fullProcessConfig.user_permissions.allow_modify_strictness,
                }"
                @click="fullProcessConfig.user_permissions.allow_modify_strictness && (fullProcessConfig.audit_strictness = opt.value)"
              >
                <div class="strictness-option-radio" />
                <div>
                  <div class="strictness-option-label">{{ opt.label }}</div>
                  <div class="strictness-option-desc">{{ opt.desc }}</div>
                </div>
              </div>
            </div>
            <div style="margin-top: 20px;">
              <h4 class="config-section-title">{{ t('settings.workbench.kbMode') }}</h4>
              <p class="config-section-desc">
                {{ t('common.currentMode') }}<span style="font-weight: 600;">
                  {{ fullProcessConfig.kb_mode === 'rules_only' ? t('settings.workbench.kbRulesOnly') : fullProcessConfig.kb_mode === 'rag_only' ? t('settings.workbench.kbRagOnly') : t('settings.workbench.kbHybrid') }}
                </span>
                （{{ t('settings.workbench.configuredByAdmin') }}）
              </p>
            </div>
          </div>

          <div class="settings-actions">
            <a-button type="primary" size="large" :loading="saving" @click="handleSaveWorkbench">
              <SaveOutlined v-if="!saving" />
              {{ t('common.save') }}
            </a-button>
          </div>
        </div>

        <div v-else-if="workbenchLoading" class="process-config-panel" style="display:flex;align-items:center;justify-content:center;min-height:300px;">
          <a-spin :tip="t('common.loading')" />
        </div>
        <div v-else class="process-config-empty">
          <a-empty :description="t('settings.workbench.selectProcess')" />
        </div>
      </div>
    </div>

    <a-modal v-model:open="showFieldPicker" :title="t('settings.workbench.fieldPickerTitle')" :width="720" :footer="null">
      <div class="field-picker-modal">
        <!-- 左侧：待选 -->
        <div class="field-picker-panel">
          <div class="field-picker-panel-header" style="justify-content: flex-start; gap: 8px;">
            <a-checkbox
              :checked="leftSelectedKeys.length === unselectedFieldsFlat.length && unselectedFieldsFlat.length > 0"
              :indeterminate="leftSelectedKeys.length > 0 && leftSelectedKeys.length < unselectedFieldsFlat.length"
              @change="toggleLeftSelectAll"
            />
            <span style="flex: 1;">{{ t('settings.workbench.availableFields') }} <span class="field-count">({{ unselectedFieldsFlat.length }})</span></span>
            <a-button type="primary" size="small" :disabled="leftSelectedKeys.length === 0" @click="batchPick">
              {{ t('common.add')}}
            </a-button>
          </div>
          <div class="field-picker-search">
            <a-input v-model:value="fieldSearchQuery" :placeholder="t('settings.workbench.fieldSearchPlaceholder')" allow-clear size="small">
              <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
            </a-input>
          </div>
          <div class="field-picker-list custom-scrollbar">
            <div v-if="unselectedPagination.paged.value.length > 0" class="space-y-1">
              <div
                v-for="field in unselectedPagination.paged.value"
                :key="field.field_key + '_' + field.source"
                class="field-picker-item"
                @click="toggleLeftSelect(field.field_key + '_' + field.source)"
              >
                <div class="field-picker-item-checkbox" @click.stop="toggleLeftSelect(field.field_key + '_' + field.source)">
                  <a-checkbox :checked="leftSelectedKeys.includes(field.field_key + '_' + field.source)" />
                </div>
                <div class="field-picker-item-info">
                  <div class="field-picker-item-name">{{ field.field_name }} <span class="field-source-tag">({{ field.sourceLabel }})</span></div>
                  <div class="field-picker-item-meta">
                    <span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span>
                    <span class="field-key">{{ field.field_key }}</span>
                  </div>
                </div>
                <button class="icon-btn icon-btn--sm" @click.stop="pickField(field)" style="margin-left: auto;">
                  <SwapRightOutlined />
                </button>
              </div>
            </div>
            <div v-else class="field-picker-empty">
              {{ fieldSearchQuery ? t('settings.workbench.noMatchField') : t('settings.workbench.allFieldsAdded') }}
            </div>
          </div>
          <!-- 待选分页 -->
          <div class="pagination-wrapper">
            <a-pagination
              v-model:current="unselectedPagination.current.value"
              v-model:page-size="unselectedPagination.pageSize.value"
              :total="unselectedFieldsFlat.length"
              size="small"
              show-size-changer
              :page-size-options="['6', '20', '50']"
            />
          </div>
        </div>

        <!-- 右侧：已选 -->
        <div class="field-picker-panel field-picker-panel--right">
          <div class="field-picker-panel-header" style="justify-content: flex-start; gap: 8px;">
             <a-checkbox
              :checked="rightSelectedKeys.length === selectedFieldsFiltered.length && selectedFieldsFiltered.length > 0"
              :indeterminate="rightSelectedKeys.length > 0 && rightSelectedKeys.length < selectedFieldsFiltered.length"
              @change="toggleRightSelectAll"
            />
            <span style="flex: 1;">{{ t('settings.workbench.selectedFields') }} <span class="field-picker-count">{{ selectedFieldCount }}</span></span>
             <a-button danger size="small" :disabled="rightSelectedKeys.length === 0" @click="batchUnpick">
              {{ t('common.remove') }}
            </a-button>
          </div>
          <div class="field-picker-search">
            <a-input v-model:value="fieldSelectedSearchQuery" :placeholder="t('settings.workbench.fieldSearchPlaceholder')" allow-clear size="small">
              <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
            </a-input>
          </div>
          <div class="field-picker-list custom-scrollbar">
            <div v-if="selectedPagination.paged.value.length > 0" class="space-y-1">
              <div
                v-for="field in selectedPagination.paged.value"
                :key="field.field_key + '_' + field.source"
                class="field-picker-item field-picker-item--selected"
                @click="toggleRightSelect(field.field_key + '_' + field.source)"
              >
                <div class="field-picker-item-checkbox" @click.stop="toggleRightSelect(field.field_key + '_' + field.source)">
                  <a-checkbox :checked="rightSelectedKeys.includes(field.field_key + '_' + field.source)" :disabled="isFieldLocked(field)" />
                </div>
                <div class="field-picker-item-info">
                  <div class="flex items-center gap-1.5">
                    <span class="field-picker-item-name">{{ field.field_name }} <span class="field-source-tag">({{ field.sourceLabel }})</span></span>
                    <a-tooltip v-if="isFieldLocked(field)" :title="t('settings.workbench.fieldLocked') || '租户预设或已保存字段，不可删除'">
                      <LockOutlined style="font-size: 10px; color: var(--color-text-tertiary);" />
                    </a-tooltip>
                  </div>
                  <div class="field-picker-item-meta">
                    <span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span>
                    <span class="field-key">{{ field.field_key }}</span>
                  </div>
                </div>
                <button v-if="!isFieldLocked(field)" class="field-picker-remove" @click.stop="unpickField(field)" style="margin-left: auto;">
                  <CloseOutlined />
                </button>
              </div>
            </div>
            <div v-else class="field-picker-empty">{{ t('settings.workbench.noFieldsSelected') }}</div>
          </div>
          <!-- 已选分页 -->
          <div class="pagination-wrapper">
            <a-pagination
              v-model:current="selectedPagination.current.value"
              v-model:page-size="selectedPagination.pageSize.value"
              :total="selectedFieldsFiltered.length"
              size="small"
              show-size-changer
              :page-size-options="['6', '20', '50']"
            />
          </div>
        </div>
      </div>
    </a-modal>

    <!-- ===== 定时任务 Tab ===== -->
    <div v-if="activeTab === 'cron'" class="tab-content">
      <div class="settings-card" style="max-width: 700px;">
        <h4 class="perm-card-title">
          <MailOutlined style="color: var(--color-primary);" />
          {{ t('settings.cron.defaultEmailTitle') }}
        </h4>
        <p class="config-section-desc" style="margin-bottom: 16px;">{{ t('settings.cron.defaultEmailDesc') }}</p>
        <a-input
          v-model:value="cronDefaultEmail"
          :placeholder="t('settings.cron.defaultEmailPlaceholder')"
          size="large"
          :disabled="cronLoading"
        >
          <template #prefix><MailOutlined class="input-icon" /></template>
        </a-input>
        <p class="config-section-desc" style="margin-top: 4px; margin-bottom: 0;">{{ t('settings.cron.multiEmailHint') }}</p>
        <div class="settings-actions" style="margin-top: 20px;">
          <a-button type="primary" size="large" :loading="saving" @click="handleSaveCron">
            <SaveOutlined v-if="!saving" />
            {{ t('common.save') }}
          </a-button>
        </div>
      </div>
    </div>

    <!-- ===== 归档复盘 Tab ===== -->
    <div v-if="activeTab === 'archive'" class="tab-content">
      <div v-if="archiveLoading && !archiveList.length" class="loading-placeholder">
        <a-spin :tip="t('common.loading')" />
      </div>
      <div v-else-if="!archiveList.length" class="settings-card">
        <a-empty :description="t('settings.archive.noProcess')" />
      </div>
      <div v-else class="workbench-layout">
        <!-- 左：归档流程列表 -->
        <div class="process-list-panel">
          <div class="process-list-header">
            <SafetyCertificateOutlined />
            <span>{{ t('settings.archive.reviewProcesses') }}</span>
          </div>
          <div
            v-for="cfg in archiveList"
            :key="cfg.process_type"
            class="process-list-item"
            :class="{ 'process-list-item--active': selectedArchiveProcessType === cfg.process_type }"
            @click="selectArchiveProcess(cfg.process_type)"
          >
            <div class="process-list-item-name">{{ cfg.process_type_label || cfg.process_type }}</div>
            <div v-if="cfg.process_type_label" class="process-list-item-path">{{ cfg.process_type }}</div>
          </div>
        </div>

        <!-- 右：归档配置详情 -->
        <div v-if="fullArchiveConfig && !archiveLoading" class="process-config-panel">
          <h3 class="config-title">{{ fullArchiveConfig.process_type_label || fullArchiveConfig.process_type }} — {{ t('settings.archive.personalReviewConfig') }}</h3>

          <!-- 子导航 -->
          <div class="section-nav">
            <button
              v-for="sec in [
                { key: 'fields', label: t('settings.archive.fieldsTab'), icon: AppstoreOutlined },
                { key: 'rules', label: t('settings.archive.rulesTab'), icon: AuditOutlined },
                { key: 'ai', label: t('settings.archive.aiTab'), icon: ControlOutlined },
              ]"
              :key="sec.key"
              class="section-nav-btn"
              :class="{ 'section-nav-btn--active': archiveSection === sec.key }"
              @click="archiveSection = sec.key as any"
            >
              <component :is="sec.icon" />
              {{ sec.label }}
            </button>
          </div>

          <!-- 字段部分 -->
          <div v-if="archiveSection === 'fields'" class="config-section">
            <div class="section-header-row">
              <h4 class="config-section-title">{{ t('settings.archive.fieldsTitle') }}</h4>
              <span v-if="!fullArchiveConfig.user_permissions.allow_custom_fields" class="locked-tag">
                <LockOutlined /> {{ t('settings.workbench.lockedByAdmin') }}
              </span>
            </div>
            <p class="config-section-desc">
              {{ fullArchiveConfig.field_mode === 'all' ? t('settings.archive.allFieldsMode') : t('settings.archive.selectedFieldsMode') }}
              <template v-if="fullArchiveConfig.user_permissions.allow_custom_fields && fullArchiveConfig.field_mode === 'selected'">
                {{ t('settings.workbench.canToggleFields') }}
              </template>
            </p>
            <template v-if="fullArchiveConfig.field_mode === 'selected'">
              <div class="field-picker-toolbar">
                <span class="field-count">{{ t('settings.workbench.fieldSelected', [archiveSelectedCount, archiveAllFields.length]) }}</span>
                <a-button v-if="fullArchiveConfig.user_permissions.allow_custom_fields" type="primary" size="small" @click="showArchiveFieldPicker = true; archiveFieldSearchQuery = ''">
                  <AppstoreOutlined /> {{ t('settings.workbench.selectFields') }}
                </a-button>
              </div>
                <!-- 字段列表展示 -->
                <div v-if="archiveSelectedFieldsFlat.length > 0">
                  <div v-for="group in archiveGroupedSelected" :key="group.source" class="selected-field-group">
                    <div class="field-group-label">{{ group.sourceLabel }}</div>
                    <div class="selected-fields-display">
                      <div
                        v-for="field in group.fields"
                        :key="field.field_key + '_' + field.source"
                        class="selected-field-tag"
                      >
                        <span class="selected-field-name">{{ field.field_name }}</span>
                        <span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span>
                      </div>
                    </div>
                  </div>
                  <!-- 分页控制 -->
                  <div v-if="archiveSelectedFieldsFlat.length > 24" class="pagination-wrapper" style="margin-top: 20px; border-top: none;">
                    <a-pagination
                      v-model:current="archiveDisplaySelectedPagination.current.value"
                      v-model:page-size="archiveDisplaySelectedPagination.pageSize.value"
                      :total="archiveSelectedFieldsFlat.length"
                      size="small"
                      show-size-changer
                      :page-size-options="['24', '48', '96']"
                    />
                  </div>
                </div>
                <div v-else class="text-center py-8 bg-slate-50 rounded-xl border border-dashed border-slate-200">
                  <div class="text-slate-400 text-sm">{{ t('settings.archive.noFieldsSelected') }}</div>
                </div>
            </template>
            <template v-else>
              <div class="field-count" style="margin-top: 8px;">{{ t('settings.archive.allFieldsModeDesc') }}</div>
            </template>
          </div>

          <!-- 规则部分 -->
          <div v-if="archiveSection === 'rules'" class="config-section">
            <!-- 租户规则 -->
            <div class="section-header-row">
              <h4 class="config-section-title">{{ t('settings.archive.tenantRules') }}</h4>
            </div>
            <div v-if="fullArchiveConfig.tenant_rules.length" class="rule-config-list">
              <div v-for="rule in fullArchiveConfig.tenant_rules" :key="rule.id" class="rule-config-item">
                <div class="rule-config-content">
                  <span class="rule-config-text">{{ rule.rule_content }}</span>
                  <span v-if="rule.related_flow" class="rule-flow-tag">
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
                <a-switch v-model:checked="rule.enabled" size="small" :disabled="rule.rule_scope === 'mandatory'" />
              </div>
            </div>
            <div v-else class="field-empty-hint">{{ t('settings.archive.noTenantRules') }}</div>

            <!-- 自定义规则 -->
            <div class="section-header-row" style="margin-top: 20px;">
              <h4 class="config-section-title">{{ t('settings.archive.personalRules') }}</h4>
              <span v-if="!fullArchiveConfig.user_permissions.allow_custom_rules" class="locked-tag">
                <LockOutlined /> {{ t('settings.workbench.lockedByAdmin') }}
              </span>
            </div>
            <p class="config-section-desc">
              {{ fullArchiveConfig.user_permissions.allow_custom_rules ? t('settings.archive.personalRulesAllowed') : t('settings.archive.personalRulesDenied') }}
            </p>
            <div v-if="fullArchiveConfig.custom_rules?.length" class="rule-config-list">
              <div v-for="rule in fullArchiveConfig.custom_rules" :key="rule.id" class="rule-config-item">
                <div class="rule-config-content">
                  <span class="rule-config-text">{{ rule.content }}</span>
                  <span v-if="rule.related_flow" class="rule-flow-tag">
                    <NodeIndexOutlined /> {{ t('settings.workbench.relatedFlow') }}
                  </span>
                  <span class="rule-scope-tag rule-scope-tag--custom">{{ t('settings.workbench.personal') }}</span>
                </div>
                <div class="rule-config-actions">
                  <a-switch v-model:checked="rule.enabled" size="small" />
                  <a-popconfirm v-if="fullArchiveConfig.user_permissions.allow_custom_rules" :title="t('settings.workbench.confirmDelete')" @confirm="removeArchiveCustomRule(rule.id)">
                    <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
                  </a-popconfirm>
                </div>
              </div>
            </div>
            <div v-if="fullArchiveConfig.user_permissions.allow_custom_rules" class="add-rule-row">
              <a-input
                v-model:value="newArchiveRuleContent"
                :placeholder="t('settings.archive.addRulePlaceholder')"
                @pressEnter="addArchiveCustomRule"
              />
              <a-tooltip :title="t('settings.workbench.relatedFlowTip')">
                <button class="icon-btn" :class="{ 'icon-btn--active': newArchiveRuleRelatedFlow }" @click="newArchiveRuleRelatedFlow = !newArchiveRuleRelatedFlow">
                  <NodeIndexOutlined />
                </button>
              </a-tooltip>
              <a-button type="primary" :disabled="!newArchiveRuleContent.trim()" @click="addArchiveCustomRule">
                <PlusOutlined /> {{ t('settings.workbench.add') }}
              </a-button>
            </div>
          </div>

          <!-- AI 复核尺度 -->
          <div v-if="archiveSection === 'ai'" class="config-section">
            <div class="section-header-row">
              <h4 class="config-section-title">{{ t('settings.archive.strictnessTitle') }}</h4>
              <span v-if="!fullArchiveConfig.user_permissions.allow_modify_strictness" class="locked-tag">
                <LockOutlined /> {{ t('settings.workbench.lockedByAdmin') }}
              </span>
            </div>
            <p class="config-section-desc">{{ t('settings.workbench.strictnessDesc') }}</p>
            <div class="strictness-options">
              <div
                v-for="opt in strictnessOptions"
                :key="opt.value"
                class="strictness-option"
                :class="{
                  'strictness-option--active': fullArchiveConfig.audit_strictness === opt.value,
                  'strictness-option--disabled': !fullArchiveConfig.user_permissions.allow_modify_strictness,
                }"
                @click="fullArchiveConfig.user_permissions.allow_modify_strictness && (fullArchiveConfig.audit_strictness = opt.value)"
              >
                <div class="strictness-option-radio" />
                <div>
                  <div class="strictness-option-label">{{ opt.label }}</div>
                  <div class="strictness-option-desc">{{ opt.desc }}</div>
                </div>
              </div>
            </div>
            <div style="margin-top: 20px;">
              <h4 class="config-section-title">{{ t('settings.workbench.kbMode') }}</h4>
              <p class="config-section-desc">
                {{ t('common.currentMode') }}<span style="font-weight: 600;">
                  {{ fullArchiveConfig.kb_mode === 'rules_only' ? t('settings.workbench.kbRulesOnly') : fullArchiveConfig.kb_mode === 'rag_only' ? t('settings.workbench.kbRagOnly') : t('settings.workbench.kbHybrid') }}
                </span>
                （{{ t('settings.workbench.configuredByAdmin') }}）
              </p>
            </div>
          </div>

          <div class="settings-actions">
            <a-button type="primary" size="large" :loading="saving" @click="handleSaveArchive">
              <SaveOutlined v-if="!saving" />
              {{ t('common.save') }}
            </a-button>
          </div>
        </div>

        <div v-else-if="archiveLoading" class="process-config-panel" style="display:flex;align-items:center;justify-content:center;min-height:300px;">
          <a-spin :tip="t('common.loading')" />
        </div>
        <div v-else class="process-config-empty">
          <a-empty :description="t('settings.archive.selectProcess')" />
        </div>
      </div>
    </div>

    <!-- 归档字段选择器 Modal -->
    <a-modal v-model:open="showArchiveFieldPicker" :title="t('settings.archive.fieldPickerTitle')" :width="720" :footer="null">
       <div class="field-picker-modal">
        <!-- 左侧：待选 -->
        <div class="field-picker-panel">
          <div class="field-picker-panel-header" style="justify-content: flex-start; gap: 8px;">
             <a-checkbox
              :checked="archiveLeftSelectedKeys.length === archiveUnselectedFieldsFlat.length && archiveUnselectedFieldsFlat.length > 0"
              :indeterminate="archiveLeftSelectedKeys.length > 0 && archiveLeftSelectedKeys.length < archiveUnselectedFieldsFlat.length"
              @change="toggleArchiveLeftSelectAll"
            />
            <span style="flex: 1;">{{ t('settings.workbench.availableFields') }} <span class="field-count">({{ archiveUnselectedFieldsFlat.length }})</span></span>
            <a-button type="primary" size="small" :disabled="archiveLeftSelectedKeys.length === 0" @click="batchArchivePick">
              {{ t('common.add') || '添加' }}
            </a-button>
          </div>
          <div class="field-picker-search">
            <a-input v-model:value="archiveFieldSearchQuery" :placeholder="t('settings.workbench.fieldSearchPlaceholder')" allow-clear size="small">
              <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
            </a-input>
          </div>
          <div class="field-picker-list custom-scrollbar">
            <div v-if="archiveUnselectedPagination.paged.value.length > 0" class="space-y-1">
              <div
                v-for="field in archiveUnselectedPagination.paged.value"
                :key="field.field_key + '_' + field.source"
                class="field-picker-item"
                @click="toggleArchiveLeftSelect(field.field_key + '_' + field.source)"
              >
                <div class="field-picker-item-checkbox" @click.stop="toggleArchiveLeftSelect(field.field_key + '_' + field.source)">
                  <a-checkbox :checked="archiveLeftSelectedKeys.includes(field.field_key + '_' + field.source)" />
                </div>
                <div class="field-picker-item-info">
                  <div class="field-picker-item-name">{{ field.field_name }} <span class="field-source-tag">({{ field.sourceLabel }})</span></div>
                  <div class="field-picker-item-meta">
                    <span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span>
                    <span class="field-key">{{ field.field_key }}</span>
                  </div>
                </div>
                <button class="icon-btn icon-btn--sm" @click.stop="archivePickField(field)" style="margin-left: auto;">
                  <SwapRightOutlined />
                </button>
              </div>
            </div>
            <div v-else class="field-picker-empty">
              {{ archiveFieldSearchQuery ? t('settings.workbench.noMatchField') : t('settings.workbench.allFieldsAdded') }}
            </div>
          </div>
          <!-- 待选分页 -->
          <div class="pagination-wrapper">
            <a-pagination
              v-model:current="archiveUnselectedPagination.current.value"
              v-model:page-size="archiveUnselectedPagination.pageSize.value"
              :total="archiveUnselectedFieldsFlat.length"
              size="small"
              show-size-changer
              :page-size-options="['6', '20', '50']"
            />
          </div>
        </div>

        <!-- 右侧：已选 -->
        <div class="field-picker-panel field-picker-panel--right">
          <div class="field-picker-panel-header" style="justify-content: flex-start; gap: 8px;">
            <a-checkbox
              :checked="archiveRightSelectedKeys.length === archiveSelectedFieldsFiltered.length && archiveSelectedFieldsFiltered.length > 0"
              :indeterminate="archiveRightSelectedKeys.length > 0 && archiveRightSelectedKeys.length < archiveSelectedFieldsFiltered.length"
              @change="toggleArchiveRightSelectAll"
            />
            <span style="flex: 1;">{{ t('settings.workbench.selectedFields') }} <span class="field-picker-count">{{ archiveSelectedCount }}</span></span>
            <a-button danger size="small" :disabled="archiveRightSelectedKeys.length === 0" @click="batchArchiveUnpick">
              {{ t('common.remove') }}
            </a-button>
          </div>
          <div class="field-picker-search">
            <a-input v-model:value="archiveFieldSelectedSearchQuery" :placeholder="t('settings.workbench.fieldSearchPlaceholder')" allow-clear size="small">
              <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
            </a-input>
          </div>
          <div class="field-picker-list custom-scrollbar">
            <div v-if="archiveSelectedPagination.paged.value.length > 0" class="space-y-1">
              <div
                v-for="field in archiveSelectedPagination.paged.value"
                :key="field.field_key + '_' + field.source"
                class="field-picker-item field-picker-item--selected"
                @click="toggleArchiveRightSelect(field.field_key + '_' + field.source)"
              >
                <div class="field-picker-item-checkbox" @click.stop="toggleArchiveRightSelect(field.field_key + '_' + field.source)">
                  <a-checkbox :checked="archiveRightSelectedKeys.includes(field.field_key + '_' + field.source)" :disabled="isArchiveFieldLocked(field)" />
                </div>
                <div class="field-picker-item-info">
                  <div class="flex items-center gap-1.5">
                    <span class="field-picker-item-name">{{ field.field_name }} <span class="field-source-tag">({{ field.sourceLabel }})</span></span>
                    <a-tooltip v-if="isArchiveFieldLocked(field)" :title="t('settings.workbench.fieldLocked') || '租户预设或已保存字段，不可删除'">
                      <LockOutlined style="font-size: 10px; color: var(--color-text-tertiary);" />
                    </a-tooltip>
                  </div>
                  <div class="field-picker-item-meta">
                    <span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span>
                    <span class="field-key">{{ field.field_key }}</span>
                  </div>
                </div>
                <button v-if="!isArchiveFieldLocked(field)" class="field-picker-remove" @click.stop="archiveUnpickField(field)" style="margin-left: auto;">
                  <CloseOutlined />
                </button>
              </div>
            </div>
            <div v-else class="field-picker-empty">{{ t('settings.workbench.noFieldsSelected') }}</div>
          </div>
          <!-- 已选分页 -->
          <div class="pagination-wrapper">
            <a-pagination
              v-model:current="archiveSelectedPagination.current.value"
              v-model:page-size="archiveSelectedPagination.pageSize.value"
              :total="archiveSelectedFieldsFiltered.length"
              size="small"
              show-size-changer
              :page-size-options="['6', '20', '50']"
            />
          </div>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<style scoped>
.page-header { margin-bottom: 24px; }
.page-title { font-size: 24px; font-weight: 700; color: var(--color-text-primary); margin: 0; }
.page-subtitle { font-size: 14px; color: var(--color-text-tertiary); margin: 4px 0 0; }

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

.field-type-tag {
  font-size: 10px; font-weight: 600; padding: 1px 6px; border-radius: var(--radius-sm);
  background: var(--color-bg-hover); color: var(--color-text-tertiary);
}

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
.icon-btn--sm { width: 24px; height: 24px; font-size: 12px; }

.field-count { font-size: 12px; color: var(--color-text-tertiary); font-weight: normal; }
.field-source-tag { font-size: 11px; color: var(--color-text-tertiary); font-weight: normal; margin-left: 4px; }

.pagination-wrapper { padding: 12px 16px; border-top: 1px solid var(--color-border-light); display: flex; justify-content: center; }

.add-rule-row { display: flex; gap: 8px; }
.add-rule-row :deep(.ant-btn-primary) { font-weight: 600; min-width: 80px; }
.add-rule-row :deep(.ant-btn-primary[disabled]) { background: var(--color-primary); opacity: 0.5; color: #fff; }

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

.loading-placeholder { display: flex; align-items: center; justify-content: center; padding: 80px; }

.perm-card-title {
  font-size: 15px; font-weight: 600; color: var(--color-text-primary);
  margin: 0 0 16px; display: flex; align-items: center; gap: 8px;
}
.perm-info-row { display: flex; align-items: center; gap: 12px; margin-bottom: 10px; }
.perm-info-label { font-size: 13px; font-weight: 500; color: var(--color-text-secondary); min-width: 72px; }
.perm-info-value { font-size: 13px; color: var(--color-text-primary); }
.perm-role-badge {
  font-size: 12px; font-weight: 600; padding: 2px 12px; border-radius: var(--radius-full);
  background: var(--color-primary-bg); color: var(--color-primary);
}
.perm-role-badges { display: flex; flex-wrap: wrap; gap: 4px; }
.perm-pages-section { display: flex; align-items: flex-start; gap: 12px; margin-top: 12px; }
.perm-page-tags { display: flex; flex-wrap: wrap; gap: 6px; }
.perm-page-tag {
  font-size: 11px; padding: 2px 10px; border-radius: var(--radius-sm);
  background: var(--color-bg-hover); color: var(--color-text-secondary); font-weight: 500;
}
.perm-hint-text {
  font-size: 12px; color: var(--color-text-tertiary); margin: 14px 0 0;
  padding-top: 12px; border-top: 1px solid var(--color-border-light);
}

.language-options { display: flex; gap: 12px; flex-wrap: wrap; }
.language-option {
  display: flex; align-items: center; gap: 12px; padding: 16px 24px;
  border-radius: var(--radius-lg); border: 2px solid var(--color-border);
  cursor: pointer; transition: all 0.2s ease; min-width: 180px; position: relative;
}
.language-option:hover { border-color: var(--color-primary-light); background: var(--color-bg-hover); }
.language-option--active { border-color: var(--color-primary); background: var(--color-primary-bg); box-shadow: 0 0 0 3px rgba(79,70,229,0.1); }
.language-flag { font-size: 24px; }
.language-label { font-size: 15px; font-weight: 600; color: var(--color-text-primary); }
.language-check { color: var(--color-primary); font-size: 16px; margin-left: auto; }

.settings-divider { height: 1px; background: var(--color-border-light); margin: 24px 0; }

.password-strength { margin-top: 8px; }
.strength-bar { height: 4px; background: var(--color-bg-hover); border-radius: 2px; overflow: hidden; margin-bottom: 4px; }
.strength-fill { height: 100%; border-radius: 2px; transition: width 0.3s ease, background 0.3s ease; }
.strength-label { font-size: 12px; font-weight: 500; }
.security-info { margin-top: 16px; }
.security-info-row { display: flex; align-items: center; gap: 12px; padding: 8px 0; }
.login-history-list { display: flex; flex-direction: column; gap: 8px; margin-top: 12px; }
.login-history-item {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 14px; border-radius: var(--radius-md); background: var(--color-bg-hover);
}
.login-history-item:hover { background: var(--color-bg-active); }
.login-history-time { font-size: 13px; font-weight: 500; color: var(--color-text-primary); }
.login-history-details { font-size: 12px; color: var(--color-text-tertiary); display: flex; align-items: center; gap: 6px; }
.login-history-sep { opacity: 0.4; }

.field-group-label {
  font-size: 13px; font-weight: 600; color: var(--color-text-secondary);
  margin: 12px 0 8px; padding-left: 4px; border-left: 3px solid var(--color-primary);
}
.rule-flow-tag {
  display: inline-flex; align-items: center; gap: 4px; font-size: 11px; font-weight: 500;
  padding: 1px 8px; border-radius: var(--radius-full); background: var(--color-info-bg); color: var(--color-info);
}

.field-picker-toolbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 14px; }
.selected-fields-display { display: flex; flex-wrap: wrap; gap: 8px; }
.selected-field-tag {
  display: inline-flex; align-items: center; gap: 6px; padding: 6px 12px;
  border-radius: var(--radius-md); background: var(--color-primary-bg);
  border: 1px solid var(--color-primary-lighter); font-size: 13px; color: var(--color-text-primary);
}
.selected-field-name { font-weight: 500; }
.selected-field-group { margin-bottom: 12px; }
.field-empty-hint {
  padding: 24px; text-align: center; color: var(--color-text-tertiary);
  font-size: 13px; background: var(--color-bg-hover); border-radius: var(--radius-md);
}
.field-count { font-size: 13px; color: var(--color-text-secondary); }

.field-picker-modal {
  display: grid; grid-template-columns: 1fr 1fr; gap: 16px; min-height: 480px; margin-top: 12px;
}
.field-picker-panel {
  border: 1px solid var(--color-border-light); border-radius: var(--radius-lg);
  display: flex; flex-direction: column; overflow: hidden; background: #fff;
}
.field-picker-panel--right { background: var(--color-bg-page); }
.field-picker-panel-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 12px 16px; background: var(--color-bg-hover);
  font-size: 13px; font-weight: 600; color: var(--color-text-primary);
  border-bottom: 1px solid var(--color-border-light);
}
.field-picker-count {
  font-size: 11px; font-weight: 500; padding: 2px 8px;
  border-radius: var(--radius-full); background: var(--color-primary-bg); color: var(--color-primary);
}
.field-picker-search { padding: 10px; border-bottom: 1px solid var(--color-border-light); }
.field-picker-list { flex: 1; overflow-y: auto; padding: 8px; max-height: 320px; }
.field-picker-item {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 12px; border-radius: var(--radius-md); cursor: pointer;
  transition: all var(--transition-fast); gap: 12px; border: 1px solid transparent;
}
.field-picker-item:hover { background: var(--color-primary-bg); border-color: var(--color-primary-lighter); }
.field-picker-item--selected { cursor: default; }
.field-picker-item--selected:hover { background: #fff; border-color: transparent; }
.field-picker-item-name { font-size: 13px; font-weight: 500; color: var(--color-text-primary); }
.field-picker-item-meta { display: flex; align-items: center; gap: 6px; margin-top: 2px; }
.field-picker-item-info { flex: 1; min-width: 0; }
.field-key { font-size: 11px; color: var(--color-text-tertiary); }
.field-picker-arrow { color: var(--color-primary); font-size: 14px; opacity: 0; transition: opacity 0.2s; }
.field-picker-item:hover .field-picker-arrow { opacity: 1; }
.field-picker-remove {
  width: 24px; height: 24px; border: none; background: transparent;
  border-radius: var(--radius-sm); cursor: pointer; display: flex; align-items: center;
  justify-content: center; color: var(--color-text-tertiary); font-size: 12px;
  transition: all var(--transition-fast);
}
.field-picker-remove:hover { background: var(--color-danger-bg); color: var(--color-danger); }
.field-picker-empty { padding: 48px 16px; text-align: center; color: var(--color-text-tertiary); font-size: 13px; }

@media (max-width: 768px) {
  .form-row { grid-template-columns: 1fr; }
  .workbench-layout { grid-template-columns: 1fr; }
  .tab-nav { width: 100%; overflow-x: auto; -webkit-overflow-scrolling: touch; scrollbar-width: none; }
  .tab-nav::-webkit-scrollbar { display: none; }
  .tab-btn { flex-shrink: 0; padding: 8px 14px; font-size: 13px; }
  .section-nav { width: 100%; overflow-x: auto; -webkit-overflow-scrolling: touch; scrollbar-width: none; }
  .section-nav::-webkit-scrollbar { display: none; }
  .section-nav-btn { flex-shrink: 0; }
  .settings-card { padding: 16px; }
  .process-config-panel { padding: 16px; }
  .strictness-options { gap: 6px; }
  .strictness-option { padding: 10px 12px; }
  .add-rule-row { flex-direction: column; }
  .add-rule-row .ant-btn { width: 100%; }
  .rule-config-item { flex-wrap: wrap; gap: 8px; padding: 8px 10px; }
  .perm-info-row { flex-direction: column; align-items: flex-start; gap: 4px; }
  .perm-pages-section { flex-direction: column; gap: 8px; }
  .field-picker-modal { grid-template-columns: 1fr; }
}
@media (max-width: 480px) {
  .page-title { font-size: 20px; }
  .tab-btn { padding: 6px 10px; font-size: 12px; gap: 4px; }
  .profile-avatar-section { flex-direction: column; text-align: center; }
  .settings-card { padding: 14px; }
}
</style>
