<script setup lang="ts">
definePageMeta({ middleware: 'auth', layout: 'default' })

import {useI18n} from '~/composables/useI18n'
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  CloudServerOutlined,
  DatabaseOutlined,
  DeleteOutlined,
  EditOutlined,
  GlobalOutlined,
  KeyOutlined,
  MailOutlined,
  PlusOutlined,
  RobotOutlined,
  SafetyCertificateOutlined,
  SaveOutlined,
  SettingOutlined,
  SyncOutlined,
  ToolOutlined,
  UserOutlined,
} from '@ant-design/icons-vue'
import {message} from 'ant-design-vue'
import type {AIModelConfig, OADatabaseConnection, OASystemConfig, SystemGeneralConfig} from '~/composables/useMockData'

const { t } = useI18n()

const { mockOASystemConfigs, mockOADatabaseConnections, mockAIModelConfigs, mockSystemGeneralConfig } = useMockData()

const activeTab = ref('oa')
const oaSystems = ref<OASystemConfig[]>([...mockOASystemConfigs])
const oaDbConnections = ref<OADatabaseConnection[]>(JSON.parse(JSON.stringify(mockOADatabaseConnections)))
const aiModels = ref<AIModelConfig[]>(JSON.parse(JSON.stringify(mockAIModelConfigs)))
const generalConfig = ref<SystemGeneralConfig>({ ...mockSystemGeneralConfig })
const saving = ref(false)

// ===== OA Database Connection CRUD =====
const showAddOADb = ref(false)
const editingOADb = ref<OADatabaseConnection | null>(null)
const testingOADbId = ref<string | null>(null)

const oaTypeOptions = [
  { value: 'weaver_e9', label: '泛微 Ecology E9' },
  { value: 'weaver_ebridge', label: '泛微 E-Bridge' },
  { value: 'zhiyuan_a8', label: '致远互联 A8+' },
  { value: 'landray_ekp', label: '蓝凌 EKP' },
  { value: 'custom', label: '自定义' },
]

const driverOptions = [
  { label: 'MySQL', value: 'mysql' },
  { label: 'PostgreSQL', value: 'postgresql' },
  { label: 'Oracle', value: 'oracle' },
  { label: 'SQL Server', value: 'sqlserver' },
]

const getDriverPort = (driver: string) => {
  const ports: Record<string, number> = { mysql: 3306, postgresql: 5432, oracle: 1521, sqlserver: 1433 }
  return ports[driver] || 3306
}

const newOADb = ref<Partial<OADatabaseConnection>>({
  name: '',
  oa_type: 'weaver_e9',
  oa_type_label: '泛微 Ecology E9',
  description: '',
  sync_interval: 60,
  jdbc_config: {
    driver: 'mysql', host: '', port: 3306, database: '',
    username: '', password: '', pool_size: 10,
    connection_timeout: 30, test_on_borrow: true,
  },
})

const resetNewOADb = () => {
  newOADb.value = {
    name: '', oa_type: 'weaver_e9', oa_type_label: '泛微 Ecology E9', description: '', sync_interval: 60,
    jdbc_config: { driver: 'mysql', host: '', port: 3306, database: '', username: '', password: '', pool_size: 10, connection_timeout: 30, test_on_borrow: true },
  }
}

const openAddOADb = () => {
  editingOADb.value = null
  resetNewOADb()
  showAddOADb.value = true
}

const openEditOADb = (conn: OADatabaseConnection) => {
  editingOADb.value = conn
  newOADb.value = JSON.parse(JSON.stringify(conn))
  showAddOADb.value = true
}

const onOATypeChange = (val: string) => {
  const opt = oaTypeOptions.find(o => o.value === val)
  if (opt) newOADb.value.oa_type_label = opt.label
}

const onDriverChange = (driver: any) => {
  if (newOADb.value.jdbc_config) {
    newOADb.value.jdbc_config.port = getDriverPort(driver as string)
  }
}

const saveOADb = () => {
  if (!newOADb.value.name?.trim()) {
    message.warning(t('admin.settings.oaDbNameRequired'))
    return
  }
  if (!newOADb.value.jdbc_config?.host?.trim()) {
    message.warning(t('admin.settings.oaDbHostRequired'))
    return
  }
  if (editingOADb.value) {
    const idx = oaDbConnections.value.findIndex(c => c.id === editingOADb.value!.id)
    if (idx >= 0) {
      oaDbConnections.value[idx] = { ...oaDbConnections.value[idx], ...newOADb.value } as OADatabaseConnection
    }
    message.success(t('admin.settings.oaDbUpdated'))
  } else {
    const newConn: OADatabaseConnection = {
      id: `OADB-${Date.now()}`,
      name: newOADb.value.name!,
      oa_type: newOADb.value.oa_type as any,
      oa_type_label: newOADb.value.oa_type_label!,
      jdbc_config: { ...newOADb.value.jdbc_config! } as any,
      status: 'disconnected',
      last_sync: '',
      sync_interval: newOADb.value.sync_interval || 60,
      enabled: true,
      created_at: new Date().toISOString().slice(0, 10),
      description: newOADb.value.description || '',
    }
    oaDbConnections.value.push(newConn)
    message.success(t('admin.settings.oaDbAdded'))
  }
  showAddOADb.value = false
}

