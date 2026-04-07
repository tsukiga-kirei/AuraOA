<script setup lang="ts">
import {
  SearchOutlined,
  UserOutlined,
  EyeOutlined,
  ExportOutlined,
  AppstoreOutlined,
  ClockCircleOutlined,
  FolderOpenOutlined,
  ControlOutlined,
  NodeIndexOutlined,
  SwapOutlined,
  ReloadOutlined,
  MailOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import * as XLSX from 'xlsx'
import type { AdminUserConfigItem, AdminProcessDetail, AdminCronTaskDetail } from '~/types/user-config'
import { useI18n } from '~/composables/useI18n'
import { usePagination } from '~/composables/usePagination'
import { messages } from '~/locales'

definePageMeta({ middleware: 'auth', layout: 'default' })

const { t, te, locale } = useI18n()
const { configs, loading, listUserConfigs } = useAdminUserConfigApi()

// =====================================================================
// 数据加载
// =====================================================================
const loadConfigs = async () => {
  try { await listUserConfigs() }
  catch (e) { console.error('[user-configs] 加载失败', e) }
}

onMounted(loadConfigs)

// =====================================================================
// 过滤选项（从真实数据派生）
// =====================================================================
const departmentOptions = computed(() => {
  const depts = new Set(configs.value.map(c => c.department).filter(Boolean))
  return Array.from(depts).sort()
})

const roleOptions = computed(() => {
  const roles = new Set<string>()
  configs.value.forEach(c => c.role_names.forEach(r => roles.add(r)))
  return Array.from(roles).sort()
})

// =====================================================================
// 筛选逻辑
// =====================================================================
const search = ref('')
const deptFilter = ref<string | undefined>(undefined)
const roleFilter = ref<string | undefined>(undefined)
const hasConfigFilter = ref<string | undefined>(undefined)
const selectedIds = ref<string[]>([])

const totalChanges = (c: AdminUserConfigItem) =>
  c.audit_process_count + c.cron_task_count + c.archive_process_count

const filteredConfigs = computed(() =>
  configs.value.filter(c => {
    const q = search.value.trim().toLowerCase()
    if (q && !c.display_name.toLowerCase().includes(q) && !c.username.toLowerCase().includes(q)) return false
    if (deptFilter.value && c.department !== deptFilter.value) return false
    if (roleFilter.value && !c.role_names.includes(roleFilter.value)) return false
    const total = totalChanges(c)
    if (hasConfigFilter.value === 'configured' && total === 0) return false
    if (hasConfigFilter.value === 'none' && total > 0) return false
    return true
  })
)

const { paged, current, pageSize, total, onChange } = usePagination(filteredConfigs, 10)

// =====================================================================
// 统计卡
// =====================================================================
const totalAuditChanges = computed(() => configs.value.reduce((s, c) => s + c.audit_process_count, 0))
const totalCronChanges = computed(() => configs.value.reduce((s, c) => s + c.cron_task_count, 0))
const totalArchiveChanges = computed(() => configs.value.reduce((s, c) => s + c.archive_process_count, 0))

// =====================================================================
// 选择
// =====================================================================
const toggleSelect = (id: string) => {
  const idx = selectedIds.value.indexOf(id)
  if (idx >= 0) selectedIds.value.splice(idx, 1)
  else selectedIds.value.push(id)
}
const toggleSelectAll = () => {
  if (selectedIds.value.length === filteredConfigs.value.length) selectedIds.value = []
  else selectedIds.value = filteredConfigs.value.map(c => c.user_id)
}

// =====================================================================
// 导出
// =====================================================================
const handleExport = () => {
  if (selectedIds.value.length === 0) {
    message.warning(t('admin.userConfigs.selectToExport'))
    return
  }
  const data = configs.value
    .filter(c => selectedIds.value.includes(c.user_id))
    .map(c => ({
      username: c.username,
      display_name: c.display_name,
      department: c.department,
      roles: c.role_names.join(', '),
      audit_process_count: c.audit_process_count,
      cron_task_count: c.cron_task_count,
      archive_process_count: c.archive_process_count,
      last_modified: c.last_modified || '-',
    }))
  const ws = XLSX.utils.json_to_sheet(data)
  const wb = XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(wb, ws, 'UserPrefs')
  XLSX.writeFile(wb, `user_preferences_${Date.now()}.xlsx`)
  message.success(t('common.success'))
}

// =====================================================================
// 详情抽屉
// =====================================================================
const showDetail = ref(false)
const detailConfig = ref<AdminUserConfigItem | null>(null)
const detailTab = ref<'audit' | 'cron' | 'archive'>('audit')

const openDetail = (c: AdminUserConfigItem) => {
  detailConfig.value = c
  if (c.audit_details.length > 0) detailTab.value = 'audit'
  else if (c.cron_task_count > 0) detailTab.value = 'cron'
  else if (c.archive_details.length > 0) detailTab.value = 'archive'
  else detailTab.value = 'audit'
  showDetail.value = true
}

const strictnessLabels: Record<string, { label: string; color: string }> = {
  strict:   { label: t('admin.ruleConfig.strict'),   color: 'var(--color-danger)' },
  standard: { label: t('admin.ruleConfig.standard'), color: 'var(--color-primary)' },
  loose:    { label: t('admin.ruleConfig.loose'),    color: 'var(--color-warning)' },
}

// 判断某个流程详情是否有实质内容
const hasProcessContent = (proc: AdminProcessDetail) =>
  !!proc.strictness_override ||
  proc.custom_rules.length > 0 ||
  proc.field_overrides.length > 0 ||
  proc.rule_toggle_overrides.length > 0

const formatDateTime = (value?: string | null) => {
  if (!value) return '-'
  return new Date(value).toLocaleString(locale.value, { dateStyle: 'short', timeStyle: 'short' })
}

const describeCronExpression = (expr: string): string => {
  const map: Record<string, string> = {
    '0 9 * * 1-5': t('cron.describe.weekday9'),
    '0 18 * * 1-5': t('cron.describe.weekday18'),
    '0 2 * * *': t('cron.describe.daily2'),
    '0 10 * * 1': t('cron.describe.monday10'),
    '0 9 1 * *': t('cron.describe.monthly1_9'),
    '0 * * * *': t('cron.describe.hourly'),
    '0 12 * * *': t('cron.describe.daily12'),
    '0 16 * * *': t('cron.describe.daily16'),
  }
  return map[expr] || expr
}

const cronTaskTypeLabel = (task: AdminCronTaskDetail): string => {
  const key = `cron.taskType.${task.task_type}`
  return te(key) ? t(key) : (task.task_label || task.task_type)
}

const cronTaskDefaultLabels = (taskType: string): string[] => {
  const key = `cron.taskType.${taskType}`
  return [
    messages['zh-CN']?.[key],
    messages['en-US']?.[key],
    taskType,
  ].filter((label): label is string => !!label)
}

const cronTaskCustomLabel = (task: AdminCronTaskDetail): string => {
  const customLabel = task.task_label?.trim()
  if (!customLabel) return ''
  return cronTaskDefaultLabels(task.task_type).includes(customLabel) ? '' : customLabel
}

const cronTaskEmails = (task: AdminCronTaskDetail): string[] =>
  task.push_email.split(',').map(email => email.trim()).filter(Boolean)
</script>

<template>
  <div class="data-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('admin.userConfigs.title') }}</h1>
        <p class="page-subtitle">{{ t('admin.userConfigs.subtitle') }}</p>
      </div>
      <a-button :loading="loading" @click="loadConfigs">
        <ReloadOutlined /> {{ t('common.refresh') }}
      </a-button>
    </div>

    <!-- 统计卡 -->
    <div class="stats-row">
      <div class="stat-card stat-card--primary">
        <div class="stat-card-icon"><AppstoreOutlined /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ totalAuditChanges }}</span>
          <span class="stat-card-label">{{ t('admin.userConfigs.totalAuditChanges') }}</span>
        </div>
      </div>
      <div class="stat-card stat-card--info">
        <div class="stat-card-icon"><ClockCircleOutlined /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ totalCronChanges }}</span>
          <span class="stat-card-label">{{ t('admin.userConfigs.totalCronChanges') }}</span>
        </div>
      </div>
      <div class="stat-card stat-card--warning">
        <div class="stat-card-icon"><FolderOpenOutlined /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ totalArchiveChanges }}</span>
          <span class="stat-card-label">{{ t('admin.userConfigs.totalArchiveChanges') }}</span>
        </div>
      </div>
    </div>

    <!-- 工具栏 -->
    <div class="toolbar">
      <div class="toolbar-left">
        <a-input v-model:value="search" :placeholder="t('admin.userConfigs.searchPlaceholder')" allow-clear style="width: 200px;">
          <template #prefix><SearchOutlined /></template>
        </a-input>
        <a-select v-model:value="deptFilter" :placeholder="t('admin.userConfigs.department')" allow-clear style="width: 140px;">
          <a-select-option v-for="d in departmentOptions" :key="d" :value="d">{{ d }}</a-select-option>
        </a-select>
        <a-select v-model:value="roleFilter" :placeholder="t('admin.userConfigs.role')" allow-clear style="width: 140px;">
          <a-select-option v-for="r in roleOptions" :key="r" :value="r">{{ r }}</a-select-option>
        </a-select>
        <a-select v-model:value="hasConfigFilter" :placeholder="t('admin.userConfigs.configStatus')" allow-clear style="width: 140px;">
          <a-select-option value="configured">{{ t('admin.userConfigs.hasConfig') }}</a-select-option>
          <a-select-option value="none">{{ t('admin.userConfigs.noConfig') }}</a-select-option>
        </a-select>
      </div>
      <div class="toolbar-right">
        <span v-if="selectedIds.length > 0" class="batch-selected-hint">{{ t('admin.userConfigs.selected', `${selectedIds.length}`) }}</span>
        <a-button @click="handleExport"><ExportOutlined /> {{ t('admin.userConfigs.export') }}</a-button>
      </div>
    </div>

    <!-- 表格 -->
    <div class="data-table-card">
      <div v-if="loading" class="loading-cell">
        <a-spin />
      </div>
      <table v-else class="data-table">
        <thead>
          <tr>
            <th style="width: 40px;">
              <a-checkbox
                :checked="selectedIds.length > 0 && selectedIds.length === filteredConfigs.length"
                :indeterminate="selectedIds.length > 0 && selectedIds.length < filteredConfigs.length"
                @change="toggleSelectAll"
              />
            </th>
            <th>{{ t('admin.userConfigs.thUser') }}</th>
            <th>{{ t('admin.userConfigs.thDepartment') }}</th>
            <th>{{ t('admin.userConfigs.thRole') }}</th>
            <th>{{ t('admin.userConfigs.thAuditWorkbench') }}</th>
            <th>{{ t('admin.userConfigs.thCronConfig') }}</th>
            <th>{{ t('admin.userConfigs.thArchiveReview') }}</th>
            <th>{{ t('admin.userConfigs.thLastModified') }}</th>
            <th>{{ t('admin.userConfigs.thAction') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="c in paged" :key="c.user_id">
            <td>
              <a-checkbox :checked="selectedIds.includes(c.user_id)" @change="toggleSelect(c.user_id)" />
            </td>
            <td>
              <div class="user-cell">
                <a-avatar :size="28" class="user-avatar">
                  <template #icon><UserOutlined /></template>
                </a-avatar>
                <div>
                  <div class="user-name">{{ c.display_name || c.username }}</div>
                  <div class="user-username">{{ c.username }}</div>
                </div>
              </div>
            </td>
            <td class="text-secondary">{{ c.department || '-' }}</td>
            <td>
              <div class="role-tags">
                <span v-if="c.role_names.length === 0" class="text-secondary">-</span>
                <span v-for="r in c.role_names" :key="r" class="role-tag">{{ r }}</span>
              </div>
            </td>
            <td>
              <span v-if="c.audit_process_count > 0" class="count-badge count-badge--primary">
                {{ t('admin.userConfigs.processCount', [c.audit_process_count]) }}
              </span>
              <span v-else class="text-secondary">-</span>
            </td>
            <td>
              <span v-if="c.cron_task_count > 0" class="count-badge count-badge--info">
                {{ t('admin.userConfigs.taskCount', [c.cron_task_count]) }}
              </span>
              <span v-else class="text-secondary">-</span>
            </td>
            <td>
              <span v-if="c.archive_process_count > 0" class="count-badge count-badge--warning">
                {{ t('admin.userConfigs.processCount', [c.archive_process_count]) }}
              </span>
              <span v-else class="text-secondary">-</span>
            </td>
            <td class="text-secondary text-mono">{{ formatDateTime(c.last_modified) }}</td>
            <td>
              <div class="action-btns">
                <button class="icon-btn" :title="t('admin.userConfigs.viewDetail')" @click="openDetail(c)"><EyeOutlined /></button>
              </div>
            </td>
          </tr>
          <tr v-if="!loading && filteredConfigs.length === 0">
            <td colspan="9" class="empty-cell">{{ t('admin.userConfigs.noData') }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="pagination-wrapper">
      <a-pagination
        :current="current"
        :page-size="pageSize"
        :total="total"
        size="small"
        show-size-changer
        show-quick-jumper
        :page-size-options="['10', '20', '50']"
        @change="onChange"
        @showSizeChange="onChange"
      />
    </div>

    <!-- 详情抽屉 -->
    <a-drawer
      v-model:open="showDetail"
      :title="detailConfig ? t('admin.userConfigs.prefDetail', [detailConfig.display_name || detailConfig.username]) : ''"
      width="600"
      placement="right"
    >
      <template v-if="detailConfig">
        <!-- 用户头部 -->
        <div class="detail-user-header">
          <a-avatar :size="40" class="user-avatar">
            <template #icon><UserOutlined /></template>
          </a-avatar>
          <div>
            <div class="detail-user-name">{{ detailConfig.display_name || detailConfig.username }}</div>
            <div class="detail-user-meta">{{ detailConfig.username }} · {{ detailConfig.department || '-' }}</div>
            <div class="detail-user-roles">
              <span v-for="r in detailConfig.role_names" :key="r" class="role-tag role-tag--sm">{{ r }}</span>
            </div>
          </div>
        </div>

        <!-- 无配置状态 -->
        <div v-if="totalChanges(detailConfig) === 0" class="detail-empty">
          <a-empty :description="t('admin.userConfigs.noCustomConfig')" />
        </div>

        <!-- Tab 导航 -->
        <div v-else class="detail-tab-nav">
          <button
            v-for="tab in [
              { key: 'audit',   label: t('admin.userConfigs.tabAudit'),   icon: AppstoreOutlined,    count: detailConfig.audit_details.length },
              { key: 'cron',    label: t('admin.userConfigs.tabCron'),    icon: ClockCircleOutlined, count: detailConfig.cron_task_count },
              { key: 'archive', label: t('admin.userConfigs.tabArchive'), icon: FolderOpenOutlined,  count: detailConfig.archive_details.length },
            ]"
            :key="tab.key"
            class="detail-tab-btn"
            :class="{ 'detail-tab-btn--active': detailTab === tab.key }"
            @click="detailTab = tab.key as any"
          >
            <component :is="tab.icon" />
            {{ tab.label }}
            <span v-if="tab.count > 0" class="detail-tab-count">{{ tab.count }}</span>
          </button>
        </div>

        <!-- ===== 审核工作台 ===== -->
        <div v-if="detailTab === 'audit' && totalChanges(detailConfig) > 0" class="detail-content">
          <div v-if="detailConfig.audit_details.length === 0" class="detail-empty-tab">
            {{ t('admin.userConfigs.noAuditConfig') }}
          </div>
          <div v-for="proc in detailConfig.audit_details" :key="proc.process_type" class="detail-process-card">
            <div class="detail-process-header">
              <span class="detail-process-name">{{ proc.process_type }}</span>
            </div>

            <div v-if="proc.strictness_override" class="detail-config-block">
              <div class="detail-config-label"><ControlOutlined /> {{ t('admin.userConfigs.auditStrictness') }}</div>
              <div class="detail-config-value">
                <span class="strictness-tag" :style="{ color: strictnessLabels[proc.strictness_override]?.color }">
                  {{ strictnessLabels[proc.strictness_override]?.label || proc.strictness_override }}
                </span>
                <span class="text-secondary" style="font-size: 12px; margin-left: 4px;">{{ t('admin.userConfigs.userCustom') }}</span>
              </div>
            </div>

            <div v-if="proc.custom_rules.length > 0" class="detail-config-block">
              <div class="detail-config-label"><NodeIndexOutlined /> {{ t('admin.userConfigs.customRules') }}</div>
              <div class="detail-rule-list">
                <div v-for="rule in proc.custom_rules" :key="rule.id" class="detail-rule-item">
                  <span class="detail-rule-dot" :class="rule.enabled ? 'detail-rule-dot--on' : 'detail-rule-dot--off'" />
                  <span class="detail-rule-text">{{ rule.content }}</span>
                </div>
              </div>
            </div>

            <div v-if="proc.field_overrides.length > 0" class="detail-config-block">
              <div class="detail-config-label"><AppstoreOutlined /> {{ t('admin.userConfigs.fieldChanges') }}</div>
              <div class="field-override-list">
                <div v-for="f in proc.field_overrides" :key="f.field_key + f.table_name" class="field-override-item" :class="'field-override-item--' + f.status">
                  <span class="field-source">[{{ f.table_label }}]</span>
                  <span class="field-name">{{ f.field_name }}</span>
                  <span class="field-status-tag">{{ t(`admin.userConfigs.status.${f.status}`) }}</span>
                </div>
              </div>
            </div>


            <div v-if="proc.rule_toggle_overrides.length > 0" class="detail-config-block">
              <div class="detail-config-label"><SwapOutlined /> {{ t('admin.userConfigs.ruleToggleChanges') }}</div>
              <div class="detail-rule-list">
                <div v-for="r in proc.rule_toggle_overrides" :key="r.rule_id" class="detail-rule-item detail-rule-item--toggle">
                  <span class="detail-rule-dot" :class="r.enabled ? 'detail-rule-dot--on' : 'detail-rule-dot--off'" />
                  <span class="detail-rule-text">{{ r.rule_content || r.rule_id }}</span>
                  <span class="rule-toggle-compare">
                    <span class="rule-toggle-admin" :class="r.admin_enabled ? 'rule-toggle-status--on' : 'rule-toggle-status--off'">
                      {{ t('admin.userConfigs.adminDefault') }}:&nbsp;{{ r.admin_enabled ? t('admin.userConfigs.enabled') : t('admin.userConfigs.disabled') }}
                    </span>
                    <span class="rule-toggle-arrow">→</span>
                    <span
                      class="rule-toggle-user"
                      :class="[
                        r.enabled ? 'rule-toggle-status--on' : 'rule-toggle-status--off',
                        r.enabled !== r.admin_enabled ? 'rule-toggle-user--changed' : ''
                      ]"
                    >
                      {{ r.enabled !== r.admin_enabled ? t('admin.userConfigs.userOverride') : t('admin.userConfigs.unchanged') }}:&nbsp;{{ r.enabled ? t('admin.userConfigs.enabled') : t('admin.userConfigs.disabled') }}
                    </span>
                  </span>
                </div>
              </div>
            </div>

            <div v-if="!hasProcessContent(proc)" class="detail-empty-tab">
              {{ t('admin.userConfigs.noProcessConfig') }}
            </div>
          </div>
        </div>

        <!-- ===== 定时任务 ===== -->
        <div v-if="detailTab === 'cron' && totalChanges(detailConfig) > 0" class="detail-content">
          <div v-if="detailConfig.cron_task_count === 0" class="detail-empty-tab">
            {{ t('admin.userConfigs.noCronConfig') }}
          </div>
          <template v-else>
            <div v-for="task in detailConfig.cron_tasks" :key="task.id" class="detail-process-card">
              <div class="detail-process-header">
                <span class="detail-process-name">{{ cronTaskTypeLabel(task) }}</span>
                <span v-if="task.is_builtin" class="role-tag role-tag--sm">{{ t('cron.builtin') }}</span>
                <span class="count-badge" :class="task.is_active ? 'count-badge--success' : 'count-badge--muted'">
                  {{ task.is_active ? t('admin.userConfigs.cronActive') : t('admin.userConfigs.cronInactive') }}
                </span>
              </div>

              <div v-if="cronTaskCustomLabel(task)" class="detail-config-block">
                <div class="detail-config-label">{{ t('cron.taskLabel') }}</div>
                <div class="detail-config-value">{{ cronTaskCustomLabel(task) }}</div>
              </div>

              <div class="detail-config-block">
                <div class="detail-config-label"><ClockCircleOutlined /> {{ t('admin.userConfigs.cronExpression') }}</div>
                <div class="detail-config-value">
                  <code class="detail-inline-code">{{ task.cron_expression }}</code>
                  <span class="text-secondary detail-inline-desc">{{ describeCronExpression(task.cron_expression) }}</span>
                </div>
              </div>

              <div v-if="task.task_type.endsWith('_batch')" class="detail-config-block">
                <div class="detail-config-label"><AppstoreOutlined /> {{ t('admin.userConfigs.cronWorkflows') }}</div>
                <div class="detail-tag-list">
                  <span v-if="task.workflow_ids.length === 0" class="detail-field-tag detail-field-tag--neutral">{{ t('common.all') }}</span>
                  <template v-else>
                    <span v-for="workflow in task.workflow_ids" :key="workflow" class="detail-field-tag">{{ workflow }}</span>
                  </template>
                </div>
              </div>

              <div v-if="task.task_type.endsWith('_batch')" class="detail-config-block">
                <div class="detail-config-label"><ControlOutlined /> {{ t('admin.userConfigs.cronDateRange') }}</div>
                <div class="detail-config-value">{{ t(`cron.dateRange.${task.date_range || 30}`) }}</div>
              </div>

              <div v-if="cronTaskEmails(task).length > 0" class="detail-config-block">
                <div class="detail-config-label"><MailOutlined /> {{ t('admin.userConfigs.cronPushEmail') }}</div>
                <div class="detail-tag-list">
                  <span v-for="email in cronTaskEmails(task)" :key="email" class="detail-field-tag">{{ email }}</span>
                </div>
              </div>
            </div>
          </template>
        </div>

        <!-- ===== 归档复盘 ===== -->
        <div v-if="detailTab === 'archive' && totalChanges(detailConfig) > 0" class="detail-content">
          <div v-if="detailConfig.archive_details.length === 0" class="detail-empty-tab">
            {{ t('admin.userConfigs.noArchiveConfig') }}
          </div>
          <div v-for="arc in detailConfig.archive_details" :key="arc.process_type" class="detail-process-card">
            <div class="detail-process-header">
              <span class="detail-process-name">{{ arc.process_type }}</span>
            </div>

            <div v-if="arc.strictness_override" class="detail-config-block">
              <div class="detail-config-label"><ControlOutlined /> {{ t('admin.userConfigs.reviewStrictness') }}</div>
              <div class="detail-config-value">
                <span class="strictness-tag" :style="{ color: strictnessLabels[arc.strictness_override]?.color }">
                  {{ strictnessLabels[arc.strictness_override]?.label || arc.strictness_override }}
                </span>
                <span class="text-secondary" style="font-size: 12px; margin-left: 4px;">{{ t('admin.userConfigs.userCustom') }}</span>
              </div>
            </div>

            <div v-if="arc.custom_rules.length > 0" class="detail-config-block">
              <div class="detail-config-label"><NodeIndexOutlined /> {{ t('admin.userConfigs.customReviewRules') }}</div>
              <div class="detail-rule-list">
                <div v-for="rule in arc.custom_rules" :key="rule.id" class="detail-rule-item">
                  <span class="detail-rule-dot" :class="rule.enabled ? 'detail-rule-dot--on' : 'detail-rule-dot--off'" />
                  <span class="detail-rule-text">{{ rule.content }}</span>
                </div>
              </div>
            </div>

            <div v-if="arc.field_overrides.length > 0" class="detail-config-block">
              <div class="detail-config-label"><AppstoreOutlined /> {{ t('admin.userConfigs.fieldChanges') }}</div>
              <div class="field-override-list">
                <div v-for="f in arc.field_overrides" :key="f.field_key + f.table_name" class="field-override-item" :class="'field-override-item--' + f.status">
                  <span class="field-source">[{{ f.table_label }}]</span>
                  <span class="field-name">{{ f.field_name }}</span>
                  <span class="field-status-tag">{{ t(`admin.userConfigs.status.${f.status}`) }}</span>
                </div>
              </div>
            </div>


            <div v-if="arc.rule_toggle_overrides.length > 0" class="detail-config-block">
              <div class="detail-config-label"><SwapOutlined /> {{ t('admin.userConfigs.ruleToggleChanges') }}</div>
              <div class="detail-rule-list">
                <div v-for="r in arc.rule_toggle_overrides" :key="r.rule_id" class="detail-rule-item detail-rule-item--toggle">
                  <span class="detail-rule-dot" :class="r.enabled ? 'detail-rule-dot--on' : 'detail-rule-dot--off'" />
                  <span class="detail-rule-text">{{ r.rule_content || r.rule_id }}</span>
                  <span class="rule-toggle-compare">
                    <span class="rule-toggle-admin" :class="r.admin_enabled ? 'rule-toggle-status--on' : 'rule-toggle-status--off'">
                      {{ t('admin.userConfigs.adminDefault') }}:&nbsp;{{ r.admin_enabled ? t('admin.userConfigs.enabled') : t('admin.userConfigs.disabled') }}
                    </span>
                    <span class="rule-toggle-arrow">→</span>
                    <span
                      class="rule-toggle-user"
                      :class="[
                        r.enabled ? 'rule-toggle-status--on' : 'rule-toggle-status--off',
                        r.enabled !== r.admin_enabled ? 'rule-toggle-user--changed' : ''
                      ]"
                    >
                      {{ r.enabled !== r.admin_enabled ? t('admin.userConfigs.userOverride') : t('admin.userConfigs.unchanged') }}:&nbsp;{{ r.enabled ? t('admin.userConfigs.enabled') : t('admin.userConfigs.disabled') }}
                    </span>
                  </span>
                </div>
              </div>
            </div>

            <div v-if="!hasProcessContent(arc)" class="detail-empty-tab">
              {{ t('admin.userConfigs.noProcessConfig') }}
            </div>
          </div>
        </div>

        <!-- 页脚 -->
        <div v-if="detailConfig.last_modified" class="detail-footer-info">
          {{ t('admin.userConfigs.thLastModified') }}：{{ formatDateTime(detailConfig.last_modified) }}
        </div>
      </template>
    </a-drawer>
  </div>
