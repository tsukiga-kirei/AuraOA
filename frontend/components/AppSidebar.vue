<script setup lang="ts">
import {
  LogoutOutlined,
  UserOutlined,
  SettingOutlined,
} from '@ant-design/icons-vue'

defineProps<{
  collapsed: boolean
  mobileMenuOpen: boolean
}>()

const emit = defineEmits<{
  (e: 'update:mobileMenuOpen', val: boolean): void
}>()

const { currentUser, logout } = useAuth()
const { sections, isMenuActive, logoTarget } = useSidebarMenu()

const displayName = computed(() => currentUser.value?.display_name || '用户')
const roleLabel = computed(() => currentUser.value?.role_label || '用户')

const handleMenuClick = (path: string) => {
  navigateTo(path)
  emit('update:mobileMenuOpen', false)
}
</script>

<template>
  <aside
    class="sidebar"
    :class="{
      'sidebar--collapsed': collapsed,
      'sidebar--mobile-open': mobileMenuOpen,
    }"
  >
    <!-- Logo -->
    <div class="sidebar-logo" @click="navigateTo(logoTarget)">
      <div class="sidebar-logo-icon">
        <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none"
          stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path>
          <path d="M9 12l2 2 4-4"></path>
        </svg>
      </div>
      <transition name="fade">
        <span v-if="!collapsed || mobileMenuOpen" class="sidebar-logo-text">OA智审</span>
      </transition>
      <!-- Mobile close button -->
      <button
        v-if="mobileMenuOpen"
        class="sidebar-close-btn"
        @click.stop="emit('update:mobileMenuOpen', false)"
        aria-label="关闭菜单"
      >
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none"
          stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="18" y1="6" x2="6" y2="18"></line>
          <line x1="6" y1="6" x2="18" y2="18"></line>
        </svg>
      </button>
    </div>

    <!-- Navigation sections driven purely by permissions -->
    <nav class="sidebar-nav">
      <div v-for="section in sections" :key="section.id" class="sidebar-section">
        <div v-if="!collapsed || mobileMenuOpen" class="sidebar-section-title">{{ section.title }}</div>
        <a-tooltip
          v-for="item in section.items"
          :key="item.key"
          :title="collapsed ? item.label : ''"
          placement="right"
          :mouse-enter-delay="0.1"
        >
          <div
            class="sidebar-item"
            :class="{ 'sidebar-item--active': isMenuActive(item.key) }"
            @click="handleMenuClick(item.key)"
          >
            <component :is="item.icon" class="sidebar-item-icon" />
            <transition name="fade">
              <span v-if="!collapsed || mobileMenuOpen" class="sidebar-item-label">{{ item.label }}</span>
            </transition>
            <transition name="fade">
              <span v-if="!collapsed && item.badge || mobileMenuOpen && item.badge" class="sidebar-item-badge">{{ item.badge }}</span>
            </transition>
            <div v-if="isMenuActive(item.key)" class="sidebar-item-indicator" />
          </div>
        </a-tooltip>
      </div>
    </nav>

    <!-- User profile footer — only settings + logout, no duplicate nav -->
    <div class="sidebar-footer">
      <a-popover
        placement="rightBottom"
        trigger="click"
        overlayClassName="user-profile-popover"
        :arrow="false"
      >
        <template #content>
          <div class="user-dropdown-panel">
            <div class="dropdown-item" @click="handleMenuClick('/settings')">
              <SettingOutlined class="dropdown-item-icon" />
              <span>个人设置</span>
            </div>
            <div class="dropdown-divider" />
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
          <div v-if="!collapsed || mobileMenuOpen" class="sidebar-user-info">
            <div class="sidebar-user-name">{{ displayName }}</div>
            <div class="sidebar-user-role">{{ roleLabel }}</div>
          </div>
        </div>
      </a-popover>
    </div>
  </aside>
</template>

<style scoped>
.sidebar {
  width: var(--sidebar-width);
  background: var(--color-bg-sidebar);
  border-right: 1px solid var(--color-sidebar-border);
  display: flex; flex-direction: column;
  position: fixed; top: 0; left: 0; bottom: 0;
  z-index: 100;
  transition: width var(--transition-slow), transform 0.3s cubic-bezier(0.4, 0, 0.2, 1), box-shadow 0.3s ease;
  overflow: hidden;
}
.sidebar--collapsed { width: var(--sidebar-collapsed-width); }

.sidebar-logo {
  height: var(--header-height);
  display: flex; align-items: center;
  padding: 0 20px; gap: 12px;
  cursor: pointer; flex-shrink: 0;
  border-bottom: 1px solid var(--color-sidebar-border);
}
.sidebar-logo-icon {
  width: 36px; height: 36px;
  background: var(--color-bg-hover);
  border-radius: 10px;
  display: flex; align-items: center; justify-content: center;
  color: var(--color-primary); font-size: 18px; flex-shrink: 0;
}
.sidebar-logo-text {
  font-size: 18px; font-weight: 700;
  color: var(--color-sidebar-logo-text);
  white-space: nowrap; letter-spacing: -0.02em;
}

.sidebar-close-btn {
  display: none;
  width: 32px; height: 32px;
  border: none; background: var(--color-bg-hover);
  border-radius: var(--radius-md);
  cursor: pointer;
  align-items: center; justify-content: center;
  color: var(--color-text-secondary);
  margin-left: auto;
  flex-shrink: 0;
  transition: all var(--transition-fast);
}
.sidebar-close-btn:hover {
  background: var(--color-danger-bg);
  color: var(--color-danger);
}

