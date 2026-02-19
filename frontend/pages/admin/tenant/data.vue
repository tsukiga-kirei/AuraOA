<script setup lang="ts">
import {
  SearchOutlined,
  FileTextOutlined,
  ClockCircleOutlined,
  FolderOpenOutlined,
  ExportOutlined,
  DeleteOutlined,
  EyeOutlined,
  ReloadOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  SyncOutlined,
  AppstoreOutlined, // Added for new tab icon
} from '@ant-design/icons-vue'
import { message, Modal } from 'ant-design-vue'
import * as XLSX from 'xlsx'
import { useI18n } from '~/composables/useI18n'
import { usePagination } from '~/composables/usePagination'
import {
  mockAuditLogs,
  mockCronLogs, // Kept this as mockCronLogs, not mockCronTaskLogs as in the instruction, to match existing variable name
  mockArchiveLogs
} from '~/composables/useMockData'
import type { AuditLog, CronLog, ArchiveLog } from '~/composables/useMockData' // Keep types

definePageMeta({ middleware: 'auth', layout: 'default' })

// Removed: const { mockAuditLogs, mockCronLogs, mockArchiveLogs } = useMockData() as it's now imported directly

const activeTab = ref<'audit' | 'cron' | 'archive'>('audit') // Renamed topTab to activeTab

// ===== Audit logs =====
const auditLogs = ref<AuditLog[]>(JSON.parse(JSON.stringify(mockAuditLogs)))
const auditSearch = ref('')
const auditActionFilter = ref<string | undefined>(undefined)

const filteredAuditLogs = computed(() => {
  return auditLogs.value.filter(l => {
    if (auditSearch.value && !l.title.includes(auditSearch.value) && !l.process_id.includes(auditSearch.value) && !l.operator.includes(auditSearch.value)) return false
    if (auditActionFilter.value && l.action !== auditActionFilter.value) return false
    return true
  })
})

// ===== Cron logs =====
const cronLogs = ref<CronLog[]>(JSON.parse(JSON.stringify(mockCronLogs)))
const cronSearch = ref('')
const cronStatusFilter = ref<string | undefined>(undefined)

const filteredCronLogs = computed(() => {
  return cronLogs.value.filter(l => {
    if (cronSearch.value && !l.task_label.includes(cronSearch.value) && !l.task_id.includes(cronSearch.value)) return false
    if (cronStatusFilter.value && l.status !== cronStatusFilter.value) return false
    return true
  })
})

// ===== Archive logs =====
const archiveLogs = ref<ArchiveLog[]>(JSON.parse(JSON.stringify(mockArchiveLogs)))
const archiveSearch = ref('')
const archiveActionFilter = ref<string | undefined>(undefined)

const filteredArchiveLogs = computed(() => {
  return archiveLogs.value.filter(l => {
    if (archiveSearch.value && !l.title.includes(archiveSearch.value) && !l.process_id.includes(archiveSearch.value)) return false
    if (archiveActionFilter.value && l.action !== archiveActionFilter.value) return false
    return true
  })
})

// Pagination for logs
const auditLogPagination = usePagination(filteredAuditLogs, 10)
const cronLogPagination = usePagination(filteredCronLogs, 10)
const archiveLogPagination = usePagination(filteredArchiveLogs, 10)

const { t } = useI18n()

// Options for filters
const auditActionOptions = computed(() => [
  { value: 'ai_audit', label: t('admin.data.aiAudit') },
  { value: 'manual_approve', label: t('admin.data.manualApprove') },
  { value: 'manual_reject', label: t('admin.data.manualReject') },
  { value: 'feedback', label: t('admin.data.feedback') },
])
const archiveActionOptions = computed(() => [
  { value: 're_audit', label: t('admin.data.reAudit') },
  { value: 'export', label: t('admin.data.exportAction') },
  { value: 'view', label: t('admin.data.viewAction') },
])

// Mapping labels for display
const auditActionMap = computed(() => auditActionOptions.value.reduce((acc, cur) => {
  acc[cur.value] = cur.label
  return acc
}, {} as Record<string, string>))

const archiveActionMap = computed(() => archiveActionOptions.value.reduce((acc, cur) => {
  acc[cur.value] = cur.label
  return acc
}, {} as Record<string, string>))

