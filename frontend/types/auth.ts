//types/auth.ts — 认证相关类型

export type UserRole = 'business' | 'tenant_admin' | 'system_admin'
export type PermissionGroup = 'business' | 'tenant_admin' | 'system_admin'

export interface RoleInfo {
  id: string
  role: UserRole
  tenant_id: string | null
  tenant_name: string | null
  label: string
}

export interface LoginRequest {
  username: string
  password: string
  tenant_id: string
  preferred_role?: UserRole
}

export interface LoginResponse {
  access_token: string
  refresh_token: string
  user: {
    id: string
    username: string
    display_name: string
    email: string
    phone: string
    avatar_url: string
    locale: string
  }
  roles: RoleInfo[]
  active_role: RoleInfo
  permissions: string[]
}

export interface SwitchRoleResponse {
  access_token: string
  active_role: RoleInfo
  permissions: string[]
  menus: MenuItem[]
}

export interface MenuItem {
  key: string
  label: string
  icon?: string
  path: string
  children?: MenuItem[]
}

export interface TenantOption {
  id: string
  name: string
  code: string
}
