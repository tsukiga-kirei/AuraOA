<script setup lang="ts">
/**
 * 表格单元格溢出提示：仅当内容超出可视宽度/高度时，才用主题 tooltip 展示全文。
 */
const props = withDefaults(defineProps<{
  text?: string
  block?: boolean
  delay?: number
}>(), {
  delay: 0.35,
})

const overflowing = ref(false)

function isOverflowing(el: HTMLElement) {
  if (el.scrollWidth > el.clientWidth + 1 || el.scrollHeight > el.clientHeight + 1) return true
  return Array.from(el.querySelectorAll('[data-overflow-check]')).some((child) => {
    const node = child as HTMLElement
    return node.scrollWidth > node.clientWidth + 1 || node.scrollHeight > node.clientHeight + 1
  })
}

function onEnter(e: MouseEvent) {
  overflowing.value = isOverflowing(e.currentTarget as HTMLElement)
}
</script>

<template>
  <a-tooltip :title="overflowing && text ? text : undefined" :mouse-enter-delay="props.delay">
    <span
        class="overflow-tooltip-host"
        :class="{ 'overflow-tooltip-host--block': block }"
        @mouseenter="onEnter"
    >
      <slot />
    </span>
  </a-tooltip>
</template>

<style scoped>
.overflow-tooltip-host {
  display: inline-block;
  max-width: 100%;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  vertical-align: middle;
}

.overflow-tooltip-host--block {
  display: block;
  width: 100%;
}

.overflow-tooltip-host :deep(.source-action-stack),
.overflow-tooltip-host :deep(.result-tag--fit) {
  max-width: 100%;
  min-width: 0;
  vertical-align: middle;
}
</style>
