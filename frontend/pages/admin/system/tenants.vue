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

const openDetail = async (tenant: TenantData) => {
  selectedTenant.value = { ...tenant }
  detailActiveTab.value = 'basic'
  ssoBasicPassword.value = ''
  ssoPasswordVisible.value = false
  rotatedEmbedToken.value = ''
  manualEmbedToken.value = ''
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

const getQuotaPercent = (used: number, total: number) => Math.round((used / total) * 100)

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
    var store = WfForm.getGlobalStore();
    var currentUserId = store && store.commonParam && store.commonParam.currentUserid != null
      ? String(store.commonParam.currentUserid).trim()
      : '';
    return {
      action: action,
      event_id: createEventId(),
      occurred_at_ms: occurredAtMs,
      requestid: getRequestId(),
      workflow_id: base.workflowid != null ? String(base.workflowid).trim() : '',
      oa_belong_user_id: base.f_weaver_belongto_userid != null
        ? String(base.f_weaver_belongto_userid).trim()
        : '',
      oa_current_user_id: currentUserId
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

  function buildPayload(requestid) {
    return {
      type: MSG_REQUESTID,
      requestid: requestid,
      embed_token: EMBED_ACCESS_TOKEN
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
      ['oa_belong_user_id', context.oa_belong_user_id],
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

const exportEmbedScript = (target: 'all' | 'audit' | 'summary' = embedScriptTarget.value) => {
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
  const cfg = getEmbedScriptConfig(target)
  downloadTextFile(cfg.filename, buildEmbedNotifyScript(target, token))
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
              {{ (tenant.token_used / 1000).toFixed(1) }}K / {{ (tenant.token_quota / 1000).toFixed(0) }}K
              <span class="token-usage-percent" :style="{ color: getQuotaColor(getQuotaPercent(tenant.token_used, tenant.token_quota)) }">
                {{ getQuotaPercent(tenant.token_used, tenant.token_quota) }}%
              </span>
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
            { key: 'chat', label: t('agentAdmin.title'), icon: RobotOutlined },
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
            <a-form-item :label="t('admin.tenants.tokenQuota')">
              <a-input-number v-model:value="newTenant.token_quota" :min="1000" :step="1000" style="width: 100%;" size="large" />
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
      :title="selectedTenant?.name + ' — ' + t('admin.tenants.tabBasic')"
      placement="right"
      :width="720"
      @close="showDetail = false"
    >
      <template v-if="selectedTenant">
        <div class="detail-tabs">
          <button
            v-for="tab in [
              { key: 'basic', label: t('admin.tenants.tabBasic'), icon: InfoCircleOutlined },
              { key: 'oadb', label: t('admin.tenants.tabOADb'), icon: DatabaseOutlined },
              { key: 'chat', label: t('agentAdmin.title'), icon: RobotOutlined },
              { key: 'ai', label: t('admin.tenants.tabAI'), icon: RobotOutlined },
              { key: 'quota', label: t('admin.tenants.tabQuota'), icon: ThunderboltOutlined },
              { key: 'embed', label: t('admin.tenants.tabEmbed'), icon: LinkOutlined },
              { key: 'sso', label: t('admin.tenants.tabSSO'), icon: LockOutlined },
              { key: 'members', label: t('admin.tenants.tabMembers'), icon: TeamOutlined },
              { key: 'security', label: t('admin.tenants.tabSecurity'), icon: SafetyCertificateOutlined },
            ]"
            :key="tab.key"
            class="detail-tab-btn"
            :class="{ 'detail-tab-btn--active': detailActiveTab === tab.key }"
            @click="detailActiveTab = tab.key"
          >
            <component :is="tab.icon" />
            {{ tab.label }}
          </button>
        </div>

        <ChatAllocationPanel v-if="detailActiveTab === 'chat'" :tenant-id="selectedTenant.id" :models="availableModels" />
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
                  <a-form-item :label="t('admin.tenants.tokenQuota')">
                    <a-input-number v-model:value="selectedTenant.token_quota" :min="1000" :step="1000" style="width: 100%;" size="large" />
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
                  <span>{{ t('admin.tenants.usedTokens', [selectedTenant.token_used.toLocaleString(), selectedTenant.token_quota.toLocaleString()]) }}</span>
                  <span :style="{ color: getQuotaColor(getQuotaPercent(selectedTenant.token_used, selectedTenant.token_quota)) }">
                    {{ getQuotaPercent(selectedTenant.token_used, selectedTenant.token_quota) }}%
                  </span>
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
                    <a-button type="text" size="small" @click="exportEmbedScript('audit')"><DownloadOutlined /></a-button>
                  </template>
                </a-input>
              </a-form-item>
              <a-form-item :label="t('admin.tenants.embedSummaryUrl')">
                <a-input :value="embedSummaryUrl" readonly>
                  <template #suffix>
                    <a-button type="text" size="small" @click="copyText(embedSummaryUrl)"><CopyOutlined /></a-button>
                    <a-button type="text" size="small" @click="exportEmbedScript('summary')"><DownloadOutlined /></a-button>
                  </template>
                </a-input>
              </a-form-item>
              <div class="embed-script-panel">
                <div class="embed-script-header">
                  <div>
                    <div class="embed-script-title">{{ t('admin.tenants.embedScriptExportTitle') }}</div>
                    <div class="embed-script-desc">{{ t('admin.tenants.embedScriptExportDesc') }}</div>
                  </div>
                  <a-button type="primary" :disabled="!selectedTenant.embed_token_configured" @click="exportEmbedScript()">
                    <DownloadOutlined /> {{ t('admin.tenants.exportEmbedScript') }}
                  </a-button>
                </div>
                <div class="embed-script-fields">
                  <div class="embed-script-token-field">
                    <a-form-item :label="t('admin.tenants.embedScriptTokenInput')">
                      <a-input-password
                        v-model:value="manualEmbedToken"
                        autocomplete="off"
                      />
                      <div class="form-hint">{{ t('admin.tenants.embedScriptTokenHint') }}</div>
                    </a-form-item>
                  </div>
                  <div class="embed-script-target-field">
                    <a-form-item :label="t('admin.tenants.embedScriptTarget')">
                      <a-radio-group v-model:value="embedScriptTarget" button-style="solid" class="embed-script-target-options">
                        <a-radio-button v-for="opt in embedScriptTargetOptions" :key="opt.value" :value="opt.value">
                          {{ opt.label }}
                        </a-radio-button>
                      </a-radio-group>
                      <div class="form-hint">{{ t('admin.tenants.embedScriptTargetHint') }}</div>
                    </a-form-item>
                  </div>
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
      <div class="embed-token-modal-actions">
        <a-button @click="exportEmbedScript('audit')"><DownloadOutlined /> {{ t('admin.tenants.embedScriptTargetAudit') }}</a-button>
        <a-button @click="exportEmbedScript('summary')"><DownloadOutlined /> {{ t('admin.tenants.embedScriptTargetSummary') }}</a-button>
        <a-button type="primary" @click="exportEmbedScript('all')"><DownloadOutlined /> {{ t('admin.tenants.embedScriptTargetAll') }}</a-button>
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
  border-radius: var(--radius-lg); margin-bottom: 24px; flex-wrap: nowrap;
  overflow-x: auto; -webkit-overflow-scrolling: touch; scrollbar-width: none;
}
.detail-tabs::-webkit-scrollbar { display: none; }
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
  display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 16px; align-items: start;
}
.embed-script-token-field,
.embed-script-target-field { min-width: 0; }
.embed-script-target-options {
  display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); width: 100%;
}
.embed-script-target-options :deep(.ant-radio-button-wrapper) {
  margin-inline-start: 0; padding: 0 10px; text-align: center;
}
.embed-script-target-options :deep(.ant-radio-button-wrapper span:last-child) {
  display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
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
