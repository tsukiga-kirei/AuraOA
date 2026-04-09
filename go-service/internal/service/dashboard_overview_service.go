package service

import (
	"log"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/repository"
)

// DashboardOverviewService 聚合仪表盘数据。
type DashboardOverviewService struct {
	auditSnapshotRepo   *repository.AuditProcessSnapshotRepo
	archiveSnapshotRepo *repository.ArchiveProcessSnapshotRepo
	auditLogRepo        *repository.AuditLogRepo
	archiveLogRepo      *repository.ArchiveLogRepo
	cronLogRepo         *repository.CronLogRepo
	llmLogRepo          *repository.LLMMessageLogRepo
	tenantRepo          *repository.TenantRepo
	orgRepo             *repository.OrgRepo
}

// NewDashboardOverviewService 创建 DashboardOverviewService。
func NewDashboardOverviewService(
	auditSnapshotRepo *repository.AuditProcessSnapshotRepo,
	archiveSnapshotRepo *repository.ArchiveProcessSnapshotRepo,
	auditLogRepo *repository.AuditLogRepo,
	archiveLogRepo *repository.ArchiveLogRepo,
	cronLogRepo *repository.CronLogRepo,
	llmLogRepo *repository.LLMMessageLogRepo,
	tenantRepo *repository.TenantRepo,
	orgRepo *repository.OrgRepo,
) *DashboardOverviewService {
	return &DashboardOverviewService{
		auditSnapshotRepo:   auditSnapshotRepo,
		archiveSnapshotRepo: archiveSnapshotRepo,
		auditLogRepo:        auditLogRepo,
		archiveLogRepo:      archiveLogRepo,
		cronLogRepo:         cronLogRepo,
		llmLogRepo:          llmLogRepo,
		tenantRepo:          tenantRepo,
		orgRepo:             orgRepo,
	}
}

func tenantUUIDFromContext(c *gin.Context) (uuid.UUID, error) {
	tid, ok := c.Get("tenant_id")
	if !ok || tid == nil || tid == "" {
		return uuid.Nil, newServiceError(errcode.ErrParamValidation, "缺少租户上下文，请在系统管理员模式下指定 tenant_id 参数")
	}
	return uuid.Parse(tid.(string))
}

// BuildOverview 构建当前租户仪表盘数据。
// business: 仅个人数据；tenant_admin: 租户级汇总。
func (s *DashboardOverviewService) BuildOverview(c *gin.Context, activeRole string, viewerUserID uuid.UUID, viewerUsername string) (*dto.DashboardOverviewResponse, error) {
	if _, err := tenantUUIDFromContext(c); err != nil {
		return nil, err
	}

	var userScope *uuid.UUID
	if activeRole == "business" {
		u := viewerUserID
		userScope = &u
	}

	out := &dto.DashboardOverviewResponse{}

	// ── 本周概览（快照表 + cron_logs）──
	auditWeek, err := s.auditSnapshotRepo.CountThisWeek(c, userScope)
	if err != nil {
		log.Printf("dashboard: auditSnapshotRepo.CountThisWeek error: %v", err)
	}
	archiveWeek, err := s.archiveSnapshotRepo.CountThisWeek(c, userScope)
	if err != nil {
		log.Printf("dashboard: archiveSnapshotRepo.CountThisWeek error: %v", err)
	}
	cronWeek, err := s.cronLogRepo.CountThisWeek(c, userScope)
	if err != nil {
		log.Printf("dashboard: cronLogRepo.CountThisWeek error: %v", err)
	}
	out.WeeklyOverview = &dto.WeeklyOverviewData{
		AuditCount:   auditWeek,
		ArchiveCount: archiveWeek,
		CronCount:    cronWeek,
		Total:        auditWeek + archiveWeek + cronWeek,
	}

	// ── 审核趋势（堆叠柱状图）──
	auditTrend, _ := s.auditSnapshotRepo.WeeklyTrendByDay(c, userScope)
	cronTrend, _ := s.cronLogRepo.WeeklyTrendByDay(c, userScope)
	archiveTrend, _ := s.archiveSnapshotRepo.WeeklyTrendByDay(c, userScope)
	out.WeeklyTrend = mergeWeeklyTrend(auditTrend, cronTrend, archiveTrend)

	// ── 最近动态（前 10 条，带详细标注）──
	out.RecentActivity = s.buildEnrichedActivity(c, userScope, viewerUsername, 10)

	// ── business 专属 ──
	if activeRole == "business" {
		out.PendingTasks = s.buildPendingTasks(c, viewerUserID)
		out.CronTasks = s.buildCronTaskPreview(c, viewerUserID, viewerUsername)
	}

	// ── tenant_admin 专属 ──
	if activeRole == "tenant_admin" {
		out.DeptDistribution = s.buildDeptDistribution(c)
		out.UserActivity = s.buildUserActivityRanking(c)
	}

	return out, nil
}

