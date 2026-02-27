<script setup lang="ts">
import {
  SearchOutlined,
  FilterOutlined,
  DownloadOutlined,
  SafetyCertificateOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ExclamationCircleOutlined,
  ExportOutlined,
  AuditOutlined,
  NodeIndexOutlined,
  FileProtectOutlined,
  FieldTimeOutlined,
  ThunderboltOutlined,
  RightOutlined,
  ReloadOutlined,
  UnorderedListOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type { ArchivedProcess, ArchiveAuditResult } from '~/composables/useMockData'
import { useI18n } from '~/composables/useI18n'

definePageMeta({ middleware: 'auth' })

const { t } = useI18n()

const { mockArchivedProcesses, mockArchiveAuditResult } = useMockData()

const processList = ref<ArchivedProcess[]>([...mockArchivedProcesses])
const selectedProcess = ref<ArchivedProcess | null>(null)
const auditResult = ref<ArchiveAuditResult | null>(null)
const loading = ref(false)
const searchText = ref('')
const showFilters = ref(false)
const showAuditModal = ref(false)
const batchLoading = ref(false)
const filters = ref({
  department: undefined as string | undefined,
  processType: undefined as string | undefined,
})

// Track audit records per process_id
const auditRecords = ref<Record<string, ArchiveAuditResult>>({})

const departments = [...new Set(mockArchivedProcesses.map(p => p.department))]
const processTypes = [...new Set(mockArchivedProcesses.map(p => p.process_type))]

const filteredList = computed(() => {
  let list = [...processList.value]
  if (searchText.value) {
    const q = searchText.value.toLowerCase()
    list = list.filter(p =>
      p.title.toLowerCase().includes(q) ||
      p.applicant.toLowerCase().includes(q) ||
      p.process_id.toLowerCase().includes(q)
    )
  }
  if (filters.value.department) {
    list = list.filter(p => p.department === filters.value.department)
  }
  if (filters.value.processType) {
    list = list.filter(p => p.process_type === filters.value.processType)
  }
  return list
})

// Pagination for archive process list
const { paged: pagedArchiveList, current: archivePage, pageSize: archivePageSize, total: archiveTotal, onChange: onArchivePageChange } = usePagination(filteredList, 10)

// Count how many in current filtered list have been audited
const auditedCount = computed(() => filteredList.value.filter(p => auditRecords.value[p.process_id]).length)

const clearFilters = () => {
  filters.value = { department: undefined, processType: undefined }
  searchText.value = ''
}

const selectProcess = (proc: ArchivedProcess) => {
  selectedProcess.value = proc
  // Load existing audit record if available
  auditResult.value = auditRecords.value[proc.process_id] || null
}

// Generate a mock audit result for a given process
const generateAuditResult = (proc: ArchivedProcess): ArchiveAuditResult => {
  const complianceOptions: ArchiveAuditResult['overall_compliance'][] = ['compliant', 'non_compliant', 'partially_compliant']
  // Deterministic-ish based on process_id hash
  const hash = proc.process_id.split('').reduce((a, c) => a + c.charCodeAt(0), 0)
  const compliance = complianceOptions[hash % 3]
  const score = compliance === 'compliant' ? 85 + (hash % 15) : compliance === 'partially_compliant' ? 55 + (hash % 25) : 20 + (hash % 30)

  return {
    ...mockArchiveAuditResult,
    trace_id: `ATR-${Date.now().toString(36).toUpperCase()}`,
    process_id: proc.process_id,
    overall_compliance: compliance,
    overall_score: score,
    duration_ms: 1500 + (hash % 3000),
    flow_audit: {
      ...mockArchiveAuditResult.flow_audit,
      node_results: proc.flow_nodes.map(n => ({
        node_id: n.node_id,
        node_name: n.node_name,
        compliant: hash % 5 !== 0 || n.action === 'approve',
        reasoning: n.action === 'approve' ? t('archive.auditReason.approve') : t('archive.auditReason.attention', [n.node_name, n.action]),
      })),
    },
  }
}

const startComplianceAudit = async () => {
  if (!selectedProcess.value) return
  showAuditModal.value = true
  loading.value = true
  auditResult.value = null
  await new Promise(resolve => setTimeout(resolve, 2200))
  const result = generateAuditResult(selectedProcess.value)
  auditResult.value = result
  auditRecords.value[selectedProcess.value.process_id] = result
  loading.value = false
}

const batchComplianceAudit = async () => {
  const unaudited = filteredList.value.filter(p => !auditRecords.value[p.process_id])
  if (unaudited.length === 0) {
    message.info(t('archive.allReviewed'))
    return
  }
  batchLoading.value = true
  message.loading({ content: t('archive.batchProgress', `${unaudited.length}`), key: 'batch', duration: 0 })
  for (let i = 0; i < unaudited.length; i++) {
    await new Promise(resolve => setTimeout(resolve, 800))
    const result = generateAuditResult(unaudited[i])
    auditRecords.value[unaudited[i].process_id] = result
    // If this process is currently selected, update the view
    if (selectedProcess.value?.process_id === unaudited[i].process_id) {
      auditResult.value = result
    }
  }
  message.success({ content: t('archive.batchDone', `${unaudited.length}`), key: 'batch' })
  batchLoading.value = false
}

const handleExport = (format: string) => {
  message.success(t('archive.exporting', format.toUpperCase()))
}

const jumpToOA = (processId: string) => {
  message.info(t('archive.jumpingToOA', processId))
}

const complianceConfig = computed(() => ({
  compliant: { color: 'var(--color-success)', bg: 'var(--color-success-bg)', label: t('archive.compliant') },
  non_compliant: { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)', label: t('archive.nonCompliant') },
  partially_compliant: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', label: t('archive.partiallyCompliant') },
}))

const actionConfig = computed(() => ({
  approve: { color: 'var(--color-success)', label: t('archive.actionApprove') },
  return: { color: 'var(--color-warning)', label: t('archive.actionReturn') },
}))
</script>

<template>
  <div class="archive-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('archive.title') }}</h1>
        <p class="page-subtitle">{{ t('archive.subtitle') }}</p>
      </div>
      <div class="page-header-actions">
        <a-button @click="showFilters = !showFilters">
          <FilterOutlined /> {{ t('archive.filter') }}
        </a-button>
        <a-button :loading="batchLoading" @click="batchComplianceAudit">
          <UnorderedListOutlined /> {{ t('archive.batchAudit') }}
          <a-badge
            v-if="filteredList.length - auditedCount > 0"
            :count="filteredList.length - auditedCount"
            :number-style="{ backgroundColor: 'var(--color-warning)', fontSize: '10px', marginLeft: '4px' }"
            :offset="[4, -2]"
          />
        </a-button>
        <a-dropdown v-if="auditResult">
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

    <!-- Filters -->
    <transition name="slide">
      <div v-if="showFilters" class="filter-bar">
        <a-input
          v-model:value="searchText"
          :placeholder="t('archive.searchPlaceholder')"
          allow-clear
          style="width: 260px;"
        >
          <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
        </a-input>
        <a-select v-model:value="filters.department" :placeholder="t('archive.department')" allow-clear style="width: 140px;">
          <a-select-option v-for="d in departments" :key="d" :value="d">{{ d }}</a-select-option>
        </a-select>
        <a-select v-model:value="filters.processType" :placeholder="t('archive.processType')" allow-clear style="width: 140px;">
          <a-select-option v-for="t in processTypes" :key="t" :value="t">{{ t }}</a-select-option>
        </a-select>
        <a-button @click="clearFilters">{{ t('archive.reset') }}</a-button>
      </div>
    </transition>

    <!-- Main layout: left list + right detail -->
    <div class="archive-grid">
      <!-- Left: Archived process list -->
      <div class="list-panel">
        <div class="panel-header">
          <h3 class="panel-title">
            <FileProtectOutlined style="color: var(--color-primary);" />
            {{ t('archive.archivedProcesses') }}
            <a-badge :count="filteredList.length" :number-style="{ backgroundColor: 'var(--color-primary)' }" />
          </h3>
          <span v-if="auditedCount > 0" class="panel-header-hint">
            {{ t('archive.reviewed') }} {{ auditedCount }}/{{ filteredList.length }}
          </span>
        </div>
        <div class="process-list">
          <div
            v-for="proc in pagedArchiveList"
            :key="proc.process_id"
            class="process-item"
            :class="{ 'process-item--selected': selectedProcess?.process_id === proc.process_id }"
            @click="selectProcess(proc)"
          >
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
                <span>{{ proc.process_type }}</span>
              </div>
              <div class="process-item-meta">
                <FieldTimeOutlined />
                <span>{{ t('archive.archivedAt') }} {{ proc.archive_time }}</span>
              </div>
            </div>
            <div class="process-item-right">
              <span v-if="proc.amount" class="process-item-amount">¥{{ proc.amount.toLocaleString() }}</span>
              <span class="process-item-nodes">{{ proc.flow_nodes.length }} {{ t('archive.nodes') }}</span>
            </div>
          </div>
          <div v-if="filteredList.length === 0" class="list-empty">
            <a-empty :description="t('archive.noMatch')" />
          </div>
        </div>

        <!-- Pagination -->
        <div class="pagination-wrapper">
          <a-pagination
            :current="archivePage"
            :page-size="archivePageSize"
            :total="archiveTotal"
            size="small"
            show-size-changer
            show-quick-jumper
            :page-size-options="['10', '20', '50']"
            @change="onArchivePageChange"
            @showSizeChange="onArchivePageChange"
          />
        </div>
      </div>

      <!-- Right: Detail & audit result -->
      <div class="detail-panel">
        <div class="panel-header">
          <h3 class="panel-title">
            <AuditOutlined style="color: var(--color-primary);" />
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

          <!-- Process selected: show detail -->
          <template v-else>
            <!-- Process info header -->
            <div class="process-info-card">
              <div class="process-info-header">
                <div>
                  <h4 class="process-info-title">{{ selectedProcess.title }}</h4>
                  <div class="process-info-meta">
                    {{ selectedProcess.applicant }} · {{ selectedProcess.department }} · {{ selectedProcess.process_type }}
                    <span v-if="selectedProcess.amount"> · ¥{{ selectedProcess.amount.toLocaleString() }}</span>
                  </div>
                  <div class="process-info-meta" style="margin-top: 4px;">
                    {{ t('archive.submitLabel') }}: {{ selectedProcess.submit_time }} → {{ t('archive.archiveLabel') }}: {{ selectedProcess.archive_time }}
                  </div>
                </div>
                <div class="process-info-actions">
                  <a-button @click="jumpToOA(selectedProcess.process_id)">
                    <ExportOutlined /> OA
                  </a-button>
                  <a-button
                    type="primary"
                    :loading="loading"
                    @click="startComplianceAudit"
                  >
                    <template v-if="auditRecords[selectedProcess.process_id]">
                      <ReloadOutlined /> {{ t('archive.reAudit') }}
                    </template>
                    <template v-else>
                      <ThunderboltOutlined /> {{ t('archive.startAudit') }}
                    </template>
                  </a-button>
                </div>
              </div>
            </div>

            <!-- Inline audit result hint (clickable to open modal) -->
            <div v-if="auditResult && !loading" class="audit-result-hint" @click="showAuditModal = true">
              <div
                class="audit-result-hint-banner"
                :style="{
                  background: complianceConfig[auditResult.overall_compliance]?.bg,
                  borderColor: complianceConfig[auditResult.overall_compliance]?.color,
                }"
              >
                <SafetyCertificateOutlined :style="{ color: complianceConfig[auditResult.overall_compliance]?.color, fontSize: '20px' }" />
                <div style="flex: 1;">
                  <div :style="{ color: complianceConfig[auditResult.overall_compliance]?.color, fontWeight: 700, fontSize: '14px' }">
                    {{ complianceConfig[auditResult.overall_compliance]?.label }} · {{ auditResult.overall_score }} 分
                  </div>
                  <div style="font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px;">
                    {{ t('archive.viewReport') }}
                  </div>
                </div>
                <RightOutlined style="color: var(--color-text-tertiary);" />
              </div>
            </div>

            <!-- AI Summary (moved above flow timeline) -->
            <div v-if="auditResult && !loading" class="section-block">
              <h4 class="section-title"><ThunderboltOutlined /> {{ t('archive.aiSummary') }}</h4>
              <div class="ai-summary">
                <pre>{{ auditResult.ai_summary }}</pre>
              </div>
            </div>

            <!-- Flow timeline -->
            <div class="section-block">
              <h4 class="section-title"><NodeIndexOutlined /> {{ t('archive.flowChain') }}（{{ selectedProcess.flow_nodes.length }} {{ t('archive.nodes') }}）</h4>
              <div class="flow-timeline">
                <div
                  v-for="(node, idx) in selectedProcess.flow_nodes"
                  :key="node.node_id"
                  class="flow-node"
                >
                  <div class="flow-node-timeline">
                    <div class="flow-node-dot" :style="{ background: actionConfig[node.action]?.color }" />
                    <div v-if="idx < selectedProcess.flow_nodes.length - 1" class="flow-node-line" />
                  </div>
                  <div class="flow-node-card">
                    <div class="flow-node-header">
                      <span class="flow-node-name">{{ node.node_name }}</span>
                      <span
                        class="flow-action-tag"
                        :style="{ color: actionConfig[node.action]?.color }"
                      >
                        {{ actionConfig[node.action]?.label }}
                      </span>
                    </div>
                    <div class="flow-node-meta">{{ node.approver }} · {{ node.action_time }}</div>
                    <div class="flow-node-opinion">{{ node.opinion }}</div>
                  </div>
                </div>
              </div>
            </div>

            <!-- Fields -->
            <div class="section-block">
              <h4 class="section-title"><FileProtectOutlined /> {{ t('archive.keyFields') }}</h4>
              <div class="fields-grid">
                <div v-for="(val, key) in selectedProcess.fields" :key="key" class="field-item">
                  <span class="field-label">{{ key }}</span>
                  <span class="field-value">{{ val }}</span>
                </div>
              </div>
            </div>
          </template>
        </div>
      </div>
    </div>

    <!-- Compliance Audit Modal -->
    <a-modal
      v-model:open="showAuditModal"
      :title="selectedProcess ? `${t('archive.complianceTitle')} - ${selectedProcess.title}` : t('archive.complianceTitle')"
      :width="720"
      :footer="null"
      :bodyStyle="{ maxHeight: '70vh', overflowY: 'auto', padding: '24px' }"
      centered
    >
      <!-- Loading state -->
      <div v-if="loading" class="audit-loading">
        <div class="loading-animation">
          <div class="loading-pulse" />
          <div class="loading-text">{{ t('archive.aiAuditing') }}</div>
          <div class="loading-subtext">{{ t('archive.auditChecks') }}</div>
        </div>
      </div>

      <!-- Audit result -->
      <template v-if="auditResult && !loading">
        <!-- Overall compliance banner -->
        <div
          class="compliance-banner"
          :style="{
            background: complianceConfig[auditResult.overall_compliance]?.bg,
            borderColor: complianceConfig[auditResult.overall_compliance]?.color,
          }"
        >
          <SafetyCertificateOutlined
            class="compliance-banner-icon"
            :style="{ color: complianceConfig[auditResult.overall_compliance]?.color }"
          />
          <div class="compliance-banner-info">
            <div
              class="compliance-banner-title"
              :style="{ color: complianceConfig[auditResult.overall_compliance]?.color }"
            >
              {{ complianceConfig[auditResult.overall_compliance]?.label }}
            </div>
            <div class="compliance-banner-meta">
              {{ t('archive.overallScore') }} {{ auditResult.overall_score }} {{ t('archive.score') }} · {{ t('archive.durationLabel') }} {{ auditResult.duration_ms }}ms · {{ auditResult.trace_id }}
            </div>
          </div>
          <div class="compliance-score" :style="{ color: complianceConfig[auditResult.overall_compliance]?.color }">
            {{ auditResult.overall_score }}
          </div>
        </div>

        <!-- AI Summary (first in modal) -->
        <div class="section-block">
          <h4 class="section-title"><ThunderboltOutlined /> {{ t('archive.aiSummary') }}</h4>
          <div class="ai-summary">
            <pre>{{ auditResult.ai_summary }}</pre>
          </div>
        </div>

        <!-- Flow audit section -->
        <div class="section-block">
          <h4 class="section-title">
            <NodeIndexOutlined /> {{ t('archive.flowCompliance') }}
            <span class="section-badge" :class="auditResult.flow_audit.is_complete ? 'section-badge--pass' : 'section-badge--fail'">
              {{ auditResult.flow_audit.is_complete ? t('archive.flowComplete') : t('archive.flowMissing') }}
            </span>
          </h4>
          <div v-if="auditResult.flow_audit.missing_nodes.length > 0" class="missing-nodes-alert">
            <ExclamationCircleOutlined /> {{ t('archive.missingNodes') }}: {{ auditResult.flow_audit.missing_nodes.join('、') }}
          </div>
          <div class="audit-checks">
            <div
              v-for="nr in auditResult.flow_audit.node_results"
              :key="nr.node_id"
              class="audit-check-item"
              :class="nr.compliant ? 'audit-check-item--pass' : 'audit-check-item--fail'"
            >
              <div class="audit-check-status">
                <CheckCircleOutlined v-if="nr.compliant" style="color: var(--color-success);" />
                <CloseCircleOutlined v-else style="color: var(--color-danger);" />
              </div>
              <div class="audit-check-content">
                <div class="audit-check-name">{{ nr.node_name }}</div>
                <div class="audit-check-reasoning">{{ nr.reasoning }}</div>
              </div>
            </div>
          </div>
        </div>

        <!-- Field audit section -->
        <div class="section-block">
          <h4 class="section-title"><FileProtectOutlined /> {{ t('archive.fieldAudit') }}</h4>
          <div class="audit-checks">
            <div
              v-for="fa in auditResult.field_audit"
              :key="fa.field_name"
              class="audit-check-item"
              :class="fa.passed ? 'audit-check-item--pass' : 'audit-check-item--fail'"
            >
              <div class="audit-check-status">
                <CheckCircleOutlined v-if="fa.passed" style="color: var(--color-success);" />
                <CloseCircleOutlined v-else style="color: var(--color-danger);" />
              </div>
              <div class="audit-check-content">
                <div class="audit-check-name">{{ fa.field_name }}</div>
                <div class="audit-check-reasoning">{{ fa.reasoning }}</div>
              </div>
            </div>
          </div>
        </div>

        <!-- Rule audit section -->
        <div class="section-block">
          <h4 class="section-title"><SafetyCertificateOutlined /> {{ t('archive.ruleAudit') }}</h4>
          <div class="audit-checks">
            <div
              v-for="ra in auditResult.rule_audit"
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
          </div>
        </div>

        <!-- Export actions in modal -->
        <div class="modal-export-actions">
          <a-dropdown>
            <a-button type="primary">
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
      </template>
    </a-modal>
  </div>
