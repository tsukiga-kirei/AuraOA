<script setup lang="ts">
import {
  LinkOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ToolOutlined,
  RobotOutlined,
  CloudServerOutlined,
  SettingOutlined,
  SaveOutlined,
  SyncOutlined,
  InfoCircleOutlined,
  MailOutlined,
  LockOutlined,
  DatabaseOutlined,
  ThunderboltOutlined,
  ClockCircleOutlined,
  GlobalOutlined,
  SafetyCertificateOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type { OASystemConfig, AIModelConfig, SystemGeneralConfig } from '~/composables/useMockData'

const { mockOASystemConfigs, mockAIModelConfigs, mockSystemGeneralConfig } = useMockData()

const activeTab = ref('oa')
const oaSystems = ref<OASystemConfig[]>([...mockOASystemConfigs])
const aiModels = ref<AIModelConfig[]>([...mockAIModelConfigs])
const generalConfig = ref<SystemGeneralConfig>({ ...mockSystemGeneralConfig })
const saving = ref(false)

// OA System Methods
const toggleOASystem = (id: string) => {
  const sys = oaSystems.value.find(s => s.id === id)
  if (sys) {
    sys.enabled = !sys.enabled
    message.success(sys.enabled ? `已启用 ${sys.name}` : `已停用 ${sys.name}`)
  }
}

const testOAConnection = async (id: string) => {
  const sys = oaSystems.value.find(s => s.id === id)
  if (!sys) return
  sys.status = 'testing'
  await new Promise(resolve => setTimeout(resolve, 2000))
  if (sys.enabled) {
    sys.status = 'connected'
    sys.last_sync = new Date().toLocaleString('zh-CN')
    message.success(`${sys.name} 连接测试成功`)
  } else {
    sys.status = 'disconnected'
    message.warning(`${sys.name} 未启用`)
  }
}

// AI Model Methods
const toggleAIModel = (id: string) => {
  const model = aiModels.value.find(m => m.id === id)
  if (model) {
    model.enabled = !model.enabled
    message.success(model.enabled ? `已启用 ${model.display_name}` : `已停用 ${model.display_name}`)
  }
}

const getStatusConfig = (status: string) => {
  const configs: Record<string, { color: string; bg: string; label: string; icon: any }> = {
    connected: { color: 'var(--color-success)', bg: 'var(--color-success-bg)', label: '已连接', icon: CheckCircleOutlined },
    disconnected: { color: 'var(--color-text-tertiary)', bg: 'var(--color-bg-hover)', label: '未连接', icon: CloseCircleOutlined },
    testing: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', label: '测试中', icon: SyncOutlined },
    online: { color: 'var(--color-success)', bg: 'var(--color-success-bg)', label: '在线', icon: CheckCircleOutlined },
    offline: { color: 'var(--color-text-tertiary)', bg: 'var(--color-bg-hover)', label: '离线', icon: CloseCircleOutlined },
    maintenance: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', label: '维护中', icon: ToolOutlined },
  }
  return configs[status] || configs.disconnected
}

const getModelTypeTag = (type: string) => {
  return type === 'local'
    ? { label: '本地部署', color: 'var(--color-success)', bg: 'var(--color-success-bg)' }
    : { label: '云端API', color: 'var(--color-info)', bg: 'var(--color-info-bg)' }
}

const saveGeneralConfig = async () => {
  saving.value = true
  await new Promise(resolve => setTimeout(resolve, 1000))
  saving.value = false
  message.success('系统配置已保存')
}

const enabledOASystems = computed(() => oaSystems.value.filter(s => s.enabled).length)
const enabledAIModels = computed(() => aiModels.value.filter(m => m.enabled).length)
const onlineAIModels = computed(() => aiModels.value.filter(m => m.status === 'online' && m.enabled).length)
</script>

<template>
  <div class="settings-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">系统设置</h1>
        <p class="page-subtitle">配置 OA 系统集成、AI 模型管理与平台全局参数</p>
      </div>
    </div>

    <!-- Overview Stats -->
    <div class="overview-stats">
      <div class="overview-stat">
        <div class="overview-stat-icon overview-stat-icon--primary">
          <LinkOutlined />
        </div>
        <div class="overview-stat-info">
          <div class="overview-stat-value">{{ enabledOASystems }} / {{ oaSystems.length }}</div>
          <div class="overview-stat-label">已启用 OA 系统</div>
        </div>
      </div>
      <div class="overview-stat">
        <div class="overview-stat-icon overview-stat-icon--success">
          <RobotOutlined />
        </div>
        <div class="overview-stat-info">
          <div class="overview-stat-value">{{ onlineAIModels }} / {{ enabledAIModels }}</div>
          <div class="overview-stat-label">在线 AI 模型</div>
        </div>
      </div>
      <div class="overview-stat">
        <div class="overview-stat-icon overview-stat-icon--info">
          <GlobalOutlined />
        </div>
        <div class="overview-stat-info">
          <div class="overview-stat-value">{{ generalConfig.platform_version }}</div>
          <div class="overview-stat-label">平台版本</div>
        </div>
      </div>
    </div>

    <!-- Tab Navigation -->
    <div class="tab-nav">
      <button
        v-for="tab in [
          { key: 'oa', label: 'OA 系统管理', icon: LinkOutlined },
          { key: 'ai', label: 'AI 模型管理', icon: RobotOutlined },
          { key: 'general', label: '平台配置', icon: SettingOutlined },
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

    <!-- OA Systems Tab -->
    <div v-if="activeTab === 'oa'" class="tab-content">
      <div class="tab-content-header">
        <p class="tab-desc">管理已适配和计划适配的 OA 系统，启用后可供租户选择并配置连接</p>
      </div>

      <div class="oa-grid">
        <div v-for="sys in oaSystems" :key="sys.id" class="oa-card" :class="{ 'oa-card--disabled': !sys.enabled }">
          <div class="oa-card-header">
            <div class="oa-card-icon" :class="{ 'oa-card-icon--active': sys.enabled }">
              <LinkOutlined />
            </div>
            <div class="oa-card-info">
              <h3 class="oa-card-name">{{ sys.name }}</h3>
              <span class="oa-card-version">{{ sys.type_label }} {{ sys.version }}</span>
            </div>
            <div class="oa-card-status" :style="{ color: getStatusConfig(sys.status).color, background: getStatusConfig(sys.status).bg }">
              <component :is="getStatusConfig(sys.status).icon" :spin="sys.status === 'testing'" />
              {{ getStatusConfig(sys.status).label }}
            </div>
          </div>

          <p class="oa-card-desc">{{ sys.description }}</p>

          <div class="oa-card-meta">
            <div class="oa-meta-item">
              <span class="oa-meta-label">适配器版本</span>
              <span class="oa-meta-value">{{ sys.adapter_version }}</span>
            </div>
            <div class="oa-meta-item">
              <span class="oa-meta-label">同步间隔</span>
              <span class="oa-meta-value">{{ sys.sync_interval }}s</span>
            </div>
            <div v-if="sys.last_sync" class="oa-meta-item">
              <span class="oa-meta-label">最后同步</span>
              <span class="oa-meta-value">{{ sys.last_sync }}</span>
            </div>
          </div>

          <div class="oa-card-actions">
            <a-switch
              :checked="sys.enabled"
              @change="toggleOASystem(sys.id)"
              checked-children="启用"
              un-checked-children="停用"
            />
            <a-button
              type="primary"
              ghost
              size="small"
              :loading="sys.status === 'testing'"
              :disabled="!sys.enabled"
              @click="testOAConnection(sys.id)"
            >
              <SyncOutlined /> 测试连接
            </a-button>
          </div>
        </div>
      </div>
    </div>

    <!-- AI Models Tab -->
    <div v-if="activeTab === 'ai'" class="tab-content">
      <div class="tab-content-header">
        <p class="tab-desc">管理平台可用的 AI 大模型，启用后可供租户和流程配置选用</p>
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

          <!-- Capabilities -->
          <div class="ai-capabilities">
            <span v-for="cap in model.capabilities" :key="cap" class="capability-tag">
              {{ cap === 'text' ? '文本' : cap === 'code' ? '代码' : cap === 'reasoning' ? '推理' : cap === 'vision' ? '视觉' : cap === 'analysis' ? '分析' : cap }}
            </span>
          </div>

          <div class="ai-card-meta">
            <div class="ai-meta-row">
              <div class="ai-meta-item">
                <span class="ai-meta-label">上下文窗口</span>
                <span class="ai-meta-value">{{ (model.context_window / 1024).toFixed(0) }}K</span>
              </div>
              <div class="ai-meta-item">
                <span class="ai-meta-label">最大Token</span>
                <span class="ai-meta-value">{{ (model.max_tokens / 1024).toFixed(0) }}K</span>
              </div>
              <div class="ai-meta-item">
                <span class="ai-meta-label">成本/千Token</span>
                <span class="ai-meta-value">{{ model.cost_per_1k_tokens > 0 ? '¥' + model.cost_per_1k_tokens.toFixed(2) : '免费' }}</span>
              </div>
            </div>
            <div class="ai-meta-row">
              <div class="ai-meta-item">
                <span class="ai-meta-label">端点</span>
                <span class="ai-meta-value ai-meta-value--mono">{{ model.endpoint }}</span>
              </div>
              <div class="ai-meta-item">
                <span class="ai-meta-label">API Key</span>
                <span class="ai-meta-value">
                  <CheckCircleOutlined v-if="model.api_key_configured" style="color: var(--color-success);" /> 
                  {{ model.api_key_configured ? '已配置' : model.type === 'local' ? '无需配置' : '未配置' }}
                </span>
              </div>
            </div>
          </div>

          <div class="ai-card-actions">
            <a-switch
              :checked="model.enabled"
              @change="toggleAIModel(model.id)"
              checked-children="启用"
              un-checked-children="停用"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- General Config Tab -->
    <div v-if="activeTab === 'general'" class="tab-content">
      <div class="tab-content-header">
        <p class="tab-desc">配置平台全局参数，包括安全策略、邮件通知和数据备份</p>
      </div>

      <div class="config-sections">
        <!-- Platform Info -->
        <div class="config-section">
          <div class="config-section-header">
            <div class="config-section-icon config-section-icon--primary">
              <GlobalOutlined />
            </div>
            <div>
              <h3>平台基本信息</h3>
              <p>平台名称、版本与语言设置</p>
            </div>
          </div>
          <a-form layout="vertical">
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="平台名称">
                  <a-input v-model:value="generalConfig.platform_name" size="large" />
                </a-form-item>
              </a-col>
              <a-col :span="6">
                <a-form-item label="版本号">
                  <a-input v-model:value="generalConfig.platform_version" size="large" disabled />
                </a-form-item>
              </a-col>
              <a-col :span="6">
                <a-form-item label="默认语言">
                  <a-select v-model:value="generalConfig.default_language" size="large">
                    <a-select-option value="zh-CN">简体中文</a-select-option>
                    <a-select-option value="en-US">English</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="会话超时(分钟)">
                  <a-input-number v-model:value="generalConfig.session_timeout" :min="5" :max="1440" style="width: 100%;" size="large" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="最大上传大小(MB)">
                  <a-input-number v-model:value="generalConfig.max_upload_size" :min="1" :max="500" style="width: 100%;" size="large" />
                </a-form-item>
              </a-col>
            </a-row>
          </a-form>
        </div>

        <!-- Security Settings -->
        <div class="config-section">
          <div class="config-section-header">
            <div class="config-section-icon config-section-icon--success">
              <SafetyCertificateOutlined />
            </div>
            <div>
              <h3>安全与合规</h3>
              <p>审计追踪、数据加密等安全策略</p>
            </div>
          </div>
          <div class="toggle-grid">
            <div class="toggle-item">
              <div class="toggle-info">
                <div class="toggle-label">审计追踪</div>
                <div class="toggle-desc">记录所有用户操作和系统事件</div>
              </div>
              <a-switch v-model:checked="generalConfig.enable_audit_trail" />
            </div>
            <div class="toggle-item">
              <div class="toggle-info">
                <div class="toggle-label">数据加密</div>
                <div class="toggle-desc">对敏感字段进行加密存储</div>
              </div>
              <a-switch v-model:checked="generalConfig.enable_data_encryption" />
            </div>
          </div>
        </div>

        <!-- Backup Settings -->
        <div class="config-section">
          <div class="config-section-header">
            <div class="config-section-icon config-section-icon--warning">
              <DatabaseOutlined />
            </div>
            <div>
              <h3>数据备份</h3>
              <p>自动备份策略与保留周期配置</p>
            </div>
          </div>
          <a-form layout="vertical">
            <div class="toggle-item" style="margin-bottom: 16px;">
              <div class="toggle-info">
                <div class="toggle-label">启用自动备份</div>
                <div class="toggle-desc">按 Cron 表达式定时执行数据库备份</div>
              </div>
              <a-switch v-model:checked="generalConfig.backup_enabled" />
            </div>
            <a-row v-if="generalConfig.backup_enabled" :gutter="16">
              <a-col :span="12">
                <a-form-item label="备份计划 (Cron)">
                  <a-input v-model:value="generalConfig.backup_cron" size="large" placeholder="0 2 * * *" />
                  <div class="form-hint">默认每天凌晨 2 点执行</div>
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="备份保留天数">
                  <a-input-number v-model:value="generalConfig.backup_retention_days" :min="1" :max="365" style="width: 100%;" size="large" />
                </a-form-item>
              </a-col>
            </a-row>
          </a-form>
        </div>

        <!-- Email / SMTP Settings -->
        <div class="config-section">
          <div class="config-section-header">
            <div class="config-section-icon config-section-icon--info">
              <MailOutlined />
            </div>
            <div>
              <h3>邮件通知</h3>
              <p>SMTP 服务器配置，用于系统通知和报告推送</p>
            </div>
          </div>
          <a-form layout="vertical">
            <a-form-item label="通知邮箱">
              <a-input v-model:value="generalConfig.notification_email" size="large" placeholder="admin@example.com" />
            </a-form-item>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="SMTP 主机">
                  <a-input v-model:value="generalConfig.smtp_host" size="large" placeholder="smtp.example.com" />
                </a-form-item>
              </a-col>
              <a-col :span="6">
                <a-form-item label="端口">
                  <a-input-number v-model:value="generalConfig.smtp_port" :min="1" :max="65535" style="width: 100%;" size="large" />
                </a-form-item>
              </a-col>
              <a-col :span="6">
                <a-form-item label="SSL/TLS">
                  <a-switch v-model:checked="generalConfig.smtp_ssl" />
                  <span class="switch-label-inline">{{ generalConfig.smtp_ssl ? '已启用' : '未启用' }}</span>
                </a-form-item>
              </a-col>
            </a-row>
            <a-form-item label="SMTP 用户名">
              <a-input v-model:value="generalConfig.smtp_username" size="large" placeholder="发件人地址" />
            </a-form-item>
          </a-form>
        </div>

        <!-- Save Button -->
        <div class="config-save">
          <a-button type="primary" size="large" :loading="saving" @click="saveGeneralConfig">
            <SaveOutlined /> 保存所有设置
          </a-button>
        </div>
      </div>
    </div>
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

/* Overview Stats */
.overview-stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.overview-stat {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 18px 20px;
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-xl);
  transition: all 0.3s ease;
}

