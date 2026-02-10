<script setup lang="ts">
import {
  UserOutlined,
  MailOutlined,
  PhoneOutlined,
  SafetyCertificateOutlined,
  SaveOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'

definePageMeta({ middleware: 'auth' })

const { isDark, toggle: toggleTheme } = useTheme()
const { userRole } = useAuth()

const profile = ref({
  nickname: '管理员',
  email: 'admin@example.com',
  phone: '138****8888',
  department: '信息技术部',
  position: '系统管理员',
})

const notifications = ref({
  audit_complete: true,
  daily_report: true,
  weekly_report: false,
  system_alert: true,
})

const saving = ref(false)

const handleSave = async () => {
  saving.value = true
  await new Promise(r => setTimeout(r, 800))
  saving.value = false
  message.success('设置已保存')
}

const roleLabels: Record<string, string> = {
  business: '业务用户',
  tenant_admin: '租户管理员',
  system_admin: '系统管理员',
}
</script>

<template>
  <div class="settings-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">个人设置</h1>
        <p class="page-subtitle">管理您的账户信息与偏好设置</p>
      </div>
    </div>

    <div class="settings-grid">
      <!-- Profile card -->
      <div class="settings-card">
        <div class="settings-card-header">
          <h3>基本信息</h3>
        </div>
        <div class="settings-card-body">
          <div class="profile-avatar-section">
            <a-avatar :size="72" class="profile-avatar">
              <template #icon><UserOutlined /></template>
            </a-avatar>
            <div class="profile-avatar-info">
              <div class="profile-name">{{ profile.nickname }}</div>
              <div class="profile-role">
                <span class="role-badge">{{ roleLabels[userRole] || '业务用户' }}</span>
              </div>
            </div>
          </div>

          <a-form layout="vertical" class="settings-form">
            <div class="form-row">
              <a-form-item label="昵称" class="form-col">
                <a-input v-model:value="profile.nickname" size="large">
                  <template #prefix><UserOutlined class="input-icon" /></template>
                </a-input>
              </a-form-item>
              <a-form-item label="邮箱" class="form-col">
                <a-input v-model:value="profile.email" size="large">
                  <template #prefix><MailOutlined class="input-icon" /></template>
                </a-input>
              </a-form-item>
            </div>
            <div class="form-row">
              <a-form-item label="手机号" class="form-col">
                <a-input v-model:value="profile.phone" size="large">
                  <template #prefix><PhoneOutlined class="input-icon" /></template>
                </a-input>
              </a-form-item>
              <a-form-item label="部门" class="form-col">
                <a-input v-model:value="profile.department" size="large" disabled />
              </a-form-item>
            </div>
            <a-form-item label="职位">
              <a-input v-model:value="profile.position" size="large" disabled />
            </a-form-item>
          </a-form>
        </div>
      </div>

      <!-- Preferences card -->
      <div class="settings-card">
        <div class="settings-card-header">
          <h3>偏好设置</h3>
        </div>
        <div class="settings-card-body">
          <div class="pref-item">
            <div class="pref-info">
              <div class="pref-label">深色模式</div>
              <div class="pref-desc">切换界面的明暗主题</div>
            </div>
            <a-switch :checked="isDark" @change="toggleTheme" />
          </div>

          <div class="pref-divider" />

          <div class="pref-item">
            <div class="pref-info">
              <div class="pref-label">审核完成通知</div>
              <div class="pref-desc">AI 审核完成后推送通知</div>
            </div>
            <a-switch v-model:checked="notifications.audit_complete" />
          </div>

          <div class="pref-item">
            <div class="pref-info">
              <div class="pref-label">日报推送</div>
              <div class="pref-desc">每日审核摘要邮件推送</div>
            </div>
            <a-switch v-model:checked="notifications.daily_report" />
          </div>

          <div class="pref-item">
            <div class="pref-info">
              <div class="pref-label">周报推送</div>
              <div class="pref-desc">每周审核统计报告</div>
            </div>
            <a-switch v-model:checked="notifications.weekly_report" />
          </div>

          <div class="pref-item">
            <div class="pref-info">
              <div class="pref-label">系统告警</div>
              <div class="pref-desc">系统异常或配额预警通知</div>
            </div>
            <a-switch v-model:checked="notifications.system_alert" />
          </div>
        </div>
      </div>
    </div>

    <!-- Save button -->
    <div class="settings-actions">
      <a-button type="primary" size="large" :loading="saving" @click="handleSave">
        <SaveOutlined /> 保存设置
      </a-button>
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

.settings-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
}

.settings-card {
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light);
  overflow: hidden;
}

.settings-card-header {
  padding: 18px 24px;
  border-bottom: 1px solid var(--color-border-light);
}

.settings-card-header h3 {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}

.settings-card-body {
  padding: 24px;
}

/* Profile */
.profile-avatar-section {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
}

.profile-avatar {
  background: linear-gradient(135deg, #4f46e5, #7c3aed) !important;
  flex-shrink: 0;
}

.profile-name {
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.role-badge {
  font-size: 12px;
  font-weight: 500;
  padding: 2px 10px;
  border-radius: var(--radius-full);
  background: var(--color-primary-bg);
  color: var(--color-primary);
}

.settings-form :deep(.ant-form-item) {
  margin-bottom: 16px;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.input-icon {
  color: var(--color-text-tertiary);
}

/* Preferences */
.pref-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 0;
}

.pref-label {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-primary);
}

.pref-desc {
  font-size: 12px;
  color: var(--color-text-tertiary);
  margin-top: 2px;
}

.pref-divider {
  height: 1px;
  background: var(--color-border-light);
  margin: 8px 0;
}

/* Actions */
.settings-actions {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
}

@media (max-width: 768px) {
  .settings-grid {
    grid-template-columns: 1fr;
  }

  .form-row {
    grid-template-columns: 1fr;
  }
}
</style>