const deleteOADb = (id: string) => {
  oaDbConnections.value = oaDbConnections.value.filter(c => c.id !== id)
  message.success(t('admin.settings.oaDbDeleted'))
}

const toggleOADb = (id: string) => {
  const conn = oaDbConnections.value.find(c => c.id === id)
  if (conn) {
    conn.enabled = !conn.enabled
    message.success(conn.enabled ? t('admin.settings.enabled', conn.name) : t('admin.settings.disabled', conn.name))
  }
}

const testOADbConnection = async (id: string) => {
  const conn = oaDbConnections.value.find(c => c.id === id)
  if (!conn) return
  testingOADbId.value = id
  conn.status = 'testing'
  await new Promise(resolve => setTimeout(resolve, 2000))
  if (conn.enabled && conn.jdbc_config.host) {
    conn.status = 'connected'
    conn.last_sync = new Date().toLocaleString('zh-CN')
    message.success(t('admin.settings.connSuccess', conn.name))
  } else {
    conn.status = 'disconnected'
    message.warning(t('admin.settings.notEnabled', conn.name))
  }
  testingOADbId.value = null
}

// ===== AI Model CRUD =====
const showAddAIModel = ref(false)
const editingAIModel = ref<AIModelConfig | null>(null)

const newAIModel = ref<Partial<AIModelConfig>>({
  provider: t('admin.ruleConfig.localDeploy'), model_name: '', display_name: '', type: 'local',
  endpoint: '', api_key_configured: false, max_tokens: 4096, context_window: 65536,
  cost_per_1k_tokens: 0, status: 'offline', enabled: true, description: '',
  capabilities: ['text'],
})
const resetNewAIModel = () => {
  newAIModel.value = {
    provider: t('admin.ruleConfig.localDeploy'), model_name: '', display_name: '', type: 'local',
    endpoint: '', api_key_configured: false, max_tokens: 4096, context_window: 65536,
    cost_per_1k_tokens: 0, status: 'offline', enabled: true, description: '',
    capabilities: ['text'],
  }
}

const openAddAIModel = () => {
  editingAIModel.value = null
  resetNewAIModel()
  showAddAIModel.value = true
}

const onModelTypeChange = (val: string) => {
  if (val === 'local') {
    newAIModel.value.provider = t('admin.ruleConfig.localDeploy')
    newAIModel.value.cost_per_1k_tokens = 0
    newAIModel.value.api_key_configured = false
  } else {
    newAIModel.value.provider = t('admin.ruleConfig.cloudAPI')
  }
}

const capabilityOptions = computed(() => [
  { value: 'text', label: t('admin.settings.capability.text') },
  { value: 'code', label: t('admin.settings.capability.code') },
  { value: 'reasoning', label: t('admin.settings.capability.reasoning') },
  { value: 'vision', label: t('admin.settings.capability.vision') },
  { value: 'analysis', label: t('admin.settings.capability.analysis') },
])

const saveAIModel = () => {
  if (!newAIModel.value.display_name?.trim()) {
    message.warning(t('admin.settings.aiModelNameRequired'))
    return
  }
  if (!newAIModel.value.model_name?.trim()) {
    message.warning(t('admin.settings.aiModelIdRequired'))
    return
  }
  const newModel: AIModelConfig = {
    id: `AI-${Date.now()}`,
    provider: newAIModel.value.provider || t('admin.ruleConfig.localDeploy'),
    model_name: newAIModel.value.model_name!,
    display_name: newAIModel.value.display_name!,
    type: newAIModel.value.type || 'local',
    endpoint: newAIModel.value.endpoint || '',
    api_key_configured: newAIModel.value.api_key_configured || false,
    max_tokens: newAIModel.value.max_tokens || 4096,
    context_window: newAIModel.value.context_window || 65536,
    cost_per_1k_tokens: newAIModel.value.cost_per_1k_tokens || 0,
    status: 'offline',
    enabled: true,
    description: newAIModel.value.description || '',
    capabilities: newAIModel.value.capabilities || ['text'],
  }
  aiModels.value.push(newModel)
  showAddAIModel.value = false
  message.success(t('admin.settings.aiModelAdded'))
}

const toggleAIModel = (id: string) => {
  const model = aiModels.value.find(m => m.id === id)
  if (model) {
    model.enabled = !model.enabled
    message.success(model.enabled ? t('admin.settings.enabled', model.display_name) : t('admin.settings.disabled', model.display_name))
  }
}

const deleteAIModel = (id: string) => {
  aiModels.value = aiModels.value.filter(m => m.id !== id)
  message.success(t('admin.settings.aiModelDeleted'))
}

// ===== OA System toggle (legacy) =====
const toggleOASystem = (id: string) => {
  const sys = oaSystems.value.find(s => s.id === id)
  if (sys) {
    sys.enabled = !sys.enabled
    message.success(sys.enabled ? t('admin.settings.enabled', sys.name) : t('admin.settings.disabled', sys.name))
  }
}

