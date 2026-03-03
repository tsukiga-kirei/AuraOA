package model

import (
	"time"

	"github.com/google/uuid"
)

// OrgMember represents a member within a tenant's organization, linking a user to a department and roles.
type OrgMember struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID     uuid.UUID  `gorm:"type:uuid;not null;index"`
	UserID       uuid.UUID  `gorm:"type:uuid;not null;index"`
	DepartmentID uuid.UUID  `gorm:"type:uuid;not null;index"`
	Position     string     `gorm:"size:100"`
	Status       string     `gorm:"size:20;not null;default:active"` // active|disabled
	CreatedAt    time.Time
	UpdatedAt    time.Time

	// Associations
	User       User       `gorm:"foreignKey:UserID"`
	Department Department `gorm:"foreignKey:DepartmentID"`
	Roles      []OrgRole  `gorm:"many2many:org_member_roles"`
}
