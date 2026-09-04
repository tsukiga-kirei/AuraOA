<script setup lang="ts">
import {
  DownOutlined,
} from '@ant-design/icons-vue'
import type { EffectiveAgentItem } from '~/types/chat'

const props = defineProps<{
  agents: EffectiveAgentItem[]
  currentAgentCode?: string
}>()

const emit = defineEmits<{
  (e: 'select', agentCode: string): void
}>()

const { t } = useI18n()

const currentAgent = computed(() => {
  return props.agents.find(a => a.code === props.currentAgentCode) || props.agents[0]
})
</script>

<template>
  <div class="chat-header-wrapper">
    <div class="agent-selector">
      <a-dropdown :trigger="['click']">
        <div class="agent-badge">
          <span class="agent-emoji">{{ currentAgent?.avatar_emoji || '🤖' }}</span>
          <span class="agent-name">{{ currentAgent?.name || t('chat.defaultAgentName') }}</span>
          <DownOutlined class="dropdown-icon" />
        </div>
        <template #overlay>
          <a-menu>
            <a-menu-item
              v-for="agent in agents"
              :key="agent.code"
              @click="emit('select', agent.code)"
            >
              <div class="menu-agent-item">
                <span class="menu-emoji">{{ agent.avatar_emoji }}</span>
                <div class="menu-agent-info">
                  <div class="menu-agent-title">{{ agent.name }}</div>
                  <div class="menu-agent-desc">{{ agent.description }}</div>
                </div>
              </div>
            </a-menu-item>
          </a-menu>
        </template>
      </a-dropdown>
    </div>

    <div class="chat-header-desc" v-if="currentAgent?.description">
      {{ currentAgent.description }}
    </div>
  </div>
</template>

<style scoped>
.chat-header-wrapper {
  padding: 12px 20px;
  border-bottom: 1px solid #ebe5df;
  background: #ffffff;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.agent-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 4px 10px;
  border-radius: 6px;
  transition: background-color 0.15s;
}
.agent-badge:hover {
  background: rgba(0, 0, 0, 0.04);
}
.agent-emoji {
  font-size: 18px;
}
.agent-name {
  font-size: 14px;
  font-weight: 600;
  color: #1f2937;
}
.dropdown-icon {
  font-size: 10px;
  color: #8c8c8c;
}
.chat-header-desc {
  font-size: 12px;
  color: #6b7280;
}
.menu-agent-item {
  display: flex;
  gap: 10px;
  padding: 4px 0;
  max-width: 260px;
}
.menu-emoji {
  font-size: 16px;
}
.menu-agent-title {
  font-weight: 500;
  color: #1f2937;
}
.menu-agent-desc {
  font-size: 11px;
  color: #8c8c8c;
  white-space: normal;
}
</style>
