<script setup lang="ts">
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
import type { Department, OrgRole, OrgMember } from '~/composables/useMockData'

definePageMeta({ middleware: 'auth', layout: 'default' })

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
    message.warning('请填写姓名和用户名')
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
    message.success('人员信息已更新')
  } else {
    members.value.push({
      id: `M-${Date.now()}`,
      ...memberForm.value,
      department_name: dept?.name || '',
      role_name: role?.name || '',
      status: 'active',
      created_at: new Date().toISOString().slice(0, 10),
    })
    message.success('人员已添加')
  }
  showMemberModal.value = false
}

const toggleMemberStatus = (m: OrgMember) => {
  m.status = m.status === 'active' ? 'disabled' : 'active'
  message.success(m.status === 'active' ? '已启用' : '已禁用')
}

const deleteMember = (m: OrgMember) => {
  members.value = members.value.filter(x => x.id !== m.id)
  message.success('已删除')
}

// ===== Roles =====
const roles = ref<OrgRole[]>(JSON.parse(JSON.stringify(mockOrgRoles)))
const showRoleModal = ref(false)
const editingRole = ref<OrgRole | null>(null)
const roleForm = ref({ name: '', description: '', page_permissions: [] as string[] })

const allPages = [
  { path: '/dashboard', label: '审核工作台' },
  { path: '/cron', label: '定时任务' },
  { path: '/archive', label: '归档复盘' },
  { path: '/settings', label: '个人设置' },
  { path: '/admin/tenant', label: '规则配置' },
  { path: '/admin/tenant/org', label: '组织人员' },
  { path: '/admin/tenant/data', label: '数据信息' },
  { path: '/admin/system', label: '全局监控' },
  { path: '/admin/system/tenants', label: '租户管理' },
  { path: '/admin/system/settings', label: '系统设置' },
]

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
    message.warning('请填写角色名称')
    return
  }
  if (editingRole.value) {
    Object.assign(editingRole.value, roleForm.value)
    // Update member role names
    members.value.forEach(m => {
      if (m.role_id === editingRole.value!.id) m.role_name = roleForm.value.name
    })
    message.success('角色已更新')
  } else {
    roles.value.push({
      id: `ROLE-${Date.now()}`,
      ...roleForm.value,
      is_system: false,
    })
    message.success('角色已添加')
  }
  showRoleModal.value = false
}

const deleteRole = (r: OrgRole) => {
  if (r.is_system) { message.warning('系统角色不可删除'); return }
  const usedBy = members.value.filter(m => m.role_id === r.id)
  if (usedBy.length > 0) { message.warning(`该角色下有 ${usedBy.length} 名成员，请先调整`); return }
  roles.value = roles.value.filter(x => x.id !== r.id)
  message.success('角色已删除')
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
    message.warning('请填写部门名称')
    return
  }
  if (editingDept.value) {
    editingDept.value.name = deptForm.value.name
    editingDept.value.manager = deptForm.value.manager
    // Update member department names
    members.value.forEach(m => {
      if (m.department_id === editingDept.value!.id) m.department_name = deptForm.value.name
    })
    message.success('部门已更新')
  } else {
    const newDept: Department = {
      id: `D-${Date.now()}`,
      name: deptForm.value.name,
      parent_id: null,
      manager: deptForm.value.manager,
      member_count: 0,
    }
    departments.value.push(newDept)
    message.success('部门已添加')
  }
  showDeptModal.value = false
}

const deleteDept = (d: Department) => {
  const usedBy = members.value.filter(m => m.department_id === d.id)
  if (usedBy.length > 0) { message.warning(`该部门下有 ${usedBy.length} 名成员，请先调整`); return }
  departments.value = departments.value.filter(x => x.id !== d.id)
  message.success('部门已删除')
}

const getDeptMemberCount = (deptId: string) => members.value.filter(m => m.department_id === deptId).length
</script>

