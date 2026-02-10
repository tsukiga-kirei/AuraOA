<script setup lang="ts">
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  LockOutlined,
  UnlockOutlined,
  DatabaseOutlined,
  FileTextOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'

definePageMeta({ middleware: 'auth' })

const { mockRules } = useMockData()

const activeTab = ref('rules')
const rules = ref([...mockRules])
const showEditor = ref(false)
const editingRule = ref<any>(null)

const kbMode = ref('rules_only')

const scopeConfig: Record<string, { label: string; color: string; bg: string; icon: any }> = {
  mandatory: { label: '强制执行', color: '#ef4444', bg: '#fef2f2', icon: LockOutlined },
  default_on: { label: '默认开启', color: '#4f46e5', bg: '#eef2ff', icon: UnlockOutlined },
  default_off: { label: '默认关闭', color: '#94a3b8', bg: '#f1f5f9', icon: UnlockOutlined },
}

const handleSaveRule = (rule: any) => {
  if (editingRule.value) {
    const idx = rules.value.findIndex(r => r.id === editingRule.value.id)
    if (idx >= 0) rules.value[idx] = { ...editingRule.value, ...rule }
  } else {
    rules.value.push({ id: `R${Date.now()}`, ...rule, enabled: true })
  }
  showEditor.value = false
  editingRule.value = null
  message.success('规则已保存')
}

const deleteRule = (id: string) => {
  rules.value = rules.value.filter(r => r.id !== id)
  message.success('已删除')
}

const openEditor = (rule?: any) => {
  editingRule.value = rule || null
  showEditor.value = true
}

const kbModes = [
  { key: 'rules_only', icon: FileTextOutlined, title: '仅规则库', desc: '结构化 Checklist 审核', available: true },
  { key: 'rag_only', icon: DatabaseOutlined, title: '仅制度库 (RAG)', desc: 'PDF/Word 文档检索增强', available: false },
  { key: 'hybrid', icon: ThunderboltOutlined, title: '混合模式', desc: '规则库 + 制度库联合审核', available: false },
]

const retentionPolicy = ref('permanent')
</script>

<template>
  <div class="tenant-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">租户配置</h1>
        <p class="page-subtitle">知识库模式与审核规则管理</p>
      </div>
    </div>

    <div class="tab-nav">
      <button
        v-for="tab in [
          { key: 'rules', label: '审核规则' },
          { key: 'kb', label: '知识库模式' },
          { key: 'retention', label: '日志留存' },
        ]"
        :key="tab.key"
        class="tab-btn"
        :class="{ 'tab-btn--active': activeTab === tab.key }"
        @click="activeTab = tab.key"
      >
        {{ tab.label }}
      </button>
    </div>

    <!-- Rules tab -->
    <div v-if="activeTab === 'rules'" class="tab-content">
      <div class="tab-content-header">
        <span class="tab-content-count">共 {{ rules.length }} 条规则</span>
        <a-button type="primary" @click="openEditor()">
          <PlusOutlined /> 新增规则
        </a-button>
      </div>

      <div class="rules-list">
        <div v-for="rule in rules" :key="rule.id" class="rule-card">
          <div class="rule-card-left">
            <div class="rule-scope-badge" :style="{ color: scopeConfig[rule.rule_scope]?.color, background: scopeConfig[rule.rule_scope]?.bg }">
              <component :is="scopeConfig[rule.rule_scope]?.icon" />
              {{ scopeConfig[rule.rule_scope]?.label }}
            </div>
            <div class="rule-card-body">
              <div class="rule-card-content">{{ rule.rule_content }}</div>
              <div class="rule-card-meta">
                <span class="rule-type-tag">{{ rule.process_type }}</span>
                <span>优先级: {{ rule.priority }}</span>
              </div>
            </div>
          </div>
          <div class="rule-card-actions">
            <a-switch v-model:checked="rule.enabled" size="small" />
            <button class="icon-btn" @click="openEditor(rule)">
              <EditOutlined />
            </button>
            <a-popconfirm title="确认删除此规则？" @confirm="deleteRule(rule.id)">
              <button class="icon-btn icon-btn--danger">
                <DeleteOutlined />
              </button>
            </a-popconfirm>
          </div>
        </div>
      </div>
    </div>

    <!-- Knowledge base tab -->
    <div v-if="activeTab === 'kb'" class="tab-content">
      <div class="kb-modes">
        <div
          v-for="mode in kbModes"
          :key="mode.key"
          class="kb-mode-card"
          :class="{
            'kb-mode-card--active': kbMode === mode.key,
            'kb-mode-card--disabled': !mode.available,
          }"
          @click="mode.available && (kbMode = mode.key)"
        >
          <div class="kb-mode-icon">
            <component :is="mode.icon" />
          </div>
          <div class="kb-mode-info">
            <div class="kb-mode-title">{{ mode.title }}</div>
            <div class="kb-mode-desc">{{ mode.desc }}</div>
          </div>
          <div v-if="kbMode === mode.key" class="kb-mode-check">✓</div>
          <div v-if="!mode.available" class="kb-mode-badge">即将推出</div>
        </div>
      </div>
    </div>

    <!-- Retention tab -->
    <div v-if="activeTab === 'retention'" class="tab-content">
      <div class="retention-card">
        <h4>审计日志保留策略</h4>
        <p>所有审核记录不可篡改，选择保留时长后将自动归档过期数据。</p>
        <div class="retention-options">
          <div
            v-for="opt in [
              { value: 'permanent', label: '永久保存', desc: '所有记录永不删除' },
              { value: '1095', label: '保存 3 年', desc: '超过 3 年的记录自动归档' },
              { value: '365', label: '保存 1 年', desc: '超过 1 年的记录自动归档' },
            ]"
            :key="opt.value"
            class="retention-option"
            :class="{ 'retention-option--active': retentionPolicy === opt.value }"
            @click="retentionPolicy = opt.value"
          >
            <div class="retention-option-radio" />
            <div>
              <div class="retention-option-label">{{ opt.label }}</div>
              <div class="retention-option-desc">{{ opt.desc }}</div>
            </div>
          </div>
        </div>
        <a-button type="primary" style="margin-top: 20px;" @click="message.success('保存成功')">
          保存设置
        </a-button>
      </div>
    </div>

    <!-- Rule editor -->
    <RuleEditor :open="showEditor" :rule="editingRule" @close="showEditor = false; editingRule = null" @save="handleSaveRule" />
  </div>