.overview-stat:hover {
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
}

.overview-stat-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  flex-shrink: 0;
}

.overview-stat-icon--primary { background: var(--color-primary-bg); color: var(--color-primary); }
.overview-stat-icon--success { background: var(--color-success-bg); color: var(--color-success); }
.overview-stat-icon--info { background: var(--color-info-bg); color: var(--color-info); }

.overview-stat-value { font-size: 22px; font-weight: 700; color: var(--color-text-primary); }
.overview-stat-label { font-size: 13px; color: var(--color-text-tertiary); margin-top: 2px; }

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
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 20px;
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

.tab-content-header {
  margin-bottom: 20px;
}

.tab-desc {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin: 0;
}

/* OA Grid */
.oa-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(460px, 1fr));
  gap: 20px;
}

.oa-card {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-xl);
  padding: 22px;
  transition: all 0.3s ease;
}

.oa-card:hover {
  box-shadow: var(--shadow-md);
}

.oa-card--disabled {
  opacity: 0.65;
}

.oa-card-header {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 14px;
}

.oa-card-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-lg);
  background: var(--color-bg-hover);
  color: var(--color-text-tertiary);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  flex-shrink: 0;
  transition: all 0.3s ease;
}

.oa-card-icon--active {
  background: var(--color-primary-bg);
  color: var(--color-primary);
}

