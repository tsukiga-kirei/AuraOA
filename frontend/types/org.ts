//types/org.ts — 组织管理相关类型

export interface Department {
  id: string
  name: string
  parent_id: string | null
  manager: string
  member_count: number
}

export interface OrgRole {
  id: string
  name: string
  description: string
  page_permissions: string[]
  is_system: boolean
}

export interface OrgMember {
  id: string
  name: string
  username: string
  department_id: string
  department_name: string
  role_ids: string[]
  role_names: string[]
  email: string
  phone: string
  position: string
  status: 'active' | 'disabled'
  created_at: string
}
