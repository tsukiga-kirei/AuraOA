<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  DashboardOutlined,
  ClockCircleOutlined,
  FolderOpenOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  BellOutlined,
  UserOutlined,
  AppstoreOutlined,
  ApartmentOutlined,
  DatabaseOutlined,
  MonitorOutlined,
  TeamOutlined,
  UpOutlined,
} from '@ant-design/icons-vue'

const route = useRoute()
const collapsed = ref(false)
const mobileMenuOpen = ref(false)
const isMobile = ref(false)

const { isDark, toggle: toggleTheme, restore: restoreTheme } = useTheme()
const { logout, userRole, currentUser } = useAuth()

onMounted(() => {
  restoreTheme()
  checkMobile()
  window.addEventListener('resize', checkMobile)
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
})

const checkMobile = () => {
  isMobile.value = window.innerWidth < 768
  if (isMobile.value) collapsed.value = true
}

const selectedKeys = computed(() => [route.path])

const businessMenuItems = [
  { key: '/dashboard', icon: DashboardOutlined, label: '审核工作台', badge: 6 },
  { key: '/cron', icon: ClockCircleOutlined, label: '定时任务', badge: 0 },
  { key: '/archive', icon: FolderOpenOutlined, label: '归档复盘', badge: 0 },
]

// Only show tenant admin entry if user is tenant_admin or system_admin
const showTenantAdmin = computed(() =>
  userRole.value === 'tenant_admin' || userRole.value === 'system_admin'
)

// Only show system admin entry if user is system_admin
const showSystemAdmin = computed(() => userRole.value === 'system_admin')

const displayName = computed(() => currentUser.value?.display_name || '用户')

const handleMenuClick = (path: string) => {
  navigateTo(path)
  if (isMobile.value) mobileMenuOpen.value = false
}

watch(route, () => {
  if (isMobile.value) mobileMenuOpen.value = false
})
</script>

