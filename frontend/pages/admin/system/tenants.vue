<script setup lang="ts">
definePageMeta({ middleware: 'auth', layout: 'default' })

import {useI18n} from '~/composables/useI18n'
import {usePagination} from '~/composables/usePagination'
import {writeClipboardText} from '~/utils/clipboard'
import {generateStrongPassword} from '~/utils/password'
import {
  ClockCircleOutlined,
  CopyOutlined,
  DatabaseOutlined,
  DeleteOutlined,
  EditOutlined,
  ExclamationCircleOutlined,
  GlobalOutlined,
  InfoCircleOutlined,
  KeyOutlined,
  LinkOutlined,
  LockOutlined,
  MailOutlined,
  PhoneOutlined,
  PlusOutlined,
  RobotOutlined,
  SafetyCertificateOutlined,
  TeamOutlined,
  ThunderboltOutlined,
  UserOutlined,
  DownloadOutlined,
} from '@ant-design/icons-vue'
import {message} from 'ant-design-vue'

const { t } = useI18n()

const {
  listTenants: fetchTenants, createTenant: apiCreateTenant, updateTenant: apiUpdateTenant,
  deleteTenant: apiDeleteTenant, getTenantStats: apiGetTenantStats,
  listOAConnections, listAIModels, listTenantMembers: apiListTenantMembers,
  rotateTenantEmbedToken: apiRotateTenantEmbedToken,
} = useSystemApi()

interface TenantData {
  id: string; name: string; code: string; description: string; status: string
  embed_enabled: boolean; embed_token_configured: boolean; embed_token_hint: string; embed_token_rotated_at: string
  sso_basic_enabled: boolean; sso_basic_password_set: boolean
  sso_basic_allowed_ips: string; sso_basic_allowed_domains: string
  oa_db_connection_id: string; token_quota: number; token_used: number; max_concurrency: number
  primary_model_id: string; fallback_model_id: string
  max_tokens_per_request: number; temperature: number; timeout_seconds: number; retry_count: number
  log_retention_days: number; data_retention_days: number
  contact_name: string; contact_email: string; contact_phone: string
  admin_user_id: string
  created_at: string; updated_at: string
}

interface TenantMember {
  id: string; username: string; display_name: string; email: string; phone: string
  department_name: string; role_names: string[]; position: string; status: string; created_at: string
}

const tenants = ref<TenantData[]>([])
const loading = ref(false)
const selectedTenant = ref<TenantData | null>(null)
const showCreate = ref(false)
const showDetail = ref(false)
const detailActiveTab = ref('basic')
const chatAllocRef = ref<{ save: (silent?: boolean) => Promise<void> } | null>(null)
const ssoBasicPassword = ref('')
const ssoPasswordVisible = ref(false)
const ssoEndpoint = computed(() => process.client
  ? `${window.location.origin}/api/auth/sso/basic-redirection`
  : '/api/auth/sso/basic-redirection')
const ssoUsernameExample = computed(() => `${selectedTenant.value?.code || ''}/<OA loginid>`)

const copyText = async (text: string, successMessage = t('common.copySuccess')) => {
  if (!text) return
  try {
    await writeClipboardText(text)
    message.success(successMessage)
  } catch {
    message.error(t('common.copyFailed'))
  }
}

const copySSOEndpoint = () => {
  return copyText(ssoEndpoint.value, t('admin.tenants.ssoEndpointCopied'))
}

const generateSSOPassword = () => {
  try {
    ssoBasicPassword.value = generateStrongPassword()
    ssoPasswordVisible.value = true
    message.success(t('admin.tenants.ssoPasswordGenerated'))
  } catch {
    message.error(t('admin.tenants.ssoPasswordGenerateFailed'))
  }
}

// 后端获取的 OA 连接 & AI 模型
const oaConnections = ref<any[]>([])
const aiModels = ref<any[]>([])

// 租户成员列表（详情抽屉中使用）
const tenantMembers = ref<TenantMember[]>([])
const membersLoading = ref(false)
const { paged: pagedMembers, current: memberPage, pageSize: memberPageSize, total: memberTotal, onChange: onMemberPageChange } = usePagination(tenantMembers, 20)

// 租户统计数据（成员数、部门数、角色数）
const tenantStatsMap = ref<Record<string, { member_count: number; department_count: number; role_count: number }>>({})

const loadAllTenantStats = async () => {
  const results = await Promise.allSettled(
    tenants.value.map(t => apiGetTenantStats(t.id))
  )
  results.forEach((r, i) => {
    if (r.status === 'fulfilled' && r.value) {
      tenantStatsMap.value[tenants.value[i].id] = r.value
    }
  })
}

const getTenantStat = (tenantId: string, key: 'member_count' | 'department_count' | 'role_count') => {
  return tenantStatsMap.value[tenantId]?.[key] ?? '-'
}

// 页面初始化：并行加载租户列表、OA 连接和 AI 模型数据
onMounted(async () => {
  loading.value = true
  try {
    const [tenantData, oaData, aiData] = await Promise.all([
      fetchTenants(),
      listOAConnections(),
      listAIModels(),
    ])
    tenants.value = tenantData
    oaConnections.value = oaData
    aiModels.value = aiData
    // 加载完租户列表后并行获取统计
    loadAllTenantStats()
  } catch (e) {
    message.error('加载数据失败')
  } finally {
    loading.value = false
  }
})

// 租户配置下拉列表的可用 AI 模型（只显示已启用的）
const availableModels = computed(() => aiModels.value.filter(m => m.enabled))

// 系统设置中可用的 OA 数据库连接（只显示已启用的）
const availableOADbs = computed(() => oaConnections.value.filter(c => c.enabled))

// 通过id获取OA DB连接名称
const getOADbName = (id: string) => {
  const conn = oaConnections.value.find(c => c.id === id)
  return conn ? conn.name : t('admin.tenants.notConfigured')
}

const getOADbInfo = (id: string) => {
  return oaConnections.value.find(c => c.id === id) || null
}



// ===== 新增租户 - 分页签表单 =====
const createTab = ref<'basic' | 'admin' | 'ai'>('basic')

const newTenant = ref({
  name: '',
  code: '',
  oa_db_connection_id: '',
  token_quota: 10000,
  max_concurrency: 10,
  description: '',
  primary_model_id: '',
  fallback_model_id: '',
  max_tokens_per_request: 8192,
  temperature: 0.3,
  timeout_seconds: 60,
  retry_count: 3,
  // 管理员信息
  admin_username: '',
  admin_display_name: '',
  admin_password: '',
  admin_email: '',
  admin_phone: '',
  admin_dept_name: '',
})

const resetNewTenant = () => {
  newTenant.value = {
    name: '', code: '', oa_db_connection_id: '', token_quota: 10000, max_concurrency: 10,
    description: '', primary_model_id: '', fallback_model_id: '',
    max_tokens_per_request: 8192, temperature: 0.3, timeout_seconds: 60, retry_count: 3,
    admin_username: '', admin_display_name: '', admin_password: '',
    admin_email: '', admin_phone: '', admin_dept_name: '',
  }
  createTab.value = 'basic'
}

const validateCreateForm = (): boolean => {
  // 基本信息校验
  if (!newTenant.value.name.trim()) {
    createTab.value = 'basic'
    message.warning(t('admin.tenants.fillRequired'))
    return false
  }
  // 租户编码校验（如果手动填写）
  if (newTenant.value.code.trim()) {
    const codeRegex = /^[a-zA-Z0-9_]+$/
    if (!codeRegex.test(newTenant.value.code)) {
      createTab.value = 'basic'
      message.warning(t('admin.tenants.codeFormatError'))
      return false
    }
  }
  // 管理员校验
  if (!newTenant.value.admin_username.trim() || !newTenant.value.admin_display_name.trim() || !newTenant.value.admin_dept_name.trim()) {
    createTab.value = 'admin'
    message.warning(t('admin.tenants.adminRequired'))
    return false
  }
  const usernameRegex = /^[a-zA-Z0-9_]{1,100}$/
  if (!usernameRegex.test(newTenant.value.admin_username)) {
    createTab.value = 'admin'
    message.warning(t('admin.org.usernameFormatError'))
    return false
  }
  if (newTenant.value.admin_email.trim()) {
    const emailRegex = /^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$/
    if (!emailRegex.test(newTenant.value.admin_email)) {
      createTab.value = 'admin'
      message.warning(t('admin.org.emailFormatError'))
      return false
    }
  }
  if (newTenant.value.admin_phone.trim()) {
    const phoneRegex = /^\d{11}$/
    if (!phoneRegex.test(newTenant.value.admin_phone)) {
      createTab.value = 'admin'
      message.warning(t('admin.org.phoneFormatError'))
      return false
    }
  }
  return true
}

const createTenant = async () => {
  if (!validateCreateForm()) return
  try {
    const d = newTenant.value
    const created = await apiCreateTenant({
      name: d.name,
      code: d.code || undefined,
      oa_db_connection_id: d.oa_db_connection_id || undefined,
      token_quota: d.token_quota,
      max_concurrency: d.max_concurrency,
      primary_model_id: d.primary_model_id || undefined,
      fallback_model_id: d.fallback_model_id || undefined,
      max_tokens_per_request: d.max_tokens_per_request,
      temperature: d.temperature,
      timeout_seconds: d.timeout_seconds,
      retry_count: d.retry_count,
      description: d.description,
      admin_username: d.admin_username,
      admin_display_name: d.admin_display_name,
      admin_password: d.admin_password || undefined,
      admin_email: d.admin_email || undefined,
      admin_phone: d.admin_phone || undefined,
      admin_dept_name: d.admin_dept_name || undefined,
    })
    tenants.value.push(created)
    showCreate.value = false
    message.success(t('admin.tenants.createSuccess'))
    resetNewTenant()
    openDetail(created)
  } catch (e: any) {
    message.error(e.message || '创建租户失败')
  }
}

// 记忆租户自定义设置的 Token 配额，避免切换“不限制”后丢失原有配置值
const previousQuotaMap = ref<Record<string, number>>({})
const previousNewTenantQuota = ref(100000)

const handleToggleUnlimitedQuota = (checked: any) => {
  if (!selectedTenant.value) return
  if (Boolean(checked)) {
    if (selectedTenant.value.token_quota > 0) {
      previousQuotaMap.value[selectedTenant.value.id] = selectedTenant.value.token_quota
    }
    selectedTenant.value.token_quota = -1
  } else {
    const remembered = previousQuotaMap.value[selectedTenant.value.id]
    selectedTenant.value.token_quota = (remembered && remembered > 0) ? remembered : 100000
  }
}

const openDetail = async (tenant: TenantData) => {
  selectedTenant.value = { ...tenant }
  if (tenant.token_quota > 0) {
    previousQuotaMap.value[tenant.id] = tenant.token_quota
  }
  detailActiveTab.value = 'basic'
  ssoBasicPassword.value = ''
  ssoPasswordVisible.value = false
  rotatedEmbedToken.value = ''
  manualEmbedToken.value = ''
  embedScriptPlatform.value = 'pc'
  embedScriptTarget.value = 'all'
  showDetail.value = true
  // 加载成员列表 & 刷新统计
  loadTenantMembers(tenant.id)
  apiGetTenantStats(tenant.id).then(s => {
    if (s) tenantStatsMap.value[tenant.id] = s
  }).catch(() => {})
}


const loadTenantMembers = async (tenantId: string) => {
  membersLoading.value = true
  try {
    tenantMembers.value = await apiListTenantMembers(tenantId)
  } catch {
    tenantMembers.value = []
  } finally {
    membersLoading.value = false
  }
}

const saveTenantDetail = async () => {
  if (!selectedTenant.value) return
  // 联系人邮箱校验
  if (selectedTenant.value.contact_email.trim()) {
    const emailRegex = /^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$/
    if (!emailRegex.test(selectedTenant.value.contact_email)) {
      detailActiveTab.value = 'basic'
      message.warning(t('admin.org.emailFormatError'))
      return
    }
  }
  // 联系人手机号校验
  if (selectedTenant.value.contact_phone.trim()) {
    const phoneRegex = /^\d{11}$/
    if (!phoneRegex.test(selectedTenant.value.contact_phone)) {
      detailActiveTab.value = 'basic'
      message.warning(t('admin.org.phoneFormatError'))
      return
    }
  }
  if (selectedTenant.value.sso_basic_enabled && !selectedTenant.value.sso_basic_password_set && !ssoBasicPassword.value) {
    detailActiveTab.value = 'sso'
    message.warning(t('admin.tenants.ssoPasswordRequired'))
    return
  }
  if (ssoBasicPassword.value && ssoBasicPassword.value.length < 8) {
    detailActiveTab.value = 'sso'
    message.warning(t('admin.tenants.ssoPasswordLength'))
    return
  }
  try {
    const s = selectedTenant.value
    const updated = await apiUpdateTenant(s.id, {
      name: s.name,
      description: s.description,
      status: s.status,
      embed_enabled: s.embed_enabled,
      sso_basic_enabled: s.sso_basic_enabled,
      ...(ssoBasicPassword.value ? { sso_basic_password: ssoBasicPassword.value } : {}),
      sso_basic_allowed_ips: s.sso_basic_allowed_ips,
      sso_basic_allowed_domains: s.sso_basic_allowed_domains,
      oa_db_connection_id: s.oa_db_connection_id || null,
      token_quota: s.token_quota,
      max_concurrency: s.max_concurrency,
      primary_model_id: s.primary_model_id || null,
      fallback_model_id: s.fallback_model_id || null,
      max_tokens_per_request: s.max_tokens_per_request,
      temperature: s.temperature,
      timeout_seconds: s.timeout_seconds,
      retry_count: s.retry_count,
      log_retention_days: s.log_retention_days,
      data_retention_days: s.data_retention_days,
      contact_name: s.contact_name,
      contact_email: s.contact_email,
      contact_phone: s.contact_phone,
    })
    await chatAllocRef.value?.save(true)
    const idx = tenants.value.findIndex(t => t.id === s.id)
    if (idx >= 0) tenants.value[idx] = { ...tenants.value[idx], ...updated }
    showDetail.value = false
    message.success(t('admin.tenants.saveSuccess'))
  } catch (e: any) {
    message.error(e.message || '保存失败')
  }
}

const toggleTenantStatus = async (id: string) => {
  const tVal = tenants.value.find(x => x.id === id)
  if (!tVal) return
  const newStatus = tVal.status === 'active' ? 'inactive' : 'active'
  try {
    await apiUpdateTenant(id, { status: newStatus })
    tVal.status = newStatus
    message.success(newStatus === 'active' ? t('admin.tenants.enabled') : t('admin.tenants.disabled'))
  } catch (e: any) {
    message.error(e.message || '操作失败')
  }
}

const isUnlimitedQuota = (total: number) => total < 0
const getQuotaPercent = (used: number, total: number) => total <= 0 ? 0 : Math.round((used / total) * 100)

const getQuotaColor = (percent: number) => {
  if (percent >= 90) return '#ef4444'
  if (percent >= 70) return '#f59e0b'
  return '#10b981'
}

const formatDateTime = (iso: string) => {
  return formatDateTimeInAppZone(iso, 'zh-CN')
}

// ===== 删除租户 =====
const showDeleteConfirm = ref(false)
const deletingTenant = ref<TenantData | null>(null)
const deletePassword = ref('')
const deleting = ref(false)
const rotatingEmbedToken = ref(false)
const showEmbedTokenModal = ref(false)
const rotatedEmbedToken = ref('')
const manualEmbedToken = ref('')
const embedScriptTarget = ref<'all' | 'audit' | 'summary'>('all')
const embedScriptPlatform = ref<'pc' | 'mobile'>('pc')

const embedOrigin = computed(() => {
  if (import.meta.client && window.location?.origin) return window.location.origin
  return ''
})
const embedAuditUrl = computed(() => embedOrigin.value ? `${embedOrigin.value}/embed/audit` : '/embed/audit')
const embedSummaryUrl = computed(() => embedOrigin.value ? `${embedOrigin.value}/embed/summary` : '/embed/summary')
const effectiveEmbedToken = computed(() => manualEmbedToken.value.trim() || rotatedEmbedToken.value.trim())
const embedScriptTargetOptions = computed(() => [
  { value: 'all', label: t('admin.tenants.embedScriptTargetAll') },
  { value: 'audit', label: t('admin.tenants.embedScriptTargetAudit') },
  { value: 'summary', label: t('admin.tenants.embedScriptTargetSummary') },
])
const embedScriptPlatformOptions = computed(() => [
  { value: 'pc', label: t('admin.tenants.embedScriptPlatformPC') },
  { value: 'mobile', label: t('admin.tenants.embedScriptPlatformMobile') },
])

const getEmbedScriptConfig = (target: 'all' | 'audit' | 'summary') => {
  if (target === 'audit') {
    return {
      label: t('admin.tenants.embedScriptTargetAudit'),
      filename: 'aura-embed-audit-notify.js',
      urls: [embedAuditUrl.value],
      iframeIds: ['aura-embed-audit'],
    }
  }
  if (target === 'summary') {
    return {
      label: t('admin.tenants.embedScriptTargetSummary'),
      filename: 'aura-embed-summary-notify.js',
      urls: [embedSummaryUrl.value],
      iframeIds: ['aura-embed-summary'],
    }
  }
  return {
    label: t('admin.tenants.embedScriptTargetAll'),
    filename: 'aura-embed-notify.js',
    urls: [embedAuditUrl.value, embedSummaryUrl.value],
    iframeIds: ['aura-embed-audit', 'aura-embed-summary'],
  }
}

const getEmbedMobileScriptConfig = (target: 'all' | 'audit' | 'summary') => {
  if (target === 'summary') {
    return {
      label: `${t('admin.tenants.embedScriptPlatformMobile')} - ${t('admin.tenants.embedScriptTargetSummary')}`,
      filename: 'aura-embed-summary-mobile-notify.js',
      urls: [embedSummaryUrl.value],
      embedType: 'summary',
    }
  }
  if (target === 'audit') {
    return {
      label: `${t('admin.tenants.embedScriptPlatformMobile')} - ${t('admin.tenants.embedScriptTargetAudit')}`,
      filename: 'aura-embed-audit-mobile-notify.js',
      urls: [embedAuditUrl.value],
      embedType: 'audit',
    }
  }
  return {
    label: `${t('admin.tenants.embedScriptPlatformMobile')} - ${t('admin.tenants.embedScriptTargetAll')}`,
    filename: 'aura-embed-mobile-notify.js',
    urls: [embedAuditUrl.value, embedSummaryUrl.value],
    embedType: 'all',
  }
}

