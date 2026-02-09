<script setup lang="ts">
import { PlusOutlined } from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'

definePageMeta({ middleware: 'auth' })

const config = useRuntimeConfig()
const { token } = useAuth()

const activeTab = ref('rules')
const rules = ref<any[]>([])
const showEditor = ref(false)
const loading = ref(false)

const headers = computed(() => ({
  Authorization: `Bearer ${token.value}`,
}))

const fetchRules = async () => {
  loading.value = true
  try {
    const data = await $fetch<{ rules: any[] }>(`${config.public.apiBase}/api/admin/rules`, {
      headers: headers.value,
    })
    rules.value = data.rules
  } catch {
    rules.value = []
  } finally {
    loading.value = false
  }
}

const handleSaveRule = async (rule: any) => {
  try {
    await $fetch(`${config.public.apiBase}/api/admin/rules`, {
      method: 'POST',
      headers: headers.value,
      body: rule,
    })
    message.success('规则已保存')
    showEditor.value = false
    await fetchRules()
  } catch {
    message.error('保存失败')
  }
}

onMounted(fetchRules)

const ruleColumns = [
  { title: '流程类型', dataIndex: 'process_type', key: 'process_type' },
  { title: '规则内容', dataIndex: 'rule_content', key: 'content', ellipsis: true },
  { title: '级别', dataIndex: 'rule_scope', key: 'scope' },
  { title: '优先级', dataIndex: 'priority', key: 'priority' },
]

const scopeColors: Record<string, string> = {
  mandatory: 'red',
  default_on: 'blue',
  default_off: 'default',
}
</script>

<template>
  <div>
    <a-page-header title="租户配置" sub-title="知识库模式与规则管理" />

    <a-tabs v-model:activeKey="activeTab">
      <a-tab-pane key="rules" tab="审核规则">
        <a-button type="primary" style="margin-bottom: 16px;" @click="showEditor = true">
          <PlusOutlined /> 新增规则
        </a-button>
        <a-table :columns="ruleColumns" :data-source="rules" :loading="loading" row-key="id" size="small">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'scope'">
              <a-tag :color="scopeColors[record.rule_scope]">{{ record.rule_scope }}</a-tag>
            </template>
          </template>
        </a-table>
      </a-tab-pane>

      <a-tab-pane key="kb" tab="知识库模式">
        <a-alert message="第一阶段仅支持 Rules_Only 模式" type="info" show-icon style="margin-bottom: 16px;" />
        <a-radio-group value="rules_only">
          <a-radio-button value="rules_only">仅规则库</a-radio-button>
          <a-radio-button value="rag_only" disabled>仅制度库 (RAG)</a-radio-button>
          <a-radio-button value="hybrid" disabled>混合模式</a-radio-button>
        </a-radio-group>
      </a-tab-pane>

      <a-tab-pane key="retention" tab="日志留存">
        <a-form layout="inline">
          <a-form-item label="保留策略">
            <a-select style="width: 200px;" default-value="permanent">
              <a-select-option value="permanent">永久保存</a-select-option>
              <a-select-option value="1095">保存 3 年</a-select-option>
              <a-select-option value="365">保存 1 年</a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item>
            <a-button type="primary">保存</a-button>
          </a-form-item>
        </a-form>
      </a-tab-pane>
    </a-tabs>

    <RuleEditor :open="showEditor" @close="showEditor = false" @save="handleSaveRule" />
  </div>
</template>
