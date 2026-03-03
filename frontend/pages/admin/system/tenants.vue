<script setup lang="ts">
definePageMeta({ middleware: 'auth', layout: 'default' })

import {useI18n} from '~/composables/useI18n'
import {
  ClockCircleOutlined,
  DatabaseOutlined,
  EditOutlined,
  InfoCircleOutlined,
  LinkOutlined,
  MailOutlined,
  PhoneOutlined,
  PlusOutlined,
  RobotOutlined,
  SafetyCertificateOutlined,
  TeamOutlined,
  ThunderboltOutlined,
  UserOutlined,
} from '@ant-design/icons-vue'
import {message} from 'ant-design-vue'

const { t } = useI18n()

const { mockAIModelConfigs, mockOADatabaseConnections } = useMockData()
const { listTenants: fetchTenants, createTenant: apiCreateTenant, updateTenant: apiUpdateTenant } = useSystemApi()

interface TenantData {
  id: string; name: string; code: string; description: string; status: string
  oa_type: string; token_quota: number; token_used: number; max_concurrency: number
  ai_config: any; sso_enabled: boolean; sso_endpoint: string
  log_retention_days: number; data_retention_days: number; allow_custom_model: boolean
  contact_name: string; contact_email: string; contact_phone: string
  created_at: string; updated_at: string
  oa_db_connection_id?: string
}

const tenants = ref<TenantData[]>([])
const loading = ref(false)
const selectedTenant = ref<TenantData | null>(null)
const showCreate = ref(false)
const showDetail = ref(false)
const detailActiveTab = ref('basic')

onMounted(async () => {
  loading.value = true
  try {
    const data = await fetchTenants()
    tenants.value = data.map((t: any) => ({
      ...t,
      ai_config: t.ai_config || { default_provider: '', default_model: '', fallback_provider: '', fallback_model: '', max_tokens_per_request: 4096, temperature: 0.3, timeout_seconds: 60, retry_count: 2 },
    }))
  } catch (e) {
    message.error('加载租户列表失败')
  } finally {
    loading.value = false
  }
})

//租户配置下拉列表的可用 AI 模型
const availableModels = computed(() => mockAIModelConfigs.filter(m => m.enabled))

//系统设置中可用的 OA 数据库连接
const availableOADbs = computed(() => mockOADatabaseConnections.filter(c => c.enabled))

//通过id获取OA DB连接名称
const getOADbName = (id: string) => {
  const conn = mockOADatabaseConnections.find(c => c.id === id)
  return conn ? conn.name : t('admin.tenants.notConfigured')
}

const getOADbInfo = (id: string) => {
  return mockOADatabaseConnections.find(c => c.id === id) || null
}

const newTenant = ref({
  name: '',
  code: '',
  oa_db_connection_id: '',
  token_quota: 10000,
  max_concurrency: 10,
  contact_name: '',
  contact_email: '',
  contact_phone: '',
  description: '',
  ai_provider: 'Xinference',
  ai_model: '',
})

const createTenant = async () => {
  if (!newTenant.value.name || !newTenant.value.code) {
    message.warning(t('admin.tenants.fillRequired'))
    return
  }
  try {
    const created = await apiCreateTenant({
      name: newTenant.value.name,
      code: newTenant.value.code,
      token_quota: newTenant.value.token_quota,
      max_concurrency: newTenant.value.max_concurrency,
      contact_name: newTenant.value.contact_name,
      contact_email: newTenant.value.contact_email,
      contact_phone: newTenant.value.contact_phone,
      description: newTenant.value.description,
      ai_config: { default_provider: newTenant.value.ai_provider, default_model: newTenant.value.ai_model || '' },
    })
    const tenantObj: TenantData = {
      ...created,
      ai_config: created.ai_config || { default_provider: '', default_model: '', fallback_provider: '', fallback_model: '', max_tokens_per_request: 4096, temperature: 0.3, timeout_seconds: 60, retry_count: 2 },
    }
    tenants.value.push(tenantObj)
    showCreate.value = false
    message.success(t('admin.tenants.createSuccess'))
    newTenant.value = { name: '', code: '', oa_db_connection_id: '', token_quota: 10000, max_concurrency: 10, contact_name: '', contact_email: '', contact_phone: '', description: '', ai_provider: 'Xinference', ai_model: '' }
    openDetail(tenantObj)
  } catch (e: any) {
    message.error(e.message || '创建租户失败')
  }
}
const openDetail = (tenant: TenantData) => {
  selectedTenant.value = { ...tenant, ai_config: { ...(tenant.ai_config || {}) } }
  detailActiveTab.value = 'basic'
  showDetail.value = true
}

