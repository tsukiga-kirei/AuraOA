package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
	"auraoa/go-service/internal/pkg/errcode"
	excelpkg "auraoa/go-service/internal/pkg/excel"
	"auraoa/go-service/internal/pkg/response"
	"auraoa/go-service/internal/repository"
	"auraoa/go-service/internal/service"
)

// ProcessSummaryHandler 处理流程总结运行时与数据管理请求。
type ProcessSummaryHandler struct {
	summaryService *service.ProcessSummaryService
}

func NewProcessSummaryHandler(summaryService *service.ProcessSummaryService) *ProcessSummaryHandler {
	return &ProcessSummaryHandler{summaryService: summaryService}
}

func (h *ProcessSummaryHandler) GetEmbedContext(c *gin.Context) {
	processID := c.Query("process_id")
	if processID == "" {
		processID = c.Query("requestid")
	}
	if processID == "" {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "process_id 不能为空")
		return
	}
	data, err := h.summaryService.GetEmbedContext(c, processID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, data)
}

func (h *ProcessSummaryHandler) ExecuteEmbed(c *gin.Context) {
	var req service.SummaryExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败: "+err.Error())
		return
	}
	result, err := h.summaryService.ExecuteEmbed(c, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	if result.Status == "pending" {
		c.JSON(http.StatusAccepted, response.Response{Code: 0, Message: "accepted", Data: result})
		return
	}
	response.Success(c, result)
}

func (h *ProcessSummaryHandler) GetJobStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "任务 ID 无效")
		return
	}
	data, err := h.summaryService.GetJobStatus(c, id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, data)
}

func (h *ProcessSummaryHandler) GetJobStream(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "任务 ID 无效")
		return
	}
	ch, closeSub, err := h.summaryService.SubscribeJobStream(c.Request.Context(), id)
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

func (h *ProcessSummaryHandler) ListSnapshots(c *gin.Context) {
	filter, page, pageSize := parseProcessSummarySnapshotQuery(c)
	items, total, err := h.summaryService.ListSnapshots(c, filter, page, pageSize)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	type itemDTO struct {
		repository.ProcessSummarySnapshotListRow
		UpdatedAtFmt string `json:"updated_at_fmt"`
		CreatedAtFmt string `json:"created_at_fmt"`
	}
	out := make([]itemDTO, len(items))
	for i, row := range items {
		out[i] = itemDTO{
			ProcessSummarySnapshotListRow: row,
			UpdatedAtFmt:                  row.UpdatedAt.Local().Format("2006/1/2 15:04"),
			CreatedAtFmt:                  row.CreatedAt.Local().Format("2006/1/2 15:04"),
		}
	}
	response.Success(c, gin.H{"items": out, "total": total, "page": page, "page_size": pageSize})
}

func (h *ProcessSummaryHandler) GetSnapshotStats(c *gin.Context) {
	filter, _, _ := parseProcessSummarySnapshotQuery(c)
	stats, err := h.summaryService.GetSnapshotStats(c, filter.Channel)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, stats)
}

func (h *ProcessSummaryHandler) ExportSnapshots(c *gin.Context) {
	filter, _, _ := parseProcessSummarySnapshotQuery(c)
	items, err := h.summaryService.ListSnapshotsForExport(c, filter)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	locale := excelpkg.ResolveLocale(c)
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		channel := item.Channel
		if channel == "" {
			channel = model.AuditSnapshotChannelWorkbench
		}
		rows = append(rows, []string{
			item.ProcessID,
			item.Title,
			item.Operator,
			item.Department,
			item.ProcessType,
			excelpkg.TranslateEnum(excelpkg.EnumSourceChannel, channel, locale),
			fmt.Sprintf("%d", item.BlockCount),
			fmt.Sprintf("%d", countSummarySnapshotLogIDs(item.ValidLogIDs)),
			item.UpdatedAt.Local().Format("2006-01-02 15:04:05"),
		})
	}
	filename := fmt.Sprintf("summary_snapshots_%s", apptime.Now().Format("20060102_150405"))
	config := excelpkg.ExportConfig{
		ExportType: excelpkg.ExportTypeSummarySnapshot,
		Locale:     locale,
		Filename:   filename,
	}
	if err := excelpkg.WriteExcel(c, config, rows); err != nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "导出失败: "+err.Error())
	}
}

func countSummarySnapshotLogIDs(raw []byte) int {
	var ids []string
	if err := json.Unmarshal(raw, &ids); err != nil {
		return 0
	}
	return len(ids)
}

func (h *ProcessSummaryHandler) GetSnapshotChain(c *gin.Context) {
	processID := c.Param("processId")
	if processID == "" {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "流程ID不能为空")
		return
	}
	chain, err := h.summaryService.GetSnapshotChain(c, processID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"chain": chain})
}

func parseProcessSummarySnapshotQuery(c *gin.Context) (repository.ProcessSummarySnapshotFilter, int, int) {
	filter := repository.ProcessSummarySnapshotFilter{
		Channel:     c.Query("channel"),
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
			end := t.Add(24*time.Hour - time.Second)
			filter.EndDate = &end
		}
	}
	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "page_size", 20)
	return filter, page, pageSize
}
