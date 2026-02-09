package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"oa-smart-audit/internal/auth"
	"oa-smart-audit/internal/oa"
	"oa-smart-audit/internal/rule"
	"oa-smart-audit/internal/security"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler orchestrates the full audit flow.
type Handler struct {
	aiServiceURL string
	httpClient   *http.Client
}

func NewHandler() *Handler {
	url := os.Getenv("AI_SERVICE_URL")
	if url == "" {
		url = "http://localhost:8000"
	}
	return &Handler{
		aiServiceURL: url,
		httpClient:   &http.Client{Timeout: 60 * time.Second},
	}
}

// RegisterRoutes registers audit endpoints.
func (h *Handler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/audit/todo", h.GetTodoList)
	api.POST("/audit/execute", h.ExecuteAudit)
	api.POST("/audit/feedback", h.SubmitFeedback)
}

// ExecuteAuditRequest is the frontend request payload.
type ExecuteAuditRequest struct {
	ProcessID string `json:"process_id" binding:"required"`
}

// GetTodoList returns pending processes for the current user.
func (h *Handler) GetTodoList(c *gin.Context) {
	// TODO: call OA adapter to fetch todo processes
	c.JSON(http.StatusOK, gin.H{"processes": []oa.OAProcess{}})
}

// ExecuteAudit orchestrates: load rules → mask data → call AI → save snapshot.
func (h *Handler) ExecuteAudit(c *gin.Context) {
	var req ExecuteAuditRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	claims := auth.GetClaims(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	traceID := uuid.New().String()
	log.Printf("[Trace: %s] Audit started for process %s by user %s", traceID, req.ProcessID, claims.UserID)

	// 1. Fetch process form data from OA (placeholder)
	formData := oa.ProcessFormData{
		ProcessID:  req.ProcessID,
		FormFields: []oa.FormField{},
	}

	// 2. Load and merge rules
	// TODO: load from database, for now use empty
	tenantRules := []rule.MergedRule{}
	userPrivateRules := []rule.MergedRule{}
	userToggles := map[string]bool{}
	mergedRules := rule.MergeRulesLogic(tenantRules, userPrivateRules, userToggles)

	// 3. Mask sensitive data
	maskingRules := []security.MaskingRule{}
	maskedData := security.MaskFormDataLogic(formData, maskingRules)

	// 4. Call Python AI service
	startTime := time.Now()
	aiResp, err := h.callAIService(traceID, maskedData, mergedRules)
	duration := time.Since(startTime)
	log.Printf("[Trace: %s] AI service responded in %v", traceID, duration)

	if err != nil {
		log.Printf("[Trace: %s] AI service error: %v", traceID, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "AI service unavailable", "trace_id": traceID})
		return
	}

	// 5. TODO: Save audit snapshot to MongoDB (append-only)
	// 6. TODO: Increment token_used for tenant

	c.JSON(http.StatusOK, gin.H{
		"trace_id":       traceID,
		"process_id":     req.ProcessID,
		"recommendation": aiResp.Recommendation,
		"details":        aiResp.Details,
		"ai_reasoning":   aiResp.AIReasoning,
	})
}

// SubmitFeedback records user's adoption decision.
func (h *Handler) SubmitFeedback(c *gin.Context) {
	var body struct {
		ProcessID   string `json:"process_id" binding:"required"`
		Adopted     bool   `json:"adopted"`
		ActionTaken string `json:"action_taken"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	// TODO: update snapshot in MongoDB with user feedback
	c.JSON(http.StatusOK, gin.H{"message": "feedback recorded"})
}

// AIAuditResponse mirrors the Python service response.
type AIAuditResponse struct {
	Recommendation string          `json:"recommendation"`
	Details        json.RawMessage `json:"details"`
	AIReasoning    string          `json:"ai_reasoning"`
	TraceID        string          `json:"trace_id"`
}

func (h *Handler) callAIService(traceID string, formData oa.ProcessFormData, rules []rule.MergedRule) (*AIAuditResponse, error) {
	payload := map[string]interface{}{
		"form_data": map[string]interface{}{
			"process_id":  formData.ProcessID,
			"form_fields": formData.FormFields,
		},
		"rules":     rules,
		"kb_mode":   "rules_only",
		"ai_config": map[string]interface{}{"model_provider": "local", "model_name": "default", "context_window_size": 4096},
		"trace_id":  traceID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", h.aiServiceURL+"/api/audit", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Trace-ID", traceID)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call AI service: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI service returned %d: %s", resp.StatusCode, string(respBody))
	}

	var aiResp AIAuditResponse
	if err := json.Unmarshal(respBody, &aiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &aiResp, nil
}
