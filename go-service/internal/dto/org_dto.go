package dto

//--- 部门 DTO ---

//CreateDepartmentRequest 是 POST /api/tenant/org/departments 的请求正文
type CreateDepartmentRequest struct {
	Name      string  `json:"name" binding:"required"`
	ParentID  *string `json:"parent_id"`
	Manager   string  `json:"manager"`
	SortOrder int     `json:"sort_order"`
}

//UpdateDepartmentRequest 是 PUT /api/tenant/org/departments/:id 的请求正文
type UpdateDepartmentRequest struct {
	Name      string  `json:"name"`
	ParentID  *string `json:"parent_id"`
	Manager   string  `json:"manager"`
	SortOrder int     `json:"sort_order"`
}

//DepartmentResponse 是部门端点的响应正文
type DepartmentResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	ParentID    *string `json:"parent_id"`
	Manager     string  `json:"manager"`
	SortOrder   int     `json:"sort_order"`
	MemberCount int64   `json:"member_count"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

//--- OrgRole DTO ---

//CreateRoleRequest 是 POST /api/tenant/org/roles 的请求正文
type CreateRoleRequest struct {
	Name            string      `json:"name" binding:"required"`
	Description     string      `json:"description"`
	PagePermissions interface{} `json:"page_permissions"` //JSON数组
}

//UpdateRoleRequest 是 PUT /api/tenant/org/roles/:id 的请求正文
type UpdateRoleRequest struct {
	Name            string      `json:"name"`
	Description     string      `json:"description"`
	PagePermissions interface{} `json:"page_permissions"`
}

//RoleResponse 是组织角色端点的响应正文
type RoleResponse struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	Description     string      `json:"description"`
	PagePermissions interface{} `json:"page_permissions"`
	IsSystem        bool        `json:"is_system"`
	CreatedAt       string      `json:"created_at"`
	UpdatedAt       string      `json:"updated_at"`
}

//--- 成员 DTO ---

//CreateMemberRequest 是 POST /api/tenant/org/members 的请求正文
type CreateMemberRequest struct {
	Username     string   `json:"username" binding:"required"`
	DisplayName  string   `json:"display_name" binding:"required"`
	Password     string   `json:"password"`
	Email        string   `json:"email"`
	Phone        string   `json:"phone"`
	DepartmentID string   `json:"department_id" binding:"required"`
	RoleIDs      []string `json:"role_ids" binding:"required"`
	Position     string   `json:"position"`
}

//UpdateMemberRequest 是 PUT /api/tenant/org/members/:id 的请求正文
type UpdateMemberRequest struct {
	DisplayName  string   `json:"display_name"`
	Email        string   `json:"email"`
	Phone        string   `json:"phone"`
	DepartmentID string   `json:"department_id"`
	RoleIDs      []string `json:"role_ids"`
	Position     string   `json:"position"`
	Status       string   `json:"status"`
}

//MemberResponse 是成员端点的响应正文
type MemberResponse struct {
	ID         string             `json:"id"`
	User       MemberUserInfo     `json:"user"`
	Department DepartmentResponse `json:"department"`
	Roles      []RoleResponse     `json:"roles"`
	Position   string             `json:"position"`
	Status     string             `json:"status"`
	CreatedAt  string             `json:"created_at"`
	UpdatedAt  string             `json:"updated_at"`
}

//MemberUserInfo 包含嵌入在会员响应中的用户详细信息
type MemberUserInfo struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	AvatarURL   string `json:"avatar_url"`
}
