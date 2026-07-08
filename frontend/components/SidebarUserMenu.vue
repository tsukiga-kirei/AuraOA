<script setup lang="ts">
import {
  LogoutOutlined,
  UserOutlined,
  SettingOutlined,
  BellOutlined,
  CheckOutlined,
  RightOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'

import type { RoleInfo } from '~/types/auth'
import type { ThemeMode } from '~/composables/useTheme'

defineProps<{
  collapsed: boolean
  mobileMenuOpen: boolean
}>()

const emit = defineEmits<{
  (e: 'navigate', path: string): void
}>()

const { t } = useI18n()
const { currentUser, logout, allRoles, activeRole, switchRole, getMenu } = useAuth()
const { mode, setTheme } = useTheme()
const { unreadCount } = useNotifications()

const menuOpen = ref(false)
const activeSubmenu = ref<'theme' | 'role' | null>(null)

const displayName = computed(() => currentUser.value?.display_name || t('sidebar.defaultUser'))
const showRoleSwitcher = computed(() => allRoles.value.length > 1)
const activeRoleId = computed(() => activeRole.value?.id || '')
const unreadLabel = computed(() => (unreadCount.value > 99 ? '99+' : String(unreadCount.value)))

const systemRoles = computed(() => allRoles.value.filter(r => r.role === 'system_admin'))
const tenantAdminRoles = computed(() => allRoles.value.filter(r => r.role === 'tenant_admin'))
const businessRoles = computed(() => allRoles.value.filter(r => r.role === 'business'))

const themeOptions: { mode: ThemeMode; labelKey: string }[] = [
  { mode: 'light', labelKey: 'header.lightMode' },
  { mode: 'dark', labelKey: 'header.darkMode' },
  { mode: 'warm', labelKey: 'header.warmMode' },
]

const menuRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLElement | null>(null)
const themeRowRef = ref<HTMLElement | null>(null)
const roleRowRef = ref<HTMLElement | null>(null)

const submenuTop = computed(() => {
  const row = activeSubmenu.value === 'theme'
    ? themeRowRef.value
    : activeSubmenu.value === 'role'
      ? roleRowRef.value
      : null
  return row?.offsetTop ?? 0
})

function closeMenu() {
  menuOpen.value = false
  activeSubmenu.value = null
}

function toggleMenu() {
  menuOpen.value = !menuOpen.value
  if (!menuOpen.value) activeSubmenu.value = null
}

function openSubmenu(key: 'theme' | 'role') {
  activeSubmenu.value = key
}

function closeSubmenu() {
  activeSubmenu.value = null
}

function pickTheme(next: ThemeMode) {
  if (mode.value !== next) setTheme(next)
}

function navigate(path: string) {
  closeMenu()
  emit('navigate', path)
}

async function handleSwitchRole(role: RoleInfo) {
  if (role.id === activeRoleId.value) return
  closeMenu()
  const result = await switchRole(role.id)
  if (!result.ok) {
    message.error(result.errorMsg || t('header.switchRoleFailed'))
    return
  }
  await getMenu()
  navigateTo('/overview')
}

function handleLogout() {
  closeMenu()
  logout()
}

function onDocumentClick(event: MouseEvent) {
  if (!menuOpen.value) return
  const target = event.target as Node
  if (menuRef.value?.contains(target) || triggerRef.value?.contains(target)) return
  closeMenu()
}

onMounted(() => document.addEventListener('click', onDocumentClick))
onUnmounted(() => document.removeEventListener('click', onDocumentClick))
</script>

<template>
  <div class="sidebar-user-menu" :class="{ 'sidebar-user-menu--open': menuOpen }">
    <div
      v-if="menuOpen"
      ref="menuRef"
      class="account-menu-shell"
      role="menu"
      @click.stop
    >
      <div class="account-menu-panel">
        <!--主菜单：始终完整显示，不被压缩-->
        <div class="account-menu">
          <div
            ref="themeRowRef"
            class="account-menu-item account-menu-item--submenu"
            :class="{ 'account-menu-item--active': activeSubmenu === 'theme' }"
            role="menuitem"
            @mouseenter="openSubmenu('theme')"
          >
            <span class="account-menu-icon" aria-hidden="true">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="12" cy="12" r="4" />
                <path d="M12 2v2" /><path d="M12 20v2" />
                <path d="m4.93 4.93 1.41 1.41" /><path d="m17.66 17.66 1.41 1.41" />
                <path d="M2 12h2" /><path d="M20 12h2" />
                <path d="m6.34 17.66-1.41 1.41" /><path d="m19.07 4.93-1.41 1.41" />
              </svg>
            </span>
            <span class="account-menu-label">{{ t('header.toggleTheme') }}</span>
            <span class="account-menu-trailing"><RightOutlined /></span>
          </div>

          <button
            type="button"
            class="account-menu-item"
            role="menuitem"
            @mouseenter="closeSubmenu"
            @click="navigate('/messages')"
          >
            <span class="account-menu-icon" aria-hidden="true"><BellOutlined /></span>
            <span class="account-menu-label">{{ t('messages.title') }}</span>
            <span v-if="unreadCount > 0" class="account-menu-trailing account-menu-badge">{{ unreadLabel }}</span>
            <span v-else class="account-menu-trailing account-menu-trailing--empty" />
          </button>

          <div
            v-if="showRoleSwitcher"
            ref="roleRowRef"
            class="account-menu-item account-menu-item--submenu"
            :class="{ 'account-menu-item--active': activeSubmenu === 'role' }"
            role="menuitem"
            @mouseenter="openSubmenu('role')"
          >
            <span class="account-menu-icon" aria-hidden="true">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
                <circle cx="9" cy="7" r="4" />
                <path d="M22 21v-2a4 4 0 0 0-3-3.87" />
                <path d="M16 3.13a4 4 0 0 1 0 7.75" />
              </svg>
            </span>
            <span class="account-menu-label">{{ t('header.switchRole') }}</span>
            <span class="account-menu-trailing"><RightOutlined /></span>
          </div>

          <button
            type="button"
            class="account-menu-item"
            role="menuitem"
            @mouseenter="closeSubmenu"
            @click="navigate('/settings')"
          >
            <span class="account-menu-icon" aria-hidden="true"><SettingOutlined /></span>
            <span class="account-menu-label">{{ t('sidebar.personalSettings') }}</span>
            <span class="account-menu-trailing account-menu-trailing--empty" />
          </button>

          <div class="account-menu-divider" />

          <button
            type="button"
            class="account-menu-item account-menu-item--logout"
            role="menuitem"
            @mouseenter="closeSubmenu"
            @click="handleLogout"
          >
            <span class="account-menu-icon" aria-hidden="true"><LogoutOutlined /></span>
            <span class="account-menu-label">{{ t('sidebar.logout') }}</span>
            <span class="account-menu-trailing account-menu-trailing--empty" />
          </button>
        </div>

        <!--子菜单占位区：横向展开预留空间，避免遮挡/截断-->
        <div
          class="account-submenu-slot"
          :class="{
            'account-submenu-slot--open': activeSubmenu === 'theme',
            'account-submenu-slot--wide': activeSubmenu === 'role',
          }"
        >
          <div
            v-if="activeSubmenu === 'theme'"
            class="account-submenu"
            :style="{ top: `${submenuTop}px` }"
            role="menu"
            @mouseenter="openSubmenu('theme')"
          >
          <button
            v-for="option in themeOptions"
            :key="option.mode"
            type="button"
            class="account-submenu-item"
            role="menuitem"
            @click="pickTheme(option.mode)"
          >
            <span class="account-menu-icon" aria-hidden="true">
              <svg v-if="option.mode === 'light'" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <circle cx="12" cy="12" r="4" /><path d="M12 2v2" /><path d="M12 20v2" />
                <path d="m4.93 4.93 1.41 1.41" /><path d="m17.66 17.66 1.41 1.41" />
                <path d="M2 12h2" /><path d="M20 12h2" />
                <path d="m6.34 17.66-1.41 1.41" /><path d="m19.07 4.93-1.41 1.41" />
              </svg>
              <svg v-else-if="option.mode === 'dark'" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z" />
              </svg>
              <svg v-else xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M9.937 15.5A2 2 0 0 0 8.5 14.063l-6.135-1.582a.5.5 0 0 1 0-.962L8.5 9.936A2 2 0 0 0 9.937 8.5l1.582-6.135a.5.5 0 0 1 .963 0L14.063 8.5A2 2 0 0 0 15.5 9.937l6.135 1.581a.5.5 0 0 1 0 .964L15.5 14.063a2 2 0 0 0-1.437 1.437l-1.582 6.135a.5.5 0 0 1-.963 0z" />
                <path d="M20 3v4" /><path d="M22 5h-4" /><path d="M4 17v2" /><path d="M5 18H3" />
              </svg>
            </span>
            <span class="account-menu-label">{{ t(option.labelKey) }}</span>
            <span class="account-menu-trailing">
              <CheckOutlined v-if="mode === option.mode" class="account-submenu-check" />
            </span>
          </button>
          </div>

          <div
            v-if="activeSubmenu === 'role' && showRoleSwitcher"
            class="account-submenu account-submenu--wide"
            :style="{ top: `${submenuTop}px` }"
            role="menu"
            @mouseenter="openSubmenu('role')"
          >
          <template v-if="businessRoles.length">
            <div class="account-submenu-group">{{ t('login.portal.business') }}</div>
            <button
              v-for="role in businessRoles"
              :key="role.id"
              type="button"
              class="account-submenu-item"
              role="menuitem"
              @click="handleSwitchRole(role)"
            >
              <span class="account-menu-icon" aria-hidden="true">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <line x1="18" y1="20" x2="18" y2="10" /><line x1="12" y1="20" x2="12" y2="4" /><line x1="6" y1="20" x2="6" y2="14" />
                </svg>
              </span>
              <span class="account-menu-label">{{ role.tenant_name }}</span>
              <span class="account-menu-trailing">
                <CheckOutlined v-if="role.id === activeRoleId" class="account-submenu-check" />
              </span>
            </button>
          </template>

          <template v-if="tenantAdminRoles.length">
            <div class="account-submenu-group">{{ t('login.portal.tenantAdmin') }}</div>
            <button
              v-for="role in tenantAdminRoles"
              :key="role.id"
              type="button"
              class="account-submenu-item"
              role="menuitem"
              @click="handleSwitchRole(role)"
            >
              <span class="account-menu-icon" aria-hidden="true">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <line x1="4" y1="21" x2="4" y2="14" /><line x1="4" y1="10" x2="4" y2="3" />
                  <line x1="12" y1="21" x2="12" y2="12" /><line x1="12" y1="8" x2="12" y2="3" />
                  <line x1="20" y1="21" x2="20" y2="16" /><line x1="20" y1="12" x2="20" y2="3" />
                </svg>
              </span>
              <span class="account-menu-label">{{ role.tenant_name }}</span>
              <span class="account-menu-trailing">
                <CheckOutlined v-if="role.id === activeRoleId" class="account-submenu-check" />
              </span>
            </button>
          </template>

          <template v-if="systemRoles.length">
            <div class="account-submenu-group">{{ t('login.portal.systemAdmin') }}</div>
            <button
              v-for="role in systemRoles"
              :key="role.id"
              type="button"
              class="account-submenu-item"
              role="menuitem"
              @click="handleSwitchRole(role)"
            >
              <span class="account-menu-icon" aria-hidden="true">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
                </svg>
              </span>
              <span class="account-menu-label">{{ role.label }}</span>
              <span class="account-menu-trailing">
                <CheckOutlined v-if="role.id === activeRoleId" class="account-submenu-check" />
              </span>
            </button>
          </template>
          </div>
        </div>
      </div>
    </div>

    <button
      ref="triggerRef"
      type="button"
      class="sidebar-user-profile"
      :class="{ 'sidebar-user-profile--collapsed': collapsed && !mobileMenuOpen }"
      :aria-expanded="menuOpen"
      aria-haspopup="menu"
      @click.stop="toggleMenu"
    >
      <span class="sidebar-avatar-wrap">
        <a-avatar :size="36" class="sidebar-avatar">
          <template #icon><UserOutlined /></template>
        </a-avatar>
        <span v-if="unreadCount > 0" class="avatar-unread-badge">{{ unreadLabel }}</span>
      </span>
      <div v-if="!collapsed || mobileMenuOpen" class="sidebar-user-info">
        <div class="sidebar-user-name">{{ displayName }}</div>
      </div>
    </button>
  </div>