// BuildPlatformOverview 系统管理员全平台仪表盘（不依赖 tenant_id）。
func (s *DashboardOverviewService) BuildPlatformOverview() (*dto.PlatformDashboardOverviewResponse, error) {
	out := &dto.PlatformDashboardOverviewResponse{}

	// ── 租户规模 ──
	out.TenantStats = s.buildTenantStats()

	// ── AI 模型表现 ──
	out.AIPerformance = s.buildAIPerformanceByModel()

	// ── 租户资源用量 ──
	out.TenantUsageList = s.buildTenantUsageList()

	// ── 租户审核排名 ──
	out.TenantRanking = s.buildTenantRankingEnriched()

	return out, nil
}

// ── 辅助方法 ──────────────────────────────────────────────────────────────

// mergeWeeklyTrend 合并三个功能的每日数据为堆叠柱状图格式。
func mergeWeeklyTrend(audit, cron, archive []repository.DayCount) []dto.WeeklyTrendDayData {
	dateMap := make(map[string]*dto.WeeklyTrendDayData)
	var dates []string

	for _, d := range audit {
		if _, ok := dateMap[d.Date]; !ok {
			dateMap[d.Date] = &dto.WeeklyTrendDayData{Date: d.Date}
			dates = append(dates, d.Date)
		}
		dateMap[d.Date].AuditCount = d.Count
	}
	for _, d := range cron {
		if _, ok := dateMap[d.Date]; !ok {
			dateMap[d.Date] = &dto.WeeklyTrendDayData{Date: d.Date}
			dates = append(dates, d.Date)
		}
		dateMap[d.Date].CronCount = d.Count
	}
	for _, d := range archive {
		if _, ok := dateMap[d.Date]; !ok {
			dateMap[d.Date] = &dto.WeeklyTrendDayData{Date: d.Date}
			dates = append(dates, d.Date)
		}
		dateMap[d.Date].ArchiveCount = d.Count
	}

	sort.Strings(dates)
	// 去重
	unique := dates[:0]
	seen := make(map[string]bool)
	for _, d := range dates {
		if !seen[d] {
			seen[d] = true
			unique = append(unique, d)
		}
	}

	result := make([]dto.WeeklyTrendDayData, 0, len(unique))
	for _, d := range unique {
		result = append(result, *dateMap[d])
	}
	return result
}

type enrichedSort struct {
	at   time.Time
	item dto.ActivityItemEnriched
}

// buildEnrichedActivity 构建带标注的最近动态。
func (s *DashboardOverviewService) buildEnrichedActivity(c *gin.Context, userScope *uuid.UUID, viewerUsername string, limit int) []dto.ActivityItemEnriched {
	// 审核快照
	auditRows, err := s.auditSnapshotRepo.RecentEnriched(c, limit, userScope)
	if err != nil {
		log.Printf("dashboard: auditSnapshotRepo.RecentEnriched error: %v", err)
	}
	// 归档快照
	archiveRows, err := s.archiveSnapshotRepo.RecentEnriched(c, limit, userScope)
	if err != nil {
		log.Printf("dashboard: archiveSnapshotRepo.RecentEnriched error: %v", err)
	}
	// 定时任务日志
	tid, _ := tenantUUIDFromContext(c)
	cronRows, err := s.cronLogRepo.RecentEnriched(tid, limit, userScope)
	if err != nil {
		log.Printf("dashboard: cronLogRepo.RecentEnriched error: %v", err)
	}

	var buf []enrichedSort
	for _, a := range auditRows {
		buf = append(buf, enrichedSort{
			at: a.CreatedAt,
			item: dto.ActivityItemEnriched{
				ID:             "a-" + a.ID.String(),
				Kind:           "audit",
				Title:          a.Title,
				UserName:       a.UserName,
				CreatedAt:      a.CreatedAt.UTC().Format(time.RFC3339),
				Recommendation: a.Recommendation,
				Score:          a.Score,
			},
		})
	}
	for _, a := range archiveRows {
		buf = append(buf, enrichedSort{
			at: a.CreatedAt,
			item: dto.ActivityItemEnriched{
				ID:              "ar-" + a.ID.String(),
				Kind:            "archive",
				Title:           a.Title,
				UserName:        a.UserName,
				CreatedAt:       a.CreatedAt.UTC().Format(time.RFC3339),
				Compliance:      a.Compliance,
				ComplianceScore: a.ComplianceScore,
			},
		})
	}
	for _, cl := range cronRows {
		buf = append(buf, enrichedSort{
			at: cl.CreatedAt,
			item: dto.ActivityItemEnriched{
				ID:         "c-" + cl.ID.String(),
				Kind:       "cron",
				Title:      cl.TaskLabel,
				UserName:   cl.UserName,
				CreatedAt:  cl.CreatedAt.UTC().Format(time.RFC3339),
				CronStatus: cl.Status,
				TaskLabel:  cl.TaskLabel,
			},
		})
	}

	sort.Slice(buf, func(i, j int) bool { return buf[i].at.After(buf[j].at) })
	if len(buf) > limit {
		buf = buf[:limit]
	}

	result := make([]dto.ActivityItemEnriched, 0, len(buf))
	for _, x := range buf {
		result = append(result, x.item)
	}
	return result
}