<template>
  <div class="app-layout" :class="{ 'app-layout--collapsed': collapsed }">
    <!-- Sidebar -->
    <aside
      class="sidebar"
      :class="{
        'sidebar--collapsed': collapsed,
        'sidebar--mobile-open': mobileMenuOpen,
      }"
    >
      <!-- Logo -->
      <div class="sidebar-logo" @click="navigateTo('/dashboard')">
        <div class="sidebar-logo-icon">
          <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path><path d="M9 12l2 2 4-4"></path></svg>
        </div>
        <transition name="fade">
          <span v-if="!collapsed" class="sidebar-logo-text">OA智审</span>
        </transition>
      </div>

      <!-- Navigation -->
      <nav class="sidebar-nav">
        <div class="sidebar-section">
          <div v-if="!collapsed" class="sidebar-section-title">业务功能</div>
          <a-tooltip
            v-for="item in businessMenuItems"
            :key="item.key"
            :title="collapsed ? item.label : ''"
            placement="right"
            :mouse-enter-delay="0.1"
          >
            <div
              class="sidebar-item"
              :class="{ 'sidebar-item--active': selectedKeys.includes(item.key) }"
              @click="handleMenuClick(item.key)"
            >
              <component :is="item.icon" class="sidebar-item-icon" />
              <transition name="fade">
                <span v-if="!collapsed" class="sidebar-item-label">{{ item.label }}</span>
              </transition>
              <transition name="fade">
                <span v-if="!collapsed && item.badge" class="sidebar-item-badge">{{ item.badge }}</span>
              </transition>
              <div v-if="selectedKeys.includes(item.key)" class="sidebar-item-indicator" />
            </div>
          </a-tooltip>
        </div>
      </nav>

      <!-- Sidebar footer with User Profile -->
      <div class="sidebar-footer">
        <a-popover
          placement="rightBottom"
          trigger="click"
          overlayClassName="user-profile-popover"
          :arrow="false"
        >
          <template #content>
            <div class="user-dropdown-panel">
              <!-- Tenant Admin Section -->
              <template v-if="showTenantAdmin">
                <div class="dropdown-section-title">租户管理</div>
                <div class="dropdown-item" @click="navigateTo('/admin/tenant')">
                  <AppstoreOutlined class="dropdown-item-icon" />
                  <span>规则配置</span>
                </div>
                <div class="dropdown-item" @click="navigateTo('/admin/tenant/org')">
                  <ApartmentOutlined class="dropdown-item-icon" />
                  <span>组织人员</span>
                </div>
                <div class="dropdown-item" @click="navigateTo('/admin/tenant/data')">
                  <DatabaseOutlined class="dropdown-item-icon" />
                  <span>数据信息</span>
                </div>
                <div class="dropdown-divider" />
              </template>

              <!-- System Admin Section -->
              <template v-if="showSystemAdmin">
                <div class="dropdown-section-title">系统管理</div>
                <div class="dropdown-item" @click="navigateTo('/admin/system')">
                  <MonitorOutlined class="dropdown-item-icon" />
                  <span>全局监控</span>
                </div>
                <div class="dropdown-item" @click="navigateTo('/admin/system/tenants')">
                  <TeamOutlined class="dropdown-item-icon" />
                  <span>租户管理</span>
                </div>
                <div class="dropdown-item" @click="navigateTo('/admin/system/settings')">
                  <SettingOutlined class="dropdown-item-icon" />
                  <span>系统设置</span>
                </div>
                <div class="dropdown-divider" />
              </template>

              <!-- Common Actions -->
              <div class="dropdown-item" @click="navigateTo('/settings')">
                <UserOutlined class="dropdown-item-icon" />
                <span>个人设置</span>
              </div>
              <div class="dropdown-item dropdown-item--danger" @click="logout">
                <LogoutOutlined class="dropdown-item-icon" />
                <span>退出登录</span>
              </div>
            </div>
          </template>
          
          <div class="sidebar-user-profile" :class="{ 'sidebar-user-profile--collapsed': collapsed }">
            <a-avatar :size="36" class="sidebar-avatar">
              <template #icon><UserOutlined /></template>
            </a-avatar>
            <div v-if="!collapsed" class="sidebar-user-info">
              <div class="sidebar-user-name">{{ displayName }}</div>
              <div class="sidebar-user-role">
                {{ userRole === 'system_admin' ? '系统管理员' : userRole === 'tenant_admin' ? '租户管理员' : '普通用户' }}
              </div>
            </div>
          </div>
        </a-popover>
      </div>
    </aside>

    <!-- Mobile overlay -->
    <div
      v-if="mobileMenuOpen && isMobile"
      class="sidebar-overlay"
      @click="mobileMenuOpen = false"
    />

    <!-- Main content -->
    <div class="main-wrapper">
      <!-- Header -->
      <header class="app-header">
        <div class="app-header-left">
          <button
            class="header-toggle"
            @click="isMobile ? (mobileMenuOpen = !mobileMenuOpen) : (collapsed = !collapsed)"
          >
            <MenuUnfoldOutlined v-if="collapsed && !isMobile" />
            <MenuFoldOutlined v-else-if="!isMobile" />
            <MenuUnfoldOutlined v-else />
          </button>
        </div>

        <div class="app-header-right">
          <a-tooltip title="切换主题" placement="bottom" :mouse-enter-delay="0.5">
            <button class="header-action" @click="toggleTheme">
              <transition name="rotate-icon" mode="out-in">
                <svg v-if="isDark" key="moon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path></svg>
                <svg v-else key="sun" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="5"></circle><line x1="12" y1="1" x2="12" y2="3"></line><line x1="12" y1="21" x2="12" y2="23"></line><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line><line x1="1" y1="12" x2="3" y2="12"></line><line x1="21" y1="12" x2="23" y2="12"></line><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line></svg>
              </transition>
            </button>
          </a-tooltip>

          <a-tooltip title="消息通知" placement="bottom" :mouse-enter-delay="0.5">
            <a-badge :count="3" :offset="[-4, 4]">
              <button class="header-action">
                <BellOutlined />
              </button>
            </a-badge>
          </a-tooltip>
        </div>
      </header>

      <!-- Page content -->
      <main class="app-content">
        <slot />
      </main>
    </div>
  </div>