</template>

<style scoped>
.page-header { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 24px; }
.page-title { font-size: 24px; font-weight: 700; color: var(--color-text-primary); margin: 0; }
.page-subtitle { font-size: 14px; color: var(--color-text-tertiary); margin: 4px 0 0; }

.toolbar { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; gap: 12px; flex-wrap: wrap; }
.toolbar-left { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.toolbar-right { display: flex; align-items: center; gap: 8px; }
.batch-selected-hint {
  font-size: 12px; font-weight: 500; color: var(--color-primary);
  padding: 2px 10px; border-radius: var(--radius-full);
  background: var(--color-primary-bg);
}

.stats-row { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; margin-bottom: 20px; }
.stat-card {
  background: var(--color-bg-card); border-radius: var(--radius-lg); padding: 20px;
  display: flex; align-items: center; gap: 16px; border: 2px solid var(--color-border-light);
  transition: all var(--transition-base);
}
.stat-card:hover { transform: translateY(-2px); box-shadow: var(--shadow-md); }
.stat-card-icon {
  width: 48px; height: 48px; border-radius: var(--radius-lg);
  display: flex; align-items: center; justify-content: center; font-size: 22px; flex-shrink: 0;
}
.stat-card--primary .stat-card-icon { background: var(--color-primary-bg); color: var(--color-primary); }
.stat-card--info .stat-card-icon { background: var(--color-info-bg); color: var(--color-info); }
.stat-card--warning .stat-card-icon { background: var(--color-warning-bg); color: var(--color-warning); }
.stat-card-info { display: flex; flex-direction: column; }
.stat-card-value { font-size: 28px; font-weight: 700; color: var(--color-text-primary); line-height: 1.2; }
.stat-card-label { font-size: 13px; color: var(--color-text-tertiary); margin-top: 2px; }

.data-table-card {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); overflow: hidden;
}
.loading-cell { padding: 48px; text-align: center; }
.data-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.data-table th {
  padding: 12px 16px; text-align: left; font-weight: 600; color: var(--color-text-secondary);
  background: var(--color-bg-page); border-bottom: 1px solid var(--color-border-light);
  font-size: 12px; text-transform: uppercase; letter-spacing: 0.04em; white-space: nowrap;
}
.data-table td {
  padding: 12px 16px; border-bottom: 1px solid var(--color-border-light);
  color: var(--color-text-primary);
}
.data-table tbody tr:hover { background: var(--color-bg-hover); }
.data-table tbody tr:last-child td { border-bottom: none; }
.text-secondary { color: var(--color-text-tertiary); }
.text-mono { font-family: monospace; font-size: 12px; color: var(--color-text-secondary); }
.empty-cell { text-align: center; padding: 32px 16px !important; color: var(--color-text-tertiary); }

