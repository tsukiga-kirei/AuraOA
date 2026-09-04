<script setup lang="ts">
import { ref, computed } from 'vue'
import { message } from 'ant-design-vue'
import {
  UserOutlined,
  LockOutlined,
  DashboardOutlined,
  SettingOutlined,
  ControlOutlined,
} from '@ant-design/icons-vue'

import type { TenantOption } from '~/types/auth'
import { useI18n } from '~/composables/useI18n'

definePageMeta({ layout: false, middleware: 'auth' })

// 鉴权与主题相关 composable
const { login } = useAuth()
const { restore: restoreTheme } = useTheme()
const { t } = useI18n()
const config = useRuntimeConfig()

// 可选租户列表，用于业务用户和租户管理员登录时选择租户
const tenants = ref<TenantOption[]>([])

// 页面挂载时：恢复主题偏好，恢复记住的登录信息，并从后端拉取租户列表
onMounted(async () => {
  restoreTheme()
  // 先恢复记住的登录信息（tenant_id 在租户列表加载后会自动匹配）
  restoreRemembered()
  try {
    const res = await $fetch<{ code: number; message: string; data: TenantOption[]; trace_id: string }>(
      `${config.public.apiBase}/api/tenants/list`
    )
    if (res.code === 0 && res.data) {
      tenants.value = res.data
    }
  } catch (e) {
    console.error('[login] 获取租户列表失败:', e)
  }
})

// 登录入口类型：业务用户 / 租户管理员 / 系统管理员
type PortalType = 'business' | 'tenant_admin' | 'system_admin'

// 三种入口的展示配置（图标、标题、描述、主题色）
const portals = computed(() => [
  { key: 'business' as PortalType, icon: DashboardOutlined, title: t('login.portal.business'), desc: t('login.portal.businessDesc'), color: 'var(--color-role-business)' },
  { key: 'tenant_admin' as PortalType, icon: SettingOutlined, title: t('login.portal.tenantAdmin'), desc: t('login.portal.tenantAdminDesc'), color: 'var(--color-role-tenant)' },
  { key: 'system_admin' as PortalType, icon: ControlOutlined, title: t('login.portal.systemAdmin'), desc: t('login.portal.systemAdminDesc'), color: 'var(--color-role-system)' },
])

// 当前选中的登录入口
const activePortal = ref<PortalType>('business')
// 登录表单数据
const form = ref<{ username: string; password: string; tenant_id: string | undefined }>({ username: '', password: '', tenant_id: undefined })
// 登录请求加载状态
const loading = ref(false)
// 记住我选项
const rememberMe = ref(false)
// 当前激活入口的完整配置
const currentPortal = computed(() => portals.value.find(p => p.key === activePortal.value)!)
// 当前激活入口的索引，用于驱动分段控件滑块动画
const activePortalIndex = computed(() => Math.max(0, portals.value.findIndex(p => p.key === activePortal.value)))
const activePortalOffset = computed(() => {
  const offsets = ['0px', 'calc(100% + 4px)', 'calc(200% + 8px)']
  return offsets[activePortalIndex.value] || offsets[0]
})

// 记住登录信息的 localStorage key
const REMEMBER_ME_KEY = 'login_remember'

/** 从 localStorage 恢复记住的登录信息 */
const restoreRemembered = () => {
  try {
    const raw = localStorage.getItem(REMEMBER_ME_KEY)
    if (!raw) return
    const saved = JSON.parse(raw)
    rememberMe.value = true
    form.value.username = saved.username || ''
    form.value.password = saved.password || ''
    if (saved.portal && ['business', 'tenant_admin', 'system_admin'].includes(saved.portal)) {
      activePortal.value = saved.portal as PortalType
    }
    // tenant_id 需要等租户列表加载完成后再回填
    if (saved.tenant_id) {
      form.value.tenant_id = saved.tenant_id
    }
  } catch { /* 忽略解析错误 */ }
}

