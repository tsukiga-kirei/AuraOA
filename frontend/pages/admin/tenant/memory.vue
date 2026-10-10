<script setup lang="ts">
import { DatabaseOutlined, CommentOutlined } from '@ant-design/icons-vue'
definePageMeta({ middleware: 'auth', layout: 'default' })
const { t } = useI18n()
const activeTab = ref('data')
const tabs = computed(() => [
  { key: 'data', label: t('query.title'), icon: DatabaseOutlined },
  { key: 'review', label: t('memory.review'), icon: CommentOutlined },
])
</script>
<template>
  <div class="tenant-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('memory.title') }}</h1>
        <p class="page-subtitle">{{ t('memory.subtitle') }}</p>
      </div>
    </div>
    <!-- 沿用规则配置页的顶级胶囊导航；切换页签保留未保存的查询草稿。 -->
    <div class="top-tab-nav" role="tablist" :aria-label="t('memory.title')">
      <button
        v-for="tab in tabs"
        :id="`memory-tab-${tab.key}`"
        :key="tab.key"
        type="button"
        role="tab"
        :aria-selected="activeTab === tab.key"
        :aria-controls="`memory-panel-${tab.key}`"
        class="top-tab-btn"
        :class="{ 'top-tab-btn--active': activeTab === tab.key }"
        @click="activeTab = tab.key"
      >
        <component :is="tab.icon" />
        {{ tab.label }}
      </button>
    </div>
    <div
      id="memory-panel-data"
      v-show="activeTab === 'data'"
      role="tabpanel"
      aria-labelledby="memory-tab-data"
    >
      <AuditDataQueryWorkspace embedded />
    </div>
    <div
      id="memory-panel-review"
      v-show="activeTab === 'review'"
      role="tabpanel"
      aria-labelledby="memory-tab-review"
    >
      <div class="review-placeholder">
        <CommentOutlined />
        <h2>{{ t('memory.review') }}</h2>
        <p>{{ t('memory.reviewHint') }}</p>
      </div>
    </div>
  </div>
</template>
<style scoped>
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 24px;
}
.page-title {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin: 0;
}
.page-subtitle {
  font-size: 14px;
  color: var(--color-text-tertiary);
  margin: 4px 0 0;
}
.top-tab-nav {
  display: flex;
  gap: 4px;
  background: var(--color-bg-hover);
  padding: 4px;
  border-radius: var(--radius-lg);
  margin-bottom: 24px;
  width: fit-content;
  max-width: 100%;
  overflow-x: auto;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
}
.top-tab-nav::-webkit-scrollbar {
  display: none;
}
.top-tab-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 20px;
  border: none;
  background: transparent;
  border-radius: var(--radius-md);
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-secondary);
  cursor: pointer;
  white-space: nowrap;
  flex-shrink: 0;
  transition: all var(--transition-fast);
}
.top-tab-btn:hover {
  color: var(--color-text-primary);
}
.top-tab-btn--active {
  background: var(--color-bg-card);
  color: var(--color-primary);
  font-weight: 600;
  box-shadow: var(--shadow-xs);
}
.top-tab-btn:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}
.review-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  min-height: 350px;
  padding: 24px;
  text-align: center;
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-lg);
}
.review-placeholder > .anticon {
  font-size: 40px;
  color: var(--color-text-tertiary);
}
.review-placeholder h2 {
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
}
.review-placeholder p {
  color: var(--color-text-tertiary);
  font-size: 14px;
  margin: 0;
}
@media (max-width: 760px) {
  .page-header {
    flex-direction: column;
  }
  .page-title {
    font-size: 20px;
  }
}
</style>