const saveTenantDetail = async () => {
  if (!selectedTenant.value) return
  try {
    const updated = await apiUpdateTenant(selectedTenant.value.id, {
      name: selectedTenant.value.name,
      description: selectedTenant.value.description,
      status: selectedTenant.value.status,
      token_quota: selectedTenant.value.token_quota,
      max_concurrency: selectedTenant.value.max_concurrency,
      ai_config: selectedTenant.value.ai_config,
      sso_enabled: selectedTenant.value.sso_enabled,
      sso_endpoint: selectedTenant.value.sso_endpoint,
      log_retention_days: selectedTenant.value.log_retention_days,
      data_retention_days: selectedTenant.value.data_retention_days,
      allow_custom_model: selectedTenant.value.allow_custom_model,
      contact_name: selectedTenant.value.contact_name,
      contact_email: selectedTenant.value.contact_email,
      contact_phone: selectedTenant.value.contact_phone,
    })
    const idx = tenants.value.findIndex(t => t.id === selectedTenant.value!.id)
    if (idx >= 0) tenants.value[idx] = { ...tenants.value[idx], ...updated }
    showDetail.value = false
    message.success(t('admin.tenants.saveSuccess'))
  } catch (e: any) {
    message.error(e.message || '保存失败')
  }
}

const toggleTenantStatus = async (id: string) => {
  const tVal = tenants.value.find(x => x.id === id)
  if (!tVal) return
  const newStatus = tVal.status === 'active' ? 'inactive' : 'active'
  try {
    await apiUpdateTenant(id, { status: newStatus })
    tVal.status = newStatus
    message.success(newStatus === 'active' ? t('admin.tenants.enabled') : t('admin.tenants.disabled'))
  } catch (e: any) {
    message.error(e.message || '操作失败')
  }
}

const testConnection = async () => {
  //不再需要 - 在系统级别管理连接
}

const getQuotaPercent = (used: number, total: number) => Math.round((used / total) * 100)

const getQuotaColor = (percent: number) => {
  if (percent >= 90) return '#ef4444'
  if (percent >= 70) return '#f59e0b'
  return '#10b981'
}

//===== 按提供商筛选 AI 模型 =====
const providerOptions = ['Xinference', '阿里云百炼']

const filteredModelsForProvider = (provider: string) => {
  if (provider === 'Xinference') return availableModels.value.filter(m => m.type === 'local')
  if (provider === '阿里云百炼') return availableModels.value.filter(m => m.type === 'cloud')
  return availableModels.value
}

const primaryFilteredModels = computed(() => {
  if (!selectedTenant.value) return availableModels.value
  return filteredModelsForProvider(selectedTenant.value.ai_config.default_provider)
})

const fallbackFilteredModels = computed(() => {
  if (!selectedTenant.value) return availableModels.value
  return filteredModelsForProvider(selectedTenant.value.ai_config.fallback_provider)
})

const newTenantFilteredModels = computed(() => {
  return filteredModelsForProvider(newTenant.value.ai_provider)
})

const onPrimaryProviderChange = () => {
  if (selectedTenant.value) {
    selectedTenant.value.ai_config.default_model = ''
  }
}

const onFallbackProviderChange = () => {
  if (selectedTenant.value) {
    selectedTenant.value.ai_config.fallback_model = ''
  }
}

const onNewTenantProviderChange = () => {
  newTenant.value.ai_model = ''
}
</script>

