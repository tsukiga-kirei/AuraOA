<script setup lang="ts">
import {
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  BellOutlined,
  SwapOutlined,
  CheckOutlined,
} from '@ant-design/icons-vue'

defineProps<{
  collapsed: boolean
  isMobile: boolean
  notificationCount?: number
}>()

const emit = defineEmits<{
  (e: 'toggleSidebar'): void
  (e: 'toggleMobileMenu'): void
}>()

const { isDark, toggle: toggleTheme } = useTheme()
const { t } = useI18n()
const { allRoles, activeRole, switchRole, getMenu } = useAuth()

import type { UserRoleAssignment } from '~/composables/useMockData'

// ===== Role Switching =====
// Group roles by type for organized display
const systemRoles = computed(() => allRoles.value.filter(r => r.role === 'system_admin'))
const tenantAdminRoles = computed(() => allRoles.value.filter(r => r.role === 'tenant_admin'))
const businessRoles = computed(() => allRoles.value.filter(r => r.role === 'business'))

// Show switcher when user has more than one role assignment
const showRoleSwitcher = computed(() => allRoles.value.length > 1)

const activeRoleId = computed(() => activeRole.value?.id || '')
const activeRoleLabel = computed(() => activeRole.value?.label || '')

const roleTypeLabel = (role: string) => {
  if (role === 'system_admin') return t('login.portal.systemAdmin')
  if (role === 'tenant_admin') return t('login.portal.tenantAdmin')
  return t('login.portal.business')
}

const roleTypeIcon = (role: string) => {
  if (role === 'system_admin') return '🛡️'
  if (role === 'tenant_admin') return '⚙️'
  return '📊'
}

const handleSwitchRole = async (role: UserRoleAssignment) => {
  if (role.id === activeRoleId.value) return
  await switchRole(role.id)
  await getMenu()
  navigateTo('/overview')
}
</script>

<template>
  <header class="app-header">
    <div class="app-header-left">
      <button
        class="header-toggle"
        @click="isMobile ? emit('toggleMobileMenu') : emit('toggleSidebar')"
      >
        <MenuUnfoldOutlined v-if="collapsed && !isMobile" />
        <MenuFoldOutlined v-else-if="!isMobile" />
        <MenuUnfoldOutlined v-else />
      </button>
    </div>

    <div class="app-header-right">
      <a-tooltip :title="t('header.toggleTheme')" placement="bottom" :mouse-enter-delay="0.5">
        <button
          class="header-action theme-toggle-btn"
          :class="{ 'theme-toggle-btn--dark': isDark }"
          @click="toggleTheme"
          :aria-label="isDark ? t('header.lightMode') : t('header.darkMode')"
        >
          <span class="theme-toggle-track">
            <span class="theme-toggle-thumb">
              <transition name="theme-icon" mode="out-in">
                <svg v-if="isDark" key="moon" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="currentColor" stroke="none"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
                <svg v-else key="sun" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="4"/><line x1="12" y1="2" x2="12" y2="5"/><line x1="12" y1="19" x2="12" y2="22"/><line x1="4.93" y1="4.93" x2="6.76" y2="6.76"/><line x1="17.24" y1="17.24" x2="19.07" y2="19.07"/><line x1="2" y1="12" x2="5" y2="12"/><line x1="19" y1="12" x2="22" y2="12"/><line x1="4.93" y1="19.07" x2="6.76" y2="17.24"/><line x1="17.24" y1="6.76" x2="19.07" y2="4.93"/></svg>
              </transition>
            </span>
          </span>
        </button>
      </a-tooltip>

      <!-- Role Switcher Dropdown -->
      <a-dropdown v-if="showRoleSwitcher" placement="bottomRight" :trigger="['click']">
        <button class="header-action role-switch-btn" :title="t('header.switchRole')">
          <SwapOutlined />
          <span class="role-switch-label">{{ activeRoleLabel }}</span>
        </button>
        <template #overlay>
          <div class="role-dropdown">
            <div class="role-dropdown-title">{{ t('header.switchRole') }}</div>

            <!-- System Admin roles -->
            <template v-if="systemRoles.length">
              <div class="role-dropdown-group">
                <span class="role-dropdown-group-icon">🛡️</span>
                <span>{{ t('login.portal.systemAdmin') }}</span>
              </div>
              <div
                v-for="role in systemRoles"
                :key="role.id"
                class="role-dropdown-item"
                :class="{ 'role-dropdown-item--active': role.id === activeRoleId }"
                @click="handleSwitchRole(role)"
              >
                <div class="role-dropdown-info">
                  <div class="role-dropdown-name">{{ role.label }}</div>
                </div>
                <CheckOutlined v-if="role.id === activeRoleId" class="role-dropdown-check" />
              </div>
            </template>

            <!-- Tenant Admin roles -->
            <template v-if="tenantAdminRoles.length">
              <div class="role-dropdown-group">
                <span class="role-dropdown-group-icon">⚙️</span>
                <span>{{ t('login.portal.tenantAdmin') }}</span>
              </div>
              <div
                v-for="role in tenantAdminRoles"
                :key="role.id"
                class="role-dropdown-item"
                :class="{ 'role-dropdown-item--active': role.id === activeRoleId }"
                @click="handleSwitchRole(role)"
              >
                <div class="role-dropdown-info">
                  <div class="role-dropdown-name">{{ role.tenant_name }}</div>
                  <div class="role-dropdown-desc">租户管理员</div>
                </div>
                <CheckOutlined v-if="role.id === activeRoleId" class="role-dropdown-check" />
              </div>
            </template>

            <!-- Business roles -->
            <template v-if="businessRoles.length">
              <div class="role-dropdown-group">
                <span class="role-dropdown-group-icon">📊</span>
                <span>{{ t('login.portal.business') }}</span>
              </div>
              <div
                v-for="role in businessRoles"
                :key="role.id"
                class="role-dropdown-item"
                :class="{ 'role-dropdown-item--active': role.id === activeRoleId }"
                @click="handleSwitchRole(role)"
              >
                <div class="role-dropdown-info">
                  <div class="role-dropdown-name">{{ role.tenant_name }}</div>
                  <div class="role-dropdown-desc">业务用户</div>
                </div>
                <CheckOutlined v-if="role.id === activeRoleId" class="role-dropdown-check" />
              </div>
            </template>
          </div>
        </template>
      </a-dropdown>

      <a-tooltip :title="t('header.notifications')" placement="bottom" :mouse-enter-delay="0.5">
        <a-badge :count="notificationCount ?? 0" :offset="[-4, 4]">
          <button class="header-action">
            <BellOutlined />
          </button>
        </a-badge>
      </a-tooltip>
    </div>
  </header>
