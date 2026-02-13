<script setup lang="ts">
import {
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  BellOutlined,
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
      <a-tooltip title="切换主题" placement="bottom" :mouse-enter-delay="0.5">
        <button class="header-action" @click="toggleTheme">
          <transition name="rotate-icon" mode="out-in">
            <svg v-if="isDark" key="moon" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path></svg>
            <svg v-else key="sun" xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="5"></circle><line x1="12" y1="1" x2="12" y2="3"></line><line x1="12" y1="21" x2="12" y2="23"></line><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line><line x1="1" y1="12" x2="3" y2="12"></line><line x1="21" y1="12" x2="23" y2="12"></line><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line></svg>
          </transition>
        </button>
      </a-tooltip>

      <a-tooltip title="消息通知" placement="bottom" :mouse-enter-delay="0.5">
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


.rotate-icon-enter-active,
.rotate-icon-leave-active { transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1); }
.rotate-icon-enter-from { opacity: 0; transform: rotate(-120deg) scale(0.5); }
.rotate-icon-leave-to { opacity: 0; transform: rotate(120deg) scale(0.5); }

@media (max-width: 768px) {
  .app-header { padding: 0 16px; }
}
</style>
