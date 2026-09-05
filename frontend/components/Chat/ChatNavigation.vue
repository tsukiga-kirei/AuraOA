<script setup lang="ts">
import { EditOutlined, PlusOutlined, MoreOutlined, SearchOutlined, FolderOutlined, DownOutlined } from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
defineProps<{ compact?: boolean }>()
const emit = defineEmits<{ navigate: [path: string] }>()
const { t } = useI18n()
const route = useRoute()
const { sessions, effectiveAgents, currentSessionId, selectedAgentCode, loading, error, total, initialize, newConversation, fetchSessions, renameSession, deleteSession } = useChatSession()
const keyword = ref('')
const collapsedAgents = useState<Record<string, boolean>>('chat-collapsed-agents', () => ({}))
const groups = computed(() => {
  const all = new Map(effectiveAgents.value.map(agent => [agent.agent_code, { code: agent.agent_code, name: agent.name, available: true, sessions: [] as typeof sessions.value }]))
  for (const session of sessions.value) {
    if (!all.has(session.agent_code)) all.set(session.agent_code, { code: session.agent_code, name: session.agent_name || session.agent_code, available: false, sessions: [] })
    all.get(session.agent_code)!.sessions.push(session)
  }
  return [...all.values()].filter(group => !keyword.value.trim() || group.sessions.length)
})
const expanded = (code: string) => !!keyword.value.trim() || !collapsedAgents.value[code]
watch(selectedAgentCode, code => { if (code) collapsedAgents.value[code] = false })
let searchTimer: ReturnType<typeof setTimeout> | undefined
const search = () => { clearTimeout(searchTimer); searchTimer = setTimeout(() => fetchSessions(keyword.value), 200) }
onBeforeUnmount(() => clearTimeout(searchTimer))
const editing = ref<string | null>(null)
const title = ref('')
onMounted(initialize)
const start = (code = selectedAgentCode.value || effectiveAgents.value[0]?.agent_code) => { if (code) collapsedAgents.value[code] = false; newConversation(code); emit('navigate', '/chat') }
const rename = async () => {
  if (!editing.value || !title.value.trim()) return
  try { await renameSession(editing.value, title.value.trim()); editing.value = null }
  catch (err: any) { message.error(err.message) }
}
const remove = async (id: string) => {
  try { await deleteSession(id); if (route.query.session === id) emit('navigate', '/chat') }
  catch (err: any) { message.error(err.message) }
}
</script>

<template>
  <section class="chat-navigation" :class="{ compact }" :aria-label="t('chat.navigation')">
    <button class="nav-new" :title="t('chat.newSession')" :aria-label="t('chat.newSession')" @click="start()"><EditOutlined /><span v-if="!compact">{{ t('chat.newSession') }}</span></button>
    <template v-if="!compact">
      <div class="nav-search"><SearchOutlined /><input v-model="keyword" :placeholder="t('chat.searchSessions')" :aria-label="t('chat.searchSessions')" @input="search" /></div>
      <div class="nav-caption">{{ t('chat.agents') }}</div>
      <section v-for="group in groups" :key="group.code" class="agent-folder">
        <div class="folder-heading" :class="{ selected: route.path === '/chat' && selectedAgentCode === group.code && !currentSessionId }">
          <button class="nav-agent" :aria-expanded="expanded(group.code)" @click="collapsedAgents[group.code] = expanded(group.code)"><DownOutlined class="folder-chevron" :class="{ closed: !expanded(group.code) }" /><FolderOutlined /><span>{{ group.name }}</span></button>
          <button v-if="group.available" class="new-icon" :aria-label="t('chat.newAgentSession', [group.name])" :title="t('chat.newSession')" @click="start(group.code)"><PlusOutlined /></button>
        </div>
        <div v-if="expanded(group.code)" class="folder-sessions">
      <div v-for="session in group.sessions" :key="session.id" class="nav-session" :class="{ selected: route.path === '/chat' && currentSessionId === session.id }">
        <button class="session-title" :title="session.title" @click="emit('navigate', `/chat?session=${session.id}`)">{{ session.title || t('chat.newSession') }}</button>
        <a-dropdown :trigger="['click']"><button class="session-more" :aria-label="t('chat.sessionActions')"><MoreOutlined /></button>
          <template #overlay><a-menu>
            <a-menu-item @click="editing = session.id; title = session.title">{{ t('chat.renameSession') }}</a-menu-item>
            <a-menu-item danger><a-popconfirm :title="t('chat.deleteConfirm')" @confirm="remove(session.id)">{{ t('chat.deleteSession') }}</a-popconfirm></a-menu-item>
          </a-menu></template>
        </a-dropdown>
      </div>
          <button v-if="!group.sessions.length && group.available" class="folder-empty" @click="start(group.code)">{{ t('chat.startAgentSession') }}</button>
        </div>
      </section>
      <div v-if="error" class="nav-empty" role="alert">{{ error }}</div>
      <div v-else-if="loading" class="nav-empty" role="status">{{ t('chat.searching') }}</div>
      <div v-else-if="!groups.length" class="nav-empty">{{ keyword ? t('chat.noSearchResults') : t('chat.noAgents') }}</div>
      <button v-if="sessions.length < total" class="nav-load" :disabled="loading" @click="fetchSessions(keyword, true)">{{ t('chat.loadMore') }}</button>
    </template>
    <a-modal :open="!!editing" :title="t('chat.renameSession')" @ok="rename" @cancel="editing = null"><a-input v-model:value="title" :maxlength="128" @pressEnter="rename" /></a-modal>
  </section>
