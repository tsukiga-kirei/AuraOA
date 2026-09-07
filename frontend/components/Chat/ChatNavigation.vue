<script setup lang="ts">
import { PlusOutlined, MoreOutlined, SearchOutlined, DownOutlined, FolderOutlined } from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
defineProps<{ compact?: boolean }>()
const emit = defineEmits<{ navigate: [path: string] }>()
const { t } = useI18n()
const route = useRoute()
const { sessions, effectiveAgents, currentSessionId, selectedAgentCode, loading, error, total, initialize, newConversation, fetchSessions, renameSession, deleteSession } = useChatSession()
const collapsedAgents = useState<Record<string, boolean>>('chat-collapsed-agents', () => ({}))
const searchOpen = ref(false)
const groups = computed(() => {
  const all = new Map(effectiveAgents.value.map(agent => [agent.agent_code, { code: agent.agent_code, name: agent.name, available: true, sessions: [] as typeof sessions.value }]))
  for (const session of sessions.value) {
    if (!all.has(session.agent_code)) all.set(session.agent_code, { code: session.agent_code, name: session.agent_name || session.agent_code, available: false, sessions: [] })
    all.get(session.agent_code)!.sessions.push(session)
  }
  return [...all.values()]
})
const expanded = (code: string) => !collapsedAgents.value[code]
watch(selectedAgentCode, code => { if (code) collapsedAgents.value[code] = false })
const editing = ref<string | null>(null)
const title = ref('')
onMounted(initialize)
const start = (code: string) => {
  if (!code) return
  collapsedAgents.value[code] = false
  newConversation(code)
  emit('navigate', `/chat?agent=${encodeURIComponent(code)}`)
}
const rename = async () => {
  if (!editing.value || !title.value.trim()) return
  try { await renameSession(editing.value, title.value.trim()); editing.value = null }
  catch (err: any) { message.error(err.message) }
}
const remove = async (id: string) => {
  try { await deleteSession(id); if (route.query.session === id) emit('navigate', '/chat') }
  catch (err: any) { message.error(err.message) }
}
const isAgentActive = (code: string) => route.path === '/chat' && selectedAgentCode.value === code && !currentSessionId.value
const isSessionActive = (id: string) => route.path === '/chat' && currentSessionId.value === id
</script>

<template>
  <section class="chat-navigation" :class="{ compact }" :aria-label="t('chat.navigation')">
    <button
      class="sidebar-item search-entry"
      :class="{ 'sidebar-item--active': searchOpen }"
      :title="t('chat.search.title')"
      @click="searchOpen = true"
    >
      <SearchOutlined class="sidebar-item-icon" />
      <span v-if="!compact" class="sidebar-item-label">{{ t('chat.search.title') }}</span>
    </button>

    <template v-if="!compact">
      <section v-for="group in groups" :key="group.code" class="agent-folder">
        <div class="folder-heading" :class="{ 'sidebar-item--active': isAgentActive(group.code) }">
          <button class="nav-agent" :aria-expanded="expanded(group.code)" @click="collapsedAgents[group.code] = expanded(group.code)">
            <DownOutlined class="folder-chevron" :class="{ closed: !expanded(group.code) }" />
            <FolderOutlined class="sidebar-item-icon" />
            <span class="sidebar-item-label">{{ group.name }}</span>
          </button>
          <button v-if="group.available" class="new-icon" :aria-label="t('chat.newAgentSession', [group.name])" :title="t('chat.newSession')" @click="start(group.code)"><PlusOutlined /></button>
          <div v-if="isAgentActive(group.code)" class="sidebar-item-indicator" />
        </div>
        <div v-if="expanded(group.code)" class="folder-sessions">
          <div v-for="session in group.sessions" :key="session.id" class="nav-session" :class="{ 'sidebar-item--active': isSessionActive(session.id) }">
            <button class="session-title" :title="session.title" @click="emit('navigate', `/chat?session=${session.id}`)">{{ session.title || t('chat.newSession') }}</button>
            <a-dropdown :trigger="['click']">
              <button class="session-more" :aria-label="t('chat.sessionActions')"><MoreOutlined /></button>
              <template #overlay>
                <a-menu>
                  <a-menu-item @click="editing = session.id; title = session.title">{{ t('chat.renameSession') }}</a-menu-item>
                  <a-menu-item danger><a-popconfirm :title="t('chat.deleteConfirm')" @confirm="remove(session.id)">{{ t('chat.deleteSession') }}</a-popconfirm></a-menu-item>
                </a-menu>
              </template>
            </a-dropdown>
            <div v-if="isSessionActive(session.id)" class="sidebar-item-indicator" />
          </div>
          <button v-if="!group.sessions.length && group.available" class="folder-empty" @click="start(group.code)">{{ t('chat.startAgentSession') }}</button>
        </div>
      </section>
      <div v-if="error" class="nav-empty" role="alert">{{ error }}</div>
      <div v-else-if="loading" class="nav-empty" role="status">{{ t('chat.searching') }}</div>
      <div v-else-if="!groups.length" class="nav-empty">{{ t('chat.noAgents') }}</div>
      <button v-if="sessions.length < total" class="nav-load" :disabled="loading" @click="fetchSessions('', true)">{{ t('chat.loadMore') }}</button>
    </template>

    <a-modal :open="!!editing" :title="t('chat.renameSession')" @ok="rename" @cancel="editing = null">
      <a-input v-model:value="title" :maxlength="128" @pressEnter="rename" />
    </a-modal>
    <ChatSearchModal :open="searchOpen" @close="searchOpen = false" @navigate="emit('navigate', $event)" @start="start" />
  </section>
