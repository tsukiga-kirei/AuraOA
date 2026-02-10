<script setup lang="ts">
import {
  PlusOutlined,
  CheckCircleOutlined,
  ApiOutlined,
  TeamOutlined,
  LinkOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'

definePageMeta({ middleware: 'auth' })

const { mockTenants } = useMockData()

const activeTab = ref('tenants')
const tenants = ref([...mockTenants])
const showCreate = ref(false)

const newTenant = ref({
  name: '',
  oa_type: 'weaver_e9',
  token_quota: 10000,
  max_concurrency: 10,
})

const createTenant = () => {
  tenants.value.push({
    id: `T-${Date.now()}`,
    ...newTenant.value,
    token_used: 0,
    status: 'active' as const,
    created_at: new Date().toISOString().slice(0, 10),
  })
  showCreate.value = false
  message.success('租户创建成功')
  newTenant.value = { name: '', oa_type: 'weaver_e9', token_quota: 10000, max_concurrency: 10 }
}

const toggleTenantStatus = (id: string) => {
  const t = tenants.value.find(t => t.id === id)
  if (t) {
    t.status = t.status === 'active' ? 'inactive' : 'active'
    message.success(t.status === 'active' ? '已启用' : '已停用')
  }
}

const getQuotaPercent = (used: number, total: number) => Math.round((used / total) * 100)

const getQuotaColor = (percent: number) => {
  if (percent >= 90) return '#ef4444'
  if (percent >= 70) return '#f59e0b'
  return '#10b981'
}
</script>

<template>
  <div class="system-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">系统管理</h1>
        <p class="page-subtitle">租户管理、OA 集成与并发控制</p>
      </div>
    </div>

    <div class="tab-nav">
      <button
        v-for="tab in [
          { key: 'tenants', label: '租户管理' },
          { key: 'oa', label: 'OA 集成' },
          { key: 'concurrency', label: '并发控制' },
        ]"
        :key="tab.key"
        class="tab-btn"
        :class="{ 'tab-btn--active': activeTab === tab.key }"
        @click="activeTab = tab.key"
      >
        {{ tab.label }}
      </button>
    </div>

    <!-- Tenants tab -->
    <div v-if="activeTab === 'tenants'" class="tab-content">
      <div class="tab-content-header">
        <span class="tab-content-count">共 {{ tenants.length }} 个租户</span>
        <a-button type="primary" @click="showCreate = true">
          <PlusOutlined /> 新增租户
        </a-button>
      </div>

      <div class="tenant-grid">
        <div v-for="tenant in tenants" :key="tenant.id" class="tenant-card">
          <div class="tenant-card-header">
            <div class="tenant-avatar">
              <TeamOutlined />
            </div>
            <div class="tenant-info">
              <div class="tenant-name">{{ tenant.name }}</div>
              <div class="tenant-id">{{ tenant.id }}</div>
            </div>
            <div
              class="tenant-status"
              :class="tenant.status === 'active' ? 'tenant-status--active' : 'tenant-status--inactive'"
            >
              <span class="tenant-status-dot" />
              {{ tenant.status === 'active' ? '运行中' : '已停用' }}
            </div>
          </div>

          <div class="tenant-details">
            <div class="tenant-detail-row">
              <span class="tenant-detail-label">OA 类型</span>
              <span class="tenant-detail-value">
                <span class="oa-tag">泛微 E9</span>
              </span>
            </div>
            <div class="tenant-detail-row">
              <span class="tenant-detail-label">Token 用量</span>
              <span class="tenant-detail-value">
                {{ tenant.token_used.toLocaleString() }} / {{ tenant.token_quota.toLocaleString() }}
              </span>
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

          <div class="tenant-detail-row">
            <span class="tenant-detail-label">最大并发</span>
            <span class="tenant-detail-value">{{ tenant.max_concurrency }}</span>
          </div>

          <div class="tenant-card-footer">
            <span class="tenant-created">创建于 {{ tenant.created_at }}</span>
            <a-button size="small" @click="toggleTenantStatus(tenant.id)">
              {{ tenant.status === 'active' ? '停用' : '启用' }}
            </a-button>
          </div>
        </div>
      </div>
    </div>

    <!-- OA Integration tab -->
    <div v-if="activeTab === 'oa'" class="tab-content">
      <div class="oa-card">
        <div class="oa-card-header">
          <div class="oa-icon">
            <LinkOutlined />
          </div>
          <div>
            <h3>泛微 E9</h3>
            <p>当前已适配的 OA 系统</p>
          </div>
          <div class="oa-status">
            <CheckCircleOutlined style="color: var(--color-success);" />
            已连接
          </div>
        </div>

        <div class="oa-details">
          <div class="oa-detail-item">
            <span class="oa-detail-label">连接方式</span>
            <span class="oa-detail-value">JDBC 数据库直连</span>
          </div>
          <div class="oa-detail-item">
            <span class="oa-detail-label">数据同步</span>
            <span class="oa-detail-value">实时轮询（30s 间隔）</span>
          </div>
          <div class="oa-detail-item">
            <span class="oa-detail-label">适配版本</span>
            <span class="oa-detail-value">E9 v10.x</span>
          </div>
          <div class="oa-detail-item">
            <span class="oa-detail-label">最后同步</span>
            <span class="oa-detail-value">2025-06-10 15:30:22</span>
          </div>
        </div>

        <div class="oa-future">
          <h4>计划适配</h4>
          <div class="oa-future-list">
            <span class="oa-future-tag">致远 A8+</span>
            <span class="oa-future-tag">泛微 E-Bridge</span>
            <span class="oa-future-tag">蓝凌 EKP</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Concurrency tab -->
    <div v-if="activeTab === 'concurrency'" class="tab-content">
      <div class="concurrency-card">
        <div class="concurrency-icon">
          <ThunderboltOutlined />
        </div>
        <h3>并发控制说明</h3>
        <p>并发数在租户级别配置，通过 Token 配额和最大并发数共同限制。每个租户的审核请求将受到以下约束：</p>
        <div class="concurrency-rules">
          <div class="concurrency-rule">
            <span class="concurrency-rule-num">1</span>
            <span>单租户同时进行的审核任务不超过其配置的最大并发数</span>
          </div>
          <div class="concurrency-rule">
            <span class="concurrency-rule-num">2</span>
            <span>Token 用量达到配额上限后，新的审核请求将被排队等待</span>
          </div>
          <div class="concurrency-rule">
            <span class="concurrency-rule-num">3</span>
            <span>系统管理员可随时调整各租户的配额和并发限制</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Create tenant modal -->
    <a-modal v-model:open="showCreate" title="新增租户" @ok="createTenant" okText="创建" cancelText="取消">
      <a-form layout="vertical" style="margin-top: 16px;">
        <a-form-item label="租户名称">
          <a-input v-model:value="newTenant.name" placeholder="如：XX集团总部" size="large" />
        </a-form-item>
        <a-form-item label="OA 类型">
          <a-select v-model:value="newTenant.oa_type" size="large">
            <a-select-option value="weaver_e9">泛微 E9</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="Token 配额">
          <a-input-number v-model:value="newTenant.token_quota" :min="1000" :step="1000" style="width: 100%;" size="large" />
        </a-form-item>
        <a-form-item label="最大并发数">
          <a-input-number v-model:value="newTenant.max_concurrency" :min="1" :max="100" style="width: 100%;" size="large" />
        </a-form-item>
      </a-form>
    </a-modal>
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