<template>
  <div class="system-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('admin.tenants.title') }}</h1>
        <p class="page-subtitle">{{ t('admin.tenants.subtitle') }}</p>
      </div>
      <a-button type="primary" size="large" @click="showCreate = true">
        <PlusOutlined /> {{ t('admin.tenants.addTenant') }}
      </a-button>
    </div>

    <!--租户卡网格-->
    <div class="tenant-grid">
      <div v-for="tenant in tenants" :key="tenant.id" class="tenant-card" @click="openDetail(tenant)">
        <div class="tenant-card-header">
          <div class="tenant-avatar">
            <TeamOutlined />
          </div>
          <div class="tenant-info">
            <div class="tenant-name">{{ tenant.name }}</div>
            <div class="tenant-code">{{ tenant.code }} · {{ tenant.id }}</div>
          </div>
          <div
            class="tenant-status"
            :class="tenant.status === 'active' ? 'tenant-status--active' : 'tenant-status--inactive'"
          >
            <span class="tenant-status-dot" />
            {{ tenant.status === 'active' ? t('admin.tenants.running') : t('admin.tenants.stopped') }}
          </div>
        </div>

        <!--快速信息标签-->
        <div class="tenant-tags">
          <span class="info-tag info-tag--primary">
            <DatabaseOutlined /> {{ getOADbName(tenant.oa_db_connection_id) }}
          </span>
          <span class="info-tag info-tag--info">
            <RobotOutlined /> {{ tenant.ai_config.default_model }}
          </span>
          <span v-if="tenant.sso_enabled" class="info-tag info-tag--success">
            <SafetyCertificateOutlined /> SSO
          </span>
        </div>

        <!--统计行-->
        <div class="tenant-stats">
          <div class="stat-item">
            <span class="stat-label">{{ t('admin.tenants.tokenUsage') }}</span>
            <span class="stat-value">
              {{ (tenant.token_used / 1000).toFixed(1) }}K / {{ (tenant.token_quota / 1000).toFixed(0) }}K
            </span>
          </div>
          <div class="stat-item">
            <span class="stat-label">{{ t('admin.tenants.maxConcurrency') }}</span>
            <span class="stat-value">{{ tenant.max_concurrency }}</span>
          </div>
        </div>

        <!--代币使用栏-->
        <div class="quota-bar-wrapper">
          <div class="quota-bar">
            <div
              class="quota-bar-fill"
              :style="{
                width: getQuotaPercent(tenant.token_used, tenant.token_quota) + '%',
                background: getQuotaColor(getQuotaPercent(tenant.token_used, tenant.token_quota)),
              }"
            />
          </div>
          <span class="quota-percent" :style="{ color: getQuotaColor(getQuotaPercent(tenant.token_used, tenant.token_quota)) }">
            {{ getQuotaPercent(tenant.token_used, tenant.token_quota) }}%
          </span>
        </div>

        <div class="tenant-card-footer">
          <span class="tenant-created">
            <ClockCircleOutlined /> {{ tenant.created_at }}
          </span>
          <div class="tenant-card-actions" @click.stop>
            <a-button size="small" type="text" @click="openDetail(tenant)">
              <EditOutlined /> {{ t('admin.tenants.configure') }}
            </a-button>
            <a-button size="small" type="text" @click="toggleTenantStatus(tenant.id)">
              {{ tenant.status === 'active' ? t('admin.tenants.stop') : t('admin.tenants.enable') }}
            </a-button>
          </div>
        </div>
      </div>
    </div>

    <!--创建租户模式-->
    <a-modal v-model:open="showCreate" :title="t('admin.tenants.createTenant')" @ok="createTenant" :okText="t('admin.tenants.create')" :cancelText="t('admin.tenants.cancel')" width="560px">
      <a-form layout="vertical" style="margin-top: 16px;">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.tenantName')" required>
              <a-input v-model:value="newTenant.name" :placeholder="t('admin.tenants.tenantNamePlaceholder')" size="large" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.tenantCode')" required>
              <a-input v-model:value="newTenant.code" :placeholder="t('admin.tenants.tenantCodePlaceholder')" size="large" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item :label="t('admin.tenants.oaDbConnection')">
          <a-select v-model:value="newTenant.oa_db_connection_id" size="large" :placeholder="t('admin.tenants.selectOADb')" allowClear>
            <a-select-option v-for="conn in availableOADbs" :key="conn.id" :value="conn.id">
              {{ conn.name }} ({{ conn.oa_type_label }})
            </a-select-option>
          </a-select>
          <div style="font-size: 12px; color: var(--color-text-tertiary); margin-top: 4px;">
            {{ t('admin.tenants.oaDbHint') }}
            <a @click="navigateTo('/admin/system/settings')" style="cursor: pointer;">{{ t('admin.tenants.systemSettings') }}</a>
          </div>
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.tokenQuota')">
              <a-input-number v-model:value="newTenant.token_quota" :min="1000" :step="1000" style="width: 100%;" size="large" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.maxConcurrencyLabel')">
              <a-input-number v-model:value="newTenant.max_concurrency" :min="1" :max="100" style="width: 100%;" size="large" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item :label="t('admin.tenants.contact')">
              <a-input v-model:value="newTenant.contact_name" :placeholder="t('admin.tenants.contactPlaceholder')" size="large" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('admin.tenants.contactEmail')">
              <a-input v-model:value="newTenant.contact_email" placeholder="admin@example.com" size="large" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('admin.tenants.contactPhone')">
              <a-input v-model:value="newTenant.contact_phone" :placeholder="t('admin.tenants.contactPhonePlaceholder')" size="large" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.aiProvider')">
              <a-select v-model:value="newTenant.ai_provider" size="large" @change="onNewTenantProviderChange">
                <a-select-option v-for="p in providerOptions" :key="p" :value="p">{{ p }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('admin.tenants.modelName')">
              <a-select v-model:value="newTenant.ai_model" size="large" :placeholder="t('admin.tenants.selectModel')">
                <a-select-option v-for="m in newTenantFilteredModels" :key="m.model_name" :value="m.model_name">
                  {{ m.display_name }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item :label="t('admin.tenants.description')">
          <a-textarea v-model:value="newTenant.description" :rows="2" :placeholder="t('admin.tenants.descPlaceholder')" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!--租户细节抽屉-->
    <a-drawer
      v-model:open="showDetail"
      :title="selectedTenant?.name + t('admin.tenants.tenantConfig', '')"
      placement="right"
      :width="720"
      @close="showDetail = false"
    >
      <template v-if="selectedTenant">
        <div class="detail-tabs">
          <button
            v-for="tab in [
              { key: 'basic', label: t('admin.tenants.tabBasic'), icon: InfoCircleOutlined },
              { key: 'oadb', label: t('admin.tenants.tabOADb'), icon: DatabaseOutlined },
              { key: 'ai', label: t('admin.tenants.tabAI'), icon: RobotOutlined },
              { key: 'quota', label: t('admin.tenants.tabQuota'), icon: ThunderboltOutlined },
              { key: 'security', label: t('admin.tenants.tabSecurity'), icon: SafetyCertificateOutlined },
            ]"
            :key="tab.key"
            class="detail-tab-btn"
            :class="{ 'detail-tab-btn--active': detailActiveTab === tab.key }"
            @click="detailActiveTab = tab.key"
          >
            <component :is="tab.icon" />
            {{ tab.label }}
          </button>
        </div>

        <!--基本信息选项卡-->
        <div v-if="detailActiveTab === 'basic'" class="detail-section">
          <div class="section-header">
            <h3><UserOutlined /> {{ t('admin.tenants.basicInfo') }}</h3>
          </div>
          <a-form layout="vertical">
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item :label="t('admin.tenants.tenantName')">
                  <a-input v-model:value="selectedTenant.name" size="large" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item :label="t('admin.tenants.tenantCode')">
                  <a-input v-model:value="selectedTenant.code" size="large" disabled />
                </a-form-item>
              </a-col>
            </a-row>
            <a-form-item :label="t('admin.tenants.description')">
              <a-textarea v-model:value="selectedTenant.description" :rows="3" />
            </a-form-item>
            <a-row :gutter="16">
              <a-col :span="8">
                <a-form-item :label="t('admin.tenants.contact')">
                  <a-input v-model:value="selectedTenant.contact_name" :placeholder="t('admin.tenants.contactNamePlaceholder')">
                    <template #prefix><UserOutlined /></template>
                  </a-input>
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item :label="t('admin.tenants.contactEmail')">
                  <a-input v-model:value="selectedTenant.contact_email" :placeholder="t('admin.tenants.contactEmailPlaceholder')">
                    <template #prefix><MailOutlined /></template>
                  </a-input>
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item :label="t('admin.tenants.contactPhone')">
                  <a-input v-model:value="selectedTenant.contact_phone" :placeholder="t('admin.tenants.contactPhonePlaceholder')">
                    <template #prefix><PhoneOutlined /></template>
                  </a-input>
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item :label="t('admin.tenants.createdDate')">
                  <a-input :value="selectedTenant.created_at" size="large" disabled />
                </a-form-item>
              </a-col>
            </a-row>
          </a-form>
        </div>

        <!--OA 数据库连接选项卡-->
        <div v-if="detailActiveTab === 'oadb'" class="detail-section">
          <div class="section-header">
            <h3><DatabaseOutlined /> {{ t('admin.tenants.oaDbConfig') }}</h3>
          </div>
          <div class="jdbc-hint">
            <InfoCircleOutlined /> {{ t('admin.tenants.oaDbSelectHint') }}
            <a @click="navigateTo('/admin/system/settings')" style="cursor: pointer; margin: 0 4px;">{{ t('admin.tenants.systemSettings') }}</a>
          </div>
          <a-form layout="vertical">
            <a-form-item :label="t('admin.tenants.oaDbConnection')">
              <a-select v-model:value="selectedTenant.oa_db_connection_id" size="large" :placeholder="t('admin.tenants.selectOADb')" allowClear>
                <a-select-option v-for="conn in availableOADbs" :key="conn.id" :value="conn.id">
                  {{ conn.name }} ({{ conn.oa_type_label }})
                </a-select-option>
              </a-select>
            </a-form-item>

            <!--显示所选连接详细信息（只读）-->
            <div v-if="selectedTenant.oa_db_connection_id && getOADbInfo(selectedTenant.oa_db_connection_id)" class="oadb-detail-card">
              <div class="oadb-detail-header">
                <LinkOutlined />
                <span>{{ getOADbInfo(selectedTenant.oa_db_connection_id)!.name }}</span>
                <span class="oadb-detail-type">{{ getOADbInfo(selectedTenant.oa_db_connection_id)!.oa_type_label }}</span>
              </div>
              <div class="oadb-detail-meta">
                <div class="oadb-meta-item">
                  <span class="oadb-meta-label">{{ t('admin.tenants.dbDriver') }}</span>
                  <span class="oadb-meta-value">{{ getOADbInfo(selectedTenant.oa_db_connection_id)!.jdbc_config.driver.toUpperCase() }}</span>
                </div>
                <div class="oadb-meta-item">
                  <span class="oadb-meta-label">{{ t('admin.tenants.hostAddress') }}</span>
                  <span class="oadb-meta-value">{{ getOADbInfo(selectedTenant.oa_db_connection_id)!.jdbc_config.host }}:{{ getOADbInfo(selectedTenant.oa_db_connection_id)!.jdbc_config.port }}</span>
                </div>
                <div class="oadb-meta-item">
                  <span class="oadb-meta-label">{{ t('admin.tenants.dbName') }}</span>
                  <span class="oadb-meta-value">{{ getOADbInfo(selectedTenant.oa_db_connection_id)!.jdbc_config.database }}</span>
                </div>
                <div class="oadb-meta-item">
                  <span class="oadb-meta-label">{{ t('admin.settings.syncInterval') }}</span>
                  <span class="oadb-meta-value">{{ getOADbInfo(selectedTenant.oa_db_connection_id)!.sync_interval }}s</span>
                </div>
              </div>
              <div v-if="getOADbInfo(selectedTenant.oa_db_connection_id)!.description" class="oadb-detail-desc">
                {{ getOADbInfo(selectedTenant.oa_db_connection_id)!.description }}
              </div>
            </div>

            <div v-else-if="!selectedTenant.oa_db_connection_id" class="oadb-empty">
              <InfoCircleOutlined /> {{ t('admin.tenants.noOADbSelected') }}
            </div>
          </a-form>
        </div>

        <!--AI模型选项卡-->
        <div v-if="detailActiveTab === 'ai'" class="detail-section">
          <div class="section-header">
            <h3><RobotOutlined /> {{ t('admin.tenants.aiModelSelect') }}</h3>
          </div>
          <div class="jdbc-hint">
            <InfoCircleOutlined /> {{ t('admin.tenants.aiModelHint') }}<a @click="navigateTo('/admin/system/settings')" style="cursor: pointer; margin: 0 4px;">{{ t('admin.tenants.systemSettings') }}</a>)
          </div>
          <a-form layout="vertical">
            <div class="config-group">
              <div class="config-group-title">{{ t('admin.tenants.primaryModel') }}</div>
              <a-row :gutter="16">
                <a-col :span="12">
                  <a-form-item :label="t('admin.tenants.aiProvider')">
                    <a-select v-model:value="selectedTenant.ai_config.default_provider" size="large" :placeholder="t('admin.tenants.selectProvider')" @change="onPrimaryProviderChange">
                      <a-select-option v-for="p in providerOptions" :key="p" :value="p">{{ p }}</a-select-option>
                    </a-select>
                  </a-form-item>
                </a-col>
                <a-col :span="12">
                  <a-form-item :label="t('admin.tenants.modelName')">
                    <a-select v-model:value="selectedTenant.ai_config.default_model" size="large" :placeholder="t('admin.tenants.selectModel')">
                      <a-select-option v-for="m in primaryFilteredModels" :key="m.model_name" :value="m.model_name">
                        {{ m.display_name }}
                      </a-select-option>
                    </a-select>
                  </a-form-item>
                </a-col>
              </a-row>
            </div>

            <div class="config-group">
              <div class="config-group-title">{{ t('admin.tenants.fallbackModel') }}</div>
              <a-row :gutter="16">
                <a-col :span="12">
                  <a-form-item :label="t('admin.tenants.fallbackProvider')">
                    <a-select v-model:value="selectedTenant.ai_config.fallback_provider" size="large" allowClear :placeholder="t('admin.tenants.noConfig')" @change="onFallbackProviderChange">
                      <a-select-option v-for="p in providerOptions" :key="p" :value="p">{{ p }}</a-select-option>
                    </a-select>
                  </a-form-item>
                </a-col>
                <a-col :span="12">
                  <a-form-item :label="t('admin.tenants.fallbackModelLabel')">
                    <a-select v-model:value="selectedTenant.ai_config.fallback_model" size="large" allowClear :placeholder="t('admin.tenants.noConfig')">
                      <a-select-option v-for="m in fallbackFilteredModels" :key="m.model_name" :value="m.model_name">
                        {{ m.display_name }}
                      </a-select-option>
                    </a-select>
                  </a-form-item>
                </a-col>
              </a-row>
            </div>

            <a-divider>{{ t('admin.tenants.callParams') }}</a-divider>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item :label="t('admin.tenants.maxTokenPerReq')">
                  <a-input-number v-model:value="selectedTenant.ai_config.max_tokens_per_request" :min="512" :max="32768" :step="512" style="width: 100%;" size="large" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item :label="t('admin.tenants.temperature')">
                  <a-slider v-model:value="selectedTenant.ai_config.temperature" :min="0" :max="1" :step="0.1" />
                  <span class="slider-value">{{ selectedTenant.ai_config.temperature }}</span>
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item :label="t('admin.tenants.timeout')">
                  <a-input-number v-model:value="selectedTenant.ai_config.timeout_seconds" :min="10" :max="300" style="width: 100%;" size="large" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item :label="t('admin.tenants.retryCount')">
                  <a-input-number v-model:value="selectedTenant.ai_config.retry_count" :min="0" :max="10" style="width: 100%;" size="large" />
                </a-form-item>
              </a-col>
            </a-row>
            <a-form-item :label="t('admin.tenants.allowCustomModel')">
              <a-switch v-model:checked="selectedTenant.allow_custom_model" />
              <span class="switch-label">{{ selectedTenant.allow_custom_model ? t('admin.tenants.allowCustomModelDesc') : t('admin.tenants.onlyDefaultModel') }}</span>
            </a-form-item>
          </a-form>
        </div>

        <!--配额和政策选项卡-->
        <div v-if="detailActiveTab === 'quota'" class="detail-section">
          <div class="section-header">
            <h3><ThunderboltOutlined /> {{ t('admin.tenants.quotaPolicy') }}</h3>
          </div>
          <a-form layout="vertical">
            <div class="config-group">
              <div class="config-group-title">{{ t('admin.tenants.resourceQuota') }}</div>
              <a-row :gutter="16">
                <a-col :span="12">
                  <a-form-item :label="t('admin.tenants.tokenQuota')">
                    <a-input-number v-model:value="selectedTenant.token_quota" :min="1000" :step="1000" style="width: 100%;" size="large" />
                  </a-form-item>
                </a-col>
                <a-col :span="12">
                  <a-form-item :label="t('admin.tenants.maxConcurrency')">
                    <a-input-number v-model:value="selectedTenant.max_concurrency" :min="1" :max="100" style="width: 100%;" size="large" />
                  </a-form-item>
                </a-col>
              </a-row>
              <!--当前使用情况显示-->
              <div class="usage-display">
                <div class="usage-info">
                  <span>{{ t('admin.tenants.usedTokens', [selectedTenant.token_used.toLocaleString(), selectedTenant.token_quota.toLocaleString()]) }}</span>
                  <span :style="{ color: getQuotaColor(getQuotaPercent(selectedTenant.token_used, selectedTenant.token_quota)) }">
                    {{ getQuotaPercent(selectedTenant.token_used, selectedTenant.token_quota) }}%
                  </span>
                </div>
                <div class="quota-bar" style="height: 8px;">
                  <div
                    class="quota-bar-fill"
                    :style="{
                      width: getQuotaPercent(selectedTenant.token_used, selectedTenant.token_quota) + '%',
                      background: getQuotaColor(getQuotaPercent(selectedTenant.token_used, selectedTenant.token_quota)),
                    }"
                  />
                </div>
              </div>
            </div>

            <div class="config-group">
              <div class="config-group-title">{{ t('admin.tenants.dataRetention') }}</div>
              <a-row :gutter="16">
                <a-col :span="12">
                  <a-form-item :label="t('admin.tenants.logRetention')">
                    <a-input-number v-model:value="selectedTenant.log_retention_days" :min="7" :max="3650" style="width: 100%;" size="large" />
                    <div class="form-hint">{{ t('admin.tenants.logRetentionHint') }}</div>
                  </a-form-item>
                </a-col>
                <a-col :span="12">
                  <a-form-item :label="t('admin.tenants.auditDataRetention')">
                    <a-input-number v-model:value="selectedTenant.data_retention_days" :min="30" :max="3650" style="width: 100%;" size="large" />
                    <div class="form-hint">{{ t('admin.tenants.auditDataRetentionHint') }}</div>
                  </a-form-item>
                </a-col>
              </a-row>
            </div>
          </a-form>
        </div>

        <!--安全选项卡-->
        <div v-if="detailActiveTab === 'security'" class="detail-section">
          <div class="section-header">
            <h3><SafetyCertificateOutlined /> {{ t('admin.tenants.securitySettings') }}</h3>
          </div>
          <a-form layout="vertical">
            <div class="config-group">
              <div class="config-group-title">{{ t('admin.tenants.sso') }}</div>
              <a-form-item :label="t('admin.tenants.enableSSO')">
                <a-switch v-model:checked="selectedTenant.sso_enabled" />
                <span class="switch-label">{{ selectedTenant.sso_enabled ? t('admin.tenants.ssoEnabled') : t('admin.tenants.ssoDisabled') }}</span>
              </a-form-item>
              <a-form-item v-if="selectedTenant.sso_enabled" :label="t('admin.tenants.ssoEndpoint')">
                <a-input v-model:value="selectedTenant.sso_endpoint" placeholder="https://sso.example.com/oauth2" size="large" />
              </a-form-item>
            </div>

            <div class="config-group">
              <div class="config-group-title">{{ t('admin.tenants.tenantStatus') }}</div>
              <div class="status-display">
                <div class="status-info">
                  <span>{{ t('admin.tenants.currentStatus') }}</span>
                  <a-tag :color="selectedTenant.status === 'active' ? 'green' : 'default'">
                    {{ selectedTenant.status === 'active' ? t('admin.tenants.running') : t('admin.tenants.stopped') }}
                  </a-tag>
                </div>
                <a-button
                  :danger="selectedTenant.status === 'active'"
                  @click="toggleTenantStatus(selectedTenant.id); selectedTenant.status = selectedTenant.status === 'active' ? 'inactive' : 'active'"
                >
                  {{ selectedTenant.status === 'active' ? t('admin.tenants.disableTenant') : t('admin.tenants.enableTenant') }}
                </a-button>
              </div>
            </div>
          </a-form>
        </div>

        <!--页脚操作-->
        <div class="detail-footer">
          <a-button @click="showDetail = false">{{ t('admin.tenants.cancel') }}</a-button>
          <a-button type="primary" @click="saveTenantDetail">{{ t('admin.tenants.saveConfig') }}</a-button>
        </div>
      </template>
    </a-drawer>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 28px;
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

