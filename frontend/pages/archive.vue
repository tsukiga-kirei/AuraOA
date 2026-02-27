<script setup lang="ts">
import {
  SearchOutlined,
  FilterOutlined,
  DownloadOutlined,
  SafetyCertificateOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ExportOutlined,
  ThunderboltOutlined,
  ReloadOutlined,
  FileProtectOutlined,
  FieldTimeOutlined,
  LoadingOutlined,
  FireOutlined,
  AlertOutlined,
  BulbOutlined,
  RightOutlined,
  HistoryOutlined,
  LockOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type { ArchivedProcess, ArchiveAuditResult, ArchiveReviewConfig } from '~/composables/useMockData'
import { useI18n } from '~/composables/useI18n'

definePageMeta({ middleware: 'auth' })

const { t } = useI18n()
const { currentUser } = useAuth()

const {
  mockArchivedProcesses,
  mockArchiveAuditResult,
  mockArchiveReviewConfigs,
  mockOrgMembers,
  mockOrgRoles,
} = useMockData()

// ===== Permission: filter accessible archive configs based on current user =====
const accessibleConfigs = computed<ArchiveReviewConfig[]>(() => {
  const uname = currentUser.value?.username || ''
  const member = mockOrgMembers.find(m => m.username === uname)
  if (!member) return []
  return mockArchiveReviewConfigs.filter(cfg => {
    // Check role-based access
    if (cfg.allowed_roles.includes(member.role_id)) return true
    // Check member-based access
    if (cfg.allowed_members.includes(member.id)) return true
    return false
  })
})

const accessibleProcessTypes = computed(() => accessibleConfigs.value.map(c => c.process_type))

// ===== Process list (filtered by accessible types) =====
const processList = ref<ArchivedProcess[]>(
  mockArchivedProcesses.filter(p => accessibleProcessTypes.value.includes(p.process_type))
)

// ===== View state =====
const selectedProcess = ref<ArchivedProcess | null>(null)
const loading = ref(false)
const phase1Done = ref(false)
const searchText = ref('')
const showFilters = ref(false)
const batchAuditing = ref(false)
const selectedProcessIds = ref<string[]>([])
const processAuditLoading = ref<Record<string, boolean>>({})

// ===== Filters =====
const filterProcessType = ref<string[]>([])
const filterDepartment = ref<string | undefined>(undefined)
const filterAuditStatus = ref<string | undefined>(undefined)

const departmentOptions = computed(() => [...new Set(processList.value.map(p => p.department))])
const processTypeOptions = computed(() =>
  accessibleConfigs.value.map(c => {
    const labelSuffix = c.process_type_label ? ` · ${c.process_type_label}` : ''
    return { label: c.process_type + labelSuffix, value: c.process_type }
  })
)

const hasActiveFilters = computed(() =>
  !!searchText.value || filterProcessType.value.length > 0 || !!filterDepartment.value || !!filterAuditStatus.value
)

const clearFilters = () => {
  searchText.value = ''
  filterProcessType.value = []
  filterDepartment.value = undefined
  filterAuditStatus.value = undefined
}

// ===== Audit results cache =====
const auditRecords = ref<Record<string, ArchiveAuditResult>>({})

const filteredList = computed(() => {
  let list = [...processList.value]
  if (filterProcessType.value.length > 0) {
    list = list.filter(p => filterProcessType.value.includes(p.process_type))
  }
  if (filterDepartment.value) {
    list = list.filter(p => p.department === filterDepartment.value)
  }
  if (searchText.value) {
    const q = searchText.value.toLowerCase()
    list = list.filter(p =>
      p.title.toLowerCase().includes(q) ||
      p.applicant.toLowerCase().includes(q) ||
      p.process_id.toLowerCase().includes(q)
    )
  }
  if (filterAuditStatus.value) {
    if (filterAuditStatus.value === 'unaudited') {
      list = list.filter(p => !auditRecords.value[p.process_id])
    } else {
      list = list.filter(p => auditRecords.value[p.process_id]?.overall_compliance === filterAuditStatus.value)
    }
  }
  return list
})

const { paged: pagedList, current: listPage, pageSize: listPageSize, total: listTotal, onChange: onListPageChange } = usePagination(filteredList, 10)

// ===== Selection =====
const toggleSelectProcess = (id: string) => {
  const idx = selectedProcessIds.value.indexOf(id)
  if (idx >= 0) selectedProcessIds.value.splice(idx, 1)
  else selectedProcessIds.value.push(id)
}

const toggleSelectAll = () => {
  if (selectedProcessIds.value.length === filteredList.value.length) {
    selectedProcessIds.value = []
  } else {
    selectedProcessIds.value = filteredList.value.map(p => p.process_id)
  }
}

// ===== Generate mock audit result =====
const generateAuditResult = (proc: ArchivedProcess): ArchiveAuditResult => {
  const complianceOptions: ArchiveAuditResult['overall_compliance'][] = ['compliant', 'non_compliant', 'partially_compliant']
  const hash = proc.process_id.split('').reduce((a, c) => a + c.charCodeAt(0), 0)
  const compliance = complianceOptions[hash % 3]
  const score = compliance === 'compliant' ? 85 + (hash % 15) : compliance === 'partially_compliant' ? 55 + (hash % 25) : 20 + (hash % 30)
  const cfg = accessibleConfigs.value.find(c => c.process_type === proc.process_type)
  const rules = cfg?.rules || []
  return {
    ...mockArchiveAuditResult,
    trace_id: `ATR-${Date.now().toString(36).toUpperCase()}`,
    process_id: proc.process_id,
    overall_compliance: compliance,
    overall_score: score,
    duration_ms: 1500 + (hash % 3000),
    rule_audit: rules.map((r, i) => ({
      rule_id: r.id,
      rule_name: r.rule_content.slice(0, 20) + (r.rule_content.length > 20 ? '...' : ''),
      passed: (hash + i) % 3 !== 0,
      reasoning: (hash + i) % 3 !== 0 ? '经核查，该规则项符合要求' : '经核查，该规则项存在不合规情况，需关注',
    })),
    flow_audit: {
      is_complete: hash % 4 !== 0,
      missing_nodes: hash % 4 === 0 ? ['财务总监审批'] : [],
      node_results: proc.flow_nodes.map((n, i) => ({
        node_id: n.node_id,
        node_name: n.node_name,
        compliant: (hash + i) % 5 !== 0 || n.action === 'approve',
        reasoning: n.action === 'approve' ? '审批节点完整，操作合规' : `节点 ${n.node_name} 存在退回操作，需关注`,
      })),
    },
    ai_summary: compliance === 'compliant'
      ? `该${proc.process_type}流程整体合规，审批链完整，规则校验全部通过，建议归档留存。`
      : compliance === 'partially_compliant'
        ? `该${proc.process_type}流程存在部分合规问题，规则校验有不通过项，建议关注并整改。`
        : `该${proc.process_type}流程存在较多合规问题，规则校验多项不通过，建议重点审查。`,
  }
}

// ===== Single audit =====
const currentResult = ref<ArchiveAuditResult | null>(null)

const selectProcess = (proc: ArchivedProcess) => {
  selectedProcess.value = proc
  currentResult.value = auditRecords.value[proc.process_id] || null
  phase1Done.value = !!currentResult.value
}

const handleAudit = async () => {
  if (!selectedProcess.value) return
  loading.value = true
  phase1Done.value = false
  currentResult.value = null
  await new Promise(r => setTimeout(r, 2200))
  phase1Done.value = true
  await new Promise(r => setTimeout(r, 1650))
  const result = generateAuditResult(selectedProcess.value)
  currentResult.value = result
  auditRecords.value[selectedProcess.value.process_id] = result
  loading.value = false
}

const handleReAudit = async () => {
  currentResult.value = null
  await handleAudit()
}

// ===== Batch audit =====
const handleBatchAudit = async () => {
  if (selectedProcessIds.value.length === 0) return
  batchAuditing.value = true
  const ids = [...selectedProcessIds.value]
  for (const id of ids) processAuditLoading.value[id] = true
  for (const id of ids) {
    await new Promise(r => setTimeout(r, 800 + Math.random() * 1000))
    const proc = processList.value.find(p => p.process_id === id)
    if (proc) {
      const result = generateAuditResult(proc)
      auditRecords.value[id] = result
      if (selectedProcess.value?.process_id === id) currentResult.value = result
    }
    processAuditLoading.value[id] = false
  }
  batchAuditing.value = false
  selectedProcessIds.value = []
  message.success(t('archive.batchDone', `${ids.length}`))
}

// ===== Export (only shown after process selected) =====
const handleExport = (format: string) => {
  message.success(t('archive.exporting', format.toUpperCase()))
}

const jumpToOA = (processId: string) => {
  message.info(t('archive.jumpingToOA', processId))
}

// ===== Config helpers =====
const complianceConfig = computed(() => ({
  compliant: { color: 'var(--color-success)', bg: 'var(--color-success-bg)', label: t('archive.compliant') },
  non_compliant: { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)', label: t('archive.nonCompliant') },
  partially_compliant: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', label: t('archive.partiallyCompliant') },
}))

const actionConfig = computed(() => ({
  approve: { color: 'var(--color-success)', label: t('archive.actionApprove') },
  return: { color: 'var(--color-warning)', label: t('archive.actionReturn') },
}))

const auditedCount = computed(() => filteredList.value.filter(p => auditRecords.value[p.process_id]).length)
</script>

<template>
  <div class="archive-page fade-in">
    <!-- Page header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('archive.title') }}</h1>
        <p class="page-subtitle">{{ t('archive.subtitle') }}</p>
      </div>
      <div class="page-header-actions">
        <!-- Export button: only shown when a process is selected -->
        <a-dropdown v-if="selectedProcess">
          <a-button>
            <DownloadOutlined /> {{ t('archive.exportReport') }}
          </a-button>
          <template #overlay>
            <a-menu>
              <a-menu-item key="json" @click="handleExport('json')">{{ t('archive.exportJSON') }}</a-menu-item>
              <a-menu-item key="csv" @click="handleExport('csv')">{{ t('archive.exportCSV') }}</a-menu-item>
              <a-menu-item key="excel" @click="handleExport('excel')">{{ t('archive.exportExcel') }}</a-menu-item>
            </a-menu>
          </template>
        </a-dropdown>
      </div>
    </div>

    <!-- Stats row -->
    <div class="stats-row">
      <div class="stat-card stat-card--primary" :class="{ 'stat-card--selected': !filterAuditStatus }" @click="filterAuditStatus = undefined">
        <div class="stat-card-icon"><FileProtectOutlined /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ filteredList.length }}</span>
          <span class="stat-card-label">{{ t('archive.statTotal') }}</span>
        </div>
      </div>
      <div class="stat-card stat-card--success" :class="{ 'stat-card--selected': filterAuditStatus === 'compliant' }" @click="filterAuditStatus = filterAuditStatus === 'compliant' ? undefined : 'compliant'">
        <div class="stat-card-icon"><CheckCircleOutlined /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ Object.values(auditRecords).filter(r => r.overall_compliance === 'compliant').length }}</span>
          <span class="stat-card-label">{{ t('archive.statCompliant') }}</span>
        </div>
      </div>
      <div class="stat-card stat-card--warning" :class="{ 'stat-card--selected': filterAuditStatus === 'partially_compliant' }" @click="filterAuditStatus = filterAuditStatus === 'partially_compliant' ? undefined : 'partially_compliant'">
        <div class="stat-card-icon"><AlertOutlined /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ Object.values(auditRecords).filter(r => r.overall_compliance === 'partially_compliant').length }}</span>
          <span class="stat-card-label">{{ t('archive.statPartial') }}</span>
        </div>
      </div>
      <div class="stat-card stat-card--danger" :class="{ 'stat-card--selected': filterAuditStatus === 'non_compliant' }" @click="filterAuditStatus = filterAuditStatus === 'non_compliant' ? undefined : 'non_compliant'">
        <div class="stat-card-icon"><CloseCircleOutlined /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ Object.values(auditRecords).filter(r => r.overall_compliance === 'non_compliant').length }}</span>
          <span class="stat-card-label">{{ t('archive.statNonCompliant') }}</span>
        </div>
      </div>
    </div>

    <!-- Main layout -->
    <div class="archive-grid">
      <!-- Left: Process list -->
      <div class="list-panel">
        <div class="panel-header">
          <div class="panel-header-row">
            <h3 class="panel-title">
              <FireOutlined style="color: var(--color-primary);" />
              {{ t('archive.archivedProcesses') }}
              <a-badge :count="filteredList.length" :number-style="{ backgroundColor: 'var(--color-primary)' }" />
            </h3>
            <a-button size="small" @click="showFilters = !showFilters" :class="{ 'filter-toggle-btn--active': hasActiveFilters }">
              <FilterOutlined /> {{ t('archive.filter') }}
              <span v-if="hasActiveFilters" class="filter-active-dot" />
            </a-button>
          </div>

          <!-- Filters -->
          <transition name="slide">
            <div v-if="showFilters" class="filter-bar">
              <a-input v-model:value="searchText" :placeholder="t('archive.searchPlaceholder')" allow-clear style="flex: 2; min-width: 140px;">
                <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
              </a-input>
              <a-select v-model:value="filterDepartment" :placeholder="t('archive.department')" allow-clear style="flex: 1; min-width: 100px;">
                <a-select-option v-for="d in departmentOptions" :key="d" :value="d">{{ d }}</a-select-option>
              </a-select>
              <a-select v-model:value="filterProcessType" mode="multiple" :placeholder="t('archive.processType')" allow-clear style="flex: 1; min-width: 100px;" :options="processTypeOptions" :max-tag-count="1" />
              <a-select v-model:value="filterAuditStatus" :placeholder="t('archive.filterStatus')" allow-clear style="flex: 1; min-width: 100px;">
                <a-select-option value="unaudited">{{ t('archive.statusUnaudited') }}</a-select-option>
                <a-select-option value="compliant">{{ t('archive.compliant') }}</a-select-option>
                <a-select-option value="partially_compliant">{{ t('archive.partiallyCompliant') }}</a-select-option>
                <a-select-option value="non_compliant">{{ t('archive.nonCompliant') }}</a-select-option>
              </a-select>
              <a-button size="small" @click="clearFilters">{{ t('archive.reset') }}</a-button>
            </div>
          </transition>

          <!-- Batch toolbar -->
          <div class="batch-toolbar">
            <a-checkbox
              :checked="selectedProcessIds.length === filteredList.length && filteredList.length > 0"
              :indeterminate="selectedProcessIds.length > 0 && selectedProcessIds.length < filteredList.length"
              @change="toggleSelectAll"
            >
              {{ selectedProcessIds.length > 0 ? t('archive.selected', `${selectedProcessIds.length}`) : t('archive.selectAll') }}
            </a-checkbox>
            <a-button v-if="selectedProcessIds.length > 0" type="primary" size="small" :disabled="batchAuditing" @click="handleBatchAudit">
              <LoadingOutlined v-if="batchAuditing" />
              <ThunderboltOutlined v-else />
              {{ t('archive.batchAudit') }}
            </a-button>
            <span v-if="auditedCount > 0" class="panel-header-hint">{{ t('archive.reviewed') }} {{ auditedCount }}/{{ filteredList.length }}</span>
          </div>
        </div>

        <!-- Process list -->
        <div class="process-list">
          <div
            v-for="proc in pagedList"
            :key="proc.process_id"
            class="process-item"
            :class="{
              'process-item--selected': selectedProcess?.process_id === proc.process_id,
              'process-item--compliant': auditRecords[proc.process_id]?.overall_compliance === 'compliant',
              'process-item--partial': auditRecords[proc.process_id]?.overall_compliance === 'partially_compliant',
              'process-item--noncompliant': auditRecords[proc.process_id]?.overall_compliance === 'non_compliant',
            }"
            @click="selectProcess(proc)"
          >
            <div class="process-item-checkbox" @click.stop="toggleSelectProcess(proc.process_id)">
              <a-checkbox :checked="selectedProcessIds.includes(proc.process_id)" />
            </div>
            <div class="process-item-main">
              <div class="process-item-title-row">
                <span class="process-item-title">{{ proc.title }}</span>
                <span
                  v-if="auditRecords[proc.process_id]"
                  class="process-audit-badge"
                  :style="{
                    color: complianceConfig[auditRecords[proc.process_id].overall_compliance]?.color,
                    background: complianceConfig[auditRecords[proc.process_id].overall_compliance]?.bg,
                  }"
                >
                  <SafetyCertificateOutlined />
                  {{ complianceConfig[auditRecords[proc.process_id].overall_compliance]?.label }}
                  {{ auditRecords[proc.process_id].overall_score }}{{ t('archive.score') }}
                </span>
              </div>
              <div class="process-item-meta">
                <span>{{ proc.applicant }}</span>
                <span class="meta-dot">·</span>
                <span>{{ proc.department }}</span>
                <span class="meta-dot">·</span>
                <span>{{ proc.submit_time }}</span>
              </div>
              <div class="process-item-footer">
                <span class="process-type-tag">{{ proc.process_type }}</span>
                <span v-if="processAuditLoading[proc.process_id]" class="process-auditing">
                  <LoadingOutlined style="font-size: 11px;" /> {{ t('archive.auditingItem') }}
                </span>
                <a-tooltip :title="t('archive.jumpOA')" :mouse-enter-delay="0.5">
                  <button class="oa-jump-btn" @click.stop="jumpToOA(proc.process_id)">
                    <ExportOutlined />
                  </button>
                </a-tooltip>
              </div>
            </div>
          </div>
          <div v-if="filteredList.length === 0" class="list-empty">
            <a-empty :description="t('archive.noMatch')" />
          </div>
        </div>

        <!-- Pagination -->
        <div class="pagination-wrapper">
          <a-pagination
            :current="listPage"
            :page-size="listPageSize"
            :total="listTotal"
            size="small"
            show-size-changer
            show-quick-jumper
            :page-size-options="['10', '20', '50']"
            @change="onListPageChange"
            @showSizeChange="onListPageChange"
          />
        </div>
      </div>

      <!-- Right: Detail panel -->
      <div class="detail-panel">
        <div class="panel-header">
          <h3 class="panel-title">
            <SafetyCertificateOutlined style="color: var(--color-primary);" />
            {{ t('archive.complianceTitle') }}
          </h3>
        </div>

        <div class="detail-content">
          <!-- Empty state -->
          <div v-if="!selectedProcess" class="detail-empty">
            <div class="detail-empty-icon"><SafetyCertificateOutlined /></div>
            <h4>{{ t('archive.selectProcess') }}</h4>
            <p>{{ t('archive.selectProcessDesc') }}</p>
          </div>

          <template v-else>
            <!-- Process info card -->
            <div class="process-info-card">
              <div class="process-info-header">
                <div>
                  <h4 class="process-info-title">{{ selectedProcess.title }}</h4>
                  <div class="process-info-meta">
                    {{ selectedProcess.applicant }} · {{ selectedProcess.department }} · {{ selectedProcess.process_type }}
                  </div>
                  <div class="process-info-meta" style="margin-top: 4px;">
                    <FieldTimeOutlined /> {{ t('archive.submitLabel') }}: {{ selectedProcess.submit_time }}
                    &nbsp;→&nbsp; {{ t('archive.archiveLabel') }}: {{ selectedProcess.archive_time }}
                  </div>
                </div>
                <div class="process-info-actions">
                  <a-button @click="jumpToOA(selectedProcess.process_id)">
                    <ExportOutlined /> OA
                  </a-button>
                  <a-button type="primary" :loading="loading" @click="currentResult ? handleReAudit() : handleAudit()">
                    <template v-if="currentResult && !loading">
                      <ReloadOutlined /> {{ t('archive.reAudit') }}
                    </template>
                    <template v-else-if="!loading">
                      <ThunderboltOutlined /> {{ t('archive.startAudit') }}
                    </template>
                  </a-button>
                </div>
              </div>
            </div>

            <!-- Audit in progress -->
            <template v-if="loading">
              <div class="audit-progress">
                <div class="audit-phase" :class="{ 'audit-phase--done': phase1Done, 'audit-phase--active': !phase1Done }">
                  <div class="audit-phase-dot">
                    <LoadingOutlined v-if="!phase1Done" />
                    <CheckCircleOutlined v-else style="color: var(--color-success);" />
                  </div>
                  <div class="audit-phase-info">
                    <div class="audit-phase-title">{{ t('archive.phase1Title') }}</div>
                    <div class="audit-phase-desc">{{ t('archive.phase1Desc') }}</div>
                  </div>
                </div>
                <div class="audit-phase" :class="{ 'audit-phase--active': phase1Done, 'audit-phase--pending': !phase1Done }">
                  <div class="audit-phase-dot">
                    <LoadingOutlined v-if="phase1Done" />
                    <div v-else class="phase-pending-dot" />
                  </div>
                  <div class="audit-phase-info">
                    <div class="audit-phase-title">{{ t('archive.phase2Title') }}</div>
                    <div class="audit-phase-desc">{{ t('archive.phase2Desc') }}</div>
                  </div>
                </div>
              </div>
            </template>

            <!-- Audit result -->
            <template v-if="currentResult && !loading">
              <!-- Compliance banner -->
              <div
                class="compliance-banner"
                :style="{
                  background: complianceConfig[currentResult.overall_compliance]?.bg,
                  borderColor: complianceConfig[currentResult.overall_compliance]?.color,
                }"
              >
                <SafetyCertificateOutlined
                  class="compliance-banner-icon"
                  :style="{ color: complianceConfig[currentResult.overall_compliance]?.color }"
                />
                <div class="compliance-banner-info">
                  <div class="compliance-banner-title" :style="{ color: complianceConfig[currentResult.overall_compliance]?.color }">
                    {{ complianceConfig[currentResult.overall_compliance]?.label }}
                  </div>
                  <div class="compliance-banner-meta">
                    {{ t('archive.overallScore') }} {{ currentResult.overall_score }} {{ t('archive.score') }}
                    · {{ t('archive.durationLabel') }} {{ currentResult.duration_ms }}ms
                    · {{ currentResult.trace_id }}
                  </div>
                </div>
                <div class="compliance-score" :style="{ color: complianceConfig[currentResult.overall_compliance]?.color }">
                  {{ currentResult.overall_score }}
                </div>
              </div>

              <!-- Rule checks -->
              <div class="section-block">
                <h4 class="section-title"><SafetyCertificateOutlined /> {{ t('archive.ruleAudit') }}</h4>
                <div class="audit-checks">
                  <div
                    v-for="ra in currentResult.rule_audit"
                    :key="ra.rule_id"
                    class="audit-check-item"
                    :class="ra.passed ? 'audit-check-item--pass' : 'audit-check-item--fail'"
                  >
                    <div class="audit-check-status">
                      <CheckCircleOutlined v-if="ra.passed" style="color: var(--color-success);" />
                      <CloseCircleOutlined v-else style="color: var(--color-danger);" />
                    </div>
                    <div class="audit-check-content">
                      <div class="audit-check-name">{{ ra.rule_name }}</div>
                      <div class="audit-check-reasoning">{{ ra.reasoning }}</div>
                    </div>
                  </div>
                  <div v-if="!currentResult.rule_audit?.length" class="audit-check-empty">
                    {{ t('archive.noRules') }}
                  </div>
                </div>
              </div>

              <!-- Risk points & suggestions -->
              <div v-if="currentResult.overall_compliance !== 'compliant'" class="risk-suggestions-row">
                <div class="risk-card">
                  <h4 class="section-title"><AlertOutlined style="color: var(--color-danger);" /> {{ t('archive.riskPoints') }}</h4>
                  <div v-if="currentResult.flow_audit.missing_nodes.length > 0" class="risk-list">
                    <div v-for="node in currentResult.flow_audit.missing_nodes" :key="node" class="risk-item">
                      <CloseCircleOutlined style="color: var(--color-danger); flex-shrink: 0;" />
                      <span>{{ t('archive.missingNode') }}: {{ node }}</span>
                    </div>
                  </div>
                  <div class="risk-list">
                    <div v-for="(ra, i) in currentResult.rule_audit.filter(r => !r.passed)" :key="i" class="risk-item">
                      <CloseCircleOutlined style="color: var(--color-danger); flex-shrink: 0;" />
                      <span>{{ ra.rule_name }}</span>
                    </div>
                  </div>
                  <div v-if="!currentResult.flow_audit.missing_nodes.length && !currentResult.rule_audit.filter(r => !r.passed).length" class="risk-empty">
                    {{ t('archive.noRiskPoints') }}
                  </div>
                </div>
                <div class="suggestion-card">
                  <h4 class="section-title"><BulbOutlined style="color: var(--color-warning);" /> {{ t('archive.suggestions') }}</h4>
                  <div class="suggestion-list">
                    <div v-for="(fa, i) in currentResult.field_audit.filter(f => !f.passed)" :key="i" class="suggestion-item">
                      <RightOutlined style="color: var(--color-warning); flex-shrink: 0;" />
                      <span>{{ fa.reasoning }}</span>
                    </div>
                    <div v-if="!currentResult.field_audit.filter(f => !f.passed).length" class="suggestion-item">
                      <RightOutlined style="color: var(--color-warning); flex-shrink: 0;" />
                      <span>{{ t('archive.reviewSuggestion') }}</span>
                    </div>
                  </div>
                </div>
              </div>

              <!-- AI Summary -->
              <div class="section-block">
                <h4 class="section-title"><ThunderboltOutlined /> {{ t('archive.aiSummary') }}</h4>
                <div class="ai-summary">
                  <pre>{{ currentResult.ai_summary }}</pre>
                </div>
              </div>
            </template>

            <!-- No result yet (not loading) -->
            <div v-if="!currentResult && !loading" class="no-result-hint">
              <HistoryOutlined style="font-size: 32px; color: var(--color-text-tertiary);" />
              <p>{{ t('archive.noResultHint') }}</p>
            </div>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.archive-page { animation: fadeIn 0.3s ease-out; }
