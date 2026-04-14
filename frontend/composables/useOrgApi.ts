import type { Department, OrgRole, OrgMember } from '~/types/org'

// ============================================================
// 后端响应 DTO 类型（与 Go 服务返回的嵌套结构对应）
// ============================================================

/** 成员关联的用户基本信息 */
interface ApiMemberUserInfo {
  id: string; username: string; display_name: string
  email: string; phone: string; avatar_url: string
}

/** 后端返回的部门数据结构 */
interface ApiDepartmentResponse {
  id: string; name: string; parent_id?: string | null
  manager: string; sort_order: number; member_count: number
  created_at: string; updated_at: string
}

/** 后端返回的角色数据结构 */
interface ApiRoleResponse {
  id: string; name: string; description: string
  page_permissions: string[] | any; is_system: boolean
  created_at: string; updated_at: string
}

/** 后端返回的成员数据结构（含嵌套用户、部门、角色信息） */
interface ApiMemberResponse {
  id: string; user: ApiMemberUserInfo; department: ApiDepartmentResponse
  roles: ApiRoleResponse[]; position: string; status: string
  created_at: string; updated_at: string
}

// ============================================================
// 后端 DTO → 前端模型转换函数
// ============================================================

/** 将后端部门响应转换为前端 Department 模型 */
function mapDepartment(d: ApiDepartmentResponse): Department {
  return {
    id: d.id, name: d.name, parent_id: d.parent_id ?? null,
    manager: d.manager, member_count: d.member_count ?? 0,
  }
}

/**
 * 将后端角色响应转换为前端 OrgRole 模型。
 * page_permissions 字段可能为数组或 JSON 字符串，统一转换为字符串数组。
 */
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

/** 将后端成员响应转换为前端 OrgMember 模型，提取角色 ID 和名称列表 */
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

  /** 部门列表（响应式，供模板直接绑定） */
  const departments = ref<Department[]>([])
  /** 角色列表（响应式） */
  const roles = ref<OrgRole[]>([])
  /** 成员列表（响应式） */
  const members = ref<OrgMember[]>([])
  /** 加载状态标志 */
  const loading = ref(false)
  /** 错误信息（null 表示无错误） */
  const error = ref<string | null>(null)

  // ============================================================
  // 部门管理
  // ============================================================

  /** 获取当前租户所有部门列表 */
  async function listDepartments(): Promise<Department[]> {
    loading.value = true
    error.value = null
    try {
      const data = await authFetch<ApiDepartmentResponse[]>('/api/tenant/org/departments')
      departments.value = data.map(d => mapDepartment(d))
      return departments.value
    }
    catch (e: any) {
      error.value = e.message || '加载部门列表失败'
      console.error('[useOrgApi] listDepartments failed', e)
      throw e
    }
    finally { loading.value = false }
  }

  /**
   * 创建新部门。
   * @param dept 部门信息（名称、上级部门、负责人等）
   */
  async function createDepartment(dept: Omit<Department, 'id' | 'member_count'>): Promise<Department> {
    const data = await authFetch<ApiDepartmentResponse>('/api/tenant/org/departments', { method: 'POST', body: dept })
    return mapDepartment(data)
  }

  /**
   * 更新指定部门信息。
   * @param id 部门 ID
   * @param dept 要更新的字段
   */
  async function updateDepartment(id: string, dept: Partial<Department>): Promise<Department> {
    const data = await authFetch<ApiDepartmentResponse>(`/api/tenant/org/departments/${id}`, { method: 'PUT', body: dept })
    return mapDepartment(data)
  }

  /**
   * 删除指定部门（部门下有成员时后端会拦截）。
   * @param id 部门 ID
   */
  async function deleteDepartment(id: string): Promise<void> {
    await authFetch<null>(`/api/tenant/org/departments/${id}`, { method: 'DELETE' })
  }

  // ============================================================
  // 角色管理
  // ============================================================

  /** 获取当前租户所有角色列表（含系统内置角色） */
  async function listRoles(): Promise<OrgRole[]> {
    loading.value = true
    error.value = null
    try {
      const data = await authFetch<ApiRoleResponse[]>('/api/tenant/org/roles')
      roles.value = data.map(mapRole)
      return roles.value
    }
    catch (e: any) {
      error.value = e.message || '加载角色列表失败'
      console.error('[useOrgApi] listRoles failed', e)
      throw e
    }
    finally { loading.value = false }
  }

  /**
   * 创建新角色。
   * @param role 角色信息（名称、描述、页面权限列表）
   */
  async function createRole(role: Omit<OrgRole, 'id'>): Promise<OrgRole> {
    const data = await authFetch<ApiRoleResponse>('/api/tenant/org/roles', { method: 'POST', body: role })
    return mapRole(data)
  }

  /**
   * 更新指定角色信息。
   * @param id 角色 ID
   * @param role 要更新的字段
   */
  async function updateRole(id: string, role: Partial<OrgRole>): Promise<OrgRole> {
    const data = await authFetch<ApiRoleResponse>(`/api/tenant/org/roles/${id}`, { method: 'PUT', body: role })
    return mapRole(data)
  }

  /**
   * 删除指定角色（系统内置角色后端会拦截）。
   * @param id 角色 ID
   */
  async function deleteRole(id: string): Promise<void> {
    await authFetch<null>(`/api/tenant/org/roles/${id}`, { method: 'DELETE' })
  }

  // ============================================================
  // 成员管理
  // ============================================================

  /** 获取当前租户所有成员列表（含用户信息、部门、角色） */
  async function listMembers(): Promise<OrgMember[]> {
    loading.value = true
    error.value = null
    try {
      const data = await authFetch<ApiMemberResponse[]>('/api/tenant/org/members')
      members.value = data.map(mapMember)
      return members.value
    }
    catch (e: any) {
      error.value = e.message || '加载成员列表失败'
      console.error('[useOrgApi] listMembers failed', e)
      throw e
    }
    finally { loading.value = false }
  }

  /**
   * 创建新成员（同时创建系统用户账号）。
   * @param member 成员信息（用户名、显示名、邮箱、部门、角色等）
   */
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
    return mapMember(data)
  }

  /**
   * 更新指定成员信息（仅传入需要修改的字段）。
   * @param id 成员 ID
   * @param member 要更新的字段
   */
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
    return mapMember(data)
  }

  /**
   * 删除指定成员（同时禁用其系统账号）。
   * @param id 成员 ID
   */
  async function deleteMember(id: string): Promise<void> {
    await authFetch<null>(`/api/tenant/org/members/${id}`, { method: 'DELETE' })
  }

  /** 并发加载部门、角色、成员列表（页面初始化时使用） */
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
