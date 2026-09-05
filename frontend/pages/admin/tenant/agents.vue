<script setup lang="ts">
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ThunderboltOutlined,
  ApiOutlined,
  BookOutlined,
  RobotOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type {
  AgentDefinitionItem,
  MCPServerItem,
  AgentSkillItem,
  SystemToolCatalogItem,
  SaveAgentRequest,
  SaveMCPServerRequest,
  SaveSkillRequest,
} from '~/types/chat'

definePageMeta({
  layout: 'default', middleware: ['auth'],
})

const { authFetch } = useAuth()
const { t } = useI18n()
const activeTab = ref('agents')
const loading = ref(false)

// 列表数据
const agents = ref<AgentDefinitionItem[]>([])
const mcpServers = ref<MCPServerItem[]>([])
const skills = ref<AgentSkillItem[]>([])
const systemTools = ref<SystemToolCatalogItem[]>([])

// 模态框状态
const agentModalVisible = ref(false)
const mcpModalVisible = ref(false)
const skillModalVisible = ref(false)

const agentForm = ref<SaveAgentRequest>({
  agent_code: '',
  name: '',
  description: '',

  system_prompt: '',
  enabled: true,

  tool_codes: [],
})

const mcpForm = ref<SaveMCPServerRequest>({
  server_code: '',
  name: '',
  description: '',
  transport_type: 'http',
  endpoint_url: '',
  headers: '',
  enabled: true,
})

const skillForm = ref<SaveSkillRequest>({
  skill_code: '',
  name: '',
  description: '',
  content: '', enabled: true,
})

