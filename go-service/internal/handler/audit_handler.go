package handler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
	"oa-smart-audit/go-service/internal/pkg/response"
	"oa-smart-audit/go-service/internal/repository"
	"oa-smart-audit/go-service/internal/service"
)

// AuditHandler 审核工作台相关 HTTP 请求处理。
type AuditHandler struct {
	auditService *service.AuditExecuteService
	snapshotRepo *repository.AuditProcessSnapshotRepo
	auditLogRepo *repository.AuditLogRepo
}

func NewAuditHandler(auditService *service.AuditExecuteService, snapshotRepo *repository.AuditProcessSnapshotRepo, auditLogRepo *repository.AuditLogRepo) *AuditHandler {
	return &AuditHandler{auditService: auditService, snapshotRepo: snapshotRepo, auditLogRepo: auditLogRepo}
}

// ListProcesses GET /api/audit/processes?tab=pending_ai&page=1&page_size=20&start_date=&end_date=
func (h *AuditHandler) ListProcesses(c *gin.Context) {
	if getUsername(c) == "" {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "用户信息缺失")
		return
	}

	params := parseAuditListParams(c)
	resp, err := h.auditService.ListProcessesPaged(c, params)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, resp)
}

// GetStats GET /api/audit/stats（与列表共用 start_date / end_date 时统计口径一致）
func (h *AuditHandler) GetStats(c *gin.Context) {
	if getUsername(c) == "" {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "用户信息缺失")
		return
	}
	stats, err := h.auditService.GetStatsWithParams(c, parseAuditListParams(c))
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, stats)
}

// Execute POST /api/audit/execute
func (h *AuditHandler) Execute(c *gin.Context) {
	var req service.AuditExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败: "+err.Error())
		return
	}

	result, err := h.auditService.Execute(c, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	if result.Status == model.AuditStatusPending {
		c.JSON(http.StatusAccepted, response.Response{
			Code:    0,
			Message: "accepted",
			Data:    result,
		})
		return
	}
	response.Success(c, result)
}

// GetJobStatus GET /api/audit/jobs/:id
func (h *AuditHandler) GetJobStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "任务 ID 无效")
		return
	}
	data, err := h.auditService.GetAuditJobStatus(c, id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, data)
}

// CancelJob POST /api/audit/cancel/:id
func (h *AuditHandler) CancelJob(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "任务 ID 无效")
		return
	}
	if err := h.auditService.CancelJob(c, id); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"status": "cancelled"})
}