const buildEmbedNotifyScript = (target: 'all' | 'audit' | 'summary', token: string) => {
  const cfg = getEmbedScriptConfig(target)
  const urlComment = cfg.urls.map(url => ` * - ${url}`).join('\n')
  return `/**
 * 泛微 Ecology9 — AuraOA 嵌入页：传递打开上下文，并在点击保存、提交时安排后台检查
 *
 * 导出范围：${cfg.label}
 * 对应嵌入地址：
${urlComment}
 *
 * 使用步骤：
 * 1. 将本文件上传到 OA 静态目录（如 /oa-front/workflow/AuraOA/${cfg.filename}）
 * 2. 流程 → 基础设置 → 自定义页面 → 填入 js 路径并启用
 * 3. 表单/门户 HTML 中 iframe 的 id 与 IFRAME_IDS 一致
 */
(function () {
  console.log('[aura-embed] 脚本已加载');

  // ========== AuraOA 已配置项 ==========
  var AURA_EMBED_ORIGIN = ${JSON.stringify(embedOrigin.value)};
  var EMBED_ACCESS_TOKEN = ${JSON.stringify(token)};
  var IFRAME_IDS = ${JSON.stringify(cfg.iframeIds)};
  // ====================================

  var MSG_REQUEST = 'aura-oa-request-requestid';
  var MSG_REQUESTID = 'aura-oa-requestid';

  function getRequestId() {
    try {
      if (typeof WfForm !== 'undefined' && WfForm.getBaseInfo) {
        var base = WfForm.getBaseInfo();
        var requestid = base && base.requestid != null ? String(base.requestid).trim() : '';
        if (requestid &&
            requestid !== '-1' &&
            requestid !== '0' &&
            requestid.toLowerCase() !== 'null' &&
            requestid.toLowerCase() !== 'undefined') {
          return requestid;
        }
      }
    } catch (e) {
      console.warn('[aura-embed] WfForm.getBaseInfo 失败', e);
    }
    return '';
  }

  function captureOperationContext(action) {
    var occurredAtMs = Date.now();
    var base = WfForm.getBaseInfo() || {};
    return {
      action: action,
      event_id: createEventId(),
      occurred_at_ms: occurredAtMs,
      requestid: getRequestId(),
      workflow_id: base.workflowid != null ? String(base.workflowid).trim() : '',
      oa_current_user_id: getCurrentUserId()
    };
  }

  function getIframes() {
    var seen = {};
    var ids = Array.isArray(IFRAME_IDS) && IFRAME_IDS.length ? IFRAME_IDS : [];
    return ids.map(function (id) {
      if (!id || seen[id]) return null;
      seen[id] = true;
      return document.getElementById(id);
    }).filter(Boolean);
  }

  function getCurrentUserId() {
    try {
      if (typeof WfForm !== 'undefined' && WfForm.getBaseInfo) {
        var base = WfForm.getBaseInfo() || {};
        if (base.f_weaver_belongto_userid != null) {
          return String(base.f_weaver_belongto_userid).trim();
        }
      }
    } catch (e) {}
    return '';
  }

  function buildPayload(requestid) {
    return {
      type: MSG_REQUESTID,
      requestid: requestid,
      embed_token: EMBED_ACCESS_TOKEN,
      oa_current_user_id: getCurrentUserId()
    };
  }

  function postContextToAura(win) {
    if (!win) return;
    var requestid = getRequestId();
    if (!EMBED_ACCESS_TOKEN) {
      console.warn('[aura-embed] 未配置 EMBED_ACCESS_TOKEN');
      return;
    }
    win.postMessage(buildPayload(requestid), AURA_EMBED_ORIGIN);
  }

  function notifyAuraIframes() {
    getIframes().forEach(function (iframe) {
      if (iframe && iframe.contentWindow) {
        postContextToAura(iframe.contentWindow);
      }
    });
  }

  function createEventId() {
    return 'oa-' + Date.now() + '-' + Math.random().toString(16).slice(2);
  }

  function buildEventBody(context) {
    return [
      ['embed_token', EMBED_ACCESS_TOKEN],
      ['process_id', context.requestid],
      ['workflow_id', context.workflow_id],
      ['oa_current_user_id', context.oa_current_user_id],
      ['occurred_at_ms', String(context.occurred_at_ms)],
      ['action', context.action],
      ['event_id', context.event_id]
    ].map(function (item) {
      return encodeURIComponent(item[0]) + '=' + encodeURIComponent(item[1] || '');
    }).join('&');
  }

  function notifyBeforeRelease(action, callback) {
    var released = false;
    var timeoutId = null;
    var release = function () {
      if (released) return;
      released = true;
      callback();
    };
    var context = captureOperationContext(action);
    console.log('[aura-embed] OA 操作事件已触发', {
      action: context.action,
      requestid: context.requestid || '(待解析)',
      workflow_id: context.workflow_id,
      event_id: context.event_id
    });
    timeoutId = setTimeout(function () {
      console.warn('[aura-embed] OA 操作事件提交超时，已放行 OA', {
        action: context.action,
        event_id: context.event_id
      });
      release();
    }, 800);
    try {
      var request = fetch(AURA_EMBED_ORIGIN + '/api/embed/events', {
        method: 'POST',
        mode: 'no-cors',
        credentials: 'omit',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8'
        },
        body: buildEventBody(context)
      });
      Promise.resolve(request).then(function () {
        clearTimeout(timeoutId);
        console.log('[aura-embed] OA 操作事件已提交', {
          action: context.action,
          requestid: context.requestid || '(待解析)',
          workflow_id: context.workflow_id,
          event_id: context.event_id
        });
        release();
      }, function (error) {
        clearTimeout(timeoutId);
        console.warn('[aura-embed] OA 操作事件提交失败，已放行 OA', error);
        release();
      });
    } catch (e) {
      clearTimeout(timeoutId);
      console.warn('[aura-embed] OA 操作事件提交失败，已放行 OA', e);
      release();
    }
  }

  function registerOAEvents() {
    WfForm.registerCheckEvent(WfForm.OPER_SAVE, function (callback) {
      notifyBeforeRelease('save_requested', callback);
    });
    WfForm.registerCheckEvent(WfForm.OPER_SUBMIT, function (callback) {
      notifyBeforeRelease('submit_requested', callback);
    });
    console.log('[aura-embed] 已注册 OA 保存/提交事件');
  }

  function initMessageListener() {
    window.addEventListener('message', function (event) {
      if (event.origin !== AURA_EMBED_ORIGIN) return;
      if (!event.data) return;
      if (event.data.type !== MSG_REQUEST) return;

      var requestid = getRequestId();
      if (!EMBED_ACCESS_TOKEN) {
        console.warn('[aura-embed] 未配置 EMBED_ACCESS_TOKEN');
        return;
      }
      if (event.source) {
        event.source.postMessage(buildPayload(requestid), event.origin);
      }
    });
  }

  function init() {
    console.log('[aura-embed] 脚本已初始化');
    initMessageListener();

    getIframes().forEach(function (iframe) {
      iframe.addEventListener('load', notifyAuraIframes);
    });

    var contextTries = 0;
    var contextTimer = setInterval(function () {
      contextTries++;
      notifyAuraIframes();
      if (getRequestId() || contextTries >= 200) {
        clearInterval(contextTimer);
      }
    }, 300);
    registerOAEvents();

    window.addEventListener('hashchange', function () {
      notifyAuraIframes();
    });
    window.addEventListener('popstate', function () {
      notifyAuraIframes();
    });
  }

  jQuery().ready(function () {
    init();
  });
})();
`
}