.oa-card-info {
  flex: 1;
}

.oa-card-name {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}

.oa-card-version {
  font-size: 12px;
  color: var(--color-text-tertiary);
}

.oa-card-status {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  font-weight: 500;
  padding: 4px 10px;
  border-radius: var(--radius-full);
  flex-shrink: 0;
}

.oa-card-desc {
  font-size: 13px;
  color: var(--color-text-secondary);
  line-height: 1.5;
  margin: 0 0 14px;
}

.oa-card-meta {
  display: flex;
  gap: 20px;
  padding: 10px 14px;
  background: var(--color-bg-page);
  border-radius: var(--radius-md);
  margin-bottom: 14px;
  flex-wrap: wrap;
}

.oa-meta-label {
  font-size: 11px;
  color: var(--color-text-tertiary);
  display: block;
}

.oa-meta-value {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-primary);
  margin-top: 2px;
  display: block;
}

.oa-card-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 14px;
  border-top: 1px solid var(--color-border-light);
}

/* AI Grid */
.ai-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(460px, 1fr));
  gap: 20px;
}

.ai-card {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-xl);
  padding: 22px;
  transition: all 0.3s ease;
}

.ai-card:hover {
  box-shadow: var(--shadow-md);
}

.ai-card--disabled {
  opacity: 0.65;
}

