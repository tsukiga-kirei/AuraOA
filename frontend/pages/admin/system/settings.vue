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
import { type SystemGeneralConfig, mapConfigItems, configToUpdateRequest } from '~/types/settings'

const { t } = useI18n()

const {
  getConfigs, updateConfigs,
  listOATypes, listDBDrivers, listAIProviders, listAIDeployTypes,
  listOAConnections, createOAConnection, updateOAConnection, deleteOAConnection: apiDeleteOAConnection, testOAConnection: apiTestOAConnection, testOAConnectionParams: apiTestOAConnectionParams,
  listAIModels, createAIModel, updateAIModel, deleteAIModel: apiDeleteAIModel, testAIModelConnection: apiTestAIModelConnection, testAIModelConnectionById: apiTestAIModelConnectionById,
} = useSystemApi()

const loading = ref(false)
const activeTab = ref('oa')

// ===== 后端选项数据 =====
const oaTypeOptions = ref<{value: string; label: string}[]>([])
const driverOptions = ref<{value: string; label: string; default_port?: number}[]>([])
const aiDeployTypeOptions = ref<{value: string; label: string}[]>([])
const aiProviderOptions = ref<{value: string; label: string; deploy_type?: string}[]>([])

// ===== 数据列表 =====
interface OADbConnection {
  id: string; name: string; oa_type: string; oa_type_label: string;
  driver: string; host: string; port: number; database_name: string;
  username: string; pool_size: number; connection_timeout: number; test_on_borrow: boolean;
  status: string; sync_interval: number; enabled: boolean;
  description: string; created_at: string; updated_at: string;
}
interface AIModel {
  id: string; provider: string; provider_label: string; model_name: string; display_name: string;
  deploy_type: string; endpoint: string; api_key_configured: boolean;
  max_tokens: number; context_window: number; cost_per_1k_tokens: number;
  status: string; enabled: boolean; description: string; capabilities: string[];
  created_at: string; updated_at: string;
}

const oaDbConnections = ref<OADbConnection[]>([])
const aiModels = ref<AIModel[]>([])
const generalConfig = ref<SystemGeneralConfig>({} as SystemGeneralConfig)
const saving = ref(false)

onMounted(async () => {
  loading.value = true
  try {
    // 并行加载所有数据
    const [configs, oaTypes, drivers, deployTypes, providers, oaConns, models] = await Promise.all([
      getConfigs(),
      listOATypes(),
      listDBDrivers(),
      listAIDeployTypes(),
      listAIProviders(),
      listOAConnections(),
      listAIModels(),
    ])
    // 系统配置
    generalConfig.value = { ...generalConfig.value, ...mapConfigItems(configs) }
    // 选项数据
    oaTypeOptions.value = (oaTypes || []).map((o: any) => ({ value: o.code, label: o.label }))
    driverOptions.value = (drivers || []).map((o: any) => ({ value: o.code, label: o.label, default_port: o.default_port }))
    aiDeployTypeOptions.value = (deployTypes || []).map((o: any) => ({ value: o.code, label: o.label }))
    aiProviderOptions.value = (providers || []).map((o: any) => ({ value: o.code, label: o.label, deploy_type: o.deploy_type }))
    // 列表数据
    oaDbConnections.value = oaConns || []
    aiModels.value = models || []
  } catch (e) {
    message.error(t('admin.settings.loadFailed', '加载配置失败'))
  } finally {
    loading.value = false
  }
})

//===== OA数据库连接CRUD =====
const showAddOADb = ref(false)
const editingOADb = ref<OADbConnection | null>(null)
const testingOADbId = ref<string | null>(null)

const getDriverPort = (driver: string) => {
  const opt = driverOptions.value.find(o => o.value === driver)
  return opt?.default_port || 3306
}

const newOADb = ref<Record<string, any>>({
  name: '', oa_type: '', oa_type_label: '', description: '', sync_interval: 60,
  driver: 'mysql', host: '', port: 3306, database_name: '',
  username: '', password: '', pool_size: 10, connection_timeout: 30, test_on_borrow: true,
})

