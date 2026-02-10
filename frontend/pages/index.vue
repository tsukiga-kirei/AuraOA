<script setup lang="ts">
definePageMeta({ layout: false })

onMounted(() => {
  navigateTo('/dashboard')
})
</script>

<template>
  <div style="min-height: 100vh; display: flex; align-items: center; justify-content: center; background: var(--color-bg-page);">
    <div style="text-align: center; color: var(--color-text-tertiary);">
      正在跳转...
    </div>
  </div>
</template>
