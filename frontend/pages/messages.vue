<script setup lang="ts">
/**
 * 消息中心页面 — 参照审核工作台风格。
 * 顶部统计卡片 + 左右分栏（左侧消息列表含未读/已读页签，右侧消息详情）。
 */
import {
  CheckOutlined,
  MailOutlined,
  CheckCircleOutlined,
  InboxOutlined,
} from '@ant-design/icons-vue'
import type { UserNotificationItem } from '~/types/user-notifications'
import { extractScore, scoreColor } from '~/utils/scoreColor'

definePageMeta({ middleware: 'auth' })

const route = useRoute()
const { items, unreadCount, listLoading, refreshList, markOneRead, markAllRead, formatRelative } = useNotifications()
const { t, te } = useI18n()

/** 当前页签：unread / read */
const activeTab = ref<'unread' | 'read'>('unread')

/** 当前选中消息 ID */
const selectedId = ref<string | null>(null)

/** 按页签过滤后的消息列表 */
const filteredItems = computed(() =>
  activeTab.value === 'unread'
    ? items.value.filter(i => !i.read)
    : items.value.filter(i => i.read),
)

/** 已读消息数 */
const readCount = computed(() => items.value.filter(i => i.read).length)

/** 当前选中消息对象 */
const selectedItem = computed(() => items.value.find(i => i.id === selectedId.value) ?? null)

/** 将通知分类 key 转换为本地化标签 */
function categoryLabel(cat: string) {
  const key = `notifications.category.${cat}`
  return te(key) ? t(key) : cat
}

/** 将 body 中的「评分 {数字}」渲染为带颜色的 HTML */
function renderBodyWithScore(body: string | undefined): string {
  if (!body) return ''
  const score = extractScore(body)
  if (score === null) return body
  const color = scoreColor(score)
  return body.replace(/评分\s*(\d+)/, `<span style="color:${color};font-weight:600">评分 $1</span>`)
}

/** 点击消息条目 */
async function onSelectMessage(item: UserNotificationItem) {
  selectedId.value = item.id
  if (!item.read) await markOneRead(item.id)
}

/** 全部已读 */
async function onMarkAllRead() {
  await markAllRead()
}

/** 切换页签 */
function switchTab(tab: 'unread' | 'read') {
  activeTab.value = tab
  selectedId.value = null
}

// 页面加载时拉取列表，处理 URL query 参数
onMounted(async () => {
  await refreshList()
  const queryId = route.query.id as string | undefined
  if (queryId) {
    selectedId.value = queryId
    const item = items.value.find(i => i.id === queryId)
    if (item) {
      activeTab.value = item.read ? 'read' : 'unread'
      if (!item.read) await markOneRead(item.id)
    }
  }
})
</script>

<template>
  <div class="messages-page">
    <!-- 页面标题 -->
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('messages.title') }}</h1>
        <p class="page-subtitle">{{ t('messages.subtitle', `${unreadCount}`) }}</p>
      </div>
      <button
        v-if="unreadCount > 0"
        class="mark-all-read-btn"
        @click="onMarkAllRead"
      >
        <CheckOutlined />
        <span>{{ t('messages.markAllRead') }}</span>
      </button>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-row">
      <div
        class="stat-card stat-card--primary"
        :class="{ 'stat-card--selected': activeTab === 'unread' }"
        @click="switchTab('unread')"
      >
        <div class="stat-card-icon"><MailOutlined /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ unreadCount }}</span>
          <span class="stat-card-label">{{ t('messages.unread') }}</span>
        </div>
      </div>
      <div
        class="stat-card stat-card--success"
        :class="{ 'stat-card--selected': activeTab === 'read' }"
        @click="switchTab('read')"
      >
        <div class="stat-card-icon"><CheckCircleOutlined /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ readCount }}</span>
          <span class="stat-card-label">{{ t('messages.read') }}</span>
        </div>
      </div>
    </div>

    <!-- 主体：左右分栏 -->
    <div class="messages-grid">
      <!-- 左侧消息列表 -->
      <div class="list-panel">
        <div class="panel-header">
          <h3 class="panel-title">
            <MailOutlined v-if="activeTab === 'unread'" style="color: var(--color-primary)" />
            <CheckCircleOutlined v-else style="color: var(--color-success)" />
            {{ activeTab === 'unread' ? t('messages.unread') : t('messages.read') }}
            <a-badge
              :count="activeTab === 'unread' ? unreadCount : readCount"
              :number-style="{ backgroundColor: activeTab === 'unread' ? 'var(--color-primary)' : 'var(--color-success)' }"
            />
          </h3>
        </div>
        <div class="panel-body">
          <a-spin :spinning="listLoading" style="min-height: 100px">
            <div v-if="!filteredItems.length && !listLoading" class="list-empty">
              <InboxOutlined style="font-size: 28px; margin-bottom: 8px" />
              <span>{{ t('messages.noMessages') }}</span>
            </div>
            <div
              v-for="item in filteredItems"
              :key="item.id"
              class="message-item"
              :class="{
                'message-item--selected': item.id === selectedId,
                'message-item--unread': !item.read,
              }"
              @click="onSelectMessage(item)"
            >
              <span v-if="!item.read" class="unread-dot" />
              <div class="message-item-content">
                <div class="message-item-top">
                  <span class="message-item-category">{{ categoryLabel(item.category) }}</span>
                  <span class="message-item-time">{{ formatRelative(item.created_at) }}</span>
                </div>
                <div class="message-item-title" :class="{ 'message-item-title--unread': !item.read }">
                  {{ item.title }}
                </div>
                <div v-if="item.body" class="message-item-body" v-html="renderBodyWithScore(item.body)" />
              </div>
            </div>
          </a-spin>
        </div>
      </div>

      <!-- 右侧消息详情 -->
      <div class="detail-panel">
        <template v-if="selectedItem">
          <div class="detail-header">
            <h3 class="detail-title">{{ selectedItem.title }}</h3>
            <div class="detail-meta">
              <span class="detail-category">{{ categoryLabel(selectedItem.category) }}</span>
              <span class="detail-separator">·</span>
              <span class="detail-time">{{ formatRelative(selectedItem.created_at) }}</span>
            </div>
          </div>
          <div class="detail-body" v-html="renderBodyWithScore(selectedItem.body)" />
        </template>
        <div v-else class="detail-empty">
          <InboxOutlined style="font-size: 36px; margin-bottom: 12px" />
          <span>{{ t('messages.emptyDetail') }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.messages-page {
  animation: fadeIn 0.3s ease;
}
@keyframes fadeIn { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

/* 页面标题 — 与审核工作台一致 */
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 24px;
}
.page-title {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin: 0;
  letter-spacing: -0.02em;
}
.page-subtitle {
  font-size: 14px;
  color: var(--color-text-tertiary);
  margin: 4px 0 0;
}
.mark-all-read-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 16px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-card);
  color: var(--color-primary);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
  margin-top: 4px;
}
.mark-all-read-btn:hover {
  background: var(--color-primary-bg);
  border-color: var(--color-primary);
}