@keyframes fadeIn { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

.page-header {
  display: flex; justify-content: space-between; align-items: flex-start;
  margin-bottom: 20px; flex-wrap: wrap; gap: 16px;
}
.page-title { font-size: 24px; font-weight: 700; color: var(--color-text-primary); margin: 0; letter-spacing: -0.02em; }
.page-subtitle { font-size: 14px; color: var(--color-text-tertiary); margin: 4px 0 0; }
.page-header-actions { display: flex; gap: 8px; align-items: center; }

/* Stats row */
.stats-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; margin-bottom: 20px; }
.stat-card {
  display: flex; align-items: center; gap: 12px; padding: 14px 18px;
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); cursor: pointer;
  transition: all var(--transition-fast);
}
.stat-card:hover { box-shadow: var(--shadow-md); transform: translateY(-1px); }
.stat-card--selected { border-width: 2px; }
.stat-card--primary.stat-card--selected { border-color: var(--color-primary); }
.stat-card--success.stat-card--selected { border-color: var(--color-success); }
.stat-card--warning.stat-card--selected { border-color: var(--color-warning); }
.stat-card--danger.stat-card--selected { border-color: var(--color-danger); }
.stat-card-icon { font-size: 22px; }
.stat-card--primary .stat-card-icon { color: var(--color-primary); }
.stat-card--success .stat-card-icon { color: var(--color-success); }
.stat-card--warning .stat-card-icon { color: var(--color-warning); }
.stat-card--danger .stat-card-icon { color: var(--color-danger); }
.stat-card-info { display: flex; flex-direction: column; }
.stat-card-value { font-size: 22px; font-weight: 700; color: var(--color-text-primary); line-height: 1.2; }
.stat-card-label { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }

/* Main grid */
.archive-grid { display: grid; grid-template-columns: 400px 1fr; gap: 20px; align-items: start; }

/* Panels */
.list-panel, .detail-panel {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); overflow: hidden;
}
.panel-header {
  padding: 14px 18px; border-bottom: 1px solid var(--color-border-light);
  display: flex; flex-direction: column; gap: 10px;
}
.panel-header-row { display: flex; justify-content: space-between; align-items: center; }
.panel-title {
  font-size: 15px; font-weight: 600; color: var(--color-text-primary);
  margin: 0; display: flex; align-items: center; gap: 8px;
}
.panel-header-hint { font-size: 12px; color: var(--color-text-tertiary); }
.filter-toggle-btn--active { color: var(--color-primary); border-color: var(--color-primary); }
.filter-active-dot {
  display: inline-block; width: 6px; height: 6px; border-radius: 50%;
  background: var(--color-primary); margin-left: 4px; vertical-align: middle;
}

/* Filter bar */
.filter-bar {
  display: flex; gap: 8px; align-items: center; flex-wrap: wrap;
  padding: 10px 0 2px;
}
.slide-enter-active, .slide-leave-active { transition: all 0.2s ease; }
.slide-enter-from, .slide-leave-to { opacity: 0; transform: translateY(-6px); }