<template>
  <div class="org-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">组织人员</h1>
        <p class="page-subtitle">管理组织架构、角色权限与人员信息</p>
      </div>
    </div>

    <!-- Top tabs -->
    <div class="tab-nav">
      <button
        v-for="tab in [
          { key: 'members', label: '人员管理', icon: TeamOutlined },
          { key: 'roles', label: '角色权限', icon: KeyOutlined },
          { key: 'departments', label: '部门管理', icon: ApartmentOutlined },
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
          <a-input v-model:value="memberSearch" placeholder="搜索姓名/用户名" allow-clear style="width: 200px;">
            <template #prefix><SearchOutlined /></template>
          </a-input>
          <a-select v-model:value="memberDeptFilter" placeholder="部门筛选" allow-clear style="width: 150px;">
            <a-select-option v-for="d in departments" :key="d.id" :value="d.id">{{ d.name }}</a-select-option>
          </a-select>
          <a-select v-model:value="memberRoleFilter" placeholder="角色筛选" allow-clear style="width: 150px;">
            <a-select-option v-for="r in roles" :key="r.id" :value="r.id">{{ r.name }}</a-select-option>
          </a-select>
        </div>
        <a-button type="primary" @click="openAddMember"><PlusOutlined /> 添加人员</a-button>
      </div>

      <div class="data-table-card">
        <table class="data-table">
          <thead>
            <tr>
              <th>姓名</th>
              <th>用户名</th>
              <th>部门</th>
              <th>角色</th>
              <th>职位</th>
              <th>邮箱</th>
              <th>状态</th>
              <th>操作</th>
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
                  {{ m.status === 'active' ? '正常' : '已禁用' }}
                </span>
              </td>
              <td>
                <div class="action-btns">
                  <button class="icon-btn" title="编辑" @click="openEditMember(m)"><EditOutlined /></button>
                  <a-popconfirm :title="m.status === 'active' ? '确认禁用？' : '确认启用？'" @confirm="toggleMemberStatus(m)">
                    <button class="icon-btn" :title="m.status === 'active' ? '禁用' : '启用'">
                      <StopOutlined v-if="m.status === 'active'" />
                      <CheckOutlined v-else />
                    </button>
                  </a-popconfirm>
                  <a-popconfirm title="确认删除该人员？" @confirm="deleteMember(m)">
                    <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
                  </a-popconfirm>
                </div>
              </td>
            </tr>
            <tr v-if="filteredMembers.length === 0">
              <td colspan="8" class="empty-cell">暂无数据</td>
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
          <span class="toolbar-hint">定义角色并分配页面访问权限，角色与人员关联后生效</span>
        </div>
        <a-button type="primary" @click="openAddRole"><PlusOutlined /> 新建角色</a-button>
      </div>

      <div class="role-grid">
        <div v-for="r in roles" :key="r.id" class="role-card">
          <div class="role-card-header">
            <div class="role-card-title">
              <SafetyCertificateOutlined class="role-card-icon" />
              <span>{{ r.name }}</span>
              <span v-if="r.is_system" class="system-tag">系统</span>
            </div>
            <div class="role-card-actions">
              <button class="icon-btn" @click="openEditRole(r)"><EditOutlined /></button>
              <a-popconfirm v-if="!r.is_system" title="确认删除？" @confirm="deleteRole(r)">
                <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
              </a-popconfirm>
            </div>
          </div>
          <p class="role-card-desc">{{ r.description }}</p>
          <div class="role-card-meta">
            <span class="role-meta-item"><TeamOutlined /> {{ getRoleMemberCount(r.id) }} 人</span>
            <span class="role-meta-item"><KeyOutlined /> {{ r.page_permissions.length }} 项权限</span>
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
          <span class="toolbar-hint">管理组织部门结构</span>
        </div>
        <a-button type="primary" @click="openAddDept"><PlusOutlined /> 新建部门</a-button>
      </div>

      <div class="dept-grid">
        <div v-for="d in departments" :key="d.id" class="dept-card">
          <div class="dept-card-header">
            <ApartmentOutlined class="dept-card-icon" />
            <span class="dept-card-name">{{ d.name }}</span>
            <div class="dept-card-actions">
              <button class="icon-btn" @click="openEditDept(d)"><EditOutlined /></button>
              <a-popconfirm title="确认删除？" @confirm="deleteDept(d)">
                <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
              </a-popconfirm>
            </div>
          </div>
          <div class="dept-card-body">
            <div class="dept-meta"><UserOutlined /> 负责人：{{ d.manager || '未设置' }}</div>
            <div class="dept-meta"><TeamOutlined /> 成员：{{ getDeptMemberCount(d.id) }} 人</div>
          </div>
        </div>
      </div>
    </div>

    <!-- ===== Member Modal ===== -->
    <a-modal v-model:open="showMemberModal" :title="editingMember ? '编辑人员' : '添加人员'" @ok="handleSaveMember" ok-text="保存" cancel-text="取消" :width="520">
      <a-form layout="vertical" style="margin-top: 16px;">
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px;">
          <a-form-item label="姓名" required>
            <a-input v-model:value="memberForm.name" placeholder="请输入姓名" />
          </a-form-item>
          <a-form-item label="用户名" required>
            <a-input v-model:value="memberForm.username" placeholder="请输入用户名" :disabled="!!editingMember" />
          </a-form-item>
        </div>
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px;">
          <a-form-item label="部门">
            <a-select v-model:value="memberForm.department_id" placeholder="选择部门">
              <a-select-option v-for="d in departments" :key="d.id" :value="d.id">{{ d.name }}</a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item label="角色">
            <a-select v-model:value="memberForm.role_id" placeholder="选择角色">
              <a-select-option v-for="r in roles" :key="r.id" :value="r.id">{{ r.name }}</a-select-option>
            </a-select>
          </a-form-item>
        </div>
        <a-form-item label="职位">
          <a-input v-model:value="memberForm.position" placeholder="请输入职位" />
        </a-form-item>
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px;">
          <a-form-item label="邮箱">
            <a-input v-model:value="memberForm.email" placeholder="请输入邮箱" />
          </a-form-item>
          <a-form-item label="手机号">
            <a-input v-model:value="memberForm.phone" placeholder="请输入手机号" />
          </a-form-item>
        </div>
      </a-form>
    </a-modal>

    <!-- ===== Role Modal ===== -->
    <a-modal v-model:open="showRoleModal" :title="editingRole ? '编辑角色' : '新建角色'" @ok="handleSaveRole" ok-text="保存" cancel-text="取消" :width="560">
      <a-form layout="vertical" style="margin-top: 16px;">
        <a-form-item label="角色名称" required>
          <a-input v-model:value="roleForm.name" placeholder="请输入角色名称" />
        </a-form-item>
        <a-form-item label="角色描述">
          <a-textarea v-model:value="roleForm.description" placeholder="请输入角色描述" :rows="2" />
        </a-form-item>
        <a-form-item label="页面访问权限">
          <p class="perm-hint">勾选该角色可访问的页面</p>
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
    <a-modal v-model:open="showDeptModal" :title="editingDept ? '编辑部门' : '新建部门'" @ok="handleSaveDept" ok-text="保存" cancel-text="取消" :width="440">
      <a-form layout="vertical" style="margin-top: 16px;">
        <a-form-item label="部门名称" required>
          <a-input v-model:value="deptForm.name" placeholder="请输入部门名称" />
        </a-form-item>
        <a-form-item label="负责人">
          <a-input v-model:value="deptForm.manager" placeholder="请输入负责人姓名" />
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
