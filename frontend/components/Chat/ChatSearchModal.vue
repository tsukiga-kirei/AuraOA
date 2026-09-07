<script setup lang="ts">
import { SearchOutlined, CloseOutlined, RobotOutlined, ToolOutlined, MessageOutlined } from '@ant-design/icons-vue'
import type { ChatSessionItem, EffectiveAgentItem } from '~/types/chat'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{
  close: []
  navigate: [path: string]
  start: [agentCode: string]
}>()

const { t, te } = useI18n()
const { authFetch } = useAuth()
const { effectiveAgents } = useChatSession()

type Scope = 'agents' | 'tools' | 'conversations'
const scopes: { value: Scope; icon: typeof RobotOutlined }[] = [
  { value: 'agents', icon: RobotOutlined },
  { value: 'tools', icon: ToolOutlined },
  { value: 'conversations', icon: MessageOutlined },
]

const query = ref('')
const activeScopes = ref<Scope[]>(['agents', 'tools', 'conversations'])
const loading = ref(false)
const conversations = ref<ChatSessionItem[]>([])
const conversationTotal = ref(0)
const page = ref(1)
const searchInput = ref<HTMLInputElement | null>(null)
let requestId = 0
let timer: ReturnType<typeof setTimeout> | undefined

const toggleScope = (scope: Scope) => {
  page.value = 1
  activeScopes.value = activeScopes.value.includes(scope)
    ? activeScopes.value.filter(item => item !== scope)
    : [...activeScopes.value, scope]
}

const toolLabel = (code: string) => te(`chat.tools.${code}`) ? t(`chat.tools.${code}`) : code.replace(/^(skill:|mcp:)/, '').replace(/:/g, ' / ')
const toolKind = (code: string) => code.startsWith('skill:') ? 'skill' : code.startsWith('mcp:') ? 'mcp' : 'system'

const toolItems = computed(() => {
  const keyword = query.value.trim().toLowerCase()
  const seen = new Map<string, { code: string; name: string; agents: string[] }>()
  for (const agent of effectiveAgents.value) {
    for (const code of agent.tool_codes || []) {
      const name = toolLabel(code)
      if (keyword && !code.toLowerCase().includes(keyword) && !name.toLowerCase().includes(keyword) && !agent.name.toLowerCase().includes(keyword)) continue
      const current = seen.get(code) || { code, name, agents: [] }
      current.agents.push(agent.name)
      seen.set(code, current)
    }
  }
  return [...seen.values()]
})

const agentItems = computed(() => {
  const keyword = query.value.trim().toLowerCase()
  return effectiveAgents.value.filter(agent => {
    if (!keyword) return true
    return [agent.name, agent.agent_code, agent.description].some(text => (text || '').toLowerCase().includes(keyword))
  })
})

const searchConversations = async (append = false) => {
  if (!activeScopes.value.includes('conversations')) {
    conversations.value = []
    conversationTotal.value = 0
    return
  }
  const id = ++requestId
  loading.value = true
  try {
    const nextPage = append ? page.value + 1 : 1
    const params = new URLSearchParams({ page: String(nextPage), page_size: '20', keyword: query.value.trim() })
    const data = await authFetch<{ items: ChatSessionItem[]; total: number }>(`/api/chat/sessions?${params}`)
    if (id !== requestId) return
    conversations.value = append ? [...conversations.value, ...(data.items || [])] : data.items || []
    conversationTotal.value = data.total
    page.value = nextPage
  } finally {
    if (id === requestId) loading.value = false
  }
}

watch([query, activeScopes, () => props.open], () => {
  if (!props.open) return
  clearTimeout(timer)
  timer = setTimeout(() => { page.value = 1; void searchConversations() }, query.value.trim() ? 220 : 0)
})

watch(() => props.open, open => {
  if (!open) return
  query.value = ''
  activeScopes.value = ['agents', 'tools', 'conversations']
  page.value = 1
  nextTick(() => searchInput.value?.focus())
})

