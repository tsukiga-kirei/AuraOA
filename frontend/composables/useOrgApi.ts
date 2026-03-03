import type { Department, OrgRole, OrgMember } from '~/types/org'

export const useOrgApi = () => {
  const { authFetch } = useAuth()

  //反应式数据数组 — 从空开始，由 API 填充
  const departments = ref<Department[]>([])
  const roles = ref<OrgRole[]>([])
  const members = ref<OrgMember[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  // ============================================================
  //部门
  // ============================================================

  async function listDepartments(): Promise<Department[]> {
    loading.value = true
    error.value = null
    try {
      const data = await authFetch<Department[]>('/api/tenant/org/departments')
      departments.value = data
      return data
    }
    catch (e: any) {
      error.value = e.message || 'Failed to load departments'
      console.error('[useOrgApi] listDepartments failed', e)
      throw e
    }
    finally { loading.value = false }
  }

  async function createDepartment(dept: Omit<Department, 'id' | 'member_count'>): Promise<Department> {
    const data = await authFetch<Department>('/api/tenant/org/departments', { method: 'POST', body: dept })
    departments.value.push(data)
    return data
  }

  async function updateDepartment(id: string, dept: Partial<Department>): Promise<Department> {
    const data = await authFetch<Department>(`/api/tenant/org/departments/${id}`, { method: 'PUT', body: dept })
    const idx = departments.value.findIndex(d => d.id === id)
    if (idx !== -1) departments.value[idx] = data
    return data
  }

  async function deleteDepartment(id: string): Promise<void> {
    await authFetch<null>(`/api/tenant/org/departments/${id}`, { method: 'DELETE' })
    departments.value = departments.value.filter(d => d.id !== id)
  }

  // ============================================================
  //角色
  // ============================================================

  async function listRoles(): Promise<OrgRole[]> {
    loading.value = true
    error.value = null
    try {
      const data = await authFetch<OrgRole[]>('/api/tenant/org/roles')
      roles.value = data
      return data
    }
    catch (e: any) {
      error.value = e.message || 'Failed to load roles'
      console.error('[useOrgApi] listRoles failed', e)
      throw e
    }
    finally { loading.value = false }
  }

  async function createRole(role: Omit<OrgRole, 'id'>): Promise<OrgRole> {
    const data = await authFetch<OrgRole>('/api/tenant/org/roles', { method: 'POST', body: role })
    roles.value.push(data)
    return data
  }

  async function updateRole(id: string, role: Partial<OrgRole>): Promise<OrgRole> {
    const data = await authFetch<OrgRole>(`/api/tenant/org/roles/${id}`, { method: 'PUT', body: role })
    const idx = roles.value.findIndex(r => r.id === id)
    if (idx !== -1) roles.value[idx] = data
    return data
  }

  async function deleteRole(id: string): Promise<void> {
    await authFetch<null>(`/api/tenant/org/roles/${id}`, { method: 'DELETE' })
    roles.value = roles.value.filter(r => r.id !== id)
  }

  // ============================================================
  //会员
  // ============================================================

  async function listMembers(): Promise<OrgMember[]> {
    loading.value = true
    error.value = null
    try {
      const data = await authFetch<OrgMember[]>('/api/tenant/org/members')
      members.value = data
      return data
    }
    catch (e: any) {
      error.value = e.message || 'Failed to load members'
      console.error('[useOrgApi] listMembers failed', e)
      throw e
    }
    finally { loading.value = false }
  }

  async function createMember(member: Omit<OrgMember, 'id' | 'created_at'>): Promise<OrgMember> {
    const data = await authFetch<OrgMember>('/api/tenant/org/members', { method: 'POST', body: member })
    members.value.push(data)
    return data
  }

  async function updateMember(id: string, member: Partial<OrgMember>): Promise<OrgMember> {
    const data = await authFetch<OrgMember>(`/api/tenant/org/members/${id}`, { method: 'PUT', body: member })
    const idx = members.value.findIndex(m => m.id === id)
    if (idx !== -1) members.value[idx] = data
    return data
  }

  async function deleteMember(id: string): Promise<void> {
    await authFetch<null>(`/api/tenant/org/members/${id}`, { method: 'DELETE' })
    members.value = members.value.filter(m => m.id !== id)
  }

  /** 从API加载所有组织数据*/
  async function loadAll(): Promise<void> {
    await Promise.all([listDepartments(), listRoles(), listMembers()])
  }

  return {
    //反应性数据
    departments,
    roles,
    members,
    loading,
    error,

    //加载全部
    loadAll,

    //部门增删改查
    listDepartments,
    createDepartment,
    updateDepartment,
    deleteDepartment,

    //角色增删改查
    listRoles,
    createRole,
    updateRole,
    deleteRole,

    //会员增删改查
    listMembers,
    createMember,
    updateMember,
    deleteMember,
  }
}
