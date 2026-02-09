package history

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"oa-smart-audit/internal/auth"

	"github.com/gin-gonic/gin"
)

// Handler exposes history search and export endpoints.
type Handler struct {
	// In-memory store for demo; replace with MongoDB client
	snapshots []AuditSnapshot
}

func NewHandler() *Handler {
	return &Handler{snapshots: []AuditSnapshot{}}
}

// RegisterRoutes registers history routes.
func (h *Handler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/history/search", h.Search)
	api.GET("/history/export", h.Export)
}

func (h *Handler) Search(c *gin.Context) {
	claims := auth.GetClaims(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	query := SearchQuery{
		TenantID: claims.TenantID,
		Page:     1,
		PageSize: 20,
	}

	if t := c.Query("time_from"); t != "" {
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			query.TimeFrom = &parsed
		}
	}
	if t := c.Query("time_to"); t != "" {
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			query.TimeTo = &parsed
		}
	}
	query.Department = c.Query("department")
	query.ProcessType = c.Query("process_type")

	// TODO: query MongoDB with filters
	result := SearchResult{
		Total:     0,
		Snapshots: []AuditSnapshot{},
	}

	c.JSON(http.StatusOK, result)
}

func (h *Handler) Export(c *gin.Context) {
	format := ExportFormat(c.DefaultQuery("format", "json"))

	// TODO: query MongoDB and stream results
	snapshots := []AuditSnapshot{}

	switch format {
	case ExportCSV:
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=audit_export_%s.csv", time.Now().Format("20060102")))
		w := csv.NewWriter(c.Writer)
		_ = w.Write([]string{"snapshot_id", "tenant_id", "user_id", "process_id", "recommendation", "created_at"})
		for _, s := range snapshots {
			_ = w.Write([]string{s.ID, s.TenantID, s.UserID, s.ProcessID, s.AuditResult.Recommendation, s.CreatedAt.Format(time.RFC3339)})
		}
		w.Flush()
	default:
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=audit_export_%s.json", time.Now().Format("20060102")))
		data, _ := json.MarshalIndent(snapshots, "", "  ")
		c.Writer.Write(data)
	}
}

// AppendSnapshot adds a snapshot (append-only, no update/delete).
func (h *Handler) AppendSnapshot(snapshot AuditSnapshot) {
	h.snapshots = append(h.snapshots, snapshot)
}
