<script setup lang="ts">
import type { QueryEvidence } from '~/types/audit-data-query'
defineProps<{ items: QueryEvidence[] }>()
const { t } = useI18n()
</script>
<template>
  <a-collapse v-if="items.length" class="query-evidence-list">
    <a-collapse-panel key="queries" :header="t('query.ruleTitle')">
      <a-collapse>
        <a-collapse-panel
          v-for="(item, index) in items"
          :key="index"
          :header="item.name + ' · v' + item.version_no"
        >
          <p class="query-evidence-rule">{{ item.rule_content }}</p>
          <QueryEvidencePreview :evidence="item" />
        </a-collapse-panel>
      </a-collapse>
    </a-collapse-panel>
  </a-collapse>
</template>
<style scoped>
.query-evidence-list {
  margin: 16px 0;
}
.query-evidence-rule {
  color: var(--color-text-secondary);
  font-size: 13px;
  line-height: 1.7;
}
</style>
