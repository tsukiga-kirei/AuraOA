package auth

import "context"

type MenuItem struct {
	Key      string     `json:"key"`
	Label    string     `json:"label"`
	Icon     string     `json:"icon,omitempty"`
	Path     string     `json:"path"`
	Children []MenuItem `json:"children,omitempty"`
}

// RBACService handles role-based access control.
type RBACService interface {
	GetUserMenus(ctx context.Context, userID string, role Role) ([]MenuItem, error)
	CheckPermission(ctx context.Context, userID string, resource string, action string) (bool, error)
}
