<script setup lang="ts">
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  ThunderboltOutlined,
  ApiOutlined,
  BookOutlined,
  RobotOutlined,
  FullscreenOutlined,
  FullscreenExitOutlined,
  QuestionCircleOutlined,
  CheckCircleOutlined,
  FileSearchOutlined,
  UnorderedListOutlined,
  SearchOutlined,
  HourglassOutlined,
  ClockCircleOutlined,
  BarChartOutlined,
  BulbOutlined,
  FileTextOutlined,
  TeamOutlined,
  UserOutlined,
  AppstoreOutlined,
  CheckOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type { AccessControl } from '~/types/archive-config'
import type {
  AgentDefinitionItem,
  MCPServerItem,
  AgentSkillItem,
  SystemToolCatalogItem,
  SaveAgentRequest,
  SaveMCPServerRequest,
  SaveSkillRequest,
  QuickQuestionItem,
} from '~/types/chat'
import { useOrgApi } from '~/composables/useOrgApi'

definePageMeta({
  layout: 'default',
  middleware: ['auth'],
})

const { authFetch } = useAuth()
const { t } = useI18n()
const { systemPromptVariables } = usePromptSystemVariables()
const activeTab = ref('agents')
const loading = ref(false)

// 提示词输入框引用与光标位置记忆
const agentPromptTextareaRef = ref<any>(null)
const expandedPromptTextareaRef = ref<any>(null)
const skillContentTextareaRef = ref<any>(null)

const insertVariableToTextarea = (textareaRef: any, currentValRef: Ref<string>, variable: string) => {
  const el: HTMLTextAreaElement | null = textareaRef?.$el?.querySelector?.('textarea') || textareaRef?.resizableTextArea?.textArea || null
  const currentVal = currentValRef.value || ''
  if (!el) {
    currentValRef.value = currentVal ? `${currentVal} ${variable}` : variable
    return
  }
  const start = el.selectionStart ?? currentVal.length
  const end = el.selectionEnd ?? start
  currentValRef.value = currentVal.slice(0, start) + variable + currentVal.slice(end)
  nextTick(() => {
    el.focus()
    el.setSelectionRange(start + variable.length, start + variable.length)
  })
}

const insertAgentPromptVariable = (variable: string) => {
  if (promptExpanded.value) {
    insertVariableToTextarea(expandedPromptTextareaRef.value, toRef(agentForm.value, 'system_prompt'), variable)
  } else {
    insertVariableToTextarea(agentPromptTextareaRef.value, toRef(agentForm.value, 'system_prompt'), variable)
  }
}

const insertSkillPromptVariable = (variable: string) => {
  insertVariableToTextarea(skillContentTextareaRef.value, toRef(skillForm.value, 'content'), variable)
}

// 列表数据
const agents = ref<AgentDefinitionItem[]>([])
const mcpServers = ref<MCPServerItem[]>([])
const skills = ref<AgentSkillItem[]>([])
const systemTools = ref<SystemToolCatalogItem[]>([])
const mcpTools = ref<SystemToolCatalogItem[]>([])
const skillTools = ref<SystemToolCatalogItem[]>([])
const agentOptions = computed(() => agents.value.map(item => ({ label: item.name, value: item.agent_code })))

// 组织架构人员与权限支持
const { departments, roles, members, loadAll: loadOrgData } = useOrgApi()

const emptyAccessControl = (): AccessControl => ({
  allow_all: true,
  allowed_roles: [],
  allowed_members: [],
  allowed_departments: [],
})

const normalizeAccessControl = (ac?: Partial<AccessControl> | null): AccessControl => ({
  allow_all: ac?.allow_all !== false,
  allowed_roles: Array.isArray(ac?.allowed_roles) ? [...ac.allowed_roles] : [],
  allowed_members: Array.isArray(ac?.allowed_members) ? [...ac.allowed_members] : [],
  allowed_departments: Array.isArray(ac?.allowed_departments) ? [...ac.allowed_departments] : [],
})

const agentRoleSearch = ref('')
const agentMemberSearch = ref('')
const agentDeptSearch = ref('')

const filteredAgentRoles = computed(() => {
  const q = agentRoleSearch.value.toLowerCase().trim()
  if (!q) return roles.value
  return roles.value.filter(r => r.name.toLowerCase().includes(q))
})

const filteredAgentMembers = computed(() => {
  const q = agentMemberSearch.value.toLowerCase().trim()
  if (!q) return members.value
  return members.value.filter(m => m.name.toLowerCase().includes(q) || (m.department_name && m.department_name.toLowerCase().includes(q)))
})

const filteredAgentDepts = computed(() => {
  const q = agentDeptSearch.value.toLowerCase().trim()
  if (!q) return departments.value
  return departments.value.filter(d => d.name.toLowerCase().includes(q))
})

const toggleAgentRole = (roleId: string) => {
  if (!agentForm.value.access_control) agentForm.value.access_control = emptyAccessControl()
  const ac = agentForm.value.access_control
  const idx = ac.allowed_roles.indexOf(roleId)
  if (idx >= 0) ac.allowed_roles.splice(idx, 1)
  else ac.allowed_roles.push(roleId)
}

const toggleAgentMember = (memberId: string) => {
  if (!agentForm.value.access_control) agentForm.value.access_control = emptyAccessControl()
  const ac = agentForm.value.access_control
  const idx = ac.allowed_members.indexOf(memberId)
  if (idx >= 0) ac.allowed_members.splice(idx, 1)
  else ac.allowed_members.push(memberId)
}