const resetNewOADb = () => {
  const defaultOAType = oaTypeOptions.value[0]
  newOADb.value = {
    name: '', oa_type: defaultOAType?.value || '', oa_type_label: defaultOAType?.label || '',
    description: '', sync_interval: 60,
    driver: 'mysql', host: '', port: 3306, database_name: '',
    username: '', password: '', pool_size: 10, connection_timeout: 30, test_on_borrow: true,
  }
}

const openAddOADb = () => {
  editingOADb.value = null
  resetNewOADb()
  showAddOADb.value = true
}

const openEditOADb = (conn: OADbConnection) => {
  editingOADb.value = conn
  newOADb.value = {
    name: conn.name, oa_type: conn.oa_type, oa_type_label: conn.oa_type_label,
    description: conn.description, sync_interval: conn.sync_interval,
    driver: conn.driver, host: conn.host, port: conn.port, database_name: conn.database_name,
    username: conn.username, password: '', pool_size: conn.pool_size,
    connection_timeout: conn.connection_timeout, test_on_borrow: conn.test_on_borrow,
  }
  showAddOADb.value = true
}

const onOATypeChange = (val: any) => {
  const opt = oaTypeOptions.value.find(o => o.value === val)
  if (opt) newOADb.value.oa_type_label = opt.label
}

const onDriverChange = (driver: any) => {
  newOADb.value.port = getDriverPort(driver as string)
}

const saveOADb = async () => {
  if (!newOADb.value.name?.trim()) {
    message.warning(t('admin.settings.oaDbNameRequired'))
    return
  }
  if (!newOADb.value.host?.trim()) {
    message.warning(t('admin.settings.oaDbHostRequired'))
    return
  }
  if (!newOADb.value.port) {
    message.warning(t('admin.settings.oaDbPortRequired', '请填写端口'))
    return
  }
  if (!newOADb.value.database_name?.trim()) {
    message.warning(t('admin.settings.oaDbDatabaseRequired', '请填写数据库名称'))
    return
  }
  if (!newOADb.value.username?.trim()) {
    message.warning(t('admin.settings.oaDbUsernameRequired', '请填写用户名'))
    return
  }
  if (!newOADb.value.password?.trim() && !editingOADb.value) {
    message.warning(t('admin.settings.oaDbPasswordRequired', '请填写密码'))
    return
  }
  try {
    if (editingOADb.value) {
      const updated = await updateOAConnection(editingOADb.value.id, newOADb.value)
      const idx = oaDbConnections.value.findIndex(c => c.id === editingOADb.value!.id)
      if (idx >= 0) oaDbConnections.value[idx] = updated
      message.success(t('admin.settings.oaDbUpdated'))
    } else {
      const created = await createOAConnection(newOADb.value)
      oaDbConnections.value.push(created)
      message.success(t('admin.settings.oaDbAdded'))
    }
    showAddOADb.value = false
  } catch (e) {
    message.error('操作失败')
  }
}

const deleteOADb = async (id: string) => {
  try {
    await apiDeleteOAConnection(id)
    oaDbConnections.value = oaDbConnections.value.filter(c => c.id !== id)
    message.success(t('admin.settings.oaDbDeleted'))
  } catch (e) {
    message.error('删除失败')
  }
}

const toggleOADb = async (id: string) => {
  const conn = oaDbConnections.value.find(c => c.id === id)
  if (!conn) return
  try {
    const updated = await updateOAConnection(id, { enabled: !conn.enabled })
    conn.enabled = updated.enabled
    message.success(conn.enabled ? t('admin.settings.enabled', conn.name) : t('admin.settings.disabled', conn.name))
  } catch (e) {
    message.error('操作失败')
  }
}