/*租户网格*/
.tenant-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
}
@media (max-width: 900px) {
  .tenant-grid { grid-template-columns: 1fr; }
}

.tenant-card {
  background: var(--color-bg-card);
  border-radius: var(--radius-xl);
  border: 1px solid var(--color-border-light);
  padding: 22px;
  transition: all var(--transition-base);
  cursor: pointer;
}

.tenant-card:hover {
  box-shadow: var(--shadow-lg);
  transform: translateY(-3px);
  border-color: var(--color-primary);
}

.tenant-card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.tenant-avatar {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-lg);
  background: linear-gradient(135deg, var(--color-primary-bg), var(--color-primary-lighter));
  color: var(--color-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  flex-shrink: 0;
}

.tenant-info {
  flex: 1;
  min-width: 0;
}

.tenant-name {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.tenant-code {
  font-size: 12px;
  color: var(--color-text-tertiary);
  font-family: var(--font-mono);
}

.tenant-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 500;
  flex-shrink: 0;
  padding: 4px 10px;
  border-radius: var(--radius-full);
}

.tenant-status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
}

.tenant-status--active {
  color: var(--color-success);
  background: var(--color-success-bg);
}

.tenant-status--active .tenant-status-dot {
  background: var(--color-success);
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.2);
}

.tenant-status--inactive {
  color: var(--color-text-tertiary);
  background: var(--color-bg-hover);
}

