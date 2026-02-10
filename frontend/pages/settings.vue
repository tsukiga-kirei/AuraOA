<script setup lang="ts">
import {
  UserOutlined,
  MailOutlined,
  PhoneOutlined,
  SaveOutlined,
  PlusOutlined,
  DeleteOutlined,
  SettingOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'

definePageMeta({ middleware: 'auth' })

const { userRole } = useAuth()

const activeTab = ref('profile')

// ===== Profile tab =====
const profile = ref({
  nickname: '张明',
  email: 'zhangming@example.com',
  phone: '138****8888',
  department: '研发部',
  position: '高级工程师',
})

const roleLabels: Record<string, string> = {
  business: '业务用户',
  tenant_admin: '租户管理员',
  system_admin: '系统管理员',
}

// ===== Audit workbench tab =====
// Mock: user's assigned process types with audit config
const assignedProcesses = ref([
  {
    id: 'AP-001',
    process_type: '采购审批',
    flow_path: '部门经理 → 财务总监 → 总经理',
    audit_strictness: 'standard' as string,
    custom_rules: [
      { id: 'CR-001', content: '供应商必须在合格名录中', enabled: true },
    ],
    system_rules: [
      { id: 'R001', content: '单笔采购金额不得超过部门季度预算上限', scope: 'mandatory', enabled: true, locked: true },
      { id: 'R002', content: '超过10万元需提供至少3家供应商比价', scope: 'mandatory', enabled: true, locked: true },
    ],
  },
  {
    id: 'AP-002',
    process_type: '费用报销',
    flow_path: '部门经理 → 财务部',
    audit_strictness: 'loose' as string,
    custom_rules: [],
    system_rules: [
      { id: 'R003', content: '单次报销金额超过5000元需附发票原件', scope: 'default_on', enabled: true, locked: false },
      { id: 'R006', content: '差旅住宿标准不超过城市限额', scope: 'default_off', enabled: false, locked: false },
    ],
  },
  {
    id: 'AP-003',
    process_type: '合同审批',
    flow_path: '部门经理 → 法务部 → 总经理',
    audit_strictness: 'strict' as string,
    custom_rules: [
      { id: 'CR-002', content: '合同期限超过2年需额外审批', enabled: true },
    ],
    system_rules: [
      { id: 'R004', content: '合同金额超过50万需法务部会签', scope: 'mandatory', enabled: true, locked: true },
    ],
  },
])

const selectedProcessId = ref(assignedProcesses.value[0]?.id || '')

const selectedProcess = computed(() =>
  assignedProcesses.value.find(p => p.id === selectedProcessId.value)
)

const strictnessOptions = [
  { value: 'strict', label: '严格', desc: '所有规则严格执行，零容忍' },
  { value: 'standard', label: '标准', desc: '按规则默认配置执行' },
  { value: 'loose', label: '宽松', desc: '仅校验强制规则，其余提示' },
]

const newRuleContent = ref('')

const addCustomRule = () => {
  if (!newRuleContent.value.trim() || !selectedProcess.value) return
  selectedProcess.value.custom_rules.push({
    id: `CR-${Date.now()}`,
    content: newRuleContent.value.trim(),
    enabled: true,
  })
  newRuleContent.value = ''
  message.success('自定义规则已添加')
}

const removeCustomRule = (ruleId: string) => {
  if (!selectedProcess.value) return
  selectedProcess.value.custom_rules = selectedProcess.value.custom_rules.filter(r => r.id !== ruleId)
  message.success('已删除')
}

const saving = ref(false)

const handleSave = async () => {
  saving.value = true
  await new Promise(r => setTimeout(r, 800))
  saving.value = false
  message.success('设置已保存')
}
</script>

<template>
  <div class="settings-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">个人设置</h1>
        <p class="page-subtitle">管理您的账户信息与审核偏好</p>
      </div>
    </div>

    <!-- Tab navigation -->
    <div class="tab-nav">
      <button
        v-for="tab in [
          { key: 'profile', label: '基本信息' },
          { key: 'workbench', label: '审核工作台' },
        ]"
        :key="tab.key"
        class="tab-btn"
        :class="{ 'tab-btn--active': activeTab === tab.key }"
        @click="activeTab = tab.key"
      >
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
              <span class="role-badge">{{ roleLabels[userRole] || '业务用户' }}</span>
            </div>
          </div>
        </div>

        <a-form layout="vertical" class="settings-form">
          <div class="form-row">
            <a-form-item label="昵称" class="form-col">
              <a-input v-model:value="profile.nickname" size="large">
                <template #prefix><UserOutlined class="input-icon" /></template>
              </a-input>
            </a-form-item>
            <a-form-item label="邮箱" class="form-col">
              <a-input v-model:value="profile.email" size="large">
                <template #prefix><MailOutlined class="input-icon" /></template>
              </a-input>
            </a-form-item>
          </div>
          <div class="form-row">
            <a-form-item label="手机号" class="form-col">
              <a-input v-model:value="profile.phone" size="large">
                <template #prefix><PhoneOutlined class="input-icon" /></template>
              </a-input>
            </a-form-item>
            <a-form-item label="部门" class="form-col">
              <a-input v-model:value="profile.department" size="large" disabled />
            </a-form-item>
          </div>
          <a-form-item label="职位">
            <a-input v-model:value="profile.position" size="large" disabled />
          </a-form-item>
        </a-form>

        <div class="settings-actions">
          <a-button type="primary" size="large" :loading="saving" @click="handleSave">
            <SaveOutlined /> 保存
          </a-button>
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
            v-for="proc in assignedProcesses"
            :key="proc.id"
            class="process-list-item"
            :class="{ 'process-list-item--active': selectedProcessId === proc.id }"
            @click="selectedProcessId = proc.id"
          >
            <div class="process-list-item-name">{{ proc.process_type }}</div>
            <div class="process-list-item-path">{{ proc.flow_path }}</div>
          </div>
          <div v-if="assignedProcesses.length === 0" class="process-list-empty">
            暂无分配的审核流程
          </div>
        </div>

        <!-- Right: config detail -->
        <div v-if="selectedProcess" class="process-config-panel">
          <h3 class="config-title">{{ selectedProcess.process_type }} - 审核配置</h3>
          <p class="config-subtitle">流程路径：{{ selectedProcess.flow_path }}</p>

          <!-- Audit strictness -->
          <div class="config-section">
            <h4 class="config-section-title">审核尺度</h4>
            <div class="strictness-options">
              <div
                v-for="opt in strictnessOptions"
                :key="opt.value"
                class="strictness-option"
                :class="{ 'strictness-option--active': selectedProcess.audit_strictness === opt.value }"
                @click="selectedProcess.audit_strictness = opt.value"
              >
                <div class="strictness-option-radio" />
                <div>
                  <div class="strictness-option-label">{{ opt.label }}</div>
                  <div class="strictness-option-desc">{{ opt.desc }}</div>
                </div>
              </div>
            </div>
          </div>

          <!-- System rules (from tenant config) -->
          <div class="config-section">
            <h4 class="config-section-title">通用审核规则（租户配置）</h4>
            <div class="rule-config-list">
              <div v-for="rule in selectedProcess.system_rules" :key="rule.id" class="rule-config-item">
                <div class="rule-config-content">
                  <span class="rule-config-text">{{ rule.content }}</span>
                  <span v-if="rule.scope === 'mandatory'" class="rule-scope-tag rule-scope-tag--mandatory">强制</span>
                  <span v-else-if="rule.scope === 'default_on'" class="rule-scope-tag rule-scope-tag--on">默认开启</span>
                  <span v-else class="rule-scope-tag rule-scope-tag--off">默认关闭</span>
                </div>
                <a-switch
                  v-model:checked="rule.enabled"
                  size="small"
                  :disabled="rule.locked"
                />
              </div>
            </div>
          </div>

          <!-- Custom rules (user private) -->
          <div class="config-section">
            <h4 class="config-section-title">个人自定义规则</h4>
            <p class="config-section-desc">您可以为此流程添加个人审核规则，优先级低于租户强制规则</p>
            <div class="rule-config-list">
              <div v-for="rule in selectedProcess.custom_rules" :key="rule.id" class="rule-config-item">
                <div class="rule-config-content">
                  <span class="rule-config-text">{{ rule.content }}</span>
                  <span class="rule-scope-tag rule-scope-tag--custom">个人</span>
                </div>
                <div class="rule-config-actions">
                  <a-switch v-model:checked="rule.enabled" size="small" />
                  <a-popconfirm title="确认删除？" @confirm="removeCustomRule(rule.id)">
                    <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
                  </a-popconfirm>
                </div>
              </div>
            </div>
            <div class="add-rule-row">
              <a-input
                v-model:value="newRuleContent"
                placeholder="输入自定义规则内容..."
                @pressEnter="addCustomRule"
              />
              <a-button type="primary" :disabled="!newRuleContent.trim()" @click="addCustomRule">
                <PlusOutlined /> 添加
              </a-button>
            </div>
          </div>

          <div class="settings-actions">
            <a-button type="primary" size="large" :loading="saving" @click="handleSave">
              <SaveOutlined /> 保存配置
            </a-button>
          </div>
        </div>

        <div v-else class="process-config-empty">
          <a-empty description="请选择左侧流程查看配置" />
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
  display: flex; gap: 4px; background: var(--color-bg-hover); padding: 4px;
  border-radius: var(--radius-lg); margin-bottom: 24px; width: fit-content;
}
.tab-btn {
  padding: 8px 20px; border: none; background: transparent; border-radius: var(--radius-md);
  font-size: 14px; font-weight: 500; color: var(--color-text-secondary); cursor: pointer;
  transition: all var(--transition-fast);
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
.workbench-layout { display: grid; grid-template-columns: 260px 1fr; gap: 20px; align-items: start; }

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
.process-list-empty { padding: 24px; text-align: center; color: var(--color-text-tertiary); font-size: 13px; }

.process-config-panel {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); padding: 24px;
}
.process-config-empty {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); padding: 48px;
}