.user-cell { display: flex; align-items: center; gap: 10px; }
.user-avatar { background: var(--color-primary-bg); color: var(--color-primary); flex-shrink: 0; }
.user-name { font-weight: 600; font-size: 13px; color: var(--color-text-primary); }
.user-username { font-size: 11px; color: var(--color-text-tertiary); font-family: monospace; }

.role-tags { display: flex; flex-wrap: wrap; gap: 4px; }
.role-tag {
  font-size: 11px; font-weight: 500; padding: 1px 8px; border-radius: var(--radius-full);
  background: var(--color-bg-hover); color: var(--color-text-secondary); white-space: nowrap;
}
.role-tag--sm { font-size: 10px; padding: 1px 6px; }

.count-badge {
  font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: var(--radius-full); white-space: nowrap;
}
.count-badge--primary { background: var(--color-primary-bg); color: var(--color-primary); }
.count-badge--info { background: var(--color-info-bg); color: var(--color-info); }
.count-badge--warning { background: var(--color-warning-bg); color: var(--color-warning); }
.count-badge--success { background: var(--color-success-bg); color: var(--color-success); }
.count-badge--muted { background: var(--color-bg-hover); color: var(--color-text-tertiary); }

.action-btns { display: flex; gap: 4px; }
.icon-btn {
  width: 28px; height: 28px; border: 1px solid var(--color-border); background: transparent;
  border-radius: var(--radius-sm); cursor: pointer; display: flex; align-items: center;
  justify-content: center; color: var(--color-text-tertiary); transition: all var(--transition-fast);
}
.icon-btn:hover { border-color: var(--color-primary); color: var(--color-primary); }