</template>

<style scoped>
.archive-page { animation: fadeIn 0.3s ease-out; }
@keyframes fadeIn { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
  flex-wrap: wrap; /* Allow wrapping */
  gap: 16px; /* Space when wrapped */
}

/* ... existing title styles ... */
.page-title { font-size: 24px; font-weight: 700; color: var(--color-text-primary); margin: 0; letter-spacing: -0.02em; }
.page-subtitle { font-size: 14px; color: var(--color-text-tertiary); margin: 4px 0 0; }
.page-header-actions { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; } /* Allow buttons to wrap */

/* Filters */
.filter-bar {
  display: flex; gap: 12px; align-items: center; padding: 16px 20px;
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); margin-bottom: 20px; flex-wrap: wrap;
}
.slide-enter-active, .slide-leave-active { transition: all 0.2s ease; }
.slide-enter-from, .slide-leave-to { opacity: 0; transform: translateY(-8px); }

/* Main grid */
.archive-grid {
  display: grid;
  grid-template-columns: 400px 1fr;
  gap: 24px;
  align-items: start;
}

/* Panels */
.list-panel, .detail-panel {
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light);
  overflow: hidden;
}

.panel-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--color-border-light);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.panel-title {
  font-size: 15px; font-weight: 600; color: var(--color-text-primary);
  margin: 0; display: flex; align-items: center; gap: 8px;
}

