<script setup lang="ts">
import {
  PlusOutlined,
  SearchOutlined,
  MessageOutlined,
  MoreOutlined,
  EditOutlined,
  DeleteOutlined,
  PushpinOutlined,
  CheckOutlined,
  CloseOutlined,
} from '@ant-design/icons-vue'
import type { ChatSessionItem } from '~/types/chat'

const props = defineProps<{
  sessions: ChatSessionItem[]
  currentSessionId: string | null
  loading: boolean
}>()

const emit = defineEmits<{
  (e: 'select', id: string): void
  (e: 'create'): void
  (e: 'rename', id: string, newTitle: string): void
  (e: 'delete', id: string): void
  (e: 'search', keyword: string): void
}>()

const { t } = useI18n()
const searchKeyword = ref('')
const renamingId = ref<string | null>(null)
const renameDraft = ref('')

const handleSearch = () => {
  emit('search', searchKeyword.value.trim())
}

const startRename = (session: ChatSessionItem) => {
  renamingId.value = session.id
  renameDraft.value = session.title
}

const commitRename = (id: string) => {
  if (!renameDraft.value.trim()) {
    renamingId.value = null
    return
  }
  emit('rename', id, renameDraft.value.trim())
  renamingId.value = null
}

const cancelRename = () => {
  renamingId.value = null
}
</script>

<template>
  <div class="chat-sidebar-wrapper">
    <div class="chat-sidebar-header">
      <a-button type="primary" block class="new-chat-btn" @click="emit('create')">
        <template #icon><PlusOutlined /></template>
        {{ t('chat.newSession') }}
      </a-button>

      <div class="search-box">
        <a-input
          v-model:value="searchKeyword"
          :placeholder="t('chat.searchSessions')"
          allow-clear
          size="small"
          @press-enter="handleSearch"
        >
          <template #prefix><SearchOutlined style="color: #bfbfbf" /></template>
        </a-input>
      </div>
    </div>

    <div class="chat-session-list" :class="{ 'is-loading': loading }">
      <div v-if="sessions.length === 0 && !loading" class="empty-sessions">
        {{ t('chat.emptySessions') }}
      </div>

      <div
        v-for="item in sessions"
        :key="item.id"
        class="session-item"
        :class="{ 'is-active': currentSessionId === item.id }"
        @click="emit('select', item.id)"
      >
        <span class="session-avatar">{{ item.agent_avatar_emoji || '🤖' }}</span>

        <!-- 标题重命名态 -->
        <div v-if="renamingId === item.id" class="session-renaming" @click.stop>
          <input
            v-model="renameDraft"
            class="rename-input"
            @keydown.enter="commitRename(item.id)"
            @keydown.esc="cancelRename"
          />
          <button class="rename-action-btn" @click="commitRename(item.id)"><CheckOutlined /></button>
          <button class="rename-action-btn" @click="cancelRename"><CloseOutlined /></button>
        </div>

        <!-- 正常展示态 -->
        <div v-else class="session-info">
          <span class="session-title">{{ item.title || t('chat.newSession') }}</span>
          <span class="session-time">{{ item.last_message_at ? new Date(item.last_message_at).toLocaleDateString() : '' }}</span>
        </div>

        <!-- 操作下拉菜单 -->
        <div class="session-actions" @click.stop v-if="renamingId !== item.id">
          <a-dropdown :trigger="['click']">
            <button class="session-menu-btn"><MoreOutlined /></button>
            <template #overlay>
              <a-menu>
                <a-menu-item @click="startRename(item)">
                  <EditOutlined /> {{ t('chat.renameSession') }}
                </a-menu-item>
                <a-menu-item danger @click="emit('delete', item.id)">
                  <DeleteOutlined /> {{ t('chat.deleteSession') }}
                </a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.chat-sidebar-wrapper {
  width: 260px;
  height: 100%;
  border-right: 1px solid #ebe5df;
  background: #faf8f5;
  display: flex;
  flex-direction: column;
}
.chat-sidebar-header {
  padding: 16px 12px 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  border-bottom: 1px solid #ebe5df;
}
.new-chat-btn {
  border-radius: 8px;
}
.chat-session-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.empty-sessions {
  text-align: center;
  color: #8c8c8c;
  font-size: 13px;
  padding: 32px 0;
}
.session-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 8px;
  cursor: pointer;
  transition: background-color 0.15s;
  user-select: none;
}
.session-item:hover {
  background: rgba(0, 0, 0, 0.04);
}
.session-item.is-active {
  background: #e6f7ff;
}
.session-avatar {
  font-size: 16px;
  flex-shrink: 0;
}
.session-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.session-title {
  font-size: 13.5px;
  color: #1f2937;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.session-time {
  font-size: 11px;
  color: #9ca3af;
}
.session-actions {
  opacity: 0;
  transition: opacity 0.15s;
}
.session-item:hover .session-actions {
  opacity: 1;
}
.session-menu-btn {
  background: transparent;
  border: none;
  cursor: pointer;
  color: #8c8c8c;
  padding: 2px 4px;
  border-radius: 4px;
}
.session-menu-btn:hover {
  color: #1890ff;
  background: rgba(0, 0, 0, 0.05);
}
.session-renaming {
  display: flex;
  align-items: center;
  gap: 4px;
  flex: 1;
}
.rename-input {
  flex: 1;
  border: 1px solid #1890ff;
  border-radius: 4px;
  padding: 2px 6px;
  font-size: 12px;
  outline: none;
}
.rename-action-btn {
  background: transparent;
  border: none;
  cursor: pointer;
  padding: 2px;
  font-size: 12px;
}
</style>