const getStatusConfig = (status: string) => {
  const configs: Record<string, { color: string; bg: string; label: string; icon: any }> = {
    connected: { color: 'var(--color-success)', bg: 'var(--color-success-bg)', label: t('admin.settings.connected'), icon: CheckCircleOutlined },
    disconnected: { color: 'var(--color-text-tertiary)', bg: 'var(--color-bg-hover)', label: t('admin.settings.disconnected'), icon: CloseCircleOutlined },
    testing: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', label: t('admin.settings.testing'), icon: SyncOutlined },
    online: { color: 'var(--color-success)', bg: 'var(--color-success-bg)', label: t('admin.settings.online'), icon: CheckCircleOutlined },
    offline: { color: 'var(--color-text-tertiary)', bg: 'var(--color-bg-hover)', label: t('admin.settings.offline'), icon: CloseCircleOutlined },
    maintenance: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', label: t('admin.settings.maintenance'), icon: ToolOutlined },
  }
  return configs[status] || configs.disconnected
}

const getModelTypeTag = (type: string) => {
  return type === 'local'
    ? { label: t('admin.ruleConfig.localDeploy'), color: 'var(--color-success)', bg: 'var(--color-success-bg)' }
    : { label: t('admin.ruleConfig.cloudAPI'), color: 'var(--color-info)', bg: 'var(--color-info-bg)' }
}

const saveGeneralConfig = async () => {
  saving.value = true
  await new Promise(resolve => setTimeout(resolve, 1000))
  saving.value = false
  message.success(t('admin.settings.saved'))
}

const enabledOADbs = computed(() => oaDbConnections.value.filter(c => c.enabled).length)
const enabledAIModels = computed(() => aiModels.value.filter(m => m.enabled).length)
const onlineAIModels = computed(() => aiModels.value.filter(m => m.status === 'online' && m.enabled).length)
</script>

