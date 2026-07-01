<script setup lang="ts">
import {
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  BellOutlined,
  CheckOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'

import type { RoleInfo } from '~/types/auth'
import type { UserNotificationItem } from '~/types/user-notifications'
import { extractScore, scoreColor } from '~/utils/scoreColor'

// props：侧边栏折叠状态 / 是否移动端布局
defineProps<{
  collapsed: boolean
  isMobile: boolean
}>()

// emit：切换侧边栏折叠 / 切换移动端菜单
const emit = defineEmits<{
  (e: 'toggleSidebar'): void
  (e: 'toggleMobileMenu'): void
}>()

const { t, te } = useI18n()
const { allRoles, activeRole, switchRole, getMenu } = useAuth()

const {
  items: notifItems,
  unreadCount,
  listLoading,
  refreshList,
  markOneRead,
  markAllRead,
  formatRelative,
} = useNotifications()

// 通知下拉面板开关，打开时自动刷新列表
const notifOpen = ref(false)
watch(notifOpen, open => {
  if (open) refreshList()
})

// 将通知分类 key 转换为本地化标签，找不到时回退原始值
function categoryLabel(cat: string) {
  const key = `notifications.category.${cat}`
  return te(key) ? t(key) : cat
}

// 点击通知条目：标记已读并跳转到消息中心，选中该消息
async function onNotifItemClick(it: UserNotificationItem) {
  if (!it.read) await markOneRead(it.id)
  notifOpen.value = false
  await navigateTo(`/messages?id=${it.id}`)
}

async function handleMarkAllNotificationsRead() {
  await markAllRead()
}

// 将 body 中的「评分 {数字}」渲染为带颜色的 HTML
function renderBodyWithScore(body: string | undefined): string {
  if (!body) return ''
  const score = extractScore(body)
  if (score === null) return body
  const color = scoreColor(score)
  return body.replace(/评分\s*(\d+)/, `<span style="color:${color};font-weight:600">评分 $1</span>`)
}

// 查看全部消息：关闭面板并跳转到消息页面
function goToMessages() {
  notifOpen.value = false
  navigateTo('/messages')
}

// 按角色类型分组，用于下拉菜单分区展示
const systemRoles = computed(() => allRoles.value.filter(r => r.role === 'system_admin'))
const tenantAdminRoles = computed(() => allRoles.value.filter(r => r.role === 'tenant_admin'))
const businessRoles = computed(() => allRoles.value.filter(r => r.role === 'business'))

// 仅当用户拥有多个角色时才显示切换器
const showRoleSwitcher = computed(() => allRoles.value.length > 1)

// 当前激活角色的 id、显示名称、角色类型
const activeRoleId = computed(() => activeRole.value?.id || '')
const activeRoleLabel = computed(() => activeRole.value?.label || '')
const activeRoleType = computed(() => activeRole.value?.role || 'business')

// 下拉面板可见状态
const dropdownOpen = ref(false)

// 图标动画触发键，每次切换角色时递增以重新触发过渡动画
const iconKey = ref(0)

// 切换角色：相同角色不重复请求，切换成功后刷新菜单并跳转首页
const handleSwitchRole = async (role: RoleInfo) => {
  if (role.id === activeRoleId.value) return
  dropdownOpen.value = false
  iconKey.value++
  const result = await switchRole(role.id)
  if (!result.ok) {
    message.error(result.errorMsg || t('header.switchRoleFailed'))
    return
  }
  await getMenu()
  navigateTo('/overview')
}
</script>

<template>
  <header class="app-header">
    <div class="app-header-left">
      <button
        class="header-toggle"
        @click="isMobile ? emit('toggleMobileMenu') : emit('toggleSidebar')"
      >
        <MenuUnfoldOutlined v-if="collapsed && !isMobile" />
        <MenuFoldOutlined v-else-if="!isMobile" />
        <MenuUnfoldOutlined v-else />
      </button>
    </div>

    <div class="app-header-right">
      <ThemeSwitcher />

      <!--角色切换下拉菜单-->
      <a-dropdown
        v-if="showRoleSwitcher"
        v-model:open="dropdownOpen"
        placement="bottomRight"
        :trigger="['hover']"
      >
        <button
          class="header-action role-switch-btn"
          :class="[
            { 'role-switch-btn--open': dropdownOpen },
            `role-switch-btn--${activeRoleType}`
          ]"
          :title="t('header.switchRole')"
        >
          <transition name="role-icon" mode="out-in">
            <!--业务角色图标：条形图-->
            <svg v-if="activeRoleType === 'business'" :key="'biz-' + iconKey" class="role-icon" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>
            <!--租户管理员图标：滑块-->
            <svg v-else-if="activeRoleType === 'tenant_admin'" :key="'ta-' + iconKey" class="role-icon" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="4" y1="21" x2="4" y2="14"/><line x1="4" y1="10" x2="4" y2="3"/><line x1="12" y1="21" x2="12" y2="12"/><line x1="12" y1="8" x2="12" y2="3"/><line x1="20" y1="21" x2="20" y2="16"/><line x1="20" y1="12" x2="20" y2="3"/><line x1="1" y1="14" x2="7" y2="14"/><line x1="9" y1="8" x2="15" y2="8"/><line x1="17" y1="16" x2="23" y2="16"/></svg>
            <!--系统管理员图标：盾牌-->
            <svg v-else :key="'sa-' + iconKey" class="role-icon" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
          </transition>
          <span class="role-switch-label">{{ activeRoleLabel }}</span>
        </button>
        <template #overlay>
          <div class="role-dropdown">
            <div class="role-dropdown-title">{{ t('header.switchRole') }}</div>

            <!--业务角色-->
            <template v-if="businessRoles.length">
              <div class="role-dropdown-group role-dropdown-group--first">
                <span class="role-dropdown-group-icon role-dropdown-group-icon--business">
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>
                </span>
                <span>{{ t('login.portal.business') }}</span>
              </div>
              <div
                v-for="role in businessRoles"
                :key="role.id"
                class="role-dropdown-item"
                :class="{ 'role-dropdown-item--active': role.id === activeRoleId }"
                @click="handleSwitchRole(role)"
              >
                <div class="role-dropdown-info">
                  <div class="role-dropdown-name">{{ role.tenant_name }}</div>
                </div>
                <CheckOutlined v-if="role.id === activeRoleId" class="role-dropdown-check" />
              </div>
            </template>

            <!--租户管理员角色-->
            <template v-if="tenantAdminRoles.length">
              <div class="role-dropdown-group" :class="{ 'role-dropdown-group--first': !businessRoles.length }">
                <span class="role-dropdown-group-icon role-dropdown-group-icon--tenant">
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="4" y1="21" x2="4" y2="14"/><line x1="4" y1="10" x2="4" y2="3"/><line x1="12" y1="21" x2="12" y2="12"/><line x1="12" y1="8" x2="12" y2="3"/><line x1="20" y1="21" x2="20" y2="16"/><line x1="20" y1="12" x2="20" y2="3"/><line x1="1" y1="14" x2="7" y2="14"/><line x1="9" y1="8" x2="15" y2="8"/><line x1="17" y1="16" x2="23" y2="16"/></svg>
                </span>
                <span>{{ t('login.portal.tenantAdmin') }}</span>
              </div>
              <div
                v-for="role in tenantAdminRoles"
                :key="role.id"
                class="role-dropdown-item"
                :class="{ 'role-dropdown-item--active': role.id === activeRoleId }"
                @click="handleSwitchRole(role)"
              >
                <div class="role-dropdown-info">
                  <div class="role-dropdown-name">{{ role.tenant_name }}</div>
                </div>
                <CheckOutlined v-if="role.id === activeRoleId" class="role-dropdown-check" />
              </div>
            </template>

            <!--系统管理员角色-->
            <template v-if="systemRoles.length">
              <div class="role-dropdown-group" :class="{ 'role-dropdown-group--first': !businessRoles.length && !tenantAdminRoles.length }">
                <span class="role-dropdown-group-icon role-dropdown-group-icon--system">
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
                </span>
                <span>{{ t('login.portal.systemAdmin') }}</span>
              </div>
              <div
                v-for="role in systemRoles"
                :key="role.id"
                class="role-dropdown-item"
                :class="{ 'role-dropdown-item--active': role.id === activeRoleId }"
                @click="handleSwitchRole(role)"
              >
                <div class="role-dropdown-info">
                  <div class="role-dropdown-name">{{ role.label }}</div>
                </div>
                <CheckOutlined v-if="role.id === activeRoleId" class="role-dropdown-check" />
              </div>
            </template>
          </div>
        </template>
      </a-dropdown>

      <a-dropdown
        v-model:open="notifOpen"
        placement="bottomRight"
        :trigger="['click']"
      >
        <a-tooltip :title="t('header.notifications')" placement="bottom" :mouse-enter-delay="0.5">
          <a-badge :count="unreadCount" :overflow-count="99" :show-zero="false" :offset="[-4, 4]">
            <button type="button" class="header-action" :aria-label="t('header.notifications')">
              <BellOutlined />
            </button>
          </a-badge>
        </a-tooltip>
        <template #overlay>
          <div class="notif-panel">
            <div class="notif-panel-head">
              <span class="notif-panel-title">{{ t('header.notificationsTitle') }}</span>
              <button
                v-if="unreadCount > 0"
                type="button"
                class="notif-mark-all"
                @click.stop="handleMarkAllNotificationsRead"
              >
                {{ t('header.notificationsMarkAllRead') }}
              </button>
            </div>
            <a-spin :spinning="listLoading">
              <div v-if="!notifItems.length && !listLoading" class="notif-empty">
                {{ t('header.notificationsEmpty') }}
              </div>
              <ul v-else class="notif-list">
                <li
                  v-for="it in notifItems"
                  :key="it.id"
                  class="notif-item"
                  :class="{ 'notif-item--unread': !it.read }"
                  role="button"
                  tabindex="0"
                  @click="onNotifItemClick(it)"
                  @keydown.enter.prevent="onNotifItemClick(it)"
                >
                  <div class="notif-item-top">
                    <span class="notif-cat">{{ categoryLabel(it.category) }}</span>
                    <span class="notif-time">{{ formatRelative(it.created_at) }}</span>
                  </div>
                  <div class="notif-title">{{ it.title }}</div>
                  <div v-if="it.body" class="notif-body" v-html="renderBodyWithScore(it.body)" />
                </li>
              </ul>
            </a-spin>
            <div class="notif-panel-footer">
              <button
                v-if="unreadCount > 0"
                type="button"
                class="notif-footer-action notif-footer-mark-all"
                @click.stop="handleMarkAllNotificationsRead"
              >
                {{ t('header.notificationsMarkAllRead') }}
              </button>
              <span v-if="unreadCount > 0" class="notif-footer-divider">|</span>
              <button type="button" class="notif-footer-action" @click="goToMessages">
                {{ t('header.notificationsViewAll') }}
              </button>
            </div>
          </div>
        </template>
      </a-dropdown>
    </div>
  </header>