</template>

<style scoped>
.chat-navigation { border-top: 1px solid var(--color-sidebar-border); margin-top: 12px; padding: 14px 0; color: var(--color-text-sidebar); }
button { font: inherit; color: inherit; cursor: pointer; }
.nav-new,.nav-agent { display:flex; align-items:center; gap:10px; width:100%; border:0; border-radius:9px; padding:10px 12px; background:transparent; text-align:left; }
.nav-new { background:var(--color-bg-sidebar-hover); font-weight:500; }
.nav-caption { font-size:11px; color:var(--color-text-tertiary); padding:12px 12px 8px; letter-spacing:.05em; }
.nav-agent { font-size:13px; min-width:0; padding:9px 6px; gap:8px; }
.nav-agent span { min-width:0; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; }
.folder-heading { display:flex; align-items:center; border-radius:8px; }
.folder-heading:hover,.nav-session:hover { background:var(--color-bg-sidebar-hover); }
.folder-chevron { font-size:9px; color:var(--color-text-tertiary); transition:transform .15s; }.folder-chevron.closed { transform:rotate(-90deg); }
.new-icon { flex-shrink:0; background:none; border:0; padding:8px; opacity:0; color:var(--color-text-secondary); }
.folder-heading:hover .new-icon,.new-icon:focus-visible { opacity:1; }
.folder-sessions { margin:2px 0 10px 23px; padding-left:4px; border-left:1px solid var(--color-sidebar-border); }
.folder-empty { display:block; border:0; background:none; color:var(--color-text-tertiary); padding:8px 10px; font-size:12px; }
.selected { background:var(--color-bg-sidebar-active)!important; color:var(--color-text-sidebar-active); }
.nav-session { display:flex; align-items:center; border-radius:8px; padding:0 6px 0 12px; min-height:35px; }
.session-title { flex:1; min-width:0; text-align:left; white-space:nowrap; overflow:hidden; text-overflow:ellipsis; font-size:12px; padding:8px 0; }
.session-title,.session-more { background:transparent; border:0; }
.session-more { padding:6px; opacity:0; }
.nav-session:hover .session-more,.session-more:focus-visible { opacity:1; }
.nav-search { display:flex; align-items:center; gap:8px; margin-top:12px; padding:8px 12px; font-size:12px; border:1px solid transparent; border-radius:8px; }.nav-search:focus-within { border-color:var(--color-border); background:var(--color-bg-sidebar-hover); }
.nav-search input { width:100%; min-width:0; outline:none; border:0; background:transparent; color:inherit; }
.nav-empty,.nav-load { font-size:12px; color:var(--color-text-tertiary); padding:10px 12px; }
.nav-load { border:0; background:none; }
.compact .nav-new { justify-content:center; padding:10px; }
@media (hover:none) { .session-more,.new-icon { opacity:1; } }
</style>
