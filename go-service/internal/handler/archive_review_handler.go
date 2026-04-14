// 归档复盘运行时处理器，负责归档任务执行、状态查询、日志管理及快照数据管理。
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
	"oa-smart-audit/go-service/internal/pkg/response"
	"oa-smart-audit/go-service/internal/repository"
	"oa-smart-audit/go-service/internal/service"
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
			c.SSEvent("message", msg)
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

	filename := fmt.Sprintf("archive_logs_%s.csv", time.Now().Format("20060102150405"))
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
		if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
			params.ArchiveDateStart = &t
		}
	}
	if s := c.Query("end_date"); s != "" {
		if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
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
		if t, err := time.Parse("2006-01-02", s); err == nil {
			filter.StartDate = &t
		}
	}
	if s := c.Query("end_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			// 结束日期扩展到当天末尾（23:59:59）
			end := t.Add(24*time.Hour - time.Second)
			filter.EndDate = &end
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
	type itemDTO struct {
		repository.ArchiveSnapshotListRow
		UpdatedAtFmt string `json:"updated_at_fmt"`
		CreatedAtFmt string `json:"created_at_fmt"`
	}
	out := make([]itemDTO, len(items))
	for i, row := range items {
		out[i] = itemDTO{
			ArchiveSnapshotListRow: row,
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