.panel-header-hint {
  font-size: 12px; color: var(--color-text-tertiary);
}

/* Process list */
.process-list { max-height: calc(100vh - 280px); overflow-y: auto; }

.process-item {
  display: flex; align-items: flex-start; justify-content: space-between;
  padding: 14px 20px; cursor: pointer; transition: all var(--transition-fast);
  border-bottom: 1px solid var(--color-border-light); gap: 12px;
}
.process-item:last-child { border-bottom: none; }
.process-item:hover { background: var(--color-bg-hover); }
.process-item--selected { background: var(--color-primary-bg); border-left: 3px solid var(--color-primary); }

.process-item-main { flex: 1; min-width: 0; }
.process-item-title-row {
  display: flex; align-items: center; gap: 8px; margin-bottom: 4px; flex-wrap: wrap;
}
.process-item-title {
  font-size: 14px; font-weight: 500; color: var(--color-text-primary);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.process-audit-badge {
  display: inline-flex; align-items: center; gap: 4px;
  font-size: 11px; font-weight: 600; padding: 1px 8px; border-radius: var(--radius-full);
  white-space: nowrap; flex-shrink: 0;
}
.process-item-meta {
  font-size: 12px; color: var(--color-text-tertiary);
  display: flex; align-items: center; gap: 4px; flex-wrap: wrap;
}
.meta-dot { color: var(--color-border); }

.process-item-right { display: flex; flex-direction: column; align-items: flex-end; gap: 4px; flex-shrink: 0; }
.process-item-amount { font-size: 13px; font-weight: 600; color: var(--color-text-primary); }
.process-item-nodes { font-size: 11px; color: var(--color-text-tertiary); padding: 2px 8px; background: var(--color-bg-hover); border-radius: var(--radius-full); }

.list-empty { padding: 48px 20px; }

/* Detail panel */
.detail-content { padding: 20px; }

.detail-empty { text-align: center; padding: 60px 20px; }
.detail-empty-icon {
  width: 64px; height: 64px; border-radius: 50%; background: var(--color-primary-bg);
  color: var(--color-primary); font-size: 28px; display: flex; align-items: center;
  justify-content: center; margin: 0 auto 16px;
}
.detail-empty h4 { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 8px; }
.detail-empty p { font-size: 13px; color: var(--color-text-tertiary); margin: 0 auto; max-width: 320px; }

/* Process info card */
.process-info-card {
  padding: 16px 20px; background: var(--color-bg-page);
  border-radius: var(--radius-lg); border: 1px solid var(--color-border-light); margin-bottom: 20px;
}
.process-info-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 16px; }
.process-info-title { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 6px; }
.process-info-meta { font-size: 13px; color: var(--color-text-tertiary); }
.process-info-actions { display: flex; gap: 8px; flex-shrink: 0; }