</template>

<style scoped>
.sidebar-user-menu {
  position: relative;
}

.sidebar-user-profile {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 8px 12px;
  border: none;
  border-radius: 10px;
  background: transparent;
  cursor: pointer;
  transition: background var(--transition-fast);
}
.sidebar-user-profile:hover,
.sidebar-user-menu--open .sidebar-user-profile {
  background: var(--color-bg-sidebar-hover);
}
.sidebar-user-profile--collapsed {
  justify-content: center;
  padding: 8px;
}

.sidebar-avatar-wrap {
  position: relative;
  flex-shrink: 0;
  display: inline-flex;
}
.sidebar-avatar {
  background: linear-gradient(135deg, var(--color-primary), var(--color-primary-light)) !important;
}
.avatar-unread-badge {
  position: absolute;
  top: -4px;
  right: -4px;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 999px;
  background: var(--color-danger);
  color: #fff;
  font-size: 10px;
  font-weight: 700;
  line-height: 18px;
  text-align: center;
  border: 2px solid var(--color-bg-sidebar);
  box-sizing: border-box;
  pointer-events: none;
}

.sidebar-user-info { min-width: 0; flex: 1; text-align: left; }
.sidebar-user-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.account-menu-shell {
  position: absolute;
  left: 0;
  bottom: calc(100% + 8px);
  z-index: 120;
  width: max-content;
}