.ai-card-header {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  margin-bottom: 12px;
}

.ai-card-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  flex-shrink: 0;
}

.ai-card-icon--local {
  background: var(--color-success-bg);
  color: var(--color-success);
}

.ai-card-icon--cloud {
  background: var(--color-info-bg);
  color: var(--color-info);
}

.ai-card-info {
  flex: 1;
}

.ai-card-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}

.ai-card-provider {
  font-size: 12px;
  color: var(--color-text-tertiary);
}

.ai-card-badges {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.ai-type-badge, .ai-status-badge {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 500;
  padding: 3px 10px;
  border-radius: var(--radius-full);
  white-space: nowrap;
}

.ai-card-desc {
  font-size: 13px;
  color: var(--color-text-secondary);
  line-height: 1.5;
  margin: 0 0 12px;
}

.ai-capabilities {
  display: flex;
  gap: 6px;
  margin-bottom: 14px;
  flex-wrap: wrap;
}

.capability-tag {
  font-size: 11px;
  padding: 2px 10px;
  border-radius: var(--radius-full);
  background: var(--color-bg-hover);
  color: var(--color-text-secondary);
  font-weight: 500;
}

.ai-card-meta {
  padding: 10px 14px;
  background: var(--color-bg-page);
  border-radius: var(--radius-md);
  margin-bottom: 14px;
}

.ai-meta-row {
  display: flex;
  gap: 20px;
  flex-wrap: wrap;
}

.ai-meta-row + .ai-meta-row {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px dashed var(--color-border-light);
}

.ai-meta-label {
  font-size: 11px;
  color: var(--color-text-tertiary);
  display: block;
}

.ai-meta-value {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-primary);
  margin-top: 2px;
  display: block;
}

