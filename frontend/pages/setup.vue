<script setup lang="ts">
import { ref } from 'vue'
import { message } from 'ant-design-vue'
import { UserOutlined, LockOutlined, IdcardOutlined, SafetyCertificateOutlined } from '@ant-design/icons-vue'
import { useI18n } from '~/composables/useI18n'

definePageMeta({ layout: false, middleware: 'auth' })

const { t } = useI18n()
const { isDark, toggle: toggleTheme, restore: restoreTheme } = useTheme()
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
      <a-tooltip :title="t('header.toggleTheme')" placement="bottom" :mouse-enter-delay="0.5">
        <button
          type="button"
          class="theme-toggle-btn"
          :class="{ 'theme-toggle-btn--dark': isDark }"
          :aria-label="isDark ? t('header.lightMode') : t('header.darkMode')"
          @click="toggleTheme"
        >
          <span class="theme-toggle-track">
            <span class="theme-toggle-thumb">
              <transition name="theme-icon" mode="out-in">
                <svg v-if="isDark" key="moon" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="currentColor" stroke="none"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" /></svg>
                <svg v-else key="sun" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="4" /><line x1="12" y1="2" x2="12" y2="5" /><line x1="12" y1="19" x2="12" y2="22" /><line x1="4.93" y1="4.93" x2="6.76" y2="6.76" /><line x1="17.24" y1="17.24" x2="19.07" y2="19.07" /><line x1="2" y1="12" x2="5" y2="12" /><line x1="19" y1="12" x2="22" y2="12" /><line x1="4.93" y1="19.07" x2="6.76" y2="17.24" /><line x1="17.24" y1="6.76" x2="19.07" y2="4.93" /></svg>
              </transition>
            </span>
          </span>
        </button>
      </a-tooltip>
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
/* 与 AppHeader 相同的药丸式主题切换 */
.setup-theme-floating {
  position: absolute;
  top: 20px;
  right: 20px;
  z-index: 10;
}
.theme-toggle-btn {
  width: auto !important;
  padding: 0 !important;
  background: transparent !important;
  border: none !important;
}
.theme-toggle-btn:hover {
  background: transparent !important;
}
.theme-toggle-track {
  display: flex;
  align-items: center;
  width: 52px;
  height: 28px;
  border-radius: 14px;
  background: #e2e8f0;
  padding: 3px;
  transition: background 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  cursor: pointer;
  position: relative;
}
.theme-toggle-btn--dark .theme-toggle-track {
  background: #334155;
}
.theme-toggle-thumb {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.15);
  transition:
    transform 0.35s cubic-bezier(0.4, 0, 0.2, 1),
    background 0.35s ease;
  color: #f59e0b;
}
.theme-toggle-btn--dark .theme-toggle-thumb {
  transform: translateX(24px);
  background: #1e293b;
  color: #818cf8;
}
.theme-icon-enter-active,
.theme-icon-leave-active {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}
.theme-icon-enter-from {
  opacity: 0;
  transform: rotate(-90deg) scale(0.5);
}
.theme-icon-leave-to {
  opacity: 0;
  transform: rotate(90deg) scale(0.5);
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
  background: linear-gradient(135deg, #4f46e5, #7c3aed);
  top: -120px;
  left: -80px;
}
.setup-bg-shape--2 {
  width: 400px;
  height: 400px;
  background: linear-gradient(135deg, #06b6d4, #3b82f6);
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
  color: var(--color-primary, #4f46e5);
  margin-bottom: 12px;
}
.setup-card-head h1 {
  margin: 0 0 8px;
  font-size: 22px;
  font-weight: 700;
  color: var(--color-text-primary);
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