<template>
  <div class="settings-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('admin.settings.title') }}</h1>
        <p class="page-subtitle">{{ t('admin.settings.subtitle') }}</p>
      </div>
    </div>

    <!-- Overview Stats -->
    <div class="overview-stats">
      <div class="overview-stat">
        <div class="overview-stat-icon overview-stat-icon--primary">
          <DatabaseOutlined />
        </div>
        <div class="overview-stat-info">
          <div class="overview-stat-value">{{ enabledOADbs }} / {{ oaDbConnections.length }}</div>
          <div class="overview-stat-label">{{ t('admin.settings.enabledOADb') }}</div>
        </div>
      </div>
      <div class="overview-stat">
        <div class="overview-stat-icon overview-stat-icon--success">
          <RobotOutlined />
        </div>
        <div class="overview-stat-info">
          <div class="overview-stat-value">{{ onlineAIModels }} / {{ enabledAIModels }}</div>
          <div class="overview-stat-label">{{ t('admin.settings.onlineAI') }}</div>
        </div>
      </div>
      <div class="overview-stat">
        <div class="overview-stat-icon overview-stat-icon--info">
          <GlobalOutlined />
        </div>
        <div class="overview-stat-info">
          <div class="overview-stat-value">{{ generalConfig.platform_version }}</div>
          <div class="overview-stat-label">{{ t('admin.settings.platformVersion') }}</div>
        </div>
      </div>
    </div>

    <!-- Tab Navigation -->
    <div class="tab-nav">
      <button
        v-for="tab in [
          { key: 'oa', label: t('admin.settings.tabOA'), icon: DatabaseOutlined },
          { key: 'ai', label: t('admin.settings.tabAI'), icon: RobotOutlined },
          { key: 'general', label: t('admin.settings.tabGeneral'), icon: SettingOutlined },
        ]"
        :key="tab.key"
        class="tab-btn"
        :class="{ 'tab-btn--active': activeTab === tab.key }"
        @click="activeTab = tab.key"
      >
        <component :is="tab.icon" />
        {{ tab.label }}
      </button>
    </div>

    <!-- OA Database Connections Tab -->
    <div v-if="activeTab === 'oa'" class="tab-content">
      <div class="tab-content-header">
        <p class="tab-desc">{{ t('admin.settings.oaDbDesc') }}</p>
        <a-button type="primary" @click="openAddOADb">
          <PlusOutlined /> {{ t('admin.settings.addOADb') }}
        </a-button>
      </div>

      <div class="oa-grid">
        <div v-for="conn in oaDbConnections" :key="conn.id" class="oa-card" :class="{ 'oa-card--disabled': !conn.enabled }">
          <div class="oa-card-header">
            <div class="oa-card-icon" :class="{ 'oa-card-icon--active': conn.enabled }">
              <DatabaseOutlined />
            </div>
            <div class="oa-card-info">
              <h3 class="oa-card-name">{{ conn.name }}</h3>
              <span class="oa-card-version">{{ conn.oa_type_label }}</span>
            </div>
            <div class="oa-card-status" :style="{ color: getStatusConfig(conn.status).color, background: getStatusConfig(conn.status).bg }">
              <component :is="getStatusConfig(conn.status).icon" :spin="conn.status === 'testing'" />
              {{ getStatusConfig(conn.status).label }}
            </div>
          </div>

          <p v-if="conn.description" class="oa-card-desc">{{ conn.description }}</p>

          <div class="oa-card-meta">
            <div class="oa-meta-item">
              <span class="oa-meta-label">{{ t('admin.settings.oaDbDriver') }}</span>
              <span class="oa-meta-value">{{ conn.jdbc_config.driver.toUpperCase() }}</span>
            </div>
            <div class="oa-meta-item">
              <span class="oa-meta-label">{{ t('admin.settings.oaDbHost') }}</span>
              <span class="oa-meta-value">{{ conn.jdbc_config.host }}:{{ conn.jdbc_config.port }}</span>
            </div>
            <div class="oa-meta-item">
              <span class="oa-meta-label">{{ t('admin.settings.oaDbDatabase') }}</span>
              <span class="oa-meta-value">{{ conn.jdbc_config.database }}</span>
            </div>
            <div class="oa-meta-item">
              <span class="oa-meta-label">{{ t('admin.settings.syncInterval') }}</span>
              <span class="oa-meta-value">{{ conn.sync_interval }}s</span>
            </div>
            <div v-if="conn.last_sync" class="oa-meta-item">
              <span class="oa-meta-label">{{ t('admin.settings.lastSync') }}</span>
              <span class="oa-meta-value">{{ conn.last_sync }}</span>
            </div>
          </div>

          <div class="oa-card-actions">
            <a-switch
              :checked="conn.enabled"
              @change="toggleOADb(conn.id)"
              :checked-children="t('admin.ruleConfig.enable')"
              :un-checked-children="t('admin.ruleConfig.disable')"
            />
            <div class="oa-card-btns">
              <a-button size="small" type="text" @click="openEditOADb(conn)">
                <EditOutlined />
              </a-button>
              <a-popconfirm :title="t('admin.settings.confirmDeleteOADb')" @confirm="deleteOADb(conn.id)" :okText="t('admin.tenants.create')" :cancelText="t('admin.tenants.cancel')">
                <a-button size="small" type="text" danger>
                  <DeleteOutlined />
                </a-button>
              </a-popconfirm>
              <a-button
                size="small"
                :disabled="!conn.enabled || conn.status === 'testing'"
                @click="testOADbConnection(conn.id)"
                class="test-conn-btn"
              >
                <SyncOutlined :spin="testingOADbId === conn.id" /> {{ testingOADbId === conn.id ? t('admin.settings.testingConn') : t('admin.settings.testConnection') }}
              </a-button>
            </div>
          </div>
        </div>
      </div>

      <div v-if="oaDbConnections.length === 0" class="empty-state">
        <DatabaseOutlined class="empty-icon" />
        <p>{{ t('admin.settings.noOADb') }}</p>
        <a-button type="primary" @click="openAddOADb">
          <PlusOutlined /> {{ t('admin.settings.addOADb') }}
        </a-button>
      </div>
    </div>

    <!-- AI Models Tab -->
    <div v-if="activeTab === 'ai'" class="tab-content">
      <div class="tab-content-header">
        <p class="tab-desc">{{ t('admin.settings.aiDesc') }}</p>
        <a-button type="primary" @click="openAddAIModel">
          <PlusOutlined /> {{ t('admin.settings.addAIModel') }}
        </a-button>
      </div>

      <div class="ai-grid">
        <div v-for="model in aiModels" :key="model.id" class="ai-card" :class="{ 'ai-card--disabled': !model.enabled }">
          <div class="ai-card-header">
            <div class="ai-card-icon" :class="{ 'ai-card-icon--local': model.type === 'local', 'ai-card-icon--cloud': model.type === 'cloud' }">
              <RobotOutlined v-if="model.type === 'local'" />
              <CloudServerOutlined v-else />
            </div>
            <div class="ai-card-info">
              <h3 class="ai-card-name">{{ model.display_name }}</h3>
              <span class="ai-card-provider">{{ model.provider }}</span>
            </div>
            <div class="ai-card-badges">
              <div class="ai-type-badge" :style="{ color: getModelTypeTag(model.type).color, background: getModelTypeTag(model.type).bg }">
                {{ getModelTypeTag(model.type).label }}
              </div>
              <div class="ai-status-badge" :style="{ color: getStatusConfig(model.status).color, background: getStatusConfig(model.status).bg }">
                <component :is="getStatusConfig(model.status).icon" />
                {{ getStatusConfig(model.status).label }}
              </div>
            </div>
          </div>

          <p class="ai-card-desc">{{ model.description }}</p>

          <div class="ai-capabilities">
            <span v-for="cap in model.capabilities" :key="cap" class="capability-tag">
              {{ t('admin.settings.capability.' + cap) }}
            </span>
          </div>

          <div class="ai-card-meta">
            <div class="ai-meta-row">
              <div class="ai-meta-item">
                <span class="ai-meta-label">{{ t('admin.settings.contextWindow') }}</span>
                <span class="ai-meta-value">{{ (model.context_window / 1024).toFixed(0) }}K</span>
              </div>
              <div class="ai-meta-item">
                <span class="ai-meta-label">{{ t('admin.settings.maxTokens') }}</span>
                <span class="ai-meta-value">{{ (model.max_tokens / 1024).toFixed(0) }}K</span>
              </div>
              <div class="ai-meta-item">
                <span class="ai-meta-label">{{ t('admin.settings.costPerToken') }}</span>
                <span class="ai-meta-value">{{ model.cost_per_1k_tokens > 0 ? '¥' + model.cost_per_1k_tokens.toFixed(2) : t('admin.settings.free') }}</span>
              </div>
            </div>
            <div class="ai-meta-row">
              <div class="ai-meta-item">
                <span class="ai-meta-label">{{ t('admin.settings.endpoint') }}</span>
                <span class="ai-meta-value ai-meta-value--mono">{{ model.endpoint }}</span>
              </div>
              <div class="ai-meta-item">
                <span class="ai-meta-label">API Key</span>
                <span class="ai-meta-value">
                  <CheckCircleOutlined v-if="model.api_key_configured" style="color: var(--color-success);" />
                  {{ model.api_key_configured ? t('admin.settings.apiKeyConfigured') : model.type === 'local' ? t('admin.settings.apiKeyLocal') : t('admin.settings.apiKeyMissing') }}
                </span>
              </div>
            </div>
          </div>

          <div class="ai-card-actions">
            <a-switch
              :checked="model.enabled"
              @change="toggleAIModel(model.id)"
              :checked-children="t('admin.ruleConfig.enable')"
              :un-checked-children="t('admin.ruleConfig.disable')"
            />
            <a-popconfirm :title="t('admin.settings.confirmDeleteAIModel')" @confirm="deleteAIModel(model.id)" :okText="t('admin.tenants.create')" :cancelText="t('admin.tenants.cancel')">
              <a-button size="small" type="text" danger>
                <DeleteOutlined />
              </a-button>
            </a-popconfirm>
          </div>
        </div>
      </div>
    </div>

    <!-- General Config Tab -->
    <div v-if="activeTab === 'general'" class="tab-content">
      <div class="tab-content-header">
        <p class="tab-desc">{{ t('admin.settings.generalDesc') }}</p>
      </div>

      <div class="config-sections">
        <div class="config-section">
          <div class="config-section-header">
            <div class="config-section-icon config-section-icon--primary"><GlobalOutlined /></div>
            <div>
              <h3>{{ t('admin.settings.platformInfo') }}</h3>
              <p>{{ t('admin.settings.platformInfoDesc') }}</p>
            </div>
          </div>
          <a-form layout="vertical">
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item :label="t('admin.settings.platformName')">
                  <a-input v-model:value="generalConfig.platform_name" size="large" />
                </a-form-item>
              </a-col>
              <a-col :span="6">
                <a-form-item :label="t('admin.settings.version')">
                  <a-input v-model:value="generalConfig.platform_version" size="large" disabled />
                </a-form-item>
              </a-col>
              <a-col :span="6">
                <a-form-item :label="t('admin.settings.defaultLanguage')">
                  <a-select v-model:value="generalConfig.default_language" size="large" :placeholder="t('admin.settings.selectLanguage')">
                    <a-select-option value="zh-CN">{{ t('admin.settings.zhCN') }}</a-select-option>
                    <a-select-option value="en-US">English</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item :label="t('admin.settings.sessionTimeout')">
                  <a-input-number v-model:value="generalConfig.session_timeout" :min="5" :max="1440" style="width: 100%;" size="large" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item :label="t('admin.settings.maxUpload')">
                  <a-input-number v-model:value="generalConfig.max_upload_size" :min="1" :max="500" style="width: 100%;" size="large" />
                </a-form-item>
              </a-col>
            </a-row>
          </a-form>
        </div>

        <div class="config-section">
          <div class="config-section-header">
            <div class="config-section-icon config-section-icon--success"><SafetyCertificateOutlined /></div>
            <div>
              <h3>{{ t('admin.settings.security') }}</h3>
              <p>{{ t('admin.settings.securityDesc') }}</p>
            </div>
          </div>
          <div class="toggle-grid">
            <div class="toggle-item">
              <div class="toggle-info">
                <div class="toggle-label">{{ t('admin.settings.auditTrail') }}</div>
                <div class="toggle-desc">{{ t('admin.settings.auditTrailDesc') }}</div>
              </div>
              <a-switch v-model:checked="generalConfig.enable_audit_trail" />
            </div>
            <div class="toggle-item">
              <div class="toggle-info">
                <div class="toggle-label">{{ t('admin.settings.encryption') }}</div>
                <div class="toggle-desc">{{ t('admin.settings.encryptionDesc') }}</div>
              </div>
              <a-switch v-model:checked="generalConfig.enable_data_encryption" />
            </div>
          </div>
        </div>

        <div class="config-section">
          <div class="config-section-header">
            <div class="config-section-icon config-section-icon--warning"><DatabaseOutlined /></div>
            <div>
              <h3>{{ t('admin.settings.backup') }}</h3>
              <p>{{ t('admin.settings.backupDesc') }}</p>
            </div>
          </div>
          <a-form layout="vertical">
            <div class="toggle-item" style="margin-bottom: 16px;">
              <div class="toggle-info">
                <div class="toggle-label">{{ t('admin.settings.enableBackup') }}</div>
                <div class="toggle-desc">{{ t('admin.settings.enableBackupDesc') }}</div>
              </div>
              <a-switch v-model:checked="generalConfig.backup_enabled" />
            </div>
            <a-row v-if="generalConfig.backup_enabled" :gutter="16">
              <a-col :span="12">
                <a-form-item :label="t('admin.settings.backupCron')">
                  <a-input v-model:value="generalConfig.backup_cron" size="large" placeholder="0 2 * * *" />
                  <div class="form-hint">{{ t('admin.settings.backupDefault') }}</div>
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item :label="t('admin.settings.backupRetention')">
                  <a-input-number v-model:value="generalConfig.backup_retention_days" :min="1" :max="365" style="width: 100%;" size="large" />
                </a-form-item>
              </a-col>
            </a-row>
          </a-form>
        </div>

        <div class="config-section">
          <div class="config-section-header">
            <div class="config-section-icon config-section-icon--info"><MailOutlined /></div>
            <div>
              <h3>{{ t('admin.settings.email') }}</h3>
              <p>{{ t('admin.settings.emailDesc') }}</p>
            </div>
          </div>
          <a-form layout="vertical">
            <a-form-item :label="t('admin.settings.notifEmail')">
              <a-input v-model:value="generalConfig.notification_email" size="large" placeholder="admin@example.com" />
            </a-form-item>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item :label="t('admin.settings.smtpHost')">
                  <a-input v-model:value="generalConfig.smtp_host" size="large" placeholder="smtp.example.com" />
                </a-form-item>
              </a-col>
              <a-col :span="6">
                <a-form-item :label="t('admin.settings.smtpPort')">
                  <a-input-number v-model:value="generalConfig.smtp_port" :min="1" :max="65535" style="width: 100%;" size="large" />
                </a-form-item>
              </a-col>
              <a-col :span="6">
                <a-form-item label="SSL/TLS">
                  <a-switch v-model:checked="generalConfig.smtp_ssl" />
                  <span class="switch-label-inline">{{ generalConfig.smtp_ssl ? t('admin.settings.sslEnabled') : t('admin.settings.sslDisabled') }}</span>
                </a-form-item>
              </a-col>
            </a-row>
            <a-form-item :label="t('admin.settings.smtpUsername')">
              <a-input v-model:value="generalConfig.smtp_username" size="large" :placeholder="t('admin.settings.smtpUserPlaceholder')" />
            </a-form-item>
          </a-form>
        </div>

        <div class="config-save">
          <a-button type="primary" size="large" :loading="saving" @click="saveGeneralConfig">
            <SaveOutlined /> {{ t('admin.settings.saveAll') }}
          </a-button>
        </div>
      </div>
    </div>

    <!-- Add/Edit OA Database Connection Modal -->
    <a-modal
      v-model:open="showAddOADb"
      :title="editingOADb ? t('admin.settings.editOADb') : t('admin.settings.addOADb')"
      @ok="saveOADb"
      :okText="editingOADb ? t('admin.settings.saveAll') : t('admin.tenants.create')"
      :cancelText="t('admin.tenants.cancel')"
      width="640px"
    >
      <a-form layout="vertical" style="margin-top: 16px;">
        <a-form-item :label="t('admin.settings.oaDbConnName')" required>
          <a-input v-model:value="newOADb.name" :placeholder="t('admin.settings.oaDbConnNamePlaceholder')" size="large" />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item :label="t('admin.settings.oaDbOAType')" required>
              <a-select v-model:value="newOADb.oa_type" size="large" @change="onOATypeChange">
                <a-select-option v-for="opt in oaTypeOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.dbDriver')">
              <a-select v-model:value="newOADb.jdbc_config!.driver" size="large" @change="onDriverChange">
                <a-select-option v-for="opt in driverOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="16">
            <a-form-item :label="t('admin.tenants.hostAddress')" required>
              <a-input v-model:value="newOADb.jdbc_config!.host" placeholder="192.168.1.100" size="large" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('admin.tenants.port')">
              <a-input-number v-model:value="newOADb.jdbc_config!.port" :min="1" :max="65535" style="width: 100%;" size="large" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item :label="t('admin.tenants.dbName')">
          <a-input v-model:value="newOADb.jdbc_config!.database" placeholder="ecology" size="large" />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.username')">
              <a-input v-model:value="newOADb.jdbc_config!.username" placeholder="oa_reader" size="large">
                <template #prefix><UserOutlined /></template>
              </a-input>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.password')">
              <a-input-password v-model:value="newOADb.jdbc_config!.password" :placeholder="t('admin.tenants.dbPassword')" size="large">
                <template #prefix><KeyOutlined /></template>
              </a-input-password>
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item :label="t('admin.tenants.poolSize')">
              <a-input-number v-model:value="newOADb.jdbc_config!.pool_size" :min="1" :max="100" style="width: 100%;" size="large" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('admin.tenants.connTimeout')">
              <a-input-number v-model:value="newOADb.jdbc_config!.connection_timeout" :min="5" :max="300" style="width: 100%;" size="large" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('admin.settings.syncInterval')">
              <a-input-number v-model:value="newOADb.sync_interval" :min="10" :max="3600" style="width: 100%;" size="large" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item :label="t('admin.tenants.description')">
          <a-textarea v-model:value="newOADb.description" :rows="2" :placeholder="t('admin.settings.oaDbDescPlaceholder')" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Add AI Model Modal -->
    <a-modal
      v-model:open="showAddAIModel"
      :title="t('admin.settings.addAIModel')"
      @ok="saveAIModel"
      :okText="t('admin.tenants.create')"
      :cancelText="t('admin.tenants.cancel')"
      width="640px"
    >
      <a-form layout="vertical" style="margin-top: 16px;">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item :label="t('admin.settings.aiModelDisplayName')" required>
              <a-input v-model:value="newAIModel.display_name" :placeholder="t('admin.settings.aiModelDisplayNamePlaceholder')" size="large" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('admin.settings.aiModelId')" required>
              <a-input v-model:value="newAIModel.model_name" :placeholder="t('admin.settings.aiModelIdPlaceholder')" size="large" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item :label="t('admin.settings.aiModelType')">
              <a-select v-model:value="newAIModel.type" size="large" @change="onModelTypeChange">
                <a-select-option value="local">{{ t('admin.ruleConfig.localDeploy') }}</a-select-option>
                <a-select-option value="cloud">{{ t('admin.ruleConfig.cloudAPI') }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('admin.settings.aiModelProvider')">
              <a-input v-model:value="newAIModel.provider" size="large" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item :label="t('admin.settings.endpoint')">
          <a-input v-model:value="newAIModel.endpoint" placeholder="http://192.168.1.50:8000/v1" size="large" />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item :label="t('admin.settings.maxTokens')">
              <a-input-number v-model:value="newAIModel.max_tokens" :min="512" :max="131072" :step="512" style="width: 100%;" size="large" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('admin.settings.contextWindow')">
              <a-input-number v-model:value="newAIModel.context_window" :min="2048" :max="1000000" :step="1024" style="width: 100%;" size="large" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('admin.settings.costPerToken')">
              <a-input-number v-model:value="newAIModel.cost_per_1k_tokens" :min="0" :step="0.01" :precision="2" style="width: 100%;" size="large" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item :label="t('admin.settings.aiModelCapabilities')">
          <a-checkbox-group v-model:value="newAIModel.capabilities">
            <a-checkbox v-for="cap in capabilityOptions" :key="cap.value" :value="cap.value">{{ cap.label }}</a-checkbox>
          </a-checkbox-group>
        </a-form-item>
        <a-form-item :label="t('admin.tenants.description')">
          <a-textarea v-model:value="newAIModel.description" :rows="2" :placeholder="t('admin.settings.aiModelDescPlaceholder')" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.page-header { margin-bottom: 24px; }
.page-title { font-size: 24px; font-weight: 700; color: var(--color-text-primary); margin: 0; }
.page-subtitle { font-size: 14px; color: var(--color-text-tertiary); margin: 4px 0 0; }

.overview-stats { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; margin-bottom: 24px; }
.overview-stat { display: flex; align-items: center; gap: 14px; padding: 18px 20px; background: var(--color-bg-card); border: 1px solid var(--color-border-light); border-radius: var(--radius-xl); transition: all 0.3s ease; }
.overview-stat:hover { box-shadow: var(--shadow-md); transform: translateY(-2px); }
.overview-stat-icon { width: 48px; height: 48px; border-radius: var(--radius-lg); display: flex; align-items: center; justify-content: center; font-size: 22px; flex-shrink: 0; }
.overview-stat-icon--primary { background: var(--color-primary-bg); color: var(--color-primary); }
.overview-stat-icon--success { background: var(--color-success-bg); color: var(--color-success); }
.overview-stat-icon--info { background: var(--color-info-bg); color: var(--color-info); }
.overview-stat-value { font-size: 22px; font-weight: 700; color: var(--color-text-primary); }
.overview-stat-label { font-size: 13px; color: var(--color-text-tertiary); margin-top: 2px; }

.tab-nav { display: flex; gap: 4px; background: var(--color-bg-hover); padding: 4px; border-radius: var(--radius-lg); margin-bottom: 24px; width: fit-content; }
.tab-btn { display: flex; align-items: center; gap: 6px; padding: 10px 20px; border: none; background: transparent; border-radius: var(--radius-md); font-size: 14px; font-weight: 500; color: var(--color-text-secondary); cursor: pointer; transition: all var(--transition-fast); }
.tab-btn:hover { color: var(--color-text-primary); }
.tab-btn--active { background: var(--color-bg-card); color: var(--color-primary); box-shadow: var(--shadow-xs); }

.tab-content-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.tab-desc { font-size: 14px; color: var(--color-text-secondary); margin: 0; }

.oa-grid, .ai-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(460px, 1fr)); gap: 20px; }
.oa-card, .ai-card { background: var(--color-bg-card); border: 1px solid var(--color-border-light); border-radius: var(--radius-xl); padding: 22px; transition: all 0.3s ease; display: flex; flex-direction: column; }
.oa-card:hover, .ai-card:hover { box-shadow: var(--shadow-md); }
.oa-card--disabled, .ai-card--disabled { opacity: 0.65; }

