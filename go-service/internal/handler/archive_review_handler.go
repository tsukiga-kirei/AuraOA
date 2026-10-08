// 归档复盘运行时处理器，负责归档任务执行、状态查询、日志管理及快照数据管理。
package handler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"

	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
	"auraoa/go-service/internal/pkg/errcode"
	excelpkg "auraoa/go-service/internal/pkg/excel"
	"auraoa/go-service/internal/pkg/response"
	"auraoa/go-service/internal/repository"
	"auraoa/go-service/internal/service"
)

// ArchiveReviewHandler 处理归档复盘运行时相关的 HTTP 请求。
type ArchiveReviewHandler struct {
	archiveService *service.ArchiveReviewService
	snapshotRepo   *repository.ArchiveProcessSnapshotRepo
	archiveLogRepo *repository.ArchiveLogRepo
}

// NewArchiveReviewHandler 创建归档复盘运行时处理器实例。
func NewArchiveReviewHandler(archiveService *service.ArchiveReviewService, snapshotRepo *repository.ArchiveProcessSnapshotRepo, archiveLogRepo *repository.ArchiveLogRepo) *ArchiveReviewHandler {
	return &ArchiveReviewHandler{archiveService: archiveService, snapshotRepo: snapshotRepo, archiveLogRepo: archiveLogRepo}
}

// ListProcesses 分页查询归档流程列表。
// GET /api/archive/processes
// 查询参数：keyword、applicant、process_type、department、audit_status、start_date、end_date、page、page_size
// 返回：分页结果（items + total）。
func (h *ArchiveReviewHandler) ListProcesses(c *gin.Context) {
	params := parseArchiveListParams(c)
	resp, err := h.archiveService.ListProcessesPaged(c, params)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, resp)
}

// ExportProcesses 按当前页签和筛选条件导出全量归档流程为 Excel 文件。
// GET /api/archive/processes/export
// 查询参数：同 ListProcesses（不含分页参数）
// 响应：xlsx 文件流，文件名含页签名和时间戳，支持 UTF-8 编码。
func (h *ArchiveReviewHandler) ExportProcesses(c *gin.Context) {
	params := parseArchiveListParams(c)
	locale := excelpkg.ResolveLocale(c)

	records, err := h.archiveService.ListAllProcesses(c, params)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "导出失败: "+err.Error())
		return
	}

	auditStatus := c.Query("audit_status")

	var exportType excelpkg.ExportType
	if auditStatus == "unaudited" || auditStatus == "" {
		exportType = excelpkg.ExportTypeArchiveUnaudited
	} else {
		exportType = excelpkg.ExportTypeArchiveReviewed
	}

	rows := make([][]string, 0, len(records))
	for _, rec := range records {
		getString := func(key string) string {
			if v, ok := rec[key]; ok && v != nil {
				return fmt.Sprintf("%v", v)
			}
			return ""
		}

		row := []string{
			getString("process_id"),
			getString("title"),
			getString("applicant"),
			getString("department"),
			getString("process_type"),
			getString("archive_time"),
		}

		if exportType == excelpkg.ExportTypeArchiveReviewed {
			compliance := getString("snapshot_compliance")
			complianceText := excelpkg.TranslateEnum(excelpkg.EnumCompliance, compliance, locale)

			score := ""
			confidence := ""
			reviewTime := ""
			if result, ok := rec["archive_result"].(map[string]interface{}); ok && result != nil {
				if v := result["overall_score"]; v != nil {
					score = fmt.Sprintf("%v", v)
				}
				if v := result["confidence"]; v != nil {
					confidence = fmt.Sprintf("%v", v)
				}
				if v := result["created_at"]; v != nil {
					reviewTime = fmt.Sprintf("%v", v)
				}
			}

			row = append(row, complianceText, score, confidence, reviewTime)
		}

		rows = append(rows, row)
	}

	timestamp := apptime.Now().Format("20060102_150405")
	filename := fmt.Sprintf("archive_%s_%s", auditStatus, timestamp)

	config := excelpkg.ExportConfig{
		ExportType: exportType,
		Locale:     locale,
		Filename:   filename,
	}

	if err := excelpkg.WriteExcel(c, config, rows); err != nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "生成 Excel 失败: "+err.Error())
	}
}

