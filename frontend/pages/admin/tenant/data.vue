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
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type { AuditLog, CronLog, ArchiveLog } from '~/composables/useMockData'

definePageMeta({ middleware: 'auth', layout: 'default' })

const { mockAuditLogs, mockCronLogs, mockArchiveLogs } = useMockData()

const topTab = ref<'audit' | 'cron' | 'archive'>('audit')

// ===== Audit logs =====
const auditLogs = ref<AuditLog[]>(JSON.parse(JSON.stringify(mockAuditLogs)))
const auditSearch = ref('')
const auditActionFilter = ref<string>('')

const filteredAuditLogs = computed(() => {
  return auditLogs.value.filter(l => {
    if (auditSearch.value && !l.title.includes(auditSearch.value) && !l.process_id.includes(auditSearch.value) && !l.operator.includes(auditSearch.value)) return false
    if (auditActionFilter.value && l.action !== auditActionFilter.value) return false
    return true
  })
})

const auditActionOptions = [
  { value: 'ai_audit', label: 'AI 审核' },
  { value: 'manual_approve', label: '手动通过' },
  { value: 'manual_reject', label: '手动驳回' },
  { value: 'feedback', label: '反馈' },
]

// ===== Cron logs =====
const cronLogs = ref<CronLog[]>(JSON.parse(JSON.stringify(mockCronLogs)))
const cronSearch = ref('')
const cronStatusFilter = ref<string>('')

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
const archiveActionFilter = ref<string>('')

const filteredArchiveLogs = computed(() => {
  return archiveLogs.value.filter(l => {
    if (archiveSearch.value && !l.title.includes(archiveSearch.value) && !l.process_id.includes(archiveSearch.value)) return false
    if (archiveActionFilter.value && l.action !== archiveActionFilter.value) return false
    return true
  })
})

const archiveActionOptions = [
  { value: 're_audit', label: '合规复核' },
  { value: 'export', label: '导出' },
  { value: 'view', label: '查看' },
]

const handleExport = (tab: string) => {
  message.success(`${tab}数据导出中...`)
}

const handleDeleteLog = (tab: 'audit' | 'cron' | 'archive', id: string) => {
  if (tab === 'audit') auditLogs.value = auditLogs.value.filter(l => l.id !== id)
  else if (tab === 'cron') cronLogs.value = cronLogs.value.filter(l => l.id !== id)
  else archiveLogs.value = archiveLogs.value.filter(l => l.id !== id)
  message.success('已删除')
}
</script>