// 提交登录表单
const handleLogin = async () => {
  if (!form.value.username || !form.value.password) {
    message.warning(t('login.emptyWarning'))
    return
  }
  loading.value = true
  try {
    const result = await login({
      ...form.value,
      tenant_id: form.value.tenant_id || '',
      preferred_role: activePortal.value
    })
    if (result.ok) {
      // 登录成功后才处理"记住我"逻辑
      if (rememberMe.value) {
        localStorage.setItem(REMEMBER_ME_KEY, JSON.stringify({
          username: form.value.username,
          password: form.value.password,
          portal: activePortal.value,
          tenant_id: form.value.tenant_id || '',
        }))
      } else {
        localStorage.removeItem(REMEMBER_ME_KEY)
      }
      message.success(t('login.successRedirect'))
      navigateTo('/overview')
    } else {
      message.error(result.errorMsg || t('login.failed'))
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

    <div class="login-theme-floating">
      <ThemeSwitcher />
    </div>

    <div class="login-container">
      <!--左：品牌-->
      <div class="login-branding">
        <div class="login-branding-content">
          <div class="login-logo">
            <img src="/favicon.svg" alt="AuraOA" width="30" height="30" />
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

      <!--右：登录表格-->
      <div class="login-form-wrapper">
        <div class="login-form-inner">
          <div class="login-form-header">
            <h2>{{ t('login.welcomeBack') }}</h2>
            <p>{{ t('login.selectIdentity') }}</p>
          </div>

          <!--入口选择器：水平药丸选项卡，固定尺寸-->
          <div
            class="portal-selector"
            :style="{ '--active-offset': activePortalOffset, '--active-color': currentPortal.color }"
          >
            <div
              v-for="portal in portals"
              :key="portal.key"
              class="portal-pill"
              :class="{ 'portal-pill--active': activePortal === portal.key }"
              @click="activePortal = portal.key"
            >
              <component :is="portal.icon" class="portal-pill-icon" />
              <span class="portal-pill-title">{{ portal.title }}</span>
            </div>
          </div>

          <!--活动门户描述（外部选择器，固定位置）-->
          <div class="portal-active-desc">
            <span class="portal-active-dot" :style="{ background: currentPortal.color }" />
            {{ currentPortal.desc }}
          </div>

          <a-form layout="vertical" class="login-form">
            <a-form-item v-if="activePortal !== 'system_admin'">
              <a-select v-model:value="form.tenant_id" :placeholder="t('login.tenantRequired')" size="large" class="login-select">
                <a-select-option v-for="tenant in tenants" :key="tenant.id" :value="tenant.id">
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

            <a-form-item>
              <a-button
                type="primary" block size="large" :loading="loading"
                class="login-btn"
                @click="handleLogin"
              >
                {{ loading ? t('login.logging') : t('login.loginAs', currentPortal.title) }}
              </a-button>
            </a-form-item>
          </a-form>
          <div class="login-footer"><span>{{ t('app.name') }} © 2026</span></div>
        </div>
      </div>
    </div>

    <div class="login-mobile-brand">
      <img class="login-mobile-logo" src="/favicon.svg" alt="AuraOA" width="24" height="24" />
      <span>{{ t('app.name') }}</span>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh; display: flex; align-items: center; justify-content: center;
  position: relative; overflow: hidden;
  background: var(--color-login-bg);
  transition: background 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}
.login-theme-floating {
  position: absolute;
  top: 20px;
  right: 20px;
  z-index: 10;
}

.login-bg { position: absolute; inset: 0; overflow: hidden; }
.login-bg-shape { position: absolute; border-radius: 50%; filter: blur(80px); opacity: var(--color-login-blob-opacity); animation: float 20s ease-in-out infinite; }
.login-bg-shape--1 { width: 600px; height: 600px; background: linear-gradient(135deg, var(--color-primary), var(--color-primary-light)); top: -200px; left: -100px; }
.login-bg-shape--2 { width: 500px; height: 500px; background: linear-gradient(135deg, var(--color-accent), var(--color-info)); bottom: -150px; right: -100px; animation-delay: -7s; }
.login-bg-shape--3 { width: 400px; height: 400px; background: linear-gradient(135deg, var(--color-primary-lighter), var(--color-accent-light)); top: 50%; left: 50%; transform: translate(-50%,-50%); animation-delay: -14s; }
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
  overflow: hidden; box-shadow: var(--color-login-container-shadow);
}

/* 左侧品牌展示区域 */
.login-branding {
  width: 360px; flex-shrink: 0;
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-light));
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
  position: relative;
  width: 76px; height: 76px;
  border-radius: 18px; display: flex; align-items: center; justify-content: center;
  margin-bottom: 24px;
  isolation: isolate;
}
.login-logo::before {
  content: '';
  position: absolute;
  inset: -18px;
  z-index: -1;
  border-radius: 36px;
  background:
    radial-gradient(circle at 30% 20%, rgba(255,255,255,0.72), transparent 30%),
    radial-gradient(circle at 76% 76%, rgba(34,211,238,0.78), transparent 38%),
    radial-gradient(circle at 38% 78%, rgba(16,185,129,0.58), transparent 34%),
    linear-gradient(135deg, rgba(165,180,252,0.78), rgba(6,182,212,0.42));
  filter: blur(14px);
  opacity: 0.86;
  animation: login-logo-breathe 2.15s ease-in-out infinite;
}
.login-logo::after {
  content: '';
  position: absolute;
  inset: -1px;
  border-radius: 19px;
  background: linear-gradient(135deg, rgba(255,255,255,0.30), rgba(255,255,255,0.04) 52%, rgba(103,232,249,0.24));
  opacity: 0.45;
  pointer-events: none;
  mix-blend-mode: screen;
}
.login-logo img {
  position: relative;
  z-index: 1;
  display: block;
  width: 100%;
  height: 100%;
  border-radius: 18px;
  box-shadow:
    0 18px 42px rgba(17,24,39,0.24),
    0 0 0 1px rgba(255,255,255,0.08);
}
@keyframes login-logo-breathe {
  0%, 100% { transform: scale(0.92); opacity: 0.48; }
  50% { transform: scale(1.18); opacity: 1; }
}
.login-logo-icon { font-size: 30px; color: #fff; }
.login-brand-title { font-size: 32px; font-weight: 700; color: #fff; margin: 0 0 8px; letter-spacing: 0; }
.login-brand-subtitle { font-size: 16px; color: rgba(255,255,255,0.8); margin: 0 0 40px; }
.login-features { display: flex; flex-direction: column; gap: 16px; }
.login-feature-item { display: flex; align-items: center; gap: 12px; color: rgba(255,255,255,0.9); font-size: 14px; }
.login-feature-dot { width: 8px; height: 8px; border-radius: 50%; background: #22d3ee; flex-shrink: 0; }

/* 右侧登录表单区域 */
.login-form-wrapper {
  flex: 1; background: var(--color-bg-card);
  padding: 36px 40px; display: flex; flex-direction: column;
  justify-content: center; overflow-y: auto;
  border-left: 1px solid var(--color-border-light);
}
.login-form-inner { max-width: 400px; width: 100%; margin: 0 auto; }
.login-form-header { margin-bottom: 20px; }
.login-form-header h2 { font-size: 24px; font-weight: 700; color: var(--color-text-primary); margin: 0 0 6px; font-family: var(--font-display); }
.login-form-header p { font-size: 14px; color: var(--color-text-tertiary); margin: 0; }

/* ===== 入口选择器（连体分段控件） ===== */
.portal-selector {
  --segment-gap: 4px;
  --segment-padding: 4px;
  --segment-track-remainder: 16px;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: var(--segment-gap);
  margin-bottom: 8px;
  overflow-x: auto; scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
  position: relative;
  padding: var(--segment-padding);
  border: 1px solid var(--color-segment-track-border);
  border-radius: 16px;
  background: var(--color-segment-track);
  box-shadow: inset 0 1px 2px var(--color-segment-shadow);
  isolation: isolate;
  --active-offset: 0px;
  --active-color: var(--color-primary);
}
.portal-selector::before {
  content: "";
  position: absolute;
  top: var(--segment-padding);
  bottom: var(--segment-padding);
  left: var(--segment-padding);
  width: calc((100% - var(--segment-track-remainder)) / 3);
  border-radius: 12px;
  background: var(--color-segment-thumb);
  border: 1px solid var(--color-segment-track-border);
  box-shadow: 0 4px 12px rgba(15, 23, 42, 0.08), 0 1px 3px rgba(15, 23, 42, 0.05);
  transform: translateX(var(--active-offset));
  transition:
    transform 0.38s cubic-bezier(0.22, 1, 0.36, 1),
    background-color 0.24s ease,
    box-shadow 0.24s ease;
  will-change: transform;
  z-index: 0;
}
.portal-selector::-webkit-scrollbar { display: none; }

.portal-pill {
  min-width: 0;
  display: flex; align-items: center; justify-content: center; gap: 6px;
  min-height: 38px;
  padding: 8px 10px;
  border: 0;
  border-radius: 12px;
  background: transparent;
  cursor: pointer;
  transition:
    color 0.22s ease,
    transform 0.22s ease,
    opacity 0.22s ease;
  white-space: nowrap;
  position: relative;
  z-index: 1;
}
.portal-pill:hover:not(.portal-pill--active) {
  background: color-mix(in srgb, var(--color-segment-thumb) 50%, transparent);
}
.portal-pill:hover:not(.portal-pill--active) .portal-pill-icon,
.portal-pill:hover:not(.portal-pill--active) .portal-pill-title {
  color: var(--color-segment-inactive-text);
}
.portal-pill--active {
  transform: translateY(-0.5px);
}
.portal-pill-icon {
  font-size: 15px;
  color: var(--color-segment-inactive-icon);
  transition: color 0.22s ease;
}
.portal-pill--active .portal-pill-icon {
  color: var(--active-color);
}
.portal-pill-title {
  font-size: 13px; font-weight: 500;
  color: var(--color-segment-inactive-text);
  transition: color 0.22s ease;
}
.portal-pill--active .portal-pill-title {
  font-weight: 600;
  color: var(--active-color);
}

/* 当前选中入口的描述行 */
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

/* 表单输入样式 */
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
  line-height: normal !important; /* 允许弹性容器垂直居中 */
}

.login-input:hover { border-color: var(--color-text-tertiary) !important; }
:deep(.ant-input-affix-wrapper:focus),
:deep(.ant-input-affix-wrapper-focused) {
  border-color: var(--color-primary) !important;
  box-shadow: 0 0 0 3px var(--color-primary-ring) !important;
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
  box-shadow: 0 0 0 3px var(--color-primary-ring) !important;
}
.login-options { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }

.login-btn {
  height: 46px !important; border-radius: var(--radius-lg) !important;
  font-size: 15px !important; font-weight: 600 !important; border: none !important;
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-light)) !important;
  box-shadow: 0 4px 16px var(--color-primary-shadow) !important;
  transition: all 0.3s ease !important;
  display: flex !important; align-items: center !important;
  justify-content: center !important; text-align: center !important; line-height: 1 !important;
}
.login-btn:hover {
  box-shadow: 0 6px 24px var(--color-primary-shadow) !important;
  transform: translateY(-1px) !important; opacity: 0.95;
}