/* Section blocks */
.section-block { margin-bottom: 20px; }
.section-title {
  font-size: 14px; font-weight: 600; color: var(--color-text-primary);
  margin: 0 0 12px; display: flex; align-items: center; gap: 8px;
}
.section-badge {
  font-size: 11px; font-weight: 600; padding: 2px 10px; border-radius: var(--radius-full);
}
.section-badge--pass { background: var(--color-success-bg); color: var(--color-success); }
.section-badge--fail { background: var(--color-danger-bg); color: var(--color-danger); }

/* Flow timeline */
.flow-timeline { display: flex; flex-direction: column; }
.flow-node { display: flex; gap: 14px; }
.flow-node-timeline { display: flex; flex-direction: column; align-items: center; width: 18px; flex-shrink: 0; }
.flow-node-dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; margin-top: 6px; }
.flow-node-line { width: 2px; flex: 1; background: var(--color-border-light); min-height: 16px; }
.flow-node-card {
  flex: 1; padding: 10px 14px; border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md); margin-bottom: 8px; transition: background var(--transition-fast);
}
.flow-node-card:hover { background: var(--color-bg-hover); }
.flow-node-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 4px; }
.flow-node-name { font-size: 13px; font-weight: 500; color: var(--color-text-primary); }
.flow-action-tag { font-size: 12px; font-weight: 600; }
.flow-node-meta { font-size: 12px; color: var(--color-text-tertiary); margin-bottom: 4px; }
.flow-node-opinion { font-size: 13px; color: var(--color-text-secondary); line-height: 1.5; }

