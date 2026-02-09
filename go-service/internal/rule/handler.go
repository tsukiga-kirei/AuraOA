package rule

import (
	"net/http"
	"sort"

	"oa-smart-audit/internal/auth"

	"github.com/gin-gonic/gin"
)

// Handler exposes rule engine HTTP endpoints.
type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// RegisterAdminRoutes registers rule management routes for tenant admins.
func (h *Handler) RegisterAdminRoutes(admin *gin.RouterGroup) {
	admin.GET("/rules", h.ListRules)
	admin.POST("/rules", h.CreateRule)
	admin.PUT("/rules/:id", h.UpdateRule)
	admin.DELETE("/rules/:id", h.DeleteRule)
}

// RegisterUserRoutes registers user preference routes.
func (h *Handler) RegisterUserRoutes(api *gin.RouterGroup) {
	api.GET("/preferences", h.GetPreferences)
	api.PUT("/preferences/toggle", h.ToggleRule)
	api.POST("/preferences/private-rules", h.AddPrivateRule)
	api.DELETE("/preferences/private-rules/:id", h.DeletePrivateRule)
	api.PUT("/preferences/sensitivity", h.UpdateSensitivity)
}

func (h *Handler) ListRules(c *gin.Context) {
	// TODO: query from database by tenant_id
	c.JSON(http.StatusOK, gin.H{"rules": []interface{}{}})
}

func (h *Handler) CreateRule(c *gin.Context) {
	var body struct {
		ProcessType string    `json:"process_type" binding:"required"`
		RuleContent string    `json:"rule_content" binding:"required"`
		RuleScope   RuleScope `json:"rule_scope" binding:"required"`
		Priority    int       `json:"priority"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	isLocked := body.RuleScope == RuleScopeMandatory
	c.JSON(http.StatusCreated, gin.H{
		"message":   "rule created",
		"scope":     body.RuleScope,
		"is_locked": isLocked,
	})
}

func (h *Handler) UpdateRule(c *gin.Context) {
	ruleID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "rule updated", "id": ruleID})
}

func (h *Handler) DeleteRule(c *gin.Context) {
	ruleID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "rule deleted", "id": ruleID})
}

func (h *Handler) GetPreferences(c *gin.Context) {
	claims := auth.GetClaims(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	// TODO: query user preferences, private rules, sensitivity from database
	c.JSON(http.StatusOK, gin.H{
		"user_id":       claims.UserID,
		"toggles":       []interface{}{},
		"private_rules": []interface{}{},
		"sensitivity":   "normal",
	})
}

func (h *Handler) ToggleRule(c *gin.Context) {
	var body struct {
		RuleID  string `json:"rule_id" binding:"required"`
		Enabled bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "rule toggled", "rule_id": body.RuleID, "enabled": body.Enabled})
}

func (h *Handler) AddPrivateRule(c *gin.Context) {
	var body struct {
		RuleContent string `json:"rule_content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "private rule added"})
}

func (h *Handler) DeletePrivateRule(c *gin.Context) {
	ruleID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "private rule deleted", "id": ruleID})
}

func (h *Handler) UpdateSensitivity(c *gin.Context) {
	var body struct {
		Level string `json:"level" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "sensitivity updated", "level": body.Level})
}

// MergeRulesLogic implements the priority merge: mandatory > user private > default.
func MergeRulesLogic(tenantRules []MergedRule, userPrivateRules []MergedRule, userToggles map[string]bool) []MergedRule {
	var result []MergedRule

	// 1. Add all mandatory rules (locked, always active)
	for _, r := range tenantRules {
		if r.Scope == RuleScopeMandatory {
			r.IsLocked = true
			r.Source = RuleSourceTenant
			result = append(result, r)
		}
	}

	// 2. Add user private rules
	for _, r := range userPrivateRules {
		r.Source = RuleSourceUser
		result = append(result, r)
	}

	// 3. Add default rules (respecting user toggles)
	for _, r := range tenantRules {
		if r.Scope == RuleScopeMandatory {
			continue
		}
		enabled, hasToggle := userToggles[r.ID]
		if hasToggle {
			if !enabled {
				continue // user disabled this rule
			}
		} else {
			// No user toggle: use default
			if r.Scope == RuleScopeDefaultOff {
				continue
			}
		}
		r.Source = RuleSourceTenant
		result = append(result, r)
	}

	// Sort by priority descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].Priority > result[j].Priority
	})

	return result
}
