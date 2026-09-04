<script setup lang="ts">
import type { JumpTurnItem } from '~/utils/chatJump'

const props = defineProps<{
  turns: JumpTurnItem[]
  activeId?: string
}>()

const emit = defineEmits<{
  (e: 'jump', id: string): void
}>()

const hoveredTurn = ref<JumpTurnItem | null>(null)
const previewPos = ref<{ x: number; y: number }>({ x: 0, y: 0 })

const handleMouseEnter = (event: MouseEvent, turn: JumpTurnItem) => {
  hoveredTurn.value = turn
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  previewPos.value = {
    x: rect.left - 12,
    y: rect.top + rect.height / 2,
  }
}

const handleMouseLeave = () => {
  hoveredTurn.value = null
}
</script>

<template>
  <div v-if="turns.length > 0" class="chat-jump-rail" :class="{ 'is-dense': turns.length > 12 }">
    <button
      v-for="turn in turns"
      :key="turn.id"
      type="button"
      class="chat-jump-tick"
      :class="{
        'is-active': activeId === turn.id,
        'is-hovered': hoveredTurn?.id === turn.id,
      }"
      :aria-label="turn.title"
      @click="emit('jump', turn.id)"
      @mouseenter="handleMouseEnter($event, turn)"
      @mouseleave="handleMouseLeave"
    />

    <!-- 悬停预览浮层 -->
    <Teleport to="body">
      <div
        v-if="hoveredTurn"
        class="chat-jump-preview"
        :style="{
          left: `${previewPos.x}px`,
          top: `${previewPos.y}px`,
        }"
      >
        <strong>{{ hoveredTurn.title }}</strong>
        <p>{{ hoveredTurn.preview }}</p>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.chat-jump-rail {
  position: absolute;
  z-index: 8;
  top: 56px;
  right: 8px;
  bottom: calc(var(--composer-height, 80px) + 24px);
  width: 22px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: flex-end;
  gap: 6px;
  pointer-events: none;
}
.chat-jump-rail.is-dense {
  gap: 3px;
}
.chat-jump-tick {
  pointer-events: auto;
  width: 18px;
  height: 10px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  border: 0;
  padding: 0;
  background: transparent;
  cursor: pointer;
}
.chat-jump-tick::after {
  content: "";
  width: 11px;
  height: 2px;
  border-radius: 99px;
  background: rgba(92, 76, 57, 0.28);
  transition: width 140ms ease, background-color 140ms ease;
}
.chat-jump-tick.is-active::after {
  width: 16px;
  height: 2.5px;
  background: #1890ff;
}
.chat-jump-tick.is-hovered::after,
.chat-jump-tick:hover::after {
  width: 18px;
  height: 3px;
  background: #096dd9;
}
.chat-jump-preview {
  position: fixed;
  z-index: 9999;
  width: min(300px, calc(100vw - 48px));
  transform: translate(-100%, -50%);
  display: grid;
  gap: 6px;
  border: 1px solid rgba(92, 76, 57, 0.15);
  border-radius: 10px;
  padding: 10px 12px;
  background: #ffffff;
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.12), 0 2px 6px rgba(0, 0, 0, 0.04);
  pointer-events: none;
}
.chat-jump-preview strong {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  overflow: hidden;
  color: #1f2937;
  font-size: 13px;
  font-weight: 600;
  line-height: 1.4;
}
.chat-jump-preview p {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
  overflow: hidden;
  margin: 0;
  color: #6b7280;
  font-size: 12px;
  line-height: 1.5;
}
</style>