.pagination-wrapper { margin-top: 16px; display: flex; justify-content: flex-end; }

/* ===== 详情抽屉 ===== */
.detail-user-header {
  display: flex; align-items: flex-start; gap: 12px; margin-bottom: 20px;
  padding-bottom: 16px; border-bottom: 1px solid var(--color-border-light);
}
.detail-user-name { font-size: 16px; font-weight: 700; color: var(--color-text-primary); }
.detail-user-meta { font-size: 13px; color: var(--color-text-tertiary); }
.detail-user-roles { display: flex; flex-wrap: wrap; gap: 4px; margin-top: 6px; }

.detail-empty { padding: 40px 0; }
.detail-empty-tab {
  padding: 16px; text-align: center; color: var(--color-text-tertiary);
  font-size: 13px; background: var(--color-bg-page); border-radius: var(--radius-md);
}

.detail-tab-nav {
  display: flex; gap: 4px; background: var(--color-bg-hover); padding: 4px;
  border-radius: var(--radius-lg); margin-bottom: 16px;
}
.detail-tab-btn {
  padding: 6px 14px; border: none; background: transparent; border-radius: var(--radius-md);
  font-size: 13px; font-weight: 500; color: var(--color-text-secondary); cursor: pointer;
  transition: all var(--transition-fast); display: flex; align-items: center; gap: 6px; flex: 1;
  justify-content: center;
}
.detail-tab-btn:hover { color: var(--color-text-primary); }
.detail-tab-btn--active { background: var(--color-bg-card); color: var(--color-primary); box-shadow: var(--shadow-xs); }
.detail-tab-count {
  font-size: 10px; font-weight: 700; background: var(--color-primary-bg); color: var(--color-primary);
  padding: 1px 6px; border-radius: var(--radius-full); min-width: 18px; text-align: center;
}

