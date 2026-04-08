<script setup lang="ts">
import { useI18n } from '~/composables/useI18n'
import { usePagination } from '~/composables/usePagination'
import {
  TeamOutlined,
  UserOutlined,
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  SafetyCertificateOutlined,
  ApartmentOutlined,
  CheckOutlined,
  StopOutlined,
  KeyOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type { Department, OrgRole, OrgMember } from '~/types/org'
import { useOrgApi } from '~/composables/useOrgApi'

definePageMeta({ middleware: 'auth', layout: 'default' })

const { t } = useI18n()
const {
  departments, roles, members, loadAll,
  createDepartment, updateDepartment, deleteDepartment: apiDeleteDept,
  createRole, updateRole, deleteRole: apiDeleteRole,
  createMember, updateMember, deleteMember: apiDeleteMember,
} = useOrgApi()

const { currentUser, refreshRoles } = useAuth()

onMounted(() => {
  loadAll()
})

//顶级选项卡
const topTab = ref<'members' | 'roles' | 'departments'>('members')

//=====成员=====
const memberSearch = ref('')
const memberDeptFilter = ref<string | undefined>(undefined)
const memberRoleFilter = ref<string | undefined>(undefined)

const filteredMembers = computed(() => {
  return members.value.filter(m => {
    if (memberSearch.value && !m.name.includes(memberSearch.value) && !m.username.includes(memberSearch.value)) return false
    if (memberDeptFilter.value && m.department_id !== memberDeptFilter.value) return false
    if (memberRoleFilter.value && !m.role_ids.includes(memberRoleFilter.value)) return false
    return true
  })
})

//成员分页
const { paged: pagedMembers, current: memberPage, pageSize: memberPageSize, total: memberTotal, onChange: onMemberPageChange } = usePagination(filteredMembers, 10)

const showMemberModal = ref(false)
const editingMember = ref<OrgMember | null>(null)
const memberForm = ref({
  name: '', username: '', department_id: '', role_ids: [] as string[], email: '', phone: '', position: '',
})

const openAddMember = () => {
  editingMember.value = null
  memberForm.value = { name: '', username: '', department_id: undefined as any, role_ids: [], email: '', phone: '', position: '' }
  showMemberModal.value = true
}

const openEditMember = (m: OrgMember) => {
  editingMember.value = m
  memberForm.value = {
    name: m.name, username: m.username, department_id: m.department_id,
    role_ids: [...m.role_ids],
    email: m.email, phone: m.phone, position: m.position,
  }
  showMemberModal.value = true
}

/** 解析role_ids以显示名称*/
const resolveRoleNames = (roleIds: string[]): string[] => {
  return roleIds.map(rid => roles.value.find(r => r.id === rid)?.name || rid)
}

const handleSaveMember = async () => {
  if (!memberForm.value.name.trim() || !memberForm.value.username.trim()) {
    message.warning(t('admin.org.fillNameRequired'))
    return
  }
  // 部门和角色必填校验
  if (!memberForm.value.department_id) {
    message.warning(t('admin.org.deptRequired'))
    return
  }
  if (!memberForm.value.role_ids || memberForm.value.role_ids.length === 0) {
    message.warning(t('admin.org.roleRequired'))
    return
  }
  // 用户名校验：只能英文字母、数字、下划线，且以字母开头
  const usernameRegex = /^[a-zA-Z][a-zA-Z0-9_]*$/
  if (!usernameRegex.test(memberForm.value.username)) {
    message.warning(t('admin.org.usernameFormatError'))
    return
  }
  // 邮箱格式校验（如果填写了）
  if (memberForm.value.email.trim()) {
    const emailRegex = /^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$/
    if (!emailRegex.test(memberForm.value.email)) {
      message.warning(t('admin.org.emailFormatError'))
      return
    }
  }
  // 手机号校验：必须为11位数字（如果填写了）
  if (memberForm.value.phone.trim()) {
    const phoneRegex = /^\d{11}$/
    if (!phoneRegex.test(memberForm.value.phone)) {
      message.warning(t('admin.org.phoneFormatError'))
      return
    }
  }
  try {
    if (editingMember.value) {
      const mapped = await updateMember(editingMember.value.id, {
        name: memberForm.value.name,
        username: memberForm.value.username,
        department_id: memberForm.value.department_id,
        role_ids: [...memberForm.value.role_ids],
        email: memberForm.value.email,
        phone: memberForm.value.phone,
        position: memberForm.value.position,
      })
      const idx = members.value.findIndex(m => m.id === editingMember.value!.id)
      if (idx !== -1) members.value[idx] = mapped
      
      // 如果修改的是当前登录用户，刷新角色列表
      if (currentUser.value && editingMember.value.username === currentUser.value.username) {
        refreshRoles()
      }
      
      message.success(t('admin.org.memberUpdated'))
    } else {
      const mapped = await createMember({
        name: memberForm.value.name,
        username: memberForm.value.username,
        department_id: memberForm.value.department_id,
        department_name: departments.value.find(d => d.id === memberForm.value.department_id)?.name || '',
        role_ids: [...memberForm.value.role_ids],
        role_names: resolveRoleNames(memberForm.value.role_ids),
        email: memberForm.value.email,
        phone: memberForm.value.phone,
        position: memberForm.value.position,
        status: 'active',
      })
      members.value.push(mapped)
      message.success(t('admin.org.memberAdded'))
    }
    showMemberModal.value = false
  } catch (e: any) {
    message.error(e.message || t('admin.org.operationFailed'))
  }
}

const toggleMemberStatus = async (m: OrgMember) => {
  const newStatus = m.status === 'active' ? 'disabled' : 'active'
  try {
    const mapped = await updateMember(m.id, { status: newStatus })
    const idx = members.value.findIndex(mem => mem.id === m.id)
    if (idx !== -1) members.value[idx] = mapped
    
    // 如果修改的是当前登录用户，刷新角色列表
    if (currentUser.value && m.username === currentUser.value.username) {
      refreshRoles()
    }
    
    message.success(newStatus === 'active' ? t('admin.org.memberEnabled') : t('admin.org.memberDisabled'))
  } catch (e: any) {
    message.error(e.message || t('admin.org.operationFailed'))
  }
}

const removeMember = async (m: OrgMember) => {
  try {
    await apiDeleteMember(m.id)
    members.value = members.value.filter(mem => mem.id !== m.id)
    
    // 如果删除的是当前登录用户，尝试刷新或登出（通常不建议在管理台删除自己，但需处理逻辑）
    if (currentUser.value && m.username === currentUser.value.username) {
      refreshRoles()
    }
    
    message.success(t('admin.org.memberDeleted'))
  } catch (e: any) {
    message.error(e.message || t('admin.org.operationFailed'))
  }
}

//=====角色=====
const showRoleModal = ref(false)
const editingRole = ref<OrgRole | null>(null)
const roleForm = ref({ name: '', description: '', page_permissions: [] as string[] })

interface PageConfig {
  path: string
  label: string
  group: 'common' | 'business' | 'admin'
  alwaysOn?: boolean
  dependsOn?: string | string[]
}

/**
 * 用于角色权限分配的租户范围页面列表。
 * 仅包含租户范围内的页面 — 无系统管理页面。
 * 分为：普通、业务、管理。*/
const ALL_PAGES_CONFIG: PageConfig[] = [
  { path: '/overview', label: t('admin.org.page.overview'), group: 'common', alwaysOn: true },
  { path: '/dashboard', label: t('admin.org.page.dashboard'), group: 'business' },
  { path: '/cron', label: t('admin.org.page.cron'), group: 'business', dependsOn: ['/dashboard', '/archive'] },
  { path: '/archive', label: t('admin.org.page.archive'), group: 'business' },
  { path: '/settings', label: t('admin.org.page.settings'), group: 'common', alwaysOn: true },
  { path: '/admin/tenant/rules', label: t('admin.org.page.tenantConfig'), group: 'admin' },
  { path: '/admin/tenant/org', label: t('admin.org.page.tenantOrg'), group: 'admin' },
  { path: '/admin/tenant/data', label: t('admin.org.page.tenantData'), group: 'admin' },
  { path: '/admin/tenant/user-configs', label: t('menu.tenant.userConfigs'), group: 'admin' },
]

const allPages = computed<PageConfig[]>(() => ALL_PAGES_CONFIG)

/** 通过依赖强制处理权限复选框*/
const handlePermToggle = (path: string, checked: boolean) => {
  const page = allPages.value.find(p => p.path === path)
  if (!page || page.alwaysOn) return

  if (checked) {
    if (!roleForm.value.page_permissions.includes(path)) {
      roleForm.value.page_permissions.push(path)
    }
    // 如果此页面有依赖项，则自动添加
    if (page.dependsOn) {
      const deps = Array.isArray(page.dependsOn) ? page.dependsOn : [page.dependsOn]
      if (!deps.some(dep => roleForm.value.page_permissions.includes(dep))) {
        roleForm.value.page_permissions.push(deps[0])
        message.info(t('admin.org.depAutoAdded'))
      }
    }
  } else {
    roleForm.value.page_permissions = roleForm.value.page_permissions.filter(p => p !== path)
    // 检查是否有其他页面依赖当前被移除的页面
    allPages.value.forEach(p => {
      if (!p.dependsOn) return
      const deps = Array.isArray(p.dependsOn) ? p.dependsOn : [p.dependsOn]
      
      // 如果当前页面 p 依赖于刚被移除的 path
      if (deps.includes(path)) {
        // 如果 p 所有的候选依赖项都不再被勾选，则连带移除 p
        const hasAnyDep = deps.some(d => roleForm.value.page_permissions.includes(d))
        if (!hasAnyDep && roleForm.value.page_permissions.includes(p.path)) {
          roleForm.value.page_permissions = roleForm.value.page_permissions.filter(perm => perm !== p.path)
        }
      }
    })
  }
}

/** 检查是否应禁用页面复选框*/
const isPermDisabled = (path: string) => {
  const page = allPages.value.find(p => p.path === path)
  if (page?.alwaysOn) return true
  
  // 特殊逻辑：定时任务依赖审核工作台或归档复盘
  if (path === '/cron') {
    return !(roleForm.value.page_permissions.includes('/dashboard') || roleForm.value.page_permissions.includes('/archive'))
  }
  
  return false
}

const openAddRole = () => {
  editingRole.value = null
  roleForm.value = { name: '', description: '', page_permissions: ['/overview', '/dashboard', '/settings'] }
  showRoleModal.value = true
}

const openEditRole = (r: OrgRole) => {
  editingRole.value = r
  roleForm.value = { name: r.name, description: r.description, page_permissions: [...r.page_permissions] }
  showRoleModal.value = true
}

const handleSaveRole = async () => {
  if (!roleForm.value.name.trim()) {
    message.warning(t('admin.org.fillRoleName'))
    return
  }
  try {
    if (editingRole.value) {
      const mapped = await updateRole(editingRole.value.id, {
        name: roleForm.value.name,
        description: roleForm.value.description,
        page_permissions: [...roleForm.value.page_permissions],
      })
      const idx = roles.value.findIndex(r => r.id === editingRole.value!.id)
      if (idx !== -1) roles.value[idx] = mapped
      message.success(t('admin.org.roleUpdated'))
    } else {
      const mapped = await createRole({
        name: roleForm.value.name,
        description: roleForm.value.description,
        page_permissions: [...roleForm.value.page_permissions],
        is_system: false,
      })
      roles.value.push(mapped)
      message.success(t('admin.org.roleAdded'))
    }
    showRoleModal.value = false
  } catch (e: any) {
    message.error(e.message || t('admin.org.operationFailed'))
  }
}

const removeRole = async (r: OrgRole) => {
  if (r.is_system) { message.warning(t('admin.org.systemRoleProtected')); return }
  const usedBy = members.value.filter(m => m.role_ids.includes(r.id))
  if (usedBy.length > 0) { message.warning(t('admin.org.roleHasMembers', [usedBy.length])); return }
  try {
    await apiDeleteRole(r.id)
    roles.value = roles.value.filter(role => role.id !== r.id)
    message.success(t('admin.org.roleDeleted'))
  } catch (e: any) {
    message.error(e.message || t('admin.org.operationFailed'))
  }
}

const getRoleMemberCount = (roleId: string) => members.value.filter(m => m.role_ids?.includes(roleId)).length

//=====部门=====
const showDeptModal = ref(false)
const editingDept = ref<Department | null>(null)
const deptForm = ref({ name: '', manager: '' })

const openAddDept = () => {
  editingDept.value = null
  deptForm.value = { name: '', manager: '' }
  showDeptModal.value = true
}

const openEditDept = (d: Department) => {
  editingDept.value = d
  deptForm.value = { name: d.name, manager: d.manager }
  showDeptModal.value = true
}

const handleSaveDept = async () => {
  if (!deptForm.value.name.trim()) {
    message.warning(t('admin.org.fillDeptName'))
    return
  }
  try {
    if (editingDept.value) {
      const mapped = await updateDepartment(editingDept.value.id, {
        name: deptForm.value.name,
        manager: deptForm.value.manager,
      })
      const idx = departments.value.findIndex(d => d.id === editingDept.value!.id)
      if (idx !== -1) departments.value[idx] = mapped
      message.success(t('admin.org.deptUpdated'))
    } else {
      const mapped = await createDepartment({
        name: deptForm.value.name,
        parent_id: null,
        manager: deptForm.value.manager,
      })
      departments.value.push(mapped)
      message.success(t('admin.org.deptAdded'))
    }
    showDeptModal.value = false
  } catch (e: any) {
    message.error(e.message || t('admin.org.operationFailed'))
  }
}

const removeDept = async (d: Department) => {
  const usedBy = members.value.filter(m => m.department_id === d.id)
  if (usedBy.length > 0) { message.warning(t('admin.org.deptHasMembers', [usedBy.length])); return }
  try {
    await apiDeleteDept(d.id)
    departments.value = departments.value.filter(dept => dept.id !== d.id)
    message.success(t('admin.org.deptDeleted'))
  } catch (e: any) {
    message.error(e.message || t('admin.org.operationFailed'))
  }
}

const getDeptMemberCount = (deptId: string) => members.value.filter(m => m.department_id === deptId).length
</script>

<template>
  <div class="org-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('admin.org.title') }}</h1>
        <p class="page-subtitle">{{ t('admin.org.subtitle') }}</p>
      </div>
    </div>

    <!--顶部选项卡-->
    <div class="tab-nav">
      <button
        v-for="tab in [
          { key: 'members', label: t('admin.org.tabMembers'), icon: TeamOutlined },
          { key: 'roles', label: t('admin.org.tabRoles'), icon: KeyOutlined },
          { key: 'departments', label: t('admin.org.tabDepts'), icon: ApartmentOutlined },
        ]"
        :key="tab.key"
        class="tab-btn"
        :class="{ 'tab-btn--active': topTab === tab.key }"
        @click="topTab = tab.key as any"
      >
        <component :is="tab.icon" style="font-size: 14px;" />
        {{ tab.label }}
      </button>
    </div>

    <!--===== 会员选项卡 =====-->
    <div v-if="topTab === 'members'" class="tab-content">
      <div class="toolbar">
        <div class="toolbar-left">
          <a-input v-model:value="memberSearch" :placeholder="t('admin.org.searchMember')" allow-clear style="width: 200px;">
            <template #prefix><SearchOutlined /></template>
          </a-input>
          <a-select v-model:value="memberDeptFilter" :placeholder="t('admin.org.deptFilter')" allow-clear style="width: 150px;">
            <a-select-option v-for="d in departments" :key="d.id" :value="d.id">{{ d.name }}</a-select-option>
          </a-select>
          <a-select v-model:value="memberRoleFilter" :placeholder="t('admin.org.roleFilter')" allow-clear style="width: 150px;">
            <a-select-option v-for="r in roles" :key="r.id" :value="r.id">{{ r.name }}</a-select-option>
          </a-select>
        </div>
        <a-button type="primary" @click="openAddMember"><PlusOutlined /> {{ t('admin.org.addMember') }}</a-button>
      </div>

      <div class="data-table-card">
        <table class="data-table">
          <thead>
            <tr>
              <th>{{ t('admin.org.thName') }}</th>
              <th>{{ t('admin.org.thUsername') }}</th>
              <th>{{ t('admin.org.thDepartment') }}</th>
              <th>{{ t('admin.org.thRole') }}</th>
              <th>{{ t('admin.org.thPosition') }}</th>
              <th>{{ t('admin.org.thEmail') }}</th>
              <th>{{ t('admin.org.thStatus') }}</th>
              <th>{{ t('admin.org.thAction') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="m in pagedMembers" :key="m.id">
              <td>
                <div class="member-name-cell">
                  <a-avatar :size="28" class="member-avatar"><template #icon><UserOutlined /></template></a-avatar>
                  {{ m.name }}
                </div>
              </td>
              <td class="text-secondary">{{ m.username }}</td>
              <td>{{ m.department_name }}</td>
              <td>
                <span v-for="rn in m.role_names" :key="rn" class="role-tag" style="margin-right: 4px;">{{ rn }}</span>
              </td>
              <td class="text-secondary">{{ m.position }}</td>
              <td class="text-secondary">{{ m.email }}</td>
              <td>
                <span class="status-tag" :class="m.status === 'active' ? 'status-tag--active' : 'status-tag--disabled'">
                  {{ m.status === 'active' ? t('admin.org.active') : t('admin.org.disabled') }}
                </span>
              </td>
              <td>
                <div class="action-btns">
                  <button class="icon-btn" :title="t('admin.org.edit')" @click="openEditMember(m)"><EditOutlined /></button>
                  <a-popconfirm :title="m.status === 'active' ? t('admin.org.confirmDisable') : t('admin.org.confirmEnable')" @confirm="toggleMemberStatus(m)">
                    <button class="icon-btn" :title="m.status === 'active' ? t('admin.org.disable') : t('admin.org.enable')">
                      <StopOutlined v-if="m.status === 'active'" />
                      <CheckOutlined v-else />
                    </button>
                  </a-popconfirm>
                  <a-popconfirm :title="t('admin.org.confirmDeleteMember')" @confirm="removeMember(m)">
                    <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
                  </a-popconfirm>
                </div>
              </td>
            </tr>
            <tr v-if="filteredMembers.length === 0">
              <td colspan="8" class="empty-cell">{{ t('admin.org.noData') }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!--分页-->
      <div class="pagination-wrapper">
        <a-pagination
          :current="memberPage"
          :page-size="memberPageSize"
          :total="memberTotal"
          size="small"
          show-size-changer
          show-quick-jumper
          :page-size-options="['10', '20', '50']"
          @change="onMemberPageChange"
          @showSizeChange="onMemberPageChange"
        />
      </div>
    </div>

    <!--===== 角色选项卡 =====-->
    <div v-if="topTab === 'roles'" class="tab-content">
      <div class="toolbar">
        <div class="toolbar-left">
          <span class="toolbar-hint">{{ t('admin.org.rolesHint') }}</span>
        </div>
        <a-button type="primary" @click="openAddRole"><PlusOutlined /> {{ t('admin.org.addRole') }}</a-button>
      </div>

      <div class="role-grid">
        <div v-for="r in roles" :key="r.id" class="role-card">
          <div class="role-card-header">
            <div class="role-card-title">
              <SafetyCertificateOutlined class="role-card-icon" />
              <span>{{ r.name }}</span>
              <span v-if="r.is_system" class="system-tag">{{ t('admin.org.system') }}</span>
            </div>
            <div class="role-card-actions">
              <button class="icon-btn" @click="openEditRole(r)"><EditOutlined /></button>
              <a-popconfirm v-if="!r.is_system" :title="t('admin.org.confirmDelete')" @confirm="removeRole(r)">
                <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
              </a-popconfirm>
            </div>
          </div>
          <p class="role-card-desc">{{ r.description }}</p>
          <div class="role-card-meta">
            <span class="role-meta-item"><TeamOutlined /> {{ t('admin.org.members', [getRoleMemberCount(r.id)]) }}</span>
            <span class="role-meta-item"><KeyOutlined /> {{ t('admin.org.permissions', [r.page_permissions.length]) }}</span>
          </div>
          <div class="role-card-perms">
            <span v-for="p in r.page_permissions" :key="p" class="perm-tag">
              {{ allPages.find(x => x.path === p)?.label || p }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!--===== 部门选项卡 =====-->
    <div v-if="topTab === 'departments'" class="tab-content">
      <div class="toolbar">
        <div class="toolbar-left">
          <span class="toolbar-hint">{{ t('admin.org.deptsHint') }}</span>
        </div>
        <a-button type="primary" @click="openAddDept"><PlusOutlined /> {{ t('admin.org.addDept') }}</a-button>
      </div>

      <div class="dept-grid">
        <div v-for="d in departments" :key="d.id" class="dept-card">
          <div class="dept-card-header">
            <ApartmentOutlined class="dept-card-icon" />
            <span class="dept-card-name">{{ d.name }}</span>
            <div class="dept-card-actions">
              <button class="icon-btn" @click="openEditDept(d)"><EditOutlined /></button>
              <a-popconfirm :title="t('admin.org.confirmDelete')" @confirm="removeDept(d)">
                <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
              </a-popconfirm>
            </div>
          </div>
          <div class="dept-card-body">
            <div class="dept-meta"><UserOutlined /> {{ t('admin.org.manager', [d.manager || t('admin.org.notSet')]) }}</div>
            <div class="dept-meta"><TeamOutlined /> {{ t('admin.org.deptMembers', [getDeptMemberCount(d.id)]) }}</div>
          </div>
        </div>
      </div>
    </div>

    <!--=====会员模式=====-->
    <a-modal v-model:open="showMemberModal" :title="editingMember ? t('admin.org.editMember') : t('admin.org.addMemberTitle')" @ok="handleSaveMember" :ok-text="t('admin.org.save')" :cancel-text="t('admin.org.cancel')" :width="520">
      <a-form layout="vertical" style="margin-top: 16px;">
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px;">
          <a-form-item :label="t('admin.org.name')" required>
            <a-input v-model:value="memberForm.name" :placeholder="t('admin.org.namePlaceholder')" />
          </a-form-item>
          <a-form-item :label="t('admin.org.username')" required>
            <a-input v-model:value="memberForm.username" :placeholder="!editingMember ? t('admin.org.usernameHint') : t('admin.org.usernamePlaceholder')" :disabled="!!editingMember" />
          </a-form-item>
        </div>
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px;">
          <a-form-item :label="t('admin.org.department')" required>
            <a-select v-model:value="memberForm.department_id" :placeholder="t('admin.org.selectDept')">
              <a-select-option v-for="d in departments" :key="d.id" :value="d.id">{{ d.name }}</a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item :label="t('admin.org.role')" required>
            <a-select v-model:value="memberForm.role_ids" mode="multiple" :placeholder="t('admin.org.selectRoles')">
              <a-select-option v-for="r in roles" :key="r.id" :value="r.id">{{ r.name }}</a-select-option>
            </a-select>
          </a-form-item>
        </div>
        <a-form-item :label="t('admin.org.position')">
          <a-input v-model:value="memberForm.position" :placeholder="t('admin.org.positionPlaceholder')" />
        </a-form-item>
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px;">
          <a-form-item :label="t('admin.org.email')">
            <a-input v-model:value="memberForm.email" :placeholder="t('admin.org.emailPlaceholder')" />
          </a-form-item>
          <a-form-item :label="t('admin.org.phone')">
            <a-input v-model:value="memberForm.phone" :placeholder="t('admin.org.phonePlaceholder')" :maxlength="11" />
          </a-form-item>
        </div>
      </a-form>
    </a-modal>

    <!--=====角色模态=====-->
    <a-modal v-model:open="showRoleModal" :title="editingRole ? t('admin.org.editRole') : t('admin.org.addRoleTitle')" @ok="handleSaveRole" :ok-text="t('admin.org.save')" :cancel-text="t('admin.org.cancel')" :width="560">
      <a-form layout="vertical" style="margin-top: 16px;">
        <a-form-item :label="t('admin.org.roleName')" required>
          <a-input v-model:value="roleForm.name" :placeholder="t('admin.org.roleNamePlaceholder')" />
        </a-form-item>
        <a-form-item :label="t('admin.org.roleDesc')">
          <a-textarea v-model:value="roleForm.description" :placeholder="t('admin.org.roleDescPlaceholder')" :rows="2" />
        </a-form-item>
        <a-form-item :label="t('admin.org.pagePermissions')">
          <p class="perm-hint">{{ t('admin.org.permHint') }}</p>
          <div class="perm-check-grid">
            <template v-for="group in ['common', 'business', 'admin']" :key="group">
              <div v-if="allPages.filter(p => p.group === group).length" class="perm-group-label">
                {{ group === 'common' ? t('admin.org.groupCommon') : group === 'business' ? t('admin.org.groupBusiness') : t('admin.org.groupAdmin') }}
              </div>
              <label v-for="page in allPages.filter(p => p.group === group)" :key="page.path" class="perm-check-item">
                <a-checkbox
                  :checked="roleForm.page_permissions.includes(page.path) || !!page.alwaysOn"
                  :disabled="isPermDisabled(page.path)"
                  @change="(e: any) => handlePermToggle(page.path, e.target.checked)"
                />
                <span class="perm-check-label">{{ page.label }}</span>
                <span v-if="page.dependsOn" class="perm-dep-hint">
                  <InfoCircleOutlined /> {{ t('admin.org.depHint') }}
                </span>
                <span class="perm-check-path">{{ page.path }}</span>
              </label>
            </template>
          </div>
        </a-form-item>
      </a-form>
    </a-modal>

    <!--===== 部门模态 =====-->
    <a-modal v-model:open="showDeptModal" :title="editingDept ? t('admin.org.editDept') : t('admin.org.addDeptTitle')" @ok="handleSaveDept" :ok-text="t('admin.org.save')" :cancel-text="t('admin.org.cancel')" :width="440">
      <a-form layout="vertical" style="margin-top: 16px;">
        <a-form-item :label="t('admin.org.deptName')" required>
          <a-input v-model:value="deptForm.name" :placeholder="t('admin.org.deptNamePlaceholder')" />
        </a-form-item>
        <a-form-item :label="t('admin.org.managerLabel')">
          <a-input v-model:value="deptForm.manager" :placeholder="t('admin.org.managerPlaceholder')" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.page-header { margin-bottom: 24px; }
.page-title { font-size: 24px; font-weight: 700; color: var(--color-text-primary); margin: 0; }
.page-subtitle { font-size: 14px; color: var(--color-text-tertiary); margin: 4px 0 0; }

.tab-nav {
  display: flex; gap: 4px; background: var(--color-bg-hover); padding: 4px;
  border-radius: var(--radius-lg); margin-bottom: 24px; width: fit-content;
}
.tab-btn {
  padding: 8px 20px; border: none; background: transparent; border-radius: var(--radius-md);
  font-size: 14px; font-weight: 500; color: var(--color-text-secondary); cursor: pointer;
  transition: all var(--transition-fast); display: flex; align-items: center; gap: 6px;
}
.tab-btn:hover { color: var(--color-text-primary); }
.tab-btn--active { background: var(--color-bg-card); color: var(--color-primary); box-shadow: var(--shadow-xs); }

.toolbar { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; gap: 12px; flex-wrap: wrap; }
.toolbar-left { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.toolbar-hint { font-size: 13px; color: var(--color-text-tertiary); }

/*数据表*/
.data-table-card {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); overflow: hidden;
}
.data-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.data-table th {
  padding: 12px 16px; text-align: left; font-weight: 600; color: var(--color-text-secondary);
  background: var(--color-bg-page); border-bottom: 1px solid var(--color-border-light);
  font-size: 12px; text-transform: uppercase; letter-spacing: 0.04em;
}
.data-table td {
  padding: 12px 16px; border-bottom: 1px solid var(--color-border-light);
  color: var(--color-text-primary);
}
.data-table tbody tr:hover { background: var(--color-bg-hover); }
.data-table tbody tr:last-child td { border-bottom: none; }
.text-secondary { color: var(--color-text-tertiary); }
.empty-cell { text-align: center; padding: 32px 16px !important; color: var(--color-text-tertiary); }

.member-name-cell { display: flex; align-items: center; gap: 8px; font-weight: 500; }
.member-avatar { background: linear-gradient(135deg, #4f46e5, #7c3aed) !important; flex-shrink: 0; }

.role-tag {
  font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: var(--radius-full);
  background: var(--color-primary-bg); color: var(--color-primary);
}
.status-tag {
  font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: var(--radius-full);
}
.status-tag--active { background: var(--color-success-bg); color: var(--color-success); }
.status-tag--disabled { background: var(--color-bg-hover); color: var(--color-text-tertiary); }

.action-btns { display: flex; gap: 4px; }
.icon-btn {
  width: 28px; height: 28px; border: 1px solid var(--color-border); background: transparent;
  border-radius: var(--radius-sm); cursor: pointer; display: flex; align-items: center;
  justify-content: center; color: var(--color-text-tertiary); transition: all var(--transition-fast);
}
.icon-btn:hover { border-color: var(--color-primary); color: var(--color-primary); }
.icon-btn--danger:hover { border-color: var(--color-danger); color: var(--color-danger); }

/*角色网格*/
.role-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(340px, 1fr)); gap: 16px; }
.role-card {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); padding: 20px;
}
.role-card-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.role-card-title { display: flex; align-items: center; gap: 8px; font-size: 15px; font-weight: 600; color: var(--color-text-primary); }
.role-card-icon { color: var(--color-primary); font-size: 16px; }
.role-card-actions { display: flex; gap: 4px; }
.role-card-desc { font-size: 13px; color: var(--color-text-tertiary); margin: 0 0 12px; line-height: 1.5; }
.role-card-meta { display: flex; gap: 16px; margin-bottom: 12px; }
.role-meta-item { font-size: 12px; color: var(--color-text-secondary); display: flex; align-items: center; gap: 4px; }
.role-card-perms { display: flex; flex-wrap: wrap; gap: 4px; }
.perm-tag {
  font-size: 11px; padding: 2px 8px; border-radius: var(--radius-sm);
  background: var(--color-bg-hover); color: var(--color-text-secondary);
}
.system-tag {
  font-size: 10px; font-weight: 600; padding: 1px 6px; border-radius: var(--radius-full);
  background: var(--color-warning-bg); color: var(--color-warning);
}

/*部门网格*/
.dept-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 16px; }
.dept-card {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); padding: 20px;
}
.dept-card-header { display: flex; align-items: center; gap: 10px; margin-bottom: 12px; }
.dept-card-icon { font-size: 18px; color: var(--color-primary); }
.dept-card-name { font-size: 15px; font-weight: 600; color: var(--color-text-primary); flex: 1; }
.dept-card-actions { display: flex; gap: 4px; }
.dept-card-body { display: flex; flex-direction: column; gap: 6px; }
.dept-meta { font-size: 13px; color: var(--color-text-secondary); display: flex; align-items: center; gap: 6px; }

/*权限模式*/
.perm-hint { font-size: 12px; color: var(--color-text-tertiary); margin: 0 0 8px; }
.perm-check-grid { display: flex; flex-direction: column; gap: 6px; }
.perm-group-label { font-size: 12px; font-weight: 600; color: var(--color-text-secondary); margin-top: 8px; padding: 4px 8px; text-transform: uppercase; letter-spacing: 0.04em; }
.perm-dep-hint { font-size: 11px; color: var(--color-warning); display: flex; align-items: center; gap: 3px; }
.perm-check-item { display: flex; align-items: center; gap: 8px; padding: 6px 8px; border-radius: var(--radius-sm); }
.perm-check-item:hover { background: var(--color-bg-hover); }
.perm-check-label { font-size: 13px; font-weight: 500; color: var(--color-text-primary); min-width: 80px; }
.perm-check-path { font-size: 11px; color: var(--color-text-tertiary); font-family: monospace; }

@media (max-width: 768px) {
  .role-grid { grid-template-columns: 1fr; }
  .dept-grid { grid-template-columns: 1fr; }
  .data-table-card { overflow-x: auto; -webkit-overflow-scrolling: touch; }
  .data-table { min-width: 700px; }
  .toolbar { flex-direction: column; align-items: stretch; }
  .toolbar-left { flex-direction: column; }
  .toolbar-left > * { width: 100% !important; }
  .page-title { font-size: 20px; }
  .tab-nav { width: 100%; overflow-x: auto; -webkit-overflow-scrolling: touch; }
  .tab-btn { flex-shrink: 0; padding: 8px 14px; font-size: 13px; }
}
</style>
