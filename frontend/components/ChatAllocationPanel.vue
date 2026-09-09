<script setup lang="ts">
import {
  RobotOutlined,
  ApiOutlined,
  BookOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type { TenantChatAllocationItem } from '~/types/chat'

const props = defineProps<{
  tenantId: string
  models: { id: string; name?: string; model_name?: string }[]
}>()

const { authFetch } = useAuth()
const { t } = useI18n()

const allocation = ref<TenantChatAllocationItem | null>(null)
const loading = ref(false)
const error = ref('')
const previousRetentionDays = ref(90)

const isPermanentRetention = (days?: number | null) => {
  return days === -1 || days === 0
}

const handleTogglePermanentRetention = (checked: boolean) => {
  if (!allocation.value) return
  if (checked) {
    if (allocation.value.chat_retention_days && allocation.value.chat_retention_days > 0) {
      previousRetentionDays.value = allocation.value.chat_retention_days
    }
    allocation.value.chat_retention_days = -1
  } else {
    allocation.value.chat_retention_days = (previousRetentionDays.value && previousRetentionDays.value > 0)
      ? previousRetentionDays.value
      : 90
  }
}

watch(
  () => props.tenantId,
  async (id) => {
    if (!id) return
    allocation.value = null
    loading.value = true
    error.value = ''
    try {
      const value = await authFetch<TenantChatAllocationItem>(`/api/admin/tenants/${id}/chat-allocation`)
      if (props.tenantId === id) {
        allocation.value = value
        if (value && value.chat_retention_days > 0) {
          previousRetentionDays.value = value.chat_retention_days
        }
      }
    } catch (err: any) {
      error.value = err.message
    } finally {
      loading.value = false
    }
  },
  { immediate: true },
)

const save = async (silent = false) => {
  if (!allocation.value) return
  loading.value = true
  try {
    await authFetch(`/api/admin/tenants/${props.tenantId}/chat-allocation`, {
      method: 'PUT',
      body: {
        chat_enabled: allocation.value.chat_enabled,
        chat_retention_days: allocation.value.chat_retention_days,
        chat_primary_model_id: allocation.value.chat_primary_model_id || null,
        chat_fallback_model_id: allocation.value.chat_fallback_model_id || null,
        allow_custom_skills: allocation.value.allow_custom_skills,
        allow_tenant_mcp: allocation.value.allow_tenant_mcp,
        max_mcp_servers: allocation.value.max_mcp_servers,
      },
    })
    if (!silent) message.success(t('agentAdmin.saveSuccess', '配置保存成功'))
  } catch (err: any) {
    message.error(err.message)
    throw err
  } finally {
    loading.value = false
  }
}

const modelOptions = computed(() =>
  props.models.map(model => ({
    value: model.id,
    label: model.name || model.model_name,
  })),
)

defineExpose({ save })
</script>

<template>
  <div class="chat-alloc-wrapper">
    <a-spin :spinning="loading">
      <a-alert v-if="error" :message="error" type="error" show-icon style="margin-bottom: 16px;" />

      <div v-if="allocation" class="detail-section">
        <div class="section-header">
          <h3><RobotOutlined /> {{ t('agentAdmin.title', '智能体与能力分配') }}</h3>
        </div>

        <a-form layout="vertical">
          <!-- 功能开关网格卡片 -->
          <div class="config-group">
            <div class="config-group-title">能力开关与权限</div>
            <div class="switches-grid">
              <div class="switch-card">
                <div class="switch-card-info">
                  <div class="switch-card-title">
                    <RobotOutlined style="color: var(--color-primary);" />
                    <span>{{ t('chat.allocation.enabled', '启用 AI 助理/对话') }}</span>
                  </div>
                  <div class="switch-card-desc">控制该租户用户是否可使用 AI 助理对话功能</div>
                </div>
                <a-switch v-model:checked="allocation.chat_enabled" />
              </div>

              <div class="switch-card">
                <div class="switch-card-info">
                  <div class="switch-card-title">
                    <BookOutlined style="color: var(--color-success);" />
                    <span>{{ t('chat.allocation.customSkills', '允许自定义技能') }}</span>
                  </div>
                  <div class="switch-card-desc">允许该租户管理员创建和装配领域专属 Skills</div>
                </div>
                <a-switch v-model:checked="allocation.allow_custom_skills" />
              </div>

              <div class="switch-card">
                <div class="switch-card-info">
                  <div class="switch-card-title">
                    <ApiOutlined style="color: var(--color-warning);" />
                    <span>{{ t('chat.allocation.mcp', '允许接入 MCP 服务') }}</span>
                  </div>
                  <div class="switch-card-desc">允许该租户注册外部 MCP 协议服务扩展工具</div>
                </div>
                <a-switch v-model:checked="allocation.allow_tenant_mcp" />
              </div>
            </div>
          </div>

          <!-- 模型与参数策略 -->
          <div class="config-group" style="margin-top: 20px;">
            <div class="config-group-title">模型配置与存储策略</div>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item :label="t('chat.allocation.primaryModel', '智能体默认主模型')">
                  <a-select
                    :value="allocation.chat_primary_model_id || undefined"
                    allow-clear
                    :options="modelOptions"
                    :placeholder="t('chat.allocation.defaultModel', '跟随租户全局主模型')"
                    size="large"
                    @change="(value: any) => allocation && (allocation.chat_primary_model_id = value ? String(value) : null)"
                  />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item :label="t('chat.allocation.fallbackModel', '备用降级模型')">
                  <a-select
                    :value="allocation.chat_fallback_model_id || undefined"
                    allow-clear
                    :options="modelOptions"
                    :placeholder="t('chat.allocation.defaultModel', '跟随租户全局备用模型')"
                    size="large"
                    @change="(value: any) => allocation && (allocation.chat_fallback_model_id = value ? String(value) : null)"
                  />
                </a-form-item>
              </a-col>
            </a-row>

            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item>
                  <template #label>
                    <div class="retention-label-row">
                      <span>{{ t('chat.allocation.retention', '会话保留天数') }}</span>
                      <div class="retention-switch-inline">
                        <span>{{ t('chat.allocation.permanent', '永久保留') }}</span>
                        <a-switch
                          size="small"
                          :checked="isPermanentRetention(allocation.chat_retention_days)"
                          @change="handleTogglePermanentRetention"
                        />
                      </div>
                    </div>
                  </template>
                  <a-input-number
                    v-if="!isPermanentRetention(allocation.chat_retention_days)"
                    v-model:value="allocation.chat_retention_days"
                    :min="1"
                    :max="3650"
                    size="large"
                    style="width: 100%;"
                  />
                  <a-input
                    v-else
                    size="large"
                    disabled
                    :value="t('chat.allocation.permanentHint', '永久保留（不自动清理历史对话）')"
                    style="width: 100%;"
                  />
                </a-form-item>
              </a-col>
              <a-col :span="12" v-if="allocation.allow_tenant_mcp">
                <a-form-item :label="t('chat.allocation.maxMCP', '最大允许 MCP 服务数')">
                  <a-input-number
                    v-model:value="allocation.max_mcp_servers"
                    :min="0"
                    :max="100"
                    size="large"
                    style="width: 100%;"
                  />
                </a-form-item>
              </a-col>
            </a-row>
          </div>

          <p class="perm-hint">{{ t('chat.allocation.hint', '系统管理员拥有分配权，租户管理员仅可在此策略范围内进行管理与选装。') }}</p>
        </a-form>
      </div>
    </a-spin>
  </div>
</template>

<style scoped>
.chat-alloc-wrapper {
  padding: 4px 0;
}
.detail-section {
  background: var(--color-bg-card);
  border-radius: 8px;
}
.section-header {
  margin-bottom: 20px;
}
.section-header h3 {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}
.config-group {
  margin-bottom: 16px;
}
.config-group-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-secondary);
  margin-bottom: 12px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

/* 开关卡片紧凑网格 */
.switches-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 12px;
}
.switch-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 16px;
  background: var(--color-bg-page);
  border: 1px solid var(--color-border-light);
  border-radius: 8px;
}
.switch-card-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.switch-card-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-primary);
}
.switch-card-desc {
  font-size: 12px;
  color: var(--color-text-tertiary);
}

.perm-hint {
  font-size: 12px;
  color: var(--color-text-tertiary);
  margin-top: 16px;
  margin-bottom: 0;
}

.retention-label-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}
.retention-switch-inline {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-left: 12px;
  font-size: 14px;
  color: var(--color-text-secondary);
}
</style>
