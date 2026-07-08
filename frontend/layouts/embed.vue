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
  background: var(--color-bg-page);
  padding: 0;
  box-sizing: border-box;
  scrollbar-width: thin;
  scrollbar-color: transparent transparent;
}

.embed-layout--scrolling {
  scrollbar-color: color-mix(in srgb, var(--color-text-tertiary) 72%, transparent) transparent;
}

:global(html),
:global(body),
:global(#__nuxt) {
  background: var(--color-bg-page);
}

:global(html),
:global(body) {
  scrollbar-width: thin;
  scrollbar-color: transparent transparent;
}

:global(html.embed-scrollbar-active),
:global(html.embed-scrollbar-active body) {
  scrollbar-color: color-mix(in srgb, var(--color-text-tertiary) 72%, transparent) transparent;
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
  background: transparent;
  border-radius: var(--radius-full);
}

:global(html.embed-scrollbar-active::-webkit-scrollbar-thumb),
:global(html.embed-scrollbar-active body::-webkit-scrollbar-thumb) {
  background: color-mix(in srgb, var(--color-text-tertiary) 72%, transparent);
}
</style>
