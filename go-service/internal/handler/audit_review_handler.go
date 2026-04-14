// 审核工作台处理器，负责审核任务执行、状态查询、日志管理及快照数据管理。
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

// AuditHandler 处理审核工作台相关的 HTTP 请求。
type AuditHandler struct {
	auditService *service.AuditExecuteService
	snapshotRepo *repository.AuditProcessSnapshotRepo
	auditLogRepo *repository.AuditLogRepo
}

// NewAuditHandler 创建审核工作台处理器实例。
func NewAuditHandler(auditService *service.AuditExecuteService, snapshotRepo *repository.AuditProcessSnapshotRepo, auditLogRepo *repository.AuditLogRepo) *AuditHandler {
	return &AuditHandler{auditService: auditService, snapshotRepo: snapshotRepo, auditLogRepo: auditLogRepo}
}

// ListProcesses 分页查询审核工作台流程列表，支持多维度过滤。
// GET /api/audit/processes
// 查询参数：tab（pending_ai/等）、keyword、applicant、process_type、department、audit_status、start_date、end_date、page、page_size
// 返回：分页结果（items + total）。
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

// GetStats 获取审核工作台统计数据，与列表共用相同过滤条件。
// GET /api/audit/stats
// 查询参数：同 ListProcesses
// 返回：各状态计数统计对象。
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

// Execute 对单条流程发起 AI 审核任务。
// POST /api/audit/execute
// 请求体：AuditExecuteRequest（流程 ID、配置参数等）
// 返回：任务结果；若为异步任务则返回 202 Accepted + 任务状态。
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
	// 异步任务返回 202，前端轮询状态
	if result.Status == model.JobStatusPending {
		c.JSON(http.StatusAccepted, response.Response{
			Code:    0,
			Message: "accepted",
			Data:    result,
		})
		return
	}
	response.Success(c, result)
}

// GetJobStatus 查询审核异步任务的当前状态。
// GET /api/audit/jobs/:id
// 路径参数：id（任务 UUID）
// 返回：任务状态及结果数据。
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

// CancelJob 取消指定审核异步任务。
// POST /api/audit/cancel/:id
// 路径参数：id（任务 UUID）
// 返回：{"status": "cancelled"}。
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

// BatchExecute 批量对多条流程发起 AI 审核任务。
// POST /api/audit/batch
// 请求体：{"items": [...AuditExecuteRequest]}
// 返回：批量任务结果汇总。
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

// GetJobStream 以 SSE 方式推送审核任务的实时流式输出。
// GET /api/audit/stream/:id
// 路径参数：id（任务 UUID）
// 返回：text/event-stream 格式的流式消息。
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

// GetAuditChain 获取指定流程的完整审核链（所有历史审核记录）。
// GET /api/audit/chain/:processId
// 路径参数：processId（OA 流程编号）
// 返回：审核链记录数组，按时间排序。
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

// ListLogs 分页查询审核日志（租户管理员数据管理页）。
// GET /api/audit/logs
// 查询参数：status_group、keyword、process_type、recommendation、start_date、end_date、page、page_size
// 返回：分页日志列表（items + total + page + page_size）。
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

// GetLogStats 获取审核日志的统计汇总（租户管理员数据管理页）。
// GET /api/audit/logs/stats
// 返回：按审核建议分类的统计数据。
func (h *AuditHandler) GetLogStats(c *gin.Context) {
	stats, err := h.auditService.GetAuditLogStats(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, stats)
}

// ExportLogs 导出审核日志为 CSV 文件（最多 5000 条）。
// GET /api/audit/logs/export
// 查询参数：同 ListLogs（不分页）
// 返回：text/csv 格式文件下载，含 UTF-8 BOM 以兼容 Excel。
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
	// 写入 UTF-8 BOM，确保 Excel 正确识别中文
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

// ListSnapshots 分页查询审核流程快照列表（租户管理员数据管理页）。
// GET /api/audit/snapshots
// 查询参数：recommendation、keyword、process_type、operator、department、start_date、end_date、page、page_size
// 返回：分页快照列表，时间字段附带格式化字符串。
func (h *AuditHandler) ListSnapshots(c *gin.Context) {
	filter, page, pageSize := parseAuditSnapshotQuery(c)
	items, total, err := h.snapshotRepo.ListPagedWithUser(c, filter, page, pageSize)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	// 附加格式化时间字段，方便前端直接展示
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

// GetSnapshotStats 获取审核快照按审核建议分类的统计数据。
// GET /api/audit/snapshots/stats
// 返回：各审核建议状态的快照数量统计。
func (h *AuditHandler) GetSnapshotStats(c *gin.Context) {
	stats, err := h.snapshotRepo.CountStatsByRecommendation(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, stats)
}

// GetSnapshotChain 获取指定流程的审核链详情（所有有效审核记录按时间排序）。
// GET /api/audit/snapshots/:processId/chain
// 路径参数：processId（OA 流程编号）
// 返回：{"chain": [...]} 审核链记录数组。
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
	// 解析快照中存储的有效审核记录 ID 列表
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

// parseAuditSnapshotQuery 解析审核快照列表的过滤参数及分页参数。
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

// parseAuditLogQuery 解析审核日志列表的过滤参数及分页参数。
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

// parseAuditListParams 解析审核工作台列表与统计的公共查询参数。
// start_date、end_date 格式为 YYYY-MM-DD，按 OA 提交时间过滤。
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
			// 结束日期取次日零点（不含），实现闭区间查询
			excl := t.AddDate(0, 0, 1)
			p.SubmitDateEndExclusive = &excl
		}
	}
	return p
}

// getUsername 从 gin.Context 的 JWT claims 中提取用户名。
// 若 claims 不存在或类型断言失败则返回空字符串。
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