// buildPendingTasks 构建待办任务数据（近 90 天）。
func (s *DashboardOverviewService) buildPendingTasks(c *gin.Context, userID uuid.UUID) *dto.PendingTasksData {
	since := time.Now().UTC().AddDate(0, 0, -90)

	auditPending, err := s.auditLogRepo.CountPendingSince(c, &userID, since)
	if err != nil {
		log.Printf("dashboard: auditLogRepo.CountPendingSince error: %v", err)
	}
	archivePending, err := s.archiveLogRepo.CountPendingSince(c, &userID, since)
	if err != nil {
		log.Printf("dashboard: archiveLogRepo.CountPendingSince error: %v", err)
	}

	return &dto.PendingTasksData{
		AuditPending:   auditPending,
		ArchivePending: archivePending,
		Total:          auditPending + archivePending,
	}
}

// buildCronTaskPreview 构建定时任务预览列表。
func (s *DashboardOverviewService) buildCronTaskPreview(c *gin.Context, userID uuid.UUID, username string) []dto.CronTaskPreview {
	tid, err := tenantUUIDFromContext(c)
	if err != nil {
		return nil
	}

	var cronLogs []model.CronLog
	cronLogs, err = s.cronLogRepo.ListByTenantForDashboardMember(tid, userID, username, 5)
	if err != nil {
		log.Printf("dashboard: cronLogRepo.ListByTenantForDashboardMember error: %v", err)
		return nil
	}

	result := make([]dto.CronTaskPreview, 0, len(cronLogs))
	for _, cl := range cronLogs {
		label := cl.TaskLabel
		if label == "" {
			label = cl.TaskType
		}
		result = append(result, dto.CronTaskPreview{
			ID:          cl.ID.String(),
			TaskLabel:   label,
			TaskType:    cl.TaskType,
			Description: label,
			IsActive:    cl.Status == string(model.AuditStatusCompleted),
		})
	}
	return result
}

// buildDeptDistribution 构建部门分布数据（三个功能分别统计）。
func (s *DashboardOverviewService) buildDeptDistribution(c *gin.Context) []dto.DeptDistributionData {
	auditDepts, _ := s.auditSnapshotRepo.CountByDepartment(c)
	archiveDepts, _ := s.archiveSnapshotRepo.CountByDepartment(c)
	cronDepts, _ := s.cronLogRepo.CountByDepartment(c)

	deptMap := make(map[string]*dto.DeptDistributionData)
	for _, d := range auditDepts {
		if _, ok := deptMap[d.Department]; !ok {
			deptMap[d.Department] = &dto.DeptDistributionData{Department: d.Department}
		}
		deptMap[d.Department].AuditCount = d.Count
	}
	for _, d := range archiveDepts {
		if _, ok := deptMap[d.Department]; !ok {
			deptMap[d.Department] = &dto.DeptDistributionData{Department: d.Department}
		}
		deptMap[d.Department].ArchiveCount = d.Count
	}
	for _, d := range cronDepts {
		if _, ok := deptMap[d.Department]; !ok {
			deptMap[d.Department] = &dto.DeptDistributionData{Department: d.Department}
		}
		deptMap[d.Department].CronCount = d.Count
	}

	result := make([]dto.DeptDistributionData, 0, len(deptMap))
	for _, v := range deptMap {
		v.Total = v.AuditCount + v.CronCount + v.ArchiveCount
		result = append(result, *v)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Total > result[j].Total })
	if len(result) > 12 {
		result = result[:12]
	}
	return result
}

