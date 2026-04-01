package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/crypto"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
	"oa-smart-audit/go-service/internal/pkg/oa"
	"oa-smart-audit/go-service/internal/repository"
)

const (
	batchArchiveMaxLimit = 10

	archiveErrStaleMessage = "归档复盘任务超时（请重新发起）"
	archiveJobMaxAge       = 30 * time.Minute
	archiveProcessTimeout  = 25 * time.Minute
)

// archiveItemHasComplianceOutcome 以有效快照为准；无快照时回退看 archive_result（兼容旧数据）。
func archiveItemHasComplianceOutcome(item map[string]interface{}) bool {
	if sc, ok := item["snapshot_compliance"].(string); ok && sc != "" {
		return sc == "compliant" || sc == "partially_compliant" || sc == "non_compliant"
	}
	status, _ := item["archive_status"].(string)
	if status != model.AuditStatusCompleted {
		return false
	}
	result, _ := item["archive_result"].(map[string]interface{})
	if result == nil {
		return false
	}
	c, ok := result["overall_compliance"].(string)
	if !ok {
		return false
	}
	return c == "compliant" || c == "partially_compliant" || c == "non_compliant"
}

func archiveItemComplianceClass(item map[string]interface{}, want string) bool {
	if sc, ok := item["snapshot_compliance"].(string); ok && sc != "" {
		return sc == want
	}
	status, _ := item["archive_status"].(string)
	if status != model.AuditStatusCompleted {
		return false
	}
	result, _ := item["archive_result"].(map[string]interface{})
	if result == nil {
		return false
	}
	c, ok := result["overall_compliance"].(string)
	return ok && c == want
}

// ArchiveReviewService 处理归档复盘运行时业务。
type ArchiveReviewService struct {
	archiveLogRepo       *repository.ArchiveLogRepo
	archiveSnapshotRepo  *repository.ArchiveProcessSnapshotRepo
	archiveConfigRepo    *repository.ProcessArchiveConfigRepo
	archiveRuleRepo      *repository.ArchiveRuleRepo
	userConfigRepo       *repository.UserPersonalConfigRepo
	tenantRepo        *repository.TenantRepo
	oaConnRepo          *repository.OAConnectionRepo
	aiModelRepo         *repository.AIModelRepo
	aiCaller            *AIModelCallerService
	orgRepo             *repository.OrgRepo
	db                  *gorm.DB
	rdb                 *redis.Client
	cancelMap           sync.Map
}

func NewArchiveReviewService(
	archiveLogRepo *repository.ArchiveLogRepo,
	archiveSnapshotRepo *repository.ArchiveProcessSnapshotRepo,
	archiveConfigRepo *repository.ProcessArchiveConfigRepo,
	archiveRuleRepo *repository.ArchiveRuleRepo,
	userConfigRepo *repository.UserPersonalConfigRepo,
	tenantRepo *repository.TenantRepo,
	oaConnRepo *repository.OAConnectionRepo,
	aiModelRepo *repository.AIModelRepo,
	aiCaller *AIModelCallerService,
	orgRepo *repository.OrgRepo,
	db *gorm.DB,
	rdb *redis.Client,
) *ArchiveReviewService {
	return &ArchiveReviewService{
		archiveLogRepo:      archiveLogRepo,
		archiveSnapshotRepo: archiveSnapshotRepo,
		archiveConfigRepo:   archiveConfigRepo,
		archiveRuleRepo:     archiveRuleRepo,
		userConfigRepo:      userConfigRepo,
		tenantRepo:          tenantRepo,
		oaConnRepo:          oaConnRepo,
		aiModelRepo:         aiModelRepo,
		aiCaller:            aiCaller,
		orgRepo:             orgRepo,
		db:                  db,
		rdb:                 rdb,
	}
}

func (s *ArchiveReviewService) ListProcesses(c *gin.Context, params dto.ArchiveListParams) (*dto.ArchiveProcessListResponse, error) {
	tenantID, userID, err := s.extractIDs(c)
	if err != nil {
		return nil, err
	}
	_, _ = s.FailStaleArchiveJobs(context.Background())

	configs, err := s.getAccessibleArchiveConfigs(c, userID, tenantID)
	if err != nil {
		return nil, err
	}
	if len(configs) == 0 {
		return &dto.ArchiveProcessListResponse{Items: []map[string]interface{}{}, Total: 0}, nil
	}

	allowedTables := make(map[string]model.ProcessArchiveConfig, len(configs))
	allowedTypes := make(map[string]model.ProcessArchiveConfig, len(configs))
	for _, cfg := range configs {
		allowedTypes[strings.ToLower(cfg.ProcessType)] = cfg
		if cfg.MainTableName != "" {
			allowedTables[strings.ToLower(cfg.MainTableName)] = cfg
		}
	}

	adapter, err := s.getOAAdapter(tenantID)
	if err != nil {
		return nil, err
	}

	listFilter := oa.ArchivedListFilter{
		ArchiveDateStart:        params.ArchiveDateStart,
		ArchiveDateEndExclusive: params.ArchiveDateEndExclusive,
	}
	items, err := adapter.FetchArchivedList(c.Request.Context(), s.extractUsername(c), listFilter)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAQueryFailed, "获取 OA 已归档流程失败: "+err.Error())
	}

	filtered := make([]oa.ArchivedItem, 0, len(items))
	for _, item := range items {
		_, byType := allowedTypes[strings.ToLower(item.ProcessType)]
		_, byTable := allowedTables[strings.ToLower(item.MainTableName)]
		if !byType && !byTable {
			continue
		}
		if item.ProcessTypeLabel == "" {
			if cfg, ok := allowedTypes[strings.ToLower(item.ProcessType)]; ok {
				item.ProcessTypeLabel = cfg.ProcessTypeLabel
			}
		}
		filtered = append(filtered, item)
	}

	processIDs := make([]string, len(filtered))
	for i, item := range filtered {
		processIDs[i] = item.ProcessID
	}
	latestMap, err := s.archiveLogRepo.GetLatestResultMap(c, processIDs)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询归档复盘记录失败")
	}
	snapshotMap, err := s.archiveSnapshotRepo.GetMapByProcessIDs(c, processIDs)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询归档有效结论失败")
	}

	results := make([]map[string]interface{}, 0, len(filtered))
	for _, item := range filtered {
		record := map[string]interface{}{
			"process_id":         item.ProcessID,
			"title":              item.Title,
			"applicant":          item.Applicant,
			"department":         item.Department,
			"process_type":       item.ProcessType,
			"process_type_label": item.ProcessTypeLabel,
			"current_node":       item.CurrentNode,
			"submit_time":        item.SubmitTime,
			"archive_time":       item.ArchiveTime,
			"has_review":         false,
			"archive_result":     nil,
			"in_archive":         true,
		}

		snap := snapshotMap[item.ProcessID]
		latest, hasLatest := latestMap[item.ProcessID]

		if snap != nil {
			record["has_review"] = true
			record["snapshot_compliance"] = snap.Compliance
			validLog, err := s.archiveLogRepo.GetByID(c, snap.LatestValidArchiveLogID)
			if err == nil && validLog != nil {
				record["archive_status"] = model.AuditStatusCompleted
				record["archive_result"] = buildArchiveResultFromLog(validLog)
			}
		}
		if hasLatest {
			st := latest.Status
			switch st {
			case model.AuditStatusPending, model.AuditStatusAssembling, model.AuditStatusReasoning, model.AuditStatusExtracting:
				record["archive_status"] = st
				record["archive_result"] = buildArchiveResultFromLog(latest)
			case model.AuditStatusFailed:
				if snap == nil {
					record["archive_status"] = nil
					record["archive_result"] = nil
					record["has_review"] = false
					delete(record, "snapshot_compliance")
				}
			case model.AuditStatusCompleted:
				if snap == nil {
					record["archive_status"] = nil
					record["archive_result"] = nil
					record["has_review"] = false
					delete(record, "snapshot_compliance")
				}
			}
		}
		results = append(results, record)
	}

	return &dto.ArchiveProcessListResponse{
		Items: results,
		Total: len(results),
	}, nil
}