/* Fields grid */
.fields-grid {
  display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 8px;
}
.field-item {
  display: flex; flex-direction: column; gap: 2px; padding: 10px 14px;
  background: var(--color-bg-page); border-radius: var(--radius-md);
  border: 1px solid var(--color-border-light);
}
.field-label { font-size: 11px; font-weight: 600; color: var(--color-text-tertiary); text-transform: uppercase; letter-spacing: 0.04em; }
.field-value { font-size: 13px; color: var(--color-text-primary); word-break: break-all; }

/* Loading */
.audit-loading { display: flex; justify-content: center; padding: 40px 0; }
.loading-animation { text-align: center; }
.loading-pulse {
  width: 48px; height: 48px; border-radius: 50%; background: var(--color-primary);
  margin: 0 auto 16px; animation: pulse 1.5s ease-in-out infinite;
}
@keyframes pulse { 0%, 100% { transform: scale(1); opacity: 0.6; } 50% { transform: scale(1.15); opacity: 1; } }
.loading-text { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 4px; }
.loading-subtext { font-size: 13px; color: var(--color-text-tertiary); }

/* Compliance banner */
.compliance-banner {
  display: flex; align-items: center; padding: 16px 20px;
  border-radius: var(--radius-lg); border-left: 4px solid; margin-bottom: 24px; gap: 14px;
}
.compliance-banner-icon { font-size: 28px; flex-shrink: 0; }
.compliance-banner-info { flex: 1; }
.compliance-banner-title { font-size: 16px; font-weight: 700; }
.compliance-banner-meta { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }
.compliance-score { font-size: 36px; font-weight: 800; line-height: 1; }