</template>

<style scoped>
.app-header {
  height: var(--header-height);
  display: flex; align-items: center; justify-content: space-between;
  padding: 0 24px;
  position: sticky; top: 0; z-index: 50;
  background: var(--color-bg-page);
}
.app-header-left { display: flex; align-items: center; gap: 16px; }
.app-header-right { display: flex; align-items: center; gap: 8px; }

.header-toggle,
.header-action {
  width: 36px; height: 36px;
  border: none; background: transparent;
  border-radius: var(--radius-md);
  cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  font-size: 18px;
  color: var(--color-text-secondary);
  transition: all var(--transition-fast);
  outline: none;
}
.header-toggle:hover,
.header-action:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}
.header-toggle:focus-visible,
.header-action:focus-visible {
  background: var(--color-bg-hover);
  color: var(--color-primary);
  box-shadow: 0 0 0 2px var(--color-primary-bg), 0 0 0 4px var(--color-primary-ring);
}

/*===== 角色切换按钮 =====*/
.role-switch-btn {
  width: auto !important;
  padding: 0 8px !important;
  gap: 0;
  font-size: 13px;
  border: 1px solid var(--color-border) !important;
  border-radius: 20px !important;
  height: 32px !important;
  background: var(--color-bg-card) !important;
  overflow: hidden;
  transition: padding 0.3s cubic-bezier(0.4, 0, 0.2, 1),
              gap 0.3s cubic-bezier(0.4, 0, 0.2, 1),
              border-color 0.2s ease,
              color 0.2s ease,
              background 0.2s ease !important;
}