.ai-meta-value--mono {
  font-family: var(--font-mono);
  font-size: 12px;
}

.ai-card-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  padding-top: 14px;
  border-top: 1px solid var(--color-border-light);
}

/* Config Sections */
.config-sections {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.config-section {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-xl);
  padding: 24px;
}

.config-section-header {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 20px;
}

.config-section-icon {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  flex-shrink: 0;
}

.config-section-icon--primary { background: var(--color-primary-bg); color: var(--color-primary); }
.config-section-icon--success { background: var(--color-success-bg); color: var(--color-success); }
.config-section-icon--warning { background: var(--color-warning-bg); color: var(--color-warning); }
.config-section-icon--info { background: var(--color-info-bg); color: var(--color-info); }

.config-section-header h3 {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}

.config-section-header p {
  font-size: 13px;
  color: var(--color-text-tertiary);
  margin: 2px 0 0;
}

.toggle-grid {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.toggle-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 14px 0;
  border-bottom: 1px solid var(--color-border-light);
}

.toggle-item:last-child {
  border-bottom: none;
}

.toggle-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-primary);
}

.toggle-desc {
  font-size: 12px;
  color: var(--color-text-tertiary);
  margin-top: 2px;
}

.form-hint {
  font-size: 11px;
  color: var(--color-text-tertiary);
  margin-top: 4px;
}

.switch-label-inline {
  font-size: 13px;
  color: var(--color-text-tertiary);
  margin-left: 10px;
}

.config-save {
  display: flex;
  justify-content: flex-end;
  padding: 4px 0;
}

@media (max-width: 1024px) {
  .overview-stats {
    grid-template-columns: 1fr 1fr;
  }

  .oa-grid, .ai-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .overview-stats {
    grid-template-columns: 1fr;
  }

  .tab-nav {
    width: 100%;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
  }

  .tab-btn {
    flex: 1;
    text-align: center;
    padding: 8px 12px;
    justify-content: center;
  }
}
</style>
