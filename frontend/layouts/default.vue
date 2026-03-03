<script setup lang="ts">
import { ref, watch } from 'vue'

const { sidebarCollapsed: collapsed, restore: restoreLayoutPrefs } = useLayoutPrefs()
const { restore: restoreTheme } = useTheme()
const mobileMenuOpen = ref(false)
const isMobile = ref(false)

onMounted(() => {
  restoreTheme()
  restoreLayoutPrefs()
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

const route = useRoute()
watch(route, () => {
  if (isMobile.value) mobileMenuOpen.value = false
})
</script>

<template>
  <div class="app-layout" :class="{ 'app-layout--collapsed': collapsed }">
    <AppSidebar
      :collapsed="collapsed"
      :mobile-menu-open="mobileMenuOpen"
      @update:mobile-menu-open="mobileMenuOpen = $event"
    />

    <!--移动覆盖-->
    <transition name="overlay-fade">
      <div
        v-if="mobileMenuOpen && isMobile"
        class="sidebar-overlay"
        @click="mobileMenuOpen = false"
      />
    </transition>

    <!--主要内容-->
    <div class="main-wrapper">
      <AppHeader
        :collapsed="collapsed"
        :is-mobile="isMobile"
        :notification-count="3"
        @toggle-sidebar="collapsed = !collapsed"
        @toggle-mobile-menu="mobileMenuOpen = !mobileMenuOpen"
      />
      <main class="app-content">
        <slot />
      </main>
    </div>
  </div>
</template>

<style scoped>
.app-layout { display: flex; min-height: 100vh; background: var(--color-bg-page); }

.sidebar-overlay {
  position: fixed; inset: 0;
  background: rgba(0, 0, 0, 0.45);
  z-index: 99;
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
}

.overlay-fade-enter-active { transition: opacity 0.3s ease; }
.overlay-fade-leave-active { transition: opacity 0.25s ease; }
.overlay-fade-enter-from,
.overlay-fade-leave-to { opacity: 0; }

.main-wrapper {
  flex: 1;
  margin-left: var(--sidebar-width);
  transition: margin-left var(--transition-slow);
  display: flex; flex-direction: column;
  min-height: 100vh;
}
.app-layout--collapsed .main-wrapper { margin-left: var(--sidebar-collapsed-width); }

.app-content {
  flex: 1;
  padding: var(--space-page);
  max-width: 1400px;
  width: 100%;
  margin: 0 auto;
}

@media (max-width: 768px) {
  .main-wrapper { margin-left: 0 !important; }
  .app-content { padding: 16px; }
}
@media (max-width: 480px) {
  .app-content { padding: 12px; }
}
</style>
