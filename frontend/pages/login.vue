<script setup lang="ts">
import { ref, computed } from 'vue'
import { message } from 'ant-design-vue'
import {
  UserOutlined,
  LockOutlined,
  SafetyCertificateOutlined,
  DashboardOutlined,
  SettingOutlined,
  ControlOutlined,
} from '@ant-design/icons-vue'

import { getDefaultPage } from '~/composables/useMockData'
import { useI18n } from '~/composables/useI18n'

definePageMeta({ layout: false })

const { login, getMenu, setUserRole, isMockMode, MOCK_USERS } = useAuth()
const { isDark, toggle: toggleTheme, restore: restoreTheme } = useTheme()
const { t } = useI18n()
const { mockTenants } = useMockData()

onMounted(() => restoreTheme())

type PortalType = 'business' | 'tenant_admin' | 'system_admin'

const portals = computed(() => [
  { key: 'business' as PortalType, icon: DashboardOutlined, title: t('login.portal.business'), desc: t('login.portal.businessDesc'), color: '#4f46e5' },
  { key: 'tenant_admin' as PortalType, icon: SettingOutlined, title: t('login.portal.tenantAdmin'), desc: t('login.portal.tenantAdminDesc'), color: '#f59e0b' },
  { key: 'system_admin' as PortalType, icon: ControlOutlined, title: t('login.portal.systemAdmin'), desc: t('login.portal.systemAdminDesc'), color: '#ef4444' },
])

const activePortal = ref<PortalType>('business')
const form = ref({ username: '', password: '', tenant_id: mockTenants[0]?.code || 'default' })
const loading = ref(false)
const rememberMe = ref(false)
const currentPortal = computed(() => portals.value.find(p => p.key === activePortal.value)!)

// Quick-fill accounts: show users who have the selected portal role type
const quickAccounts = computed(() =>
  MOCK_USERS.filter(u => u.roles.some(r => r.role === activePortal.value)),
)

const fillAccount = (user: typeof MOCK_USERS[0]) => {
  form.value.username = user.username
  form.value.password = user.password
}