.sidebar-nav { flex: 1; padding: 12px 0; overflow-y: auto; overflow-x: hidden; }
.sidebar-section { margin-bottom: 8px; }
.sidebar-section-title {
  padding: 8px 24px 6px; font-size: 11px; font-weight: 600;
  color: var(--color-sidebar-section-title);
  text-transform: uppercase; letter-spacing: 0.08em; white-space: nowrap;
}

.sidebar-item {
  display: flex; align-items: center;
  padding: 0 16px; height: 44px;
  margin: 2px 8px; border-radius: 10px;
  cursor: pointer; transition: all var(--transition-fast);
  position: relative; gap: 12px;
  color: var(--color-text-sidebar);
}
.sidebar-item:hover { background: var(--color-bg-sidebar-hover); color: var(--color-text-primary); }
.sidebar-item--active { background: var(--color-bg-sidebar-active); color: var(--color-text-sidebar-active); }
.sidebar-item--active .sidebar-item-icon { color: var(--color-primary); }
.sidebar-item-icon { font-size: 18px; flex-shrink: 0; width: 20px; display: flex; align-items: center; justify-content: center; }
.sidebar-item-label { font-size: 14px; font-weight: 500; white-space: nowrap; flex: 1; }
.sidebar-item-badge {
  font-size: 11px; font-weight: 700;
  min-width: 20px; height: 20px; padding: 0 6px;
  border-radius: 10px; background: var(--color-primary); color: #fff;
  display: flex; align-items: center; justify-content: center;
}
.sidebar-item-indicator {
  position: absolute; right: 0; top: 50%; transform: translateY(-50%);
  width: 3px; height: 20px; background: var(--color-primary);
  border-radius: 3px 0 0 3px;
}
.sidebar--collapsed .sidebar-item--active {
  background: var(--color-bg-sidebar-active);
  box-shadow: inset 3px 0 0 var(--color-primary);
}
.sidebar--collapsed .sidebar-item--active .sidebar-item-icon {
  color: var(--color-primary); transform: scale(1.1);
  transition: transform var(--transition-fast);
}

.sidebar-footer { border-top: 1px solid var(--color-sidebar-border); padding: 8px; flex-shrink: 0; }
.sidebar-user-profile {
  display: flex; align-items: center; gap: 12px;
  padding: 8px 12px; border-radius: 10px;
  cursor: pointer; transition: background var(--transition-fast);
}
.sidebar-user-profile:hover { background: var(--color-bg-sidebar-hover); }
.sidebar-user-profile--collapsed { justify-content: center; padding: 8px; }
.sidebar-avatar { background: linear-gradient(135deg, #4f46e5, #7c3aed) !important; flex-shrink: 0; }
.sidebar-user-info { min-width: 0; flex: 1; }
.sidebar-user-name { font-size: 13px; font-weight: 600; color: var(--color-text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.sidebar-user-role { font-size: 11px; color: var(--color-text-tertiary); margin-top: 1px; }

.user-dropdown-panel { background: var(--color-bg-card); min-width: 200px; padding: 8px 0; }
.dropdown-divider { height: 1px; background: var(--color-border-light); margin: 4px 0; }
.dropdown-item {
  display: flex; align-items: center; gap: 10px;
  padding: 8px 16px; font-size: 13px; font-weight: 500;
  color: var(--color-text-primary); cursor: pointer;
  transition: all var(--transition-fast);
}
.dropdown-item:hover { background: var(--color-bg-hover); color: var(--color-primary); }
.dropdown-item-icon { font-size: 15px; color: var(--color-text-tertiary); width: 18px; display: flex; align-items: center; justify-content: center; }
.dropdown-item:hover .dropdown-item-icon { color: var(--color-primary); }
.dropdown-item--danger { color: var(--color-text-secondary); }
.dropdown-item--danger:hover { color: var(--color-danger); background: var(--color-danger-bg); }

:global(.user-profile-popover .ant-popover-inner) { padding: 0; border-radius: var(--radius-lg); box-shadow: var(--shadow-xl); border: 1px solid var(--color-border); overflow: hidden; }
:global(.user-profile-popover .ant-popover-inner-content) { padding: 0; }

.fade-enter-active, .fade-leave-active { transition: opacity 0.15s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }

@media (max-width: 768px) {
  .sidebar {
    transform: translateX(-100%);
    width: 280px;
    box-shadow: none;
  }
  .sidebar--mobile-open {
    transform: translateX(0);
    box-shadow: 4px 0 24px rgba(0, 0, 0, 0.2);
  }
  /* On mobile, always show full sidebar (not collapsed) */
  .sidebar--collapsed {
    width: 280px;
  }
  .sidebar--collapsed .sidebar-logo-text,
  .sidebar--collapsed .sidebar-item-label,
  .sidebar--collapsed .sidebar-item-badge,
  .sidebar--collapsed .sidebar-section-title,
  .sidebar--collapsed .sidebar-user-info {
    display: block !important;
    opacity: 1 !important;
  }
  .sidebar--collapsed .sidebar-item {
    padding: 0 16px;
    gap: 12px;
  }
  .sidebar--collapsed .sidebar-item--active {
    box-shadow: none;
    background: var(--color-bg-sidebar-active);
  }
  .sidebar--collapsed .sidebar-item--active .sidebar-item-icon {
    transform: none;
  }
  .sidebar--collapsed .sidebar-user-profile {
    justify-content: flex-start;
    padding: 8px 12px;
  }
  .sidebar--collapsed .sidebar-user-profile .sidebar-user-info {
    display: flex !important;
  }
  .sidebar--collapsed .sidebar-logo {
    padding: 0 20px;
    gap: 12px;
  }
  .sidebar-close-btn {
    display: flex;
  }
}
</style>