.tenant-status--inactive .tenant-status-dot {
  background: var(--color-text-tertiary);
}

/*标签*/
.tenant-tags {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
  flex-wrap: wrap;
}

.info-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 500;
  padding: 3px 10px;
  border-radius: var(--radius-full);
}

.info-tag--primary {
  background: var(--color-primary-bg);
  color: var(--color-primary);
}

.info-tag--info {
  background: var(--color-info-bg);
  color: var(--color-info);
}

.info-tag--success {
  background: var(--color-success-bg);
  color: var(--color-success);
}

/*统计数据*/
.tenant-stats {
  display: flex;
  gap: 24px;
  margin-bottom: 12px;
}

.stat-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.stat-label {
  font-size: 11px;
  color: var(--color-text-tertiary);
}

.stat-value {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}

/*配额栏*/
.quota-bar-wrapper {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
}

.quota-bar {
  flex: 1;
  height: 6px;
  background: var(--color-bg-hover);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.quota-bar-fill {
  height: 100%;
  border-radius: var(--radius-full);
  transition: width 0.5s ease;
}

.quota-percent {
  font-size: 12px;
  font-weight: 600;
  min-width: 36px;
  text-align: right;
}

.tenant-card-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 14px;
  padding-top: 14px;
  border-top: 1px solid var(--color-border-light);
}

