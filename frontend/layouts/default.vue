<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  DashboardOutlined,
  ClockCircleOutlined,
  FolderOpenOutlined,
  SettingOutlined,
  MonitorOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  BellOutlined,
  UserOutlined,
  SafetyCertificateOutlined,
  AppstoreOutlined,
  TeamOutlined,
} from '@ant-design/icons-vue'

const route = useRoute()
const collapsed = ref(false)
const mobileMenuOpen = ref(false)
const isMobile = ref(false)

const { isDark, toggle: toggleTheme, restore: restoreTheme } = useTheme()
const { logout } = useAuth()

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

// Front-end business menu
const businessMenuItems = [
  { key: '/dashboard', icon: DashboardOutlined, label: '审核工作台' },
  { key: '/cron', icon: ClockCircleOutlined, label: '定时任务' },
  { key: '/archive', icon: FolderOpenOutlined, label: '归档复盘' },
]

// Admin menu
const adminMenuItems = [
  { key: '/admin/tenant', icon: AppstoreOutlined, label: '租户配置' },
  { key: '/admin/system', icon: TeamOutlined, label: '系统管理' },
  { key: '/admin/monitor', icon: MonitorOutlined, label: '全局监控' },
]

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
          <SafetyCertificateOutlined />
        </div>
        <transition name="fade">
          <span v-if="!collapsed" class="sidebar-logo-text">OA智审</span>
        </transition>
      </div>

      <!-- Navigation -->
      <nav class="sidebar-nav">
        <div class="sidebar-section">
          <div v-if="!collapsed" class="sidebar-section-title">业务功能</div>
          <div
            v-for="item in businessMenuItems"
            :key="item.key"
            class="sidebar-item"
            :class="{ 'sidebar-item--active': selectedKeys.includes(item.key) }"
            @click="handleMenuClick(item.key)"
          >
            <component :is="item.icon" class="sidebar-item-icon" />
            <transition name="fade">
              <span v-if="!collapsed" class="sidebar-item-label">{{ item.label }}</span>
            </transition>
            <div v-if="selectedKeys.includes(item.key)" class="sidebar-item-indicator" />
          </div>
        </div>

        <div class="sidebar-section">
          <div v-if="!collapsed" class="sidebar-section-title">系统管理</div>
          <div
            v-for="item in adminMenuItems"
            :key="item.key"
            class="sidebar-item"
            :class="{ 'sidebar-item--active': selectedKeys.includes(item.key) }"
            @click="handleMenuClick(item.key)"
          >
            <component :is="item.icon" class="sidebar-item-icon" />
            <transition name="fade">
              <span v-if="!collapsed" class="sidebar-item-label">{{ item.label }}</span>
            </transition>
            <div v-if="selectedKeys.includes(item.key)" class="sidebar-item-indicator" />
          </div>
        </div>
      </nav>

      <!-- Sidebar footer -->
      <div class="sidebar-footer">
        <div class="sidebar-item sidebar-item--logout" @click="logout">
          <LogoutOutlined class="sidebar-item-icon" />
          <transition name="fade">
            <span v-if="!collapsed" class="sidebar-item-label">退出登录</span>
          </transition>
        </div>
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
          <!-- Theme toggle -->
          <button class="header-action" @click="toggleTheme" :title="isDark ? '切换亮色' : '切换暗色'">
            <span v-if="isDark" style="font-size: 18px;">🌙</span>
            <span v-else style="font-size: 18px;">☀️</span>
          </button>

          <!-- Notifications -->
          <a-badge :count="3" :offset="[-4, 4]">
            <button class="header-action">
              <BellOutlined />
            </button>
          </a-badge>

          <!-- User avatar -->
          <a-dropdown>
            <div class="header-user">
              <a-avatar :size="32" class="header-avatar">
                <template #icon><UserOutlined /></template>
              </a-avatar>
              <span class="header-username">管理员</span>
            </div>
            <template #overlay>
              <a-menu>
                <a-menu-item key="profile">个人设置</a-menu-item>
                <a-menu-divider />
                <a-menu-item key="logout" @click="logout">退出登录</a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
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
  background: var(--color-bg-page, #f8fafc);
}

/* Ensure CSS variables have fallbacks */
:deep(*) {
  --sidebar-width: 260px;
  --sidebar-collapsed-width: 72px;
  --header-height: 64px;
  --transition-fast: 150ms cubic-bezier(0.4, 0, 0.2, 1);
  --transition-slow: 300ms cubic-bezier(0.4, 0, 0.2, 1);
  --radius-md: 8px;
  --radius-full: 9999px;
  --color-bg-page: #f8fafc;
  --color-bg-card: #ffffff;
  --color-bg-hover: #f1f5f9;
  --color-bg-sidebar: #0f172a;
  --color-bg-sidebar-hover: #1e293b;
  --color-bg-sidebar-active: rgba(79, 70, 229, 0.15);
  --color-text-primary: #0f172a;
  --color-text-secondary: #475569;
  --color-text-tertiary: #94a3b8;
  --color-text-sidebar: #94a3b8;
  --color-text-sidebar-active: #ffffff;
  --color-border-light: #f1f5f9;
  --color-primary-lighter: #818cf8;
  --space-page: 24px;
}

/* ===== Sidebar ===== */
.sidebar {
  width: var(--sidebar-width);
  background: var(--color-bg-sidebar);
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
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
}

.sidebar-logo-icon {
  width: 36px;
  height: 36px;
  background: linear-gradient(135deg, #4f46e5, #7c3aed);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 18px;
  flex-shrink: 0;
}

.sidebar-logo-text {
  font-size: 18px;
  font-weight: 700;
  color: #f1f5f9;
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
  color: #475569;
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
  color: #e2e8f0;
}

.sidebar-item--active {
  background: var(--color-bg-sidebar-active);
  color: var(--color-text-sidebar-active);
}

.sidebar-item--active .sidebar-item-icon {
  color: var(--color-primary-lighter);
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
}

.sidebar-item-indicator {
  position: absolute;
  right: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 20px;
  background: var(--color-primary-lighter);
  border-radius: 3px 0 0 3px;
}

.sidebar-item--logout {
  color: #64748b;
}

.sidebar-item--logout:hover {
  color: #ef4444;
  background: rgba(239, 68, 68, 0.1);
}

/* Sidebar footer */
.sidebar-footer {
  padding: 8px 0 16px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
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
  background: var(--color-bg-card);
  border-bottom: 1px solid var(--color-border-light);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  position: sticky;
  top: 0;
  z-index: 50;
  backdrop-filter: blur(12px);
  background: rgba(255, 255, 255, 0.85);
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

.header-toggle {
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
}

.header-toggle:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}

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
}

.header-action:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
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