// buildUserActivityRanking 构建用户活跃排名（基于快照数据）。
func (s *DashboardOverviewService) buildUserActivityRanking(c *gin.Context) []dto.DashboardUserActivityRow {
	rows, err := s.auditSnapshotRepo.CountByUserRanking(c, 10)
	if err != nil {
		log.Printf("dashboard: auditSnapshotRepo.CountByUserRanking error: %v", err)
		return nil
	}
	result := make([]dto.DashboardUserActivityRow, 0, len(rows))
	for _, u := range rows {
		result = append(result, dto.DashboardUserActivityRow{
			Username:    u.Username,
			DisplayName: u.DisplayName,
			Department:  u.Department,
			AuditCount:  u.AuditCount,
			LastActive:  u.LastActive.UTC().Format(time.RFC3339),
		})
	}
	return result
}

// ── 系统管理员辅助方法 ──────────────────────────────────────────────────────

// buildTenantStats 构建租户规模数据（含人员数量 + 活跃判断）。
func (s *DashboardOverviewService) buildTenantStats() *dto.PlatformTenantStatsData {
	tenants, err := s.tenantRepo.DashboardTenantListWithUserCount()
	if err != nil {
		log.Printf("dashboard: tenantRepo.DashboardTenantListWithUserCount error: %v", err)
		return &dto.PlatformTenantStatsData{ActiveCriteria: "近30天内有审核或归档复盘快照记录"}
	}

	activeIDs, err := s.tenantRepo.DashboardActiveTenantIDs()
	if err != nil {
		log.Printf("dashboard: tenantRepo.DashboardActiveTenantIDs error: %v", err)
		activeIDs = make(map[string]bool)
	}

	rows := make([]dto.TenantStatsRow, 0, len(tenants))
	var activeCount int64
	for _, t := range tenants {
		isActive := activeIDs[t.TenantID.String()]
		if isActive {
			activeCount++
		}
		rows = append(rows, dto.TenantStatsRow{
			TenantID:   t.TenantID.String(),
			TenantName: t.TenantName,
			TenantCode: t.TenantCode,
			UserCount:  t.UserCount,
			IsActive:   isActive,
		})
	}

	return &dto.PlatformTenantStatsData{
		TenantTotal:    int64(len(tenants)),
		TenantActive:   activeCount,
		ActiveCriteria: "近30天内有审核或归档复盘快照记录",
		Tenants:        rows,
	}
}

// buildAIPerformanceByModel 构建按模型分组的 AI 性能数据。
func (s *DashboardOverviewService) buildAIPerformanceByModel() *dto.PlatformAIPerformanceData {
	stats, err := s.llmLogRepo.DashboardAIPerformanceByModel()
	if err != nil {
		log.Printf("dashboard: llmLogRepo.DashboardAIPerformanceByModel error: %v", err)
		return &dto.PlatformAIPerformanceData{Models: []dto.AIModelPerformanceRow{}}
	}

	// 按 model_config_id 分组
	type modelGroup struct {
		ModelConfigID string
		ModelName     string
		DisplayName   string
		Provider      string
		Reasoning     dto.AICallTypeStats
		Structured    dto.AICallTypeStats
	}
	groupMap := make(map[string]*modelGroup)
	var order []string

	for _, s := range stats {
		if _, ok := groupMap[s.ModelConfigID]; !ok {
			groupMap[s.ModelConfigID] = &modelGroup{
				ModelConfigID: s.ModelConfigID,
				ModelName:     s.ModelName,
				DisplayName:   s.DisplayName,
				Provider:      s.Provider,
			}
			order = append(order, s.ModelConfigID)
		}
		g := groupMap[s.ModelConfigID]
		ct := dto.AICallTypeStats{Calls: s.Calls, AvgMs: s.AvgMs, SuccessRate: 100.0}
		if s.CallType == "structured" {
			g.Structured = ct
		} else {
			g.Reasoning = ct
		}
	}

	models := make([]dto.AIModelPerformanceRow, 0, len(order))
	for _, id := range order {
		g := groupMap[id]
		total := g.Reasoning.Calls + g.Structured.Calls
		overallRate := 100.0
		if total > 0 {
			overallRate = (g.Reasoning.SuccessRate*float64(g.Reasoning.Calls) + g.Structured.SuccessRate*float64(g.Structured.Calls)) / float64(total)
		}
		models = append(models, dto.AIModelPerformanceRow{
			ModelConfigID:      g.ModelConfigID,
			ModelName:          g.ModelName,
			DisplayName:        g.DisplayName,
			Provider:           g.Provider,
			ReasoningStats:     g.Reasoning,
			StructuredStats:    g.Structured,
			OverallSuccessRate: overallRate,
			TotalCalls:         total,
		})
	}

	return &dto.PlatformAIPerformanceData{Models: models}
}

