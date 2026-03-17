import type { Department, OrgRole, OrgMember } from '~/types/org'

// ============================================================
// Backend DTO types (nested structure from Go service)
// ============================================================
interface ApiMemberUserInfo {
  id: string; username: string; display_name: string
  email: string; phone: string; avatar_url: string
}
interface ApiDepartmentResponse {
  id: string; name: string; parent_id?: string | null
  manager: string; sort_order: number; member_count: number
  created_at: string; updated_at: string
}
interface ApiRoleResponse {
  id: string; name: string; description: string
  page_permissions: string[] | any; is_system: boolean
  created_at: string; updated_at: string
}
interface ApiMemberResponse {
  id: string; user: ApiMemberUserInfo; department: ApiDepartmentResponse
  roles: ApiRoleResponse[]; position: string; status: string
  created_at: string; updated_at: string
}

// ============================================================
// DTO → Frontend model mappers
// ============================================================
function mapDepartment(d: ApiDepartmentResponse): Department {
  return {
    id: d.id, name: d.name, parent_id: d.parent_id ?? null,
    manager: d.manager, member_count: d.member_count ?? 0,
  }
}

function mapRole(r: ApiRoleResponse): OrgRole {
  let perms: string[] = []
  if (Array.isArray(r.page_permissions)) perms = r.page_permissions
  else if (r.page_permissions) {
    try { perms = typeof r.page_permissions === 'string' ? JSON.parse(r.page_permissions) : [] } catch { perms = [] }
  }
  return {
    id: r.id, name: r.name, description: r.description,
    page_permissions: perms, is_system: r.is_system,
  }
}

function mapMember(m: ApiMemberResponse): OrgMember {
  const roleIds = m.roles?.map(r => r.id) ?? []
  const roleNames = m.roles?.map(r => r.name) ?? []
  return {
    id: m.id,
    name: m.user?.display_name ?? m.user?.username ?? '',
    username: m.user?.username ?? '',
    department_id: m.department?.id ?? '',
    department_name: m.department?.name ?? '',
    role_ids: roleIds,
    role_names: roleNames,
    email: m.user?.email ?? '',
    phone: m.user?.phone ?? '',
    position: m.position ?? '',
    status: (m.status as 'active' | 'disabled') ?? 'active',
    created_at: m.created_at ?? '',
  }
}

export const useOrgApi = () => {
  const { authFetch } = useAuth()

  const departments = ref<Department[]>([])
  const roles = ref<OrgRole[]>([])
  const members = ref<OrgMember[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  // ============================================================
  // Departments
  // ============================================================

  async function listDepartments(): Promise<Department[]> {
    loading.value = true
    error.value = null
    try {
      const data = await authFetch<ApiDepartmentResponse[]>('/api/tenant/org/departments')
      departments.value = data.map(d => mapDepartment(d))
      return departments.value
    }
    catch (e: any) {
      error.value = e.message || 'Failed to load departments'
      console.error('[useOrgApi] listDepartments failed', e)
      throw e
    }
    finally { loading.value = false }
  }

  async function createDepartment(dept: Omit<Department, 'id' | 'member_count'>): Promise<Department> {
    const data = await authFetch<ApiDepartmentResponse>('/api/tenant/org/departments', { method: 'POST', body: dept })
    const mapped = mapDepartment(data)
    return mapped
  }

  async function updateDepartment(id: string, dept: Partial<Department>): Promise<Department> {
    const data = await authFetch<ApiDepartmentResponse>(`/api/tenant/org/departments/${id}`, { method: 'PUT', body: dept })
    const mapped = mapDepartment(data)
    return mapped
  }

  async function deleteDepartment(id: string): Promise<void> {
    await authFetch<null>(`/api/tenant/org/departments/${id}`, { method: 'DELETE' })
  }

  // ============================================================
  // Roles
  // ============================================================

  async function listRoles(): Promise<OrgRole[]> {
    loading.value = true
    error.value = null
    try {
      const data = await authFetch<ApiRoleResponse[]>('/api/tenant/org/roles')
      roles.value = data.map(mapRole)
      return roles.value
    }
    catch (e: any) {
      error.value = e.message || 'Failed to load roles'
      console.error('[useOrgApi] listRoles failed', e)
      throw e
    }
    finally { loading.value = false }
  }

  async function createRole(role: Omit<OrgRole, 'id'>): Promise<OrgRole> {
    const data = await authFetch<ApiRoleResponse>('/api/tenant/org/roles', { method: 'POST', body: role })
    const mapped = mapRole(data)
    return mapped
  }

  async function updateRole(id: string, role: Partial<OrgRole>): Promise<OrgRole> {
    const data = await authFetch<ApiRoleResponse>(`/api/tenant/org/roles/${id}`, { method: 'PUT', body: role })
    const mapped = mapRole(data)
    return mapped
  }

  async function deleteRole(id: string): Promise<void> {
    await authFetch<null>(`/api/tenant/org/roles/${id}`, { method: 'DELETE' })
  }

  // ============================================================
  // Members
  // ============================================================

  async function listMembers(): Promise<OrgMember[]> {
    loading.value = true
    error.value = null
    try {
      const data = await authFetch<ApiMemberResponse[]>('/api/tenant/org/members')
      members.value = data.map(mapMember)
      return members.value
    }
    catch (e: any) {
      error.value = e.message || 'Failed to load members'
      console.error('[useOrgApi] listMembers failed', e)
      throw e
    }
    finally { loading.value = false }
  }

  async function createMember(member: Omit<OrgMember, 'id' | 'created_at'>): Promise<OrgMember> {
    const body = {
      username: member.username,
      display_name: member.name,
      email: member.email,
      phone: member.phone,
      department_id: member.department_id,
      role_ids: member.role_ids,
      position: member.position,
    }
    const data = await authFetch<ApiMemberResponse>('/api/tenant/org/members', { method: 'POST', body })
    const mapped = mapMember(data)
    return mapped
  }

  async function updateMember(id: string, member: Partial<OrgMember>): Promise<OrgMember> {
    const body: Record<string, any> = {}
    if (member.name !== undefined) body.display_name = member.name
    if (member.email !== undefined) body.email = member.email
    if (member.phone !== undefined) body.phone = member.phone
    if (member.department_id !== undefined) body.department_id = member.department_id
    if (member.role_ids !== undefined) body.role_ids = member.role_ids
    if (member.position !== undefined) body.position = member.position
    if (member.status !== undefined) body.status = member.status

    const data = await authFetch<ApiMemberResponse>(`/api/tenant/org/members/${id}`, { method: 'PUT', body })
    const mapped = mapMember(data)
    return mapped
  }

  async function deleteMember(id: string): Promise<void> {
    await authFetch<null>(`/api/tenant/org/members/${id}`, { method: 'DELETE' })
  }

  async function loadAll(): Promise<void> {
    await Promise.all([listDepartments(), listRoles(), listMembers()])
  }

  return {
    departments, roles, members, loading, error,
    loadAll,
    listDepartments, createDepartment, updateDepartment, deleteDepartment,
    listRoles, createRole, updateRole, deleteRole,
    listMembers, createMember, updateMember, deleteMember,
  }
}