// GetStats 获取归档流程统计数据（与列表共用相同过滤条件）。
// GET /api/archive/stats
// 查询参数：同 ListProcesses
// 返回：各状态计数统计对象。
func (h *ArchiveReviewHandler) GetStats(c *gin.Context) {
	stats, err := h.archiveService.GetStats(c, parseArchiveListParams(c))
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, stats)
}

// Execute 对单条归档流程发起 AI 复盘任务。
// POST /api/archive/execute
// 请求体：ArchiveReviewExecuteRequest（流程 ID、配置 ID 等）
// 返回：任务结果；若为异步任务则返回 202 Accepted + 任务状态。
func (h *ArchiveReviewHandler) Execute(c *gin.Context) {
	var req dto.ArchiveReviewExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败: "+err.Error())
		return
	}

	result, err := h.archiveService.Execute(c, &req)
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

// BatchExecute 批量对多条归档流程发起 AI 复盘任务。
// POST /api/archive/batch
// 请求体：ArchiveBatchExecuteRequest（items 数组）
// 返回：批量任务结果汇总。
func (h *ArchiveReviewHandler) BatchExecute(c *gin.Context) {
	var req dto.ArchiveBatchExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败: "+err.Error())
		return
	}
	result, err := h.archiveService.BatchExecute(c, req.Items)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, result)
}

// CancelJob 取消指定归档复盘异步任务。
// POST /api/archive/cancel/:id
// 路径参数：id（任务 UUID）
// 返回：{"status": "cancelled"}。
func (h *ArchiveReviewHandler) CancelJob(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "任务 ID 无效")
		return
	}
	if err := h.archiveService.CancelJob(c, id); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"status": "cancelled"})
}

// GetJobStatus 查询归档复盘异步任务的当前状态。
// GET /api/archive/jobs/:id
// 路径参数：id（任务 UUID）
// 返回：任务状态及结果数据。
func (h *ArchiveReviewHandler) GetJobStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "任务 ID 无效")
		return
	}
	data, err := h.archiveService.GetArchiveJobStatus(c, id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, data)
}

// GetJobStream 以 SSE 方式推送归档复盘任务的实时流式输出。
// GET /api/archive/stream/:id
// 路径参数：id（任务 UUID）
// 返回：text/event-stream 格式的流式消息。
func (h *ArchiveReviewHandler) GetJobStream(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "任务 ID 无效")
		return
	}

	ch, closeSub, err := h.archiveService.SubscribeJobStream(c, id)
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
			c.SSEvent(msg.Event, msg.Data)
			c.Writer.Flush()
		}
	}
}

// GetHistory 获取指定流程的归档复盘历史记录列表。
// GET /api/archive/history/:processId
// 路径参数：processId（OA 流程编号）
// 返回：该流程的所有历史复盘记录。
func (h *ArchiveReviewHandler) GetHistory(c *gin.Context) {
	processID := c.Param("processId")
	if processID == "" {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "流程ID不能为空")
		return
	}
	items, err := h.archiveService.GetArchiveHistory(c, processID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, items)
}

// GetResult 获取单条归档复盘记录的详细结果。
// GET /api/archive/results/:id
// 路径参数：id（记录 UUID）
// 返回：复盘结果详情（AI 分析内容、合规性评分等）。
func (h *ArchiveReviewHandler) GetResult(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "记录 ID 无效")
		return
	}
	data, err := h.archiveService.GetArchiveResult(c, id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, data)
}

