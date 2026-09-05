<script setup lang="ts">
import {
  MenuFoldOutlined,
  MenuUnfoldOutlined,
} from '@ant-design/icons-vue'

// props：侧边栏折叠状态 / 移动端菜单是否展开 / 是否移动端
const props = defineProps<{
  collapsed: boolean
  mobileMenuOpen: boolean
  isMobile: boolean
}>()

// emit：更新移动端菜单展开状态 / 切换侧边栏 / 切换移动端菜单
const emit = defineEmits<{
  (e: 'update:mobileMenuOpen', val: boolean): void
  (e: 'toggleSidebar'): void
  (e: 'toggleMobileMenu'): void
}>()

const { sections, isMenuActive, logoTarget } = useSidebarMenu()
const { t } = useI18n()
const chatAllowed = computed(() => sections.value.some(section => section.items.some(item => item.key === '/chat')))
const navigationSections = computed(() => sections.value.map(section => ({ ...section, items: section.items.filter(item => item.key !== '/chat') })))

// 侧栏宽度动画期间先隐藏文字，避免折叠时文字溢出
const isSidebarTransitioning = ref(false)
let sidebarTransitionTimer: ReturnType<typeof setTimeout> | null = null

const showSidebarText = computed(() =>
  (!props.collapsed || props.mobileMenuOpen) && !isSidebarTransitioning.value,
)

watch(() => props.collapsed, () => {
  if (sidebarTransitionTimer) clearTimeout(sidebarTransitionTimer)
  isSidebarTransitioning.value = true
  sidebarTransitionTimer = setTimeout(() => {
    isSidebarTransitioning.value = false
    sidebarTransitionTimer = null
  }, 320)
})

onUnmounted(() => {
  if (sidebarTransitionTimer) clearTimeout(sidebarTransitionTimer)
})

// 点击菜单项：跳转路由并关闭移动端菜单
const handleMenuClick = (path: string) => {
  navigateTo(path)
  emit('update:mobileMenuOpen', false)
}

const handleToggleSidebar = () => {
  if (props.isMobile) {
    emit('toggleMobileMenu')
    return
  }
  emit('toggleSidebar')
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
    <!--侧栏头部：Logo + 收起/展开-->
    <div
      class="sidebar-header"
      :class="{
        'sidebar-header--expanded': !collapsed || mobileMenuOpen,
        'sidebar-header--compact': collapsed && !mobileMenuOpen,
      }"
    >
      <!--收起态：Logo 悬停切换为展开图标-->
      <template v-if="collapsed && !mobileMenuOpen">
        <button
          type="button"
          class="sidebar-compact-brand"
          :aria-label="t('sidebar.expandMenu')"
          @click="handleToggleSidebar"
        >
          <span class="sidebar-mark-slot">
            <span class="sidebar-mark-logo">
              <img src="/favicon.svg" alt="AuraOA" width="24" height="24" />
            </span>
            <span class="sidebar-mark-toggle" aria-hidden="true">
              <MenuUnfoldOutlined />
            </span>
          </span>
        </button>
        <span class="sidebar-expand-hint" aria-hidden="true">{{ t('sidebar.expandMenu') }}</span>
      </template>

      <!--展开态：品牌区 + 收起按钮-->
      <template v-else>
        <div class="sidebar-brand" @click="navigateTo(logoTarget)">
          <div class="sidebar-logo-icon">
            <img src="/favicon.svg" alt="AuraOA" width="24" height="24" />
          </div>
          <transition name="fade">
            <span v-if="showSidebarText" class="sidebar-logo-text">{{ t('app.name') }}</span>
          </transition>
        </div>
        <button
          v-if="!mobileMenuOpen"
          type="button"
          class="sidebar-toggle sidebar-hint-below"
          :aria-label="t('sidebar.collapseMenu')"
          :data-hint="t('sidebar.collapseMenu')"
          @click="handleToggleSidebar"
        >
          <MenuFoldOutlined />
        </button>
        <!--移动端关闭按钮-->
        <button
          v-else
          class="sidebar-close-btn"
          @click.stop="emit('update:mobileMenuOpen', false)"
          :aria-label="t('sidebar.closeMenu')"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none"
            stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </button>
      </template>
    </div>

    <!--权限驱动的导航区域-->
    <nav class="sidebar-nav">
      <div v-for="section in navigationSections" :key="section.id" class="sidebar-section">
        <div v-if="showSidebarText" class="sidebar-section-title">{{ t(section.titleKey) }}</div>
        <template v-for="item in section.items" :key="item.key">
          <!--折叠状态：用 Tooltip 包裹显示菜单名-->
          <a-tooltip
            v-if="collapsed && !mobileMenuOpen"
            :key="'tooltip-' + item.key"
            :title="t(item.labelKey)"
            placement="right"
            :mouse-enter-delay="0.1"
            :arrow="false"
            overlay-class-name="sidebar-nav-tooltip"
          >
            <div
              class="sidebar-item"
              :class="{ 'sidebar-item--active': isMenuActive(item.key) }"
              @click="handleMenuClick(item.key)"
            >
              <component :is="item.icon" class="sidebar-item-icon" />
              <!--折叠时隐藏文字，保持结构一致-->
              <div v-if="isMenuActive(item.key)" class="sidebar-item-indicator" />
            </div>
          </a-tooltip>

          <!--展开/移动状态：无工具提示-->
          <div
            v-else
            :key="'item-' + item.key"
            class="sidebar-item"
            :class="{ 'sidebar-item--active': isMenuActive(item.key) }"
            @click="handleMenuClick(item.key)"
          >
            <component :is="item.icon" class="sidebar-item-icon" />
            <transition name="fade">
              <span class="sidebar-item-label">{{ t(item.labelKey) }}</span>
            </transition>
            <transition name="fade">
              <span v-if="item.badge" class="sidebar-item-badge">{{ item.badge }}</span>
            </transition>
            <div v-if="isMenuActive(item.key)" class="sidebar-item-indicator" />
          </div>
        </template>
      </div>
      <ChatNavigation v-if="chatAllowed" :compact="!showSidebarText" @navigate="handleMenuClick" />
    </nav>

    <!--底部用户菜单-->
    <div class="sidebar-footer">
      <SidebarUserMenu
        :collapsed="collapsed"
        :mobile-menu-open="mobileMenuOpen"
        @navigate="handleMenuClick"
      />
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
  overflow: visible;
}
.sidebar--collapsed { width: var(--sidebar-collapsed-width); }