/* Batch toolbar */
.batch-toolbar { display: flex; align-items: center; gap: 10px; }

/* Process list */
.process-list { max-height: calc(100vh - 340px); overflow-y: auto; }
.process-item {
  display: flex; align-items: flex-start; padding: 12px 16px; cursor: pointer;
  transition: all var(--transition-fast); border-bottom: 1px solid var(--color-border-light); gap: 10px;
}
.process-item:last-child { border-bottom: none; }
.process-item:hover { background: var(--color-bg-hover); }
.process-item--selected { background: var(--color-primary-bg); border-left: 3px solid var(--color-primary); }
.process-item--compliant { border-left: 3px solid var(--color-success); }
.process-item--partial { border-left: 3px solid var(--color-warning); }
.process-item--noncompliant { border-left: 3px solid var(--color-danger); }
.process-item-checkbox { padding-top: 2px; flex-shrink: 0; }
.process-item-main { flex: 1; min-width: 0; }
.process-item-title-row { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; flex-wrap: wrap; }
.process-item-title {
  font-size: 13px; font-weight: 500; color: var(--color-text-primary);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 180px;
}
.process-audit-badge {
  display: inline-flex; align-items: center; gap: 3px;
  font-size: 11px; font-weight: 600; padding: 1px 7px; border-radius: var(--radius-full); white-space: nowrap; flex-shrink: 0;
}
.process-item-meta {
  font-size: 12px; color: var(--color-text-tertiary);
  display: flex; align-items: center; gap: 4px; flex-wrap: wrap; margin-bottom: 4px;
}
.meta-dot { color: var(--color-border); }
.process-item-footer { display: flex; align-items: center; gap: 6px; }
.process-type-tag {
  font-size: 11px; padding: 1px 8px; border-radius: var(--radius-full);
  background: var(--color-bg-hover); color: var(--color-text-tertiary); border: 1px solid var(--color-border-light);
}
.process-auditing { font-size: 11px; color: var(--color-primary); display: flex; align-items: center; gap: 4px; }
.oa-jump-btn { margin-left: auto; color: var(--color-text-tertiary); padding: 0 4px; height: 22px; }
.oa-jump-btn:hover { color: var(--color-primary); }
.list-empty { padding: 40px 20px; }