// ListProcessesPaged 分页查询已归档流程（在内存获取全量后做筛选+分页）。
func (s *ArchiveReviewService) ListProcessesPaged(c *gin.Context, params dto.ArchiveListParams) (*dto.ArchiveProcessListResponse, error) {
	full, err := s.ListProcesses(c, params)
	if err != nil {
		return nil, err
	}

	// 筛选（与 GetStats 中「归档未审核」语义一致）
	filtered := make([]map[string]interface{}, 0, len(full.Items))
	for _, item := range full.Items {
		if params.Keyword != "" {
			kw := strings.ToLower(params.Keyword)
			title, _ := item["title"].(string)
			pid, _ := item["process_id"].(string)
			if !strings.Contains(strings.ToLower(title), kw) && !strings.Contains(strings.ToLower(pid), kw) {
				continue
			}
		}
		if params.Applicant != "" {
			applicant, _ := item["applicant"].(string)
			if !strings.Contains(strings.ToLower(applicant), strings.ToLower(params.Applicant)) {
				continue
			}
		}
		if params.ProcessType != "" {
			pt, _ := item["process_type"].(string)
			if !strings.EqualFold(pt, params.ProcessType) {
				continue
			}
		}
		if params.Department != "" {
			dept, _ := item["department"].(string)
			if dept != params.Department {
				continue
			}
		}
		if params.AuditStatus != "" {
			switch params.AuditStatus {
			case "unaudited":
				if archiveItemHasComplianceOutcome(item) {
					continue
				}
			case "compliant":
				if !archiveItemComplianceClass(item, "compliant") {
					continue
				}
			case "partially_compliant":
				if !archiveItemComplianceClass(item, "partially_compliant") {
					continue
				}
			case "non_compliant":
				if !archiveItemComplianceClass(item, "non_compliant") {
					continue
				}
			}
		}
		filtered = append(filtered, item)
	}

	total := len(filtered)
	page := params.Page
	pageSize := params.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	return &dto.ArchiveProcessListResponse{
		Items:    filtered[start:end],
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *ArchiveReviewService) GetStats(c *gin.Context, params dto.ArchiveListParams) (*dto.ArchiveReviewStats, error) {
	resp, err := s.ListProcesses(c, params)
	if err != nil {
		return nil, err
	}

	stats := &dto.ArchiveReviewStats{
		TotalCount: len(resp.Items),
	}
	for _, item := range resp.Items {
		if archiveItemHasComplianceOutcome(item) {
			result, _ := item["archive_result"].(map[string]interface{})
			c, _ := result["overall_compliance"].(string)
			switch c {
			case "compliant":
				stats.CompliantCount++
			case "partially_compliant":
				stats.PartialCount++
			case "non_compliant":
				stats.NonCompliantCount++
			}
		} else {
			stats.UnauditedCount++
		}
	}
	for _, item := range resp.Items {
		status, _ := item["archive_status"].(string)
		switch status {
		case model.AuditStatusPending, model.AuditStatusAssembling, model.AuditStatusReasoning, model.AuditStatusExtracting:
			stats.RunningCount++
		}
	}
	return stats, nil
}

func (s *ArchiveReviewService) Execute(c *gin.Context, req *dto.ArchiveReviewExecuteRequest) (*dto.ArchiveReviewSubmitResponse, error) {
	if s.rdb == nil {
		return nil, newServiceError(errcode.ErrInternalServer, "异步队列未初始化（Redis 不可用）")
	}

	logID, tenantID, userID, err := s.createPendingArchiveLog(c, req)
	if err != nil {
		return nil, err
	}

	logEntry, _ := s.archiveLogRepo.GetByID(c, logID)
	if _, err := EnqueueArchiveJob(c.Request.Context(), s.rdb, logID, tenantID, userID); err != nil {
		_ = s.archiveLogRepo.UpdateFields(c, logID, map[string]interface{}{
			"status":        model.AuditStatusFailed,
			"error_message": "任务入队失败: " + err.Error(),
			"updated_at":    time.Now(),
		})
		return nil, newServiceError(errcode.ErrRedisConn, "归档复盘任务入队失败: "+err.Error())
	}

	return &dto.ArchiveReviewSubmitResponse{
		Status:    model.AuditStatusPending,
		ID:        logID.String(),
		TraceID:   fmt.Sprintf("AR-%s", logID.String()[:8]),
		ProcessID: req.ProcessID,
		CreatedAt: logEntry.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *ArchiveReviewService) BatchExecute(c *gin.Context, items []dto.ArchiveReviewExecuteRequest) (*dto.ArchiveBatchExecuteResponse, error) {
	if len(items) > batchArchiveMaxLimit {
		return nil, newServiceError(errcode.ErrBatchLimitExceeded,
			fmt.Sprintf("批量复盘上限 %d 条，当前 %d 条", batchArchiveMaxLimit, len(items)))
	}

	result := &dto.ArchiveBatchExecuteResponse{
		Results: make([]dto.ArchiveReviewSubmitResponse, 0, len(items)),
		Total:   len(items),
	}

	for _, item := range items {
		resp, err := s.Execute(c, &item)
		if err != nil {
			result.Failed++
			result.Results = append(result.Results, dto.ArchiveReviewSubmitResponse{
				Status:    model.AuditStatusFailed,
				ProcessID: item.ProcessID,
			})
			continue
		}
		result.Accepted++
		result.Results = append(result.Results, *resp)
	}

	return result, nil
}

// ListPendingForBatch 为调度器提供：按当前上下文 OA 用户拉取已归档流程，
// 供 cron archive_batch 任务批量调用（与任务归属用户一致）。
func (s *ArchiveReviewService) ListPendingForBatch(c *gin.Context, workflowIds []string, dateRangeDays int, limit int) ([]dto.ArchiveReviewExecuteRequest, error) {
	tenantID, _, err := s.extractIDs(c)
	if err != nil {
		return nil, err
	}
	username := s.extractUsername(c)
	if username == "" {
		return nil, newServiceError(errcode.ErrParamValidation, "无法解析 OA 登录用户名，请检查任务归属用户账号")
	}
	adapter, err := s.getOAAdapter(tenantID)
	if err != nil {
		return nil, err
	}

	filter := oa.ArchivedListFilter{}
	if dateRangeDays > 0 {
		start := time.Now().AddDate(0, 0, -dateRangeDays)
		filter.ArchiveDateStart = &start
	}

	allItems, err := adapter.FetchArchivedList(c.Request.Context(), username, filter)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAQueryFailed, "获取 OA 归档流程失败: "+err.Error())
	}

	// 按租户已配置的归档复盘配置过滤
	archiveCfgs, _ := s.archiveConfigRepo.ListByTenant(c)
	allowedTypes := make(map[string]struct{})
	allowedTables := make(map[string]struct{})
	for _, cfg := range archiveCfgs {
		if cfg.ProcessType != "" {
			allowedTypes[strings.ToLower(cfg.ProcessType)] = struct{}{}
		}
		if cfg.MainTableName != "" {
			allowedTables[strings.ToLower(cfg.MainTableName)] = struct{}{}
		}
	}

	// 4. 获取当前已归档项的复盘快照，排除已复盘项（已复盘流程存有快照记录）
	archPIDs := make([]string, len(allItems))
	for i, it := range allItems {
		archPIDs[i] = it.ProcessID
	}
	snapshotMap, _ := s.archiveSnapshotRepo.GetMapByProcessIDs(c, archPIDs)

	wfMap := make(map[string]bool)
	for _, id := range workflowIds {
		wfMap[id] = true
	}

	var result []dto.ArchiveReviewExecuteRequest
	for _, item := range allItems {
		// 1. 权限与配置过滤
		_, byType := allowedTypes[strings.ToLower(item.ProcessType)]
		_, byTable := allowedTables[strings.ToLower(item.MainTableName)]
		if !byType && !byTable {
			continue
		}

		// 2. 指定 ID/类型 过滤
		if len(wfMap) > 0 && !wfMap[item.ProcessID] && !wfMap[item.ProcessType] {
			continue
		}

		// 3. 排除已处理（归档未复盘逻辑：快照中不存在记录）
		if _, exists := snapshotMap[item.ProcessID]; exists {
			continue
		}

		result = append(result, dto.ArchiveReviewExecuteRequest{
			ProcessID:   item.ProcessID,
			ProcessType: item.ProcessType,
			Title:       item.Title,
		})
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (s *ArchiveReviewService) CancelJob(c *gin.Context, id uuid.UUID) error {
	tenantID, _, err := s.extractIDs(c)
	if err != nil {
		return err
	}
	if _, err := s.getAccessibleArchiveLog(c, id); err != nil {
		return err
	}
	err = s.markArchiveFailedDB(tenantID, id, "已主动中止")
	if cancelFunc, ok := s.cancelMap.Load(id.String()); ok {
		cancelFunc.(context.CancelFunc)()
	}
	return err
}

// ListArchiveLogs 数据管理页：分页查询当前租户归档复盘日志。
func (s *ArchiveReviewService) ListArchiveLogs(c *gin.Context, filter repository.ArchiveLogFilter, page, pageSize int) ([]repository.ArchiveLogWithUser2, int64, error) {
	items, total, err := s.archiveLogRepo.ListPagedWithUser(c, filter, page, pageSize)
	if err != nil {
		return nil, 0, newServiceError(errcode.ErrDatabase, "查询归档日志失败")
	}
	return items, total, nil
}

// GetArchiveLogStats 数据管理页：获取当前租户归档复盘日志统计。
func (s *ArchiveReviewService) GetArchiveLogStats(c *gin.Context) (*repository.ArchiveLogStats, error) {
	stats, err := s.archiveLogRepo.CountStats(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "统计查询失败")
	}
	return stats, nil
}

func (s *ArchiveReviewService) GetArchiveJobStatus(c *gin.Context, id uuid.UUID) (map[string]interface{}, error) {
	logEntry, err := s.getAccessibleArchiveLog(c, id)
	if err != nil {
		return nil, err
	}
	logEntry, err = s.applyStaleArchiveTimeout(c, logEntry)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询归档复盘任务失败")
	}
	out := buildArchiveResultFromLog(logEntry)
	out["updated_at"] = logEntry.UpdatedAt.Format(time.RFC3339)
	out["progress_steps"] = archiveProgressSteps(logEntry.Status)
	return out, nil
}

