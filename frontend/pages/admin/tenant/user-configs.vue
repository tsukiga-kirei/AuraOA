<script setup lang="ts">
import {
  SearchOutlined,
  UserOutlined,
  EyeOutlined,
  ExportOutlined,
  NodeIndexOutlined,
  AppstoreOutlined,
  ClockCircleOutlined,
  FolderOpenOutlined,
  MailOutlined,
  FileTextOutlined,
  ControlOutlined,
  ApartmentOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  SwapOutlined,
  EditOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type { UserPersonalConfig } from '~/composables/useMockData'

definePageMeta({ middleware: 'auth', layout: 'default' })

const { mockUserPersonalConfigs } = useMockData()

const configs = ref<UserPersonalConfig[]>(JSON.parse(JSON.stringify(mockUserPersonalConfigs)))
const search = ref('')
const deptFilter = ref<string | undefined>(undefined)
const hasConfigFilter = ref<string | undefined>(undefined)

const departments = computed(() => {
  const depts = new Set(configs.value.map(c => c.department))
  return Array.from(depts).sort()
})

const filteredConfigs = computed(() => {
  return configs.value.filter(c => {
    if (search.value && !c.display_name.includes(search.value) && !c.username.includes(search.value)) return false
    if (deptFilter.value && c.department !== deptFilter.value) return false
    if (hasConfigFilter.value === 'configured' && c.total_config_items === 0) return false
    if (hasConfigFilter.value === 'none' && c.total_config_items > 0) return false
    return true
  })
})

const { paged, current, pageSize, total, onChange } = usePagination(filteredConfigs, 10)

// Stats
const totalUsers = computed(() => configs.value.length)
const configuredUsers = computed(() => configs.value.filter(c => c.total_config_items > 0).length)
const totalCustomRules = computed(() => configs.value.reduce((s, c) => s + c.custom_rules_count + c.archive_custom_rules_count, 0))
const totalFieldOverrides = computed(() => configs.value.reduce((s, c) => s + c.field_overrides_count, 0))

// Detail drawer
const showDetail = ref(false)
const detailConfig = ref<UserPersonalConfig | null>(null)
const detailTab = ref<'audit' | 'cron' | 'archive'>('audit')

const openDetail = (c: UserPersonalConfig) => {
  detailConfig.value = c
  // Auto-select first tab that has content
  if (c.audit_details.length > 0) detailTab.value = 'audit'
  else if (c.cron_details.length > 0) detailTab.value = 'cron'
  else if (c.archive_details.length > 0) detailTab.value = 'archive'
  else detailTab.value = 'audit'
  showDetail.value = true
}

const strictnessLabels: Record<string, { label: string; color: string }> = {
  strict: { label: '严格', color: 'var(--color-danger)' },
  standard: { label: '标准', color: 'var(--color-primary)' },
  loose: { label: '宽松', color: 'var(--color-warning)' },
}

const handleExport = () => {
  message.success('用户偏好数据导出中...')
}
</script>

<template>
  <div class="data-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">用户偏好</h1>
        <p class="page-subtitle">查看租户内用户在「个人设置」中的自定义配置情况，用于偏好分析与配置审计</p>
      </div>
    </div>

    <div class="toolbar">
      <div class="toolbar-left">
        <a-input v-model:value="search" placeholder="搜索用户名/姓名" allow-clear style="width: 200px;">
          <template #prefix><SearchOutlined /></template>
        </a-input>
        <a-select v-model:value="deptFilter" placeholder="部门" allow-clear style="width: 140px;">
          <a-select-option v-for="d in departments" :key="d" :value="d">{{ d }}</a-select-option>
        </a-select>
        <a-select v-model:value="hasConfigFilter" placeholder="配置状态" allow-clear style="width: 140px;">
          <a-select-option value="configured">有自定义配置</a-select-option>
          <a-select-option value="none">无自定义配置</a-select-option>
        </a-select>
      </div>
      <a-button @click="handleExport"><ExportOutlined /> 导出</a-button>
    </div>

    <div class="stats-row">
      <div class="stat-card">
        <div class="stat-value">{{ totalUsers }}</div>
        <div class="stat-label">用户总数</div>
      </div>
      <div class="stat-card">
        <div class="stat-value">{{ configuredUsers }}</div>
        <div class="stat-label">有自定义配置</div>
      </div>
      <div class="stat-card">
        <div class="stat-value">{{ totalCustomRules }}</div>
        <div class="stat-label">自定义规则总数</div>
      </div>
      <div class="stat-card">
        <div class="stat-value">{{ totalFieldOverrides }}</div>
        <div class="stat-label">字段覆盖总数</div>
      </div>
    </div>

    <div class="data-table-card">
      <table class="data-table">
        <thead>
          <tr>
            <th>用户</th>
            <th>部门</th>
            <th>自定义规则</th>
            <th>字段覆盖</th>
            <th>尺度调整</th>
            <th>推送邮箱</th>
            <th>配置项</th>
            <th>最后修改</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="c in paged" :key="c.id">
            <td>
              <div class="user-cell">
                <a-avatar :size="28" class="user-avatar">
                  <template #icon><UserOutlined /></template>
                </a-avatar>
                <div>
                  <div class="user-name">{{ c.display_name }}</div>
                  <div class="user-username">{{ c.username }}</div>
                </div>
              </div>
            </td>
            <td class="text-secondary">{{ c.department }}</td>
            <td>
              <span v-if="c.custom_rules_count + c.archive_custom_rules_count > 0" class="count-badge count-badge--primary">
                {{ c.custom_rules_count + c.archive_custom_rules_count }}
              </span>
              <span v-else class="text-secondary">-</span>
            </td>
            <td>
              <span v-if="c.field_overrides_count > 0" class="count-badge count-badge--info">{{ c.field_overrides_count }}</span>
              <span v-else class="text-secondary">-</span>
            </td>
            <td>
              <span v-if="c.strictness_overrides_count > 0" class="count-badge count-badge--warning">{{ c.strictness_overrides_count }}</span>
              <span v-else class="text-secondary">-</span>
            </td>
            <td>
              <span v-if="c.custom_push_email" class="text-mono" style="font-size: 12px;">{{ c.custom_push_email }}</span>
              <span v-else class="text-secondary">默认</span>
            </td>
            <td>
              <span v-if="c.total_config_items > 0" class="config-total">{{ c.total_config_items }} 项</span>
              <span v-else class="text-secondary">无</span>
            </td>
            <td class="text-secondary">{{ c.last_modified || '-' }}</td>
            <td>
              <div class="action-btns">
                <button class="icon-btn" title="查看详情" @click="openDetail(c)"><EyeOutlined /></button>
              </div>
            </td>
          </tr>
          <tr v-if="filteredConfigs.length === 0">
            <td colspan="9" class="empty-cell">暂无数据</td>
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

    <!-- Detail Drawer -->
    <a-drawer
      v-model:open="showDetail"
      :title="detailConfig ? detailConfig.display_name + ' 的偏好配置详情' : ''"
      width="600"
      placement="right"
    >
      <template v-if="detailConfig">
        <!-- User header -->
        <div class="detail-user-header">
          <a-avatar :size="40" class="user-avatar">
            <template #icon><UserOutlined /></template>
          </a-avatar>
          <div>
            <div class="detail-user-name">{{ detailConfig.display_name }}</div>
            <div class="detail-user-meta">{{ detailConfig.username }} · {{ detailConfig.department }}</div>
          </div>
          <div class="detail-user-stats">
            <span class="config-total">{{ detailConfig.total_config_items }} 项配置</span>
          </div>
        </div>

        <!-- No config state -->
        <div v-if="detailConfig.total_config_items === 0" class="detail-empty">
          <a-empty description="该用户未做任何自定义配置" />
        </div>

        <!-- Tab nav -->
        <div v-else class="detail-tab-nav">
          <button
            v-for="tab in [
              { key: 'audit', label: '审核工作台', icon: AppstoreOutlined, count: detailConfig.audit_details.length },
              { key: 'cron', label: '定时任务', icon: ClockCircleOutlined, count: detailConfig.cron_details.length },
              { key: 'archive', label: '归档复盘', icon: FolderOpenOutlined, count: detailConfig.archive_details.length },
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

        <!-- ===== Audit workbench details ===== -->
        <div v-if="detailTab === 'audit' && detailConfig.total_config_items > 0" class="detail-content">
          <div v-if="detailConfig.audit_details.length === 0" class="detail-empty-tab">
            未修改审核工作台配置
          </div>
          <div v-for="proc in detailConfig.audit_details" :key="proc.process_type" class="detail-process-card">
            <div class="detail-process-header">
              <span class="detail-process-name">{{ proc.process_type }}</span>
            </div>

            <!-- Strictness override -->
            <div v-if="proc.strictness_override" class="detail-config-block">
              <div class="detail-config-label"><ControlOutlined /> 审核尺度</div>
              <div class="detail-config-value">
                <span class="strictness-tag" :style="{ color: strictnessLabels[proc.strictness_override]?.color }">
                  {{ strictnessLabels[proc.strictness_override]?.label }}
                </span>
                <span class="text-secondary" style="font-size: 12px; margin-left: 4px;">（用户自定义）</span>
              </div>
            </div>

            <!-- Custom rules -->
            <div v-if="proc.custom_rules.length > 0" class="detail-config-block">
              <div class="detail-config-label"><NodeIndexOutlined /> 自定义规则</div>
              <div class="detail-rule-list">
                <div v-for="rule in proc.custom_rules" :key="rule.id" class="detail-rule-item">
                  <span class="detail-rule-dot" :class="rule.enabled ? 'detail-rule-dot--on' : 'detail-rule-dot--off'" />
                  <span class="detail-rule-text">{{ rule.content }}</span>
                  <span class="detail-rule-status">{{ rule.enabled ? '启用' : '禁用' }}</span>
                </div>
              </div>
            </div>

            <!-- Field overrides -->
            <div v-if="proc.field_overrides.length > 0" class="detail-config-block">
              <div class="detail-config-label"><AppstoreOutlined /> 字段选择变更</div>
              <div class="detail-tag-list">
                <span v-for="f in proc.field_overrides" :key="f" class="detail-field-tag">{{ f }}</span>
              </div>
            </div>

            <!-- Rule toggle overrides -->
            <div v-if="proc.rule_toggle_overrides.length > 0" class="detail-config-block">
              <div class="detail-config-label"><SwapOutlined /> 规则开关变更</div>
              <div class="detail-rule-list">
                <div v-for="r in proc.rule_toggle_overrides" :key="r.rule_id" class="detail-rule-item">
                  <span class="detail-rule-dot" :class="r.enabled ? 'detail-rule-dot--on' : 'detail-rule-dot--off'" />
                  <span class="detail-rule-text">{{ r.rule_content }}</span>
                  <span class="detail-rule-status">{{ r.enabled ? '开启' : '关闭' }}</span>
                </div>
              </div>
            </div>

            <div v-if="!proc.strictness_override && proc.custom_rules.length === 0 && proc.field_overrides.length === 0 && proc.rule_toggle_overrides.length === 0" class="detail-empty-tab">
              该流程无自定义配置
            </div>
          </div>
        </div>

        <!-- ===== Cron details ===== -->
        <div v-if="detailTab === 'cron' && detailConfig.total_config_items > 0" class="detail-content">
          <div v-if="detailConfig.cron_details.length === 0" class="detail-empty-tab">
            未修改定时任务配置
          </div>
          <div v-for="cron in detailConfig.cron_details" :key="cron.task_type" class="detail-process-card">
            <div class="detail-process-header">
              <span class="detail-process-name">{{ cron.task_label }}</span>
            </div>

            <!-- Email override -->
            <div v-if="cron.email_override" class="detail-config-block">
              <div class="detail-config-label"><MailOutlined /> 自定义推送邮箱</div>
              <div class="detail-config-value text-mono">{{ cron.email_override }}</div>
            </div>

            <!-- Template override -->
            <div v-if="cron.template_override" class="detail-config-block">
              <div class="detail-config-label"><FileTextOutlined /> 模板自定义</div>
              <div class="detail-template-list">
                <div v-if="cron.template_override.subject" class="detail-template-item">
                  <span class="detail-template-key">邮件主题</span>
                  <span class="detail-template-val">{{ cron.template_override.subject }}</span>
                </div>
                <div v-if="cron.template_override.header" class="detail-template-item">
                  <span class="detail-template-key">头部内容</span>
                  <span class="detail-template-val">{{ cron.template_override.header }}</span>
                </div>
                <div v-if="cron.template_override.body_template" class="detail-template-item">
                  <span class="detail-template-key">正文模板</span>
                  <span class="detail-template-val">{{ cron.template_override.body_template }}</span>
                </div>
                <div v-if="cron.template_override.footer" class="detail-template-item">
                  <span class="detail-template-key">底部内容</span>
                  <span class="detail-template-val">{{ cron.template_override.footer }}</span>
                </div>
                <div v-if="cron.template_override.include_ai_summary !== undefined" class="detail-template-item">
                  <span class="detail-template-key">AI 摘要</span>
                  <span class="detail-template-val">{{ cron.template_override.include_ai_summary ? '包含' : '不包含' }}</span>
                </div>
                <div v-if="cron.template_override.include_statistics !== undefined" class="detail-template-item">
                  <span class="detail-template-key">统计数据</span>
                  <span class="detail-template-val">{{ cron.template_override.include_statistics ? '包含' : '不包含' }}</span>
                </div>
                <div v-if="cron.template_override.include_detail_list !== undefined" class="detail-template-item">
                  <span class="detail-template-key">明细列表</span>
                  <span class="detail-template-val">{{ cron.template_override.include_detail_list ? '包含' : '不包含' }}</span>
                </div>
              </div>
            </div>

            <!-- Prompt override -->
            <div v-if="cron.prompt_override" class="detail-config-block">
              <div class="detail-config-label"><EditOutlined /> 提示词自定义</div>
              <div class="detail-config-value" style="white-space: pre-wrap; font-size: 12px;">{{ cron.prompt_override }}</div>
            </div>

            <div v-if="!cron.email_override && !cron.template_override && !cron.prompt_override" class="detail-empty-tab">
              该任务类型无自定义配置
            </div>
          </div>
        </div>

        <!-- ===== Archive details ===== -->
        <div v-if="detailTab === 'archive' && detailConfig.total_config_items > 0" class="detail-content">
          <div v-if="detailConfig.archive_details.length === 0" class="detail-empty-tab">
            未修改归档复盘配置
          </div>
          <div v-for="arc in detailConfig.archive_details" :key="arc.process_type" class="detail-process-card">
            <div class="detail-process-header">
              <span class="detail-process-name">{{ arc.process_type }}</span>
            </div>

            <!-- Strictness override -->
            <div v-if="arc.strictness_override" class="detail-config-block">
              <div class="detail-config-label"><ControlOutlined /> 复核尺度</div>
              <div class="detail-config-value">
                <span class="strictness-tag" :style="{ color: strictnessLabels[arc.strictness_override]?.color }">
                  {{ strictnessLabels[arc.strictness_override]?.label }}
                </span>
                <span class="text-secondary" style="font-size: 12px; margin-left: 4px;">（用户自定义）</span>
              </div>
            </div>

            <!-- Custom rules -->
            <div v-if="arc.custom_rules.length > 0" class="detail-config-block">
              <div class="detail-config-label"><NodeIndexOutlined /> 自定义复核规则</div>
              <div class="detail-rule-list">
                <div v-for="rule in arc.custom_rules" :key="rule.id" class="detail-rule-item">
                  <span class="detail-rule-dot" :class="rule.enabled ? 'detail-rule-dot--on' : 'detail-rule-dot--off'" />
                  <span class="detail-rule-text">{{ rule.content }}</span>
                  <span class="detail-rule-status">{{ rule.enabled ? '启用' : '禁用' }}</span>
                </div>
              </div>
            </div>

            <!-- Custom flow rules -->
            <div v-if="arc.custom_flow_rules.length > 0" class="detail-config-block">
              <div class="detail-config-label"><ApartmentOutlined /> 自定义审批流规则</div>
              <div class="detail-rule-list">
                <div v-for="rule in arc.custom_flow_rules" :key="rule.id" class="detail-rule-item">
                  <span class="detail-rule-dot" :class="rule.enabled ? 'detail-rule-dot--on' : 'detail-rule-dot--off'" />
                  <span class="detail-rule-text">{{ rule.content }}</span>
                  <span class="detail-rule-status">{{ rule.enabled ? '启用' : '禁用' }}</span>
                </div>
              </div>
            </div>

            <!-- Field overrides -->
            <div v-if="arc.field_overrides.length > 0" class="detail-config-block">
              <div class="detail-config-label"><AppstoreOutlined /> 字段选择变更</div>
              <div class="detail-tag-list">
                <span v-for="f in arc.field_overrides" :key="f" class="detail-field-tag">{{ f }}</span>
              </div>
            </div>

            <div v-if="!arc.strictness_override && arc.custom_rules.length === 0 && arc.custom_flow_rules.length === 0 && arc.field_overrides.length === 0" class="detail-empty-tab">
              该流程无自定义配置
            </div>
          </div>
        </div>

        <!-- Footer -->
        <div v-if="detailConfig.last_modified" class="detail-footer-info">
          最后修改时间：{{ detailConfig.last_modified }}
        </div>
      </template>
    </a-drawer>
  </div>
</template>

<style scoped>
.page-header { margin-bottom: 24px; }
.page-title { font-size: 24px; font-weight: 700; color: var(--color-text-primary); margin: 0; }
.page-subtitle { font-size: 14px; color: var(--color-text-tertiary); margin: 4px 0 0; }

.toolbar { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; gap: 12px; flex-wrap: wrap; }
.toolbar-left { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }

.stats-row { display: flex; gap: 12px; margin-bottom: 16px; flex-wrap: wrap; }
.stat-card {
  background: var(--color-bg-card); border-radius: var(--radius-md);
  border: 1px solid var(--color-border-light); padding: 14px 20px; min-width: 120px;
}
.stat-value { font-size: 22px; font-weight: 700; color: var(--color-text-primary); }
.stat-label { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }

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

.user-cell { display: flex; align-items: center; gap: 10px; }
.user-avatar { background: var(--color-primary-bg); color: var(--color-primary); flex-shrink: 0; }
.user-name { font-weight: 600; font-size: 13px; color: var(--color-text-primary); }
.user-username { font-size: 11px; color: var(--color-text-tertiary); font-family: monospace; }

.count-badge {
  font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: var(--radius-full); white-space: nowrap;
}
.count-badge--primary { background: var(--color-primary-bg); color: var(--color-primary); }
.count-badge--info { background: var(--color-info-bg); color: var(--color-info); }
.count-badge--warning { background: var(--color-warning-bg); color: var(--color-warning); }

.config-total { font-weight: 600; color: var(--color-text-primary); font-size: 13px; }

.action-btns { display: flex; gap: 4px; }
.icon-btn {
  width: 28px; height: 28px; border: 1px solid var(--color-border); background: transparent;
  border-radius: var(--radius-sm); cursor: pointer; display: flex; align-items: center;
  justify-content: center; color: var(--color-text-tertiary); transition: all var(--transition-fast);
}
.icon-btn:hover { border-color: var(--color-primary); color: var(--color-primary); }

.pagination-wrapper { margin-top: 16px; display: flex; justify-content: flex-end; }

/* ===== Detail drawer ===== */
.detail-user-header {
  display: flex; align-items: center; gap: 12px; margin-bottom: 20px;
  padding-bottom: 16px; border-bottom: 1px solid var(--color-border-light);
}
.detail-user-name { font-size: 16px; font-weight: 700; color: var(--color-text-primary); }
.detail-user-meta { font-size: 13px; color: var(--color-text-tertiary); }
.detail-user-stats { margin-left: auto; }

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
.detail-process-header { display: flex; align-items: center; gap: 8px; }
.detail-process-name { font-size: 14px; font-weight: 600; color: var(--color-text-primary); }

.detail-config-block { display: flex; flex-direction: column; gap: 6px; }
.detail-config-label {
  font-size: 12px; font-weight: 600; color: var(--color-text-secondary);
  display: flex; align-items: center; gap: 6px;
}
.detail-config-value { font-size: 13px; color: var(--color-text-primary); padding-left: 20px; }

.strictness-tag { font-weight: 600; font-size: 13px; }

/* Rule list */
.detail-rule-list { display: flex; flex-direction: column; gap: 6px; padding-left: 20px; }
.detail-rule-item {
  display: flex; align-items: flex-start; gap: 8px; font-size: 13px;
  padding: 6px 10px; background: var(--color-bg-card); border-radius: var(--radius-sm);
  border: 1px solid var(--color-border-light);
}
.detail-rule-dot {
  width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; margin-top: 5px;
}
.detail-rule-dot--on { background: var(--color-success); }
.detail-rule-dot--off { background: var(--color-text-tertiary); }
.detail-rule-text { flex: 1; color: var(--color-text-primary); line-height: 1.5; }
.detail-rule-status {
  font-size: 11px; color: var(--color-text-tertiary); flex-shrink: 0; margin-top: 2px;
}

/* Field tags */
.detail-tag-list { display: flex; flex-wrap: wrap; gap: 6px; padding-left: 20px; }
.detail-field-tag {
  font-size: 12px; font-weight: 500; padding: 3px 10px; border-radius: var(--radius-full);
  background: var(--color-info-bg); color: var(--color-info); border: 1px solid transparent;
}

/* Template list */
.detail-template-list { display: flex; flex-direction: column; gap: 4px; padding-left: 20px; }
.detail-template-item {
  display: flex; gap: 8px; font-size: 12px; padding: 4px 0;
  border-bottom: 1px dashed var(--color-border-light);
}
.detail-template-item:last-child { border-bottom: none; }
.detail-template-key {
  font-weight: 600; color: var(--color-text-secondary); min-width: 70px; flex-shrink: 0;
}
.detail-template-val { color: var(--color-text-primary); word-break: break-all; }

.detail-footer-info {
  font-size: 12px; color: var(--color-text-tertiary);
  padding-top: 16px; margin-top: 16px; border-top: 1px solid var(--color-border-light);
}

@media (max-width: 768px) {
  .stats-row { flex-direction: column; }
  .data-table-card { overflow-x: auto; -webkit-overflow-scrolling: touch; }
  .data-table { min-width: 800px; }
  .toolbar { flex-direction: column; align-items: stretch; }
  .toolbar-left { flex-direction: column; }
  .toolbar-left > * { width: 100% !important; }
  .page-title { font-size: 20px; }
  .stat-card { min-width: auto; }
  .detail-tab-nav { flex-direction: column; }
}
</style>
