<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  LogoutOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  BellOutlined,
  UserOutlined,
  SafetyCertificateOutlined,
  ControlOutlined,
  TeamOutlined,
  MonitorOutlined,
  SettingOutlined,
  ArrowLeftOutlined,
} from '@ant-design/icons-vue'

const route = useRoute()
const collapsed = ref(false)
const mobileMenuOpen = ref(false)
const isMobile = ref(false)

const { isDark, toggle: toggleTheme, restore: restoreTheme } = useTheme()
const { logout, currentUser } = useAuth()

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

const systemMenuItems = [
  { key: '/admin/system', icon: MonitorOutlined, label: '全局监控' },
  { key: '/admin/system/tenants', icon: TeamOutlined, label: '租户管理' },
  { key: '/admin/system/settings', icon: SettingOutlined, label: '系统设置' },
]

// Match menu item as active: exact match or prefix match for nested routes
const isMenuActive = (itemKey: string) => {
  const path = route.path
  if (itemKey === '/admin/system') {
    // 全局监控 only matches exact /admin/system
    return path === '/admin/system'
  }
  return path.startsWith(itemKey)
}

const displayName = computed(() => currentUser.value?.display_name || '系统管理员')

const handleMenuClick = (path: string) => {
  navigateTo(path)
  if (isMobile.value) mobileMenuOpen.value = false
}

watch(route, () => {
  if (isMobile.value) mobileMenuOpen.value = false
})
</script>

<template>
  <div class="admin-layout" :class="{ 'admin-layout--collapsed': collapsed }">
    <aside
      class="admin-sidebar"
      :class="{
        'admin-sidebar--collapsed': collapsed,
        'admin-sidebar--mobile-open': mobileMenuOpen,
      }"
    >
      <div class="sidebar-logo" @click="navigateTo('/admin/system')">
        <div class="sidebar-logo-icon">
          <ControlOutlined />
        </div>
        <transition name="fade">
          <span v-if="!collapsed" class="sidebar-logo-text">系统管理</span>
        </transition>
      </div>

      <nav class="sidebar-nav">
        <div class="sidebar-section">
          <div v-if="!collapsed" class="sidebar-section-title">系统功能</div>
          <div
            v-for="item in systemMenuItems"
            :key="item.key"
            class="sidebar-item"
            :class="{ 'sidebar-item--active': isMenuActive(item.key) }"
            @click="handleMenuClick(item.key)"
          >
            <component :is="item.icon" class="sidebar-item-icon" />
            <transition name="fade">
              <span v-if="!collapsed" class="sidebar-item-label">{{ item.label }}</span>
            </transition>
            <div v-if="isMenuActive(item.key)" class="sidebar-item-indicator" />
          </div>
        </div>
      </nav>

      <div class="sidebar-footer">
        <div class="sidebar-item" @click="navigateTo('/dashboard')">
          <ArrowLeftOutlined class="sidebar-item-icon" />
          <transition name="fade">
            <span v-if="!collapsed" class="sidebar-item-label">返回前台</span>
          </transition>
        </div>
        <div class="sidebar-item sidebar-item--logout" @click="logout">
          <LogoutOutlined class="sidebar-item-icon" />
          <transition name="fade">
            <span v-if="!collapsed" class="sidebar-item-label">退出登录</span>
          </transition>
        </div>
      </div>
    </aside>

    <div
      v-if="mobileMenuOpen && isMobile"
      class="sidebar-overlay"
      @click="mobileMenuOpen = false"
    />

    <div class="admin-main">
      <header class="admin-header">
        <div class="admin-header-left">
          <button
            class="header-toggle"
            @click="isMobile ? (mobileMenuOpen = !mobileMenuOpen) : (collapsed = !collapsed)"
          >
            <MenuUnfoldOutlined v-if="collapsed && !isMobile" />
            <MenuFoldOutlined v-else-if="!isMobile" />
            <MenuUnfoldOutlined v-else />
          </button>
          <div class="header-breadcrumb">
            <ControlOutlined style="color: var(--color-danger); font-size: 14px;" />
            <span class="breadcrumb-sep">/</span>
            <span>系统管理</span>
          </div>
        </div>

        <div class="admin-header-right">
          <button class="header-action" @click="toggleTheme" :title="isDark ? '切换亮色' : '切换暗色'">
            <span v-if="isDark" style="font-size: 18px;">🌙</span>
            <span v-else style="font-size: 18px;">☀️</span>
          </button>
          <a-badge :count="0" :offset="[-4, 4]">
            <button class="header-action"><BellOutlined /></button>
          </a-badge>
          <a-dropdown>
            <div class="header-user">
              <a-avatar :size="32" class="header-avatar">
                <template #icon><UserOutlined /></template>
              </a-avatar>
              <span class="header-username">{{ displayName }}</span>
            </div>
            <template #overlay>
              <a-menu>
                <a-menu-item key="profile" @click="navigateTo('/settings')">个人设置</a-menu-item>
                <a-menu-divider />
                <a-menu-item key="logout" @click="logout">退出登录</a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
        </div>
      </header>

      <main class="admin-content"><slot /></main>
    </div>
  </div>
</template>

<style scoped>
.admin-layout { display: flex; min-height: 100vh; background: var(--color-bg-page); }

.admin-sidebar {
  width: var(--sidebar-width); background: var(--color-bg-sidebar);
  border-right: 1px solid var(--color-sidebar-border);
  display: flex; flex-direction: column; position: fixed;
  top: 0; left: 0; bottom: 0; z-index: 100;
  transition: width var(--transition-slow); overflow: hidden;
}
.admin-sidebar--collapsed { width: var(--sidebar-collapsed-width); }