func (s *ArchiveReviewService) GetArchiveHistory(c *gin.Context, processID string) ([]repository.ArchiveLogWithUser, error) {
	if _, err := s.ensureArchiveProcessAccessible(c, processID); err != nil {
		return nil, err
	}
	snap, err := s.archiveSnapshotRepo.GetByProcessID(c, processID)
	if err != nil {
		return nil, err
	}
	if snap == nil {
		return []repository.ArchiveLogWithUser{}, nil
	}
	ids := parseArchiveSnapshotValidIDs(snap.ValidArchiveLogIDs)
	return s.archiveLogRepo.ListByIDsWithUserOrdered(c, ids)
}

func (s *ArchiveReviewService) GetArchiveResult(c *gin.Context, id uuid.UUID) (map[string]interface{}, error) {
	logEntry, err := s.getAccessibleArchiveLog(c, id)
	if err != nil {
		return nil, err
	}
	return buildArchiveResultFromLog(logEntry), nil
}

func (s *ArchiveReviewService) SubscribeJobStream(c *gin.Context, id uuid.UUID) (<-chan string, func(), error) {
	if s.rdb == nil {
		return nil, nil, newServiceError(errcode.ErrRedisConn, "归档复盘流服务未初始化（Redis 不可用）")
	}
	if _, err := s.getAccessibleArchiveLog(c, id); err != nil {
		return nil, nil, err
	}

	ctx := c.Request.Context()
	pubsub := s.rdb.Subscribe(ctx, "archive:stream:"+id.String())
	ch := make(chan string)

	history, _ := s.rdb.Get(ctx, "archive:reasoning:"+id.String()).Result()
	go func() {
		defer close(ch)
		if history != "" {
			ch <- history
		}
		for msg := range pubsub.Channel() {
			ch <- msg.Payload
		}
	}()
	return ch, func() { _ = pubsub.Close() }, nil
}

