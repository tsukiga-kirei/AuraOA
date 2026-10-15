<script setup lang="ts">
const isPageScrolling = ref(false)
let scrollTimer: ReturnType<typeof setTimeout> | null = null

const handlePageScroll = () => {
  isPageScrolling.value = true
  document.documentElement.classList.add('embed-scrollbar-active')
  if (scrollTimer) clearTimeout(scrollTimer)
  scrollTimer = setTimeout(() => {
    isPageScrolling.value = false
    document.documentElement.classList.remove('embed-scrollbar-active')
  }, 900)
}

onMounted(() => {
  window.addEventListener('scroll', handlePageScroll, { passive: true })
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', handlePageScroll)
  if (scrollTimer) clearTimeout(scrollTimer)
  document.documentElement.classList.remove('embed-scrollbar-active')
})
</script>

<template>
  <div class="embed-layout" :class="{ 'embed-layout--scrolling': isPageScrolling }">
    <slot />
  </div>
</template>

<style scoped>
.embed-layout {
  min-height: 100vh;
  background: var(--color-bg-page, #f8fafc);
  padding: 0;
  box-sizing: border-box;
}

:global(html),
:global(body),
:global(#__nuxt) {
  background: var(--color-bg-page, #f8fafc);
}

:global(html),
:global(body) {
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: #cbd5e1 transparent;
}

:global(body::-webkit-scrollbar),
:global(html::-webkit-scrollbar) {
  width: 6px;
  height: 6px;
}

:global(body::-webkit-scrollbar-track),
:global(html::-webkit-scrollbar-track) {
  background: transparent;
}

:global(body::-webkit-scrollbar-thumb),
:global(html::-webkit-scrollbar-thumb) {
  background: #cbd5e1;
  border-radius: 999px;
  transition: background 0.2s ease;
}

:global(body::-webkit-scrollbar-thumb:hover),
:global(html::-webkit-scrollbar-thumb:hover) {
  background: #94a3b8;
}
</style>