/* Pagination */
.pagination-wrapper { padding: 12px 16px; border-top: 1px solid var(--color-border-light); display: flex; justify-content: flex-end; }

/* Detail panel */
.detail-content { padding: 18px; max-height: calc(100vh - 220px); overflow-y: auto; }
.detail-empty { text-align: center; padding: 60px 20px; }
.detail-empty-icon {
  width: 64px; height: 64px; border-radius: 50%; background: var(--color-primary-bg);
  color: var(--color-primary); font-size: 28px; display: flex; align-items: center;
  justify-content: center; margin: 0 auto 16px;
}
.detail-empty h4 { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 8px; }
.detail-empty p { font-size: 13px; color: var(--color-text-tertiary); margin: 0 auto; max-width: 300px; }

/* Process info card */
.process-info-card {
  padding: 14px 16px; background: var(--color-bg-page);
  border-radius: var(--radius-lg); border: 1px solid var(--color-border-light); margin-bottom: 16px;
}
.process-info-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; }
.process-info-title { font-size: 15px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 6px; }
.process-info-meta { font-size: 12px; color: var(--color-text-tertiary); }
.process-info-actions { display: flex; gap: 8px; flex-shrink: 0; }

/* Audit progress */
.audit-progress {
  display: flex; flex-direction: column; gap: 12px; padding: 20px;
  background: var(--color-bg-page); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); margin-bottom: 16px;
}
.audit-phase { display: flex; align-items: flex-start; gap: 12px; }
.audit-phase-dot { font-size: 18px; flex-shrink: 0; padding-top: 2px; }
.audit-phase--active .audit-phase-dot { color: var(--color-primary); animation: spin 1s linear infinite; }
.audit-phase--pending .audit-phase-dot { color: var(--color-text-tertiary); }
.phase-pending-dot { width: 18px; height: 18px; border-radius: 50%; border: 2px solid var(--color-border); }
.audit-phase-title { font-size: 14px; font-weight: 500; color: var(--color-text-primary); }
.audit-phase-desc { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }
@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }

/* Compliance banner */
.compliance-banner {
  display: flex; align-items: center; padding: 14px 18px;
  border-radius: var(--radius-lg); border-left: 4px solid; margin-bottom: 16px; gap: 12px;
}
.compliance-banner-icon { font-size: 26px; flex-shrink: 0; }
.compliance-banner-info { flex: 1; }
.compliance-banner-title { font-size: 15px; font-weight: 700; }
.compliance-banner-meta { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }
.compliance-score { font-size: 34px; font-weight: 800; line-height: 1; }

/* Section blocks */
.section-block { margin-bottom: 16px; }
.section-title {
  font-size: 13px; font-weight: 600; color: var(--color-text-primary);
  margin: 0 0 10px; display: flex; align-items: center; gap: 6px;
}

/* Audit checks */
.audit-checks { display: flex; flex-direction: column; gap: 6px; }
.audit-check-item {
  display: flex; gap: 10px; padding: 10px 14px;
  border-radius: var(--radius-md); border: 1px solid var(--color-border-light);
}
.audit-check-item--pass { border-left: 3px solid var(--color-success); }
.audit-check-item--fail { border-left: 3px solid var(--color-danger); background: var(--color-danger-bg); }
.audit-check-status { font-size: 16px; flex-shrink: 0; padding-top: 1px; }
.audit-check-content { flex: 1; min-width: 0; }
.audit-check-name { font-size: 13px; font-weight: 500; color: var(--color-text-primary); margin-bottom: 3px; }
.audit-check-reasoning { font-size: 12px; color: var(--color-text-secondary); line-height: 1.5; }
.audit-check-empty { font-size: 13px; color: var(--color-text-tertiary); padding: 12px; text-align: center; }