const testOADbConnection = async (id: string) => {
  const conn = oaDbConnections.value.find(c => c.id === id)
  if (!conn) return
  testingOADbId.value = id
  conn.status = 'testing'
  try {
    const result = await apiTestOAConnection(id)
    if (result.success) {
      conn.status = 'connected'
      message.success(t('admin.settings.connSuccess', conn.name))
    } else {
      conn.status = 'disconnected'
      message.warning(result.message || t('admin.settings.notEnabled', conn.name))
    }
  } catch (e) {
    conn.status = 'disconnected'
    message.error('测试连接失败')
  }
  testingOADbId.value = null
}

//===== AI模型CRUD =====
const showAddAIModel = ref(false)
const editingAIModel = ref<AIModel | null>(null)
const testingAIModelId = ref<string | null>(null)

const newAIModel = ref<Record<string, any>>({
  provider: '', provider_label: '', model_name: '', display_name: '', deploy_type: 'local',
  endpoint: '', api_key: '', max_tokens: 4096, context_window: 65536,
  cost_per_1k_tokens: 0, enabled: true, description: '', capabilities: ['text'],
})
const resetNewAIModel = () => {
  const defaultProvider = aiProviderOptions.value[0]
  newAIModel.value = {
    provider: defaultProvider?.value || '', provider_label: defaultProvider?.label || '',
    model_name: '', display_name: '', deploy_type: defaultProvider?.deploy_type || 'local',
    endpoint: '', api_key: '', max_tokens: 4096, context_window: 65536,
    cost_per_1k_tokens: 0, enabled: true, description: '', capabilities: ['text'],
  }
}

const openAddAIModel = () => {
  editingAIModel.value = null
  resetNewAIModel()
  showAddAIModel.value = true
}

const openEditAIModel = (model: AIModel) => {
  editingAIModel.value = model
  newAIModel.value = {
    provider: model.provider, provider_label: model.provider_label,
    model_name: model.model_name, display_name: model.display_name,
    deploy_type: model.deploy_type, endpoint: model.endpoint,
    api_key: '', max_tokens: model.max_tokens, context_window: model.context_window,
    cost_per_1k_tokens: model.cost_per_1k_tokens, enabled: model.enabled,
    description: model.description, capabilities: [...model.capabilities],
  }
  showAddAIModel.value = true
}

const onModelTypeChange = (val: any) => {
  // 按部署类型过滤服务商
  const filtered = aiProviderOptions.value.filter(p => p.deploy_type === val)
  if (filtered.length > 0) {
    newAIModel.value.provider = filtered[0].value
    newAIModel.value.provider_label = filtered[0].label
  }
  if (val === 'local') {
    newAIModel.value.cost_per_1k_tokens = 0
    newAIModel.value.api_key = ''
  }
}

const capabilityOptions = computed(() => [
  { value: 'text', label: t('admin.settings.capability.text') },
  { value: 'code', label: t('admin.settings.capability.code') },
  { value: 'reasoning', label: t('admin.settings.capability.reasoning') },
  { value: 'vision', label: t('admin.settings.capability.vision') },
  { value: 'analysis', label: t('admin.settings.capability.analysis') },
])

const onProviderChange = (val: any) => {
  const opt = aiProviderOptions.value.find(o => o.value === val)
  if (opt) newAIModel.value.provider_label = opt.label
}

const saveAIModel = async () => {
  if (!newAIModel.value.display_name?.trim()) {
    message.warning(t('admin.settings.aiModelNameRequired'))
    return
  }
  if (!newAIModel.value.model_name?.trim()) {
    message.warning(t('admin.settings.aiModelIdRequired'))
    return
  }
  if (!newAIModel.value.endpoint?.trim()) {
    message.warning(t('admin.settings.fillEndpointFirst'))
    return
  }
  try {
    const payload = { ...newAIModel.value }
    // 编辑模式下，如果 api_key 为空则不发送，避免后端覆盖已有密钥
    if (editingAIModel.value && !payload.api_key) {
      delete payload.api_key
    }
    if (editingAIModel.value) {
      const updated = await updateAIModel(editingAIModel.value.id, payload)
      const idx = aiModels.value.findIndex(m => m.id === editingAIModel.value!.id)
      if (idx >= 0) aiModels.value[idx] = updated
      message.success(t('admin.settings.aiModelUpdated', '模型已更新'))
    } else {
      const created = await createAIModel(payload)
      aiModels.value.push(created)
      message.success(t('admin.settings.aiModelAdded'))
    }
    showAddAIModel.value = false
  } catch (e) {
    message.error('操作失败')
  }
}

