<script setup lang="ts">
// Parent route wrapper for /admin/system/* pages.
// Layout is handled by admin-system.vue, child pages render via NuxtPage.
definePageMeta({ middleware: 'auth', layout: 'admin-system' })
</script>

<template>
  <NuxtPage />
</template>
