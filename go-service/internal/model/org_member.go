package model

import (
	"time"

	"github.com/google/uuid"
)

//OrgMember 代表租户组织内的成员，将用户链接到部门和角色。
type OrgMember struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID     uuid.UUID  `gorm:"type:uuid;not null;index"`
	UserID       uuid.UUID  `gorm:"type:uuid;not null;index"`
	DepartmentID uuid.UUID  `gorm:"type:uuid;not null;index"`
	Position     string     `gorm:"size:100"`
	Status       string     `gorm:"size:20;not null;default:active"` //活动|禁用
	CreatedAt    time.Time
	UpdatedAt    time.Time

	//协会
	User       User       `gorm:"foreignKey:UserID"`
	Department Department `gorm:"foreignKey:DepartmentID"`
	Roles      []OrgRole  `gorm:"many2many:org_member_roles"`
}
