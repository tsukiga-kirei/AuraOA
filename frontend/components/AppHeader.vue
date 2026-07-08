<script setup lang="ts">
import { MenuUnfoldOutlined } from '@ant-design/icons-vue'

defineProps<{
  isMobile: boolean
}>()

const emit = defineEmits<{
  (e: 'toggleMobileMenu'): void
}>()

const { t } = useI18n()
</script>

<template>
  <header v-if="isMobile" class="app-header">
    <button
      class="header-toggle"
      @click="emit('toggleMobileMenu')"
      :aria-label="t('sidebar.expandMenu')"
    >
      <MenuUnfoldOutlined />
    </button>
  </header>
</template>

<style scoped>
.app-header {
  position: sticky;
  top: 0;
  z-index: 50;
  display: flex;
  align-items: center;
  height: var(--header-height);
  padding: 0 16px;
  background: var(--color-bg-page);
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
  outline: none;
}
.header-toggle:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}
.header-toggle:focus-visible {
  background: var(--color-bg-hover);
  color: var(--color-primary);
  box-shadow: 0 0 0 2px var(--color-primary-bg), 0 0 0 4px var(--color-primary-ring);
}
</style>