/* 统计卡片 — 与审核工作台一致 */
.stats-row {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}
.stat-card {
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  border: 2px solid var(--color-border-light);
  transition: all var(--transition-base);
  cursor: pointer;
  user-select: none;
}
.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}
.stat-card--selected {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 1px var(--color-primary);
}
.stat-card--success.stat-card--selected {
  border-color: var(--color-success);
  box-shadow: 0 0 0 1px var(--color-success);
}
.stat-card-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  flex-shrink: 0;
}
.stat-card--primary .stat-card-icon {
  background: var(--color-primary-bg);
  color: var(--color-primary);
}
.stat-card--success .stat-card-icon {
  background: var(--color-success-bg);
  color: var(--color-success);
}
.stat-card-info {
  display: flex;
  flex-direction: column;
}
.stat-card-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--color-text-primary);
  line-height: 1.2;
}
.stat-card-label {
  font-size: 13px;
  color: var(--color-text-tertiary);
  margin-top: 2px;
}

/* 主体网格 — 与审核工作台一致 */
.messages-grid {
  display: grid;
  grid-template-columns: 380px 1fr;
  gap: 16px;
  min-height: 0;
}

/* 左侧列表面板 */
.list-panel {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-lg);
  display: flex;
  flex-direction: column;
  max-height: calc(100vh - var(--header-height) - 260px);
}
.panel-header {
  padding: 14px 18px;
  border-bottom: 1px solid var(--color-border-light);
  flex-shrink: 0;
}
.panel-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}
.panel-body {
  flex: 1;
  overflow-y: auto;
}

.list-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 16px;
  color: var(--color-text-tertiary);
  font-size: 13px;
}

/* 消息条目 */
.message-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 18px;
  cursor: pointer;
  border-bottom: 1px solid var(--color-border-light);
  transition: background 0.15s ease;
}
.message-item:last-child { border-bottom: none; }
.message-item:hover { background: var(--color-bg-hover); }
.message-item--selected { background: var(--color-primary-bg); }
.message-item--selected:hover { background: var(--color-primary-bg); }

.unread-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--color-primary);
  flex-shrink: 0;
  margin-top: 7px;
}

.message-item-content { flex: 1; min-width: 0; }
.message-item-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
}
.message-item-category {
  font-size: 11px;
  font-weight: 500;
  color: var(--color-primary);
  background: var(--color-primary-bg);
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  line-height: 1.6;
}
.message-item-time {
  font-size: 11px;
  color: var(--color-text-tertiary);
  flex-shrink: 0;
}
.message-item-title {
  font-size: 13px;
  color: var(--color-text-primary);
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-bottom: 3px;
}
.message-item-title--unread { font-weight: 600; }
.message-item-body {
  font-size: 12px;
  color: var(--color-text-secondary);
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 右侧详情面板 */
.detail-panel {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-lg);
  padding: 24px 28px;
  max-height: calc(100vh - var(--header-height) - 260px);
  overflow-y: auto;
}

.detail-header {
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--color-border-light);
}
.detail-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0 0 8px;
}
.detail-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--color-text-secondary);
}
.detail-category { font-weight: 500; color: var(--color-primary); }
.detail-separator { color: var(--color-text-tertiary); }
.detail-time { color: var(--color-text-tertiary); }
.detail-body {
  font-size: 14px;
  color: var(--color-text-primary);
  line-height: 1.7;
}
.detail-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  min-height: 200px;
  color: var(--color-text-tertiary);
  font-size: 14px;
}

@media (max-width: 1024px) {
  .messages-grid { grid-template-columns: 1fr; }
}
@media (max-width: 768px) {
  .stats-row { grid-template-columns: 1fr; gap: 12px; }
  .stat-card { padding: 14px; }
  .stat-card-value { font-size: 22px; }
  .stat-card-icon { width: 40px; height: 40px; font-size: 18px; }
  .page-title { font-size: 20px; }
}
</style>