const toggleAIModel = async (id: string) => {
  const model = aiModels.value.find(m => m.id === id)
  if (!model) return
  try {
    const updated = await updateAIModel(id, { enabled: !model.enabled })
    model.enabled = updated.enabled
    message.success(model.enabled ? t('admin.settings.enabled', model.display_name) : t('admin.settings.disabled', model.display_name))
  } catch (e) {
    message.error('操作失败')
  }
}

const deleteAIModel = async (id: string) => {
  try {
    await apiDeleteAIModel(id)
    aiModels.value = aiModels.value.filter(m => m.id !== id)
    message.success(t('admin.settings.aiModelDeleted'))
  } catch (e) {
    message.error('删除失败')
  }
}

//===== AI 模型卡片测试连接 =====
const testAIModelById = async (id: string) => {
  const model = aiModels.value.find(m => m.id === id)
  if (!model) return
  testingAIModelId.value = id
  model.status = 'testing'
  try {
    const result = await apiTestAIModelConnectionById(id)
    if (result.success) {
      model.status = 'online'
      message.success(t('admin.settings.connSuccess', model.display_name))
    } else {
      model.status = 'offline'
      message.warning(result.message || t('admin.settings.notEnabled', model.display_name))
    }
  } catch (e) {
    model.status = 'offline'
    message.error(t('admin.settings.dbConnFailed'))
  }
  testingAIModelId.value = null
}

//===== 测试数据库模式连接 =====
const testingDbConn = ref(false)
const testDbConnection = async () => {
  if (!newOADb.value.host) {
    message.warning(t('admin.settings.fillHostFirst'))
    return
  }
  testingDbConn.value = true
  try {
    const result = await apiTestOAConnectionParams(newOADb.value)
    if (result.success) {
      message.success(t('admin.settings.dbConnSuccess'))
    } else {
      message.error(result.message || t('admin.settings.dbConnFailed'))
    }
  } catch (e) {
    message.error(t('admin.settings.dbConnFailed'))
  } finally {
    testingDbConn.value = false
  }
}