// 加载数据
const fetchAll = async () => {
  loading.value = true
  try {
    const [agentsData, mcpData, skillsData, catalogData] = await Promise.all([
      authFetch<AgentDefinitionItem[]>('/api/tenant/agents'),
      authFetch<MCPServerItem[]>('/api/tenant/mcp-servers'),
      authFetch<AgentSkillItem[]>('/api/tenant/skills'),
      authFetch<{ tool_catalog: SystemToolCatalogItem[]; skill_catalog: AgentSkillItem[] }>('/api/tenant/agent-catalog'),
    ])
    agents.value = agentsData || []
    mcpServers.value = mcpData || []
    skills.value = skillsData || []
    systemTools.value = [...(catalogData?.tool_catalog || []), ...(catalogData?.skill_catalog || []).map(sk => ({tool_code: 'skill:' + sk.skill_code, name: sk.name, description: sk.description, ui_kind: 'skill'}))]
  } catch (err: any) {
    message.error(err.message || t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

// 智能体保存/删除
const openCreateAgent = () => {
  agentForm.value = {
    agent_code: '',
    name: '',
    description: '',

    system_prompt: '',
    enabled: true,

    tool_codes: [],
  }
  agentModalVisible.value = true
}

const openEditAgent = (item: AgentDefinitionItem) => {
  agentForm.value = {
    id: item.id,
    agent_code: item.agent_code,
    name: item.name,
    description: item.description,

    system_prompt: item.system_prompt,
    enabled: item.enabled,

    tool_codes: [...(item.tool_codes || [])],
  }
  agentModalVisible.value = true
}

const saveAgent = async () => {
  if (!agentForm.value.agent_code || !agentForm.value.name) {
    message.warning(t('agentAdmin.form.required'))
    return
  }
  try {
    await authFetch(agentForm.value.id ? `/api/tenant/agents/${agentForm.value.id}` : '/api/tenant/agents', {
      method: agentForm.value.id ? 'PUT' : 'POST',
      body: agentForm.value,
    })
    message.success(t('agentAdmin.saveSuccess'))
    agentModalVisible.value = false
    await fetchAll()
  } catch (err: any) {
    message.error(err.message || t('common.saveFailed'))
  }
}

const deleteAgent = async (id: string) => {
  try {
    await authFetch(`/api/tenant/agents/${id}`, { method: 'DELETE' })
    message.success(t('agentAdmin.deleteSuccess'))
    await fetchAll()
  } catch (err: any) {
    message.error(err.message || t('common.deleteFailed'))
  }
}

// MCP 保存/删除/测试
const openCreateMCP = () => {
  mcpForm.value = {
    server_code: '',
    name: '',
    description: '',
    transport_type: 'http',
    endpoint_url: '',
    headers: '',
    enabled: true,
  }
  mcpModalVisible.value = true
}

const saveMCP = async () => {
  if (!mcpForm.value.server_code || !mcpForm.value.endpoint_url) {
    message.warning(t('agentAdmin.form.required'))
    return
  }
  try {
    await authFetch(mcpForm.value.id ? `/api/tenant/mcp-servers/${mcpForm.value.id}` : '/api/tenant/mcp-servers', {
      method: mcpForm.value.id ? 'PUT' : 'POST',
      body: mcpForm.value,
    })
    message.success(t('agentAdmin.saveSuccess'))
    mcpModalVisible.value = false
    await fetchAll()
  } catch (err: any) {
    message.error(err.message || t('common.saveFailed'))
  }
}

const testMCP = async (id: string) => {
  try {
    const res = await authFetch<{ tools: any[] }>(`/api/tenant/mcp-servers/${id}/test`, {
      method: 'POST',
    })
    message.success(t('agentAdmin.testConnectionSuccess', [res.tools?.length || 0]))
    await fetchAll()
  } catch (err: any) {
    message.error(err.message || t('common.operationFailed'))
  }
}

const deleteMCP = async (id: string) => {
  try {
    await authFetch(`/api/tenant/mcp-servers/${id}`, { method: 'DELETE' })
    message.success(t('agentAdmin.deleteSuccess'))
    await fetchAll()
  } catch (err: any) {
    message.error(err.message || t('common.deleteFailed'))
  }
}

// Skills 保存/删除
const openCreateSkill = () => {
  skillForm.value = {
    skill_code: '',
    name: '',
    description: '',
    content: '', enabled: true,
  }
  skillModalVisible.value = true
}

const saveSkill = async () => {
  if (!skillForm.value.skill_code || !skillForm.value.content) {
    message.warning(t('agentAdmin.form.required'))
    return
  }
  try {
    await authFetch(skillForm.value.id ? `/api/tenant/skills/${skillForm.value.id}` : '/api/tenant/skills', {
      method: skillForm.value.id ? 'PUT' : 'POST',
      body: skillForm.value,
    })
    message.success(t('agentAdmin.saveSuccess'))
    skillModalVisible.value = false
    await fetchAll()
  } catch (err: any) {
    message.error(err.message || t('common.saveFailed'))
  }
}

const deleteSkill = async (id: string) => {
  try {
    await authFetch(`/api/tenant/skills/${id}`, { method: 'DELETE' })
    message.success(t('agentAdmin.deleteSuccess'))
    await fetchAll()
  } catch (err: any) {
    message.error(err.message || t('common.deleteFailed'))
  }
}

onMounted(() => {
  fetchAll()
})
</script>

<template>
  <div class="agent-admin-page">
    <div class="page-header">
      <div class="header-title">{{ t('agentAdmin.title') }}</div>
      <div class="header-subtitle">{{ t('agentAdmin.subtitle') }}</div>
    </div>

    <a-tabs v-model:activeKey="activeTab" class="admin-tabs">
      <!-- 智能体配置 -->
      <a-tab-pane key="agents" :tab="t('agentAdmin.tab.agents')">
        <div class="tab-toolbar">
          <a-button type="primary" @click="openCreateAgent">
            <template #icon><PlusOutlined /></template>
            {{ t('agentAdmin.createAgent') }}
          </a-button>
        </div>

        <a-table :dataSource="agents" :rowKey="(r: AgentDefinitionItem) => r.id" :loading="loading" :pagination="false">
          <a-table-column :title="t('agentAdmin.col.code')" dataIndex="agent_code" width="140px" />
          <a-table-column :title="t('agentAdmin.col.name')" dataIndex="name" width="160px" />
          <a-table-column :title="t('agentAdmin.col.desc')" dataIndex="description" />
          <a-table-column :title="t('agentAdmin.col.type')" dataIndex="is_system" width="100px">
            <template #default="{ text }">
              <a-tag :color="text ? 'blue' : 'green'">{{ text ? t('agentAdmin.systemBuiltin') : t('agentAdmin.tenantCustom') }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column :title="t('agentAdmin.col.tools')" dataIndex="tool_codes" width="180px">
            <template #default="{ record }">
              <span v-if="!record.tool_codes || record.tool_codes.length === 0" class="muted-text">-</span>
              <a-tag v-for="t in record.tool_codes" :key="t" color="purple">{{ t }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column :title="t('agentAdmin.col.actions')" width="150px">
            <template #default="{ record }">
              <a-space>
                <a-button type="link" size="small" @click="openEditAgent(record)">{{ t('agentAdmin.edit') }}</a-button>
                <a-popconfirm
                  v-if="!record.is_system"
                  :title="t('agentAdmin.deleteConfirm')"
                  @confirm="deleteAgent(record.id)"
                >
                  <a-button type="link" danger size="small">{{ t('agentAdmin.delete') }}</a-button>
                </a-popconfirm>
              </a-space>
            </template>
          </a-table-column>
        </a-table>
      </a-tab-pane>

      <!-- MCP 扩展服务 -->
      <a-tab-pane key="mcp" :tab="t('agentAdmin.tab.mcp')">
        <div class="tab-toolbar">
          <a-button type="primary" @click="openCreateMCP">
            <template #icon><PlusOutlined /></template>
            {{ t('agentAdmin.createMCP') }}
          </a-button>
        </div>

        <a-table :dataSource="mcpServers" :rowKey="(r: MCPServerItem) => r.id" :loading="loading" :pagination="false">
          <a-table-column :title="t('agentAdmin.col.code')" dataIndex="server_code" width="140px" />
          <a-table-column :title="t('agentAdmin.col.name')" dataIndex="name" width="160px" />
          <a-table-column :title="t('agentAdmin.col.transport')" dataIndex="transport_type" width="90px" />
          <a-table-column :title="t('agentAdmin.col.endpoint')" dataIndex="endpoint_url" />
          <a-table-column :title="t('agentAdmin.col.discoveredTools')" width="140px">
            <template #default="{ record }">
              <a-tag color="cyan">{{ record.cached_tools?.length || 0 }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column :title="t('agentAdmin.col.status')" dataIndex="enabled" width="90px">
            <template #default="{ text }">
              <a-badge :status="text ? 'success' : 'default'" :text="text ? t('agentAdmin.enabled') : t('agentAdmin.disabled')" />
            </template>
          </a-table-column>
          <a-table-column :title="t('agentAdmin.col.actions')" width="180px">
            <template #default="{ record }">
              <a-space>
                <a-button type="link" size="small" @click="mcpForm = { ...record, headers: '' }; mcpModalVisible = true">{{ t('agentAdmin.edit') }}</a-button>
                <a-button type="link" size="small" @click="testMCP(record.id)">{{ t('agentAdmin.testConnection') }}</a-button>
                <a-popconfirm :title="t('agentAdmin.deleteConfirm')" @confirm="deleteMCP(record.id)">
                  <a-button type="link" danger size="small">{{ t('agentAdmin.delete') }}</a-button>
                </a-popconfirm>
              </a-space>
            </template>
          </a-table-column>
        </a-table>
      </a-tab-pane>

      <!-- 自定义 Skills -->
      <a-tab-pane key="skills" :tab="t('agentAdmin.tab.skills')">
        <div class="tab-toolbar">
          <a-button type="primary" @click="openCreateSkill">
            <template #icon><PlusOutlined /></template>
            {{ t('agentAdmin.createSkill') }}
          </a-button>
        </div>

        <a-table :dataSource="skills" :rowKey="(r: AgentSkillItem) => r.id" :loading="loading" :pagination="false">
          <a-table-column :title="t('agentAdmin.col.code')" dataIndex="skill_code" width="160px" />
          <a-table-column :title="t('agentAdmin.col.name')" dataIndex="name" width="180px" />
          <a-table-column :title="t('agentAdmin.col.desc')" dataIndex="description" />
          <a-table-column :title="t('agentAdmin.col.source')" dataIndex="is_system" width="100px">
            <template #default="{ text }">
              <a-tag :color="text ? 'blue' : 'green'">{{ text ? t('agentAdmin.systemBuiltin') : t('agentAdmin.tenantCustom') }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column :title="t('agentAdmin.col.actions')" width="120px">
            <template #default="{ record }">
              <a-popconfirm
                v-if="!record.is_system"
                :title="t('agentAdmin.deleteConfirm')"
                @confirm="deleteSkill(record.id)"
              >
                <a-button type="link" danger size="small">{{ t('agentAdmin.delete') }}</a-button>
              </a-popconfirm>
              <a-button v-if="!record.is_system" type="link" size="small" @click="skillForm = { ...record }; skillModalVisible = true">{{ t('agentAdmin.edit') }}</a-button>
            </template>
          </a-table-column>
        </a-table>
      </a-tab-pane>
    </a-tabs>

    <!-- 智能体编辑模态框 -->
    <a-modal v-model:open="agentModalVisible" :title="t('agentAdmin.modal.agentTitle')" @ok="saveAgent">
      <a-form :model="agentForm" layout="vertical">
        <a-form-item :label="t('agentAdmin.form.code')" required>
          <a-input v-model:value="agentForm.agent_code" :placeholder="t('agentAdmin.form.codePlaceholder')" :disabled="Boolean(agentForm.id)" />
        </a-form-item>
        <a-form-item :label="t('agentAdmin.form.name')" required>
          <a-input v-model:value="agentForm.name" :placeholder="t('agentAdmin.form.namePlaceholder')" />
        </a-form-item>
        <a-form-item :label="t('agentAdmin.col.status')"><a-switch v-model:checked="agentForm.enabled" /></a-form-item>
        <a-form-item :label="t('agentAdmin.form.desc')">
          <a-input v-model:value="agentForm.description" />
        </a-form-item>
        <a-form-item :label="t('agentAdmin.form.systemPrompt')">
          <a-textarea v-model:value="agentForm.system_prompt" :rows="4" />
        </a-form-item>
        <a-form-item :label="t('agentAdmin.form.bindTools')">
          <a-checkbox-group v-model:value="agentForm.tool_codes">
            <a-row :gutter="[8, 8]">
              <a-col :span="12" v-for="tool in systemTools" :key="tool.tool_code">
                <a-checkbox :value="tool.tool_code">{{ tool.name }}</a-checkbox>
              </a-col>
            </a-row>
          </a-checkbox-group>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- MCP 注册模态框 -->
    <a-modal v-model:open="mcpModalVisible" :title="t('agentAdmin.modal.mcpTitle')" @ok="saveMCP">
      <a-form :model="mcpForm" layout="vertical">
        <a-form-item :label="t('agentAdmin.form.code')" required>
          <a-input v-model:value="mcpForm.server_code" :disabled="!!mcpForm.id" :placeholder="t('agentAdmin.form.codePlaceholder')" />
        </a-form-item>
        <a-form-item :label="t('agentAdmin.form.name')" required>
          <a-input v-model:value="mcpForm.name" :placeholder="t('agentAdmin.form.namePlaceholder')" />
        </a-form-item>
        <a-form-item :label="t('agentAdmin.form.mcpEndpoint')" required>
          <a-input v-model:value="mcpForm.endpoint_url" placeholder="https://mcp.internal.company.com/rpc" />
        </a-form-item>
        <a-form-item :label="t('agentAdmin.form.mcpHeaders')">
          <a-textarea v-model:value="mcpForm.headers" :rows="2" placeholder='{"Authorization": "Bearer key-xyz"}' />
          <p v-if="mcpForm.id" class="form-hint">{{ t('chat.mcpHeadersHint') }}</p>
        </a-form-item>
        <a-form-item :label="t('agentAdmin.col.status')"><a-switch v-model:checked="mcpForm.enabled" /></a-form-item>
      </a-form>
    </a-modal>

    <!-- Skills 模态框 -->
    <a-modal v-model:open="skillModalVisible" :title="t('agentAdmin.modal.skillTitle')" @ok="saveSkill">
      <a-form :model="skillForm" layout="vertical">
        <a-form-item :label="t('agentAdmin.form.code')" required>
          <a-input v-model:value="skillForm.skill_code" :disabled="!!skillForm.id" :placeholder="t('agentAdmin.form.codePlaceholder')" />
        </a-form-item>
        <a-form-item :label="t('agentAdmin.form.name')" required>
          <a-input v-model:value="skillForm.name" :placeholder="t('agentAdmin.form.namePlaceholder')" />
        </a-form-item>
        <a-form-item :label="t('agentAdmin.form.skillTemplate')" required>
          <a-textarea v-model:value="skillForm.content" :rows="5" />
        </a-form-item>
        <a-form-item :label="t('agentAdmin.col.status')"><a-switch v-model:checked="skillForm.enabled" /></a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.agent-admin-page {
  padding: 24px;
  background: var(--color-bg-card);
  min-height: calc(100vh - 64px);
}
.page-header {
  margin-bottom: 20px;
}
.header-title {
  font-size: 20px;
  font-weight: 600;
  color: var(--color-text-primary);
}
.header-subtitle {
  font-size: 13px;
  color: var(--color-text-secondary);
  margin-top: 4px;
}
.tab-toolbar {
  margin-bottom: 16px;
  display: flex;
  justify-content: flex-end;
}
.agent-emoji-preview {
  font-size: 20px;
}
.muted-text {
  color: #bfbfbf;
}
</style>