/* Missing nodes alert */
.missing-nodes-alert {
  display: flex; align-items: center; gap: 8px; padding: 10px 14px;
  background: var(--color-danger-bg); color: var(--color-danger);
  border-radius: var(--radius-md); font-size: 13px; font-weight: 500; margin-bottom: 12px;
}

/* Audit checks */
.audit-checks { display: flex; flex-direction: column; gap: 8px; }
.audit-check-item {
  display: flex; gap: 12px; padding: 12px 16px;
  border-radius: var(--radius-md); border: 1px solid var(--color-border-light);
  transition: background var(--transition-fast);
}
.audit-check-item:hover { background: var(--color-bg-hover); }
.audit-check-item--pass { border-left: 3px solid var(--color-success); }
.audit-check-item--fail { border-left: 3px solid var(--color-danger); background: var(--color-danger-bg); }
.audit-check-status { font-size: 18px; flex-shrink: 0; padding-top: 1px; }
.audit-check-content { flex: 1; min-width: 0; }
.audit-check-name { font-size: 14px; font-weight: 500; color: var(--color-text-primary); margin-bottom: 4px; }
.audit-check-reasoning { font-size: 13px; color: var(--color-text-secondary); line-height: 1.5; }

/* AI Summary */
.ai-summary {
  background: var(--color-bg-page); border-radius: var(--radius-md);
  padding: 16px; border: 1px solid var(--color-border-light);
}
.ai-summary pre {
  white-space: pre-wrap; word-break: break-word; font-family: var(--font-sans);
  font-size: 13px; line-height: 1.7; color: var(--color-text-secondary); margin: 0;
}