func (s *ArchiveReviewService) createPendingArchiveLog(c *gin.Context, req *dto.ArchiveReviewExecuteRequest) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	tenantID, userID, err := s.extractIDs(c)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}

	tenant, err := s.tenantRepo.FindByID(tenantID)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, newServiceError(errcode.ErrDatabase, "获取租户信息失败")
	}
	if tenant.PrimaryModelID == nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, newServiceError(errcode.ErrNoAIModelConfig, "租户未配置主用 AI 模型")
	}
	if _, err := s.aiModelRepo.FindByID(*tenant.PrimaryModelID); err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, newServiceError(errcode.ErrNoAIModelConfig, "AI 模型配置不存在")
	}

	cfg, err := s.archiveConfigRepo.GetByProcessType(c, req.ProcessType)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, newServiceError(errcode.ErrNoProcessConfig, fmt.Sprintf("流程 '%s' 的归档复盘配置不存在", req.ProcessType))
	}
	allowed, err := s.userCanAccessArchive(c, tenantID, userID, cfg)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}
	if !allowed {
		return uuid.Nil, uuid.Nil, uuid.Nil, newServiceError(errcode.ErrPermissionDenied, "当前用户无权执行该归档复盘")
	}

	logID := uuid.New()
	now := time.Now()
	logEntry := &model.ArchiveLog{
		ID:              logID,
		TenantID:        tenantID,
		UserID:          userID,
		ProcessID:       req.ProcessID,
		Title:           req.Title,
		ProcessType:     req.ProcessType,
		Status:          model.AuditStatusPending,
		Compliance:      "partially_compliant",
		ComplianceScore: 0,
		ArchiveResult:   datatypes.JSON([]byte("{}")),
		ProcessSnapshot: datatypes.JSON([]byte("{}")),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.archiveLogRepo.Create(logEntry); err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, newServiceError(errcode.ErrDatabase, "归档复盘日志写入失败")
	}
	return logID, tenantID, userID, nil
}