/* Tenant grid */
.tenant-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 20px;
}

.tenant-card {
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light);
  padding: 20px;
  transition: all var(--transition-base);
}

.tenant-card:hover {
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
}

.tenant-card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.tenant-avatar {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-lg);
  background: var(--color-primary-bg);
  color: var(--color-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  flex-shrink: 0;
}

.tenant-info {
  flex: 1;
  min-width: 0;
}

.tenant-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.tenant-id {
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
}

.tenant-status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
}

.tenant-status--active {
  color: var(--color-success);
}

.tenant-status--active .tenant-status-dot {
  background: var(--color-success);
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.2);
}

.tenant-status--inactive {
  color: var(--color-text-tertiary);
}

.tenant-status--inactive .tenant-status-dot {
  background: var(--color-text-tertiary);
}

.tenant-details {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 14px;
}

.tenant-detail-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 13px;
}

.tenant-detail-label {
  color: var(--color-text-tertiary);
}

.tenant-detail-value {
  color: var(--color-text-primary);
  font-weight: 500;
}

.oa-tag {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: var(--radius-full);
  background: var(--color-info-bg);
  color: var(--color-info);
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
}

/* OA card */
.oa-card {
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light);
  padding: 24px;
  max-width: 600px;
}

.oa-card-header {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 20px;
}

.oa-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-lg);
  background: var(--color-primary-bg);
  color: var(--color-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
}

.oa-card-header h3 {
  font-size: 16px;
  font-weight: 600;
  margin: 0;
  flex: 1;
}

.oa-card-header p {
  font-size: 12px;
  color: var(--color-text-tertiary);
  margin: 2px 0 0;
}

.oa-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 500;
  color: var(--color-success);
}

.oa-details {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  padding: 16px;
  background: var(--color-bg-page);
  border-radius: var(--radius-md);
  margin-bottom: 20px;
}

.oa-detail-label {
  font-size: 12px;
  color: var(--color-text-tertiary);
  display: block;
}

.oa-detail-value {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-primary);
  margin-top: 2px;
  display: block;
}

.oa-future h4 {
  font-size: 14px;
  font-weight: 600;
  margin: 0 0 10px;
  color: var(--color-text-secondary);
}

.oa-future-list {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.oa-future-tag {
  font-size: 12px;
  padding: 4px 12px;
  border-radius: var(--radius-full);
  background: var(--color-bg-hover);
  color: var(--color-text-tertiary);
}

/* Concurrency */
.concurrency-card {
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light);
  padding: 32px;
  max-width: 600px;
  text-align: center;
}

.concurrency-icon {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  background: var(--color-primary-bg);
  color: var(--color-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  margin: 0 auto 16px;
}

.concurrency-card h3 {
  font-size: 18px;
  font-weight: 600;
  margin: 0 0 8px;
}

.concurrency-card p {
  font-size: 14px;
  color: var(--color-text-secondary);
  margin: 0 0 24px;
  line-height: 1.6;
}

.concurrency-rules {
  display: flex;
  flex-direction: column;
  gap: 12px;
  text-align: left;
}

.concurrency-rule {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  font-size: 14px;
  color: var(--color-text-secondary);
  line-height: 1.5;
}

.concurrency-rule-num {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--color-primary-bg);
  color: var(--color-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 700;
  flex-shrink: 0;
}

@media (max-width: 640px) {
  .tab-nav {
    width: 100%;
  }

  .tab-btn {
    flex: 1;
    text-align: center;
    padding: 8px 12px;
  }

  .tenant-grid {
    grid-template-columns: 1fr;
  }

  .oa-details {
    grid-template-columns: 1fr;
  }
}
</style>