.tenant-created {
  font-size: 12px;
  color: var(--color-text-tertiary);
  display: flex;
  align-items: center;
  gap: 4px;
}

.tenant-card-actions {
  display: flex;
  gap: 4px;
}

/*=====细节抽屉=====*/
.detail-tabs {
  display: flex;
  gap: 4px;
  background: var(--color-bg-hover);
  padding: 4px;
  border-radius: var(--radius-lg);
  margin-bottom: 24px;
  flex-wrap: wrap;
}

.detail-tab-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border: none;
  background: transparent;
  border-radius: var(--radius-md);
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
  white-space: nowrap;
}

.detail-tab-btn:hover {
  color: var(--color-text-primary);
}

.detail-tab-btn--active {
  background: var(--color-bg-card);
  color: var(--color-primary);
  box-shadow: var(--shadow-xs);
}

.detail-section {
  animation: fadeIn 0.2s ease;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.section-header h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.jdbc-hint {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--color-info);
  background: var(--color-info-bg);
  padding: 10px 14px;
  border-radius: var(--radius-md);
  margin-bottom: 20px;
}

.config-group {
  background: var(--color-bg-page);
  border-radius: var(--radius-lg);
  padding: 16px 20px;
  margin-bottom: 16px;
}

.config-group-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-secondary);
  margin-bottom: 12px;
}