func (s *ArchiveReviewService) processArchiveJob(ctx context.Context, archiveLogID, tenantID, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, archiveProcessTimeout)
	s.cancelMap.Store(archiveLogID.String(), cancel)
	defer func() {
		cancel()
		s.cancelMap.Delete(archiveLogID.String())
	}()

	c := s.workerGinContext(ctx, tenantID, userID)
	logEntry, err := s.archiveLogRepo.GetByID(c, archiveLogID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if logEntry.Status != model.AuditStatusPending {
		return nil
	}
	if time.Since(logEntry.CreatedAt) > archiveJobMaxAge {
		_ = s.markArchiveFailedDB(tenantID, archiveLogID, archiveErrStaleMessage)
		return nil
	}

	startTime := time.Now()
	tenant, err := s.tenantRepo.FindByID(tenantID)
	if err != nil {
		s.markArchiveFailedOrTimeout(c, tenantID, archiveLogID, newServiceError(errcode.ErrDatabase, "获取租户信息失败"))
		return err
	}
	if tenant.PrimaryModelID == nil {
		se := newServiceError(errcode.ErrNoAIModelConfig, "租户未配置主用 AI 模型")
		s.markArchiveFailedOrTimeout(c, tenantID, archiveLogID, se)
		return se
	}
	modelCfg, err := s.aiModelRepo.FindByID(*tenant.PrimaryModelID)
	if err != nil {
		se := newServiceError(errcode.ErrNoAIModelConfig, "AI 模型配置不存在")
		s.markArchiveFailedOrTimeout(c, tenantID, archiveLogID, se)
		return se
	}
	if modelCfg.APIKey != "" {
		decrypted, err := crypto.Decrypt(modelCfg.APIKey)
		if err != nil {
			se := newServiceError(errcode.ErrInternalServer, "API Key 解密失败")
			s.markArchiveFailedOrTimeout(c, tenantID, archiveLogID, se)
			return se
		}
		modelCfg.APIKey = decrypted
	}

	config, err := s.archiveConfigRepo.GetByProcessType(c, logEntry.ProcessType)
	if err != nil {
		se := newServiceError(errcode.ErrNoProcessConfig, fmt.Sprintf("流程 '%s' 的归档复盘配置不存在", logEntry.ProcessType))
		s.markArchiveFailedOrTimeout(c, tenantID, archiveLogID, se)
		return se
	}

	var aiConfig model.ArchiveAIConfigData
	if err := json.Unmarshal(config.AIConfig, &aiConfig); err != nil {
		se := newServiceError(errcode.ErrInternalServer, "归档 AI 配置解析失败")
		s.markArchiveFailedOrTimeout(c, tenantID, archiveLogID, se)
		return se
	}

	rules, err := s.archiveRuleRepo.ListByConfigIDFilter(c, config.ID, nil, nil)
	if err != nil {
		se := newServiceError(errcode.ErrDatabase, "获取归档复盘规则失败")
		s.markArchiveFailedOrTimeout(c, tenantID, archiveLogID, se)
		return se
	}

	fieldSet, mergedRulesText := s.resolveArchiveUserConfig(c, userID, config, rules, logEntry.ProcessType)

	adapter, err := s.getOAAdapter(tenantID)
	if err != nil {
		s.markArchiveFailedOrTimeout(c, tenantID, archiveLogID, err)
		return err
	}

	_ = s.archiveLogRepo.UpdateFields(c, archiveLogID, map[string]interface{}{
		"status":     model.AuditStatusAssembling,
		"updated_at": time.Now(),
	})

	processData, err := adapter.FetchProcessData(ctx, logEntry.ProcessID)
	if err != nil {
		se := newServiceError(errcode.ErrOAQueryFailed, "拉取 OA 流程数据失败: "+err.Error())
		s.markArchiveFailedOrTimeout(c, tenantID, archiveLogID, se)
		return se
	}

	archivedItem, _ := s.fetchArchivedItem(ctx, adapter, logEntry.ProcessID)
	flowSnapshot, err := adapter.FetchProcessFlow(ctx, logEntry.ProcessID)
	if err != nil || flowSnapshot == nil {
		flowSnapshot = &oa.ProcessFlowSnapshot{
			IsComplete:   true,
			MissingNodes: []string{},
			Nodes:        []oa.ProcessFlowNode{},
			HistoryText:  "",
			GraphText:    "",
		}
	}

	currentNode := "已归档"
	processSnapshot := map[string]interface{}{
		"process_id":         logEntry.ProcessID,
		"title":              logEntry.Title,
		"process_type":       logEntry.ProcessType,
		"process_type_label": "",
		"applicant":          "",
		"department":         "",
		"current_node":       currentNode,
		"submit_time":        "",
		"archive_time":       "",
		"main_table_name":    "",
		"flow_snapshot":      flowSnapshot,
	}
	if archivedItem != nil {
		currentNode = firstNonEmpty(archivedItem.CurrentNode, currentNode)
		processSnapshot["process_type_label"] = archivedItem.ProcessTypeLabel
		processSnapshot["applicant"] = archivedItem.Applicant
		processSnapshot["department"] = archivedItem.Department
		processSnapshot["current_node"] = currentNode
		processSnapshot["submit_time"] = archivedItem.SubmitTime
		processSnapshot["archive_time"] = archivedItem.ArchiveTime
		processSnapshot["main_table_name"] = archivedItem.MainTableName
	}
	snapshotJSON, _ := json.Marshal(processSnapshot)

	reasoningReq := BuildArchiveReasoningPrompt(&aiConfig, logEntry.ProcessType, processData, mergedRulesText, currentNode, fieldSet, flowSnapshot)
	reasoningReq.Temperature = float64(tenant.Temperature)
	reasoningReq.MaxTokens = tenant.MaxTokensPerRequest
	reasoningReq.ModelConfig = modelCfg
	reasoningReq.StreamChunkFunc = func(chunk string) {
		key := "archive:reasoning:" + archiveLogID.String()
		s.rdb.Append(context.Background(), key, chunk)
		s.rdb.Expire(context.Background(), key, 24*time.Hour)
		s.rdb.Publish(context.Background(), "archive:stream:"+archiveLogID.String(), chunk)
	}

	_ = s.archiveLogRepo.UpdateFields(c, archiveLogID, map[string]interface{}{
		"status":           model.AuditStatusReasoning,
		"process_snapshot": datatypes.JSON(snapshotJSON),
		"updated_at":       time.Now(),
	})

	reasoningResp, err := s.aiCaller.Chat(c, tenantID, userID, modelCfg, reasoningReq)
	if err != nil {
		s.markArchiveFailedOrTimeout(c, tenantID, archiveLogID, err)
		return err
	}
	aiReasoning := reasoningResp.Content

	_ = s.archiveLogRepo.UpdateFields(c, archiveLogID, map[string]interface{}{
		"status":       model.AuditStatusExtracting,
		"ai_reasoning": aiReasoning,
		"updated_at":   time.Now(),
	})

	extractionReq := BuildArchiveExtractionPrompt(&aiConfig, aiReasoning, mergedRulesText)
	extractionReq.Temperature = 0.1
	extractionReq.MaxTokens = tenant.MaxTokensPerRequest
	extractionReq.ModelConfig = modelCfg
	extractionReq.SkipQuotaCheck = true

	extractionResp, err := s.aiCaller.Chat(c, tenantID, userID, modelCfg, extractionReq)
	if err != nil {
		s.markArchiveFailedOrTimeout(c, tenantID, archiveLogID, err)
		return err
	}

	totalDuration := int(time.Since(startTime).Milliseconds())
	parsed, parseErr := ParseArchiveReviewResult(extractionResp.Content)

	updates := map[string]interface{}{
		"duration_ms":      totalDuration,
		"raw_content":      extractionResp.Content,
		"ai_reasoning":     aiReasoning,
		"process_snapshot": datatypes.JSON(snapshotJSON),
		"updated_at":       time.Now(),
	}
	if parseErr != nil {
		updates["status"] = model.AuditStatusFailed
		updates["compliance"] = ""
		updates["compliance_score"] = 0
		updates["confidence"] = 0
		updates["parse_error"] = parseErr.Error()
		updates["archive_result"] = datatypes.JSON([]byte("{}"))
	} else {
		resultJSON, _ := json.Marshal(parsed)
		updates["status"] = model.AuditStatusCompleted
		updates["compliance"] = parsed.OverallCompliance
		updates["compliance_score"] = parsed.OverallScore
		updates["confidence"] = parsed.Confidence
		updates["archive_result"] = datatypes.JSON(resultJSON)
	}

	if err := s.archiveLogRepo.UpdateFields(c, archiveLogID, updates); err != nil {
		_ = s.markArchiveFailedDB(tenantID, archiveLogID, "保存归档复盘结果失败: "+err.Error())
		return err
	}
	if parseErr == nil && parsed != nil {
		if err := s.archiveSnapshotRepo.UpsertAppendValid(c, tenantID, logEntry.ProcessID, archiveLogID, logEntry.Title, logEntry.ProcessType, parsed.OverallCompliance, parsed.OverallScore, parsed.Confidence); err != nil {
			return err
		}
	}
	return nil
}

func (s *ArchiveReviewService) workerGinContext(ctx context.Context, tenantID, userID uuid.UUID) *gin.Context {
	rec := httptest.NewRecorder()
	gc, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(ctx)
	gc.Request = req
	gc.Set("tenant_id", tenantID.String())
	gc.Set("jwt_claims", &jwtpkg.JWTClaims{Sub: userID.String(), Username: ""})
	return gc
}