/*各角色类型对应的强调色*/
.role-switch-btn--business { color: var(--color-role-business); }
.role-switch-btn--tenant_admin { color: var(--color-role-tenant); }
.role-switch-btn--system_admin { color: var(--color-role-system); }

.role-switch-btn:hover,
.role-switch-btn--open {
  padding: 0 12px !important;
  gap: 6px;
  border-color: var(--color-primary) !important;
  color: var(--color-primary) !important;
  background: var(--color-primary-bg) !important;
}
.role-switch-btn:hover .role-switch-label,
.role-switch-btn--open .role-switch-label {
  max-width: 160px;
  opacity: 1;
  transform: translateX(0);
}

.role-icon {
  flex-shrink: 0;
  display: block;
}

.role-switch-label {
  font-size: 12px;
  font-weight: 500;
  max-width: 0;
  opacity: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  transform: translateX(-6px);
  transition: max-width 0.3s cubic-bezier(0.4, 0, 0.2, 1),
              opacity 0.25s ease,
              transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

/*角色图标切换过渡动画*/
.role-icon-enter-active,
.role-icon-leave-active {
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}
.role-icon-enter-from {
  opacity: 0;
  transform: rotate(-180deg) scale(0.4);
}
.role-icon-leave-to {
  opacity: 0;
  transform: rotate(180deg) scale(0.4);
}

/*===== 角色下拉面板 =====*/
.role-dropdown {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  padding: 8px;
  min-width: 260px;
  max-width: 320px;
}
.role-dropdown-title {
  padding: 8px 12px 4px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: var(--color-text-tertiary);
}

/*分组标题*/
.role-dropdown-group {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 12px 4px;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-secondary);
  border-top: 1px solid var(--color-border-light);
  margin-top: 4px;
}
.role-dropdown-group--first {
  border-top: none;
  margin-top: 0;
}