const buildEmbedMobileNotifyScript = (target: 'all' | 'audit' | 'summary', token: string) => {
  const cfg = getEmbedMobileScriptConfig(target)
  const isSummary = target === 'summary'
  const isAll = target === 'all'
  const defaultEmbedType = isSummary ? 'summary' : (isAll ? 'all' : 'audit')
  return `/**
 * 泛微 Ecology9 移动端 — AuraOA 嵌入脚本（状态胶囊按钮 + 弹窗查看详情 + 变更感知）
 *
 * 导出范围：${cfg.label}
 * 对应嵌入地址：
${cfg.urls.map(url => ` * - ${url}`).join('\n')}
 *
 * 使用步骤：
 * 1. 将本文件上传到 OA 静态目录（如 /oa-front/workflow/AuraOA/${cfg.filename}）
 * 2. 流程 → 基础设置 → 自定义页面（或移动端页面设置）填入 js 路径并启用
 * 3. 表单设计中可添加自定义 HTML 块 <div id="getMyBt"></div> 作为按钮挂载位（如无则自动以左下角悬浮方式挂载）
 */
(function () {
  console.log('[aura-embed-mobile] 移动端脚本已加载');

  // ========== AuraOA 已配置项 ==========
  var AURA_EMBED_ORIGIN = ${JSON.stringify(embedOrigin.value)};
  var EMBED_ACCESS_TOKEN = ${JSON.stringify(token)};
  var SHOW_ON_DESKTOP = false; // 是否在电脑端启用本脚本的按钮和弹窗；移动端不受影响
  var AUTO_RUN_BEFORE_OPEN = false; // true=进入表单即自动审/总结（仍受「打开即审」等配置约束）；false=点开详情才审（默认）
  var BUTTON_CONTAINER_ID = 'getMyBt';
  var EMBED_TYPE = ${JSON.stringify(defaultEmbedType)};
  // ====================================

  var MSG_STATUS = 'aura-oa-embed-status';
  var STATUS_POLL_MS = 3000;
  var STATUS_WATCH_MS = 180000;
  var BUTTON_BUSY_MIN_MS = 480; // 忙碌态至少停留，避免「加载中」未看清就跳到结果
  var STATUS_MOTION_CLASSES = [
    'aura-status-button--enter',
    'aura-status-button--loading',
    'aura-status-button--running',
    'aura-status-button--enter-shake',
    'aura-status-button--enter-pop',
    'aura-status-button--enter-bounce',
    'aura-status-button--enter-rise',
    'aura-status-button--enter-fade',
    'aura-status-button--morph',
    'aura-status-button--shine'
  ];

  // 不按窗口宽度判断，避免电脑端窄侧栏被识别为移动端；兼容 iPad 桌面 UA。
  var mobileClient = /Android|iPhone|iPad|iPod|Windows Phone/i.test(navigator.userAgent || '')
    || (/Macintosh/i.test(navigator.userAgent || '') && navigator.maxTouchPoints > 1);
  if (!mobileClient && !SHOW_ON_DESKTOP) return;

  // 动画样式仅在真实浏览器拥有 document.head 时注入
  try {
    if (typeof document !== 'undefined' && document.head && document.head.appendChild && (!document.getElementById || !document.getElementById('aura-btn-pulse-style'))) {
      var style = document.createElement('style');
      style.id = 'aura-btn-pulse-style';
      style.textContent =
        '@keyframes auraBtnSpin{to{transform:rotate(360deg)}}' +
        '@keyframes auraDraw{to{stroke-dashoffset:0}}' +
        '@keyframes auraButtonEnter{0%{opacity:0;transform:translate3d(-10px,6px,0) scale(.78);filter:blur(2px)}55%{opacity:1;transform:translate3d(1px,-1px,0) scale(1.035);filter:blur(0)}100%{opacity:1;transform:none;filter:none}}' +
        '@keyframes auraButtonEnterSoft{0%{opacity:0;transform:translate3d(-8px,5px,0) scale(.9);filter:blur(1px)}100%{opacity:1;transform:none;filter:none}}' +
        '@keyframes auraIconEnter{0%{opacity:0;transform:scale(.64) rotate(-10deg)}72%{opacity:1;transform:scale(1.08) rotate(3deg)}100%{opacity:1;transform:scale(1) rotate(0)}}' +
        '@keyframes auraTextEnter{0%{opacity:0;transform:translate3d(-5px,0,0)}100%{opacity:1;transform:none}}' +
        '@keyframes auraBtnShine{0%,100%{opacity:0;transform:translateX(-120%)}18%{opacity:.48}42%{opacity:0;transform:translateX(120%)}}' +
        '@keyframes auraStatusBreath{0%,100%{opacity:.22;transform:scale(.985)}50%{opacity:.48;transform:scale(1.01)}}' +
        '@keyframes auraStarSpin{from{transform:rotate(0deg)}to{transform:rotate(360deg)}}' +
        '@keyframes auraIconSwapOut{to{opacity:0;transform:scale(.62) rotate(-12deg)}}' +
        '@keyframes auraIconSwapIn{0%{opacity:0;transform:scale(.62) rotate(12deg)}70%{opacity:1;transform:scale(1.08) rotate(-2deg)}100%{opacity:1;transform:none}}' +
        '@keyframes auraCopySwapOut{to{opacity:0;transform:translateY(8px);filter:blur(2px)}}' +
        '@keyframes auraCopySwapIn{from{opacity:0;transform:translateY(-7px);filter:blur(2px)}to{opacity:1;transform:none;filter:none}}' +
        '@keyframes auraArrowIn{from{opacity:0;transform:translateX(-4px)}to{opacity:1;transform:none}}' +
        '@keyframes auraArrowOut{to{opacity:0;transform:translateX(4px)}}' +
        '.aura-status-button{box-sizing:border-box;appearance:none;-webkit-appearance:none;position:relative;isolation:isolate;overflow:hidden;display:inline-flex;align-items:center;gap:9px;min-height:40px;max-width:100%;padding:6px 11px 6px 7px;margin:0;border:1px solid var(--aura-status-border,rgba(148,163,184,.22));border-radius:14px;background:var(--aura-status-surface,#fff);color:#263247;box-shadow:var(--aura-status-shadow,0 4px 16px rgba(15,23,42,.08),0 1px 3px rgba(15,23,42,.04));font:600 13px/1.4 -apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;letter-spacing:.1px;text-align:left;cursor:pointer;touch-action:manipulation;-webkit-tap-highlight-color:transparent;transition:background-color .45s ease,border-color .45s ease,box-shadow .45s ease,color .22s ease,width .52s cubic-bezier(.22,.7,.2,1),transform .18s ease;}' +
        '.aura-status-button::before{content:"";position:absolute;inset:1px;border-radius:13px;background:radial-gradient(circle at 8% 50%,var(--aura-status-tint),transparent 62%);opacity:.22;pointer-events:none;z-index:0;transform-origin:8% 50%;animation:auraStatusBreath 4.2s ease-in-out infinite;transition:opacity .45s ease;}' +
        '.aura-status-button::after{content:"";position:absolute;top:0;bottom:0;left:0;width:36%;background:linear-gradient(105deg,transparent,rgba(255,255,255,.66),transparent);pointer-events:none;z-index:0;opacity:0;transform:translateX(-120%);}' +
        '.aura-status-button--enter::after,.aura-status-button--shine::after{animation:auraBtnShine 3.2s ease-out .18s both;}' +
        '.aura-status-icon,.aura-status-copy,.aura-status-text,.aura-status-score,.aura-status-arrow{position:relative;z-index:1;}' +
        '.aura-status-icon{display:flex;align-items:center;justify-content:center;width:28px;height:28px;flex:0 0 28px;border-radius:9px;background:var(--aura-status-tint);color:var(--aura-status-color);overflow:hidden;transition:background-color .45s ease,color .45s ease;}' +
        '.aura-status-icon svg{display:block;width:18px;height:18px;transform-origin:center;}' +
        '.aura-status-icon-layer{display:flex;align-items:center;justify-content:center;width:28px;height:28px;}' +
        '.aura-status-icon-layer svg{display:block;width:18px;height:18px;transform-origin:center;}' +
        '.aura-status-icon-layer--out{position:absolute;inset:0;animation:auraIconSwapOut .3s ease both;}' +
        '.aura-status-icon-layer--in{animation:auraIconSwapIn .42s cubic-bezier(.22,.7,.2,1) both;}' +
        '.aura-status-copy{display:flex;align-items:center;min-width:0;overflow:hidden;}' +
        '.aura-status-copy-inner{display:inline-flex;align-items:center;gap:9px;white-space:nowrap;}' +
        '.aura-status-copy-inner--out{position:absolute;left:0;top:0;bottom:0;animation:auraCopySwapOut .32s ease both;pointer-events:none;}' +
        '.aura-status-copy-inner--in{animation:auraCopySwapIn .4s cubic-bezier(.22,.7,.2,1) .04s both;}' +
        '.aura-status-text{min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;}' +
        '.aura-status-score{display:inline-flex;align-items:baseline;gap:2px;flex:none;padding-left:10px;border-left:1px solid #e8edf3;color:var(--aura-status-color);font-variant-numeric:tabular-nums;transition:color .45s ease;}' +
        '.aura-status-score b{font-size:17px;font-weight:700;line-height:1;}' +
        '.aura-status-score small{font-size:10px;font-weight:500;}' +
        '.aura-status-arrow{display:block;width:14px;height:14px;flex:0 0 14px;color:#94a3b8;}' +
        '.aura-status-arrow--in{animation:auraArrowIn .32s ease .08s both;}' +
        '.aura-status-arrow--out{animation:auraArrowOut .24s ease both;}' +
        '.aura-status-button:focus-visible{outline:2px solid var(--aura-status-color);outline-offset:3px;}' +
        '.aura-status-button:active{transform:scale(.98);}' +
        '.aura-status-button--morph{pointer-events:none;}' +
        '.aura-status-button[aria-disabled="true"]{cursor:default;}' +
        '.aura-status-button[aria-disabled="true"]:not(.aura-status-button--loading){color:#64748b;box-shadow:0 1px 4px rgba(15,23,42,.05);}' +
        '.aura-status-button[aria-disabled="true"]:not(.aura-status-button--loading)::before{animation:none;opacity:.12;}' +
        '.aura-status-button--enter{animation:auraButtonEnter .62s cubic-bezier(.22,.7,.2,1) both;}' +
        '.aura-status-button--enter .aura-status-icon{opacity:0;transform:scale(.76);animation:auraIconEnter .38s cubic-bezier(.22,.7,.2,1) .36s both;}' +
        '.aura-status-button--enter .aura-status-copy,.aura-status-button--enter .aura-status-text{opacity:0;transform:translate3d(-5px,0,0);animation:auraTextEnter .34s ease-out .53s both;}' +
        '.aura-status-button--enter .aura-status-score{opacity:0;transform:translate3d(-4px,0,0);animation:auraTextEnter .34s ease-out .6s both;}' +
        '.aura-status-button--enter .aura-status-arrow{opacity:0;transform:translate3d(-3px,0,0);animation:auraTextEnter .34s ease-out .66s both;}' +
        '.aura-status-button--enter.aura-status-button--loading,.aura-status-button--enter.aura-status-button--running{animation:auraButtonEnterSoft .4s cubic-bezier(.22,.7,.2,1) both;}' +
        '.aura-status-button--enter.aura-status-button--loading .aura-status-icon,.aura-status-button--enter.aura-status-button--running .aura-status-icon,.aura-status-button--enter.aura-status-button--loading .aura-status-copy,.aura-status-button--enter.aura-status-button--running .aura-status-copy,.aura-status-button--enter.aura-status-button--loading .aura-status-text,.aura-status-button--enter.aura-status-button--running .aura-status-text{opacity:1;transform:none;animation:none;}' +
        '.aura-status-button--enter.aura-status-button--enter-shake .aura-status-icon svg>*,.aura-status-button--enter.aura-status-button--enter-pop .aura-status-icon svg>*,.aura-status-button--enter.aura-status-button--enter-bounce .aura-status-icon svg>*,.aura-status-button--enter.aura-status-button--enter-rise .aura-status-icon svg>*,.aura-status-button--enter.aura-status-button--enter-fade .aura-status-icon svg>*,.aura-status-button--morph.aura-status-button--enter-shake .aura-status-icon-layer--in svg>*,.aura-status-button--morph.aura-status-button--enter-pop .aura-status-icon-layer--in svg>*,.aura-status-button--morph.aura-status-button--enter-bounce .aura-status-icon-layer--in svg>*,.aura-status-button--morph.aura-status-button--enter-rise .aura-status-icon-layer--in svg>*,.aura-status-button--morph.aura-status-button--enter-fade .aura-status-icon-layer--in svg>*{stroke-dasharray:1;stroke-dashoffset:1}' +
        '.aura-status-button--loading .aura-status-icon>svg,.aura-status-button--loading .aura-status-icon-layer--in svg{animation:auraBtnSpin .9s linear infinite}' +
        '.aura-status-button--running .aura-status-icon>svg .aura-running-star,.aura-status-button--running .aura-status-icon-layer--in .aura-running-star{transform-box:fill-box;transform-origin:center;animation:auraStarSpin 1.6s linear infinite}' +
        '.aura-status-button--enter.aura-status-button--enter-shake .aura-status-icon svg>*,.aura-status-button--morph.aura-status-button--enter-shake .aura-status-icon-layer--in svg>*{animation:auraDraw .34s cubic-bezier(.22,.7,.2,1) .08s forwards}' +
        '.aura-status-button--enter.aura-status-button--enter-shake .aura-status-icon svg>:nth-child(2),.aura-status-button--morph.aura-status-button--enter-shake .aura-status-icon-layer--in svg>:nth-child(2){animation-delay:.22s}' +
        '.aura-status-button--enter.aura-status-button--enter-pop .aura-status-icon svg>*,.aura-status-button--morph.aura-status-button--enter-pop .aura-status-icon-layer--in svg>*{animation:auraDraw .48s cubic-bezier(.22,.72,.2,1) .08s forwards}' +
        '.aura-status-button--enter.aura-status-button--enter-bounce .aura-status-icon svg>*,.aura-status-button--morph.aura-status-button--enter-bounce .aura-status-icon-layer--in svg>*{animation:auraDraw .36s cubic-bezier(.22,.7,.2,1) .08s forwards}' +
        '.aura-status-button--enter.aura-status-button--enter-bounce .aura-status-icon svg>:nth-child(2),.aura-status-button--morph.aura-status-button--enter-bounce .aura-status-icon-layer--in svg>:nth-child(2){animation-delay:.22s}' +
        '.aura-status-button--enter.aura-status-button--enter-rise .aura-status-icon svg>*,.aura-status-button--morph.aura-status-button--enter-rise .aura-status-icon-layer--in svg>*{animation:auraDraw .32s cubic-bezier(.22,.7,.2,1) .08s forwards}' +
        '.aura-status-button--enter.aura-status-button--enter-rise .aura-status-icon svg>:nth-child(2),.aura-status-button--morph.aura-status-button--enter-rise .aura-status-icon-layer--in svg>:nth-child(2){animation-delay:.18s}' +
        '.aura-status-button--enter.aura-status-button--enter-rise .aura-status-icon svg>:nth-child(3),.aura-status-button--morph.aura-status-button--enter-rise .aura-status-icon-layer--in svg>:nth-child(3){animation-delay:.26s}' +
        '.aura-status-button--enter.aura-status-button--enter-rise .aura-status-icon svg>:nth-child(4),.aura-status-button--morph.aura-status-button--enter-rise .aura-status-icon-layer--in svg>:nth-child(4){animation-delay:.34s}' +
        '.aura-status-button--enter.aura-status-button--enter-fade .aura-status-icon svg>*,.aura-status-button--morph.aura-status-button--enter-fade .aura-status-icon-layer--in svg>*{animation:auraDraw .4s cubic-bezier(.22,.7,.2,1) .08s forwards}' +
        '.aura-status-button--enter.aura-status-button--enter-fade .aura-status-icon svg>:nth-child(2),.aura-status-button--morph.aura-status-button--enter-fade .aura-status-icon-layer--in svg>:nth-child(2){animation-delay:.2s}' +
        '.aura-status-sr{position:absolute;width:1px;height:1px;padding:0;margin:-1px;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border:0;}' +
        '.aura-status-group{display:inline-flex;max-width:100%;align-items:center;gap:8px;flex-wrap:wrap;padding:4px 0;}' +
        '#auraMobileEmbedFloatContainer{position:fixed;bottom:16px;bottom:calc(16px + env(safe-area-inset-bottom,0px));left:16px;left:calc(16px + env(safe-area-inset-left,0px));max-width:calc(100vw - 32px);z-index:9999;}' +
        '@media(hover:hover){.aura-status-button:not([aria-disabled="true"]):not(.aura-status-button--morph):hover{transform:translateY(-2px);border-color:var(--aura-status-color);box-shadow:0 7px 22px rgba(15,23,42,.12);}}' +
        '@media(pointer:coarse){.aura-status-button{min-height:44px;}}' +
        '@media(prefers-reduced-motion:reduce){.aura-status-button{transition:none;animation:none!important;}.aura-status-button::before,.aura-status-button::after,.aura-status-icon,.aura-status-copy,.aura-status-text,.aura-status-score,.aura-status-arrow,.aura-status-icon-layer,.aura-status-copy-inner{animation:none!important;opacity:1!important;transform:none!important;filter:none!important;}.aura-status-icon svg,.aura-status-icon svg>*{animation:none!important;stroke-dashoffset:0!important;}}';
      document.head.appendChild(style);
    }
  } catch (e) {}

  var statusConfig = {
    gray: { color: '#475569', text: 'AI审核详情', bg: '#eef2f7', surface: '#f8fafc', border: '#cbd5e1', dot: '#94a3b8', shadow: '0 2px 8px rgba(15,23,42,.06)' },
    green: { color: '#15803d', text: '审核通过', bg: '#e5f7ed', surface: '#f1fbf5', border: '#86efac', dot: '#16a36a', shadow: '0 3px 12px rgba(22,163,106,.13)' },
    yellow: { color: '#a15c00', text: '建议关注', bg: '#fff0d2', surface: '#fff9ee', border: '#f2c078', dot: '#d98200', shadow: '0 3px 12px rgba(217,130,0,.14)' },
    red: { color: '#c62828', text: '建议退回', bg: '#ffe8e6', surface: '#fff3f2', border: '#ef8a84', dot: '#d92d20', shadow: '0 4px 14px rgba(198,40,40,.16)' },
    disabled: { color: '#94a3b8', text: '暂不可用', bg: '#f1f4f8', surface: '#f8fafc', border: '#e2e8f0', dot: '#cbd5e1', shadow: '0 1px 4px rgba(15,23,42,.05)' },
    error: { color: '#b42318', text: '加载失败', bg: '#ffe1df', surface: '#fff1f0', border: '#e97870', dot: '#d92d20', shadow: '0 4px 16px rgba(180,35,24,.2)' },
    loading: { color: '#1d4ed8', text: '加载中...', bg: '#dbeafe', surface: '#eef5ff', border: '#60a5fa', dot: '#2563eb', shadow: '0 3px 12px rgba(37,99,235,.15)' }
  };

  // 单功能、双功能及电脑端共用按钮结构：首次进入播放入场，后续在同一颗胶囊上变形。
  function escapeButtonText(value) {
    return String(value || '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
  }

  function prefersReducedMotion() {
    try {
      return !!(window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches);
    } catch (e) {
      return false;
    }
  }

  function isBusyStatus(status, isLoading) {
    return !!isLoading || status === 'loading';
  }

  function parseButtonLabel(displayText) {
    var text = String(displayText || '');
    var scoreMatch = text.match(/ [(]([0-9]+)([^()]*)[)]$/);
    return {
      label: scoreMatch ? text.slice(0, scoreMatch.index) : text,
      scoreMatch: scoreMatch
    };
  }

  function buttonTheme(status, isLoading) {
    var fetching = !!isLoading;
    var running = !fetching && status === 'loading';
    return (fetching || running) ? statusConfig.loading : (statusConfig[status] || statusConfig.gray);
  }

  function iconMarkup(feature, status, isLoading) {
    var fetching = !!isLoading;
    var running = !fetching && status === 'loading';
    var iconPath = '<path pathLength="1" d="m12 3 2.4 6.6L21 12l-6.6 2.4L12 21l-2.4-6.6L3 12l6.6-2.4Z"/>';
    if (feature === 'summary') iconPath = '<path pathLength="1" d="M14 3H6v18h12V7Z" stroke-width="2.4"/><path pathLength="1" d="M14 3v5h4" stroke-width="2.4"/><path pathLength="1" d="M9 12h6" stroke-width="2.4"/><path pathLength="1" d="M9 16h4" stroke-width="2.4"/>';
    if (status === 'green' && feature === 'audit') iconPath = '<path pathLength="1" d="m6 12 4 4 8-8" stroke-width="2.8"/>';
    if (status === 'red' || status === 'error') iconPath = '<path pathLength="1" d="M8 8l8 8" stroke-width="2.7"/><path pathLength="1" d="M16 8l-8 8" stroke-width="2.7"/>';
    if (status === 'yellow') iconPath = '<path pathLength="1" d="M12 4.1v9.4" stroke-width="3.6"/><path pathLength="1" d="M12 18.25v.02" stroke-width="4.4"/>';
    if (status === 'disabled') iconPath = '<path pathLength="1" d="M8 10V7a4 4 0 0 1 8 0v3" stroke-width="2.4"/><rect pathLength="1" x="5" y="10" width="14" height="11" rx="3" stroke-width="2.4"/>';
    if (running) iconPath = '<path class="aura-running-star" d="m12 3 2.4 6.6L21 12l-6.6 2.4L12 21l-2.4-6.6L3 12l6.6-2.4Z"/>';
    if (fetching) iconPath = '<circle cx="12" cy="12" r="8.1" fill="none" stroke="currentColor" stroke-width="2.7" opacity="0.22"/><path d="M12 3.9a8.1 8.1 0 0 1 0 16.2" fill="none" stroke="currentColor" stroke-width="2.8" stroke-linecap="round"/>';
    return iconPath;
  }

  function iconSvgMarkup(feature, status, isLoading) {
    return '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">' + iconMarkup(feature, status, isLoading) + '</svg>';
  }

  function copyInnerMarkup(label, scoreMatch) {
    return '<span class="aura-status-text">' + escapeButtonText(label) + '</span>' +
      (scoreMatch ? '<span class="aura-status-score"><b>' + escapeButtonText(scoreMatch[1]) + '</b><small>' + escapeButtonText(scoreMatch[2]) + '</small></span>' : '');
  }

  function arrowMarkup() {
    return '<svg class="aura-status-arrow" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m6 4 4 4-4 4"/></svg>';
  }

  function statusMotionClass(status, isLoading, isEnter) {
    var prefix = isEnter ? ' aura-status-button--enter' : '';
    if (isLoading) return prefix + ' aura-status-button--loading';
    if (status === 'loading') return prefix + ' aura-status-button--running';
    if (status === 'red' || status === 'error') return prefix + ' aura-status-button--enter-shake';
    if (status === 'green') return prefix + ' aura-status-button--enter-pop';
    if (status === 'yellow') return prefix + ' aura-status-button--enter-bounce';
    if (status === 'disabled') return prefix + ' aura-status-button--enter-fade';
    return prefix + ' aura-status-button--enter-rise';
  }

  function applyButtonTheme(btn, theme) {
    btn.style.setProperty('--aura-status-color', theme.color);
    btn.style.setProperty('--aura-status-tint', theme.bg);
    btn.style.setProperty('--aura-status-surface', theme.surface);
    btn.style.setProperty('--aura-status-border', theme.border);
    btn.style.setProperty('--aura-status-shadow', theme.shadow);
  }

  function syncMotionClass(btn, status, isLoading, isEnter) {
    var i;
    for (i = 0; i < STATUS_MOTION_CLASSES.length; i++) {
      btn.classList.remove(STATUS_MOTION_CLASSES[i]);
    }
    var parts = statusMotionClass(status, isLoading, isEnter).replace(/^\s+/, '').split(/\s+/);
    for (i = 0; i < parts.length; i++) {
      if (parts[i]) btn.classList.add(parts[i]);
    }
  }

  function buttonInnerHtml(feature, status, displayText, isLoading, labelClass) {
    var parsed = parseButtonLabel(displayText);
    var unavailable = status === 'disabled' || status === 'error';
    return '<span class="aura-status-icon" aria-hidden="true">' + iconSvgMarkup(feature, status, isLoading) + '</span>' +
      '<span class="aura-status-copy" aria-hidden="true"><span class="aura-status-copy-inner">' + copyInnerMarkup(parsed.label, parsed.scoreMatch) + '</span></span>' +
      (!unavailable && !isLoading ? arrowMarkup() : '') +
      '<span class="aura-status-sr ' + labelClass + '">' + escapeButtonText(displayText) + '</span>';
  }

  function buildStatusButton(btnId, feature, status, displayText, isLoading, labelClass, isEnter) {
    var theme = buttonTheme(status, isLoading);
    var unavailable = status === 'disabled' || status === 'error';
    var enter = isEnter !== false;
    return '<button id="' + btnId + '" type="button" class="aura-status-button' + statusMotionClass(status, isLoading, enter) + '" aria-disabled="' + (unavailable || isLoading ? 'true' : 'false') + '" style="--aura-status-color:' + theme.color + ';--aura-status-tint:' + theme.bg + ';--aura-status-surface:' + theme.surface + ';--aura-status-border:' + theme.border + ';--aura-status-shadow:' + theme.shadow + ';">' +
      buttonInnerHtml(feature, status, displayText, isLoading, labelClass) +
    '</button>';
  }

  function createButtonPainter(paint) {
    var timer = null;
    var next = null;
    var shownAt = 0;
    var busy = false;
    function flush() {
      timer = null;
      var payload = next;
      next = null;
      if (!payload) return;
      paint(payload);
      busy = isBusyStatus(payload.status, payload.isLoading);
      if (busy) shownAt = Date.now();
    }
    return function (payload) {
      next = payload;
      if (timer) return;
      var wait = 0;
      if (busy && !prefersReducedMotion()) {
        wait = Math.max(0, BUTTON_BUSY_MIN_MS - (Date.now() - shownAt));
      }
      if (wait === 0) {
        flush();
        return;
      }
      timer = setTimeout(flush, wait);
    };
  }

  function ensureStatusGroup() {
    var $container = jQuery('#' + BUTTON_CONTAINER_ID);
    if (!$container.length) {
      if (!jQuery('#auraMobileEmbedFloatContainer').length) {
        jQuery('body').append('<div id="auraMobileEmbedFloatContainer"></div>');
      }
      $container = jQuery('#auraMobileEmbedFloatContainer');
    }
    var group = $container.children('.aura-status-group')[0];
    if (!group) {
      $container.html('<div class="aura-status-group"></div>');
      group = $container.children('.aura-status-group')[0];
    }
    return group;
  }

  function finishButtonMorph(btn) {
    var icon = btn.querySelector('.aura-status-icon');
    var copy = btn.querySelector('.aura-status-copy');
    var inIcon = icon && icon.querySelector('.aura-status-icon-layer--in');
    var inCopy = copy && copy.querySelector('.aura-status-copy-inner--in');
    if (inIcon) icon.innerHTML = inIcon.innerHTML;
    if (inCopy) copy.innerHTML = '<span class="aura-status-copy-inner">' + inCopy.innerHTML + '</span>';
    var outArrow = btn.querySelector('.aura-status-arrow--out');
    if (outArrow && outArrow.parentNode) outArrow.parentNode.removeChild(outArrow);
    var inArrow = btn.querySelector('.aura-status-arrow--in');
    if (inArrow) inArrow.classList.remove('aura-status-arrow--in');
    btn.classList.remove('aura-status-button--morph');
    btn.style.width = '';
    btn.style.transition = '';
    btn._auraMorphTimer = null;
  }

  function morphStatusButton(btn, feature, status, displayText, isLoading, labelClass) {
    var theme = buttonTheme(status, isLoading);
    var unavailable = status === 'disabled' || status === 'error';
    var parsed = parseButtonLabel(displayText);
    var nextIcon = iconSvgMarkup(feature, status, isLoading);
    var nextCopyInner = copyInnerMarkup(parsed.label, parsed.scoreMatch);
    var showArrow = !unavailable && !isLoading;
    var iconKey = feature + '|' + status + '|' + (isLoading ? '1' : '0');
    var fromWidth = btn.getBoundingClientRect().width;

    if (btn._auraMorphTimer) {
      clearTimeout(btn._auraMorphTimer);
      finishButtonMorph(btn);
      fromWidth = btn.getBoundingClientRect().width;
    }

    applyButtonTheme(btn, theme);
    btn.setAttribute('aria-disabled', (unavailable || isLoading) ? 'true' : 'false');
    var sr = btn.querySelector('.aura-status-sr');
    if (sr) {
      sr.className = 'aura-status-sr ' + labelClass;
      sr.textContent = displayText;
    }

    if (prefersReducedMotion()) {
      syncMotionClass(btn, status, isLoading, false);
      btn.innerHTML = buttonInnerHtml(feature, status, displayText, isLoading, labelClass);
      btn.setAttribute('data-aura-icon', iconKey);
      btn.setAttribute('data-aura-copy', displayText);
      return;
    }

    var icon = btn.querySelector('.aura-status-icon');
    var copy = btn.querySelector('.aura-status-copy');
    if (!icon || !copy) {
      syncMotionClass(btn, status, isLoading, true);
      btn.innerHTML = buttonInnerHtml(feature, status, displayText, isLoading, labelClass);
      btn.setAttribute('data-aura-icon', iconKey);
      btn.setAttribute('data-aura-copy', displayText);
      return;
    }

    var iconChanged = btn.getAttribute('data-aura-icon') !== iconKey;
    var copyChanged = btn.getAttribute('data-aura-copy') !== displayText;
    btn.setAttribute('data-aura-icon', iconKey);
    btn.setAttribute('data-aura-copy', displayText);

    if (iconChanged) {
      icon.innerHTML = '<span class="aura-status-icon-layer aura-status-icon-layer--out">' + icon.innerHTML + '</span>' +
        '<span class="aura-status-icon-layer aura-status-icon-layer--in">' + nextIcon + '</span>';
    }
    if (copyChanged) {
      var oldCopy = copy.querySelector('.aura-status-copy-inner');
      var oldCopyHtml = oldCopy ? oldCopy.innerHTML : copy.innerHTML;
      copy.innerHTML = '<span class="aura-status-copy-inner aura-status-copy-inner--out">' + oldCopyHtml + '</span>' +
        '<span class="aura-status-copy-inner aura-status-copy-inner--in">' + nextCopyInner + '</span>';
    }

    var arrow = btn.querySelector('.aura-status-arrow:not(.aura-status-arrow--out)');
    if (showArrow && !arrow) {
      var holder = document.createElement('div');
      holder.innerHTML = arrowMarkup();
      var newArrow = holder.firstChild;
      newArrow.classList.add('aura-status-arrow--in');
      btn.insertBefore(newArrow, sr || null);
    } else if (!showArrow && arrow) {
      arrow.classList.add('aura-status-arrow--out');
    }

    syncMotionClass(btn, status, isLoading, false);
    btn.classList.add('aura-status-button--morph');
    btn.classList.remove('aura-status-button--shine');
    void btn.offsetWidth;
    btn.classList.add('aura-status-button--shine');

    btn.style.transition = 'background-color .45s ease,border-color .45s ease,box-shadow .45s ease,color .22s ease';
    btn.style.width = fromWidth + 'px';
    void btn.offsetWidth;
    btn.style.width = 'auto';
    var toWidth = btn.getBoundingClientRect().width;
    btn.style.width = fromWidth + 'px';
    void btn.offsetWidth;
    btn.style.transition = '';
    if (Math.abs(toWidth - fromWidth) >= 0.5) {
      btn.style.width = toWidth + 'px';
    } else {
      btn.style.width = '';
    }

    btn._auraMorphTimer = setTimeout(function () {
      finishButtonMorph(btn);
    }, 520);
  }

  function upsertStatusButton(group, btnId, feature, status, displayText, isLoading, labelClass) {
    var btn = document.getElementById(btnId);
    if (!btn || btn.parentNode !== group) {
      var wrap = document.createElement('div');
      wrap.innerHTML = buildStatusButton(btnId, feature, status, displayText, isLoading, labelClass, true);
      btn = wrap.firstChild;
      btn.setAttribute('data-aura-icon', feature + '|' + status + '|' + (isLoading ? '1' : '0'));
      btn.setAttribute('data-aura-copy', displayText);
      if (btnId === 'auraMobileEmbedAuditBtn' && group.firstChild) {
        group.insertBefore(btn, group.firstChild);
      } else {
        group.appendChild(btn);
      }
      return btn;
    }
    morphStatusButton(btn, feature, status, displayText, isLoading, labelClass);
    return btn;
  }

  function getRequestId() {
    try {
      if (typeof WfForm !== 'undefined' && WfForm.getBaseInfo) {
        var base = WfForm.getBaseInfo();
        var requestid = base && base.requestid != null ? String(base.requestid).trim() : '';
        if (requestid &&
            requestid !== '-1' &&
            requestid !== '0' &&
            requestid.toLowerCase() !== 'null' &&
            requestid.toLowerCase() !== 'undefined') {
          return requestid;
        }
      }
    } catch (e) {
      console.warn('[aura-embed-mobile] WfForm.getBaseInfo 失败', e);
    }
    return '';
  }

  function getCurrentUserId() {
    try {
      if (typeof WfForm !== 'undefined' && WfForm.getBaseInfo) {
        var base = WfForm.getBaseInfo() || {};
        if (base.f_weaver_belongto_userid != null) {
          return String(base.f_weaver_belongto_userid).trim();
        }
      }
    } catch (e) {}
    return '';
  }

  function captureOperationContext(action) {
    var occurredAtMs = Date.now();
    var base = (typeof WfForm !== 'undefined' && WfForm.getBaseInfo) ? WfForm.getBaseInfo() || {} : {};
    return {
      action: action,
      event_id: 'oa-' + Date.now() + '-' + Math.random().toString(16).slice(2),
      occurred_at_ms: occurredAtMs,
      requestid: getRequestId(),
      workflow_id: base.workflowid != null ? String(base.workflowid).trim() : '',
      oa_current_user_id: getCurrentUserId()
    };
  }

  function buildEventBody(context) {
    return [
      ['embed_token', EMBED_ACCESS_TOKEN],
      ['process_id', context.requestid],
      ['workflow_id', context.workflow_id],
      ['oa_current_user_id', context.oa_current_user_id],
      ['occurred_at_ms', String(context.occurred_at_ms)],
      ['action', context.action],
      ['event_id', context.event_id]
    ].map(function (item) {
      return encodeURIComponent(item[0]) + '=' + encodeURIComponent(item[1] || '');
    }).join('&');
  }

  var formOpenPrefetchState = {};

  // 进入表单后按需后台预审：仅当 AUTO_RUN_BEFORE_OPEN 且后端判定 should_auto_* 时发起。
  function startFormOpenPrefetch(featType, shouldAutoRun, runningJobId, onFail) {
    if (!AUTO_RUN_BEFORE_OPEN || !shouldAutoRun || runningJobId) return false;
    var requestId = getRequestId();
    if (!requestId) return false;
    var key = featType + ':' + requestId;
    var state = formOpenPrefetchState[key];
    if (state === 'inflight' || state === 'done') return true;
    if (state === 'failed') return false;
    formOpenPrefetchState[key] = 'inflight';
    var isSummary = featType === 'summary';
    var userId = getCurrentUserId();
    var apiPath = isSummary ? '/api/embed/summary/execute' : '/api/embed/execute';
    var apiUrl = AURA_EMBED_ORIGIN + apiPath
      + '?embed_token=' + encodeURIComponent(EMBED_ACCESS_TOKEN)
      + '&oa_user_id=' + encodeURIComponent(userId);
    fetch(apiUrl, {
      method: 'POST',
      credentials: 'omit',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        process_id: requestId,
        trigger_source: isSummary ? 'summary_embed_auto' : 'embed_auto',
        trigger_detail: 'form_open',
        oa_user_id: userId
      })
    }).then(function (res) {
      if (!res.ok) throw new Error('HTTP ' + res.status);
      return res.json();
    }).then(function () {
      formOpenPrefetchState[key] = 'done';
    }).catch(function (err) {
      formOpenPrefetchState[key] = 'failed';
      console.warn('[aura-embed-mobile] 进入表单预审发起失败', err);
      if (typeof onFail === 'function') onFail();
    });
    return true;
  }

  // =========================================================================
  // 单模式（audit 或 summary）
  // =========================================================================
  if (EMBED_TYPE !== 'all') {
    var statusWatchTimer = null;
    var statusWatchUntil = 0;
    var lastStatusSignature = '';
    var resumeRefreshTimer = null;
    var singleBtnRuntime = { isLoading: false, isDisabled: false, errorMessage: '', onClick: null };
    var paintSingleButton = createButtonPainter(function (payload) {
      renderButton(payload.status, payload.textOverride, payload.errorMessage, payload.isLoading, payload.onClick);
    });

    function openDesktopDialog(targetUrl) {
      var existing = document.getElementById('auraDesktopEmbedDialog');
      if (existing) return;
      var previousFocus = document.activeElement;
      var overlay = document.createElement('div');
      overlay.id = 'auraDesktopEmbedDialog';
      overlay.style.cssText = 'position:fixed;inset:0;z-index:2147483646;background:rgba(0,0,0,.45);display:flex;align-items:center;justify-content:center;padding:12px;box-sizing:border-box;';
      var dialog = document.createElement('div');
      dialog.setAttribute('role', 'dialog');
      dialog.setAttribute('aria-modal', 'true');
      dialog.setAttribute('aria-labelledby', 'auraDesktopEmbedTitle');
      dialog.style.cssText = 'width:760px;max-width:100%;height:85vh;max-height:100%;background:#fff;border-radius:12px;overflow:hidden;display:flex;flex-direction:column;box-shadow:0 16px 48px rgba(0,0,0,.2);';
      var header = document.createElement('div');
      header.style.cssText = 'display:flex;align-items:center;justify-content:space-between;gap:16px;padding:12px 16px;border-bottom:1px solid #eee;flex-shrink:0;';
      var title = document.createElement('span');
      title.id = 'auraDesktopEmbedTitle';
      title.textContent = EMBED_TYPE === 'summary' ? 'AI 流程总结' : 'AI 流程审核';
      var close = document.createElement('button');
      close.type = 'button';
      close.textContent = '关闭';
      close.style.cssText = 'cursor:pointer;padding:6px 12px;background:#fff;border:1px solid #ddd;border-radius:6px;';
      var frame = document.createElement('iframe');
      frame.title = title.textContent;
      frame.src = targetUrl;
      frame.style.cssText = 'width:100%;flex:1;min-height:0;border:0;display:block;';
      function dismiss() {
        document.removeEventListener('keydown', onKeyDown);
        overlay.remove();
        if (previousFocus && previousFocus.focus) previousFocus.focus();
        refreshCurrentStatus(true);
      }
      function onKeyDown(event) {
        if (event.key === 'Escape') dismiss();
      }
      close.onclick = dismiss;
      overlay.onclick = function (event) { if (event.target === overlay) dismiss(); };
      header.appendChild(title);
      header.appendChild(close);
      dialog.appendChild(header);
      dialog.appendChild(frame);
      overlay.appendChild(dialog);
      document.body.appendChild(overlay);
      document.addEventListener('keydown', onKeyDown);
      close.focus();
    }

    function openEmbedDialog(requestId, userId) {
      var path = EMBED_TYPE === 'summary' ? '/embed/summary' : '/embed/audit';
      var targetUrl = AURA_EMBED_ORIGIN + path
        + '?requestid=' + encodeURIComponent(requestId)
        + '&embed_token=' + encodeURIComponent(EMBED_ACCESS_TOKEN)
        + '&oa_user_id=' + encodeURIComponent(userId);

      console.log('[aura-embed-mobile] 调起嵌入弹窗');
      startStatusWatch(STATUS_WATCH_MS);

      if (!mobileClient) {
        openDesktopDialog(targetUrl);
      } else if (window.weaJs && typeof window.weaJs.showDialog === 'function') {
        window.weaJs.showDialog(targetUrl, {
          title: EMBED_TYPE === 'summary' ? 'AI 流程总结' : 'AI 流程审核',
          moduleName: 'workflow',
          style: { width: '100%', height: '100%' },
          callback: function () { refreshCurrentStatus(true); }
        });
      } else {
        window.open(targetUrl, '_blank');
      }
    }

    function renderButton(status, textOverride, errorMessage, isLoading, onClick) {
      var currentStatus = isLoading ? statusConfig.loading : (statusConfig[status] || statusConfig.gray);
      var displayText = textOverride || currentStatus.text;
      singleBtnRuntime.isLoading = !!isLoading;
      singleBtnRuntime.isDisabled = status === 'disabled' || status === 'error';
      singleBtnRuntime.errorMessage = errorMessage || '';
      singleBtnRuntime.onClick = onClick;

      var group = ensureStatusGroup();
      upsertStatusButton(group, 'auraMobileEmbedBtn', EMBED_TYPE, status, displayText, isLoading, 'aura-mobile-label');
      if (group.getAttribute('data-aura-bound') === '1') return;
      group.setAttribute('data-aura-bound', '1');
      group.addEventListener('click', function (event) {
        var target = event.target;
        while (target && target !== group && !(target.classList && target.classList.contains('aura-status-button'))) {
          target = target.parentNode;
        }
        if (!target || target === group) return;
        if (singleBtnRuntime.isLoading) {
          if (typeof WfForm !== 'undefined' && WfForm.showMessage) {
            WfForm.showMessage('数据加载中，请稍候...', 2, 2);
          }
          return;
        }
        if (singleBtnRuntime.isDisabled) {
          if (typeof WfForm !== 'undefined' && WfForm.showMessage) {
            WfForm.showMessage(singleBtnRuntime.errorMessage || '当前节点暂不可用', 2, 3);
          }
          return;
        }
        if (typeof singleBtnRuntime.onClick === 'function') {
          singleBtnRuntime.onClick();
        }
      });
    }

    function featureName() {
      return EMBED_TYPE === 'summary' ? '总结' : '审核';
    }

    function openCurrentDetails() {
      var requestId = getRequestId();
      if (!requestId) return;
      openEmbedDialog(requestId, getCurrentUserId());
    }

    function statusSignature(status, text) {
      return status + '|' + (text || '');
    }

    function renderStatusButton(status, textOverride, errorMessage, isLoading, onClick) {
      var signature = statusSignature(status, textOverride || '') + '|' + (isLoading ? '1' : '0');
      if (signature === lastStatusSignature) return;
      lastStatusSignature = signature;
      paintSingleButton({
        status: status,
        textOverride: textOverride,
        errorMessage: errorMessage,
        isLoading: isLoading,
        onClick: onClick
      });
    }

    function applyResultToButton(isSummary, result, hasResult, shouldAutoRun, runningJobId) {
      var name = featureName();
      var openDetails = openCurrentDetails;
      if (runningJobId) {
        renderStatusButton('loading', name + '分析中，查看进度', '', false, openDetails);
        return 'running';
      }
      if (hasResult && result) {
        if (result.status === 'failed' || result.status === 'cancelled' || result.parse_error) {
          renderStatusButton('red', name + '异常', 'AI ' + name + '分析出现异常', false, openDetails);
          return 'failed';
        }
        if (isSummary) {
          renderStatusButton('green', '查看流程总结', '', false, openDetails);
          return 'completed';
        }
        var rec = result.recommendation || 'review';
        var score = result.overall_score != null ? Math.round(Number(result.overall_score)) : null;
        var scoreText = score != null ? ' (' + score + '分)' : '';
        if (rec === 'approve') {
          renderStatusButton('green', '审核通过' + scoreText, '', false, openDetails);
        } else if (rec === 'return') {
          renderStatusButton('red', '建议退回' + scoreText, '', false, openDetails);
        } else {
          renderStatusButton('yellow', '建议关注' + scoreText, '', false, openDetails);
        }
        return 'completed';
      }
      if (shouldAutoRun && startFormOpenPrefetch(isSummary ? 'summary' : 'audit', true, '', function () {
        lastStatusSignature = '';
        renderStatusButton('gray', '查看并生成' + name, '', false, openDetails);
        stopStatusWatch();
      })) {
        renderStatusButton('loading', name + '分析中，查看进度', '', false, openDetails);
        return 'running';
      }
      renderStatusButton('gray', shouldAutoRun ? '查看并生成' + name : 'AI' + name + '详情', '', false, openDetails);
      return 'pending';
    }

    function applyContextData(data) {
      var isSummary = EMBED_TYPE === 'summary';
      var name = featureName();
      if (!data || typeof data.supported !== 'boolean') {
        renderStatusButton('error', '加载异常', '获取 AI ' + name + '状态失败', false);
        return 'error';
      }
      if (!data.supported) {
        renderStatusButton('disabled', '未开启' + name, data.message || '当前流程未配置 AI ' + name, false);
        return 'unsupported';
      }
      var result = isSummary ? data.summary_result : data.audit_result;
      var hasResult = isSummary ? data.has_summary : data.has_audit;
      var shouldAutoRun = isSummary ? data.should_auto_summary : data.should_auto_audit;
      return applyResultToButton(isSummary, result, hasResult, shouldAutoRun, data.running_job_id);
    }

    function stopStatusWatch() {
      if (statusWatchTimer) {
        clearInterval(statusWatchTimer);
        statusWatchTimer = null;
      }
    }

    function startStatusWatch(durationMs) {
      statusWatchUntil = Date.now() + (durationMs || STATUS_WATCH_MS);
      if (statusWatchTimer) return;
      statusWatchTimer = setInterval(function () {
        if (Date.now() > statusWatchUntil) {
          stopStatusWatch();
          return;
        }
        refreshCurrentStatus(true);
      }, STATUS_POLL_MS);
    }

    function refreshCurrentStatus(preferCached) {
      var requestId = getRequestId();
      if (!requestId) return;
      queryEmbedStatus(requestId, getCurrentUserId(), !!preferCached);
    }

    function queryEmbedStatus(requestId, userId, preferCached) {
      var isSummary = EMBED_TYPE === 'summary';
      var name = featureName();
      var apiPath = isSummary ? '/api/embed/summary/context' : '/api/embed/context';
      var apiUrl = AURA_EMBED_ORIGIN + apiPath
        + '?requestid=' + encodeURIComponent(requestId)
        + '&embed_token=' + encodeURIComponent(EMBED_ACCESS_TOKEN)
        + '&oa_user_id=' + encodeURIComponent(userId);
      if (preferCached) apiUrl += '&prefer_cached=true';

      fetch(apiUrl, { method: 'GET', credentials: 'omit' })
        .then(function (res) {
          if (!res.ok) throw new Error('HTTP ' + res.status);
          return res.json();
        })
        .then(function (res) {
          var wrapped = res && Object.prototype.hasOwnProperty.call(res, 'code');
          var data = wrapped ? res.data : res;
          if ((wrapped && res.code !== 0) || !data || typeof data.supported !== 'boolean') {
            renderStatusButton('error', '加载异常', '获取 AI ' + name + '状态失败', false);
            return;
          }
          var kind = applyContextData(data);
          if (kind === 'running') startStatusWatch(STATUS_WATCH_MS);
          else if (kind === 'completed' || kind === 'failed' || kind === 'unsupported') stopStatusWatch();
        })
        .catch(function (err) {
          console.warn('[aura-embed-mobile] 请求状态失败:', err);
          renderStatusButton('gray', '查看 AI ' + name, '', false, openCurrentDetails);
        });
    }

    function notifyBeforeRelease(action, callback) {
      var released = false;
      var release = function () {
        if (released) return;
        released = true;
        startStatusWatch(STATUS_WATCH_MS);
        callback();
      };
      var context = captureOperationContext(action);
      var timeoutId = setTimeout(function () {
        console.warn('[aura-embed-mobile] OA 操作事件提交超时，已放行 OA', { action: context.action });
        release();
      }, 800);

      try {
        var request = fetch(AURA_EMBED_ORIGIN + '/api/embed/events', {
          method: 'POST',
          mode: 'no-cors',
          credentials: 'omit',
          headers: { 'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8' },
          body: buildEventBody(context)
        });
        Promise.resolve(request).then(function () {
          clearTimeout(timeoutId);
          release();
        }, function (err) {
          clearTimeout(timeoutId);
          console.warn('[aura-embed-mobile] OA 操作事件提交失败，已放行 OA', err);
          release();
        });
      } catch (e) {
        clearTimeout(timeoutId);
        release();
      }
    }

    function registerOAEvents() {
      if (typeof WfForm === 'undefined' || !WfForm.registerCheckEvent) return;
      WfForm.registerCheckEvent(WfForm.OPER_SAVE, function (callback) {
        notifyBeforeRelease('save_requested', callback);
      });
      WfForm.registerCheckEvent(WfForm.OPER_SUBMIT, function (callback) {
        notifyBeforeRelease('submit_requested', callback);
      });
      console.log('[aura-embed-mobile] 已注册 OA 保存/提交感知事件');
    }

    function applyStatusMessage(payload) {
      var isSummary = EMBED_TYPE === 'summary';
      if (payload.running) {
        applyResultToButton(isSummary, null, false, false, 'running');
        startStatusWatch(STATUS_WATCH_MS);
        return;
      }
      var result = {
        status: payload.status || 'completed',
        parse_error: !!payload.parse_error,
        recommendation: payload.recommendation,
        overall_score: payload.overall_score
      };
      var kind = applyResultToButton(isSummary, result, !!payload.has_result, false, '');
      if (kind === 'completed' || kind === 'failed') stopStatusWatch();
    }

    function scheduleResumeRefresh() {
      if (document.visibilityState && document.visibilityState === 'hidden') return;
      if (resumeRefreshTimer) clearTimeout(resumeRefreshTimer);
      resumeRefreshTimer = setTimeout(function () {
        resumeRefreshTimer = null;
        refreshCurrentStatus(true);
      }, 300);
    }

    function bindStatusRefreshListeners() {
      window.addEventListener('message', function (event) {
        if (event.origin !== AURA_EMBED_ORIGIN) return;
        if (!event.data || event.data.type !== MSG_STATUS) return;
        if (event.data.embed_type && event.data.embed_type !== EMBED_TYPE) return;
        var requestId = getRequestId();
        if (event.data.requestid && requestId && String(event.data.requestid) !== String(requestId)) return;
        applyStatusMessage(event.data);
      });
      document.addEventListener('visibilitychange', function () {
        if (document.visibilityState === 'visible') scheduleResumeRefresh();
      });
      window.addEventListener('pageshow', scheduleResumeRefresh);
      window.addEventListener('focus', scheduleResumeRefresh);
    }

    function init() {
      bindStatusRefreshListeners();
      var requestId = getRequestId();
      var userId = getCurrentUserId();

      if (!requestId) {
        renderStatusButton('disabled', '待保存流程', '流程保存并生成编号后即可查看 AI ' + featureName(), false);
        registerOAEvents();
        return;
      }

      renderStatusButton('loading', '加载中...', '', true);
      queryEmbedStatus(requestId, userId, false);
      registerOAEvents();
    }

    jQuery().ready(function () {
      init();
    });
    return;
  }

  // =========================================================================
  // 全部功能模式（all: 挂载审核 + 总结两个状态胶囊按钮）
  // =========================================================================
  var allStatusWatchTimer = null;
  var allStatusWatchUntil = 0;
  var allResumeRefreshTimer = null;

  var dualState = {
    audit: { status: 'loading', text: '加载中...', errorMessage: '', isLoading: true, lastSig: '' },
    summary: { status: 'loading', text: '加载中...', errorMessage: '', isLoading: true, lastSig: '' }
  };

  function openDualDesktopDialog(type, targetUrl) {
    var existing = document.getElementById('auraDesktopEmbedDialog');
    if (existing) return;
    var previousFocus = document.activeElement;
    var overlay = document.createElement('div');
    overlay.id = 'auraDesktopEmbedDialog';
    overlay.style.cssText = 'position:fixed;inset:0;z-index:2147483646;background:rgba(0,0,0,.45);display:flex;align-items:center;justify-content:center;padding:12px;box-sizing:border-box;';
    var dialog = document.createElement('div');
    dialog.setAttribute('role', 'dialog');
    dialog.setAttribute('aria-modal', 'true');
    dialog.setAttribute('aria-labelledby', 'auraDesktopEmbedTitle');
    dialog.style.cssText = 'width:760px;max-width:100%;height:85vh;max-height:100%;background:#fff;border-radius:12px;overflow:hidden;display:flex;flex-direction:column;box-shadow:0 16px 48px rgba(0,0,0,.2);';
    var header = document.createElement('div');
    header.style.cssText = 'display:flex;align-items:center;justify-content:space-between;gap:16px;padding:12px 16px;border-bottom:1px solid #eee;flex-shrink:0;';
    var title = document.createElement('span');
    title.id = 'auraDesktopEmbedTitle';
    title.textContent = type === 'summary' ? 'AI 流程总结' : 'AI 流程审核';
    var close = document.createElement('button');
    close.type = 'button';
    close.textContent = '关闭';
    close.style.cssText = 'cursor:pointer;padding:6px 12px;background:#fff;border:1px solid #ddd;border-radius:6px;';
    var frame = document.createElement('iframe');
    frame.title = title.textContent;
    frame.src = targetUrl;
    frame.style.cssText = 'width:100%;flex:1;min-height:0;border:0;display:block;';
    function dismiss() {
      document.removeEventListener('keydown', onKeyDown);
      overlay.remove();
      if (previousFocus && previousFocus.focus) previousFocus.focus();
      refreshDualStatus(type, true);
    }
    function onKeyDown(event) {
      if (event.key === 'Escape') dismiss();
    }
    close.onclick = dismiss;
    overlay.onclick = function (event) { if (event.target === overlay) dismiss(); };
    header.appendChild(title);
    header.appendChild(close);
    dialog.appendChild(header);
    dialog.appendChild(frame);
    overlay.appendChild(dialog);
    document.body.appendChild(overlay);
    document.addEventListener('keydown', onKeyDown);
    close.focus();
  }

  function openDualEmbedDialog(type, requestId, userId) {
    var path = type === 'summary' ? '/embed/summary' : '/embed/audit';
    var targetUrl = AURA_EMBED_ORIGIN + path
      + '?requestid=' + encodeURIComponent(requestId)
      + '&embed_token=' + encodeURIComponent(EMBED_ACCESS_TOKEN)
      + '&oa_user_id=' + encodeURIComponent(userId);

    console.log('[aura-embed-mobile] 调起嵌入弹窗: ' + type);
    startDualStatusWatch(STATUS_WATCH_MS);

    var title = type === 'summary' ? 'AI 流程总结' : 'AI 流程审核';
    if (!mobileClient) {
      openDualDesktopDialog(type, targetUrl);
    } else if (window.weaJs && typeof window.weaJs.showDialog === 'function') {
      window.weaJs.showDialog(targetUrl, {
        title: title,
        moduleName: 'workflow',
        style: { width: '100%', height: '100%' },
        callback: function () { refreshDualStatus(type, true); }
      });
    } else {
      window.open(targetUrl, '_blank');
    }
  }

  function bindDualGroupClicks(group) {
    if (group.getAttribute('data-aura-bound') === '1') return;
    group.setAttribute('data-aura-bound', '1');
    group.addEventListener('click', function (event) {
      var target = event.target;
      while (target && target !== group && !(target.classList && target.classList.contains('aura-status-button'))) {
        target = target.parentNode;
      }
      if (!target || target === group) return;
      var featType = target.id === 'auraMobileEmbedSummaryBtn' ? 'summary' : 'audit';
      var item = dualState[featType];
      if (item.isLoading) {
        if (typeof WfForm !== 'undefined' && WfForm.showMessage) {
          WfForm.showMessage('数据加载中，请稍候...', 2, 2);
        }
        return;
      }
      if (item.status === 'disabled' || item.status === 'error') {
        if (typeof WfForm !== 'undefined' && WfForm.showMessage) {
          WfForm.showMessage(item.errorMessage || '当前流程未开启 AI ' + (featType === 'summary' ? '总结' : '审核'), 2, 3);
        }
        return;
      }
      var reqId = getRequestId();
      if (reqId) openDualEmbedDialog(featType, reqId, getCurrentUserId());
    });
  }

  function renderDualFeature(featType) {
    var item = dualState[featType];
    var currentStatus = item.isLoading ? statusConfig.loading : (statusConfig[item.status] || statusConfig.gray);
    var displayText = item.text || currentStatus.text;
    var btnId = featType === 'summary' ? 'auraMobileEmbedSummaryBtn' : 'auraMobileEmbedAuditBtn';
    var group = ensureStatusGroup();
    upsertStatusButton(group, btnId, featType, item.status, displayText, item.isLoading, 'aura-mobile-label-' + featType);
    bindDualGroupClicks(group);
  }

  var paintDualFeature = {
    audit: createButtonPainter(function () { renderDualFeature('audit'); }),
    summary: createButtonPainter(function () { renderDualFeature('summary'); })
  };

  function setDualFeatureState(featType, status, text, errorMessage, isLoading) {
    var sig = status + '|' + text + '|' + (isLoading ? '1' : '0');
    if (dualState[featType].lastSig === sig) return;
    dualState[featType].lastSig = sig;
    dualState[featType].status = status;
    dualState[featType].text = text;
    dualState[featType].errorMessage = errorMessage || '';
    dualState[featType].isLoading = !!isLoading;
    paintDualFeature[featType]({ status: status, isLoading: isLoading });
  }

  function applyDualData(featType, data) {
    var isSummary = featType === 'summary';
    var name = isSummary ? '总结' : '审核';
    if (!data || typeof data.supported !== 'boolean') {
      setDualFeatureState(featType, 'error', '加载异常', '获取 AI ' + name + '状态失败', false);
      return 'error';
    }
    if (!data.supported) {
      setDualFeatureState(featType, 'disabled', '未开启' + name, data.message || '当前流程未配置 AI ' + name, false);
      return 'unsupported';
    }
    var result = isSummary ? data.summary_result : data.audit_result;
    var hasResult = isSummary ? data.has_summary : data.has_audit;
    var shouldAutoRun = isSummary ? data.should_auto_summary : data.should_auto_audit;
    if (data.running_job_id) {
      setDualFeatureState(featType, 'loading', name + '分析中...', '', false);
      return 'running';
    }
    if (hasResult && result) {
      if (result.status === 'failed' || result.status === 'cancelled' || result.parse_error) {
        setDualFeatureState(featType, 'red', name + '异常', 'AI ' + name + '分析出现异常', false);
        return 'failed';
      }
      if (isSummary) {
        setDualFeatureState(featType, 'green', '查看流程总结', '', false);
        return 'completed';
      }
      var rec = result.recommendation || 'review';
      var score = result.overall_score != null ? Math.round(Number(result.overall_score)) : null;
      var scoreText = score != null ? ' (' + score + '分)' : '';
      if (rec === 'approve') {
        setDualFeatureState(featType, 'green', '审核通过' + scoreText, '', false);
      } else if (rec === 'return') {
        setDualFeatureState(featType, 'red', '建议退回' + scoreText, '', false);
      } else {
        setDualFeatureState(featType, 'yellow', '建议关注' + scoreText, '', false);
      }
      return 'completed';
    }
    if (shouldAutoRun && startFormOpenPrefetch(featType, true, '', function () {
      dualState[featType].lastSig = '';
      setDualFeatureState(featType, 'gray', '生成' + name, '', false);
    })) {
      setDualFeatureState(featType, 'loading', name + '分析中...', '', false);
      return 'running';
    }
    setDualFeatureState(featType, 'gray', shouldAutoRun ? '生成' + name : 'AI' + name + '详情', '', false);
    return 'pending';
  }

  function refreshDualStatus(featType, preferCached) {
    var reqId = getRequestId();
    if (!reqId) return;
    var userId = getCurrentUserId();
    var apiPath = featType === 'summary' ? '/api/embed/summary/context' : '/api/embed/context';
    var apiUrl = AURA_EMBED_ORIGIN + apiPath
      + '?requestid=' + encodeURIComponent(reqId)
      + '&embed_token=' + encodeURIComponent(EMBED_ACCESS_TOKEN)
      + '&oa_user_id=' + encodeURIComponent(userId);
    if (preferCached) apiUrl += '&prefer_cached=true';

    fetch(apiUrl, { method: 'GET', credentials: 'omit' })
      .then(function (res) {
        if (!res.ok) throw new Error('HTTP ' + res.status);
        return res.json();
      })
      .then(function (res) {
        var wrapped = res && Object.prototype.hasOwnProperty.call(res, 'code');
        var data = wrapped ? res.data : res;
        var kind = applyDualData(featType, data);
        if (kind === 'running') startDualStatusWatch(STATUS_WATCH_MS);
      })
      .catch(function (err) {
        console.warn('[aura-embed-mobile] 请求状态失败:', featType, err);
        setDualFeatureState(featType, 'gray', 'AI' + (featType === 'summary' ? '总结' : '审核') + '详情', '', false);
      });
  }

  function refreshAllDualStatus(preferCached) {
    refreshDualStatus('audit', preferCached);
    refreshDualStatus('summary', preferCached);
  }

  function startDualStatusWatch(durationMs) {
    allStatusWatchUntil = Date.now() + (durationMs || STATUS_WATCH_MS);
    if (allStatusWatchTimer) return;
    allStatusWatchTimer = setInterval(function () {
      if (Date.now() > allStatusWatchUntil) {
        clearInterval(allStatusWatchTimer);
        allStatusWatchTimer = null;
        return;
      }
      refreshAllDualStatus(true);
    }, STATUS_POLL_MS);
  }

  function notifyBeforeReleaseDual(action, callback) {
    var released = false;
    var release = function () {
      if (released) return;
      released = true;
      startDualStatusWatch(STATUS_WATCH_MS);
      callback();
    };
    var context = captureOperationContext(action);
    var timeoutId = setTimeout(function () {
      console.warn('[aura-embed-mobile] OA 操作事件提交超时，已放行 OA', { action: context.action });
      release();
    }, 800);

    try {
      var request = fetch(AURA_EMBED_ORIGIN + '/api/embed/events', {
        method: 'POST',
        mode: 'no-cors',
        credentials: 'omit',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8' },
        body: buildEventBody(context)
      });
      Promise.resolve(request).then(function () {
        clearTimeout(timeoutId);
        release();
      }, function (err) {
        clearTimeout(timeoutId);
        console.warn('[aura-embed-mobile] OA 操作事件提交失败，已放行 OA', err);
        release();
      });
    } catch (e) {
      clearTimeout(timeoutId);
      release();
    }
  }

  function registerDualOAEvents() {
    if (typeof WfForm === 'undefined' || !WfForm.registerCheckEvent) return;
    WfForm.registerCheckEvent(WfForm.OPER_SAVE, function (callback) {
      notifyBeforeReleaseDual('save_requested', callback);
    });
    WfForm.registerCheckEvent(WfForm.OPER_SUBMIT, function (callback) {
      notifyBeforeReleaseDual('submit_requested', callback);
    });
  }

  function scheduleDualResumeRefresh() {
    if (document.visibilityState && document.visibilityState === 'hidden') return;
    if (allResumeRefreshTimer) clearTimeout(allResumeRefreshTimer);
    allResumeRefreshTimer = setTimeout(function () {
      allResumeRefreshTimer = null;
      refreshAllDualStatus(true);
    }, 300);
  }

  function bindDualStatusRefreshListeners() {
    window.addEventListener('message', function (event) {
      if (event.origin !== AURA_EMBED_ORIGIN) return;
      if (!event.data || event.data.type !== MSG_STATUS) return;
      var requestId = getRequestId();
      if (event.data.requestid && requestId && String(event.data.requestid) !== String(requestId)) return;
      var featType = event.data.embed_type === 'summary' ? 'summary' : 'audit';
      if (event.data.running) {
        setDualFeatureState(featType, 'loading', (featType === 'summary' ? '总结' : '审核') + '分析中...', '', false);
        startDualStatusWatch(STATUS_WATCH_MS);
        return;
      }
      var res = {
        status: event.data.status || 'completed',
        parse_error: !!event.data.parse_error,
        recommendation: event.data.recommendation,
        overall_score: event.data.overall_score
      };
      applyDualData(featType, {
        supported: true,
        [featType === 'summary' ? 'summary_result' : 'audit_result']: res,
        [featType === 'summary' ? 'has_summary' : 'has_audit']: !!event.data.has_result
      });
    });
    document.addEventListener('visibilitychange', function () {
      if (document.visibilityState === 'visible') scheduleDualResumeRefresh();
    });
    window.addEventListener('pageshow', scheduleDualResumeRefresh);
    window.addEventListener('focus', scheduleDualResumeRefresh);
  }

  function initDual() {
    bindDualStatusRefreshListeners();
    var reqId = getRequestId();
    if (!reqId) {
      setDualFeatureState('audit', 'disabled', '待保存流程', '流程保存并生成编号后即可查看 AI 审核', false);
      setDualFeatureState('summary', 'disabled', '待保存流程', '流程保存并生成编号后即可查看 AI 总结', false);
      registerDualOAEvents();
      return;
    }
    setDualFeatureState('audit', 'loading', '审核加载中...', '', true);
    setDualFeatureState('summary', 'loading', '总结加载中...', '', true);
    refreshAllDualStatus(false);
    registerDualOAEvents();
  }

  jQuery().ready(function () {
    initDual();
  });
})();
`
}