func (s *ArchiveReviewService) markArchiveFailed(c *gin.Context, id uuid.UUID, err error) {
	msg := err.Error()
	var se *ServiceError
	if errors.As(err, &se) {
		msg = se.Message
	}
	_ = s.archiveLogRepo.UpdateFields(c, id, map[string]interface{}{
		"status":        model.AuditStatusFailed,
		"error_message": msg,
		"updated_at":    time.Now(),
	})
}

func (s *ArchiveReviewService) markArchiveFailedDB(tenantID, id uuid.UUID, message string) error {
	return s.db.Model(&model.ArchiveLog{}).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		Updates(map[string]interface{}{
			"status":        model.AuditStatusFailed,
			"error_message": message,
			"updated_at":    time.Now(),
		}).Error
}

func (s *ArchiveReviewService) markArchiveFailedOrTimeout(c *gin.Context, tenantID, id uuid.UUID, err error) {
	if err == nil {
		return
	}
	if errors.Is(err, context.DeadlineExceeded) {
		_ = s.markArchiveFailedDB(tenantID, id, "归档复盘任务执行超时（请重新发起）")
		return
	}
	s.markArchiveFailed(c, id, err)
}

func (s *ArchiveReviewService) applyStaleArchiveTimeout(c *gin.Context, logEntry *model.ArchiveLog) (*model.ArchiveLog, error) {
	if logEntry == nil {
		return nil, nil
	}
	switch logEntry.Status {
	case model.AuditStatusCompleted, model.AuditStatusFailed:
		return logEntry, nil
	}
	if time.Since(logEntry.CreatedAt) <= archiveJobMaxAge {
		return logEntry, nil
	}
	if err := s.archiveLogRepo.UpdateFields(c, logEntry.ID, map[string]interface{}{
		"status":        model.AuditStatusFailed,
		"error_message": archiveErrStaleMessage,
		"updated_at":    time.Now(),
	}); err != nil {
		return logEntry, nil
	}
	return s.archiveLogRepo.GetByID(c, logEntry.ID)
}

func (s *ArchiveReviewService) FailStaleArchiveJobs(ctx context.Context) (int64, error) {
	cutoff := time.Now().Add(-archiveJobMaxAge)
	res := s.db.WithContext(ctx).Model(&model.ArchiveLog{}).
		Where("status IN ? AND created_at < ?", []string{
			model.AuditStatusPending,
			model.AuditStatusAssembling,
			model.AuditStatusReasoning,
			model.AuditStatusExtracting,
		}, cutoff).
		Updates(map[string]interface{}{
			"status":        model.AuditStatusFailed,
			"error_message": archiveErrStaleMessage,
			"updated_at":    time.Now(),
		})
	return res.RowsAffected, res.Error
}

func (s *ArchiveReviewService) getAccessibleArchiveLog(c *gin.Context, id uuid.UUID) (*model.ArchiveLog, error) {
	logEntry, err := s.archiveLogRepo.GetByID(c, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, newServiceError(errcode.ErrResourceNotFound, "归档复盘任务不存在")
		}
		return nil, newServiceError(errcode.ErrDatabase, "查询归档复盘任务失败")
	}
	if err := s.ensureArchiveProcessTypeAccessible(c, logEntry.ProcessType); err != nil {
		return nil, err
	}
	return logEntry, nil
}

func (s *ArchiveReviewService) ensureArchiveProcessAccessible(c *gin.Context, processID string) (*model.ArchiveLog, error) {
	logEntry, err := s.archiveLogRepo.GetLatestByProcessID(c, processID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询归档复盘记录失败")
	}
	if logEntry == nil {
		return nil, newServiceError(errcode.ErrResourceNotFound, "归档复盘记录不存在")
	}
	if err := s.ensureArchiveProcessTypeAccessible(c, logEntry.ProcessType); err != nil {
		return nil, err
	}
	return logEntry, nil
}

func (s *ArchiveReviewService) ensureArchiveProcessTypeAccessible(c *gin.Context, processType string) error {
	tenantID, userID, err := s.extractIDs(c)
	if err != nil {
		return err
	}

	cfg, err := s.archiveConfigRepo.GetByProcessType(c, processType)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return newServiceError(errcode.ErrNoProcessConfig, fmt.Sprintf("流程 '%s' 的归档复盘配置不存在", processType))
		}
		return newServiceError(errcode.ErrDatabase, "查询归档复盘配置失败")
	}

	allowed, err := s.userCanAccessArchive(c, tenantID, userID, cfg)
	if err != nil {
		return err
	}
	if !allowed {
		return newServiceError(errcode.ErrPermissionDenied, "当前用户无权访问该归档复盘记录")
	}
	return nil
}

func (s *ArchiveReviewService) extractIDs(c *gin.Context) (uuid.UUID, uuid.UUID, error) {
	tidVal, exists := c.Get("tenant_id")
	if !exists {
		return uuid.Nil, uuid.Nil, newServiceError(errcode.ErrNoAuthToken, "租户ID缺失")
	}
	tenantID, err := uuid.Parse(fmt.Sprintf("%v", tidVal))
	if err != nil {
		return uuid.Nil, uuid.Nil, newServiceError(errcode.ErrNoAuthToken, "租户ID格式无效")
	}

	claimsVal, _ := c.Get("jwt_claims")
	claims, ok := claimsVal.(*jwtpkg.JWTClaims)
	if !ok {
		return uuid.Nil, uuid.Nil, newServiceError(errcode.ErrNoAuthToken, "用户认证信息缺失")
	}
	userID, err := uuid.Parse(claims.Sub)
	if err != nil {
		return uuid.Nil, uuid.Nil, newServiceError(errcode.ErrNoAuthToken, "用户ID格式无效")
	}
	return tenantID, userID, nil
}

func (s *ArchiveReviewService) extractUsername(c *gin.Context) string {
	claimsVal, _ := c.Get("jwt_claims")
	if claims, ok := claimsVal.(*jwtpkg.JWTClaims); ok {
		return claims.Username
	}
	return ""
}