// BatchExecute POST /api/audit/batch
func (h *AuditHandler) BatchExecute(c *gin.Context) {
	var req struct {
		Items []service.AuditExecuteRequest `json:"items" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败: "+err.Error())
		return
	}

	result, err := h.auditService.BatchExecute(c, req.Items)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, result)
}

// GetJobStream GET /api/audit/stream/:id
func (h *AuditHandler) GetJobStream(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "任务 ID 无效")
		return
	}

	ch, closeSub, err := h.auditService.SubscribeJobStream(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	defer closeSub()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			c.SSEvent("message", msg)
			c.Writer.Flush()
		}
	}
}

// GetAuditChain GET /api/audit/chain/:processId
func (h *AuditHandler) GetAuditChain(c *gin.Context) {
	processID := c.Param("processId")
	if processID == "" {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "流程ID不能为空")
		return
	}

	chain, err := h.auditService.GetAuditChain(c, processID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, chain)
}

// ListLogs GET /api/audit/logs (tenant_admin)
func (h *AuditHandler) ListLogs(c *gin.Context) {
	filter, page, pageSize := parseAuditLogQuery(c)
	items, total, err := h.auditService.ListAuditLogs(c, filter, page, pageSize)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetLogStats GET /api/audit/logs/stats (tenant_admin)
func (h *AuditHandler) GetLogStats(c *gin.Context) {
	stats, err := h.auditService.GetAuditLogStats(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, stats)
}

// ExportLogs GET /api/audit/logs/export (tenant_admin) — CSV 下载
func (h *AuditHandler) ExportLogs(c *gin.Context) {
	filter, _, _ := parseAuditLogQuery(c)
	// 导出不分页，最多 5000 条
	items, _, err := h.auditService.ListAuditLogs(c, filter, 1, 5000)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	filename := fmt.Sprintf("audit_logs_%s.csv", time.Now().Format("20060102150405"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("BOM", "\xef\xbb\xbf") // UTF-8 BOM for Excel
	c.Writer.Write([]byte("\xef\xbb\xbf"))

	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"记录ID", "流程编号", "流程标题", "操作人", "流程类型", "审核建议", "评分", "状态", "创建时间"})
	for _, item := range items {
		_ = w.Write([]string{
			item.ID.String(),
			item.ProcessID,
			item.Title,
			item.UserName,
			item.ProcessType,
			item.Recommendation,
			fmt.Sprintf("%d", item.Score),
			item.Status,
			item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	w.Flush()
}

// ── 快照数据管理页端点 ──────────────────────────────────────────────────────

// ListSnapshots GET /api/audit/snapshots
func (h *AuditHandler) ListSnapshots(c *gin.Context) {
	filter, page, pageSize := parseAuditSnapshotQuery(c)
	items, total, err := h.snapshotRepo.ListPagedWithUser(c, filter, page, pageSize)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	// 格式化时间
	type itemDTO struct {
		repository.AuditSnapshotListRow
		UpdatedAtFmt string `json:"updated_at_fmt"`
		CreatedAtFmt string `json:"created_at_fmt"`
	}
	out := make([]itemDTO, len(items))
	for i, row := range items {
		out[i] = itemDTO{
			AuditSnapshotListRow: row,
			UpdatedAtFmt:         row.UpdatedAt.Local().Format("2006/1/2 15:04"),
			CreatedAtFmt:         row.CreatedAt.Local().Format("2006/1/2 15:04"),
		}
	}
	response.Success(c, gin.H{
		"items":     out,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetSnapshotStats GET /api/audit/snapshots/stats
func (h *AuditHandler) GetSnapshotStats(c *gin.Context) {
	stats, err := h.snapshotRepo.CountStatsByRecommendation(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, stats)
}

// GetSnapshotChain GET /api/audit/snapshots/:processId/chain — 审核链详情
func (h *AuditHandler) GetSnapshotChain(c *gin.Context) {
	processID := c.Param("processId")
	if processID == "" {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "流程ID不能为空")
		return
	}
	snapshot, err := h.snapshotRepo.GetByProcessID(c, processID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	if snapshot == nil {
		response.Success(c, gin.H{"chain": []interface{}{}})
		return
	}
	var idStrs []string
	_ = json.Unmarshal(snapshot.ValidLogIDs, &idStrs)
	ids := make([]uuid.UUID, 0, len(idStrs))
	for _, s := range idStrs {
		if uid, err := uuid.Parse(s); err == nil {
			ids = append(ids, uid)
		}
	}
	chain, err := h.auditLogRepo.ListByIDsWithUserOrdered(c, ids)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"chain": chain})
}

func parseAuditSnapshotQuery(c *gin.Context) (repository.AuditSnapshotFilter, int, int) {
	filter := repository.AuditSnapshotFilter{
		Recommendation: c.Query("recommendation"),
		Keyword:        c.Query("keyword"),
		ProcessType:    c.Query("process_type"),
		Operator:       c.Query("operator"),
		Department:     c.Query("department"),
	}
	if s := c.Query("start_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			filter.StartDate = &t
		}
	}
	if s := c.Query("end_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			end := t.Add(24*time.Hour - time.Second)
			filter.EndDate = &end
		}
	}
	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "page_size", 20)
	return filter, page, pageSize
}

func parseAuditLogQuery(c *gin.Context) (repository.AuditLogFilter, int, int) {
	filter := repository.AuditLogFilter{
		StatusGroup:    c.Query("status_group"),
		Keyword:        c.Query("keyword"),
		ProcessType:    c.Query("process_type"),
		Recommendation: c.Query("recommendation"),
	}
	if s := c.Query("start_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			filter.StartDate = &t
		}
	}
	if s := c.Query("end_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			end := t.Add(24*time.Hour - time.Second)
			filter.EndDate = &end
		}
	}
	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "page_size", 20)
	return filter, page, pageSize
}

// parseAuditListParams 解析审核工作台列表与统计的 query（含 OA 提交时间 start_date、end_date）。
func parseAuditListParams(c *gin.Context) dto.AuditListParams {
	p := dto.AuditListParams{
		Tab:         c.DefaultQuery("tab", "pending_ai"),
		Keyword:     c.Query("keyword"),
		Applicant:   c.Query("applicant"),
		ProcessType: c.Query("process_type"),
		Department:  c.Query("department"),
		AuditStatus: c.Query("audit_status"),
		Page:        parseIntQuery(c, "page", 1),
		PageSize:    parseIntQuery(c, "page_size", 20),
	}
	if s := c.Query("start_date"); s != "" {
		if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
			p.SubmitDateStart = &t
		}
	}
	if s := c.Query("end_date"); s != "" {
		if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
			excl := t.AddDate(0, 0, 1)
			p.SubmitDateEndExclusive = &excl
		}
	}
	return p
}

func getUsername(c *gin.Context) string {
	claimsVal, exists := c.Get("jwt_claims")
	if !exists {
		return ""
	}
	claims, ok := claimsVal.(*jwtpkg.JWTClaims)
	if !ok {
		return ""
	}
	return claims.Username
}