const downloadTextFile = (filename: string, content: string) => {
  if (!import.meta.client) return
  const blob = new Blob([content], { type: 'text/javascript;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

const exportEmbedScript = (
  target: 'all' | 'audit' | 'summary' = embedScriptTarget.value,
  platform: 'pc' | 'mobile' = embedScriptPlatform.value,
) => {
  if (!selectedTenant.value?.embed_token_configured) {
    message.warning(t('admin.tenants.embedScriptNeedConfigured'))
    return
  }
  if (!embedOrigin.value) {
    message.error(t('admin.tenants.embedScriptOriginMissing'))
    return
  }
  const token = effectiveEmbedToken.value
  if (!token) {
    message.warning(t('admin.tenants.embedScriptNeedToken'))
    return
  }
  if (platform === 'mobile') {
    const cfg = getEmbedMobileScriptConfig(target)
    downloadTextFile(cfg.filename, buildEmbedMobileNotifyScript(target, token))
  } else {
    const cfg = getEmbedScriptConfig(target)
    downloadTextFile(cfg.filename, buildEmbedNotifyScript(target, token))
  }
  message.success(t('admin.tenants.embedScriptExported'))
}

const handleRotateEmbedToken = async () => {
  if (!selectedTenant.value) return
  const wasConfigured = selectedTenant.value.embed_token_configured
  rotatingEmbedToken.value = true
  try {
    const resp = await apiRotateTenantEmbedToken(selectedTenant.value.id)
    rotatedEmbedToken.value = resp.access_token
    manualEmbedToken.value = resp.access_token
    showEmbedTokenModal.value = true
    selectedTenant.value.embed_enabled = true
    selectedTenant.value.embed_token_configured = true
    selectedTenant.value.embed_token_hint = resp.token_hint
    selectedTenant.value.embed_token_rotated_at = resp.rotated_at
    const idx = tenants.value.findIndex(x => x.id === selectedTenant.value!.id)
    if (idx >= 0) {
      tenants.value[idx] = {
        ...tenants.value[idx],
        embed_enabled: true,
        embed_token_configured: true,
        embed_token_hint: resp.token_hint,
        embed_token_rotated_at: resp.rotated_at,
      }
    }
    message.success(wasConfigured ? t('admin.tenants.embedTokenReset') : t('admin.tenants.embedTokenGenerated'))
  } catch (e: any) {
    message.error(e.message || '生成嵌入密钥失败')
  } finally {
    rotatingEmbedToken.value = false
  }
}

const openDeleteConfirm = (tenant: TenantData) => {
  deletingTenant.value = tenant
  deletePassword.value = ''
  showDeleteConfirm.value = true
}

const confirmDeleteTenant = async () => {
  if (!deletingTenant.value) return
  if (!deletePassword.value.trim()) {
    message.warning(t('admin.tenants.deletePasswordRequired'))
    return
  }
  deleting.value = true
  try {
    await apiDeleteTenant(deletingTenant.value.id, deletePassword.value)
    tenants.value = tenants.value.filter(x => x.id !== deletingTenant.value!.id)
    showDeleteConfirm.value = false
    showDetail.value = false
    message.success(t('admin.tenants.deleteSuccess'))
  } catch (e: any) {
    if (e.message?.includes('密码') || e.message?.includes('password') || e.code === 40103) {
      message.error(t('admin.tenants.deletePasswordError'))
    } else {
      message.error(e.message || t('admin.tenants.deleteFailed'))
    }
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <div class="system-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('admin.tenants.title') }}</h1>
        <p class="page-subtitle">{{ t('admin.tenants.subtitle') }}</p>
      </div>
      <a-button type="primary" size="large" @click="showCreate = true; resetNewTenant()">
        <PlusOutlined /> {{ t('admin.tenants.addTenant') }}
      </a-button>
    </div>

    <!--租户卡网格-->
    <div class="tenant-grid">
      <div v-for="tenant in tenants" :key="tenant.id" class="tenant-card" @click="openDetail(tenant)">
        <div class="tenant-card-header">
          <div class="tenant-avatar">
            <TeamOutlined />
          </div>
          <div class="tenant-info">
            <div class="tenant-name">{{ tenant.name }}</div>
            <div class="tenant-code">{{ tenant.code }} · {{ tenant.description || t('admin.tenants.noDesc') }}</div>
          </div>
          <div
            class="tenant-status"
            :class="tenant.status === 'active' ? 'tenant-status--active' : 'tenant-status--inactive'"
          >
            <span class="tenant-status-dot" />
            {{ tenant.status === 'active' ? t('admin.tenants.running') : t('admin.tenants.stopped') }}
          </div>
        </div>

        <div class="tenant-tags">
          <span class="info-tag info-tag--primary">
            <DatabaseOutlined /> {{ getOADbName(tenant.oa_db_connection_id || '') }}
          </span>
          <span class="info-tag info-tag--info">
            <RobotOutlined /> {{ availableModels.find(m => m.id === tenant.primary_model_id)?.display_name || t('admin.tenants.noConfig') }}
          </span>
        </div>

        <div class="tenant-stats">
          <div class="stat-item">
            <span class="stat-label">{{ t('admin.tenants.memberCount') }}</span>
            <span class="stat-value">{{ getTenantStat(tenant.id, 'member_count') }}</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">{{ t('admin.tenants.deptCount') }}</span>
            <span class="stat-value">{{ getTenantStat(tenant.id, 'department_count') }}</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">{{ t('admin.tenants.roleCount') }}</span>
            <span class="stat-value">{{ getTenantStat(tenant.id, 'role_count') }}</span>
          </div>
          <div class="stat-item">
            <span class="stat-label">{{ t('admin.tenants.maxConcurrency') }}</span>
            <span class="stat-value">{{ tenant.max_concurrency }}</span>
          </div>
        </div>

        <div class="token-usage-block">
          <div class="token-usage-header">
            <span class="token-usage-label">{{ t('admin.tenants.tokenUsage') }}</span>
            <span class="token-usage-nums">
              <template v-if="isUnlimitedQuota(tenant.token_quota)">{{ t('admin.tenants.unlimitedQuota') }}</template>
              <template v-else>
                {{ (tenant.token_used / 1000).toFixed(1) }}K / {{ (tenant.token_quota / 1000).toFixed(0) }}K
                <span class="token-usage-percent" :style="{ color: getQuotaColor(getQuotaPercent(tenant.token_used, tenant.token_quota)) }">
                  {{ getQuotaPercent(tenant.token_used, tenant.token_quota) }}%
                </span>
              </template>
            </span>
          </div>
          <div class="quota-bar">
            <div
              class="quota-bar-fill"
              :style="{
                width: getQuotaPercent(tenant.token_used, tenant.token_quota) + '%',
                background: getQuotaColor(getQuotaPercent(tenant.token_used, tenant.token_quota)),
              }"
            />
          </div>
        </div>

        <div class="tenant-card-footer">
          <span class="tenant-created">
            <ClockCircleOutlined /> {{ formatDateTime(tenant.created_at) }}
          </span>
          <div class="tenant-card-actions" @click.stop>
            <a-button size="small" type="text" @click="openDetail(tenant)">
              <EditOutlined /> {{ t('admin.tenants.configure') }}
            </a-button>
            <a-button size="small" type="text" @click="toggleTenantStatus(tenant.id)">
              {{ tenant.status === 'active' ? t('admin.tenants.stop') : t('admin.tenants.enable') }}
            </a-button>
            <a-button size="small" type="text" danger @click="openDeleteConfirm(tenant)">
              <DeleteOutlined /> {{ t('admin.tenants.deleteTenant') }}
            </a-button>
          </div>
        </div>
      </div>
    </div>

    <!--创建租户模式 - 分页签-->
    <a-modal v-model:open="showCreate" :title="t('admin.tenants.createTenant')" @ok="createTenant" :okText="t('admin.tenants.create')" :cancelText="t('admin.tenants.cancel')" width="640px" :maskClosable="false">
      <div class="create-tabs">
        <button
          v-for="tab in [
            { key: 'basic', label: t('admin.tenants.tabCreateBasic') },
            { key: 'admin', label: t('admin.tenants.tabCreateAdmin') },
            { key: 'ai', label: t('admin.tenants.tabCreateAI') },
          ]"
          :key="tab.key"
          class="create-tab-btn"
          :class="{ 'create-tab-btn--active': createTab === tab.key }"
          @click="createTab = tab.key as any"
        >
          {{ tab.label }}
        </button>
      </div>

      <!--基本信息页签-->
      <a-form v-show="createTab === 'basic'" layout="vertical" style="margin-top: 16px;">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.tenantName')" required>
              <a-input v-model:value="newTenant.name" :placeholder="t('admin.tenants.tenantNamePlaceholder')" size="large" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.tenantCode')">
              <a-input v-model:value="newTenant.code" :placeholder="t('admin.tenants.tenantCodePlaceholder')" size="large" />
              <div style="font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px;">{{ t('admin.tenants.codeAutoHint') }}</div>
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item :label="t('admin.tenants.oaDbConnection')">
          <a-select v-model:value="newTenant.oa_db_connection_id" size="large" :placeholder="t('admin.tenants.selectOADb')" allowClear>
            <a-select-option v-for="conn in availableOADbs" :key="conn.id" :value="conn.id">
              {{ conn.name }} ({{ conn.oa_type_label }})
            </a-select-option>
          </a-select>
          <div style="font-size: 12px; color: var(--color-text-tertiary); margin-top: 4px;">
            {{ t('admin.tenants.oaDbHint') }}
            <a @click="navigateTo('/admin/system/settings')" style="cursor: pointer;">{{ t('admin.tenants.systemSettings') }}</a>
          </div>
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item>
              <template #label>
                <div class="quota-label-row">
                  <span>{{ t('admin.tenants.tokenQuota') }}</span>
                  <div class="quota-switch-inline">
                    <span>{{ t('admin.tenants.unlimitedQuota') }}</span>
                    <a-switch
                      size="small"
                      :checked="isUnlimitedQuota(newTenant.token_quota)"
                      @change="(checked: any) => {
                        if (Boolean(checked)) {
                          if (newTenant.token_quota > 0) previousNewTenantQuota = newTenant.token_quota
                          newTenant.token_quota = -1
                        } else {
                          newTenant.token_quota = previousNewTenantQuota || 100000
                        }
                      }"
                    />
                  </div>
                </div>
              </template>
              <a-input-number
                v-if="!isUnlimitedQuota(newTenant.token_quota)"
                v-model:value="newTenant.token_quota"
                :min="1000"
                :step="1000"
                style="width: 100%;"
                size="large"
              />
              <a-input
                v-else
                size="large"
                disabled
                :value="t('admin.tenants.unlimitedQuota')"
                style="width: 100%;"
              />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.maxConcurrencyLabel')">
              <a-input-number v-model:value="newTenant.max_concurrency" :min="1" :max="100" style="width: 100%;" size="large" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item :label="t('admin.tenants.description')">
          <a-textarea v-model:value="newTenant.description" :rows="2" :placeholder="t('admin.tenants.descPlaceholder')" />
        </a-form-item>
      </a-form>

      <!--管理员信息页签-->
      <a-form v-show="createTab === 'admin'" layout="vertical" style="margin-top: 16px;">
        <div class="jdbc-hint" style="margin-bottom: 16px;">
          <InfoCircleOutlined /> {{ t('admin.tenants.adminHint') }}
        </div>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.adminDisplayName')" required>
              <a-input v-model:value="newTenant.admin_display_name" :placeholder="t('admin.tenants.adminDisplayNamePlaceholder')" size="large" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.adminUsername')" required>
              <a-input v-model:value="newTenant.admin_username" :placeholder="t('admin.tenants.adminUsernamePlaceholder')" size="large" />
              <div style="font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px;">{{ t('admin.org.usernameHint') }}</div>
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.adminEmail')">
              <a-input v-model:value="newTenant.admin_email" placeholder="admin@example.com" size="large">
                <template #prefix><MailOutlined /></template>
              </a-input>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.adminPhone')">
              <a-input v-model:value="newTenant.admin_phone" :placeholder="t('admin.tenants.contactPhonePlaceholder')" size="large" :maxlength="11">
                <template #prefix><PhoneOutlined /></template>
              </a-input>
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.adminPassword')">
              <a-input-password v-model:value="newTenant.admin_password" :placeholder="t('admin.tenants.adminPasswordPlaceholder')" size="large" />
              <div style="font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px;">{{ t('admin.tenants.adminPasswordHint') }}</div>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.adminDeptName')" required>
              <a-input v-model:value="newTenant.admin_dept_name" :placeholder="t('admin.tenants.adminDeptNamePlaceholder')" size="large" />
              <div style="font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px;">{{ t('admin.tenants.adminDeptHint') }}</div>
            </a-form-item>
          </a-col>
        </a-row>
      </a-form>

      <!--AI 模型页签-->
      <a-form v-show="createTab === 'ai'" layout="vertical" style="margin-top: 16px;">
        <a-form-item :label="t('admin.tenants.primaryModel')">
          <a-select v-model:value="newTenant.primary_model_id" size="large" :placeholder="t('admin.tenants.selectModel')" allowClear>
            <a-select-option v-for="m in availableModels" :key="m.id" :value="m.id">
              {{ m.display_name }} ({{ m.provider_label || m.provider }})
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item :label="t('admin.tenants.fallbackModelLabel')">
          <a-select v-model:value="newTenant.fallback_model_id" size="large" :placeholder="t('admin.tenants.noConfig')" allowClear>
            <a-select-option v-for="m in availableModels" :key="m.id" :value="m.id">
              {{ m.display_name }} ({{ m.provider_label || m.provider }})
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-divider>{{ t('admin.tenants.callParams') }}</a-divider>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.maxTokenPerReq')">
              <a-input-number v-model:value="newTenant.max_tokens_per_request" :min="512" :max="32768" :step="512" style="width: 100%;" size="large" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.temperature')">
              <a-slider v-model:value="newTenant.temperature" :min="0" :max="1" :step="0.1" />
              <span class="slider-value">{{ newTenant.temperature }}</span>
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.timeout')">
              <a-input-number v-model:value="newTenant.timeout_seconds" :min="10" :max="300" style="width: 100%;" size="large" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.retryCount')">
              <a-input-number v-model:value="newTenant.retry_count" :min="0" :max="10" style="width: 100%;" size="large" />
            </a-form-item>
          </a-col>
        </a-row>
      </a-form>
    </a-modal>

    <!--租户细节抽屉-->
    <a-drawer
      v-model:open="showDetail"
      :title="selectedTenant?.name"
      placement="right"
      width="min(920px, 100vw)"
      @close="showDetail = false"
    >
      <template v-if="selectedTenant">
        <div class="detail-tabs">
          <button
            v-for="tab in [
              { key: 'basic', label: t('admin.tenants.tabBasic'), icon: InfoCircleOutlined },
              { key: 'oadb', label: t('admin.tenants.tabOADb'), icon: DatabaseOutlined },
              { key: 'chat', label: t('admin.tenants.tabChat', '智能体扩展'), icon: RobotOutlined },
              { key: 'ai', label: t('admin.tenants.tabAI'), icon: RobotOutlined },
              { key: 'quota', label: t('admin.tenants.tabQuota'), icon: ThunderboltOutlined },
              { key: 'embed', label: t('admin.tenants.tabEmbed'), icon: LinkOutlined },
              { key: 'sso', label: t('admin.tenants.tabSSO'), icon: LockOutlined },
              { key: 'members', label: t('admin.tenants.tabMembers'), icon: TeamOutlined },
              { key: 'security', label: t('admin.tenants.tabSecurity'), icon: SafetyCertificateOutlined },
            ]"
            :key="tab.key"
            class="detail-tab-btn"
            :aria-pressed="detailActiveTab === tab.key"
            :class="{ 'detail-tab-btn--active': detailActiveTab === tab.key }"
            @click="detailActiveTab = tab.key"
          >
            <component :is="tab.icon" />
            {{ tab.label }}
          </button>
        </div>

        <ChatAllocationPanel v-show="detailActiveTab === 'chat'" ref="chatAllocRef" :tenant-id="selectedTenant.id" :models="availableModels" />
        <!--基本信息选项卡-->
        <div v-if="detailActiveTab === 'basic'" class="detail-section">
          <div class="section-header">
            <h3><UserOutlined /> {{ t('admin.tenants.basicInfo') }}</h3>
          </div>
          <a-form layout="vertical">
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item :label="t('admin.tenants.tenantName')">
                  <a-input v-model:value="selectedTenant.name" size="large" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item :label="t('admin.tenants.tenantCode')">
                  <a-input v-model:value="selectedTenant.code" size="large" disabled />
                </a-form-item>
              </a-col>
            </a-row>
            <a-form-item :label="t('admin.tenants.description')">
              <a-textarea v-model:value="selectedTenant.description" :rows="3" />
            </a-form-item>
            <a-row :gutter="16">
              <a-col :span="8">
                <a-form-item :label="t('admin.tenants.contact')">
                  <a-input v-model:value="selectedTenant.contact_name" :placeholder="t('admin.tenants.contactNamePlaceholder')">
                    <template #prefix><UserOutlined /></template>
                  </a-input>
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item :label="t('admin.tenants.contactEmail')">
                  <a-input v-model:value="selectedTenant.contact_email" :placeholder="t('admin.tenants.contactEmailPlaceholder')">
                    <template #prefix><MailOutlined /></template>
                  </a-input>
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item :label="t('admin.tenants.contactPhone')">
                  <a-input v-model:value="selectedTenant.contact_phone" :placeholder="t('admin.tenants.contactPhonePlaceholder')">
                    <template #prefix><PhoneOutlined /></template>
                  </a-input>
                </a-form-item>
              </a-col>
            </a-row>
            <div v-if="selectedTenant.admin_user_id" class="jdbc-hint" style="margin-bottom: 12px;">
              <InfoCircleOutlined /> {{ t('admin.tenants.contactSyncHint') }}
            </div>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item :label="t('admin.tenants.createdDate')">
                  <a-input :value="formatDateTime(selectedTenant.created_at)" size="large" disabled />
                </a-form-item>
              </a-col>
            </a-row>
          </a-form>
        </div>

        <!--OA 数据库连接选项卡-->
        <div v-if="detailActiveTab === 'oadb'" class="detail-section">
          <div class="section-header">
            <h3><DatabaseOutlined /> {{ t('admin.tenants.oaDbConfig') }}</h3>
          </div>
          <div class="jdbc-hint">
            <InfoCircleOutlined /> {{ t('admin.tenants.oaDbSelectHint') }}
            <a @click="navigateTo('/admin/system/settings')" style="cursor: pointer; margin: 0 4px;">{{ t('admin.tenants.systemSettings') }}</a>
          </div>
          <a-form layout="vertical">
            <a-form-item :label="t('admin.tenants.oaDbConnection')">
              <a-select v-model:value="selectedTenant.oa_db_connection_id" size="large" :placeholder="t('admin.tenants.selectOADb')" allowClear>
                <a-select-option v-for="conn in availableOADbs" :key="conn.id" :value="conn.id">
                  {{ conn.name }} ({{ conn.oa_type_label }})
                </a-select-option>
              </a-select>
            </a-form-item>

            <div v-if="selectedTenant.oa_db_connection_id && getOADbInfo(selectedTenant.oa_db_connection_id)" class="oadb-detail-card">
              <div class="oadb-detail-header">
                <LinkOutlined />
                <span>{{ getOADbInfo(selectedTenant.oa_db_connection_id)!.name }}</span>
                <span class="oadb-detail-type">{{ getOADbInfo(selectedTenant.oa_db_connection_id)!.oa_type_label }}</span>
              </div>
              <div class="oadb-detail-meta">
                <div class="oadb-meta-item">
                  <span class="oadb-meta-label">{{ t('admin.tenants.dbDriver') }}</span>
                  <span class="oadb-meta-value">{{ getOADbInfo(selectedTenant.oa_db_connection_id)!.driver.toUpperCase() }}</span>
                </div>
                <div class="oadb-meta-item">
                  <span class="oadb-meta-label">{{ t('admin.tenants.hostAddress') }}</span>
                  <span class="oadb-meta-value">{{ getOADbInfo(selectedTenant.oa_db_connection_id)!.host }}:{{ getOADbInfo(selectedTenant.oa_db_connection_id)!.port }}</span>
                </div>
                <div class="oadb-meta-item">
                  <span class="oadb-meta-label">{{ t('admin.tenants.dbName') }}</span>
                  <span class="oadb-meta-value">{{ getOADbInfo(selectedTenant.oa_db_connection_id)!.database_name }}</span>
                </div>
                <div class="oadb-meta-item">
                  <span class="oadb-meta-label">{{ t('admin.settings.syncInterval') }}</span>
                  <span class="oadb-meta-value">{{ getOADbInfo(selectedTenant.oa_db_connection_id)!.sync_interval }}s</span>
                </div>
                <div v-if="getOADbInfo(selectedTenant.oa_db_connection_id)!.oa_base_url" class="oadb-meta-item" style="grid-column: span 2;">
                  <span class="oadb-meta-label">{{ t('admin.settings.oaBaseUrl') }}</span>
                  <span class="oadb-meta-value" style="word-break: break-all;">{{ getOADbInfo(selectedTenant.oa_db_connection_id)!.oa_base_url }}</span>
                </div>
              </div>
              <div v-if="getOADbInfo(selectedTenant.oa_db_connection_id)!.description" class="oadb-detail-desc">
                {{ getOADbInfo(selectedTenant.oa_db_connection_id)!.description }}
              </div>
            </div>

            <div v-else-if="!selectedTenant.oa_db_connection_id" class="oadb-empty">
              <InfoCircleOutlined /> {{ t('admin.tenants.noOADbSelected') }}
            </div>
          </a-form>
        </div>

        <!--AI模型选项卡-->
        <div v-if="detailActiveTab === 'ai'" class="detail-section">
          <div class="section-header">
            <h3><RobotOutlined /> {{ t('admin.tenants.aiModelSelect') }}</h3>
          </div>
          <div class="jdbc-hint">
            <InfoCircleOutlined /> {{ t('admin.tenants.aiModelHint') }}<a @click="navigateTo('/admin/system/settings')" style="cursor: pointer; margin: 0 4px;">{{ t('admin.tenants.systemSettings') }}</a>)
          </div>
          <a-form layout="vertical">
            <div class="config-group">
              <div class="config-group-title">{{ t('admin.tenants.primaryModel') }}</div>
              <a-form-item :label="t('admin.tenants.modelName')">
                <a-select v-model:value="selectedTenant.primary_model_id" size="large" :placeholder="t('admin.tenants.selectModel')" allowClear>
                  <a-select-option v-for="m in availableModels" :key="m.id" :value="m.id">
                    {{ m.display_name }} ({{ m.provider_label || m.provider }})
                  </a-select-option>
                </a-select>
              </a-form-item>
            </div>

            <div class="config-group">
              <div class="config-group-title">{{ t('admin.tenants.fallbackModel') }}</div>
              <a-form-item :label="t('admin.tenants.fallbackModelLabel')">
                <a-select v-model:value="selectedTenant.fallback_model_id" size="large" allowClear :placeholder="t('admin.tenants.noConfig')">
                  <a-select-option v-for="m in availableModels" :key="m.id" :value="m.id">
                    {{ m.display_name }} ({{ m.provider_label || m.provider }})
                  </a-select-option>
                </a-select>
              </a-form-item>
            </div>

            <a-divider>{{ t('admin.tenants.callParams') }}</a-divider>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item :label="t('admin.tenants.maxTokenPerReq')">
                  <a-input-number v-model:value="selectedTenant.max_tokens_per_request" :min="512" :max="32768" :step="512" style="width: 100%;" size="large" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item :label="t('admin.tenants.temperature')">
                  <a-slider v-model:value="selectedTenant.temperature" :min="0" :max="1" :step="0.1" />
                  <span class="slider-value">{{ selectedTenant.temperature }}</span>
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item :label="t('admin.tenants.timeout')">
                  <a-input-number v-model:value="selectedTenant.timeout_seconds" :min="10" :max="300" style="width: 100%;" size="large" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item :label="t('admin.tenants.retryCount')">
                  <a-input-number v-model:value="selectedTenant.retry_count" :min="0" :max="10" style="width: 100%;" size="large" />
                </a-form-item>
              </a-col>
            </a-row>
          </a-form>
        </div>

        <!--Basic 单点登录选项卡-->
        <div v-if="detailActiveTab === 'sso'" class="detail-section">
          <div class="section-header">
            <h3><LockOutlined /> {{ t('admin.tenants.ssoTitle') }}</h3>
          </div>
          <a-alert type="info" show-icon :message="t('admin.tenants.ssoHint')" style="margin-bottom: 16px;" />
          <a-form layout="vertical">
            <a-form-item :label="t('admin.tenants.ssoEnabled')">
              <a-switch v-model:checked="selectedTenant.sso_basic_enabled" />
              <span style="margin-left: 10px; color: var(--color-text-secondary);">{{ t('admin.tenants.ssoEnabledHint') }}</span>
            </a-form-item>
            <a-form-item :label="t('admin.tenants.ssoEndpoint')">
              <div class="sso-value-row">
                <a-input :value="ssoEndpoint" readonly size="large">
                  <template #prefix><GlobalOutlined /></template>
                </a-input>
                <a-button class="sso-action-button" size="large" @click="copySSOEndpoint">
                  <CopyOutlined /> {{ t('common.copy') }}
                </a-button>
              </div>
            </a-form-item>
            <a-form-item :label="t('admin.tenants.ssoUsernameFormat')">
              <div class="sso-value-row">
                <a-input :value="ssoUsernameExample" readonly size="large">
                  <template #prefix><UserOutlined /></template>
                </a-input>
                <a-button
                  class="sso-action-button"
                  size="large"
                  @click="copyText(ssoUsernameExample, t('admin.tenants.ssoUsernameCopied'))"
                >
                  <CopyOutlined /> {{ t('common.copy') }}
                </a-button>
              </div>
            </a-form-item>
            <a-form-item :label="t('admin.tenants.ssoSharedPassword')" :required="selectedTenant.sso_basic_enabled && !selectedTenant.sso_basic_password_set">
              <div class="sso-password-status">
                <a-tag v-if="ssoBasicPassword" color="processing">{{ t('admin.tenants.ssoPasswordPending') }}</a-tag>
                <a-tag v-else-if="selectedTenant.sso_basic_password_set" color="success">{{ t('admin.tenants.ssoPasswordConfiguredStatus') }}</a-tag>
                <a-tag v-else color="warning">{{ t('admin.tenants.ssoPasswordNotConfigured') }}</a-tag>
                <span>{{ t('admin.tenants.ssoPasswordStatusHint') }}</span>
              </div>
              <div class="sso-value-row">
                <a-input-password
                  v-model:value="ssoBasicPassword"
                  v-model:visible="ssoPasswordVisible"
                  :placeholder="selectedTenant.sso_basic_password_set ? t('admin.tenants.ssoPasswordConfigured') : t('admin.tenants.ssoPasswordPlaceholder')"
                  autocomplete="new-password"
                  size="large"
                >
                  <template #prefix><KeyOutlined /></template>
                </a-input-password>
                <a-button
                  class="sso-action-button"
                  size="large"
                  :disabled="!ssoBasicPassword"
                  @click="copyText(ssoBasicPassword, t('admin.tenants.ssoPasswordCopied'))"
                >
                  <CopyOutlined /> {{ t('common.copy') }}
                </a-button>
              </div>
              <a-button type="link" class="sso-generate-button" @click="generateSSOPassword">
                <ThunderboltOutlined /> {{ t('admin.tenants.ssoGeneratePassword') }}
              </a-button>
              <div class="form-hint sso-password-hint">{{ t('admin.tenants.ssoPasswordHint') }}</div>
            </a-form-item>
            <a-form-item :label="t('admin.tenants.ssoAllowedIPs')">
              <a-textarea v-model:value="selectedTenant.sso_basic_allowed_ips" :rows="2" :placeholder="t('admin.tenants.ssoAllowedIPsPlaceholder')" />
              <div class="form-hint">{{ t('admin.tenants.ssoAllowedIPsHint') }}</div>
            </a-form-item>
            <a-form-item :label="t('admin.tenants.ssoAllowedDomains')">
              <a-textarea v-model:value="selectedTenant.sso_basic_allowed_domains" :rows="2" :placeholder="t('admin.tenants.ssoAllowedDomainsPlaceholder')" />
              <div class="form-hint">{{ t('admin.tenants.ssoAllowedDomainsHint') }}</div>
            </a-form-item>
          </a-form>
        </div>

        <!--配额和政策选项卡-->
        <div v-if="detailActiveTab === 'quota'" class="detail-section">
          <div class="section-header">
            <h3><ThunderboltOutlined /> {{ t('admin.tenants.quotaPolicy') }}</h3>
          </div>
          <a-form layout="vertical">
            <div class="config-group">
              <div class="config-group-title">{{ t('admin.tenants.resourceQuota') }}</div>
              <a-row :gutter="16">
                <a-col :span="12">
                  <a-form-item>
                    <template #label>
                      <div class="quota-label-row">
                        <span>{{ t('admin.tenants.tokenQuota') }}</span>
                        <div class="quota-switch-inline">
                          <span>{{ t('admin.tenants.unlimitedQuota') }}</span>
                          <a-switch
                            size="small"
                            :checked="isUnlimitedQuota(selectedTenant.token_quota)"
                            @change="handleToggleUnlimitedQuota"
                          />
                        </div>
                      </div>
                    </template>
                    <a-input-number
                      v-if="!isUnlimitedQuota(selectedTenant.token_quota)"
                      v-model:value="selectedTenant.token_quota"
                      :min="1000"
                      :step="1000"
                      style="width: 100%;"
                      size="large"
                    />
                    <a-input
                      v-else
                      size="large"
                      disabled
                      :value="t('admin.tenants.unlimitedQuota')"
                      style="width: 100%;"
                    />
                  </a-form-item>
                </a-col>
                <a-col :span="12">
                  <a-form-item :label="t('admin.tenants.maxConcurrency')">
                    <a-input-number v-model:value="selectedTenant.max_concurrency" :min="1" :max="100" style="width: 100%;" size="large" />
                  </a-form-item>
                </a-col>
              </a-row>
              <div class="usage-display">
                <div class="usage-info">
                  <span v-if="isUnlimitedQuota(selectedTenant.token_quota)">{{ t('admin.tenants.usedUnlimited', [selectedTenant.token_used.toLocaleString()]) }}</span>
                  <template v-else>
                    <span>{{ t('admin.tenants.usedTokens', [selectedTenant.token_used.toLocaleString(), selectedTenant.token_quota.toLocaleString()]) }}</span>
                    <span :style="{ color: getQuotaColor(getQuotaPercent(selectedTenant.token_used, selectedTenant.token_quota)) }">
                      {{ getQuotaPercent(selectedTenant.token_used, selectedTenant.token_quota) }}%
                    </span>
                  </template>
                </div>
                <div class="quota-bar" style="height: 8px;">
                  <div
                    class="quota-bar-fill"
                    :style="{
                      width: getQuotaPercent(selectedTenant.token_used, selectedTenant.token_quota) + '%',
                      background: getQuotaColor(getQuotaPercent(selectedTenant.token_used, selectedTenant.token_quota)),
                    }"
                  />
                </div>
              </div>
            </div>

            <div class="config-group">
              <div class="config-group-title">{{ t('admin.tenants.dataRetention') }}</div>
              <a-row :gutter="16">
                <a-col :span="12">
                  <a-form-item :label="t('admin.tenants.logRetention')">
                    <a-input-number v-model:value="selectedTenant.log_retention_days" :min="7" :max="3650" style="width: 100%;" size="large" />
                    <div class="form-hint">{{ t('admin.tenants.logRetentionHint') }}</div>
                  </a-form-item>
                </a-col>
                <a-col :span="12">
                  <a-form-item :label="t('admin.tenants.auditDataRetention')">
                    <a-input-number v-model:value="selectedTenant.data_retention_days" :min="30" :max="3650" style="width: 100%;" size="large" />
                    <div class="form-hint">{{ t('admin.tenants.auditDataRetentionHint') }}</div>
                  </a-form-item>
                </a-col>
              </a-row>
            </div>
          </a-form>
        </div>

        <!--OA 嵌入选项卡-->
        <div v-if="detailActiveTab === 'embed'" class="detail-section">
          <div class="section-header">
            <h3><LinkOutlined /> {{ t('admin.tenants.embedTitle') }}</h3>
          </div>
          <div class="jdbc-hint">
            <InfoCircleOutlined /> {{ t('admin.tenants.embedHint') }}
          </div>
          <a-form layout="vertical">
            <div class="config-group">
              <div class="config-group-title">{{ t('admin.tenants.basicInfo') }}</div>
              <a-row :gutter="16">
                <a-col :span="12">
                  <a-form-item :label="t('admin.tenants.tenantName')">
                    <a-input :value="selectedTenant.name" disabled />
                  </a-form-item>
                </a-col>
                <a-col :span="12">
                  <a-form-item :label="t('admin.tenants.tenantCode')">
                    <a-input :value="selectedTenant.code" disabled />
                  </a-form-item>
                </a-col>
              </a-row>
              <a-form-item :label="t('admin.tenants.tenantId')">
                <a-input :value="selectedTenant.id" disabled />
              </a-form-item>
            </div>

            <div class="config-group">
              <a-form-item :label="t('admin.tenants.embedEnabled')">
                <a-switch v-model:checked="selectedTenant.embed_enabled" />
                <span class="switch-label">{{ t('admin.tenants.embedEnabledHint') }}</span>
              </a-form-item>
            </div>

            <div class="config-group">
              <div class="config-group-title">{{ t('admin.tenants.embedToken') }}</div>
              <div class="embed-token-panel">
                <div class="embed-token-meta">
                  <div>
                    <span class="embed-token-label">{{ t('admin.tenants.embedToken') }}</span>
                    <div class="embed-token-value">
                      {{ selectedTenant.embed_token_configured ? selectedTenant.embed_token_hint : t('admin.tenants.embedTokenNotConfigured') }}
                    </div>
                  </div>
                  <div v-if="selectedTenant.embed_token_rotated_at">
                    <span class="embed-token-label">{{ t('admin.tenants.embedTokenRotatedAt') }}</span>
                    <div class="embed-token-value">{{ formatDateTime(selectedTenant.embed_token_rotated_at) }}</div>
                  </div>
                </div>
                <div style="display: flex; gap: 8px; flex-wrap: wrap;">
                  <a-button type="primary" :loading="rotatingEmbedToken" @click="handleRotateEmbedToken">
                    {{ selectedTenant.embed_token_configured ? t('admin.tenants.resetEmbedToken') : t('admin.tenants.generateEmbedToken') }}
                  </a-button>
                </div>
              </div>
            </div>

            <div class="config-group">
              <div class="config-group-title">{{ t('admin.tenants.tabEmbed') }}</div>
              <a-form-item :label="t('admin.tenants.embedAuditUrl')">
                <a-input :value="embedAuditUrl" readonly>
                  <template #suffix>
                    <a-button type="text" size="small" @click="copyText(embedAuditUrl)"><CopyOutlined /></a-button>
                  </template>
                </a-input>
              </a-form-item>
              <a-form-item :label="t('admin.tenants.embedSummaryUrl')">
                <a-input :value="embedSummaryUrl" readonly>
                  <template #suffix>
                    <a-button type="text" size="small" @click="copyText(embedSummaryUrl)"><CopyOutlined /></a-button>
                  </template>
                </a-input>
              </a-form-item>
              <div class="embed-script-panel">
                <div class="embed-script-header">
                  <div>
                    <div class="embed-script-title">{{ t('admin.tenants.embedScriptExportTitle') }}</div>
                    <div class="embed-script-desc">{{ t('admin.tenants.embedScriptExportDesc') }}</div>
                  </div>
                </div>
                <div class="embed-script-fields">
                  <div class="embed-script-token-field">
                    <a-form-item :label="t('admin.tenants.embedScriptPlatform')">
                      <a-radio-group v-model:value="embedScriptPlatform" button-style="solid" class="embed-script-platform-options">
                        <a-radio-button v-for="opt in embedScriptPlatformOptions" :key="opt.value" :value="opt.value">
                          {{ opt.label }}
                        </a-radio-button>
                      </a-radio-group>
                      <div class="form-hint">{{ t('admin.tenants.embedScriptPlatformHint') }}</div>
                    </a-form-item>
                  </div>
                  <div class="embed-script-target-field">
                    <a-form-item :label="t('admin.tenants.embedScriptTarget')">
                      <a-radio-group v-model:value="embedScriptTarget" button-style="solid" class="embed-script-target-options">
                        <a-radio-button v-for="opt in embedScriptTargetOptions" :key="opt.value" :value="opt.value">
                          {{ opt.label }}
                        </a-radio-button>
                      </a-radio-group>
                      <div class="form-hint">{{ t(embedScriptPlatform === 'mobile' ? 'admin.tenants.embedScriptMobileTargetHint' : 'admin.tenants.embedScriptTargetHint') }}</div>
                    </a-form-item>
                  </div>
                  <div class="embed-script-token-field">
                    <a-form-item :label="t('admin.tenants.embedScriptTokenInput')">
                      <a-input-password
                        v-model:value="manualEmbedToken"
                        autocomplete="off"
                      />
                      <div class="form-hint">{{ t('admin.tenants.embedScriptTokenHint') }}</div>
                    </a-form-item>
                  </div>
                </div>
                <div class="embed-script-actions">
                  <a-button type="primary" :disabled="!selectedTenant.embed_token_configured" @click="exportEmbedScript(embedScriptTarget, embedScriptPlatform)">
                    <DownloadOutlined /> {{ t(embedScriptPlatform === 'mobile' ? 'admin.tenants.exportEmbedMobileScript' : 'admin.tenants.exportEmbedPCScript') }}
                  </a-button>
                </div>
              </div>
            </div>
          </a-form>
        </div>

        <!--人员选项卡-->
        <div v-if="detailActiveTab === 'members'" class="detail-section">
          <div class="section-header">
            <h3><TeamOutlined /> {{ t('admin.tenants.tabMembers') }}</h3>
          </div>
          <div class="jdbc-hint" style="margin-bottom: 16px;">
            <InfoCircleOutlined /> {{ t('admin.tenants.membersHint') }}
          </div>
          <a-spin :spinning="membersLoading">
            <div v-if="tenantMembers.length === 0 && !membersLoading" class="oadb-empty">
              <InfoCircleOutlined /> {{ t('admin.tenants.noMembers') }}
            </div>
            <div v-else class="members-list">
              <div v-for="m in pagedMembers" :key="m.id" class="member-card">
                <div class="member-card-left">
                  <div class="member-avatar"><UserOutlined /></div>
                  <div class="member-info">
                    <div class="member-name">{{ m.display_name }} <span class="member-username">@{{ m.username }}</span></div>
                    <div class="member-meta">
                      <span v-if="m.department_name">{{ m.department_name }}</span>
                      <span v-if="m.position"> · {{ m.position }}</span>
                    </div>
                    <div class="member-tags">
                      <span v-for="role in m.role_names" :key="role" class="info-tag info-tag--primary" style="font-size: 10px; padding: 1px 6px;">{{ role }}</span>
                    </div>
                  </div>
                </div>
                <div class="member-card-right">
                  <div v-if="m.email" class="member-contact"><MailOutlined /> {{ m.email }}</div>
                  <div v-if="m.phone" class="member-contact"><PhoneOutlined /> {{ m.phone }}</div>
                  <a-tag :color="m.status === 'active' ? 'green' : 'default'" style="margin-top: 4px;">
                    {{ m.status === 'active' ? t('admin.org.active') : t('admin.org.disabled') }}
                  </a-tag>
                </div>
              </div>
            </div>
            <div v-if="memberTotal > memberPageSize" class="pagination-wrapper" style="margin-top: 12px; text-align: right;">
              <a-pagination
                :current="memberPage"
                :page-size="memberPageSize"
                :total="memberTotal"
                size="small"
                show-size-changer
                show-quick-jumper
                :page-size-options="['10', '20', '50']"
                @change="onMemberPageChange"
                @showSizeChange="onMemberPageChange"
              />
            </div>
          </a-spin>
        </div>

        <!--运行状态选项卡-->
        <div v-if="detailActiveTab === 'security'" class="detail-section">
          <div class="section-header">
            <h3><SafetyCertificateOutlined /> {{ t('admin.tenants.tabSecurity') }}</h3>
          </div>
          <a-form layout="vertical">
            <div class="config-group">
              <div class="config-group-title">{{ t('admin.tenants.tenantStatus') }}</div>
              <div class="status-display">
                <div class="status-info">
                  <span>{{ t('admin.tenants.currentStatus') }}</span>
                  <a-tag :color="selectedTenant.status === 'active' ? 'green' : 'default'">
                    {{ selectedTenant.status === 'active' ? t('admin.tenants.running') : t('admin.tenants.stopped') }}
                  </a-tag>
                </div>
                <a-button
                  :danger="selectedTenant.status === 'active'"
                  @click="toggleTenantStatus(selectedTenant.id); selectedTenant.status = selectedTenant.status === 'active' ? 'inactive' : 'active'"
                >
                  {{ selectedTenant.status === 'active' ? t('admin.tenants.disableTenant') : t('admin.tenants.enableTenant') }}
                </a-button>
              </div>
            </div>
          </a-form>
        </div>

        <!--页脚操作-->
        <div class="detail-footer">
          <a-button danger @click="openDeleteConfirm(selectedTenant)">
            <DeleteOutlined /> {{ t('admin.tenants.deleteTenant') }}
          </a-button>
          <div style="display: flex; gap: 8px;">
            <a-button @click="showDetail = false">{{ t('admin.tenants.cancel') }}</a-button>
            <a-button type="primary" @click="saveTenantDetail">{{ t('admin.tenants.saveConfig') }}</a-button>
          </div>
        </div>
      </template>
    </a-drawer>

    <!--删除租户确认弹窗-->
    <a-modal
      v-model:open="showDeleteConfirm"
      :title="t('admin.tenants.deleteConfirmTitle')"
      :okText="t('admin.tenants.deleteTenant')"
      :cancelText="t('admin.tenants.cancel')"
      :okButtonProps="{ danger: true, loading: deleting }"
      :maskClosable="false"
      @ok="confirmDeleteTenant"
      width="520px"
    >
      <div style="padding: 8px 0;">
        <a-alert
          type="error"
          show-icon
          style="margin-bottom: 16px;"
        >
          <template #icon><ExclamationCircleOutlined /></template>
          <template #message>{{ t('admin.tenants.deleteConfirmDesc') }}</template>
        </a-alert>

        <div v-if="deletingTenant" style="padding: 12px 16px; background: var(--color-bg-hover); border-radius: var(--radius-md); margin-bottom: 16px;">
          <div style="font-size: 13px; color: var(--color-text-tertiary); margin-bottom: 4px;">{{ t('admin.tenants.deleteConfirmTenantName') }}</div>
          <div style="font-size: 16px; font-weight: 600; color: var(--color-text-primary);">
            {{ deletingTenant.name }}
            <span style="font-size: 12px; font-weight: 400; color: var(--color-text-tertiary); margin-left: 8px;">{{ deletingTenant.code }}</span>
          </div>
        </div>

        <a-form layout="vertical">
          <a-form-item :label="t('admin.tenants.deleteConfirmPassword')" required>
            <a-input-password
              v-model:value="deletePassword"
              :placeholder="t('admin.tenants.deletePasswordPlaceholder')"
              size="large"
              @pressEnter="confirmDeleteTenant"
            >
              <template #prefix><LockOutlined /></template>
            </a-input-password>
          </a-form-item>
        </a-form>
      </div>
    </a-modal>

    <a-modal
      v-model:open="showEmbedTokenModal"
      :title="t('admin.tenants.embedTokenModalTitle')"
      :footer="null"
      :maskClosable="false"
      width="560px"
    >
      <a-alert type="warning" show-icon :message="t('admin.tenants.embedTokenModalDesc')" style="margin-bottom: 16px;" />
      <a-input :value="rotatedEmbedToken" readonly>
        <template #suffix>
          <a-button type="text" size="small" @click="copyText(rotatedEmbedToken)"><CopyOutlined /></a-button>
        </template>
      </a-input>
      <div class="embed-token-modal-actions" style="margin-top: 20px; display: flex; flex-direction: column; gap: 10px;">
        <div style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px;">
          <a-button @click="exportEmbedScript('audit', 'pc')">
            <DownloadOutlined /> {{ t('admin.tenants.exportModalPCAudit') }}
          </a-button>
          <a-button @click="exportEmbedScript('summary', 'pc')">
            <DownloadOutlined /> {{ t('admin.tenants.exportModalPCSummary') }}
          </a-button>
          <a-button type="primary" @click="exportEmbedScript('all', 'pc')">
            <DownloadOutlined /> {{ t('admin.tenants.exportModalPCAll') }}
          </a-button>
        </div>
        <div style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px;">
          <a-button @click="exportEmbedScript('audit', 'mobile')">
            <DownloadOutlined /> {{ t('admin.tenants.exportModalMobileAudit') }}
          </a-button>
          <a-button @click="exportEmbedScript('summary', 'mobile')">
            <DownloadOutlined /> {{ t('admin.tenants.exportModalMobileSummary') }}
          </a-button>
          <a-button type="primary" @click="exportEmbedScript('all', 'mobile')">
            <DownloadOutlined /> {{ t('admin.tenants.exportModalMobileAll') }}
          </a-button>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 28px;
}
.page-title {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin: 0;
}
.page-subtitle {
  font-size: 14px;
  color: var(--color-text-tertiary);
  margin: 4px 0 0;
}