.switch-label {
  font-size: 13px;
  color: var(--color-text-tertiary);
  margin-left: 10px;
}

.slider-value {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-primary);
  margin-left: 8px;
}

.form-hint {
  font-size: 11px;
  color: var(--color-text-tertiary);
  margin-top: 4px;
}

.usage-display {
  background: var(--color-bg-card);
  border-radius: var(--radius-md);
  padding: 14px;
  border: 1px solid var(--color-border-light);
}

.usage-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 13px;
  color: var(--color-text-secondary);
  margin-bottom: 8px;
}

.status-display {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--color-bg-card);
  border-radius: var(--radius-md);
  padding: 14px;
  border: 1px solid var(--color-border-light);
}

.status-info {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--color-text-secondary);
}

.detail-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 32px;
  padding-top: 20px;
  border-top: 1px solid var(--color-border-light);
}

/*租户抽屉中的 OA DB 详细信息卡*/
.oadb-detail-card {
  background: var(--color-bg-page);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-lg);
  padding: 16px 20px;
  margin-top: 12px;
}
.oadb-detail-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 12px;
}
.oadb-detail-type {
  font-size: 12px;
  font-weight: 500;
  color: var(--color-primary);
  background: var(--color-primary-bg);
  padding: 2px 8px;
  border-radius: var(--radius-full);
}
.oadb-detail-meta {
  display: flex;
  gap: 20px;
  flex-wrap: wrap;
  padding: 10px 14px;
  background: var(--color-bg-card);
  border-radius: var(--radius-md);
  margin-bottom: 8px;
}
.oadb-meta-item { }
.oadb-meta-label { font-size: 11px; color: var(--color-text-tertiary); display: block; }
.oadb-meta-value { font-size: 13px; font-weight: 500; color: var(--color-text-primary); margin-top: 2px; display: block; }
.oadb-detail-desc { font-size: 12px; color: var(--color-text-tertiary); margin-top: 8px; }
.oadb-empty {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--color-text-tertiary);
  padding: 20px;
  text-align: center;
  justify-content: center;
  background: var(--color-bg-page);
  border-radius: var(--radius-md);
  margin-top: 12px;
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }

  .tenant-grid {
    grid-template-columns: 1fr;
  }

  .detail-tabs {
    flex-wrap: nowrap;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
    scrollbar-width: none;
  }
  .detail-tabs::-webkit-scrollbar { display: none; }
}

@media (max-width: 480px) {
  .page-title { font-size: 20px; }
}
</style>