.account-menu-panel {
  display: flex;
  align-items: flex-start;
}

.account-menu,
.account-submenu {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  border-radius: 14px;
  box-shadow: 0 4px 24px rgba(15, 23, 42, 0.12);
  padding: 6px;
}

.account-menu {
  width: 220px;
  flex-shrink: 0;
}

.account-submenu-slot {
  width: 0;
  height: 0;
  overflow: visible;
  position: relative;
  flex-shrink: 0;
}
.account-submenu-slot--open {
  width: 188px;
}
.account-submenu-slot--wide {
  width: 236px;
}

.account-submenu {
  position: absolute;
  left: -2px;
  width: 100%;
  box-sizing: border-box;
}
.account-submenu--wide {
  max-height: 320px;
  overflow-y: auto;
}

.account-menu-item,
.account-submenu-item {
  display: grid;
  grid-template-columns: 20px minmax(0, 1fr) 20px;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 9px 12px;
  border: none;
  border-radius: 10px;
  background: transparent;
  color: var(--color-text-primary);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: background var(--transition-fast), color var(--transition-fast);
  text-align: left;
}
.account-menu-item:hover,
.account-submenu-item:hover {
  background: var(--color-bg-hover);
}
.account-menu-item--active {
  background: var(--color-bg-hover);
}
.account-menu-item--submenu {
  cursor: default;
}

