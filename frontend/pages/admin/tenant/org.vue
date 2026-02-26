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
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { useMockData } from '~/composables/useMockData'
import type { Department, OrgRole, OrgMember } from '~/composables/useMockData'

definePageMeta({ middleware: 'auth', layout: 'default' })

const { t } = useI18n()
const { mockDepartments, mockOrgRoles, mockOrgMembers } = useMockData()

// Top-level tab
const topTab = ref<'members' | 'roles' | 'departments'>('members')

// ===== Members =====
const members = ref<OrgMember[]>(JSON.parse(JSON.stringify(mockOrgMembers)))
const memberSearch = ref('')
const memberDeptFilter = ref<string | undefined>(undefined)
const memberRoleFilter = ref<string | undefined>(undefined)

const filteredMembers = computed(() => {
  return members.value.filter(m => {
    if (memberSearch.value && !m.name.includes(memberSearch.value) && !m.username.includes(memberSearch.value)) return false
    if (memberDeptFilter.value && m.department_id !== memberDeptFilter.value) return false
    if (memberRoleFilter.value && m.role_id !== memberRoleFilter.value) return false
    return true
  })
})

// Pagination for members
const { paged: pagedMembers, current: memberPage, pageSize: memberPageSize, total: memberTotal, onChange: onMemberPageChange } = usePagination(filteredMembers, 10)

const showMemberModal = ref(false)
const editingMember = ref<OrgMember | null>(null)
const memberForm = ref({
  name: '', username: '', department_id: '', role_id: '', email: '', phone: '', position: '',
})

const openAddMember = () => {
  editingMember.value = null
  memberForm.value = { name: '', username: '', department_id: '', role_id: '', email: '', phone: '', position: '' }
  showMemberModal.value = true
}

const openEditMember = (m: OrgMember) => {
  editingMember.value = m
  memberForm.value = { name: m.name, username: m.username, department_id: m.department_id, role_id: m.role_id, email: m.email, phone: m.phone, position: m.position }
  showMemberModal.value = true
}

const handleSaveMember = () => {
  if (!memberForm.value.name.trim() || !memberForm.value.username.trim()) {
    message.warning(t('admin.org.fillNameRequired'))
    return
  }
  const dept = departments.value.find(d => d.id === memberForm.value.department_id)
  const role = roles.value.find(r => r.id === memberForm.value.role_id)
  if (editingMember.value) {
    Object.assign(editingMember.value, {
      ...memberForm.value,
      department_name: dept?.name || '',
      role_name: role?.name || '',
    })
    message.success(t('admin.org.memberUpdated'))
  } else {
    members.value.push({
      id: `M-${Date.now()}`,
      ...memberForm.value,
      department_name: dept?.name || '',
      role_name: role?.name || '',
      status: 'active',
      created_at: new Date().toISOString().slice(0, 10),
    })
    message.success(t('admin.org.memberAdded'))
  }
  showMemberModal.value = false
}

const toggleMemberStatus = (m: OrgMember) => {
  m.status = m.status === 'active' ? 'disabled' : 'active'
  message.success(m.status === 'active' ? t('admin.org.memberEnabled') : t('admin.org.memberDisabled'))
}

const deleteMember = (m: OrgMember) => {
  members.value = members.value.filter(x => x.id !== m.id)
  message.success(t('admin.org.memberDeleted'))
}

// ===== Roles =====
const roles = ref<OrgRole[]>(JSON.parse(JSON.stringify(mockOrgRoles)))
const showRoleModal = ref(false)
const editingRole = ref<OrgRole | null>(null)
const roleForm = ref({ name: '', description: '', page_permissions: [] as string[] })

const allPages = computed(() => [
  { path: '/overview', label: t('menu.overview') },
  { path: '/dashboard', label: t('admin.org.page.dashboard') },
  { path: '/cron', label: t('admin.org.page.cron') },
  { path: '/archive', label: t('admin.org.page.archive') },
  { path: '/settings', label: t('admin.org.page.settings') },
  { path: '/admin/tenant/rules', label: t('admin.org.page.tenantConfig') },
  { path: '/admin/tenant/org', label: t('admin.org.page.tenantOrg') },
  { path: '/admin/tenant/data', label: t('admin.org.page.tenantData') },
  { path: '/admin/tenant/user-configs', label: t('menu.tenant.userConfigs') },
  { path: '/admin/system/tenants', label: t('admin.org.page.sysTenants') },
  { path: '/admin/system/settings', label: t('admin.org.page.sysSettings') },
])

const openAddRole = () => {
  editingRole.value = null
  roleForm.value = { name: '', description: '', page_permissions: ['/dashboard', '/settings'] }
  showRoleModal.value = true
}

const openEditRole = (r: OrgRole) => {
  editingRole.value = r
  roleForm.value = { name: r.name, description: r.description, page_permissions: [...r.page_permissions] }
  showRoleModal.value = true
}