const toggleAgentDept = (deptId: string) => {
  if (!agentForm.value.access_control) agentForm.value.access_control = emptyAccessControl()
  const ac = agentForm.value.access_control
  const idx = ac.allowed_departments.indexOf(deptId)
  if (idx >= 0) ac.allowed_departments.splice(idx, 1)
  else ac.allowed_departments.push(deptId)
}

const getAccessSummary = (ac?: AccessControl) => {
  if (!ac || ac.allow_all !== false) return t('agentAdmin.permAll')
  const rCount = ac.allowed_roles?.length || 0
  const mCount = ac.allowed_members?.length || 0
  const dCount = ac.allowed_departments?.length || 0
  return t('agentAdmin.scopeSummary', [rCount, mCount, dCount])
}

// 右侧大抽屉展开状态
const agentDrawerVisible = ref(false)
const mcpDrawerVisible = ref(false)
const skillDrawerVisible = ref(false)
const promptExpanded = ref(false)

const savingAgent = ref(false)
const savingMCP = ref(false)
const savingSkill = ref(false)

const agentForm = ref<SaveAgentRequest>({
  agent_code: '',
  name: '',
  description: '',
  system_prompt: '',
  enabled: true,
  tool_codes: [],
  quick_questions: [],
  access_control: emptyAccessControl(),
})

const mcpForm = ref<SaveMCPServerRequest>({
  server_code: '',
  name: '',
  description: '',
  transport_type: 'http',
  endpoint_url: '',
  headers: '',
  enabled: true,
  agent_codes: [],
})

const skillForm = ref<SaveSkillRequest>({
  skill_code: '',
  name: '',
  description: '',
  content: '',
  enabled: true,
  agent_codes: [],
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
      loadOrgData(),
    ])
    agents.value = agentsData || []
    mcpServers.value = mcpData || []
    skills.value = skillsData || []
    const catalogTools = catalogData?.tool_catalog || []
    systemTools.value = catalogTools.filter(item => !item.tool_code.startsWith('mcp:') && !item.tool_code.startsWith('skill:'))
    mcpTools.value = catalogTools.filter(item => item.tool_code.startsWith('mcp:'))
    skillTools.value = (catalogData?.skill_catalog || []).map(sk => ({
      tool_code: 'skill:' + sk.skill_code,
      name: sk.name,
      description: sk.description,
      ui_kind: 'skill',
    }))
  } catch (err: any) {
    message.error(err.message || t('common.loadFailed'))
  } finally {
    loading.value = false
  }
}

// 统计装配能力
const getMountedCapabilities = (toolCodes?: string[]) => {
  if (!toolCodes || !toolCodes.length) return { system: 0, mcp: 0, skill: 0 }
  let system = 0
  let mcp = 0
  let skill = 0
  for (const code of toolCodes) {
    if (code.startsWith('mcp:')) mcp++
    else if (code.startsWith('skill:')) skill++
    else system++
  }
  return { system, mcp, skill }
}

// 智能体快捷问题管理
const quickQuestionIconOptions = computed(() => [
  { value: 'UnorderedListOutlined', label: t('agentAdmin.quickIcon.todos', '待办任务 (📋)'), icon: UnorderedListOutlined, aliases: ['clipboard', 'todolist'] },
  { value: 'FileSearchOutlined', label: t('agentAdmin.quickIcon.search', '流程检索 (🔍)'), icon: FileSearchOutlined, aliases: ['search'] },
  { value: 'HourglassOutlined', label: t('agentAdmin.quickIcon.delay', '超时滞留 (⏳)'), icon: HourglassOutlined, aliases: ['hourglass', 'clock', 'ClockCircleOutlined'] },
  { value: 'EditOutlined', label: t('agentAdmin.quickIcon.draft', '起草拟定 (✍️)'), icon: EditOutlined, aliases: ['edit'] },
  { value: 'BarChartOutlined', label: t('agentAdmin.quickIcon.stats', '统计总结 (📊)'), icon: BarChartOutlined, aliases: ['barchart', 'chart'] },
  { value: 'BulbOutlined', label: t('agentAdmin.quickIcon.suggest', '灵感建议 (💡)'), icon: BulbOutlined, aliases: ['lightbulb', 'bulb'] },
  { value: 'BookOutlined', label: t('agentAdmin.quickIcon.rules', '业务规范 (📖)'), icon: BookOutlined, aliases: ['book'] },
  { value: 'ThunderboltOutlined', label: t('agentAdmin.quickIcon.action', '快捷执行 (⚡)'), icon: ThunderboltOutlined, aliases: ['thunderbolt'] },
  { value: 'QuestionCircleOutlined', label: t('agentAdmin.quickIcon.inquiry', '疑问咨询 (❓)'), icon: QuestionCircleOutlined, aliases: ['question'] },
  { value: 'CheckCircleOutlined', label: t('agentAdmin.quickIcon.review', '核验审查 (✅)'), icon: CheckCircleOutlined, aliases: ['check'] },
  { value: 'FileTextOutlined', label: t('agentAdmin.quickIcon.doc', '单据文档 (📄)'), icon: FileTextOutlined, aliases: ['file', 'document'] },
])