onBeforeUnmount(() => {
  clearTimeout(timer)
  document.removeEventListener('keydown', onKey)
})

const onKey = (event: KeyboardEvent) => {
  if (event.key === 'Escape') emit('close')
}

watch(() => props.open, open => {
  if (open) document.addEventListener('keydown', onKey)
  else document.removeEventListener('keydown', onKey)
})

const pickAgent = (agent: EffectiveAgentItem) => {
  emit('start', agent.agent_code)
  emit('close')
}

const pickTool = (code: string) => {
  const agent = effectiveAgents.value.find(item => (item.tool_codes || []).includes(code))
  if (agent) emit('start', agent.agent_code)
  emit('close')
}

const pickConversation = (session: ChatSessionItem) => {
  emit('navigate', `/chat?session=${session.id}`)
  emit('close')
}

const empty = computed(() => {
  const hasAgents = activeScopes.value.includes('agents') && agentItems.value.length
  const hasTools = activeScopes.value.includes('tools') && toolItems.value.length
  const hasConversations = activeScopes.value.includes('conversations') && conversations.value.length
  return !hasAgents && !hasTools && !hasConversations && !loading.value
})
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="search-backdrop" @mousedown.self="emit('close')">
      <section class="search-modal" role="dialog" :aria-label="t('chat.search.title')">
        <header class="search-header">
          <div>
            <p class="search-kicker">{{ t('chat.search.kicker') }}</p>
            <h2>{{ t('chat.search.title') }}</h2>
          </div>
          <button type="button" class="search-close" :aria-label="t('common.close')" @click="emit('close')"><CloseOutlined /></button>
        </header>
        <div class="search-content">
          <div class="search-box">
            <SearchOutlined />
            <input ref="searchInput" v-model="query" :placeholder="t('chat.search.placeholder')" :aria-label="t('chat.search.placeholder')" />
          </div>
          <fieldset class="search-scopes">
            <legend>{{ t('chat.search.scope') }}</legend>
            <label v-for="scope in scopes" :key="scope.value" :class="{ 'is-checked': activeScopes.includes(scope.value) }">
              <input type="checkbox" :checked="activeScopes.includes(scope.value)" @change="toggleScope(scope.value)" />
              <component :is="scope.icon" />
              <span>{{ t(`chat.search.scope.${scope.value}`) }}</span>
            </label>
          </fieldset>

          <div v-if="empty" class="search-empty">{{ query ? t('chat.noSearchResults') : t('chat.search.empty') }}</div>
          <p v-else-if="loading && !conversations.length" class="search-empty">{{ t('chat.searching') }}</p>

          <section v-if="activeScopes.includes('agents') && agentItems.length" class="search-group">
            <h3>{{ t('chat.search.scope.agents') }}</h3>
            <button v-for="agent in agentItems" :key="agent.id" type="button" class="search-row" @click="pickAgent(agent)">
              <RobotOutlined />
              <span>
                <strong>{{ agent.name }}</strong>
                <small>{{ agent.description || agent.agent_code }}</small>
              </span>
            </button>
          </section>

          <section v-if="activeScopes.includes('tools') && toolItems.length" class="search-group">
            <h3>{{ t('chat.search.scope.tools') }}</h3>
            <button v-for="tool in toolItems" :key="tool.code" type="button" class="search-row" @click="pickTool(tool.code)">
              <ToolOutlined />
              <span>
                <strong>{{ tool.name }}</strong>
                <small>{{ t(`chat.toolKind.${toolKind(tool.code)}`) }} · {{ tool.agents.join(' / ') }}</small>
              </span>
            </button>
          </section>

          <section v-if="activeScopes.includes('conversations') && conversations.length" class="search-group">
            <h3>{{ t('chat.search.scope.conversations') }}</h3>
            <button v-for="session in conversations" :key="session.id" type="button" class="search-row" @click="pickConversation(session)">
              <MessageOutlined />
              <span>
                <strong>{{ session.title || t('chat.newSession') }}</strong>
                <small>{{ session.agent_name || session.agent_code }}</small>
              </span>
            </button>
            <button v-if="conversations.length < conversationTotal" type="button" class="search-more" :disabled="loading" @click="searchConversations(true)">{{ t('chat.loadMore') }}</button>
          </section>
        </div>
      </section>
    </div>
  </Teleport>