const handleLogin = async () => {
  if (!form.value.username || !form.value.password) {
    message.warning(t('login.emptyWarning'))
    return
  }
  loading.value = true
  try {
    const ok = await login(form.value)
    if (ok) {
      await getMenu()
      message.success(t('login.successRedirect'))
      navigateTo('/overview')
    } else {
      message.error(t('login.failed'))
    }
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-bg">
      <div class="login-bg-shape login-bg-shape--1" />
      <div class="login-bg-shape login-bg-shape--2" />
      <div class="login-bg-shape login-bg-shape--3" />
    </div>

    <button class="login-theme-toggle" @click="toggleTheme">
      <svg v-if="isDark" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path></svg>
      <svg v-else xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="5"></circle><line x1="12" y1="1" x2="12" y2="3"></line><line x1="12" y1="21" x2="12" y2="23"></line><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line><line x1="1" y1="12" x2="3" y2="12"></line><line x1="21" y1="12" x2="23" y2="12"></line><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line></svg>
    </button>

    <div class="login-container">
      <!-- Left: Branding -->
      <div class="login-branding">
        <div class="login-branding-content">
          <div class="login-logo">
            <svg xmlns="http://www.w3.org/2000/svg" width="30" height="30" viewBox="0 0 24 24" fill="none" stroke="rgba(255,255,255,0.9)" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path><path d="M9 12l2 2 4-4"></path></svg>
          </div>
          <h1 class="login-brand-title">{{ t('app.name') }}</h1>
          <p class="login-brand-subtitle">{{ t('login.subtitle') }}</p>
          <div class="login-features">
            <div class="login-feature-item"><span class="login-feature-dot" /><span>{{ t('login.feature1') }}</span></div>
            <div class="login-feature-item"><span class="login-feature-dot" /><span>{{ t('login.feature2') }}</span></div>
            <div class="login-feature-item"><span class="login-feature-dot" /><span>{{ t('login.feature3') }}</span></div>
          </div>
        </div>
      </div>

      <!-- Right: Login Form -->
      <div class="login-form-wrapper">
        <div class="login-form-inner">
          <div class="login-form-header">
            <h2>{{ t('login.welcomeBack') }}</h2>
            <p>{{ t('login.selectIdentity') }}</p>
          </div>

          <!-- Portal selector: horizontal pill tabs, fixed size -->
          <div class="portal-selector">
            <div
              v-for="portal in portals"
              :key="portal.key"
              class="portal-pill"
              :class="{ 'portal-pill--active': activePortal === portal.key }"
              :style="activePortal === portal.key ? { '--pill-color': portal.color } : {}"
              @click="activePortal = portal.key"
            >
              <component :is="portal.icon" class="portal-pill-icon" />
              <span class="portal-pill-title">{{ portal.title }}</span>
            </div>
          </div>

          <!-- Active portal description (outside selector, fixed position) -->
          <div class="portal-active-desc">
            <span class="portal-active-dot" :style="{ background: currentPortal.color }" />
            {{ currentPortal.desc }}
          </div>

          <a-form layout="vertical" class="login-form">
            <a-form-item v-if="activePortal !== 'system_admin'">
              <a-select v-model:value="form.tenant_id" :placeholder="t('login.tenantPlaceholder')" size="large" class="login-select">
                <a-select-option v-for="tenant in mockTenants" :key="tenant.id" :value="tenant.code">
                  {{ tenant.name }}（{{ tenant.code }}）
                </a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item>
              <a-input v-model:value="form.username" :placeholder="t('login.usernamePlaceholder')" size="large" class="login-input">
                <template #prefix><UserOutlined class="login-input-icon" /></template>
              </a-input>
            </a-form-item>
            <a-form-item>
              <a-input-password v-model:value="form.password" :placeholder="t('login.passwordPlaceholder')" size="large" class="login-input">
                <template #prefix><LockOutlined class="login-input-icon" /></template>
              </a-input-password>
            </a-form-item>
            <div class="login-options">
              <a-checkbox v-model:checked="rememberMe">{{ t('login.rememberMe') }}</a-checkbox>
            </div>

            <!-- Mock mode: quick-fill test accounts -->
            <div v-if="isMockMode" class="mock-accounts">
              <div class="mock-accounts-label">{{ t('login.testAccounts') }}</div>
              <div class="mock-accounts-list">
                <a-tag
                  v-for="acc in quickAccounts" :key="acc.username"
                  class="mock-account-tag" color="blue"
                  @click="fillAccount(acc)"
                >
                  {{ acc.display_name }}（{{ acc.username }}）
                </a-tag>
              </div>
            </div>

            <a-form-item>
              <a-button
                type="primary" block size="large" :loading="loading"
                class="login-btn"
                :style="{ background: `linear-gradient(135deg, ${currentPortal.color}, ${currentPortal.color}dd)` }"
                @click="handleLogin"
              >
                {{ loading ? t('login.logging') : t('login.loginAs', currentPortal.title) }}
              </a-button>
            </a-form-item>
          </a-form>
          <div class="login-footer"><span>{{ t('app.name') }} © 2025</span></div>
        </div>
      </div>
    </div>

    <div class="login-mobile-brand">
      <SafetyCertificateOutlined class="login-mobile-logo" /><span>{{ t('app.name') }}</span>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh; display: flex; align-items: center; justify-content: center;
  position: relative; overflow: hidden; background: var(--color-bg-sidebar);
  transition: background-color 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}
.login-theme-toggle {
  position: absolute; top: 20px; right: 20px; z-index: 10;
  width: 40px; height: 40px; border: 1px solid rgba(255,255,255,0.15);
  background: rgba(255,255,255,0.08); border-radius: var(--radius-md);
  cursor: pointer; display: flex; align-items: center; justify-content: center;
  font-size: 18px; backdrop-filter: blur(8px); transition: all var(--transition-fast);
}
.login-theme-toggle:hover { background: rgba(255,255,255,0.15); }

.login-bg { position: absolute; inset: 0; overflow: hidden; }
.login-bg-shape { position: absolute; border-radius: 50%; filter: blur(80px); opacity: 0.5; animation: float 20s ease-in-out infinite; }
.login-bg-shape--1 { width: 600px; height: 600px; background: linear-gradient(135deg,#4f46e5,#7c3aed); top: -200px; left: -100px; }
.login-bg-shape--2 { width: 500px; height: 500px; background: linear-gradient(135deg,#06b6d4,#3b82f6); bottom: -150px; right: -100px; animation-delay: -7s; }
.login-bg-shape--3 { width: 400px; height: 400px; background: linear-gradient(135deg,#8b5cf6,#ec4899); top: 50%; left: 50%; transform: translate(-50%,-50%); animation-delay: -14s; }
@keyframes float {
  0%,100% { transform: translate(0,0) scale(1); }
  25% { transform: translate(30px,-30px) scale(1.05); }
  50% { transform: translate(-20px,20px) scale(0.95); }
  75% { transform: translate(20px,10px) scale(1.02); }
}

.login-container {
  position: relative; z-index: 1; display: flex;
  width: 960px; max-width: calc(100vw - 32px);
  min-height: 600px; border-radius: 24px;
  overflow: hidden; box-shadow: 0 25px 60px rgba(0,0,0,0.4);
}

/* Left branding */
.login-branding {
  width: 360px; flex-shrink: 0;
  background: linear-gradient(135deg, rgba(79,70,229,0.9), rgba(124,58,237,0.9));
  backdrop-filter: blur(20px); padding: 48px 36px;
  display: flex; flex-direction: column; justify-content: center;
  position: relative; overflow: hidden;
}
.login-branding::before {
  content: ''; position: absolute; inset: 0;
  background: url("data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%23ffffff' fill-opacity='0.05'%3E%3Cpath d='M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E");
}
.login-branding-content { position: relative; z-index: 1; }
.login-logo {
  width: 64px; height: 64px; background: rgba(255,255,255,0.15);
  border-radius: 16px; display: flex; align-items: center; justify-content: center;
  margin-bottom: 24px; backdrop-filter: blur(10px); border: 1px solid rgba(255,255,255,0.2);
}
.login-logo-icon { font-size: 30px; color: #fff; }
.login-brand-title { font-size: 32px; font-weight: 700; color: #fff; margin: 0 0 8px; letter-spacing: -0.02em; }
.login-brand-subtitle { font-size: 16px; color: rgba(255,255,255,0.8); margin: 0 0 40px; }
.login-features { display: flex; flex-direction: column; gap: 16px; }
.login-feature-item { display: flex; align-items: center; gap: 12px; color: rgba(255,255,255,0.9); font-size: 14px; }
.login-feature-dot { width: 8px; height: 8px; border-radius: 50%; background: #22d3ee; flex-shrink: 0; }

/* Right form */
.login-form-wrapper {
  flex: 1; background: var(--color-bg-card);
  padding: 36px 40px; display: flex; flex-direction: column;
  justify-content: center; overflow-y: auto;
}
.login-form-inner { max-width: 400px; width: 100%; margin: 0 auto; }
.login-form-header { margin-bottom: 20px; }
.login-form-header h2 { font-size: 24px; font-weight: 700; color: var(--color-text-primary); margin: 0 0 6px; }
.login-form-header p { font-size: 14px; color: var(--color-text-tertiary); margin: 0; }

/* ===== Portal Pill Selector ===== */
.portal-selector {
  display: flex; gap: 8px; margin-bottom: 8px;
  overflow-x: auto; scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
}
.portal-selector::-webkit-scrollbar { display: none; }

.portal-pill {
  flex: 1 1 0;
  display: flex; align-items: center; justify-content: center; gap: 6px;
  padding: 10px 12px;
  border: 1.5px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-bg-card);
  cursor: pointer;
  transition: all 0.25s ease;
  white-space: nowrap;
  --pill-color: var(--color-text-tertiary);
}
.portal-pill:hover {
  border-color: var(--color-text-tertiary);
  background: var(--color-bg-hover);
}
.portal-pill--active {
  border-color: var(--pill-color);
  background: color-mix(in srgb, var(--pill-color) 8%, var(--color-bg-card));
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--pill-color) 20%, transparent);
}
.portal-pill-icon {
  font-size: 15px;
  color: var(--color-text-tertiary);
  transition: color 0.25s ease;
}
.portal-pill--active .portal-pill-icon {
  color: var(--pill-color);
}
.portal-pill-title {
  font-size: 13px; font-weight: 500;
  color: var(--color-text-secondary);
  transition: color 0.25s ease;
}
.portal-pill--active .portal-pill-title {
  font-weight: 600;
  color: var(--pill-color);
}

/* Active description line */
.portal-active-desc {
  display: flex; align-items: center; gap: 6px;
  font-size: 12px; color: var(--color-text-tertiary);
  margin-bottom: 20px; padding: 0 2px;
  min-height: 18px;
}
.portal-active-dot {
  width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0;
  transition: background 0.25s ease;
}

/* Form */
.login-form :deep(.ant-form-item) { margin-bottom: 16px; }
.login-input {
  height: 46px !important; border-radius: var(--radius-lg) !important;
  border: 1.5px solid var(--color-border) !important;
  background: var(--color-bg-input) !important;
  font-size: 14px !important; transition: all 0.2s ease !important;
  display: flex !important; align-items: center !important;
}
.login-input :deep(input) {
  height: 100% !important;
  line-height: normal !important; /* Allow flex container to center */
}
.login-input:hover { border-color: var(--color-text-tertiary) !important; }
:deep(.ant-input-affix-wrapper:focus),
:deep(.ant-input-affix-wrapper-focused) {
  border-color: var(--color-primary) !important;
  box-shadow: 0 0 0 3px rgba(79,70,229,0.1) !important;
}
.login-input-icon { color: var(--color-text-tertiary); font-size: 15px; }
.login-select {
  width: 100% !important;
}
.login-select :deep(.ant-select-selector) {
  height: 46px !important;
  border-radius: var(--radius-lg) !important;
  border: 1.5px solid var(--color-border) !important;
  background: var(--color-bg-input) !important;
  font-size: 14px !important;
  display: flex !important;
  align-items: center !important;
  padding: 0 14px !important;
  transition: all 0.2s ease !important;
}
.login-select:hover :deep(.ant-select-selector) {
  border-color: var(--color-text-tertiary) !important;
}
.login-select.ant-select-focused :deep(.ant-select-selector) {
  border-color: var(--color-primary) !important;
  box-shadow: 0 0 0 3px rgba(79,70,229,0.1) !important;
}
.login-options { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }

.login-btn {
  height: 46px !important; border-radius: var(--radius-lg) !important;
  font-size: 15px !important; font-weight: 600 !important; border: none !important;
  box-shadow: 0 4px 16px rgba(79,70,229,0.3) !important;
  transition: all 0.3s ease !important;
  display: flex !important; align-items: center !important;
  justify-content: center !important; text-align: center !important; line-height: 1 !important;
}
.login-btn:hover {
  box-shadow: 0 6px 24px rgba(79,70,229,0.4) !important;
  transform: translateY(-1px) !important; opacity: 0.95;
}

.login-footer { text-align: center; margin-top: 24px; color: var(--color-text-tertiary); font-size: 13px; }

/* Mock accounts quick-fill */
.mock-accounts {
  margin-bottom: 16px; padding: 12px;
  background: color-mix(in srgb, var(--color-primary, #4f46e5) 5%, var(--color-bg-card));
  border: 1px dashed color-mix(in srgb, var(--color-primary, #4f46e5) 30%, transparent);
  border-radius: var(--radius-lg);
}
.mock-accounts-label {
  font-size: 12px; color: var(--color-text-tertiary); margin-bottom: 8px;
}
.mock-accounts-list { display: flex; flex-wrap: wrap; gap: 6px; }
.mock-account-tag { cursor: pointer; font-size: 12px; }
.mock-account-tag:hover { opacity: 0.8; }

.login-mobile-brand {
  display: none; position: absolute; top: 24px; left: 24px; z-index: 2;
  color: #fff; font-size: 18px; font-weight: 700; align-items: center; gap: 8px;
}
.login-mobile-logo { font-size: 24px; }

@media (max-width: 768px) {
  .login-branding { display: none; }
  .login-container {
    min-height: auto;
    border-radius: 20px;
    height: auto; /* Allow content to dictate height */
    margin: 20px 0; /* Add margin to prevent sticking to edges */
  }
  .login-form-wrapper { padding: 32px 24px; border-radius: 20px; }
  .login-mobile-brand { display: flex; }
}

@media (max-width: 480px) {
  .login-page { align-items: flex-start; overflow-y: auto; } /* Allow scrolling on small screens */
  .login-container {
    max-width: calc(100vw - 24px);
    margin: 60px auto 20px; /* Top margin for mobile brand */
    box-shadow: 0 10px 30px rgba(0,0,0,0.15); /* Softer shadow */
  }
  .login-form-wrapper { padding: 24px 20px; }
  .portal-pill-title { font-size: 12px; }
  .login-form-header h2 { font-size: 20px; }
  .login-input { height: 44px !important; padding: 0 11px !important; }
  .login-btn { height: 42px !important; }
}
</style>