.sidebar-logo {
  height: var(--header-height); display: flex; align-items: center;
  padding: 0 20px; gap: 12px; cursor: pointer; flex-shrink: 0;
  border-bottom: 1px solid var(--color-sidebar-border);
}
.sidebar-logo-icon {
  width: 36px; height: 36px; background: linear-gradient(135deg, #ef4444, #dc2626);
  border-radius: 10px; display: flex; align-items: center; justify-content: center;
  color: #fff; font-size: 18px; flex-shrink: 0;
}
.sidebar-logo-text {
  font-size: 18px; font-weight: 700; color: var(--color-sidebar-logo-text);
  white-space: nowrap; letter-spacing: -0.02em;
}

.sidebar-nav { flex: 1; padding: 12px 0; overflow-y: auto; overflow-x: hidden; }
.sidebar-section { margin-bottom: 8px; }
.sidebar-section-title {
  padding: 8px 24px 6px; font-size: 11px; font-weight: 600;
  color: var(--color-sidebar-section-title); text-transform: uppercase;
  letter-spacing: 0.08em; white-space: nowrap;
}

.sidebar-item {
  display: flex; align-items: center; padding: 0 16px; height: 44px;
  margin: 2px 8px; border-radius: 10px; cursor: pointer;
  transition: all var(--transition-fast); position: relative;
  gap: 12px; color: var(--color-text-sidebar);
}
.sidebar-item:hover { background: var(--color-bg-sidebar-hover); color: var(--color-text-primary); }
.sidebar-item--active { background: var(--color-bg-sidebar-active); color: var(--color-text-sidebar-active); }
.sidebar-item--active .sidebar-item-icon { color: var(--color-primary); }
.sidebar-item-icon { font-size: 18px; flex-shrink: 0; width: 20px; display: flex; align-items: center; justify-content: center; }
.sidebar-item-label { font-size: 14px; font-weight: 500; white-space: nowrap; }
.sidebar-item-indicator { position: absolute; right: 0; top: 50%; transform: translateY(-50%); width: 3px; height: 20px; background: var(--color-primary); border-radius: 3px 0 0 3px; }
.sidebar-item--logout { color: var(--color-text-tertiary); }
.sidebar-item--logout:hover { color: #ef4444; background: rgba(239, 68, 68, 0.08); }

.sidebar-footer { padding: 8px 0 16px; border-top: 1px solid var(--color-sidebar-border); }
.sidebar-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); z-index: 99; backdrop-filter: blur(4px); }

.admin-main { flex: 1; margin-left: var(--sidebar-width); transition: margin-left var(--transition-slow); display: flex; flex-direction: column; min-height: 100vh; }
.admin-layout--collapsed .admin-main { margin-left: var(--sidebar-collapsed-width); }

.admin-header {
  height: var(--header-height); border-bottom: 1px solid var(--color-border-light);
  display: flex; align-items: center; justify-content: space-between;
  padding: 0 24px; position: sticky; top: 0; z-index: 50;
  backdrop-filter: blur(12px); background: color-mix(in srgb, var(--color-bg-card) 85%, transparent);
}
.admin-header-left { display: flex; align-items: center; gap: 16px; }
.admin-header-right { display: flex; align-items: center; gap: 8px; }

.header-toggle {
  width: 36px; height: 36px; border: none; background: transparent;
  border-radius: var(--radius-md); cursor: pointer; display: flex;
  align-items: center; justify-content: center; font-size: 18px;
  color: var(--color-text-secondary); transition: all var(--transition-fast);
}
.header-toggle:hover { background: var(--color-bg-hover); color: var(--color-text-primary); }

.header-breadcrumb { display: flex; align-items: center; gap: 8px; font-size: 13px; color: var(--color-text-secondary); font-weight: 500; }
.breadcrumb-sep { color: var(--color-text-tertiary); }

.header-action {
  width: 36px; height: 36px; border: none; background: transparent;
  border-radius: var(--radius-md); cursor: pointer; display: flex;
  align-items: center; justify-content: center; font-size: 18px;
  color: var(--color-text-secondary); transition: all var(--transition-fast);
}
.header-action:hover { background: var(--color-bg-hover); color: var(--color-text-primary); }

.header-user {
  display: flex; align-items: center; gap: 10px;
  padding: 4px 12px 4px 4px; border-radius: var(--radius-full);
  cursor: pointer; transition: background var(--transition-fast); margin-left: 4px;
}
.header-user:hover { background: var(--color-bg-hover); }
.header-avatar { background: linear-gradient(135deg, #ef4444, #dc2626) !important; }
.header-username { font-size: 14px; font-weight: 500; color: var(--color-text-primary); }

.admin-content { flex: 1; padding: var(--space-page); max-width: 1400px; width: 100%; margin: 0 auto; }

.fade-enter-active, .fade-leave-active { transition: opacity 0.15s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }

@media (max-width: 768px) {
  .admin-sidebar { transform: translateX(-100%); width: var(--sidebar-width); }
  .admin-sidebar--mobile-open { transform: translateX(0); }
  .admin-sidebar--collapsed { width: var(--sidebar-width); }
  .admin-main { margin-left: 0 !important; }
  .admin-header { padding: 0 16px; }
  .admin-content { padding: 16px; }
  .header-username { display: none; }
  .header-breadcrumb { display: none; }
}
</style>