// buildTenantUsageList 构建按租户分列的资源用量。
func (s *DashboardOverviewService) buildTenantUsageList() []dto.TenantUsageRow {
	rows, err := s.tenantRepo.DashboardTenantTokenList()
	if err != nil {
		log.Printf("dashboard: tenantRepo.DashboardTenantTokenList error: %v", err)
		return nil
	}
	result := make([]dto.TenantUsageRow, 0, len(rows))
	for _, r := range rows {
		result = append(result, dto.TenantUsageRow{
			TenantID:   r.TenantID.String(),
			TenantName: r.TenantName,
			TenantCode: r.TenantCode,
			TokenUsed:  r.TokenUsed,
			TokenQuota: r.TokenQuota,
		})
	}
	return result
}

// buildTenantRankingEnriched 构建含失败记录的租户排名。
func (s *DashboardOverviewService) buildTenantRankingEnriched() []dto.PlatformTenantRankRowEnriched {
	auditCounts, _ := s.auditSnapshotRepo.CountByTenantGlobal()
	archiveCounts, _ := s.archiveSnapshotRepo.CountByTenantGlobal()
	cronCounts, _ := s.cronLogRepo.CountByTenantGlobal()
	auditFailed, _ := s.auditLogRepo.CountFailedByTenantGlobal()
	archiveFailed, _ := s.archiveSnapshotRepo.CountFailedByTenantGlobal()

	// 获取租户名称
	tenants, _ := s.tenantRepo.List()
	tenantMap := make(map[string]model.Tenant)
	for _, t := range tenants {
		tenantMap[t.ID.String()] = t
	}

	type rankData struct {
		dto.PlatformTenantRankRowEnriched
		total int64
	}
	dataMap := make(map[string]*rankData)

	for _, a := range auditCounts {
		id := a.TenantID.String()
		if _, ok := dataMap[id]; !ok {
			t := tenantMap[id]
			dataMap[id] = &rankData{PlatformTenantRankRowEnriched: dto.PlatformTenantRankRowEnriched{
				TenantID: id, TenantName: t.Name, TenantCode: t.Code,
			}}
		}
		dataMap[id].AuditCount = a.Count
	}
	for _, a := range archiveCounts {
		id := a.TenantID.String()
		if _, ok := dataMap[id]; !ok {
			t := tenantMap[id]
			dataMap[id] = &rankData{PlatformTenantRankRowEnriched: dto.PlatformTenantRankRowEnriched{
				TenantID: id, TenantName: t.Name, TenantCode: t.Code,
			}}
		}
		dataMap[id].ArchiveCount = a.Count
	}
	for _, a := range cronCounts {
		id := a.TenantID.String()
		if _, ok := dataMap[id]; !ok {
			t := tenantMap[id]
			dataMap[id] = &rankData{PlatformTenantRankRowEnriched: dto.PlatformTenantRankRowEnriched{
				TenantID: id, TenantName: t.Name, TenantCode: t.Code,
			}}
		}
		dataMap[id].CronCount = a.Count
	}
	for _, a := range auditFailed {
		id := a.TenantID.String()
		if _, ok := dataMap[id]; !ok {
			t := tenantMap[id]
			dataMap[id] = &rankData{PlatformTenantRankRowEnriched: dto.PlatformTenantRankRowEnriched{
				TenantID: id, TenantName: t.Name, TenantCode: t.Code,
			}}
		}
		dataMap[id].AuditFailed = a.Count
	}
	for _, a := range archiveFailed {
		id := a.TenantID.String()
		if _, ok := dataMap[id]; !ok {
			t := tenantMap[id]
			dataMap[id] = &rankData{PlatformTenantRankRowEnriched: dto.PlatformTenantRankRowEnriched{
				TenantID: id, TenantName: t.Name, TenantCode: t.Code,
			}}
		}
		dataMap[id].ArchiveFailed = a.Count
	}

	result := make([]rankData, 0, len(dataMap))
	for _, v := range dataMap {
		v.total = v.AuditCount + v.ArchiveCount + v.CronCount
		result = append(result, *v)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].total > result[j].total })

	out := make([]dto.PlatformTenantRankRowEnriched, 0, len(result))
	for _, r := range result {
		out = append(out, r.PlatformTenantRankRowEnriched)
	}
	return out
}