func (s *ArchiveReviewService) resolveArchiveUserConfig(
	c *gin.Context,
	userID uuid.UUID,
	config *model.ProcessArchiveConfig,
	tenantRules []model.ArchiveRule,
	processType string,
) (SelectedFieldSet, string) {
	var perms model.ArchiveUserPermissionsData
	if err := json.Unmarshal(config.UserPermissions, &perms); err != nil {
		perms = model.ArchiveUserPermissionsData{
			AllowCustomFields: true, AllowCustomRules: true, AllowModifyStrictness: true,
		}
	}

	var userDetail *model.ArchiveDetailItem
	userCfg, _ := s.userConfigRepo.GetByUserID(c, userID)
	if userCfg != nil {
		var items []model.ArchiveDetailItem
		_ = json.Unmarshal(userCfg.ArchiveDetails, &items)
		for i := range items {
			if items[i].ProcessType == processType || items[i].ConfigID == config.ID {
				userDetail = &items[i]
				break
			}
		}
	}

	fieldSet := s.resolveArchiveFieldSet(config, userDetail, perms)
	rulesText := s.resolveArchiveRulesText(tenantRules, userDetail, perms)
	return fieldSet, rulesText
}

func (s *ArchiveReviewService) resolveArchiveFieldSet(
	config *model.ProcessArchiveConfig,
	userDetail *model.ArchiveDetailItem,
	perms model.ArchiveUserPermissionsData,
) SelectedFieldSet {
	if config.FieldMode == "all" {
		return nil
	}

	type fieldItem struct {
		FieldKey string `json:"field_key"`
		Selected bool   `json:"selected"`
	}
	type detailTableItem struct {
		TableName string      `json:"table_name"`
		Fields    []fieldItem `json:"fields"`
	}

	var mainFields []fieldItem
	var detailTables []detailTableItem
	_ = json.Unmarshal(config.MainFields, &mainFields)
	_ = json.Unmarshal(config.DetailTables, &detailTables)

	fieldSet := SelectedFieldSet{"main": make(map[string]bool)}
	for _, field := range mainFields {
		if field.Selected {
			fieldSet["main"][field.FieldKey] = true
		}
	}
	for _, table := range detailTables {
		fieldSet[table.TableName] = make(map[string]bool)
		for _, field := range table.Fields {
			if field.Selected {
				fieldSet[table.TableName][field.FieldKey] = true
			}
		}
	}

	if perms.AllowCustomFields && userDetail != nil {
		for _, override := range userDetail.FieldConfig.FieldOverrides {
			table, key := parseFieldOverride(override)
			if _, ok := fieldSet[table]; !ok {
				fieldSet[table] = make(map[string]bool)
			}
			fieldSet[table][key] = true
		}
	}
	return fieldSet
}

func (s *ArchiveReviewService) resolveArchiveRulesText(
	tenantRules []model.ArchiveRule,
	userDetail *model.ArchiveDetailItem,
	perms model.ArchiveUserPermissionsData,
) string {
	if len(tenantRules) == 0 && (userDetail == nil || !perms.AllowCustomRules || len(userDetail.RuleConfig.CustomRules) == 0) {
		return "（无归档复盘规则）"
	}

	toggleMap := map[string]bool{}
	if userDetail != nil {
		for _, toggle := range userDetail.RuleConfig.RuleToggleOverrides {
			toggleMap[toggle.RuleID] = toggle.Enabled
		}
	}

	var lines []string
	for _, rule := range tenantRules {
		enabled := rule.Enabled == nil || *rule.Enabled
		if rule.RuleScope == "mandatory" {
			enabled = true
		} else if perms.AllowCustomRules {
			if override, ok := toggleMap[rule.ID.String()]; ok {
				enabled = override
			}
		}
		if !enabled {
			continue
		}
		lines = append(lines, fmt.Sprintf("%d. [%s] %s", len(lines)+1, rule.RuleScope, rule.RuleContent))
	}

	if perms.AllowCustomRules && userDetail != nil {
		for _, rule := range userDetail.RuleConfig.CustomRules {
			if !rule.Enabled {
				continue
			}
			lines = append(lines, fmt.Sprintf("%d. [custom] %s", len(lines)+1, rule.Content))
		}
	}

	if len(lines) == 0 {
		return "（无启用的归档复盘规则）"
	}
	return strings.Join(lines, "\n")
}

func (s *ArchiveReviewService) getAccessibleArchiveConfigs(c *gin.Context, userID, tenantID uuid.UUID) ([]model.ProcessArchiveConfig, error) {
	allCfgs, err := s.archiveConfigRepo.ListByTenant(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	if len(allCfgs) == 0 {
		return []model.ProcessArchiveConfig{}, nil
	}

	member, _ := s.orgRepo.FindByUserAndTenant(userID, tenantID)
	result := make([]model.ProcessArchiveConfig, 0, len(allCfgs))
	for _, cfg := range allCfgs {
		allowed, err := s.memberCanAccessArchive(member, &cfg)
		if err != nil {
			return nil, err
		}
		if allowed {
			result = append(result, cfg)
		}
	}
	return result, nil
}

func (s *ArchiveReviewService) userCanAccessArchive(c *gin.Context, tenantID, userID uuid.UUID, cfg *model.ProcessArchiveConfig) (bool, error) {
	member, _ := s.orgRepo.FindByUserAndTenant(userID, tenantID)
	return s.memberCanAccessArchive(member, cfg)
}

func (s *ArchiveReviewService) memberCanAccessArchive(member *model.OrgMember, cfg *model.ProcessArchiveConfig) (bool, error) {
	if cfg == nil || cfg.Status != "active" {
		return false, nil
	}

	var ac model.AccessControlData
	if err := json.Unmarshal(cfg.AccessControl, &ac); err != nil {
		return true, nil
	}
	if len(ac.AllowedRoles) == 0 && len(ac.AllowedMembers) == 0 && len(ac.AllowedDepartments) == 0 {
		return true, nil
	}
	if member == nil {
		return false, nil
	}
	if sliceContains(ac.AllowedMembers, member.ID.String()) {
		return true, nil
	}
	if sliceContains(ac.AllowedDepartments, member.DepartmentID.String()) {
		return true, nil
	}
	for _, role := range member.Roles {
		if sliceContains(ac.AllowedRoles, role.ID.String()) {
			return true, nil
		}
	}
	return false, nil
}

func (s *ArchiveReviewService) decryptOAConn(conn *model.OADatabaseConnection) error {
	password, err := crypto.Decrypt(conn.Password)
	if err != nil {
		return newServiceError(errcode.ErrInternalServer, "OA 数据库密码解密失败")
	}
	conn.Password = password
	return nil
}

func (s *ArchiveReviewService) getOAAdapter(tenantID uuid.UUID) (oa.OAAdapter, error) {
	tenant, err := s.tenantRepo.FindByID(tenantID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "获取租户失败")
	}
	if tenant.OADBConnectionID == nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "租户未配置 OA 数据库连接")
	}
	conn, err := s.oaConnRepo.FindByID(*tenant.OADBConnectionID)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "OA 数据库连接配置不存在")
	}
	if err := s.decryptOAConn(conn); err != nil {
		return nil, err
	}
	adapter, err := oa.NewOAAdapter(conn.OAType, conn)
	if err != nil {
		return nil, newServiceError(errcode.ErrOAConnectionFailed, "创建 OA 适配器失败: "+err.Error())
	}
	return adapter, nil
}

