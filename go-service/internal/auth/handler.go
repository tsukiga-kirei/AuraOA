package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler exposes auth HTTP endpoints.
type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// RegisterRoutes registers auth routes on the given router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/auth/login", h.Login)
	rg.POST("/auth/refresh", h.Refresh)
	rg.GET("/auth/menu", JWTMiddleware(), h.GetMenu)
}

// Login handles user authentication.
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// TODO: validate against database
	// For now, placeholder that generates a token for demo
	token, err := GenerateTokenPair(req.Username, req.TenantID, RoleUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, token)
}

// Refresh handles token refresh.
func (h *Handler) Refresh(c *gin.Context) {
	var body struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	claims, err := ParseToken(body.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired refresh token"})
		return
	}

	token, err := GenerateTokenPair(claims.UserID, claims.TenantID, claims.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, token)
}

// GetMenu returns dynamic menu based on user role.
func (h *Handler) GetMenu(c *gin.Context) {
	claims := GetClaims(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	menus := getMenusByRole(claims.Role)
	c.JSON(http.StatusOK, gin.H{"menus": menus})
}

// getMenusByRole returns menu items filtered by role.
func getMenusByRole(role Role) []MenuItem {
	allMenus := []MenuItem{
		{Key: "dashboard", Label: "审核工作台", Icon: "DashboardOutlined", Path: "/dashboard"},
		{Key: "cron", Label: "定时任务", Icon: "ClockCircleOutlined", Path: "/cron"},
		{Key: "archive", Label: "归档复盘", Icon: "FolderOpenOutlined", Path: "/archive"},
	}

	adminMenus := []MenuItem{
		{Key: "admin-tenant", Label: "租户配置", Icon: "SettingOutlined", Path: "/admin/tenant"},
	}

	sysAdminMenus := []MenuItem{
		{Key: "admin-system", Label: "系统管理", Icon: "SettingOutlined", Path: "/admin/system"},
		{Key: "admin-monitor", Label: "全局监控", Icon: "MonitorOutlined", Path: "/admin/monitor"},
	}

	switch role {
	case RoleAdmin:
		return append(append(allMenus, adminMenus...), sysAdminMenus...)
	case RoleTenantAdmin:
		return append(allMenus, adminMenus...)
	default:
		return allMenus
	}
}
