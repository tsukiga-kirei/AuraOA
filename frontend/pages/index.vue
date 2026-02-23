<script setup lang="ts">
/**
 * Root index page — redirects to the user's default page based on their permissions.
 * If not authenticated, redirects to /login.
 */
import { getDefaultPage } from '~/composables/useMockData'

definePageMeta({ middleware: 'auth', layout: 'default' })

const { userPermissions } = useAuth()

onMounted(() => {
  navigateTo(getDefaultPage(userPermissions.value))
})
</script>

<template>
  <div style="display: flex; align-items: center; justify-content: center; height: 60vh;">
    <a-spin size="large" tip="Loading..." />
  </div>
</template>