</template>

<style scoped>
.page-header {
  margin-bottom: 24px;
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

/* Tabs */
.tab-nav {
  display: flex;
  gap: 4px;
  background: var(--color-bg-hover);
  padding: 4px;
  border-radius: var(--radius-lg);
  margin-bottom: 24px;
  width: fit-content;
}

.tab-btn {
  padding: 8px 20px;
  border: none;
  background: transparent;
  border-radius: var(--radius-md);
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.tab-btn:hover {
  color: var(--color-text-primary);
}

.tab-btn--active {
  background: var(--color-bg-card);
  color: var(--color-primary);
  box-shadow: var(--shadow-xs);
}

/* Tab content */
.tab-content-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.tab-content-count {
  font-size: 13px;
  color: var(--color-text-tertiary);
}

/* Rules */
.rules-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.rule-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light);
  transition: all var(--transition-fast);
  gap: 16px;
}

.rule-card:hover {
  box-shadow: var(--shadow-sm);
}

.rule-card-left {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  flex: 1;
  min-width: 0;
}

.rule-scope-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 600;
  padding: 4px 10px;
  border-radius: var(--radius-full);
  white-space: nowrap;
  flex-shrink: 0;
}

.rule-card-content {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-primary);
  margin-bottom: 6px;
}

.rule-card-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 12px;
  color: var(--color-text-tertiary);
}

.rule-type-tag {
  padding: 1px 8px;
  background: var(--color-bg-hover);
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 500;
}

.rule-card-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.icon-btn {
  width: 32px;
  height: 32px;
  border: 1px solid var(--color-border);
  background: transparent;
  border-radius: var(--radius-md);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-tertiary);
  transition: all var(--transition-fast);
}

.icon-btn:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.icon-btn--danger:hover {
  border-color: var(--color-danger);
  color: var(--color-danger);
}

/* KB modes */
.kb-modes {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.kb-mode-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  border: 2px solid var(--color-border-light);
  cursor: pointer;
  transition: all var(--transition-fast);
  position: relative;
}

.kb-mode-card:hover:not(.kb-mode-card--disabled) {
  border-color: var(--color-primary-lighter);
}

.kb-mode-card--active {
  border-color: var(--color-primary);
  background: var(--color-primary-bg);
}

.kb-mode-card--disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.kb-mode-icon {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  background: var(--color-bg-page);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  color: var(--color-primary);
  flex-shrink: 0;
}

.kb-mode-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.kb-mode-desc {
  font-size: 12px;
  color: var(--color-text-tertiary);
  margin-top: 2px;
}

.kb-mode-check {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: var(--color-primary);
  color: #fff;
  font-size: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.kb-mode-badge {
  position: absolute;
  top: 12px;
  right: 12px;
  font-size: 10px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: var(--radius-full);
  background: var(--color-bg-hover);
  color: var(--color-text-tertiary);
}

/* Retention */
.retention-card {
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light);
  padding: 24px;
  max-width: 560px;
}

.retention-card h4 {
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 8px;
}

.retention-card p {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin: 0 0 20px;
}

.retention-options {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.retention-option {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 14px 16px;
  border: 2px solid var(--color-border-light);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.retention-option:hover {
  border-color: var(--color-primary-lighter);
}

.retention-option--active {
  border-color: var(--color-primary);
  background: var(--color-primary-bg);
}

.retention-option-radio {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  border: 2px solid var(--color-border);
  flex-shrink: 0;
  transition: all var(--transition-fast);
}

.retention-option--active .retention-option-radio {
  border-color: var(--color-primary);
  border-width: 5px;
}

.retention-option-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-primary);
}

.retention-option-desc {
  font-size: 12px;
  color: var(--color-text-tertiary);
  margin-top: 2px;
}

@media (max-width: 640px) {
  .tab-nav {
    width: 100%;
  }

  .tab-btn {
    flex: 1;
    text-align: center;
  }

  .rule-card {
    flex-direction: column;
    align-items: flex-start;
  }

  .rule-card-actions {
    align-self: flex-end;
  }
}
</style>