</template>

<style scoped>
.app-layout {
  display: flex;
  min-height: 100vh;
  background: var(--color-bg-page);
}

/* ===== Sidebar - Light-friendly ===== */
.sidebar {
  width: var(--sidebar-width);
  background: var(--color-bg-sidebar);
  border-right: 1px solid var(--color-sidebar-border);
  display: flex;
  flex-direction: column;
  position: fixed;
  top: 0;
  left: 0;
  bottom: 0;
  z-index: 100;
  transition: width var(--transition-slow);
  overflow: hidden;
}

.sidebar--collapsed {
  width: var(--sidebar-collapsed-width);
}

/* Logo */
.sidebar-logo {
  height: var(--header-height);
  display: flex;
  align-items: center;
  padding: 0 20px;
  gap: 12px;
  cursor: pointer;
  flex-shrink: 0;
  border-bottom: 1px solid var(--color-sidebar-border);
}

.sidebar-logo-icon {
  width: 36px;
  height: 36px;
  background: var(--color-bg-hover);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-primary);
  font-size: 18px;
  flex-shrink: 0;
}

.sidebar-logo-text {
  font-size: 18px;
  font-weight: 700;
  color: var(--color-sidebar-logo-text);
  white-space: nowrap;
  letter-spacing: -0.02em;
}

/* Navigation */
.sidebar-nav {
  flex: 1;
  padding: 12px 0;
  overflow-y: auto;
  overflow-x: hidden;
}

.sidebar-section {
  margin-bottom: 8px;
}

.sidebar-section-title {
  padding: 8px 24px 6px;
  font-size: 11px;
  font-weight: 600;
  color: var(--color-sidebar-section-title);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  white-space: nowrap;
}

.sidebar-item {
  display: flex;
  align-items: center;
  padding: 0 16px;
  height: 44px;
  margin: 2px 8px;
  border-radius: 10px;
  cursor: pointer;
  transition: all var(--transition-fast);
  position: relative;
  gap: 12px;
  color: var(--color-text-sidebar);
}

.sidebar-item:hover {
  background: var(--color-bg-sidebar-hover);
  color: var(--color-text-primary);
}

.sidebar-item--active {
  background: var(--color-bg-sidebar-active);
  color: var(--color-text-sidebar-active);
}

.sidebar-item--active .sidebar-item-icon {
  color: var(--color-primary);
}

.sidebar-item-icon {
  font-size: 18px;
  flex-shrink: 0;
  width: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.sidebar-item-label {
  font-size: 14px;
  font-weight: 500;
  white-space: nowrap;
  flex: 1;
}

.sidebar-item-badge {
  font-size: 11px;
  font-weight: 700;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: 10px;
  background: var(--color-primary);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
}

.sidebar-item-indicator {
  position: absolute;
  right: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 20px;
  background: var(--color-primary);
  border-radius: 3px 0 0 3px;
}

/* Collapsed sidebar: more prominent active indicator */
.sidebar--collapsed .sidebar-item--active {
  background: var(--color-bg-sidebar-active);
  box-shadow: inset 3px 0 0 var(--color-primary);
}

.sidebar--collapsed .sidebar-item--active .sidebar-item-icon {
  color: var(--color-primary);
  transform: scale(1.1);
  transition: transform var(--transition-fast);
}

.sidebar-item--logout {
  color: var(--color-text-tertiary);
}

.sidebar-item--logout:hover {
  color: #ef4444;
  background: rgba(239, 68, 68, 0.08);
}



/* Sidebar overlay for mobile */
.sidebar-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 99;
  backdrop-filter: blur(4px);
}