const normalizeQuickIcon = (rawIcon?: string): string => {
  if (!rawIcon) return 'UnorderedListOutlined'
  const matched = quickQuestionIconOptions.value.find(
    opt => opt.value === rawIcon || opt.aliases?.includes(rawIcon) || opt.aliases?.includes(rawIcon.toLowerCase())
  )
  return matched ? matched.value : rawIcon
}

const addQuickQuestion = () => {
  if (!agentForm.value.quick_questions) {
    agentForm.value.quick_questions = []
  }
  agentForm.value.quick_questions.push({
    icon: 'UnorderedListOutlined',
    title: '',
    prompt: '',
    detail: '',
  })
}

const removeQuickQuestion = (index: number) => {
  agentForm.value.quick_questions?.splice(index, 1)
}

// 智能体保存/删除
const openCreateAgent = () => {
  agentRoleSearch.value = ''
  agentMemberSearch.value = ''
  agentDeptSearch.value = ''
  agentForm.value = {
    agent_code: '',
    name: '',
    description: '',
    system_prompt: '',
    enabled: true,
    tool_codes: [],
    quick_questions: [
      { icon: 'UnorderedListOutlined', title: '待办任务', prompt: '请帮我查询当前我有哪些待办流程？', detail: '快速检阅当前待办事项' },
      { icon: 'FileSearchOutlined', title: '流程总结', prompt: '请帮我总结一下最近的审批流程', detail: '提取流程关键要点与审批流转' },
    ],
    access_control: emptyAccessControl(),
  }
  agentDrawerVisible.value = true
}

const openEditAgent = (item: AgentDefinitionItem) => {
  agentRoleSearch.value = ''
  agentMemberSearch.value = ''
  agentDeptSearch.value = ''
  agentForm.value = {
    id: item.id,
    agent_code: item.agent_code,
    name: item.name,
    description: item.description,
    system_prompt: item.system_prompt,
    enabled: item.enabled,
    tool_codes: [...(item.tool_codes || [])],
    quick_questions: item.quick_questions
      ? item.quick_questions.map((q: any) => ({
          ...q,
          icon: normalizeQuickIcon(q.icon),
        }))
      : [],
    access_control: normalizeAccessControl(item.access_control),
  }
  agentDrawerVisible.value = true
}

const saveAgent = async () => {
  if (!agentForm.value.agent_code || !agentForm.value.name) {
    message.warning(t('agentAdmin.form.required'))
    return
  }
  savingAgent.value = true
  try {
    await authFetch(agentForm.value.id ? `/api/tenant/agents/${agentForm.value.id}` : '/api/tenant/agents', {
      method: agentForm.value.id ? 'PUT' : 'POST',
      body: agentForm.value,
    })
    message.success(t('agentAdmin.saveSuccess'))
    agentDrawerVisible.value = false
    await fetchAll()
  } catch (err: any) {
    message.error(err.message || t('common.saveFailed'))
  } finally {
    savingAgent.value = false
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
    agent_codes: [],
  }
  mcpDrawerVisible.value = true
}

const openEditMCP = (item: MCPServerItem) => {
  mcpForm.value = {
    ...item,
    headers: '',
    agent_codes: [...(item.agent_codes || [])],
  }
  mcpDrawerVisible.value = true
}

const saveMCP = async () => {
  if (!mcpForm.value.server_code || !mcpForm.value.endpoint_url) {
    message.warning(t('agentAdmin.form.required'))
    return
  }
  savingMCP.value = true
  try {
    await authFetch(mcpForm.value.id ? `/api/tenant/mcp-servers/${mcpForm.value.id}` : '/api/tenant/mcp-servers', {
      method: mcpForm.value.id ? 'PUT' : 'POST',
      body: mcpForm.value,
    })
    message.success(t('agentAdmin.saveSuccess'))
    mcpDrawerVisible.value = false
    await fetchAll()
  } catch (err: any) {
    message.error(err.message || t('common.saveFailed'))
  } finally {
    savingMCP.value = false
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
    content: '',
    enabled: true,
    agent_codes: [],
  }
  skillDrawerVisible.value = true
}

const openEditSkill = (item: AgentSkillItem) => {
  skillForm.value = {
    ...item,
    agent_codes: [...(item.agent_codes || [])],
  }
  skillDrawerVisible.value = true
}

