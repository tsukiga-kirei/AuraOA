<script setup lang="ts">
import { ref } from 'vue'
import { message } from 'ant-design-vue'
import { UserOutlined, LockOutlined, IdcardOutlined, SafetyCertificateOutlined } from '@ant-design/icons-vue'
import { useI18n } from '~/composables/useI18n'

definePageMeta({ layout: false, middleware: 'auth' })

const { t } = useI18n()
const { restore: restoreTheme } = useTheme()
const config = useRuntimeConfig()

// 初始化管理员账号的表单字段
const username = ref('')
const displayName = ref('')
const password = ref('')
const confirmPassword = ref('')
// 提交加载状态
const loading = ref(false)

// 页面挂载时恢复主题偏好
onMounted(() => {
  restoreTheme()
})

// 提交初始化表单，创建系统管理员账号
const submit = async () => {
  const u = username.value.trim()
  const d = displayName.value.trim()
  if (!u || !d || !password.value) {
    message.warning(t('login.emptyWarning'))
    return
  }
  if (password.value !== confirmPassword.value) {
    message.error(t('setup.mismatch'))
    return
  }
  loading.value = true
  try {
    const res = await $fetch<{ code: number; message: string; trace_id?: string }>(
      `${String(config.public.apiBase)}/api/auth/bootstrap`,
      {
        method: 'POST',
        body: {
          username: u,
          display_name: d,
          password: password.value,
        },
      }
    )
    if (res.code !== 0) {
      message.error(res.message || '创建失败')
      return
    }
    message.success(t('setup.success'))
    await navigateTo('/login')
  } catch (e: any) {
    const msg = e?.data?.message || e?.message || '创建失败'
    message.error(msg)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="setup-page">
    <div class="setup-theme-floating">
      <ThemeSwitcher />
    </div>

    <div class="setup-bg">
      <div class="setup-bg-shape setup-bg-shape--1" />
      <div class="setup-bg-shape setup-bg-shape--2" />
    </div>

    <div class="setup-card">
      <div class="setup-card-head">
        <SafetyCertificateOutlined class="setup-card-icon" />
        <h1>{{ t('setup.title') }}</h1>
        <p>{{ t('setup.subtitle') }}</p>
      </div>

      <a-form layout="vertical" class="setup-form">
        <a-form-item :label="t('login.username')" :extra="t('setup.usernameRule')">
          <a-input v-model:value="username" size="large" :placeholder="t('login.usernamePlaceholder')">
            <template #prefix><UserOutlined /></template>
          </a-input>
        </a-form-item>
        <a-form-item :label="t('setup.displayName')">
          <a-input v-model:value="displayName" size="large" :placeholder="t('setup.displayNamePlaceholder')">
            <template #prefix><IdcardOutlined /></template>
          </a-input>
        </a-form-item>
        <a-form-item :label="t('login.password')" :extra="t('setup.passwordHint')">
          <a-input-password v-model:value="password" size="large" :placeholder="t('login.passwordPlaceholder')">
            <template #prefix><LockOutlined /></template>
          </a-input-password>
        </a-form-item>
        <a-form-item :label="t('setup.confirmPassword')">
          <a-input-password v-model:value="confirmPassword" size="large" :placeholder="t('login.passwordPlaceholder')">
            <template #prefix><LockOutlined /></template>
          </a-input-password>
        </a-form-item>
        <a-form-item>
          <a-button type="primary" size="large" block class="setup-submit-btn" :loading="loading" @click="submit">
            {{ loading ? t('setup.submitting') : t('setup.submit') }}
          </a-button>
        </a-form-item>
      </a-form>
    </div>
  </div>
</template>

<style scoped>
.setup-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
  background: var(--color-bg-sidebar);
  padding: 24px;
  transition: background-color 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}
/* 与 AppHeader 一致的主题切换 */
.setup-theme-floating {
  position: absolute;
  top: 20px;
  right: 20px;
  z-index: 10;
}
.setup-bg {
  position: absolute;
  inset: 0;
  overflow: hidden;
  pointer-events: none;
}
.setup-bg-shape {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  opacity: 0.45;
}
.setup-bg-shape--1 {
  width: 480px;
  height: 480px;
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-light));
  top: -120px;
  left: -80px;
}
.setup-bg-shape--2 {
  width: 400px;
  height: 400px;
  background: linear-gradient(135deg, var(--color-accent), var(--color-info));
  bottom: -100px;
  right: -60px;
}
.setup-card {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 440px;
  padding: 40px 36px;
  border-radius: 20px;
  background: var(--color-bg-card);
  box-shadow: 0 25px 60px rgba(0, 0, 0, 0.35);
}
.setup-card-head {
  text-align: center;
  margin-bottom: 28px;
}
.setup-card-icon {
  font-size: 40px;
  color: var(--color-primary);
  margin-bottom: 12px;
}
.setup-card-head h1 {
  margin: 0 0 8px;
  font-size: 22px;
  font-weight: 700;
  color: var(--color-text-primary);
  font-family: var(--font-display);
}
.setup-card-head p {
  margin: 0;
  font-size: 14px;
  color: var(--color-text-tertiary);
  line-height: 1.5;
}
.setup-form :deep(.ant-form-item) {
  margin-bottom: 18px;
}
/* 块级主按钮：文字与 loading 图标整体水平居中 */
.setup-form :deep(.setup-submit-btn.ant-btn) {
  display: inline-flex !important;
  align-items: center;
  justify-content: center;
  width: 100%;
}
.setup-form :deep(.setup-submit-btn.ant-btn-loading) {
  justify-content: center;
}
</style>
