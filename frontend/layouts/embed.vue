<template>
  <div class="embed-layout">
    <slot />
  </div>
</template>

<style scoped>
.embed-layout {
  min-height: 100vh;
  background: var(--color-bg-page);
  padding: 12px;
  box-sizing: border-box;
}
</style>