const handleExport = (type: 'audit' | 'cron' | 'archive') => {
  if (type === 'audit') {
    message.loading(t('admin.data.exportingAudit'), 1)
    setTimeout(() => {
      const ws = XLSX.utils.json_to_sheet(auditLogPagination.paged.value)
      const wb = XLSX.utils.book_new()
      XLSX.utils.book_append_sheet(wb, ws, 'AuditLogs')
      XLSX.writeFile(wb, `audit_logs_${new Date().getTime()}.xlsx`)
    }, 1000)
  } else if (type === 'cron') {
    message.loading(t('admin.data.exportingCron'), 1)
    setTimeout(() => {
      const ws = XLSX.utils.json_to_sheet(cronLogPagination.paged.value)
      const wb = XLSX.utils.book_new()
      XLSX.utils.book_append_sheet(wb, ws, 'CronLogs')
      XLSX.writeFile(wb, `cron_logs_${new Date().getTime()}.xlsx`)
    }, 1000)
  } else {
    message.loading(t('admin.data.exportingArchive'), 1)
    setTimeout(() => {
      const ws = XLSX.utils.json_to_sheet(archiveLogPagination.paged.value)
      const wb = XLSX.utils.book_new()
      XLSX.utils.book_append_sheet(wb, ws, 'ArchiveLogs')
      XLSX.writeFile(wb, `archive_logs_${new Date().getTime()}.xlsx`)
    }, 1000)
  }
}

const handleDeleteLog = (id: string, type: 'audit' | 'cron' | 'archive') => {
  Modal.confirm({
    title: t('admin.data.confirmDelete'),
    onOk() {
      if (type === 'audit') auditLogs.value = auditLogs.value.filter(l => l.id !== id)
      else if (type === 'cron') cronLogs.value = cronLogs.value.filter(l => l.id !== id)
      else archiveLogs.value = archiveLogs.value.filter(l => l.id !== id)
      message.success(t('admin.data.deleted'))
    }
  })
}
</script>

