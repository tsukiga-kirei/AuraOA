<script setup lang="ts">
import type { ThemeMode } from '~/composables/useTheme'

const { mode, setTheme } = useTheme()
const { t } = useI18n()

const themeOptions: { mode: ThemeMode; labelKey: string }[] = [
  { mode: 'light', labelKey: 'header.lightMode' },
  { mode: 'dark', labelKey: 'header.darkMode' },
  { mode: 'warm', labelKey: 'header.warmMode' },
]

const pick = (next: ThemeMode) => {
  if (mode.value !== next) setTheme(next)
}
</script>

<template>
  <div class="theme-toggle-btn" role="group" :aria-label="t('header.toggleTheme')">
    <span class="theme-toggle-track">
      <button
        v-for="option in themeOptions"
        :key="option.mode"
        type="button"
        class="theme-toggle-option"
        :class="[
          `theme-toggle-option--${option.mode}`,
          { 'theme-toggle-option--active': mode === option.mode },
        ]"
        :aria-pressed="mode === option.mode"
        :aria-label="t(option.labelKey)"
        :title="t(option.labelKey)"
        @click="pick(option.mode)"
      >
        <!-- Sun (lucide) -->
        <svg
          v-if="option.mode === 'light'"
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2.25"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <circle cx="12" cy="12" r="4" />
          <path d="M12 2v2" />
          <path d="M12 20v2" />
          <path d="m4.93 4.93 1.41 1.41" />
          <path d="m17.66 17.66 1.41 1.41" />
          <path d="M2 12h2" />
          <path d="M20 12h2" />
          <path d="m6.34 17.66-1.41 1.41" />
          <path d="m19.07 4.93-1.41 1.41" />
        </svg>
        <!-- Moon (lucide) -->
        <svg
          v-else-if="option.mode === 'dark'"
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2.25"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z" />
        </svg>
        <!-- Sparkles (lucide) -->
        <svg
          v-else
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2.25"
          stroke-linecap="round"
          stroke-linejoin="round"
          aria-hidden="true"
        >
          <path d="M9.937 15.5A2 2 0 0 0 8.5 14.063l-6.135-1.582a.5.5 0 0 1 0-.962L8.5 9.936A2 2 0 0 0 9.937 8.5l1.582-6.135a.5.5 0 0 1 .963 0L14.063 8.5A2 2 0 0 0 15.5 9.937l6.135 1.581a.5.5 0 0 1 0 .964L15.5 14.063a2 2 0 0 0-1.437 1.437l-1.582 6.135a.5.5 0 0 1-.963 0z" />
          <path d="M20 3v4" />
          <path d="M22 5h-4" />
          <path d="M4 17v2" />
          <path d="M5 18H3" />
        </svg>
      </button>
    </span>
  </div>
</template>