/* 租户网格 */
.tenant-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
}
@media (max-width: 900px) {
  .tenant-grid { grid-template-columns: 1fr; }
}

.tenant-card {
  background: var(--color-bg-card);
  border-radius: var(--radius-xl);
  border: 1px solid var(--color-border-light);
  padding: 22px;
  transition: all var(--transition-base);
  cursor: pointer;
}
.tenant-card:hover {
  box-shadow: var(--shadow-lg);
  transform: translateY(-3px);
  border-color: var(--color-primary);
}
.tenant-card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}
.tenant-avatar {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-lg);
  background: linear-gradient(135deg, var(--color-primary-bg), var(--color-primary-lighter));
  color: var(--color-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  flex-shrink: 0;
}
.tenant-info { flex: 1; min-width: 0; }
.tenant-name { font-size: 16px; font-weight: 600; color: var(--color-text-primary); }
.tenant-code { font-size: 12px; color: var(--color-text-tertiary); font-family: var(--font-mono); }
.tenant-status {
  display: flex; align-items: center; gap: 6px; font-size: 12px; font-weight: 500;
  flex-shrink: 0; padding: 4px 10px; border-radius: var(--radius-full);
}
.tenant-status-dot { width: 7px; height: 7px; border-radius: 50%; }
.tenant-status--active { color: var(--color-success); background: var(--color-success-bg); }
.tenant-status--active .tenant-status-dot { background: var(--color-success); box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.2); }
.tenant-status--inactive { color: var(--color-text-tertiary); background: var(--color-bg-hover); }
.tenant-status--inactive .tenant-status-dot { background: var(--color-text-tertiary); }