<template>
  <div class="data-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('admin.data.title') }}</h1>
        <p class="page-subtitle">{{ t('admin.data.subtitle') }}</p>
      </div>
    </div>

    <!-- Top tabs -->
    <div class="tab-nav">
      <button
        v-for="tab in [
          { key: 'audit', label: t('admin.data.tabAudit'), icon: AppstoreOutlined },
          { key: 'cron', label: t('admin.data.tabCron'), icon: ClockCircleOutlined },
          { key: 'archive', label: t('admin.data.tabArchive'), icon: FolderOpenOutlined },
        ]"
        :key="tab.key"
        class="tab-btn"
        :class="{ 'tab-btn--active': activeTab === tab.key }"
        @click="activeTab = tab.key as any"
      >
        <component :is="tab.icon" style="font-size: 14px;" />
        {{ tab.label }}
      </button>
    </div>

    <!-- ===== Audit Logs Tab ===== -->
    <div v-if="activeTab === 'audit'" class="tab-content">
      <div class="toolbar">
        <div class="toolbar-left">
          <a-input v-model:value="auditSearch" :placeholder="t('admin.data.searchAudit')" allow-clear style="width: 220px;">
            <template #prefix><SearchOutlined /></template>
          </a-input>
          <a-select v-model:value="auditActionFilter" :placeholder="t('admin.data.actionType')" allow-clear style="width: 140px;">
            <a-select-option v-for="o in auditActionOptions" :key="o.value" :value="o.value">{{ o.label }}</a-select-option>
          </a-select>
        </div>
        <a-button @click="handleExport('audit')"><ExportOutlined /> {{ t('admin.data.export') }}</a-button>
      </div>

      <div class="stats-row">
        <div class="stat-card">
          <div class="stat-value">{{ auditLogs.length }}</div>
          <div class="stat-label">{{ t('admin.data.totalRecords') }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ auditLogs.filter(l => l.action === 'ai_audit').length }}</div>
          <div class="stat-label">{{ t('admin.data.aiAudit') }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ auditLogs.filter(l => l.action === 'manual_approve').length }}</div>
          <div class="stat-label">{{ t('admin.data.manualApprove') }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ auditLogs.filter(l => l.action === 'manual_reject').length }}</div>
          <div class="stat-label">{{ t('admin.data.manualReject') }}</div>
        </div>
      </div>

      <div class="data-table-card">
        <table class="data-table">
          <thead>
            <tr>
              <th>{{ t('admin.data.thProcessId') }}</th>
              <th>{{ t('admin.data.thProcessTitle') }}</th>
              <th>{{ t('admin.data.thOperator') }}</th>
              <th>{{ t('admin.data.thActionType') }}</th>
              <th>{{ t('admin.data.thResult') }}</th>
              <th>{{ t('admin.data.thTime') }}</th>
              <th>{{ t('admin.data.thAction') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="l in auditLogPagination.paged.value" :key="l.id">
              <td class="text-mono">{{ l.process_id }}</td>
              <td>{{ l.title }}</td>
              <td>{{ l.operator }}</td>
              <td><span class="action-tag" :class="'action-tag--' + l.action">{{ auditActionMap[l.action] }}</span></td>
              <td class="text-secondary">{{ l.result }}</td>
              <td class="text-secondary">{{ l.created_at }}</td>
              <td>
                <div class="action-btns">
                  <button class="icon-btn" :title="t('admin.data.viewDetail')"><EyeOutlined /></button>
                  <a-popconfirm :title="t('admin.data.confirmDelete')" @confirm="handleDeleteLog(l.id, 'audit')">
                    <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
                  </a-popconfirm>
                </div>
              </td>
            </tr>
            <tr v-if="auditLogPagination.paged.value.length === 0">
              <td colspan="7" class="empty-cell">{{ t('admin.data.noData') }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="pagination-wrapper">
        <a-pagination
          v-model:current="auditLogPagination.current.value"
          :page-size="auditLogPagination.pageSize.value"
          :total="auditLogPagination.total.value"
          size="small"
          show-size-changer
          show-quick-jumper
          :page-size-options="['10', '20', '50']"
          @change="auditLogPagination.onChange"
          @showSizeChange="auditLogPagination.onChange"
        />
      </div>
    </div>

    <!-- Tab Content: Cron -->
      <div v-if="activeTab === 'cron'" class="tab-content fade-in">
        <div class="toolbar">
          <div class="toolbar-left">
            <a-input v-model:value="cronSearch" :placeholder="t('admin.data.searchCron')" allow-clear style="width: 200px;">
              <template #prefix><SearchOutlined /></template>
            </a-input>
            <a-select v-model:value="cronStatusFilter" :placeholder="t('admin.data.execStatus')" allow-clear style="width: 140px;">
              <a-select-option value="success">{{ t('admin.data.success') }}</a-select-option>
              <a-select-option value="failed">{{ t('admin.data.failed') }}</a-select-option>
              <a-select-option value="running">{{ t('admin.data.running') }}</a-select-option>
            </a-select>
          </div>
          <div class="toolbar-right">
            <a-button @click="handleExport('cron')"><ExportOutlined /> {{ t('admin.data.export') }}</a-button>
          </div>
        </div>

      <div class="stats-row">
        <div class="stat-card">
          <div class="stat-value">{{ cronLogs.length }}</div>
          <div class="stat-label">{{ t('admin.data.totalExec') }}</div>
        </div>
        <div class="stat-card stat-card--success">
          <div class="stat-value">{{ cronLogs.filter(l => l.status === 'success').length }}</div>
          <div class="stat-label">{{ t('admin.data.success') }}</div>
        </div>
        <div class="stat-card stat-card--danger">
          <div class="stat-value">{{ cronLogs.filter(l => l.status === 'failed').length }}</div>
          <div class="stat-label">{{ t('admin.data.failed') }}</div>
        </div>
      </div>

      <div class="data-table-card">
        <table class="data-table">
          <thead>
            <tr>
              <th>{{ t('admin.data.thTaskId') }}</th>
              <th>{{ t('admin.data.thTaskType') }}</th>
              <th>{{ t('admin.data.thStatus') }}</th>
              <th>{{ t('admin.data.thRecipients') }}</th>
              <th>{{ t('admin.data.thStartTime') }}</th>
              <th>{{ t('admin.data.thEndTime') }}</th>
              <th>{{ t('admin.data.thMessage') }}</th>
              <th>{{ t('admin.data.thAction') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="l in cronLogPagination.paged.value" :key="l.id">
              <td class="text-mono">{{ l.task_id }}</td>
              <td>{{ l.task_label }}</td>
              <td>
                <span class="status-tag" :class="'status-tag--' + l.status">
                  <CheckCircleOutlined v-if="l.status === 'success'" />
                  <CloseCircleOutlined v-else-if="l.status === 'failed'" />
                  <SyncOutlined v-else spin />
                  {{ l.status === 'success' ? t('admin.data.success') : l.status === 'failed' ? t('admin.data.failed') : t('admin.data.running') }}
                </span>
              </td>
              <td class="text-secondary">{{ l.recipients }}</td>
              <td class="text-secondary">{{ l.started_at }}</td>
              <td class="text-secondary">{{ l.finished_at || '-' }}</td>
              <td class="text-secondary">{{ l.message }}</td>
              <td>
                <div class="action-btns">
                  <a-popconfirm :title="t('admin.data.confirmDelete')" @confirm="handleDeleteLog(l.id, 'cron')">
                    <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
                  </a-popconfirm>
                </div>
              </td>
            </tr>
            <tr v-if="cronLogPagination.paged.value.length === 0">
              <td colspan="8" class="empty-cell">{{ t('admin.data.noData') }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="pagination-wrapper">
        <a-pagination
          v-model:current="cronLogPagination.current.value"
          :page-size="cronLogPagination.pageSize.value"
          :total="cronLogPagination.total.value"
          size="small"
          show-size-changer
          show-quick-jumper
          :page-size-options="['10', '20', '50']"
          @change="cronLogPagination.onChange"
          @showSizeChange="cronLogPagination.onChange"
        />
      </div>
    </div>

    <!-- Tab Content: Archive -->
      <div v-if="activeTab === 'archive'" class="tab-content fade-in">
        <div class="toolbar">
          <div class="toolbar-left">
            <a-input v-model:value="archiveSearch" :placeholder="t('admin.data.searchArchive')" allow-clear style="width: 200px;">
              <template #prefix><SearchOutlined /></template>
            </a-input>
            <a-select v-model:value="archiveActionFilter" :placeholder="t('admin.data.actionType')" allow-clear style="width: 140px;">
            <a-select-option v-for="o in archiveActionOptions" :key="o.value" :value="o.value">{{ o.label }}</a-select-option>
          </a-select>
        </div>
        <a-button @click="handleExport('archive')"><ExportOutlined /> {{ t('admin.data.export') }}</a-button>
      </div>

      <div class="stats-row">
        <div class="stat-card">
          <div class="stat-value">{{ archiveLogs.length }}</div>
          <div class="stat-label">{{ t('admin.data.totalRecords') }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ archiveLogs.filter(l => l.action === 're_audit').length }}</div>
          <div class="stat-label">{{ t('admin.data.reAudit') }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ archiveLogs.filter(l => l.action === 'export').length }}</div>
          <div class="stat-label">{{ t('admin.data.exportAction') }}</div>
        </div>
      </div>

      <div class="data-table-card">
        <table class="data-table">
            <thead>
              <tr>
                <th>{{ t('admin.data.thProcessId') }}</th>
                <th>{{ t('admin.data.thProcessTitle') }}</th>
                <th>{{ t('admin.data.thOperator') }}</th>
                <th>{{ t('admin.data.thActionType') }}</th>
                <th>{{ t('admin.data.thCompliance') }}</th>
                <th>{{ t('admin.data.thTime') }}</th>
                <th>{{ t('admin.data.thAction') }}</th>
              </tr>
            </thead>
          <tbody>
            <tr v-for="l in archiveLogPagination.paged.value" :key="l.id">
              <td class="text-mono">{{ l.process_id }}</td>
              <td>{{ l.title }}</td>
              <td>{{ l.operator }}</td>
              <td><span class="action-tag" :class="'action-tag--' + l.action">{{ archiveActionMap[l.action] }}</span></td>
              <td class="text-secondary">{{ l.compliance }}</td>
              <td class="text-secondary">{{ l.created_at }}</td>
              <td>
                <div class="action-btns">
                  <button class="icon-btn" :title="t('admin.data.viewDetail')"><EyeOutlined /></button>
                  <a-popconfirm :title="t('admin.data.confirmDelete')" @confirm="handleDeleteLog(l.id, 'archive')">
                    <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
                  </a-popconfirm>
                </div>
              </td>
            </tr>
            <tr v-if="archiveLogPagination.paged.value.length === 0">
              <td colspan="7" class="empty-cell">{{ t('admin.data.noData') }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="pagination-wrapper">
        <a-pagination
          v-model:current="archiveLogPagination.current.value"
          :page-size="archiveLogPagination.pageSize.value"
          :total="archiveLogPagination.total.value"
          size="small"
          show-size-changer
          show-quick-jumper
          :page-size-options="['10', '20', '50']"
          @change="archiveLogPagination.onChange"
          @showSizeChange="archiveLogPagination.onChange"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.page-header { margin-bottom: 24px; }
.page-title { font-size: 24px; font-weight: 700; color: var(--color-text-primary); margin: 0; }
.page-subtitle { font-size: 14px; color: var(--color-text-tertiary); margin: 4px 0 0; }

.tab-nav {
  display: flex; gap: 4px; background: var(--color-bg-hover); padding: 4px;
  border-radius: var(--radius-lg); margin-bottom: 24px; width: fit-content;
}
.tab-btn {
  padding: 8px 20px; border: none; background: transparent; border-radius: var(--radius-md);
  font-size: 14px; font-weight: 500; color: var(--color-text-secondary); cursor: pointer;
  transition: all var(--transition-fast); display: flex; align-items: center; gap: 6px;
}
.tab-btn:hover { color: var(--color-text-primary); }
.tab-btn--active { background: var(--color-bg-card); color: var(--color-primary); box-shadow: var(--shadow-xs); }

.toolbar { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; gap: 12px; flex-wrap: wrap; }
.toolbar-left { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }

/* Stats row */
.stats-row { display: flex; gap: 12px; margin-bottom: 16px; flex-wrap: wrap; }
.stat-card {
  background: var(--color-bg-card); border-radius: var(--radius-md);
  border: 1px solid var(--color-border-light); padding: 14px 20px; min-width: 120px;
}
.stat-value { font-size: 22px; font-weight: 700; color: var(--color-text-primary); }
.stat-card--success .stat-value { color: var(--color-success); }
.stat-card--danger .stat-value { color: var(--color-danger); }
.stat-label { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }

/* Data table */
.data-table-card {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); overflow: hidden;
}
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

.action-tag {
  font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: var(--radius-full); white-space: nowrap;
}
.action-tag--ai_audit { background: var(--color-primary-bg); color: var(--color-primary); }
.action-tag--manual_approve { background: var(--color-success-bg); color: var(--color-success); }
.action-tag--manual_reject { background: var(--color-danger-bg); color: var(--color-danger); }
.action-tag--feedback { background: var(--color-info-bg); color: var(--color-info); }
.action-tag--re_audit { background: var(--color-primary-bg); color: var(--color-primary); }
.action-tag--export { background: var(--color-warning-bg); color: var(--color-warning); }
.action-tag--view { background: var(--color-bg-hover); color: var(--color-text-secondary); }

.status-tag {
  font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: var(--radius-full);
  display: inline-flex; align-items: center; gap: 4px;
}
.status-tag--success { background: var(--color-success-bg); color: var(--color-success); }
.status-tag--failed { background: var(--color-danger-bg); color: var(--color-danger); }
.status-tag--running { background: var(--color-primary-bg); color: var(--color-primary); }

.action-btns { display: flex; gap: 4px; }
.icon-btn {
  width: 28px; height: 28px; border: 1px solid var(--color-border); background: transparent;
  border-radius: var(--radius-sm); cursor: pointer; display: flex; align-items: center;
  justify-content: center; color: var(--color-text-tertiary); transition: all var(--transition-fast);
}
.icon-btn:hover { border-color: var(--color-primary); color: var(--color-primary); }
.icon-btn--danger:hover { border-color: var(--color-danger); color: var(--color-danger); }

@media (max-width: 768px) {
  .stats-row { flex-direction: column; }
  .data-table-card { overflow-x: auto; -webkit-overflow-scrolling: touch; }
  .data-table { min-width: 600px; }
  .toolbar { flex-direction: column; align-items: stretch; }
  .toolbar-left { flex-direction: column; }
  .toolbar-left > * { width: 100% !important; }
  .page-title { font-size: 20px; }
  .tab-nav { width: 100%; overflow-x: auto; -webkit-overflow-scrolling: touch; }
  .tab-btn { flex-shrink: 0; padding: 8px 14px; font-size: 13px; }
  .stat-card { min-width: auto; }
}
</style>