/* Risk & suggestions */
.risk-suggestions-row { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-bottom: 16px; }
.risk-card, .suggestion-card {
  padding: 14px; background: var(--color-bg-page);
  border-radius: var(--radius-lg); border: 1px solid var(--color-border-light);
}
.risk-list, .suggestion-list { display: flex; flex-direction: column; gap: 6px; }
.risk-item, .suggestion-item {
  display: flex; align-items: flex-start; gap: 8px;
  font-size: 12px; color: var(--color-text-secondary); line-height: 1.5;
}
.risk-empty { font-size: 12px; color: var(--color-text-tertiary); }

/* AI Summary */
.ai-summary {
  background: var(--color-bg-page); border-radius: var(--radius-md);
  padding: 14px; border: 1px solid var(--color-border-light);
}
.ai-summary pre {
  white-space: pre-wrap; word-break: break-word; font-family: var(--font-sans);
  font-size: 13px; line-height: 1.7; color: var(--color-text-secondary); margin: 0;
}

.oa-jump-btn {
  width: 24px; height: 24px; border: 1px solid var(--color-border);
  background: transparent; border-radius: var(--radius-sm); cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  font-size: 12px; color: var(--color-text-tertiary);
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  outline: none; flex-shrink: 0;
}
.oa-jump-btn:hover {
  border-color: var(--color-primary); color: var(--color-primary);
  background: var(--color-primary-bg);
  transform: scale(1.1);
  box-shadow: 0 2px 8px rgba(79, 70, 229, 0.15);
}
.oa-jump-btn:focus-visible {
  border-color: var(--color-primary); color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(79, 70, 229, 0.2);
  background: var(--color-primary-bg);
}
.oa-jump-btn:active { transform: scale(0.95); }
.no-result-hint {
  text-align: center; padding: 40px 20px;
  color: var(--color-text-tertiary); display: flex; flex-direction: column; align-items: center; gap: 12px;
}
.no-result-hint p { font-size: 13px; margin: 0; }

/* Responsive */
@media (max-width: 1200px) {
  .archive-grid { grid-template-columns: 360px 1fr; }
}
@media (max-width: 1024px) {
  .archive-grid { grid-template-columns: 1fr; }
  .stats-row { grid-template-columns: repeat(2, 1fr); }
}
@media (max-width: 768px) {
  .stats-row { grid-template-columns: repeat(2, 1fr); }
  .risk-suggestions-row { grid-template-columns: 1fr; }
  .process-info-header { flex-direction: column; }
  .process-info-actions { width: 100%; }
}
@media (max-width: 480px) {
  .stats-row { grid-template-columns: 1fr 1fr; }
  .page-title { font-size: 20px; }
}
</style>