.sidebar-header {
  position: relative;
  display: flex;
  align-items: center;
  height: var(--header-height);
  width: 100%;
  flex-shrink: 0;
  border-bottom: 1px solid var(--color-sidebar-border);
}
.sidebar-header--expanded {
  gap: 8px;
  padding: 0 12px 0 16px;
}
.sidebar-header--compact {
  justify-content: center;
  padding: 0;
}

.sidebar-brand {
  display: flex;
  min-width: 0;
  flex: 1;
  align-items: center;
  gap: 12px;
  cursor: pointer;
}

.sidebar-compact-brand {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: none;
  padding: 0;
  background: transparent;
  cursor: pointer;
}

.sidebar-mark-slot {
  position: relative;
  display: flex;
  width: 36px;
  height: 36px;
  align-items: center;
  justify-content: center;
}

.sidebar-mark-logo,
.sidebar-mark-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: opacity var(--transition-fast), transform var(--transition-fast);
}

.sidebar-mark-logo {
  width: 36px;
  height: 36px;
  background: var(--color-bg-hover);
  border-radius: 10px;
  color: var(--color-primary);
}

.sidebar-mark-toggle {
  position: absolute;
  inset: 0;
  border-radius: var(--radius-md);
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
  font-size: 16px;
  opacity: 0;
  transform: scale(0.92);
  pointer-events: none;
}

.sidebar-header--compact:hover .sidebar-mark-logo,
.sidebar-header--compact:focus-within .sidebar-mark-logo {
  opacity: 0;
  transform: scale(0.92);
}

.sidebar-header--compact:hover .sidebar-mark-toggle,
.sidebar-header--compact:focus-within .sidebar-mark-toggle {
  opacity: 1;
  transform: scale(1);
}

.sidebar-expand-hint {
  position: absolute;
  left: calc(100% + 6px);
  top: 50%;
  z-index: 110;
  padding: 6px 12px;
  border-radius: var(--radius-full);
  background: #0f172a;
  color: #f8fafc;
  font-size: 13px;
  font-weight: 500;
  line-height: 1.2;
  white-space: nowrap;
  box-shadow: var(--shadow-sm);
  opacity: 0;
  pointer-events: none;
  transform: translate(-4px, -50%);
  transition: opacity var(--transition-fast), transform var(--transition-fast);
}

html[data-theme='dark'] .sidebar-expand-hint {
  background: #f1f5f9;
  color: #0f172a;
}

.sidebar-header--compact:hover .sidebar-expand-hint,
.sidebar-header--compact:focus-within .sidebar-expand-hint {
  opacity: 1;
  transform: translate(0, -50%);
}

.sidebar-compact-brand:focus-visible {
  outline: 2px solid var(--color-primary-ring);
  outline-offset: 2px;
  border-radius: var(--radius-md);
}

.sidebar-toggle {
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: var(--radius-md);
  background: var(--color-bg-hover);
  color: var(--color-text-secondary);
  font-size: 16px;
  cursor: pointer;
  transition: background-color var(--transition-fast), color var(--transition-fast);
}
.sidebar-toggle:hover {
  background: var(--color-bg-sidebar-hover);
  color: var(--color-text-primary);
}

.sidebar-hint-below {
  position: relative;
}
.sidebar-hint-below::after {
  content: attr(data-hint);
  position: absolute;
  top: calc(100% + 8px);
  left: 50%;
  z-index: 110;
  padding: 6px 10px;
  border-radius: var(--radius-md);
  background: #0f172a;
  color: #f8fafc;
  font-size: 12px;
  font-weight: 500;
  line-height: 1.2;
  white-space: nowrap;
  box-shadow: var(--shadow-sm);
  opacity: 0;
  pointer-events: none;
  transform: translate(-50%, -4px);
  transition: opacity var(--transition-fast), transform var(--transition-fast);
}
html[data-theme='dark'] .sidebar-hint-below::after {
  background: #f1f5f9;
  color: #0f172a;
}
.sidebar-hint-below:hover::after,
.sidebar-hint-below:focus-visible::after {
  opacity: 1;
  transform: translate(-50%, 0);
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
  white-space: nowrap; letter-spacing: 0;
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

.sidebar-nav { flex: 1; padding: 12px 0; overflow-y: auto; overflow-x: hidden; min-height: 0; }
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
  /*移动端始终展示完整侧边栏，不折叠*/
  .sidebar--collapsed {
    width: 280px;
  }
  .sidebar--collapsed .sidebar-logo-text,
  .sidebar--collapsed .sidebar-item-label,
  .sidebar--collapsed .sidebar-item-badge,
  .sidebar--collapsed .sidebar-section-title {
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
  .sidebar--collapsed .sidebar-header {
    padding: 0 20px;
    gap: 12px;
  }
  .sidebar--collapsed .sidebar-header--compact {
    padding: 0;
  }
  .sidebar-close-btn {
    display: flex;
  }
}
</style>