</template>

<style scoped>
.app-header {
  height: var(--header-height);
  display: flex; align-items: center; justify-content: space-between;
  padding: 0 24px;
  position: sticky; top: 0; z-index: 50;
  background: var(--color-bg-page);
}
.app-header-left { display: flex; align-items: center; gap: 16px; }
.app-header-right { display: flex; align-items: center; gap: 8px; }

.header-toggle,
.header-action {
  width: 36px; height: 36px;
  border: none; background: transparent;
  border-radius: var(--radius-md);
  cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  font-size: 18px;
  color: var(--color-text-secondary);
  transition: all var(--transition-fast);
  outline: none;
}
.header-toggle:hover,
.header-action:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}
.header-toggle:focus-visible,
.header-action:focus-visible {
  background: var(--color-bg-hover);
  color: var(--color-primary);
  box-shadow: 0 0 0 2px var(--color-primary-bg), 0 0 0 4px rgba(79, 70, 229, 0.25);
}

/* Role Switcher Button */
.role-switch-btn {
  width: auto !important;
  padding: 0 12px !important;
  gap: 6px;
  font-size: 13px;
  border: 1px solid var(--color-border) !important;
  border-radius: 20px !important;
  height: 32px !important;
  background: var(--color-bg-card) !important;
}
.role-switch-btn:hover {
  border-color: var(--color-primary) !important;
  color: var(--color-primary) !important;
  background: var(--color-primary-bg) !important;
}
.role-switch-label {
  font-size: 12px;
  font-weight: 500;
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Role Dropdown panel */
.role-dropdown {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  padding: 8px;
  min-width: 260px;
  max-width: 320px;
}
.role-dropdown-title {
  padding: 8px 12px 4px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--color-text-tertiary);
}

/* Group header */
.role-dropdown-group {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 12px 4px;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-secondary);
  border-top: 1px solid var(--color-border-light);
  margin-top: 4px;
}
.role-dropdown-group:first-of-type {
  border-top: none;
  margin-top: 0;
}
.role-dropdown-group-icon {
  font-size: 14px;
}

/* Individual role item */
.role-dropdown-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px 8px 28px;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all 0.15s ease;
}
.role-dropdown-item:hover {
  background: var(--color-bg-hover);
}
.role-dropdown-item--active {
  background: var(--color-primary-bg);
}
.role-dropdown-item--active:hover {
  background: var(--color-primary-bg);
}
.role-dropdown-info {
  flex: 1;
  min-width: 0;
}
.role-dropdown-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-primary);
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.role-dropdown-desc {
  font-size: 11px;
  color: var(--color-text-tertiary);
  line-height: 1.3;
  margin-top: 1px;
}
.role-dropdown-item--active .role-dropdown-name {
  color: var(--color-primary);
}
.role-dropdown-check {
  color: var(--color-primary);
  font-size: 14px;
  flex-shrink: 0;
}

/* Theme toggle pill switch */
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
  transition: transform 0.35s cubic-bezier(0.4, 0, 0.2, 1),
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

@media (max-width: 768px) {
  .app-header { padding: 0 16px; }
  .role-switch-label { display: none; }
  .role-dropdown { min-width: 220px; }
}
</style>