/*分组图标按角色类型着色*/
.role-dropdown-group-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 6px;
  flex-shrink: 0;
}
.role-dropdown-group-icon--business {
  color: var(--color-role-business);
  background: color-mix(in srgb, var(--color-role-business) 12%, transparent);
}
.role-dropdown-group-icon--tenant {
  color: var(--color-role-tenant);
  background: color-mix(in srgb, var(--color-role-tenant) 12%, transparent);
}
.role-dropdown-group-icon--system {
  color: var(--color-role-system);
  background: color-mix(in srgb, var(--color-role-system) 12%, transparent);
}

/*单个角色条目*/
.role-dropdown-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px 8px 28px;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all 0.15s ease;
}
.role-dropdown-item:hover {
  background: var(--color-bg-hover);
}
.role-dropdown-item--active {
  background: var(--color-primary-bg);
}
.role-dropdown-item--active:hover {
  background: var(--color-primary-bg);
}
.role-dropdown-info {
  flex: 1;
  min-width: 0;
}
.role-dropdown-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-primary);
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.role-dropdown-desc {
  font-size: 11px;
  color: var(--color-text-tertiary);
  line-height: 1.3;
  margin-top: 1px;
}
.role-dropdown-item--active .role-dropdown-name {
  color: var(--color-primary);
}
.role-dropdown-check {
  color: var(--color-primary);
  font-size: 14px;
  flex-shrink: 0;
}

@media (max-width: 768px) {
  .app-header { padding: 0 16px; }
  .role-dropdown { min-width: 220px; }
}
</style>

<style scoped>
.notif-panel {
  width: min(360px, calc(100vw - 32px));
  max-height: 420px;
  display: flex;
  flex-direction: column;
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  overflow: hidden;
}
.notif-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 12px 14px 8px;
  border-bottom: 1px solid var(--color-border-light);
}
.notif-panel-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}
.notif-mark-all {
  border: none;
  background: none;
  padding: 4px 0;
  font-size: 12px;
  color: var(--color-primary);
  cursor: pointer;
}
.notif-mark-all:hover {
  text-decoration: underline;
}
.notif-empty {
  padding: 28px 16px;
  text-align: center;
  font-size: 13px;
  color: var(--color-text-tertiary);
}
.notif-list {
  list-style: none;
  margin: 0;
  padding: 0;
  max-height: 300px;
  overflow-y: auto;
}
.notif-item {
  padding: 10px 14px;
  border-bottom: 1px solid var(--color-border-light);
  cursor: pointer;
  transition: background 0.15s ease;
}
.notif-item:last-child {
  border-bottom: none;
}
.notif-item:hover {
  background: var(--color-bg-hover);
}
.notif-item--unread {
  background: var(--color-primary-bg);
}
.notif-item--unread:hover {
  background: var(--color-primary-bg);
  filter: brightness(0.97);
}
.notif-item-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 4px;
}
.notif-cat {
  font-size: 11px;
  font-weight: 600;
  color: var(--color-primary);
  text-transform: none;
}
.notif-time {
  font-size: 11px;
  color: var(--color-text-tertiary);
  flex-shrink: 0;
}
.notif-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-primary);
  line-height: 1.4;
}
.notif-body {
  margin-top: 4px;
  font-size: 12px;
  color: var(--color-text-secondary);
  line-height: 1.45;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.notif-panel-footer {
  padding: 10px 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  border-top: 1px solid var(--color-border-light);
}
.notif-footer-action {
  border: none;
  background: none;
  padding: 0;
  font-size: 13px;
  font-weight: 500;
  color: var(--color-primary);
  cursor: pointer;
  transition: opacity 0.15s ease;
}
.notif-footer-action:hover {
  opacity: 0.8;
}
.notif-footer-divider {
  color: var(--color-border);
  font-size: 12px;
}
</style>