</template>

<style scoped>
.search-backdrop {
  position: fixed; inset: 0; z-index: 180;
  display: grid; place-items: center; padding: 24px;
  background: rgba(15, 23, 42, 0.42); backdrop-filter: blur(6px);
}
.search-modal {
  width: min(760px, 100%);
  height: min(680px, calc(100vh - 48px));
  display: grid; grid-template-rows: auto minmax(0, 1fr);
  border-radius: 18px;
  background: var(--color-bg-card);
  box-shadow: 0 24px 80px rgba(15, 23, 42, 0.24);
  overflow: hidden;
}
.search-header {
  display: flex; align-items: flex-start; justify-content: space-between;
  padding: 20px 22px 12px; border-bottom: 1px solid var(--color-border-light);
}
.search-kicker { margin: 0 0 4px; font-size: 11px; color: var(--color-text-tertiary); letter-spacing: 0.06em; }
h2 { margin: 0; font-size: 18px; font-weight: 650; color: var(--color-text-primary); }
.search-close {
  width: 32px; height: 32px; border: 0; border-radius: 8px;
  background: var(--color-bg-hover); color: var(--color-text-secondary); cursor: pointer;
}
.search-content { overflow: auto; padding: 16px 22px 24px; }
.search-box {
  display: grid; grid-template-columns: 20px 1fr; align-items: center; gap: 10px;
  min-height: 44px; padding: 0 14px; border: 1px solid var(--color-border);
  border-radius: 10px; color: var(--color-text-secondary);
}
.search-box:focus-within { border-color: var(--color-primary); box-shadow: 0 0 0 3px var(--color-primary-ring, rgba(37, 99, 235, 0.12)); }
.search-box input { border: 0; outline: none; background: transparent; color: var(--color-text-primary); min-width: 0; }
.search-scopes { margin: 14px 0 8px; padding: 0; border: 0; display: flex; flex-wrap: wrap; gap: 8px; }
.search-scopes legend { width: 100%; margin-bottom: 8px; font-size: 12px; color: var(--color-text-tertiary); }
.search-scopes label {
  display: inline-flex; align-items: center; gap: 6px; min-height: 30px; padding: 0 10px;
  border: 1px solid var(--color-border); border-radius: 999px; font-size: 12px; cursor: pointer;
}
.search-scopes label.is-checked { border-color: var(--color-primary); background: var(--color-primary-bg); color: var(--color-primary); }
.search-scopes input { position: absolute; opacity: 0; pointer-events: none; }
.search-group { margin-top: 18px; }
.search-group h3 { margin: 0 0 8px; font-size: 12px; color: var(--color-text-tertiary); font-weight: 600; }
.search-row {
  width: 100%; display: flex; align-items: flex-start; gap: 10px; text-align: left;
  border: 0; background: transparent; border-radius: 10px; padding: 10px 8px; cursor: pointer; color: inherit;
}
.search-row:hover { background: var(--color-bg-hover); }
.search-row span { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.search-row strong { font-size: 13px; font-weight: 600; color: var(--color-text-primary); }
.search-row small { font-size: 12px; color: var(--color-text-tertiary); }
.search-empty, .search-more { font-size: 13px; color: var(--color-text-tertiary); padding: 18px 8px; }
.search-more { display: block; width: 100%; border: 0; background: none; cursor: pointer; }
@media (max-width: 620px) {
  .search-backdrop { align-items: end; padding: 0; }
  .search-modal { width: 100%; height: min(88vh, 720px); border-radius: 18px 18px 0 0; }
}
</style>