.config-title { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 4px; }
.config-subtitle { font-size: 13px; color: var(--color-text-tertiary); margin: 0 0 24px; }

.config-section { margin-bottom: 24px; }
.config-section-title { font-size: 14px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 12px; }
.config-section-desc { font-size: 12px; color: var(--color-text-tertiary); margin: -8px 0 12px; }

/* Strictness options */
.strictness-options { display: flex; flex-direction: column; gap: 8px; }
.strictness-option {
  display: flex; align-items: center; gap: 14px; padding: 12px 16px;
  border: 2px solid var(--color-border-light); border-radius: var(--radius-md);
  cursor: pointer; transition: all var(--transition-fast);
}
.strictness-option:hover { border-color: var(--color-primary-lighter); }
.strictness-option--active { border-color: var(--color-primary); background: var(--color-primary-bg); }
.strictness-option-radio {
  width: 18px; height: 18px; border-radius: 50%; border: 2px solid var(--color-border);
  flex-shrink: 0; transition: all var(--transition-fast);
}
.strictness-option--active .strictness-option-radio { border-color: var(--color-primary); border-width: 5px; }
.strictness-option-label { font-size: 14px; font-weight: 500; color: var(--color-text-primary); }
.strictness-option-desc { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }

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

.add-rule-row { display: flex; gap: 8px; }
.add-rule-row :deep(.ant-btn-primary) { font-weight: 600; }
.add-rule-row :deep(.ant-btn-primary[disabled]) {
  background: var(--color-primary); color: #fff; opacity: 0.45;
  border-color: var(--color-primary);
}

@media (max-width: 768px) {
  .form-row { grid-template-columns: 1fr; }
  .workbench-layout { grid-template-columns: 1fr; }
}
</style>