.detail-content { display: flex; flex-direction: column; gap: 12px; }

.detail-process-card {
  background: var(--color-bg-page); border-radius: var(--radius-md);
  border: 1px solid var(--color-border-light); padding: 14px; display: flex; flex-direction: column; gap: 12px;
}
.detail-process-header { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.detail-process-name { font-size: 14px; font-weight: 600; color: var(--color-text-primary); }

.detail-config-block { display: flex; flex-direction: column; gap: 6px; }
.detail-config-label {
  font-size: 12px; font-weight: 600; color: var(--color-text-secondary);
  display: flex; align-items: center; gap: 6px;
}
.detail-config-value { font-size: 13px; color: var(--color-text-primary); padding-left: 20px; }
.detail-inline-code {
  font-size: 12px; padding: 2px 6px; border-radius: var(--radius-sm);
  background: var(--color-bg-card); border: 1px solid var(--color-border-light);
}
.detail-inline-desc { display: inline-block; margin-left: 8px; }
.strictness-tag { font-weight: 600; font-size: 13px; }

.detail-rule-list { display: flex; flex-direction: column; gap: 6px; padding-left: 20px; }
.detail-rule-item {
  display: flex; align-items: center; gap: 8px; font-size: 13px;
  padding: 6px 10px; background: var(--color-bg-card); border-radius: var(--radius-sm);
  border: 1px solid var(--color-border-light);
}
.detail-rule-dot {
  width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0;
}
.detail-rule-dot--on { background: var(--color-success); }
.detail-rule-dot--off { background: var(--color-text-tertiary); }
.detail-rule-text { flex: 1; color: var(--color-text-primary); line-height: 1.5; }

.rule-toggle-status {
  font-size: 10px; font-weight: 600; padding: 1px 6px;
  border-radius: var(--radius-full); white-space: nowrap; flex-shrink: 0;
}
.rule-toggle-status--on { background: var(--color-success-bg); color: var(--color-success); }
.rule-toggle-status--off { background: var(--color-bg-hover); color: var(--color-text-tertiary); }

/* 规则开关对比布局 */
.detail-rule-item--toggle { flex-wrap: wrap; align-items: center; gap: 6px 8px; }
.rule-toggle-compare {
  display: flex; align-items: center; gap: 4px; flex-shrink: 0; flex-wrap: wrap;
}
.rule-toggle-admin, .rule-toggle-user {
  font-size: 10px; font-weight: 600; padding: 1px 6px;
  border-radius: var(--radius-full); white-space: nowrap;
}
.rule-toggle-arrow {
  font-size: 11px; color: var(--color-text-tertiary); flex-shrink: 0;
}
.rule-toggle-user--changed {
  outline: 1.5px solid currentColor;
  outline-offset: 1px;
}

.detail-tag-list { display: flex; flex-wrap: wrap; gap: 6px; padding-left: 20px; }
.detail-email-list { display: flex; flex-wrap: wrap; gap: 6px; }
.detail-field-tag {
  font-size: 12px; font-weight: 500; padding: 3px 10px; border-radius: var(--radius-full);
  background: var(--color-info-bg); color: var(--color-info); border: 1px solid transparent;
}
.detail-field-tag--neutral { background: var(--color-bg-hover); color: var(--color-text-secondary); }

/* 字段覆盖增强样式 */
.field-override-list { display: flex; flex-direction: column; gap: 4px; padding-left: 20px; }
.field-override-item {
  display: flex; align-items: center; gap: 8px; font-size: 13px;
  padding: 6px 12px; border-radius: var(--radius-md);
  background: var(--color-bg-card); border: 1px solid var(--color-border-light);
}
.field-source { font-size: 12px; color: var(--color-text-tertiary); font-family: monospace; }
.field-name { flex: 1; font-weight: 500; color: var(--color-text-primary); }
.field-status-tag {
  font-size: 11px; font-weight: 600; padding: 1px 8px; border-radius: var(--radius-full);
}
.field-override-item--user_added { border-left: 3px solid var(--color-info); }
.field-override-item--user_added .field-status-tag { background: var(--color-info-bg); color: var(--color-info); }
.field-override-item--abandoned { border-left: 3px solid var(--color-danger); opacity: 0.7; }
.field-override-item--abandoned .field-status-tag { background: var(--color-danger-bg); color: var(--color-danger); }
.field-override-item--abandoned .field-name { text-decoration: line-through; }


.detail-footer-info {
  font-size: 12px; color: var(--color-text-tertiary);
  padding-top: 16px; margin-top: 16px; border-top: 1px solid var(--color-border-light);
}

@media (max-width: 768px) {
  .stats-row { grid-template-columns: 1fr; }
  .stat-card { padding: 14px; }
  .stat-card-value { font-size: 22px; }
  .stat-card-icon { width: 40px; height: 40px; font-size: 18px; }
  .data-table-card { overflow-x: auto; -webkit-overflow-scrolling: touch; }
  .data-table { min-width: 800px; }
  .toolbar { flex-direction: column; align-items: stretch; }
  .toolbar-left { flex-direction: column; }
  .toolbar-left > * { width: 100% !important; }
  .page-title { font-size: 20px; }
  .detail-tab-nav { flex-direction: column; }
}
</style>
