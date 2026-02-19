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
const { t } = useI18n()
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

.rotate-icon-enter-active,
.rotate-icon-leave-active { transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1); }
.rotate-icon-enter-from { opacity: 0; transform: rotate(-120deg) scale(0.5); }
.rotate-icon-leave-to { opacity: 0; transform: rotate(120deg) scale(0.5); }

@media (max-width: 768px) {
  .app-header { padding: 0 16px; }
}
</style>