func (s *ArchiveReviewService) fetchArchivedItem(ctx context.Context, adapter oa.OAAdapter, processID string) (*oa.ArchivedItem, error) {
	items, err := adapter.FetchArchivedList(ctx, "", oa.ArchivedListFilter{})
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ProcessID == processID {
			return &items[i], nil
		}
	}
	return nil, nil
}

func buildArchiveResultFromLog(logEntry *model.ArchiveLog) map[string]interface{} {
	base := map[string]interface{}{
		"id":           logEntry.ID.String(),
		"trace_id":     fmt.Sprintf("AR-%s", logEntry.ID.String()[:8]),
		"process_id":   logEntry.ProcessID,
		"title":        logEntry.Title,
		"process_type": logEntry.ProcessType,
		"status":       logEntry.Status,
		"ai_reasoning": logEntry.AIReasoning,
		"created_at":   logEntry.CreatedAt.Format(time.RFC3339),
		"duration_ms":  logEntry.DurationMs,
	}
	if logEntry.ErrorMessage != "" {
		base["error_message"] = logEntry.ErrorMessage
	}
	if len(logEntry.ProcessSnapshot) > 0 {
		var snapshot map[string]interface{}
		if err := json.Unmarshal(logEntry.ProcessSnapshot, &snapshot); err == nil {
			base["process_snapshot"] = snapshot
		}
	}

	switch logEntry.Status {
	case model.AuditStatusPending, model.AuditStatusAssembling, model.AuditStatusReasoning, model.AuditStatusExtracting:
		return base
	case model.AuditStatusFailed:
		// 失败不写入合规结论，与「归档未审核」语义一致；前端按未完成展示，避免与部分合规混淆。
		base["flow_audit"] = emptyArchiveFlowAuditMap()
		base["field_audit"] = []interface{}{}
		base["rule_audit"] = []interface{}{}
		base["risk_points"] = []string{}
		base["suggestions"] = []string{}
		base["ai_summary"] = ""
		return base
	}

	base["overall_compliance"] = logEntry.Compliance
	base["overall_score"] = logEntry.ComplianceScore
	base["confidence"] = logEntry.Confidence

	if logEntry.ParseError != "" {
		base["parse_error"] = logEntry.ParseError
		base["raw_content"] = logEntry.RawContent
		base["flow_audit"] = emptyArchiveFlowAuditMap()
		base["field_audit"] = []interface{}{}
		base["rule_audit"] = []interface{}{}
		base["risk_points"] = []string{}
		base["suggestions"] = []string{}
		base["ai_summary"] = ""
		return base
	}

	var parsed model.ArchiveResultJSON
	if err := json.Unmarshal(logEntry.ArchiveResult, &parsed); err != nil {
		base["flow_audit"] = emptyArchiveFlowAuditMap()
		base["field_audit"] = []interface{}{}
		base["rule_audit"] = []interface{}{}
		base["risk_points"] = []string{}
		base["suggestions"] = []string{}
		base["ai_summary"] = ""
		return base
	}

	base["overall_compliance"] = parsed.OverallCompliance
	base["overall_score"] = parsed.OverallScore
	base["confidence"] = parsed.Confidence
	base["flow_audit"] = parsed.FlowAudit
	base["field_audit"] = parsed.FieldAudit
	base["rule_audit"] = parsed.RuleAudit
	base["risk_points"] = parsed.RiskPoints
	base["suggestions"] = parsed.Suggestions
	base["ai_summary"] = parsed.AISummary
	return base
}

func emptyArchiveFlowAuditMap() map[string]interface{} {
	return map[string]interface{}{
		"is_complete":   true,
		"missing_nodes": []string{},
		"node_results":  []interface{}{},
	}
}

func archiveProgressSteps(status string) []map[string]interface{} {
	defs := []struct {
		key   string
		label string
	}{
		{model.AuditStatusPending, "排队中"},
		{model.AuditStatusAssembling, "组装复盘提示词"},
		{model.AuditStatusReasoning, "推理分析"},
		{model.AuditStatusExtracting, "结构化提取"},
	}
	phaseIdx := map[string]int{
		model.AuditStatusPending:    0,
		model.AuditStatusAssembling: 1,
		model.AuditStatusReasoning:  2,
		model.AuditStatusExtracting: 3,
	}
	cur, ok := phaseIdx[status]
	if !ok {
		if status == model.AuditStatusCompleted {
			cur = 3
		} else if status == model.AuditStatusFailed {
			cur = 2
		} else {
			cur = 0
		}
	}

	var steps []map[string]interface{}
	for i, def := range defs {
		step := map[string]interface{}{"key": def.key, "label": def.label}
		switch {
		case status == model.AuditStatusFailed && i == cur:
			step["failed"] = true
		case i < cur:
			step["done"] = true
		case i == cur && cur < 4 && status != model.AuditStatusFailed:
			step["current"] = true
		}
		steps = append(steps, step)
	}
	if status == model.AuditStatusCompleted {
		steps = append(steps, map[string]interface{}{"key": "done", "label": "已完成", "done": true})
	}
	return steps
}

func parseArchiveSnapshotValidIDs(raw datatypes.JSON) []uuid.UUID {
	var s []string
	_ = json.Unmarshal(raw, &s)
	out := make([]uuid.UUID, 0, len(s))
	for _, x := range s {
		id, err := uuid.Parse(strings.TrimSpace(x))
		if err == nil {
			out = append(out, id)
		}
	}
	return out
}