.account-menu-icon {
  width: 20px;
  height: 20px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-secondary);
  font-size: 16px;
}
.account-menu-icon :deep(svg),
.account-menu-icon svg {
  width: 18px;
  height: 18px;
  display: block;
}
.account-menu-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.account-submenu .account-menu-label {
  overflow: visible;
  text-overflow: clip;
}
.account-menu-trailing {
  width: 20px;
  height: 20px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-tertiary);
  font-size: 11px;
}
.account-menu-trailing--empty {
  visibility: hidden;
}
.account-menu-badge {
  width: auto;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: 999px;
  background: var(--color-primary);
  color: #fff;
  font-size: 11px;
  font-weight: 700;
}
.account-menu-divider {
  height: 1px;
  background: var(--color-border-light);
  margin: 6px 4px;
}

.account-menu-item--logout {
  color: var(--color-danger);
}
.account-menu-item--logout .account-menu-icon {
  color: var(--color-danger);
}
.account-menu-item--logout:hover {
  background: var(--color-danger-bg);
  color: var(--color-danger);
}

.account-submenu-check {
  color: var(--color-text-primary);
  font-size: 13px;
}
.account-submenu-group {
  padding: 8px 12px 4px;
  font-size: 11px;
  font-weight: 600;
  color: var(--color-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
</style>
