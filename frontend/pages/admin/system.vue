<script setup lang="ts">
import { PlusOutlined } from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'

definePageMeta({ middleware: 'auth' })

const config = useRuntimeConfig()
const { token } = useAuth()

const activeTab = ref('tenants')
const tenants = ref<any[]>([])
const loading = ref(false)
const showCreate = ref(false)

const headers = computed(() => ({
  Authorization: `Bearer ${token.value}`,
}))

const newTenant = ref({
  name: '',
  oa_type: 'weaver_e9',
  token_quota: 10000,
  max_concurrency: 10,
})

const fetchTenants = async () => {
  loading.value = true
  try {
    const data = await $fetch<{ tenants: any[] }>(`${config.public.apiBase}/api/admin/tenants`, {
      headers: headers.value,
    })
    tenants.value = data.tenants
  } catch {
    tenants.value = []
  } finally {
    loading.value = false
  }
}

const createTenant = async () => {
  try {
    await $fetch(`${config.public.apiBase}/api/admin/tenants`, {
      method: 'POST',
      headers: headers.value,
      body: newTenant.value,
    })
    message.success('租户创建成功')
    showCreate.value = false
    await fetchTenants()
  } catch {
    message.error('创建失败')
  }
}

onMounted(fetchTenants)

const tenantColumns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: 'OA 类型', dataIndex: 'oa_type', key: 'oa_type' },
  { title: 'Token 配额', dataIndex: 'token_quota', key: 'quota' },
  { title: '已用', dataIndex: 'token_used', key: 'used' },
  { title: '并发限制', dataIndex: 'max_concurrency', key: 'concurrency' },
]
</script>

<template>
  <div>
    <a-page-header title="系统管理" sub-title="租户管理与 OA 集成" />

    <a-tabs v-model:activeKey="activeTab">
      <a-tab-pane key="tenants" tab="租户管理">
        <a-button type="primary" style="margin-bottom: 16px;" @click="showCreate = true">
          <PlusOutlined /> 新增租户
        </a-button>
        <a-table :columns="tenantColumns" :data-source="tenants" :loading="loading" row-key="id" size="small" />
      </a-tab-pane>

      <a-tab-pane key="oa" tab="OA 集成">
        <a-descriptions :column="1" bordered size="small">
          <a-descriptions-item label="当前适配">泛微 E9</a-descriptions-item>
          <a-descriptions-item label="连接状态">
            <a-badge status="success" text="已连接" />
          </a-descriptions-item>
        </a-descriptions>
      </a-tab-pane>

      <a-tab-pane key="concurrency" tab="并发控制">
        <a-alert message="并发数控制在租户级别配置，通过 Token 配额和最大并发数限制" type="info" show-icon />
      </a-tab-pane>
    </a-tabs>

    <a-modal v-model:open="showCreate" title="新增租户" @ok="createTenant">
      <a-form layout="vertical">
        <a-form-item label="租户名称">
          <a-input v-model:value="newTenant.name" placeholder="组织名称" />
        </a-form-item>
        <a-form-item label="OA 类型">
          <a-select v-model:value="newTenant.oa_type">
            <a-select-option value="weaver_e9">泛微 E9</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="Token 配额">
          <a-input-number v-model:value="newTenant.token_quota" :min="100" style="width: 100%;" />
        </a-form-item>
        <a-form-item label="最大并发数">
          <a-input-number v-model:value="newTenant.max_concurrency" :min="1" :max="100" style="width: 100%;" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>