const handleSaveRole = () => {
  if (!roleForm.value.name.trim()) {
    message.warning(t('admin.org.fillRoleName'))
    return
  }
  if (editingRole.value) {
    Object.assign(editingRole.value, roleForm.value)
    // Update member role names
    members.value.forEach(m => {
      if (m.role_id === editingRole.value!.id) m.role_name = roleForm.value.name
    })
    message.success(t('admin.org.roleUpdated'))
  } else {
    roles.value.push({
      id: `ROLE-${Date.now()}`,
      ...roleForm.value,
      is_system: false,
    })
    message.success(t('admin.org.roleAdded'))
  }
  showRoleModal.value = false
}

const deleteRole = (r: OrgRole) => {
  if (r.is_system) { message.warning(t('admin.org.systemRoleProtected')); return }
  const usedBy = members.value.filter(m => m.role_id === r.id)
  if (usedBy.length > 0) { message.warning(t('admin.org.roleHasMembers', [usedBy.length])); return }
  roles.value = roles.value.filter(x => x.id !== r.id)
  message.success(t('admin.org.roleDeleted'))
}

const getRoleMemberCount = (roleId: string) => members.value.filter(m => m.role_id === roleId).length

// ===== Departments =====
const departments = ref<Department[]>(JSON.parse(JSON.stringify(mockDepartments)))
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

const handleSaveDept = () => {
  if (!deptForm.value.name.trim()) {
    message.warning(t('admin.org.fillDeptName'))
    return
  }
  if (editingDept.value) {
    editingDept.value.name = deptForm.value.name
    editingDept.value.manager = deptForm.value.manager
    // Update member department names
    members.value.forEach(m => {
      if (m.department_id === editingDept.value!.id) m.department_name = deptForm.value.name
    })
    message.success(t('admin.org.deptUpdated'))
  } else {
    const newDept: Department = {
      id: `D-${Date.now()}`,
      name: deptForm.value.name,
      parent_id: null,
      manager: deptForm.value.manager,
      member_count: 0,
    }
    departments.value.push(newDept)
    message.success(t('admin.org.deptAdded'))
  }
  showDeptModal.value = false
}

const deleteDept = (d: Department) => {
  const usedBy = members.value.filter(m => m.department_id === d.id)
  if (usedBy.length > 0) { message.warning(t('admin.org.deptHasMembers', [usedBy.length])); return }
  departments.value = departments.value.filter(x => x.id !== d.id)
  message.success(t('admin.org.deptDeleted'))
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

    <!-- Top tabs -->
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

    <!-- ===== Members Tab ===== -->
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
              <td><span class="role-tag">{{ m.role_name }}</span></td>
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
                  <a-popconfirm :title="t('admin.org.confirmDeleteMember')" @confirm="deleteMember(m)">
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

      <!-- Pagination -->
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

    <!-- ===== Roles Tab ===== -->
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
              <a-popconfirm v-if="!r.is_system" :title="t('admin.org.confirmDelete')" @confirm="deleteRole(r)">
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

    <!-- ===== Departments Tab ===== -->
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
              <a-popconfirm :title="t('admin.org.confirmDelete')" @confirm="deleteDept(d)">
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

    <!-- ===== Member Modal ===== -->
    <a-modal v-model:open="showMemberModal" :title="editingMember ? t('admin.org.editMember') : t('admin.org.addMemberTitle')" @ok="handleSaveMember" :ok-text="t('admin.org.save')" :cancel-text="t('admin.org.cancel')" :width="520">
      <a-form layout="vertical" style="margin-top: 16px;">
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px;">
          <a-form-item :label="t('admin.org.name')" required>
            <a-input v-model:value="memberForm.name" :placeholder="t('admin.org.namePlaceholder')" />
          </a-form-item>
          <a-form-item :label="t('admin.org.username')" required>
            <a-input v-model:value="memberForm.username" :placeholder="t('admin.org.usernamePlaceholder')" :disabled="!!editingMember" />
          </a-form-item>
        </div>
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px;">
          <a-form-item :label="t('admin.org.department')">
            <a-select v-model:value="memberForm.department_id" :placeholder="t('admin.org.selectDept')">
              <a-select-option v-for="d in departments" :key="d.id" :value="d.id">{{ d.name }}</a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item :label="t('admin.org.role')">
            <a-select v-model:value="memberForm.role_id" :placeholder="t('admin.org.selectRole')">
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
            <a-input v-model:value="memberForm.phone" :placeholder="t('admin.org.phonePlaceholder')" />
          </a-form-item>
        </div>
      </a-form>
    </a-modal>

    <!-- ===== Role Modal ===== -->
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
            <label v-for="page in allPages" :key="page.path" class="perm-check-item">
              <a-checkbox
                :checked="roleForm.page_permissions.includes(page.path)"
                @change="(e: any) => {
                  if (e.target.checked) roleForm.page_permissions.push(page.path)
                  else roleForm.page_permissions = roleForm.page_permissions.filter(p => p !== page.path)
                }"
              />
              <span class="perm-check-label">{{ page.label }}</span>
              <span class="perm-check-path">{{ page.path }}</span>
            </label>
          </div>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- ===== Department Modal ===== -->
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

/* Data table */
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

/* Role grid */
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

/* Department grid */
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

/* Permission modal */
.perm-hint { font-size: 12px; color: var(--color-text-tertiary); margin: 0 0 8px; }
.perm-check-grid { display: flex; flex-direction: column; gap: 6px; }
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
