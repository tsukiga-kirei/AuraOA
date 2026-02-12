<script setup lang="ts">
import {
  PlusOutlined,
  TeamOutlined,
  EditOutlined,
  DatabaseOutlined,
  RobotOutlined,
  SafetyCertificateOutlined,
  SyncOutlined,
  InfoCircleOutlined,
  KeyOutlined,
  ClockCircleOutlined,
  MailOutlined,
  PhoneOutlined,
  UserOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type { TenantInfo } from '~/composables/useMockData'

const { mockTenants, mockAIModelConfigs } = useMockData()

const tenants = ref<TenantInfo[]>([...mockTenants])
const selectedTenant = ref<TenantInfo | null>(null)
const showCreate = ref(false)
const showDetail = ref(false)
const detailActiveTab = ref('basic')
const testingConnection = ref(false)

// Available AI models for tenant config dropdowns
const availableModels = computed(() => mockAIModelConfigs.filter(m => m.enabled))

const newTenant = ref({
  name: '',
  code: '',
  oa_type: 'weaver_e9',
  token_quota: 10000,
  max_concurrency: 10,
  contact_name: '',
  contact_email: '',
  description: '',
})

const createTenant = () => {
  if (!newTenant.value.name || !newTenant.value.code) {
    message.warning('请填写租户名称和租户编码')
    return
  }
  const t: TenantInfo = {
    id: `T-${Date.now()}`,
    name: newTenant.value.name,
    code: newTenant.value.code,
    oa_type: newTenant.value.oa_type,
    token_quota: newTenant.value.token_quota,
    token_used: 0,
    max_concurrency: newTenant.value.max_concurrency,
    status: 'active',
    created_at: new Date().toISOString().slice(0, 10),
    contact_name: newTenant.value.contact_name,
    contact_email: newTenant.value.contact_email,
    contact_phone: '',
    description: newTenant.value.description,
    jdbc_config: {
      driver: 'mysql', host: '', port: 3306, database: '',
      username: '', password: '', pool_size: 10,
      connection_timeout: 30, test_on_borrow: true,
    },
    ai_config: {
      default_provider: '本地部署', default_model: 'Qwen2.5-72B',
      fallback_provider: '', fallback_model: '',
      max_tokens_per_request: 4096, temperature: 0.3,
      timeout_seconds: 60, retry_count: 2,
    },
    log_retention_days: 180,
    data_retention_days: 730,
    allow_custom_model: false,
    sso_enabled: false,
    sso_endpoint: '',
  }
  tenants.value.push(t)
  showCreate.value = false
  message.success('租户创建成功')
  newTenant.value = { name: '', code: '', oa_type: 'weaver_e9', token_quota: 10000, max_concurrency: 10, contact_name: '', contact_email: '', description: '' }
  // Auto-open the new tenant for configuration
  openDetail(t)
}

const openDetail = (tenant: TenantInfo) => {
  selectedTenant.value = { ...tenant, jdbc_config: { ...tenant.jdbc_config }, ai_config: { ...tenant.ai_config } }
  detailActiveTab.value = 'basic'
  showDetail.value = true
}

const saveTenantDetail = () => {
  if (!selectedTenant.value) return
  const idx = tenants.value.findIndex(t => t.id === selectedTenant.value!.id)
  if (idx >= 0) {
    tenants.value[idx] = { ...selectedTenant.value }
  }
  showDetail.value = false
  message.success('租户配置已保存')
}

const toggleTenantStatus = (id: string) => {
  const t = tenants.value.find(t => t.id === id)
  if (t) {
    t.status = t.status === 'active' ? 'inactive' : 'active'
    message.success(t.status === 'active' ? '已启用' : '已停用')
  }
}

const testConnection = async () => {
  testingConnection.value = true
  await new Promise(resolve => setTimeout(resolve, 2000))
  testingConnection.value = false
  if (selectedTenant.value?.jdbc_config.host) {
    message.success('数据库连接测试成功')
  } else {
    message.error('请先配置数据库连接参数')
  }
}

const getQuotaPercent = (used: number, total: number) => Math.round((used / total) * 100)

const getQuotaColor = (percent: number) => {
  if (percent >= 90) return '#ef4444'
  if (percent >= 70) return '#f59e0b'
  return '#10b981'
}

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

const onDriverChange = (driver: string) => {
  if (selectedTenant.value) {
    selectedTenant.value.jdbc_config.port = getDriverPort(driver)
  }
}
</script>

<template>
  <div class="system-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">租户管理</h1>
        <p class="page-subtitle">管理租户开通、数据库连接、AI模型配置与资源配额</p>
      </div>
      <a-button type="primary" size="large" @click="showCreate = true">
        <PlusOutlined /> 新增租户
      </a-button>
    </div>

    <!-- Tenant Cards Grid -->
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
            {{ tenant.status === 'active' ? '运行中' : '已停用' }}
          </div>
        </div>

        <!-- Quick Info Tags -->
        <div class="tenant-tags">
          <span class="info-tag info-tag--primary">
            <DatabaseOutlined /> {{ tenant.jdbc_config.driver.toUpperCase() }}
          </span>
          <span class="info-tag info-tag--info">
            <RobotOutlined /> {{ tenant.ai_config.default_model }}
          </span>
          <span v-if="tenant.sso_enabled" class="info-tag info-tag--success">
            <SafetyCertificateOutlined /> SSO
          </span>
        </div>

        <!-- Stats Row -->
        <div class="tenant-stats">
          <div class="stat-item">
            <span class="stat-label">Token 用量</span>
            <span class="stat-value">
              {{ (tenant.token_used / 1000).toFixed(1) }}K / {{ (tenant.token_quota / 1000).toFixed(0) }}K
            </span>
          </div>
          <div class="stat-item">
            <span class="stat-label">最大并发</span>
            <span class="stat-value">{{ tenant.max_concurrency }}</span>
          </div>
        </div>

        <!-- Token usage bar -->
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
              <EditOutlined /> 配置
            </a-button>
            <a-button size="small" type="text" @click="toggleTenantStatus(tenant.id)">
              {{ tenant.status === 'active' ? '停用' : '启用' }}
            </a-button>
          </div>
        </div>
      </div>
    </div>

    <!-- Create Tenant Modal -->
    <a-modal v-model:open="showCreate" title="新增租户" @ok="createTenant" okText="创建" cancelText="取消" width="560px">
      <a-form layout="vertical" style="margin-top: 16px;">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="租户名称" required>
              <a-input v-model:value="newTenant.name" placeholder="如：XX集团总部" size="large" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="租户编码" required>
              <a-input v-model:value="newTenant.code" placeholder="如：HQ_001" size="large" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="OA 类型">
          <a-select v-model:value="newTenant.oa_type" size="large">
            <a-select-option value="weaver_e9">泛微 E9</a-select-option>
            <a-select-option value="weaver_ebridge">泛微 E-Bridge</a-select-option>
            <a-select-option value="zhiyuan_a8">致远 A8+</a-select-option>
            <a-select-option value="landray_ekp">蓝凌 EKP</a-select-option>
          </a-select>
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="Token 配额">
              <a-input-number v-model:value="newTenant.token_quota" :min="1000" :step="1000" style="width: 100%;" size="large" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="最大并发数">
              <a-input-number v-model:value="newTenant.max_concurrency" :min="1" :max="100" style="width: 100%;" size="large" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="联系人">
              <a-input v-model:value="newTenant.contact_name" placeholder="管理员姓名" size="large" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="联系邮箱">
              <a-input v-model:value="newTenant.contact_email" placeholder="admin@example.com" size="large" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="描述">
          <a-textarea v-model:value="newTenant.description" :rows="2" placeholder="租户用途简述" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Tenant Detail Drawer -->
    <a-drawer
      v-model:open="showDetail"
      :title="selectedTenant?.name + ' — 租户配置'"
      placement="right"
      :width="720"
      @close="showDetail = false"
    >
      <template v-if="selectedTenant">
        <div class="detail-tabs">
          <button
            v-for="tab in [
              { key: 'basic', label: '基本信息', icon: InfoCircleOutlined },
              { key: 'jdbc', label: '数据库连接', icon: DatabaseOutlined },
              { key: 'ai', label: 'AI 模型', icon: RobotOutlined },
              { key: 'quota', label: '配额与策略', icon: ThunderboltOutlined },
              { key: 'security', label: '安全设置', icon: SafetyCertificateOutlined },
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

        <!-- Basic Info Tab -->
        <div v-if="detailActiveTab === 'basic'" class="detail-section">
          <div class="section-header">
            <h3><UserOutlined /> 基本信息</h3>
          </div>
          <a-form layout="vertical">
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="租户名称">
                  <a-input v-model:value="selectedTenant.name" size="large" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="租户编码">
                  <a-input v-model:value="selectedTenant.code" size="large" disabled />
                </a-form-item>
              </a-col>
            </a-row>
            <a-form-item label="描述">
              <a-textarea v-model:value="selectedTenant.description" :rows="3" />
            </a-form-item>
            <a-row :gutter="16">
              <a-col :span="8">
                <a-form-item label="联系人">
                  <a-input v-model:value="selectedTenant.contact_name">
                    <template #prefix><UserOutlined /></template>
                  </a-input>
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="联系邮箱">
                  <a-input v-model:value="selectedTenant.contact_email">
                    <template #prefix><MailOutlined /></template>
                  </a-input>
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="联系电话">
                  <a-input v-model:value="selectedTenant.contact_phone">
                    <template #prefix><PhoneOutlined /></template>
                  </a-input>
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="OA 类型">
                  <a-select v-model:value="selectedTenant.oa_type" size="large">
                    <a-select-option value="weaver_e9">泛微 E9</a-select-option>
                    <a-select-option value="weaver_ebridge">泛微 E-Bridge</a-select-option>
                    <a-select-option value="zhiyuan_a8">致远 A8+</a-select-option>
                    <a-select-option value="landray_ekp">蓝凌 EKP</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="创建日期">
                  <a-input :value="selectedTenant.created_at" size="large" disabled />
                </a-form-item>
              </a-col>
            </a-row>
          </a-form>
        </div>

        <!-- JDBC Config Tab -->
        <div v-if="detailActiveTab === 'jdbc'" class="detail-section">
          <div class="section-header">
            <h3><DatabaseOutlined /> 数据库连接配置</h3>
            <a-button type="primary" ghost :loading="testingConnection" @click="testConnection">
              <SyncOutlined /> 测试连接
            </a-button>
          </div>
          <div class="jdbc-hint">
            <InfoCircleOutlined /> 配置当前租户 OA 系统的数据库连接信息，用于流程数据同步
          </div>
          <a-form layout="vertical">
            <a-form-item label="数据库驱动">
              <a-select
                v-model:value="selectedTenant.jdbc_config.driver"
                size="large"
                @change="onDriverChange"
              >
                <a-select-option v-for="opt in driverOptions" :key="opt.value" :value="opt.value">
                  {{ opt.label }}
                </a-select-option>
              </a-select>
            </a-form-item>
            <a-row :gutter="16">
              <a-col :span="16">
                <a-form-item label="主机地址">
                  <a-input v-model:value="selectedTenant.jdbc_config.host" placeholder="192.168.1.100 或 db.example.com" size="large" />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="端口">
                  <a-input-number v-model:value="selectedTenant.jdbc_config.port" :min="1" :max="65535" style="width: 100%;" size="large" />
                </a-form-item>
              </a-col>
            </a-row>
            <a-form-item label="数据库名称">
              <a-input v-model:value="selectedTenant.jdbc_config.database" placeholder="ecology" size="large" />
            </a-form-item>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="用户名">
                  <a-input v-model:value="selectedTenant.jdbc_config.username" placeholder="oa_reader" size="large">
                    <template #prefix><UserOutlined /></template>
                  </a-input>
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="密码">
                  <a-input-password v-model:value="selectedTenant.jdbc_config.password" placeholder="数据库密码" size="large">
                    <template #prefix><KeyOutlined /></template>
                  </a-input-password>
                </a-form-item>
              </a-col>
            </a-row>
            <a-divider>连接池设置</a-divider>
            <a-row :gutter="16">
              <a-col :span="8">
                <a-form-item label="连接池大小">
                  <a-input-number v-model:value="selectedTenant.jdbc_config.pool_size" :min="1" :max="100" style="width: 100%;" size="large" />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="连接超时(秒)">
                  <a-input-number v-model:value="selectedTenant.jdbc_config.connection_timeout" :min="5" :max="300" style="width: 100%;" size="large" />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="借用时测试">
                  <a-switch v-model:checked="selectedTenant.jdbc_config.test_on_borrow" />
                  <span class="switch-label">{{ selectedTenant.jdbc_config.test_on_borrow ? '已开启' : '已关闭' }}</span>
                </a-form-item>
              </a-col>
            </a-row>
          </a-form>
        </div>

        <!-- AI Model Tab -->
        <div v-if="detailActiveTab === 'ai'" class="detail-section">
          <div class="section-header">
            <h3><RobotOutlined /> AI 模型选择</h3>
          </div>
          <div class="jdbc-hint">
            <InfoCircleOutlined /> 从系统设置中已注册的模型中选择当前租户使用的 AI 模型（模型注册请到<a @click="navigateTo('/admin/system/settings')" style="cursor: pointer; margin: 0 4px;">系统设置</a>）
          </div>
          <a-form layout="vertical">
            <div class="config-group">
              <div class="config-group-title">主模型</div>
              <a-row :gutter="16">
                <a-col :span="12">
                  <a-form-item label="AI 服务商">
                    <a-select v-model:value="selectedTenant.ai_config.default_provider" size="large">
                      <a-select-option value="本地部署">本地部署</a-select-option>
                      <a-select-option value="云端API">云端API</a-select-option>
                    </a-select>
                  </a-form-item>
                </a-col>
                <a-col :span="12">
                  <a-form-item label="模型名称">
                    <a-select v-model:value="selectedTenant.ai_config.default_model" size="large">
                      <a-select-option v-for="m in availableModels" :key="m.model_name" :value="m.model_name">
                        {{ m.display_name }}
                      </a-select-option>
                    </a-select>
                  </a-form-item>
                </a-col>
              </a-row>
            </div>

            <div class="config-group">
              <div class="config-group-title">备用模型（主模型不可用时自动切换）</div>
              <a-row :gutter="16">
                <a-col :span="12">
                  <a-form-item label="备用服务商">
                    <a-select v-model:value="selectedTenant.ai_config.fallback_provider" size="large" allowClear placeholder="不配置">
                      <a-select-option value="本地部署">本地部署</a-select-option>
                      <a-select-option value="云端API">云端API</a-select-option>
                    </a-select>
                  </a-form-item>
                </a-col>
                <a-col :span="12">
                  <a-form-item label="备用模型">
                    <a-select v-model:value="selectedTenant.ai_config.fallback_model" size="large" allowClear placeholder="不配置">
                      <a-select-option v-for="m in availableModels" :key="m.model_name" :value="m.model_name">
                        {{ m.display_name }}
                      </a-select-option>
                    </a-select>
                  </a-form-item>
                </a-col>
              </a-row>
            </div>

            <a-divider>调用参数</a-divider>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="最大Token/请求">
                  <a-input-number v-model:value="selectedTenant.ai_config.max_tokens_per_request" :min="512" :max="32768" :step="512" style="width: 100%;" size="large" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="温度 (Temperature)">
                  <a-slider v-model:value="selectedTenant.ai_config.temperature" :min="0" :max="1" :step="0.1" />
                  <span class="slider-value">{{ selectedTenant.ai_config.temperature }}</span>
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="超时时间(秒)">
                  <a-input-number v-model:value="selectedTenant.ai_config.timeout_seconds" :min="10" :max="300" style="width: 100%;" size="large" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="重试次数">
                  <a-input-number v-model:value="selectedTenant.ai_config.retry_count" :min="0" :max="10" style="width: 100%;" size="large" />
                </a-form-item>
              </a-col>
            </a-row>
            <a-form-item label="允许用户自选模型">
              <a-switch v-model:checked="selectedTenant.allow_custom_model" />
              <span class="switch-label">{{ selectedTenant.allow_custom_model ? '允许租户内用户在流程配置中覆盖默认模型' : '仅使用租户默认模型' }}</span>
            </a-form-item>
          </a-form>
        </div>

        <!-- Quota & Policy Tab -->
        <div v-if="detailActiveTab === 'quota'" class="detail-section">
          <div class="section-header">
            <h3><ThunderboltOutlined /> 配额与策略</h3>
          </div>
          <a-form layout="vertical">
            <div class="config-group">
              <div class="config-group-title">资源配额</div>
              <a-row :gutter="16">
                <a-col :span="12">
                  <a-form-item label="Token 配额">
                    <a-input-number v-model:value="selectedTenant.token_quota" :min="1000" :step="1000" style="width: 100%;" size="large" />
                  </a-form-item>
                </a-col>
                <a-col :span="12">
                  <a-form-item label="最大并发数">
                    <a-input-number v-model:value="selectedTenant.max_concurrency" :min="1" :max="100" style="width: 100%;" size="large" />
                  </a-form-item>
                </a-col>
              </a-row>
              <!-- Current usage display -->
              <div class="usage-display">
                <div class="usage-info">
                  <span>已使用 {{ selectedTenant.token_used.toLocaleString() }} / {{ selectedTenant.token_quota.toLocaleString() }} Token</span>
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
              <div class="config-group-title">数据保留策略</div>
              <a-row :gutter="16">
                <a-col :span="12">
                  <a-form-item label="日志保留天数">
                    <a-input-number v-model:value="selectedTenant.log_retention_days" :min="7" :max="3650" style="width: 100%;" size="large" />
                    <div class="form-hint">AI推理日志、操作日志的保留时长</div>
                  </a-form-item>
                </a-col>
                <a-col :span="12">
                  <a-form-item label="审核数据保留天数">
                    <a-input-number v-model:value="selectedTenant.data_retention_days" :min="30" :max="3650" style="width: 100%;" size="large" />
                    <div class="form-hint">审核快照、推理过程等数据的保留时长</div>
                  </a-form-item>
                </a-col>
              </a-row>
            </div>
          </a-form>
        </div>

        <!-- Security Tab -->
        <div v-if="detailActiveTab === 'security'" class="detail-section">
          <div class="section-header">
            <h3><SafetyCertificateOutlined /> 安全设置</h3>
          </div>
          <a-form layout="vertical">
            <div class="config-group">
              <div class="config-group-title">单点登录 (SSO)</div>
              <a-form-item label="启用 SSO">
                <a-switch v-model:checked="selectedTenant.sso_enabled" />
                <span class="switch-label">{{ selectedTenant.sso_enabled ? '已启用单点登录' : '未启用' }}</span>
              </a-form-item>
              <a-form-item v-if="selectedTenant.sso_enabled" label="SSO 端点">
                <a-input v-model:value="selectedTenant.sso_endpoint" placeholder="https://sso.example.com/oauth2" size="large" />
              </a-form-item>
            </div>

            <div class="config-group">
              <div class="config-group-title">租户状态</div>
              <div class="status-display">
                <div class="status-info">
                  <span>当前状态：</span>
                  <a-tag :color="selectedTenant.status === 'active' ? 'green' : 'default'">
                    {{ selectedTenant.status === 'active' ? '运行中' : '已停用' }}
                  </a-tag>
                </div>
                <a-button
                  :danger="selectedTenant.status === 'active'"
                  @click="toggleTenantStatus(selectedTenant.id); selectedTenant.status = selectedTenant.status === 'active' ? 'inactive' : 'active'"
                >
                  {{ selectedTenant.status === 'active' ? '停用租户' : '启用租户' }}
                </a-button>
              </div>
            </div>
          </a-form>
        </div>

        <!-- Footer Actions -->
        <div class="detail-footer">
          <a-button @click="showDetail = false">取消</a-button>
          <a-button type="primary" @click="saveTenantDetail">保存配置</a-button>
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

/* Tenant grid */
.tenant-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(380px, 1fr));
  gap: 20px;
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

/* Tags */
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

/* Stats */
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

/* Quota bar */
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

/* ===== Detail Drawer ===== */
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

@media (max-width: 640px) {
  .page-header {
    flex-direction: column;
    gap: 12px;
  }

  .tenant-grid {
    grid-template-columns: 1fr;
  }

  .detail-tabs {
    flex-wrap: nowrap;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
  }
}
</style>