.login-footer { text-align: center; margin-top: 24px; color: var(--color-text-tertiary); font-size: 13px; }

.login-mobile-brand {
  display: none; position: absolute; top: 24px; left: 24px; z-index: 2;
  color: #fff; font-size: 18px; font-weight: 700; align-items: center; gap: 8px;
}
.login-mobile-logo { width: 24px; height: 24px; flex-shrink: 0; }

@media (max-width: 768px) {
  .login-branding { display: none; }
  .login-container {
    min-height: auto;
    border-radius: 20px;
    height: auto; /* 由内容撑开高度 */
    margin: 20px 0; /* 防止贴边 */
  }
  .login-form-wrapper { padding: 32px 24px; border-radius: 20px; }
  .login-mobile-brand { display: flex; }
}

@media (max-width: 480px) {
  .login-page { align-items: flex-start; overflow-y: auto; } /* 小屏幕允许滚动 */
  .login-container {
    max-width: calc(100vw - 24px);
    margin: 60px auto 20px; /* 为顶部移动端品牌留出空间 */
    box-shadow: 0 10px 30px rgba(0,0,0,0.15); /* 阴影更柔和 */
  }
  .login-form-wrapper { padding: 24px 20px; }
  .portal-pill-title { font-size: 12px; }
  .login-form-header h2 { font-size: 20px; }
  .login-input { height: 44px !important; padding: 0 11px !important; }
  .login-btn { height: 42px !important; }
}

@media (prefers-reduced-motion: reduce) {
  .login-logo::before {
    animation: none;
  }
}
</style>