.oa-card-header { display: flex; align-items: center; gap: 14px; margin-bottom: 14px; }
.oa-card-icon { width: 48px; height: 48px; border-radius: var(--radius-lg); background: var(--color-bg-hover); color: var(--color-text-tertiary); display: flex; align-items: center; justify-content: center; font-size: 22px; flex-shrink: 0; transition: all 0.3s ease; }
.oa-card-icon--active { background: var(--color-primary-bg); color: var(--color-primary); }
.oa-card-info { flex: 1; }
.oa-card-name { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin: 0; }
.oa-card-version { font-size: 12px; color: var(--color-text-tertiary); }
.oa-card-status { display: flex; align-items: center; gap: 5px; font-size: 12px; font-weight: 500; padding: 4px 10px; border-radius: var(--radius-full); flex-shrink: 0; }
.oa-card-desc { font-size: 13px; color: var(--color-text-secondary); line-height: 1.5; margin: 0 0 14px; }
.oa-card-meta { display: flex; gap: 20px; padding: 10px 14px; background: var(--color-bg-page); border-radius: var(--radius-md); margin-bottom: 14px; flex-wrap: wrap; }
.oa-meta-label { font-size: 11px; color: var(--color-text-tertiary); display: block; }
.oa-meta-value { font-size: 13px; font-weight: 500; color: var(--color-text-primary); margin-top: 2px; display: block; }
.oa-card-actions { display: flex; justify-content: space-between; align-items: center; padding-top: 14px; border-top: 1px solid var(--color-border-light); margin-top: auto; }
.oa-card-btns { display: flex; gap: 4px; align-items: center; }