<template>
  <div class="data-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">数据信息</h1>
        <p class="page-subtitle">统一管理审核记录、任务日志与归档操作历史</p>
      </div>
    </div>

    <!-- Top tabs -->
    <div class="tab-nav">
      <button
        v-for="tab in [
          { key: 'audit', label: '审核工作台', icon: FileTextOutlined },
          { key: 'cron', label: '定时任务', icon: ClockCircleOutlined },
          { key: 'archive', label: '归档复盘', icon: FolderOpenOutlined },
        ]"
        :key="tab.key"
        class="tab-btn"
        :class="{ 'tab-btn--active': topTab === tab.key }"
        @click="topTab = tab.key as any"
      >
        <component :is="tab.icon" style="font-size: 14px;" />
        {{ tab.label }}
      </button>
    </div>

    <!-- ===== Audit Logs Tab ===== -->
    <div v-if="topTab === 'audit'" class="tab-content">
      <div class="toolbar">
        <div class="toolbar-left">
          <a-input v-model:value="auditSearch" placeholder="搜索流程/操作人" allow-clear style="width: 220px;">
            <template #prefix><SearchOutlined /></template>
          </a-input>
          <a-select v-model:value="auditActionFilter" placeholder="操作类型" allow-clear style="width: 140px;">
            <a-select-option v-for="o in auditActionOptions" :key="o.value" :value="o.value">{{ o.label }}</a-select-option>
          </a-select>
        </div>
        <a-button @click="handleExport('审核日志')"><ExportOutlined /> 导出</a-button>
      </div>

      <div class="stats-row">
        <div class="stat-card">
          <div class="stat-value">{{ auditLogs.length }}</div>
          <div class="stat-label">总记录数</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ auditLogs.filter(l => l.action === 'ai_audit').length }}</div>
          <div class="stat-label">AI 审核</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ auditLogs.filter(l => l.action === 'manual_approve').length }}</div>
          <div class="stat-label">手动通过</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ auditLogs.filter(l => l.action === 'manual_reject').length }}</div>
          <div class="stat-label">手动驳回</div>
        </div>
      </div>

      <div class="data-table-card">
        <table class="data-table">
          <thead>
            <tr>
              <th>流程编号</th>
              <th>流程标题</th>
              <th>操作人</th>
              <th>操作类型</th>
              <th>结果</th>
              <th>时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="l in filteredAuditLogs" :key="l.id">
              <td class="text-mono">{{ l.process_id }}</td>
              <td>{{ l.title }}</td>
              <td>{{ l.operator }}</td>
              <td><span class="action-tag" :class="'action-tag--' + l.action">{{ l.action_label }}</span></td>
              <td class="text-secondary">{{ l.result }}</td>
              <td class="text-secondary">{{ l.created_at }}</td>
              <td>
                <div class="action-btns">
                  <button class="icon-btn" title="查看详情"><EyeOutlined /></button>
                  <a-popconfirm title="确认删除？" @confirm="handleDeleteLog('audit', l.id)">
                    <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
                  </a-popconfirm>
                </div>
              </td>
            </tr>
            <tr v-if="filteredAuditLogs.length === 0">
              <td colspan="7" class="empty-cell">暂无数据</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- ===== Cron Logs Tab ===== -->
    <div v-if="topTab === 'cron'" class="tab-content">
      <div class="toolbar">
        <div class="toolbar-left">
          <a-input v-model:value="cronSearch" placeholder="搜索任务" allow-clear style="width: 220px;">
            <template #prefix><SearchOutlined /></template>
          </a-input>
          <a-select v-model:value="cronStatusFilter" placeholder="执行状态" allow-clear style="width: 140px;">
            <a-select-option value="success">成功</a-select-option>
            <a-select-option value="failed">失败</a-select-option>
            <a-select-option value="running">运行中</a-select-option>
          </a-select>
        </div>
        <a-button @click="handleExport('任务日志')"><ExportOutlined /> 导出</a-button>
      </div>

      <div class="stats-row">
        <div class="stat-card">
          <div class="stat-value">{{ cronLogs.length }}</div>
          <div class="stat-label">总执行次数</div>
        </div>
        <div class="stat-card stat-card--success">
          <div class="stat-value">{{ cronLogs.filter(l => l.status === 'success').length }}</div>
          <div class="stat-label">成功</div>
        </div>
        <div class="stat-card stat-card--danger">
          <div class="stat-value">{{ cronLogs.filter(l => l.status === 'failed').length }}</div>
          <div class="stat-label">失败</div>
        </div>
      </div>

      <div class="data-table-card">
        <table class="data-table">
          <thead>
            <tr>
              <th>任务ID</th>
              <th>任务类型</th>
              <th>状态</th>
              <th>接收人</th>
              <th>开始时间</th>
              <th>结束时间</th>
              <th>消息</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="l in filteredCronLogs" :key="l.id">
              <td class="text-mono">{{ l.task_id }}</td>
              <td>{{ l.task_label }}</td>
              <td>
                <span class="status-tag" :class="'status-tag--' + l.status">
                  <CheckCircleOutlined v-if="l.status === 'success'" />
                  <CloseCircleOutlined v-else-if="l.status === 'failed'" />
                  <SyncOutlined v-else spin />
                  {{ l.status === 'success' ? '成功' : l.status === 'failed' ? '失败' : '运行中' }}
                </span>
              </td>
              <td class="text-secondary">{{ l.recipients }}</td>
              <td class="text-secondary">{{ l.started_at }}</td>
              <td class="text-secondary">{{ l.finished_at || '-' }}</td>
              <td class="text-secondary">{{ l.message }}</td>
              <td>
                <div class="action-btns">
                  <a-popconfirm title="确认删除？" @confirm="handleDeleteLog('cron', l.id)">
                    <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
                  </a-popconfirm>
                </div>
              </td>
            </tr>
            <tr v-if="filteredCronLogs.length === 0">
              <td colspan="8" class="empty-cell">暂无数据</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- ===== Archive Logs Tab ===== -->
    <div v-if="topTab === 'archive'" class="tab-content">
      <div class="toolbar">
        <div class="toolbar-left">
          <a-input v-model:value="archiveSearch" placeholder="搜索流程" allow-clear style="width: 220px;">
            <template #prefix><SearchOutlined /></template>
          </a-input>
          <a-select v-model:value="archiveActionFilter" placeholder="操作类型" allow-clear style="width: 140px;">
            <a-select-option v-for="o in archiveActionOptions" :key="o.value" :value="o.value">{{ o.label }}</a-select-option>
          </a-select>
        </div>
        <a-button @click="handleExport('归档日志')"><ExportOutlined /> 导出</a-button>
      </div>

      <div class="stats-row">
        <div class="stat-card">
          <div class="stat-value">{{ archiveLogs.length }}</div>
          <div class="stat-label">总记录数</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ archiveLogs.filter(l => l.action === 're_audit').length }}</div>
          <div class="stat-label">合规复核</div>
        </div>
        <div class="stat-card">
          <div class="stat-value">{{ archiveLogs.filter(l => l.action === 'export').length }}</div>
          <div class="stat-label">导出</div>
        </div>
      </div>

      <div class="data-table-card">
        <table class="data-table">
          <thead>
            <tr>
              <th>流程编号</th>
              <th>流程标题</th>
              <th>操作人</th>
              <th>操作类型</th>
              <th>合规结果</th>
              <th>时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="l in filteredArchiveLogs" :key="l.id">
              <td class="text-mono">{{ l.process_id }}</td>
              <td>{{ l.title }}</td>
              <td>{{ l.operator }}</td>
              <td><span class="action-tag" :class="'action-tag--' + l.action">{{ l.action_label }}</span></td>
              <td class="text-secondary">{{ l.compliance }}</td>
              <td class="text-secondary">{{ l.created_at }}</td>
              <td>
                <div class="action-btns">
                  <button class="icon-btn" title="查看详情"><EyeOutlined /></button>
                  <a-popconfirm title="确认删除？" @confirm="handleDeleteLog('archive', l.id)">
                    <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
                  </a-popconfirm>
                </div>
              </td>
            </tr>
            <tr v-if="filteredArchiveLogs.length === 0">
              <td colspan="7" class="empty-cell">暂无数据</td>
            </tr>
          </tbody>
        </table>
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
  .data-table-card { overflow-x: auto; }
}
</style>