const saveSkill = async () => {
  if (!skillForm.value.skill_code || !skillForm.value.content) {
    message.warning(t('agentAdmin.form.required'))
    return
  }
  savingSkill.value = true
  try {
    await authFetch(skillForm.value.id ? `/api/tenant/skills/${skillForm.value.id}` : '/api/tenant/skills', {
      method: skillForm.value.id ? 'PUT' : 'POST',
      body: skillForm.value,
    })
    message.success(t('agentAdmin.saveSuccess'))
    skillDrawerVisible.value = false
    await fetchAll()
  } catch (err: any) {
    message.error(err.message || t('common.saveFailed'))
  } finally {
    savingSkill.value = false
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
          <a-table-column :title="t('agentAdmin.col.code')" dataIndex="agent_code" width="130px" />
          <a-table-column :title="t('agentAdmin.col.name')" dataIndex="name" width="160px">
            <template #default="{ record }">
              <span style="font-weight: 500; display: inline-flex; align-items: center; gap: 6px;">
                <RobotOutlined style="color: var(--color-primary);" />
                {{ record.name }}
              </span>
            </template>
          </a-table-column>
          <a-table-column :title="t('agentAdmin.col.desc')" dataIndex="description" width="220px">
            <template #default="{ text }">
              <span class="desc-cell" :title="text">{{ text || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column :title="t('agentAdmin.col.type')" dataIndex="is_system" width="90px">
            <template #default="{ text }">
              <a-tag :color="text ? 'blue' : 'green'">{{ text ? t('agentAdmin.systemBuiltin') : t('agentAdmin.tenantCustom') }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column :title="t('agentAdmin.col.mountedCaps', '装配能力概览')" width="260px">
            <template #default="{ record }">
              <div style="display: flex; flex-wrap: wrap; gap: 4px;">
                <a-tag v-if="getMountedCapabilities(record.tool_codes).system > 0" color="blue">
                  <ThunderboltOutlined /> 系统工具 {{ getMountedCapabilities(record.tool_codes).system }}
                </a-tag>
                <a-tag v-if="getMountedCapabilities(record.tool_codes).mcp > 0" color="cyan">
                  <ApiOutlined /> MCP {{ getMountedCapabilities(record.tool_codes).mcp }}
                </a-tag>
                <a-tag v-if="getMountedCapabilities(record.tool_codes).skill > 0" color="purple">
                  <BookOutlined /> 技能 {{ getMountedCapabilities(record.tool_codes).skill }}
                </a-tag>
                <span v-if="!record.tool_codes?.length" class="muted-text">-</span>
              </div>
            </template>
          </a-table-column>
          <a-table-column title="快捷问题" width="90px">
            <template #default="{ record }">
              <a-tag color="geekblue">{{ record.quick_questions?.length || 0 }} 条</a-tag>
            </template>
          </a-table-column>
          <a-table-column :title="t('agentAdmin.col.permissions', '使用权限')" width="110px">
            <template #default="{ record }">
              <a-tag v-if="record.access_control?.allow_all !== false" color="green">
                {{ t('agentAdmin.permAll', '全员可用') }}
              </a-tag>
              <a-tooltip v-else :title="getAccessSummary(record.access_control)">
                <a-tag color="blue" style="cursor: pointer;">
                  {{ t('agentAdmin.permCustom', '指定范围') }}
                </a-tag>
              </a-tooltip>
            </template>
          </a-table-column>
          <a-table-column :title="t('agentAdmin.col.status')" dataIndex="enabled" width="80px">
            <template #default="{ text }">
              <a-badge :status="text ? 'success' : 'default'" :text="text ? t('agentAdmin.enabled') : t('agentAdmin.disabled')" />
            </template>
          </a-table-column>
          <a-table-column :title="t('agentAdmin.col.actions')" width="140px">
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
          <a-table-column :title="t('agentAdmin.col.endpoint')" dataIndex="endpoint_url" width="220px">
            <template #default="{ text }">
              <span class="desc-cell" :title="text">{{ text || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column :title="t('agentAdmin.col.mountAgents')" width="180px">
            <template #default="{ record }">
              <span v-if="!record.agent_codes?.length" class="muted-text">-</span>
              <a-tag v-for="code in record.agent_codes" :key="code">{{ agents.find(item => item.agent_code === code)?.name || code }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column :title="t('agentAdmin.col.discoveredTools')" width="140px">
            <template #default="{ record }">
              <a-tag color="cyan">{{ record.cached_tools?.length || 0 }} 个工具</a-tag>
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
                <a-button type="link" size="small" @click="openEditMCP(record)">{{ t('agentAdmin.edit') }}</a-button>
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
          <a-table-column :title="t('agentAdmin.col.desc')" dataIndex="description" width="220px">
            <template #default="{ text }">
              <span class="desc-cell" :title="text">{{ text || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column :title="t('agentAdmin.col.mountAgents')" width="180px">
            <template #default="{ record }">
              <span v-if="!record.agent_codes?.length" class="muted-text">-</span>
              <a-tag v-for="code in record.agent_codes" :key="code">{{ agents.find(item => item.agent_code === code)?.name || code }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column :title="t('agentAdmin.col.source')" dataIndex="is_system" width="100px">
            <template #default="{ text }">
              <a-tag :color="text ? 'blue' : 'green'">{{ text ? t('agentAdmin.systemBuiltin') : t('agentAdmin.tenantCustom') }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column :title="t('agentAdmin.col.actions')" width="130px">
            <template #default="{ record }">
              <a-space>
                <a-button v-if="!record.is_system" type="link" size="small" @click="openEditSkill(record)">{{ t('agentAdmin.edit') }}</a-button>
                <a-popconfirm
                  v-if="!record.is_system"
                  :title="t('agentAdmin.deleteConfirm')"
                  @confirm="deleteSkill(record.id)"
                >
                  <a-button type="link" danger size="small">{{ t('agentAdmin.delete') }}</a-button>
                </a-popconfirm>
              </a-space>
            </template>
          </a-table-column>
        </a-table>
      </a-tab-pane>
    </a-tabs>

    <!-- 1. 智能体编辑大抽屉 -->
    <a-drawer
      v-model:open="agentDrawerVisible"
      :title="agentForm.id ? t('agentAdmin.editAgent', '编辑智能体') : t('agentAdmin.createAgent', '新建智能体')"
      :width="780"
      placement="right"
    >
      <a-form :model="agentForm" layout="vertical" class="drawer-form">
        <!-- 基础信息卡片 -->
        <div class="drawer-section">
          <div class="drawer-section-title">基础配置</div>
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item :label="t('agentAdmin.form.code')" required>
                <a-input
                  v-model:value="agentForm.agent_code"
                  :placeholder="t('agentAdmin.form.codePlaceholder')"
                  :disabled="Boolean(agentForm.id)"
                />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item :label="t('agentAdmin.form.name')" required>
                <a-input
                  v-model:value="agentForm.name"
                  :placeholder="t('agentAdmin.form.namePlaceholder')"
                />
              </a-form-item>
            </a-col>
          </a-row>

          <a-row :gutter="16">
            <a-col :span="18">
              <a-form-item :label="t('agentAdmin.form.desc')">
                <a-input v-model:value="agentForm.description" placeholder="请输入智能体的主要职能与服务范畴" />
              </a-form-item>
            </a-col>
            <a-col :span="6">
              <a-form-item :label="t('agentAdmin.col.status')">
                <a-switch v-model:checked="agentForm.enabled" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <!-- 系统提示词配置卡片（支持常用系统变量与放大编辑） -->
        <div class="drawer-section">
          <div class="drawer-section-title" style="display: flex; align-items: center; justify-content: space-between;">
            <span>{{ t('agentAdmin.form.systemPrompt') }}</span>
            <a-button size="small" type="link" @click="promptExpanded = true">
              <FullscreenOutlined /> {{ t('agentAdmin.promptExpand', '放大全屏编辑') }}
            </a-button>
          </div>
          <PromptVariableBar
            :data-variables="[]"
            :system-variables="systemPromptVariables"
            @insert="insertAgentPromptVariable"
          />
          <a-form-item style="margin-bottom: 0;">
            <a-textarea
              ref="agentPromptTextareaRef"
              v-model:value="agentForm.system_prompt"
              :rows="5"
              placeholder="设定该智能体的业务角色定位、思考链规范、工具使用准则及边界约束..."
              style="font-family: var(--font-mono); font-size: 13px;"
            />
          </a-form-item>
        </div>

        <!-- 能力装配选装（直接在智能体中选装 MCP 与 Skills） -->
        <div class="drawer-section">
          <div class="drawer-section-title">{{ t('agentAdmin.capabilities', '能力装配（系统工具 / MCP 服务 / 自定义技能）') }}</div>
          <p class="drawer-section-desc">勾选装配给该智能体在对话中可以调用的扩展能力：</p>

          <a-checkbox-group v-model:value="agentForm.tool_codes" style="width: 100%;">
            <!-- 系统内置工具 -->
            <div class="caps-group">
              <div class="caps-group-header">
                <ThunderboltOutlined style="color: var(--color-primary);" />
                <span>{{ t('agentAdmin.systemToolsGroup', '系统内置工具') }} ({{ systemTools.length }})</span>
              </div>
              <div class="caps-grid">
                <div v-for="tool in systemTools" :key="tool.tool_code" class="cap-card">
                  <a-checkbox :value="tool.tool_code">
                    <span class="cap-name">{{ tool.name }}</span>
                  </a-checkbox>
                  <div class="cap-desc">{{ tool.description || tool.tool_code }}</div>
                </div>
              </div>
            </div>

            <!-- MCP 服务扩展工具 -->
            <div v-if="mcpTools.length" class="caps-group" style="margin-top: 14px;">
              <div class="caps-group-header">
                <ApiOutlined style="color: var(--color-warning);" />
                <span>{{ t('agentAdmin.mcpToolsGroup', 'MCP 服务扩展工具') }} ({{ mcpTools.length }})</span>
              </div>
              <div class="caps-grid">
                <div v-for="tool in mcpTools" :key="tool.tool_code" class="cap-card">
                  <a-checkbox :value="tool.tool_code">
                    <span class="cap-name">{{ tool.name }}</span>
                  </a-checkbox>
                  <div class="cap-desc">{{ tool.description || tool.tool_code }}</div>
                </div>
              </div>
            </div>

            <!-- 自定义技能指南 (Skills) -->
            <div v-if="skillTools.length" class="caps-group" style="margin-top: 14px;">
              <div class="caps-group-header">
                <BookOutlined style="color: var(--color-success);" />
                <span>{{ t('agentAdmin.skillToolsGroup', '提示词技能指南 (Skills)') }} ({{ skillTools.length }})</span>
              </div>
              <div class="caps-grid">
                <div v-for="tool in skillTools" :key="tool.tool_code" class="cap-card">
                  <a-checkbox :value="tool.tool_code">
                    <span class="cap-name">{{ tool.name }}</span>
                  </a-checkbox>
                  <div class="cap-desc">{{ tool.description || tool.tool_code }}</div>
                </div>
              </div>
            </div>
          </a-checkbox-group>
        </div>

        <!-- 快捷提问列表配置 -->
        <div class="drawer-section">
          <div class="drawer-section-title" style="display: flex; align-items: center; justify-content: space-between;">
            <div>
              <span>{{ t('agentAdmin.quickQuestionsTitle', '快捷输入问题配置') }}</span>
              <span style="font-size: 12px; font-weight: normal; color: var(--color-text-secondary); margin-left: 8px;">
                (为空白对话页提供定制化提问引导卡片)
              </span>
            </div>
            <a-button size="small" type="dashed" @click="addQuickQuestion">
              <PlusOutlined /> {{ t('agentAdmin.addQuestion', '添加快捷提问') }}
            </a-button>
          </div>

          <div v-if="!agentForm.quick_questions?.length" class="empty-questions">
            {{ t('agentAdmin.noQuickQuestions', '暂未配置专属快捷问题，将使用通用默认引导卡片') }}
          </div>

          <div v-else class="questions-list">
            <div v-for="(q, qIdx) in agentForm.quick_questions" :key="qIdx" class="question-edit-card">
              <div class="question-header">
                <span class="q-badge">#{{ qIdx + 1 }}</span>
                <a-input v-model:value="q.title" placeholder="卡片标题 (如：待办查询)" style="flex: 1; margin: 0 8px;" />
                <a-select
                  v-model:value="q.icon"
                  style="width: 170px; margin-right: 8px;"
                  :placeholder="t('agentAdmin.selectIcon', '选择图标')"
                  :options="quickQuestionIconOptions"
                >
                  <template #option="{ label, icon }">
                    <div style="display: flex; align-items: center; gap: 8px;">
                      <component :is="icon" style="color: var(--color-primary); font-size: 14px;" />
                      <span>{{ label }}</span>
                    </div>
                  </template>
                </a-select>
                <a-button type="text" danger size="small" @click="removeQuickQuestion(qIdx)">
                  <DeleteOutlined />
                </a-button>
              </div>
              <div class="question-inputs">
                <a-textarea
                  v-model:value="q.prompt"
                  :rows="2"
                  placeholder="点击卡片后自动发送或填入对话框的完整提示词 (Prompt)..."
                  style="margin-bottom: 8px; font-size: 13px;"
                />
                <a-input v-model:value="q.detail" placeholder="卡片下方副说明 (Detail，选填)..." />
              </div>
            </div>
          </div>
        </div>

        <!-- 人员权限配置卡片 -->
        <div class="drawer-section">
          <div class="drawer-section-title">
            <span>{{ t('agentAdmin.accessControlTitle', '人员权限') }}</span>
          </div>
          <p class="drawer-section-desc">
            {{ t('agentAdmin.accessControlDesc', '配置当前智能体的可用人员范围，默认全员均可使用；关闭全员后按角色、成员、部门任意一项命中判断放行') }}
          </p>

          <div class="access-control-section">
            <div class="access-control-everyone">
              <div class="permission-info">
                <div class="permission-label"><TeamOutlined /> {{ t('admin.ruleConfig.auditAllowAll', '所有人可访问') }}</div>
                <div class="permission-desc">{{ t('agentAdmin.allowAllDesc', '开启后，当前租户内所有具备对话权限的成员都可以使用此智能体') }}</div>
              </div>
              <a-switch
                v-model:checked="agentForm.access_control!.allow_all"
                :checked-children="t('admin.ruleConfig.allow', '允许')"
                :un-checked-children="t('admin.ruleConfig.deny', '禁止')"
              />
            </div>

            <!-- 指定范围：角色、成员、部门 -->
            <div v-if="!agentForm.access_control?.allow_all" class="access-control-group">
              <div class="access-control-label"><TeamOutlined /> {{ t('admin.ruleConfig.auditAllowedRoles', '允许的角色') }}</div>
              <div class="access-control-search">
                <a-input v-model:value="agentRoleSearch" :placeholder="t('admin.ruleConfig.auditAccessSearch', '搜索...')" allow-clear size="small" style="max-width: 280px;">
                  <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
                </a-input>
              </div>
              <div class="access-control-tags" style="gap: 8px;">
                <div
                  v-for="role in filteredAgentRoles"
                  :key="role.id"
                  class="access-tag"
                  :class="{ 'access-tag--active': (agentForm.access_control?.allowed_roles || []).includes(role.id) }"
                  @click="toggleAgentRole(role.id)"
                >
                  <CheckOutlined v-if="(agentForm.access_control?.allowed_roles || []).includes(role.id)" class="access-tag-check" />
                  {{ role.name }}
                </div>
              </div>
            </div>

            <div v-if="!agentForm.access_control?.allow_all" class="access-control-group" style="margin-top: 16px;">
              <div class="access-control-label"><UserOutlined /> {{ t('admin.ruleConfig.auditAllowedMembers', '允许的成员') }}</div>
              <div class="access-control-search">
                <a-input v-model:value="agentMemberSearch" :placeholder="t('admin.ruleConfig.auditAccessSearch', '搜索...')" allow-clear size="small" style="max-width: 280px;">
                  <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
                </a-input>
              </div>
              <div class="access-control-tags" style="gap: 8px;">
                <div
                  v-for="member in filteredAgentMembers"
                  :key="member.id"
                  class="access-tag"
                  :class="{ 'access-tag--active': (agentForm.access_control?.allowed_members || []).includes(member.id) }"
                  @click="toggleAgentMember(member.id)"
                >
                  <CheckOutlined v-if="(agentForm.access_control?.allowed_members || []).includes(member.id)" class="access-tag-check" />
                  {{ member.name }}
                  <span v-if="member.department_name" class="access-tag-dept">{{ member.department_name }}</span>
                </div>
              </div>
            </div>

            <div v-if="!agentForm.access_control?.allow_all" class="access-control-group" style="margin-top: 16px;">
              <div class="access-control-label"><AppstoreOutlined /> {{ t('admin.ruleConfig.auditAllowedDepts', '允许的部门') }}</div>
              <div class="access-control-search">
                <a-input v-model:value="agentDeptSearch" :placeholder="t('admin.ruleConfig.auditAccessSearch', '搜索...')" allow-clear size="small" style="max-width: 280px;">
                  <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
                </a-input>
              </div>
              <div class="access-control-tags" style="gap: 8px;">
                <div
                  v-for="dept in filteredAgentDepts"
                  :key="dept.id"
                  class="access-tag"
                  :class="{ 'access-tag--active': (agentForm.access_control?.allowed_departments || []).includes(dept.id) }"
                  @click="toggleAgentDept(dept.id)"
                >
                  <CheckOutlined v-if="(agentForm.access_control?.allowed_departments || []).includes(dept.id)" class="access-tag-check" />
                  {{ dept.name }}
                  <span v-if="dept.member_count !== undefined" class="access-tag-dept">{{ dept.member_count }}人</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </a-form>

      <template #footer>
        <div class="drawer-footer">
          <a-button @click="agentDrawerVisible = false">{{ t('common.cancel') }}</a-button>
          <a-button type="primary" :loading="savingAgent" @click="saveAgent">{{ t('common.save') }}</a-button>
        </div>
      </template>
    </a-drawer>

    <!-- 2. MCP 注册与编辑大抽屉 -->
    <a-drawer
      v-model:open="mcpDrawerVisible"
      :title="mcpForm.id ? t('agentAdmin.editMCP', '编辑 MCP 服务') : t('agentAdmin.createMCP', '注册 MCP 服务')"
      :width="680"
      placement="right"
    >
      <a-form :model="mcpForm" layout="vertical" class="drawer-form">
        <div class="drawer-section">
          <div class="drawer-section-title">连接配置</div>
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item :label="t('agentAdmin.form.code')" required>
                <a-input
                  v-model:value="mcpForm.server_code"
                  :disabled="!!mcpForm.id"
                  :placeholder="t('agentAdmin.form.codePlaceholder')"
                />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item :label="t('agentAdmin.form.name')" required>
                <a-input
                  v-model:value="mcpForm.name"
                  :placeholder="t('agentAdmin.form.namePlaceholder')"
                />
              </a-form-item>
            </a-col>
          </a-row>

          <a-form-item :label="t('agentAdmin.form.mcpEndpoint')" required>
            <a-input v-model:value="mcpForm.endpoint_url" placeholder="https://mcp.internal.company.com/rpc" />
          </a-form-item>

          <a-form-item :label="t('agentAdmin.form.mcpHeaders')">
            <a-textarea v-model:value="mcpForm.headers" :rows="2" placeholder='{"Authorization": "Bearer key-xyz"}' />
            <p v-if="mcpForm.id" class="form-hint">{{ t('chat.mcpHeadersHint') }}</p>
          </a-form-item>

          <a-row :gutter="16">
            <a-col :span="16">
              <a-form-item :label="t('agentAdmin.form.desc')">
                <a-input v-model:value="mcpForm.description" placeholder="简要说明该 MCP 服务提供的业务能力" />
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item :label="t('agentAdmin.col.status')">
                <a-switch v-model:checked="mcpForm.enabled" />
              </a-form-item>
            </a-col>
          </a-row>

          <a-form-item :label="t('agentAdmin.form.mountAgents')">
            <a-select
              v-model:value="mcpForm.agent_codes"
              mode="multiple"
              :options="agentOptions"
              :placeholder="t('agentAdmin.form.mountAgentsPlaceholder')"
            />
          </a-form-item>
        </div>
      </a-form>

      <template #footer>
        <div class="drawer-footer">
          <a-button @click="mcpDrawerVisible = false">{{ t('common.cancel') }}</a-button>
          <a-button type="primary" :loading="savingMCP" @click="saveMCP">{{ t('common.save') }}</a-button>
        </div>
      </template>
    </a-drawer>

    <!-- 3. Skills 编辑大抽屉 -->
    <a-drawer
      v-model:open="skillDrawerVisible"
      :title="skillForm.id ? t('agentAdmin.editSkill', '编辑技能') : t('agentAdmin.createSkill', '新建技能')"
      :width="680"
      placement="right"
    >
      <a-form :model="skillForm" layout="vertical" class="drawer-form">
        <div class="drawer-section">
          <div class="drawer-section-title">技能基础</div>
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item :label="t('agentAdmin.form.code')" required>
                <a-input
                  v-model:value="skillForm.skill_code"
                  :disabled="!!skillForm.id"
                  :placeholder="t('agentAdmin.form.codePlaceholder')"
                />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item :label="t('agentAdmin.form.name')" required>
                <a-input
                  v-model:value="skillForm.name"
                  :placeholder="t('agentAdmin.form.namePlaceholder')"
                />
              </a-form-item>
            </a-col>
          </a-row>

          <a-row :gutter="16">
            <a-col :span="18">
              <a-form-item :label="t('agentAdmin.form.desc')">
                <a-input v-model:value="skillForm.description" placeholder="简要说明该技能适用的业务场景" />
              </a-form-item>
            </a-col>
            <a-col :span="6">
              <a-form-item :label="t('agentAdmin.col.status')">
                <a-switch v-model:checked="skillForm.enabled" />
              </a-form-item>
            </a-col>
          </a-row>

          <PromptVariableBar
            :data-variables="[]"
            :system-variables="systemPromptVariables"
            @insert="insertSkillPromptVariable"
          />
          <a-form-item :label="t('agentAdmin.form.skillTemplate')" required>
            <a-textarea
              ref="skillContentTextareaRef"
              v-model:value="skillForm.content"
              :rows="8"
              placeholder="编写具体的技能操作指南或专项分析规范提示词..."
              style="font-family: var(--font-mono); font-size: 13px;"
            />
          </a-form-item>

          <a-form-item :label="t('agentAdmin.form.mountAgents')">
            <a-select
              v-model:value="skillForm.agent_codes"
              mode="multiple"
              :options="agentOptions"
              :placeholder="t('agentAdmin.form.mountAgentsPlaceholder')"
            />
          </a-form-item>
        </div>
      </a-form>

      <template #footer>
        <div class="drawer-footer">
          <a-button @click="skillDrawerVisible = false">{{ t('common.cancel') }}</a-button>
          <a-button type="primary" :loading="savingSkill" @click="saveSkill">{{ t('common.save') }}</a-button>
        </div>
      </template>
    </a-drawer>

    <!-- 4. 提示词放大/全屏编辑模态框 -->
    <a-modal
      v-model:open="promptExpanded"
      :title="t('agentAdmin.promptFullscreenTitle', '系统提示词全屏编辑')"
      width="80vw"
      :bodyStyle="{ height: '62vh', padding: '16px', display: 'flex', flexDirection: 'column', gap: '8px' }"
      :footer="null"
    >
      <PromptVariableBar
        :data-variables="[]"
        :system-variables="systemPromptVariables"
        @insert="insertAgentPromptVariable"
      />
      <a-textarea
        ref="expandedPromptTextareaRef"
        v-model:value="agentForm.system_prompt"
        style="width: 100%; flex: 1; resize: none; font-family: var(--font-mono); font-size: 14px; line-height: 1.6;"
        placeholder="设定该智能体的业务角色定位、思考链规范、工具使用准则及边界约束..."
      />
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
.muted-text {
  color: #bfbfbf;
}

/* 抽屉内部结构 */
.drawer-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.drawer-section {
  background: var(--color-bg-page);
  border: 1px solid var(--color-border-light);
  border-radius: 8px;
  padding: 16px 18px;
}
.drawer-section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 12px;
}
.drawer-section-desc {
  font-size: 12px;
  color: var(--color-text-secondary);
  margin-bottom: 12px;
}
.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

/* 能力装配 */
.caps-group {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  border-radius: 6px;
  padding: 12px 14px;
}
.caps-group-header {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 10px;
}
.drawer-form {
  overflow-x: hidden;
}
.caps-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 8px;
}
.cap-card {
  background: var(--color-bg-page);
  border: 1px solid var(--color-border-light);
  border-radius: 6px;
  padding: 8px 12px;
  min-width: 0;
  overflow-wrap: anywhere;
}
.cap-name {
  font-weight: 500;
  font-size: 13px;
}
.cap-desc {
  font-size: 11px;
  color: var(--color-text-secondary);
  margin-top: 2px;
  margin-left: 24px;
}

.desc-cell {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  text-overflow: ellipsis;
  word-break: break-word;
  line-height: 1.4;
  color: var(--color-text-secondary);
}

/* 快捷提问配置 */
.empty-questions {
  padding: 24px;
  text-align: center;
  font-size: 13px;
  color: var(--color-text-tertiary);
  background: var(--color-bg-card);
  border: 1px dashed var(--color-border-light);
  border-radius: 6px;
}
.questions-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.question-edit-card {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  border-radius: 6px;
  padding: 12px 14px;
}
.question-header {
  display: flex;
  align-items: center;
  margin-bottom: 10px;
}
.q-badge {
  font-family: var(--font-mono);
  font-size: 12px;
  font-weight: 700;
  color: var(--color-primary);
  background: var(--color-primary-bg);
  padding: 2px 8px;
  border-radius: 4px;
}
.form-hint {
  font-size: 12px;
  color: var(--color-text-tertiary);
  margin-top: 4px;
}

/* 访问控制与人员权限 */
.access-control-section { display: flex; flex-direction: column; gap: 0; }
.access-control-everyone {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  border-radius: 6px;
  margin-bottom: 16px;
}
.access-control-group {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  border-radius: 6px;
  padding: 12px 14px;
}
.access-control-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-secondary);
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 10px;
}
.access-control-search { margin-bottom: 8px; }
.access-control-tags { display: flex; flex-wrap: wrap; gap: 8px; }
.access-tag {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 10px;
  border-radius: var(--radius-full);
  border: 1px solid var(--color-border-light);
  background: var(--color-bg-hover);
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
}
.access-tag:hover { border-color: var(--color-primary-lighter); color: var(--color-primary); }
.access-tag--active { border-color: var(--color-primary); background: var(--color-primary-bg); color: var(--color-primary); }
.access-tag-check { font-size: 10px; }
.access-tag-dept {
  font-size: 10px;
  color: var(--color-text-tertiary);
  margin-left: 2px;
  padding-left: 6px;
  border-left: 1px solid var(--color-border-light);
}
.permission-info { display: flex; flex-direction: column; gap: 2px; }
.permission-label { font-size: 13px; font-weight: 600; color: var(--color-text-primary); display: flex; align-items: center; gap: 6px; }
.permission-desc { font-size: 12px; color: var(--color-text-tertiary); }
</style>