.test-conn-btn { border-color: var(--color-primary) !important; color: var(--color-primary) !important; font-weight: 500; }
.test-conn-btn:hover:not(:disabled) { background: var(--color-primary) !important; color: #fff !important; border-color: var(--color-primary) !important; }
.test-conn-btn:disabled { border-color: var(--color-border) !important; color: var(--color-text-tertiary) !important; }

.ai-card-header { display: flex; align-items: flex-start; gap: 14px; margin-bottom: 12px; }
.ai-card-icon { width: 48px; height: 48px; border-radius: var(--radius-lg); display: flex; align-items: center; justify-content: center; font-size: 22px; flex-shrink: 0; }
.ai-card-icon--local { background: var(--color-success-bg); color: var(--color-success); }
.ai-card-icon--cloud { background: var(--color-info-bg); color: var(--color-info); }
.ai-card-info { flex: 1; }
.ai-card-name { font-size: 15px; font-weight: 600; color: var(--color-text-primary); margin: 0; }
.ai-card-provider { font-size: 12px; color: var(--color-text-tertiary); }
.ai-card-badges { display: flex; gap: 6px; flex-shrink: 0; flex-wrap: wrap; justify-content: flex-end; }
.ai-type-badge, .ai-status-badge { display: flex; align-items: center; gap: 4px; font-size: 11px; font-weight: 500; padding: 3px 10px; border-radius: var(--radius-full); white-space: nowrap; }
.ai-card-desc { font-size: 13px; color: var(--color-text-secondary); line-height: 1.5; margin: 0 0 12px; }
.ai-capabilities { display: flex; gap: 6px; margin-bottom: 14px; flex-wrap: wrap; }
.capability-tag { font-size: 11px; padding: 2px 10px; border-radius: var(--radius-full); background: var(--color-bg-hover); color: var(--color-text-secondary); font-weight: 500; }
.ai-card-meta { padding: 10px 14px; background: var(--color-bg-page); border-radius: var(--radius-md); margin-bottom: 14px; }
.ai-meta-row { display: flex; gap: 20px; flex-wrap: wrap; }
.ai-meta-row + .ai-meta-row { margin-top: 8px; padding-top: 8px; border-top: 1px dashed var(--color-border-light); }
.ai-meta-label { font-size: 11px; color: var(--color-text-tertiary); display: block; }
.ai-meta-value { font-size: 13px; font-weight: 500; color: var(--color-text-primary); margin-top: 2px; display: block; }
.ai-meta-value--mono { font-family: var(--font-mono); font-size: 12px; }
.ai-card-actions { display: flex; justify-content: space-between; align-items: center; padding-top: 14px; border-top: 1px solid var(--color-border-light); margin-top: auto; }

.config-sections { display: flex; flex-direction: column; gap: 24px; }
.config-section { background: var(--color-bg-card); border: 1px solid var(--color-border-light); border-radius: var(--radius-xl); padding: 24px; }
.config-section-header { display: flex; align-items: center; gap: 14px; margin-bottom: 20px; }
.config-section-icon { width: 44px; height: 44px; border-radius: var(--radius-lg); display: flex; align-items: center; justify-content: center; font-size: 20px; flex-shrink: 0; }
.config-section-icon--primary { background: var(--color-primary-bg); color: var(--color-primary); }
.config-section-icon--success { background: var(--color-success-bg); color: var(--color-success); }
.config-section-icon--warning { background: var(--color-warning-bg); color: var(--color-warning); }
.config-section-icon--info { background: var(--color-info-bg); color: var(--color-info); }
.config-section-header h3 { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin: 0; }
.config-section-header p { font-size: 13px; color: var(--color-text-tertiary); margin: 2px 0 0; }

.toggle-grid { display: flex; flex-direction: column; gap: 0; }
.toggle-item { display: flex; justify-content: space-between; align-items: center; padding: 14px 0; border-bottom: 1px solid var(--color-border-light); }
.toggle-item:last-child { border-bottom: none; }
.toggle-label { font-size: 14px; font-weight: 500; color: var(--color-text-primary); }
.toggle-desc { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }
.form-hint { font-size: 11px; color: var(--color-text-tertiary); margin-top: 4px; }
.switch-label-inline { font-size: 13px; color: var(--color-text-tertiary); margin-left: 10px; }
.config-save { display: flex; justify-content: flex-end; padding: 4px 0; }

.empty-state { text-align: center; padding: 60px 20px; color: var(--color-text-tertiary); }
.empty-icon { font-size: 48px; margin-bottom: 16px; opacity: 0.4; }
.empty-state p { margin-bottom: 16px; }

@media (max-width: 1024px) {
  .overview-stats { grid-template-columns: 1fr 1fr; }
  .oa-grid, .ai-grid { grid-template-columns: 1fr; }
}
@media (max-width: 640px) {
  .overview-stats { grid-template-columns: 1fr; }
  .tab-nav { width: 100%; overflow-x: auto; -webkit-overflow-scrolling: touch; }
  .tab-btn { flex: 1; text-align: center; padding: 8px 12px; justify-content: center; }
}
</style>