// ListLogs 分页查询归档复盘日志（租户管理员数据管理页）。
// GET /api/archive/logs
// 查询参数：keyword、process_type、compliance、start_date、end_date、page、page_size
// 返回：分页日志列表（items + total + page + page_size）。
func (h *ArchiveReviewHandler) ListLogs(c *gin.Context) {
	filter, page, pageSize := parseArchiveLogQuery(c)
	items, total, err := h.archiveService.ListArchiveLogs(c, filter, page, pageSize)
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

// GetLogStats 获取归档复盘日志的统计汇总（租户管理员数据管理页）。
// GET /api/archive/logs/stats
// 返回：按合规性分类的统计数据。
func (h *ArchiveReviewHandler) GetLogStats(c *gin.Context) {
	stats, err := h.archiveService.GetArchiveLogStats(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, stats)
}

// ExportLogs 导出归档复盘日志为 CSV 文件（最多 5000 条）。
// GET /api/archive/logs/export
// 查询参数：同 ListLogs（不分页）
// 返回：text/csv 格式文件下载，含 UTF-8 BOM 以兼容 Excel。
func (h *ArchiveReviewHandler) ExportLogs(c *gin.Context) {
	filter, _, _ := parseArchiveLogQuery(c)
	items, _, err := h.archiveService.ListArchiveLogs(c, filter, 1, 5000)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	filename := fmt.Sprintf("archive_logs_%s.csv", apptime.Now().Format("20060102150405"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	// 写入 UTF-8 BOM，确保 Excel 正确识别中文
	c.Writer.Write([]byte("\xef\xbb\xbf"))

	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"记录ID", "流程编号", "流程标题", "操作人", "流程类型", "合规性", "评分", "状态", "创建时间"})
	for _, item := range items {
		_ = w.Write([]string{
			item.ID.String(),
			item.ProcessID,
			item.Title,
			item.UserName,
			item.ProcessType,
			item.Compliance,
			fmt.Sprintf("%d", item.ComplianceScore),
			item.Status,
			item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	w.Flush()
}

// parseArchiveListParams 解析归档列表与统计请求的公共查询参数。
// start_date、end_date 格式为 YYYY-MM-DD，按归档时间过滤 OA 数据。
func parseArchiveListParams(c *gin.Context) dto.ArchiveListParams {
	params := dto.ArchiveListParams{
		Keyword:     c.Query("keyword"),
		Applicant:   c.Query("applicant"),
		ProcessType: c.Query("process_type"),
		Department:  c.Query("department"),
		AuditStatus: c.Query("audit_status"),
		Page:        parseIntQuery(c, "page", 1),
		PageSize:    parseIntQuery(c, "page_size", 20),
	}
	if s := c.Query("start_date"); s != "" {
		if t, err := time.ParseInLocation("2006-01-02", s, apptime.Location()); err == nil {
			params.ArchiveDateStart = &t
		}
	}
	if s := c.Query("end_date"); s != "" {
		if t, err := time.ParseInLocation("2006-01-02", s, apptime.Location()); err == nil {
			// 结束日期取次日零点（不含），实现闭区间查询
			excl := t.AddDate(0, 0, 1)
			params.ArchiveDateEndExclusive = &excl
		}
	}
	return params
}

// parseArchiveLogQuery 解析归档日志列表的过滤参数及分页参数。
func parseArchiveLogQuery(c *gin.Context) (repository.ArchiveLogFilter, int, int) {
	filter := repository.ArchiveLogFilter{
		Keyword:     c.Query("keyword"),
		ProcessType: c.Query("process_type"),
		Compliance:  c.Query("compliance"),
	}
	if s := c.Query("start_date"); s != "" {
		if t, err := time.ParseInLocation("2006-01-02", s, apptime.Location()); err == nil {
			filter.StartDate = &t
		}
	}
	if s := c.Query("end_date"); s != "" {
		if t, err := time.ParseInLocation("2006-01-02", s, apptime.Location()); err == nil {
			// 结束日期使用次日零点的开区间，包含当天全部精度的记录。
			end := t.AddDate(0, 0, 1)
			filter.EndDateExclusive = &end
		}
	}
	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "page_size", 20)
	return filter, page, pageSize
}

// ── 归档快照数据管理页端点 ──────────────────────────────────────────────────

// ListSnapshots 分页查询归档流程快照列表（租户管理员数据管理页）。
// GET /api/archive/snapshots
// 查询参数：compliance、keyword、process_type、operator、department、start_date、end_date、page、page_size
// 返回：分页快照列表，时间字段附带格式化字符串。
func (h *ArchiveReviewHandler) ListSnapshots(c *gin.Context) {
	filter, page, pageSize := parseArchiveSnapshotQuery(c)
	items, total, err := h.snapshotRepo.ListPagedWithUser(c, filter, page, pageSize)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	pids := make([]string, len(items))
	for i, row := range items {
		pids[i] = row.ProcessID
	}
	var wfCodes map[string]string
	if tenantID, err := getTenantID(c); err == nil {
		wfCodes = h.archiveService.FetchWorkflowCodes(c.Request.Context(), tenantID, pids)
	}

	type itemDTO struct {
		repository.ArchiveSnapshotListRow
		WorkflowCode string `json:"workflow_code"`
		UpdatedAtFmt string `json:"updated_at_fmt"`
		CreatedAtFmt string `json:"created_at_fmt"`
	}
	out := make([]itemDTO, len(items))
	for i, row := range items {
		wfCode := ""
		if wfCodes != nil {
			wfCode = wfCodes[row.ProcessID]
		}
		out[i] = itemDTO{
			ArchiveSnapshotListRow: row,
			WorkflowCode:           wfCode,
			UpdatedAtFmt:           row.UpdatedAt.Local().Format("2006/1/2 15:04"),
			CreatedAtFmt:           row.CreatedAt.Local().Format("2006/1/2 15:04"),
		}
	}
	response.Success(c, gin.H{
		"items":     out,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetSnapshotStats 获取归档快照按合规性分类的统计数据。
// GET /api/archive/snapshots/stats
// 返回：各合规性状态的快照数量统计。
func (h *ArchiveReviewHandler) GetSnapshotStats(c *gin.Context) {
	stats, err := h.snapshotRepo.CountStatsByCompliance(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, stats)
}

// ExportSnapshots 导出归档快照为 Excel（最多 5000 条，筛选条件同列表）。
// GET /api/archive/snapshots/export
// 支持 columns 指定导出展示列，支持 include_history=true 导出复核历史。
func (h *ArchiveReviewHandler) ExportSnapshots(c *gin.Context) {
	filter, _, _ := parseArchiveSnapshotQuery(c)
	items, _, err := h.snapshotRepo.ListPagedWithUser(c, filter, 1, 5000)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	locale := excelpkg.ResolveLocale(c)
	tenantID, _ := getTenantID(c)

	columnsParam := strings.TrimSpace(c.Query("columns"))
	var selectedCols []string
	if columnsParam != "" {
		for _, col := range strings.Split(columnsParam, ",") {
			col = strings.TrimSpace(col)
			if col != "" {
				selectedCols = append(selectedCols, col)
			}
		}
	}
	if len(selectedCols) == 0 {
		selectedCols = []string{"process_id", "title", "operator", "department", "process_type", "compliance", "audit_count", "time"}
	}

	needWorkflowCode := false
	for _, col := range selectedCols {
		if col == "workflow_code" {
			needWorkflowCode = true
			break
		}
	}

	pids := make([]string, len(items))
	for i, item := range items {
		pids[i] = item.ProcessID
	}

	var wfCodes map[string]string
	if needWorkflowCode && tenantID != uuid.Nil {
		wfCodes = h.archiveService.FetchWorkflowCodes(c.Request.Context(), tenantID, pids)
	}

	includeHistory := c.Query("include_history") == "true" || c.Query("include_history") == "1"
	historyMap := make(map[string]string)
	if includeHistory && len(pids) > 0 {
		historyMap = h.buildArchiveHistoryMap(c, pids, locale)
	}

	headers := make([]string, 0)
	for _, col := range selectedCols {
		switch col {
		case "process_id":
			if locale == excelpkg.LocaleEN {
				headers = append(headers, "Process Request ID")
			} else {
				headers = append(headers, "流程请求 ID")
			}
		case "workflow_code":
			if locale == excelpkg.LocaleEN {
				headers = append(headers, "Process Number")
			} else {
				headers = append(headers, "流程编号")
			}
		case "title":
			if locale == excelpkg.LocaleEN {
				headers = append(headers, "Title")
			} else {
				headers = append(headers, "流程标题")
			}
		case "operator":
			if locale == excelpkg.LocaleEN {
				headers = append(headers, "Operator")
			} else {
				headers = append(headers, "操作人")
			}
		case "department":
			if locale == excelpkg.LocaleEN {
				headers = append(headers, "Department")
			} else {
				headers = append(headers, "部门")
			}
		case "process_type":
			if locale == excelpkg.LocaleEN {
				headers = append(headers, "Process Type")
			} else {
				headers = append(headers, "流程分类")
			}
		case "compliance", "ai_result":
			// AI 复核结果关联 3 个字段：合规性、合规评分、置信度
			if locale == excelpkg.LocaleEN {
				headers = append(headers, "Compliance", "Score", "Confidence")
			} else {
				headers = append(headers, "合规性", "合规评分", "置信度")
			}
		case "audit_count":
			if locale == excelpkg.LocaleEN {
				headers = append(headers, "Review Count")
			} else {
				headers = append(headers, "复核次数")
			}
		case "time":
			if locale == excelpkg.LocaleEN {
				headers = append(headers, "Time")
			} else {
				headers = append(headers, "时间")
			}
		}
	}
	if includeHistory {
		if locale == excelpkg.LocaleEN {
			headers = append(headers, "Review History")
		} else {
			headers = append(headers, "复核历史")
		}
	}

	rows := make([][]string, 0, len(items))
	for _, item := range items {
		row := make([]string, 0, len(headers))
		for _, col := range selectedCols {
			switch col {
			case "process_id":
				row = append(row, item.ProcessID)
			case "workflow_code":
				wf := ""
				if wfCodes != nil {
					wf = wfCodes[item.ProcessID]
				}
				row = append(row, wf)
			case "title":
				row = append(row, item.Title)
			case "operator":
				row = append(row, item.Operator)
			case "department":
				row = append(row, item.Department)
			case "process_type":
				row = append(row, item.ProcessType)
			case "compliance", "ai_result":
				compText := excelpkg.TranslateEnum(excelpkg.EnumCompliance, item.Compliance, locale)
				scoreText := fmt.Sprintf("%d", item.ComplianceScore)
				confText := fmt.Sprintf("%d%%", item.Confidence)
				row = append(row, compText, scoreText, confText)
			case "audit_count":
				row = append(row, fmt.Sprintf("%d", countArchiveSnapshotLogIDs(item.ValidArchiveLogIDs)))
			case "time":
				row = append(row, item.UpdatedAt.Local().Format("2006-01-02 15:04:05"))
			}
		}
		if includeHistory {
			row = append(row, historyMap[item.ProcessID])
		}
		rows = append(rows, row)
	}

	filename := fmt.Sprintf("archive_snapshots_%s", apptime.Now().Format("20060102_150405"))
	config := excelpkg.ExportConfig{
		CustomHeaders: headers,
		Locale:        locale,
		Filename:      filename,
	}
	if err := excelpkg.WriteExcel(c, config, rows); err != nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "导出失败: "+err.Error())
	}
}

// buildArchiveHistoryMap 查询并格式化流程的历次归档复盘历史记录为清晰中文文本。
func (h *ArchiveReviewHandler) buildArchiveHistoryMap(c *gin.Context, processIDs []string, locale excelpkg.Locale) map[string]string {
	result := make(map[string]string)
	if len(processIDs) == 0 {
		return result
	}

	type archiveLogSimple struct {
		ProcessID       string    `gorm:"column:process_id"`
		CreatedAt       time.Time `gorm:"column:created_at"`
		UserName        string    `gorm:"column:user_name"`
		OverallDecision string    `gorm:"column:overall_decision"`
		Score           int       `gorm:"column:score"`
		Confidence      int       `gorm:"column:confidence"`
	}

	var logs []archiveLogSimple
	err := h.archiveLogRepo.WithTenant(c).
		Table("archive_logs").
		Select("process_id, created_at, user_name, overall_decision, score, confidence").
		Where("process_id IN ? AND status = ?", processIDs, model.JobStatusCompleted).
		Where("COALESCE(parse_error, '') = '' AND overall_decision IN ('compliant', 'partially_compliant', 'non_compliant')").
		Order("created_at desc").
		Find(&logs).Error

	if err != nil || len(logs) == 0 {
		return result
	}

	grouped := make(map[string][]archiveLogSimple)
	for _, l := range logs {
		grouped[l.ProcessID] = append(grouped[l.ProcessID], l)
	}

	for pid, list := range grouped {
		var lines []string
		for idx, item := range list {
			timeStr := item.CreatedAt.Local().Format("2006-01-02 15:04:05")
			opName := item.UserName
			if opName == "" {
				opName = "系统"
			}
			comp := excelpkg.TranslateEnum(excelpkg.EnumCompliance, item.OverallDecision, locale)
			line := fmt.Sprintf("[%d] %s | 操作人: %s | 合规性: %s | 评分: %d分 | 置信度: %d%%",
				idx+1, timeStr, opName, comp, item.Score, item.Confidence)
			lines = append(lines, line)
		}
		result[pid] = strings.Join(lines, "\n")
	}

	return result
}

// countArchiveSnapshotLogIDs 解析 valid_archive_log_ids JSON 数组并返回其元素数量。
func countArchiveSnapshotLogIDs(data datatypes.JSON) int {
	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return 0
	}
	return len(ids)
}

// GetSnapshotChain 获取指定流程的归档复盘链详情（所有有效复盘记录按时间排序）。
// GET /api/archive/snapshots/:processId/chain
// 路径参数：processId（OA 流程编号）
// 返回：{"chain": [...]} 复盘链记录数组。
func (h *ArchiveReviewHandler) GetSnapshotChain(c *gin.Context) {
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
	// 解析快照中存储的有效复盘记录 ID 列表
	var idStrs []string
	_ = json.Unmarshal(snapshot.ValidArchiveLogIDs, &idStrs)
	ids := make([]uuid.UUID, 0, len(idStrs))
	for _, s := range idStrs {
		if uid, err := uuid.Parse(s); err == nil {
			ids = append(ids, uid)
		}
	}
	chain, err := h.archiveLogRepo.ListByIDsWithUserOrdered(c, ids)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	for i := range chain {
		service.NormalizeArchiveLogForResponse(&chain[i].ArchiveLog)
	}
	sort.Slice(chain, func(i, j int) bool {
		return chain[i].CreatedAt.After(chain[j].CreatedAt)
	})
	response.Success(c, gin.H{"chain": chain})
}

// parseArchiveSnapshotQuery 解析归档快照列表的过滤参数及分页参数。
func parseArchiveSnapshotQuery(c *gin.Context) (repository.ArchiveSnapshotFilter, int, int) {
	filter := repository.ArchiveSnapshotFilter{
		Compliance:  c.Query("compliance"),
		Keyword:     c.Query("keyword"),
		ProcessType: c.Query("process_type"),
		Operator:    c.Query("operator"),
		Department:  c.Query("department"),
	}
	if s := c.Query("start_date"); s != "" {
		if t, err := time.ParseInLocation("2006-01-02", s, apptime.Location()); err == nil {
			filter.StartDate = &t
		}
	}
	if s := c.Query("end_date"); s != "" {
		if t, err := time.ParseInLocation("2006-01-02", s, apptime.Location()); err == nil {
			end := t.AddDate(0, 0, 1)
			filter.EndDateExclusive = &end
		}
	}
	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "page_size", 20)
	return filter, page, pageSize
}