/* ===== Main wrapper ===== */
.main-wrapper {
  flex: 1;
  margin-left: var(--sidebar-width);
  transition: margin-left var(--transition-slow);
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.app-layout--collapsed .main-wrapper {
  margin-left: var(--sidebar-collapsed-width);
}

/* ===== Header ===== */
.app-header {
  height: var(--header-height);
  border-bottom: 1px solid var(--color-border-light);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  position: sticky;
  top: 0;
  z-index: 50;
  backdrop-filter: blur(12px);
  background: color-mix(in srgb, var(--color-bg-card) 85%, transparent);
}

.app-header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.app-header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-toggle,
.header-action {
  width: 36px;
  height: 36px;
  border: none;
  background: transparent;
  border-radius: var(--radius-md);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
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

.header-user {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 4px 12px 4px 4px;
  border-radius: var(--radius-full);
  cursor: pointer;
  transition: background var(--transition-fast);
  margin-left: 4px;
}

.header-user:hover {
  background: var(--color-bg-hover);
}

.header-avatar {
  background: linear-gradient(135deg, #4f46e5, #7c3aed) !important;
}

.header-username {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-primary);
}

/* ===== User Dropdown Panel ===== */
.user-dropdown-panel {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-xl);
  padding: 8px 0;
  min-width: 220px;
  max-height: 80vh;
  overflow-y: auto;
}

.dropdown-user-info {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
}

.dropdown-user-detail {
  flex: 1;
  min-width: 0;
}

.dropdown-user-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.dropdown-user-role {
  font-size: 12px;
  color: var(--color-text-tertiary);
  margin-top: 1px;
}

.dropdown-divider {
  height: 1px;
  background: var(--color-border-light);
  margin: 4px 0;
}

.dropdown-section-title {
  font-size: 11px;
  font-weight: 600;
  color: var(--color-text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  padding: 8px 16px 4px;
}

.dropdown-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 16px;
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-primary);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.dropdown-item:hover {
  background: var(--color-bg-hover);
  color: var(--color-primary);
}

.dropdown-item-icon {
  font-size: 15px;
  color: var(--color-text-tertiary);
  width: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.dropdown-item:hover .dropdown-item-icon {
  color: var(--color-primary);
}

.dropdown-item--danger {
  color: var(--color-text-secondary);
}

.dropdown-item--danger:hover {
  color: var(--color-danger);
  background: var(--color-danger-bg);
}



/* Popup style override */
:global(.user-profile-popover .ant-popover-inner) {
  padding: 0;
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-xl);
  border: 1px solid var(--color-border);
  overflow: hidden;
}

:global(.user-profile-popover .ant-popover-inner-content) {
  padding: 0;
}

/* Rotate animation for theme icon */
.rotate-icon-enter-active,
.rotate-icon-leave-active {
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}

.rotate-icon-enter-from {
  opacity: 0;
  transform: rotate(-120deg) scale(0.5);
}

.rotate-icon-leave-to {
  opacity: 0;
  transform: rotate(120deg) scale(0.5);
}

/* User Dropdown Panel Styling (Reused) */
.user-dropdown-panel {
  background: var(--color-bg-card);
  min-width: 240px;
  max-height: 80vh;
  overflow-y: auto;
  padding: 8px 0;
}

/* ===== Content ===== */
.app-content {
  flex: 1;
  padding: var(--space-page);
  max-width: 1400px;
  width: 100%;
  margin: 0 auto;
}

/* Transitions */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* ===== Responsive ===== */
@media (max-width: 768px) {
  .sidebar {
    transform: translateX(-100%);
    width: var(--sidebar-width);
  }

  .sidebar--mobile-open {
    transform: translateX(0);
  }

  .sidebar--collapsed {
    width: var(--sidebar-width);
  }

  .main-wrapper {
    margin-left: 0 !important;
  }

  .app-header {
    padding: 0 16px;
  }

  .app-content {
    padding: 16px;
  }

  .header-username {
    display: none;
  }
}
</style>
