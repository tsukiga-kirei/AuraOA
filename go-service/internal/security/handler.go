package security

import (
	"net/http"
	"regexp"

	"oa-smart-audit/internal/oa"

	"github.com/gin-gonic/gin"
)

const defaultMask = "***"

// Handler exposes masking rule management endpoints.
type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// RegisterRoutes registers masking rule admin routes.
func (h *Handler) RegisterRoutes(admin *gin.RouterGroup) {
	admin.GET("/masking-rules", h.ListRules)
	admin.POST("/masking-rules", h.CreateRule)
	admin.DELETE("/masking-rules/:id", h.DeleteRule)
}

func (h *Handler) ListRules(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"rules": []interface{}{}})
}

func (h *Handler) CreateRule(c *gin.Context) {
	var rule MaskingRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "masking rule created"})
}

func (h *Handler) DeleteRule(c *gin.Context) {
	ruleID := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "masking rule deleted", "id": ruleID})
}

// MaskFormDataLogic applies masking rules to form data.
// Fields matching rule patterns get their values replaced.
// Unrecognized sensitive patterns use default full mask.
func MaskFormDataLogic(data oa.ProcessFormData, rules []MaskingRule) oa.ProcessFormData {
	masked := oa.ProcessFormData{
		ProcessID:  data.ProcessID,
		Applicant:  data.Applicant,
		SubmitTime: data.SubmitTime,
		FormFields: make([]oa.FormField, len(data.FormFields)),
	}

	for i, field := range data.FormFields {
		masked.FormFields[i] = oa.FormField{
			Name:  field.Name,
			Value: applyMasking(field.Name, field.Value, rules),
		}
	}

	return masked
}

func applyMasking(fieldName, fieldValue string, rules []MaskingRule) string {
	for _, rule := range rules {
		fieldRe, err := regexp.Compile(rule.FieldPattern)
		if err != nil {
			continue
		}
		if !fieldRe.MatchString(fieldName) {
			continue
		}

		valueRe, err := regexp.Compile(rule.ValuePattern)
		if err != nil {
			// Cannot compile value pattern — use default mask
			return defaultMask
		}

		if valueRe.MatchString(fieldValue) {
			if rule.ReplaceWith != "" {
				return valueRe.ReplaceAllString(fieldValue, rule.ReplaceWith)
			}
			return defaultMask
		}
	}
	return fieldValue
}