//===== AI 模型模态测试连接 =====
const testingModelConn = ref(false)
const testModelConnection = async () => {
  if (!newAIModel.value.endpoint) {
    message.warning(t('admin.settings.fillEndpointFirst'))
    return
  }
  testingModelConn.value = true
  try {
    const result = await apiTestAIModelConnection(newAIModel.value)
    if (result.success) {
      message.success(t('admin.settings.modelConnSuccess'))
    } else {
      message.error(result.message || t('admin.settings.dbConnFailed'))
    }
  } catch (e) {
    message.error(t('admin.settings.dbConnFailed'))
  } finally {
    testingModelConn.value = false
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
  try {
    // 使用统一的序列化工具将前端表单模型转回后端 key-value 格式
    await updateConfigs(configToUpdateRequest(generalConfig.value))
    message.success(t('admin.settings.saved'))
  } catch (e) {
    message.error(t('admin.settings.saveFailed', '保存配置失败'))
  } finally {
    saving.value = false
  }
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

    <!--概览统计-->
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

    <!--选项卡导航-->
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

    <!--OA 数据库连接选项卡-->
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
              <span class="oa-meta-value">{{ conn.driver.toUpperCase() }}</span>
            </div>
            <div class="oa-meta-item">
              <span class="oa-meta-label">{{ t('admin.settings.oaDbHost') }}</span>
              <span class="oa-meta-value">{{ conn.host }}:{{ conn.port }}</span>
            </div>
            <div class="oa-meta-item">
              <span class="oa-meta-label">{{ t('admin.settings.oaDbDatabase') }}</span>
              <span class="oa-meta-value">{{ conn.database_name }}</span>
            </div>
            <div class="oa-meta-item">
              <span class="oa-meta-label">{{ t('admin.settings.syncInterval') }}</span>
              <span class="oa-meta-value">{{ conn.sync_interval }}s</span>
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
              <a-popconfirm :title="t('admin.settings.confirmDeleteOADb')" @confirm="deleteOADb(conn.id)" :okText="t('admin.ruleConfig.confirm')" :cancelText="t('admin.tenants.cancel')">
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

    <!--AI 模型选项卡-->
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
            <div class="ai-card-icon" :class="{ 'ai-card-icon--local': model.deploy_type === 'local', 'ai-card-icon--cloud': model.deploy_type === 'cloud' }">
              <RobotOutlined v-if="model.deploy_type === 'local'" />
              <CloudServerOutlined v-else />
            </div>
            <div class="ai-card-info">
              <h3 class="ai-card-name">{{ model.display_name }}</h3>
              <span class="ai-card-provider">{{ model.provider_label || model.provider }}</span>
            </div>
            <div class="ai-card-badges">
              <div class="ai-type-badge" :style="{ color: getModelTypeTag(model.deploy_type).color, background: getModelTypeTag(model.deploy_type).bg }">
                {{ getModelTypeTag(model.deploy_type).label }}
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
                  {{ model.api_key_configured ? t('admin.settings.apiKeyConfigured') : model.deploy_type === 'local' ? t('admin.settings.apiKeyLocal') : t('admin.settings.apiKeyMissing') }}
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
            <div class="oa-card-btns">
              <a-button size="small" type="text" @click="openEditAIModel(model)">
                <EditOutlined />
              </a-button>
              <a-popconfirm :title="t('admin.settings.confirmDeleteAIModel')" @confirm="deleteAIModel(model.id)" :okText="t('admin.ruleConfig.confirm')" :cancelText="t('admin.tenants.cancel')">
                <a-button size="small" type="text" danger>
                  <DeleteOutlined />
                </a-button>
              </a-popconfirm>
              <a-button
                size="small"
                :disabled="!model.enabled || model.status === 'testing'"
                @click="testAIModelById(model.id)"
                class="test-conn-btn"
              >
                <SyncOutlined :spin="testingAIModelId === model.id" /> {{ testingAIModelId === model.id ? t('admin.settings.testingConn') : t('admin.settings.testConnection') }}
              </a-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!--常规配置选项卡-->
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
            <!-- 平台基本信息 -->
            <div class="config-subsection-title">{{ t('admin.settings.platformInfo') }}</div>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item :label="t('admin.settings.platformName')">
                  <a-input v-model:value="generalConfig.platform_name" size="large" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item :label="t('admin.settings.version')">
                  <a-input v-model:value="generalConfig.platform_version" size="large" />
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="8">
                <a-form-item :label="t('admin.settings.defaultLanguage')">
                  <a-select v-model:value="generalConfig.default_language" size="large" style="width: 100%;">
                    <a-select-option value="zh-CN">简体中文</a-select-option>
                    <a-select-option value="en-US">English</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item :label="t('admin.settings.maxUpload')">
                  <a-input-number v-model:value="generalConfig.max_upload_size" :min="1" :max="500" style="width: 100%;" size="large" :addon-after="'MB'" />
                </a-form-item>
              </a-col>
            </a-row>

            <!-- 认证 & Token 有效期 -->
            <a-divider style="margin: 8px 0 20px;" />
            <div class="config-subsection-title">{{ t('admin.settings.authTokenConfig') }}</div>
            <a-row :gutter="16">
              <a-col :span="6">
                <a-form-item :label="t('admin.settings.loginFailLockThreshold')">
                  <a-input-number v-model:value="generalConfig.login_fail_lock_threshold" :min="1" :max="20" style="width: 100%;" size="large" :addon-after="t('admin.settings.times')" />
                </a-form-item>
              </a-col>
              <a-col :span="6">
                <a-form-item :label="t('admin.settings.accountLockMinutes')">
                  <a-input-number v-model:value="generalConfig.account_lock_minutes" :min="1" :max="1440" style="width: 100%;" size="large" :addon-after="t('admin.settings.minutes')" />
                </a-form-item>
              </a-col>
              <a-col :span="6">
                <a-form-item :label="t('admin.settings.accessTokenTtl')">
                  <a-input-number v-model:value="generalConfig.access_token_ttl_hours" :min="1" :max="168" style="width: 100%;" size="large" :addon-after="t('admin.settings.hours')" />
                </a-form-item>
              </a-col>
              <a-col :span="6">
                <a-form-item :label="t('admin.settings.refreshTokenTtl')">
                  <a-input-number v-model:value="generalConfig.refresh_token_ttl_days" :min="1" :max="365" style="width: 100%;" size="large" :addon-after="t('admin.settings.days')" />
                </a-form-item>
              </a-col>
            </a-row>

            <!-- 配额与策略 -->
            <a-divider style="margin: 8px 0 20px;" />
            <div class="config-subsection-title">{{ t('admin.settings.tenantQuotaConfig') }}</div>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item :label="t('admin.settings.tenantDefaultTokenQuota')">
                  <a-input-number v-model:value="generalConfig.tenant_default_token_quota" :min="1000" :max="10000000" :step="1000" style="width: 100%;" size="large" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item :label="t('admin.settings.tenantDefaultMaxConcurrency')">
                  <a-input-number v-model:value="generalConfig.tenant_default_max_concurrency" :min="1" :max="200" style="width: 100%;" size="large" />
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item :label="t('admin.settings.tenantDefaultLogRetentionDays')">
                  <a-input-number v-model:value="generalConfig.tenant_default_log_retention_days" :min="1" :max="3650" style="width: 100%;" size="large" :addon-after="t('admin.settings.days')" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item :label="t('admin.settings.tenantDefaultDataRetentionDays')">
                  <a-input-number v-model:value="generalConfig.tenant_default_data_retention_days" :min="1" :max="3650" style="width: 100%;" size="large" :addon-after="t('admin.settings.days')" />
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
            <a-form-item :label="t('admin.settings.smtpPassword')">
              <a-input-password v-model:value="generalConfig.smtp_password" size="large" />
            </a-form-item>
            <a-form-item :label="t('admin.settings.smtpSender')">
              <a-input v-model:value="generalConfig.smtp_sender" size="large" :placeholder="t('admin.settings.smtpUserPlaceholder')" />
            </a-form-item>
          </a-form>
        </div>

        <div class="config-save">
          <a-button type="primary" size="large" :loading="saving" @click="saveGeneralConfig">
            <template #icon>
              <SyncOutlined v-if="saving" />
              <SaveOutlined v-else />
            </template>
            {{ t('admin.settings.saveAll') }}
          </a-button>
        </div>
      </div>
    </div>

    <!--添加/编辑 OA 数据库连接模式-->
    <a-modal
      v-model:open="showAddOADb"
      :title="editingOADb ? t('admin.settings.editOADb') : t('admin.settings.addOADb')"
      :footer="null"
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
              <a-select v-model:value="newOADb.driver" size="large" @change="onDriverChange">
                <a-select-option v-for="opt in driverOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="16">
            <a-form-item :label="t('admin.tenants.hostAddress')" required>
              <a-input v-model:value="newOADb.host" placeholder="192.168.1.100" size="large" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('admin.tenants.port')" required>
              <a-input-number v-model:value="newOADb.port" :min="1" :max="65535" style="width: 100%;" size="large" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item :label="t('admin.tenants.dbName')" required>
          <a-input v-model:value="newOADb.database_name" placeholder="ecology" size="large" />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.username')" required>
              <a-input v-model:value="newOADb.username" placeholder="oa_reader" size="large">
                <template #prefix><UserOutlined /></template>
              </a-input>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.password')" required>
              <a-input-password v-model:value="newOADb.password" :placeholder="t('admin.tenants.dbPassword')" size="large">
                <template #prefix><KeyOutlined /></template>
              </a-input-password>
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item :label="t('admin.tenants.poolSize')">
              <a-input-number v-model:value="newOADb.pool_size" :min="1" :max="100" style="width: 100%;" size="large" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('admin.tenants.connTimeout')">
              <a-input-number v-model:value="newOADb.connection_timeout" :min="5" :max="300" style="width: 100%;" size="large" />
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
      <div style="display: flex; justify-content: space-between; align-items: center; padding-top: 16px; border-top: 1px solid var(--color-border-light); margin-top: 8px;">
        <a-button :loading="testingDbConn" @click="testDbConnection">
          <template #icon>
            <SyncOutlined v-if="testingDbConn" />
            <DatabaseOutlined v-else />
          </template>
          {{ t('admin.settings.testConnection') }}
        </a-button>
        <div style="display: flex; gap: 8px;">
          <a-button @click="showAddOADb = false">{{ t('admin.tenants.cancel') }}</a-button>
          <a-button type="primary" @click="saveOADb">{{ editingOADb ? t('admin.settings.saveAll') : t('admin.tenants.create') }}</a-button>
        </div>
      </div>
    </a-modal>

    <!--添加AI模型模态-->
    <a-modal
      v-model:open="showAddAIModel"
      :title="editingAIModel ? t('admin.settings.editAIModel', '编辑AI模型') : t('admin.settings.addAIModel')"
      :footer="null"
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
              <a-select v-model:value="newAIModel.deploy_type" size="large" @change="onModelTypeChange">
                <a-select-option v-for="opt in aiDeployTypeOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('admin.settings.aiModelProvider')">
              <a-select v-model:value="newAIModel.provider" size="large" @change="onProviderChange">
                <a-select-option v-for="opt in aiProviderOptions.filter(p => p.deploy_type === newAIModel.deploy_type)" :key="opt.value" :value="opt.value">{{ opt.label }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item :label="t('admin.settings.endpoint')" required>
          <a-input v-model:value="newAIModel.endpoint" placeholder="http://192.168.1.50:8000/v1" size="large" />
        </a-form-item>
        <a-form-item label="API Key">
          <a-input-password v-model:value="newAIModel.api_key" :placeholder="editingAIModel?.api_key_configured ? t('admin.settings.apiKeyKeepCurrent', '留空则保持当前密钥') : t('admin.settings.apiKeyPlaceholder', '输入 API Key')" size="large">
            <template #prefix><KeyOutlined /></template>
          </a-input-password>
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
      <div style="display: flex; justify-content: space-between; align-items: center; padding-top: 16px; border-top: 1px solid var(--color-border-light); margin-top: 8px;">
        <a-button :loading="testingModelConn" @click="testModelConnection">
          <template #icon>
            <SyncOutlined v-if="testingModelConn" />
            <RobotOutlined v-else />
          </template>
          {{ t('admin.settings.testConnection') }}
        </a-button>
        <div style="display: flex; gap: 8px;">
          <a-button @click="showAddAIModel = false">{{ t('admin.tenants.cancel') }}</a-button>
          <a-button type="primary" @click="saveAIModel">{{ editingAIModel ? t('admin.settings.saveAll') : t('admin.tenants.create') }}</a-button>
        </div>
      </div>
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
.config-subsection-title { font-size: 13px; font-weight: 600; color: var(--color-text-secondary); margin-bottom: 16px; letter-spacing: 0.02em; }

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