/* 标签 */
.tenant-tags { display: flex; gap: 8px; margin-bottom: 16px; flex-wrap: wrap; }
.info-tag {
  display: inline-flex; align-items: center; gap: 4px;
  font-size: 11px; font-weight: 500; padding: 3px 10px; border-radius: var(--radius-full);
}
.info-tag--primary { background: var(--color-primary-bg); color: var(--color-primary); }
.info-tag--info { background: var(--color-info-bg); color: var(--color-info); }
.info-tag--success { background: var(--color-success-bg); color: var(--color-success); }

/* 统计 */
.tenant-stats { display: flex; gap: 24px; margin-bottom: 12px; flex-wrap: wrap; }
.stat-item { display: flex; flex-direction: column; gap: 2px; }
.stat-label { font-size: 11px; color: var(--color-text-tertiary); }
.stat-value { font-size: 14px; font-weight: 600; color: var(--color-text-primary); }

/* Token 用量 */
.quota-unlimited-row { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
.quota-label-row { display: flex; align-items: center; justify-content: space-between; width: 100%; }
.quota-switch-inline { display: inline-flex; align-items: center; gap: 8px; margin-left: 12px; font-size: 14px; color: var(--color-text-secondary); }
.token-usage-block { margin-bottom: 14px; }
.token-usage-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
.token-usage-label { font-size: 11px; color: var(--color-text-tertiary); }
.token-usage-nums { font-size: 13px; font-weight: 600; color: var(--color-text-primary); }
.token-usage-percent { font-size: 12px; font-weight: 600; margin-left: 8px; }
.quota-bar { flex: 1; height: 6px; background: var(--color-bg-hover); border-radius: var(--radius-full); overflow: hidden; }
.quota-bar-fill { height: 100%; border-radius: var(--radius-full); transition: width 0.5s ease; }

.tenant-card-footer {
  display: flex; justify-content: space-between; align-items: center;
  margin-top: 14px; padding-top: 14px; border-top: 1px solid var(--color-border-light);
}
.tenant-created { font-size: 12px; color: var(--color-text-tertiary); display: flex; align-items: center; gap: 4px; }
.tenant-card-actions { display: flex; gap: 4px; }

/* 创建租户页签 */
.create-tabs {
  display: flex; gap: 4px; background: var(--color-bg-hover); padding: 4px;
  border-radius: var(--radius-lg); margin-top: 8px;
}
.create-tab-btn {
  flex: 1; display: flex; align-items: center; justify-content: center; gap: 6px;
  padding: 8px 14px; border: none; background: transparent; border-radius: var(--radius-md);
  font-size: 13px; font-weight: 500; color: var(--color-text-tertiary);
  cursor: pointer; transition: all var(--transition-fast); white-space: nowrap;
}
.create-tab-btn:hover { color: var(--color-text-primary); }
.create-tab-btn--active { background: var(--color-bg-card); color: var(--color-primary); box-shadow: var(--shadow-xs); }

/* 详情抽屉 */
.detail-tabs {
  display: flex; gap: 4px; background: var(--color-bg-hover); padding: 4px;
  border-radius: var(--radius-lg); margin-bottom: 24px; flex-wrap: wrap;
}
.detail-tab-btn {
  display: flex; align-items: center; gap: 6px; padding: 8px 11px; flex: 0 0 auto;
  border: none; background: transparent; border-radius: var(--radius-md);
  font-size: 13px; font-weight: 500; color: var(--color-text-tertiary);
  cursor: pointer; transition: all var(--transition-fast); white-space: nowrap;
}
.detail-tab-btn:hover { color: var(--color-text-primary); }
.detail-tab-btn--active { background: var(--color-bg-card); color: var(--color-primary); box-shadow: var(--shadow-xs); }

.detail-section { animation: fadeIn 0.2s ease; }
@keyframes fadeIn {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}
.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.section-header h3 {
  font-size: 16px; font-weight: 600; color: var(--color-text-primary);
  margin: 0; display: flex; align-items: center; gap: 8px;
}
.jdbc-hint {
  display: flex; align-items: center; gap: 8px; font-size: 13px; color: var(--color-info);
  background: var(--color-info-bg); padding: 10px 14px; border-radius: var(--radius-md); margin-bottom: 20px;
}
.config-group { background: var(--color-bg-page); border-radius: var(--radius-lg); padding: 16px 20px; margin-bottom: 16px; }
.config-group-title { font-size: 13px; font-weight: 600; color: var(--color-text-secondary); margin-bottom: 12px; }
.switch-label { font-size: 13px; color: var(--color-text-tertiary); margin-left: 10px; }
.slider-value { font-size: 14px; font-weight: 600; color: var(--color-primary); margin-left: 8px; }
.form-hint { font-size: 11px; color: var(--color-text-tertiary); margin-top: 4px; }
.sso-value-row {
  display: flex; align-items: stretch; gap: 10px; width: 100%;
}
.sso-value-row :deep(.ant-input-affix-wrapper) { min-width: 0; flex: 1; }
.sso-value-row :deep(.ant-input-prefix) { color: var(--color-text-tertiary); margin-inline-end: 10px; }
.sso-action-button {
  display: inline-flex; align-items: center; justify-content: center; gap: 6px;
  min-width: 92px; flex: 0 0 92px; font-weight: 500;
}
.sso-password-status {
  display: flex; align-items: center; flex-wrap: wrap; gap: 8px;
  color: var(--color-text-tertiary); font-size: 12px; margin-bottom: 10px;
}
.sso-password-status :deep(.ant-tag) { margin-inline-end: 0; }
.sso-generate-button {
  height: auto; padding: 8px 0 2px; display: inline-flex; align-items: center; gap: 6px;
  font-weight: 500;
}
.sso-password-hint { line-height: 1.6; }

.usage-display {
  background: var(--color-bg-card); border-radius: var(--radius-md); padding: 14px;
  border: 1px solid var(--color-border-light);
}
.usage-info {
  display: flex; justify-content: space-between; align-items: center;
  font-size: 13px; color: var(--color-text-secondary); margin-bottom: 8px;
}
.status-display {
  display: flex; justify-content: space-between; align-items: center;
  background: var(--color-bg-card); border-radius: var(--radius-md); padding: 14px;
  border: 1px solid var(--color-border-light);
}
.status-info { display: flex; align-items: center; gap: 8px; font-size: 14px; color: var(--color-text-secondary); }
.detail-footer {
  display: flex; justify-content: space-between; align-items: center; margin-top: 32px;
  padding-top: 20px; border-top: 1px solid var(--color-border-light);
}

/* OA DB 详情卡 */
.oadb-detail-card {
  background: var(--color-bg-page); border: 1px solid var(--color-border-light);
  border-radius: var(--radius-lg); padding: 16px 20px; margin-top: 12px;
}
.oadb-detail-header {
  display: flex; align-items: center; gap: 8px; font-size: 15px; font-weight: 600;
  color: var(--color-text-primary); margin-bottom: 12px;
}
.oadb-detail-type {
  font-size: 12px; font-weight: 500; color: var(--color-primary);
  background: var(--color-primary-bg); padding: 2px 8px; border-radius: var(--radius-full);
}
.oadb-detail-meta {
  display: flex; gap: 20px; flex-wrap: wrap; padding: 10px 14px;
  background: var(--color-bg-card); border-radius: var(--radius-md); margin-bottom: 8px;
}
.oadb-meta-label { font-size: 11px; color: var(--color-text-tertiary); display: block; }
.oadb-meta-value { font-size: 13px; font-weight: 500; color: var(--color-text-primary); margin-top: 2px; display: block; }
.oadb-detail-desc { font-size: 12px; color: var(--color-text-tertiary); margin-top: 8px; }
.oadb-empty {
  display: flex; align-items: center; gap: 8px; font-size: 13px; color: var(--color-text-tertiary);
  padding: 20px; text-align: center; justify-content: center;
  background: var(--color-bg-page); border-radius: var(--radius-md); margin-top: 12px;
}

/* 成员列表 */
.members-list { display: flex; flex-direction: column; gap: 12px; }
.member-card {
  display: flex; justify-content: space-between; align-items: flex-start;
  background: var(--color-bg-page); border: 1px solid var(--color-border-light);
  border-radius: var(--radius-lg); padding: 14px 18px; gap: 16px;
}
.member-card-left { display: flex; gap: 12px; align-items: flex-start; flex: 1; min-width: 0; }
.member-avatar {
  width: 36px; height: 36px; border-radius: 50%;
  background: var(--color-primary-bg); color: var(--color-primary);
  display: flex; align-items: center; justify-content: center; font-size: 16px; flex-shrink: 0;
}
.member-info { min-width: 0; }
.member-name { font-size: 14px; font-weight: 600; color: var(--color-text-primary); }
.member-username { font-size: 12px; font-weight: 400; color: var(--color-text-tertiary); margin-left: 4px; }
.member-meta { font-size: 12px; color: var(--color-text-secondary); margin-top: 2px; }
.member-tags { display: flex; gap: 4px; flex-wrap: wrap; margin-top: 4px; }
.member-card-right { display: flex; flex-direction: column; align-items: flex-end; flex-shrink: 0; }
.member-contact { font-size: 12px; color: var(--color-text-tertiary); display: flex; align-items: center; gap: 4px; }

.embed-token-panel {
  display: flex; flex-direction: column; gap: 12px;
  background: var(--color-bg-card); border-radius: var(--radius-md); padding: 14px;
  border: 1px solid var(--color-border-light);
}
.embed-token-meta { display: flex; gap: 24px; flex-wrap: wrap; }
.embed-token-label { font-size: 11px; color: var(--color-text-tertiary); display: block; }
.embed-token-value { font-size: 14px; font-weight: 600; color: var(--color-text-primary); margin-top: 4px; font-family: var(--font-mono); }
.embed-script-panel {
  background: var(--color-bg-card); border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md); padding: 14px; margin-top: 8px;
}
.embed-script-header {
  display: flex; justify-content: space-between; align-items: flex-start;
  gap: 12px; margin-bottom: 14px;
}
.embed-script-title { font-size: 14px; font-weight: 600; color: var(--color-text-primary); }
.embed-script-desc { font-size: 12px; color: var(--color-text-tertiary); margin-top: 4px; line-height: 1.5; }
.embed-script-fields {
  display: grid; grid-template-columns: minmax(0, 1fr);
  gap: 4px; align-items: start;
}
.embed-script-token-field,
.embed-script-target-field { min-width: 0; }
.embed-script-platform-options,
.embed-script-target-options {
  display: flex; flex-wrap: wrap; gap: 8px; width: 100%;
}
.embed-script-platform-options :deep(.ant-radio-button-wrapper),
.embed-script-target-options :deep(.ant-radio-button-wrapper) {
  height: auto; min-height: 36px; line-height: 1.5; padding: 8px 12px;
  border: 1px solid var(--color-border-light); border-radius: var(--radius-md);
  white-space: normal;
}
.embed-script-platform-options :deep(.ant-radio-button-wrapper::before),
.embed-script-target-options :deep(.ant-radio-button-wrapper::before) { display: none; }
.embed-script-actions {
  display: flex; justify-content: flex-end; padding-top: 16px;
  border-top: 1px solid var(--color-border-light);
}
.embed-token-modal-actions { display: flex; gap: 8px; flex-wrap: wrap; margin-top: 16px; justify-content: flex-end; }

@media (max-width: 768px) {
  .page-header { flex-direction: column; gap: 12px; align-items: stretch; }
  .tenant-grid { grid-template-columns: 1fr; }
  .member-card { flex-direction: column; }
  .member-card-right { align-items: flex-start; }
  .embed-script-header { flex-direction: column; }
  .embed-script-fields { grid-template-columns: 1fr; }
  .sso-value-row { align-items: stretch; }
}
@media (max-width: 480px) {
  .page-title { font-size: 20px; }
  .sso-value-row { flex-direction: column; }
  .sso-action-button { width: 100%; flex-basis: auto; }
}
</style>