/* Audit result hint */
.audit-result-hint { margin-bottom: 20px; cursor: pointer; }
.audit-result-hint-banner {
  display: flex; align-items: center; padding: 14px 18px;
  border-radius: var(--radius-lg); border-left: 4px solid; gap: 12px;
  transition: all var(--transition-fast);
}
.audit-result-hint-banner:hover { box-shadow: var(--shadow-md); transform: translateY(-1px); }

/* Modal export actions */
.modal-export-actions { display: flex; justify-content: flex-end; margin-top: 20px; padding-top: 16px; border-top: 1px solid var(--color-border-light); }

/* Responsive */
@media (max-width: 1024px) {
  .archive-grid { grid-template-columns: 1fr; }
  .process-list { max-height: none; }
}
@media (max-width: 768px) {
  .page-header { flex-direction: column; gap: 12px; align-items: stretch; }
  .page-header-actions { flex-wrap: wrap; }
  .filter-bar { flex-direction: column; align-items: stretch; }
  .filter-bar .ant-input,
  .filter-bar .ant-select { width: 100% !important; }
  .process-info-header { flex-direction: column; gap: 12px; }
  .process-info-actions { width: 100%; }
  .process-info-actions .ant-btn { flex: 1; }
  .fields-grid { grid-template-columns: 1fr; }
  .flow-node-card { padding: 8px 10px; }
  .compliance-banner { flex-wrap: wrap; padding: 12px 14px; }
  .compliance-score { font-size: 28px; }
  .detail-content { padding: 14px; }
  .panel-header { padding: 12px 14px; }
}
@media (max-width: 480px) {
  .page-title { font-size: 20px; }
  .page-header-actions { gap: 6px; }
  .page-header-actions .ant-btn { font-size: 12px; padding: 4px 10px; }
  .process-item { padding: 10px 14px; }
  .process-item-right { display: none; }
  .process-audit-badge { font-size: 10px; padding: 1px 6px; }
}
</style>