</template>

<style scoped>
.chat-navigation { padding: 0 0 8px; color: var(--color-text-sidebar); min-width: 0; width: 100%; max-width: 100%; overflow: hidden; }
button { font: inherit; color: inherit; cursor: pointer; }
.search-entry { margin-bottom: 6px; }
.sidebar-item {
  display: flex; align-items: center;
  padding: 0 16px; height: 44px;
  margin: 2px 8px; border-radius: 10px;
  cursor: pointer; transition: background-color var(--transition-fast), color var(--transition-fast);
  position: relative; gap: 12px; border: 0; width: calc(100% - 16px);
  background: transparent; color: var(--color-text-sidebar); text-align: left;
}
.sidebar-item:hover { background: var(--color-bg-sidebar-hover); color: var(--color-text-primary); }
.sidebar-item--active { background: var(--color-bg-sidebar-active); color: var(--color-text-sidebar-active); }
.sidebar-item-icon { font-size: 18px; flex-shrink: 0; width: 20px; display: flex; align-items: center; justify-content: center; }
.sidebar-item-label { font-size: 14px; font-weight: 500; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; flex: 1; min-width: 0; }
.sidebar-item-indicator {
  position: absolute; right: 0; top: 50%; transform: translateY(-50%);
  width: 3px; height: 20px; background: var(--color-primary);
  border-radius: 3px 0 0 3px;
}
.folder-heading {
  display: flex; align-items: center; margin: 2px 8px; border-radius: 10px; position: relative;
}
.folder-heading:hover, .nav-session:hover { background: var(--color-bg-sidebar-hover); }
.nav-agent {
  display: flex; align-items: center; gap: 8px; flex: 1; min-width: 0;
  border: 0; background: transparent; padding: 10px 8px; text-align: left;
}
.folder-chevron { font-size: 9px; color: var(--color-text-tertiary); transition: transform .18s ease; }
.folder-chevron.closed { transform: rotate(-90deg); }
.new-icon { flex-shrink: 0; background: none; border: 0; padding: 8px; opacity: 0; color: var(--color-text-secondary); }
.folder-heading:hover .new-icon, .new-icon:focus-visible { opacity: 1; }
.folder-sessions { margin: 2px 8px 8px 28px; padding-left: 4px; border-left: 1px solid var(--color-sidebar-border); min-width: 0; }
.folder-empty { display: block; width: 100%; border: 0; background: none; color: var(--color-text-tertiary); padding: 8px 10px; font-size: 12px; text-align: left; }
.nav-session { display: flex; align-items: center; border-radius: 8px; padding: 0 6px 0 10px; min-height: 36px; position: relative; }
.session-title { flex: 1; min-width: 0; text-align: left; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; font-size: 12px; padding: 8px 0; }
.session-title, .session-more { background: transparent; border: 0; }
.session-more { padding: 6px; opacity: 0; }
.nav-session:hover .session-more, .session-more:focus-visible { opacity: 1; }
.nav-empty, .nav-load { font-size: 12px; color: var(--color-text-tertiary); padding: 10px 16px; }
.nav-load { border: 0; background: none; }
.compact .search-entry { justify-content: center; padding: 0; width: calc(100% - 16px); }
@media (hover: none) { .session-more, .new-icon { opacity: 1; } }
</style>
