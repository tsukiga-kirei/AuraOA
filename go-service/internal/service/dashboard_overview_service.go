package service

import (
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
	auditLogRepo       *repository.AuditLogRepo
	archiveLogRepo     *repository.ArchiveLogRepo
	cronLogRepo        *repository.CronLogRepo
	llmLogRepo         *repository.LLMMessageLogRepo
	tenantRepo         *repository.TenantRepo
	orgRepo            *repository.OrgRepo
	auditExecuteSvc    *AuditExecuteService
}

// NewDashboardOverviewService 创建 DashboardOverviewService。
func NewDashboardOverviewService(
	auditLogRepo *repository.AuditLogRepo,
	archiveLogRepo *repository.ArchiveLogRepo,
	cronLogRepo *repository.CronLogRepo,
	llmLogRepo *repository.LLMMessageLogRepo,
	tenantRepo *repository.TenantRepo,
	orgRepo *repository.OrgRepo,
	auditExecuteSvc *AuditExecuteService,
) *DashboardOverviewService {
	return &DashboardOverviewService{
		auditLogRepo:    auditLogRepo,
		archiveLogRepo:  archiveLogRepo,
		cronLogRepo:     cronLogRepo,
		llmLogRepo:      llmLogRepo,
		tenantRepo:      tenantRepo,
		orgRepo:         orgRepo,
		auditExecuteSvc: auditExecuteSvc,
	}
}

func tenantUUIDFromContext(c *gin.Context) (uuid.UUID, error) {
	tid, ok := c.Get("tenant_id")
	if !ok || tid == nil || tid == "" {
		return uuid.Nil, newServiceError(errcode.ErrParamValidation, "缺少租户上下文，请在系统管理员模式下指定 tenant_id 参数")
	}
	return uuid.Parse(tid.(string))
}

// BuildOverview 构建当前租户仪表盘数据；tenant_admin 额外填充管理类字段。
func (s *DashboardOverviewService) BuildOverview(c *gin.Context, activeRole string) (*dto.DashboardOverviewResponse, error) {
	if _, err := tenantUUIDFromContext(c); err != nil {
		return nil, err
	}

	out := &dto.DashboardOverviewResponse{}

	stats, err := s.auditLogRepo.CountStats(c)
	if err != nil {
		return nil, err
	}
	out.AuditSummary.Total = stats.Total
	out.AuditSummary.Approved = stats.ApproveCount
	out.AuditSummary.Returned = stats.ReturnCount
	out.AuditSummary.Review = stats.ReviewCount
	out.AuditSummary.PendingAI = stats.PendingAI

	archived, err := s.archiveLogRepo.CountCompletedArchiveLogs(c)
	if err != nil {
		return nil, err
	}
	out.AuditSummary.Archived = archived

	weekRows, err := s.auditLogRepo.DashboardWeeklyCompletedTrend(c, 7)
	if err != nil {
		return nil, err
	}
	out.WeeklyTrend = make([]dto.DashboardDayCount, 0, len(weekRows))
	for _, row := range weekRows {
		out.WeeklyTrend = append(out.WeeklyTrend, dto.DashboardDayCount{Date: row.Date, Count: row.Count})
	}

	oaStats, err := s.auditExecuteSvc.GetStats(c)
	if err != nil {
		return nil, err
	}
	out.PendingOACount = oaStats["pending_ai_count"]

	if err := s.fillRecentActivity(c, out); err != nil {
		return nil, err
	}

	archRows, err := s.archiveLogRepo.DashboardRecentArchiveLogs(c, 6)
	if err != nil {
		return nil, err
	}
	out.ArchiveRecent = make([]dto.DashboardArchiveRow, 0, len(archRows))
	for _, r := range archRows {
		out.ArchiveRecent = append(out.ArchiveRecent, dto.DashboardArchiveRow{
			ID:          r.ID.String(),
			Title:       r.Title,
			Compliance:  r.Compliance,
			UserName:    r.UserName,
			CreatedAt:   r.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	if activeRole != "tenant_admin" {
		return out, nil
	}

	deptRows, err := s.auditLogRepo.DashboardDeptAuditDistribution(c, 12)
	if err != nil {
		return nil, err
	}
	out.DeptDistribution = make([]dto.DashboardDeptCount, 0, len(deptRows))
	for _, d := range deptRows {
		out.DeptDistribution = append(out.DeptDistribution, dto.DashboardDeptCount{
			Department: d.Department,
			Count:      d.Count,
		})
	}

	completed, failed, err := s.auditLogRepo.DashboardAuditOutcomeForAIStats(c)
	if err != nil {
		return nil, err
	}
	llmCalls, llmAvgMs, err := s.llmLogRepo.DashboardLLMOverallStats(c)
	if err != nil {
		return nil, err
	}
	llmWeek, err := s.llmLogRepo.DashboardLLMWeeklyTrend(c, 7)
	if err != nil {
		return nil, err
	}
	successRate := 100.0
	if completed+failed > 0 {
		successRate = float64(completed) * 100.0 / float64(completed+failed)
	}
	daily := make([]dto.DashboardLLMDailyPoint, 0, len(llmWeek))
	for _, p := range llmWeek {
		daily = append(daily, dto.DashboardLLMDailyPoint{Date: p.Date, AvgMs: p.AvgMs, Calls: p.Calls})
	}
	out.AIPerformance = &dto.DashboardAIPerformance{
		AvgResponseMs: llmAvgMs,
		SuccessRate:   successRate,
		TotalCalls:    llmCalls,
		DailyStats:    daily,
	}

	tid, _ := tenantUUIDFromContext(c)
	tn, err := s.tenantRepo.FindByID(tid)
	if err != nil {
		return nil, err
	}
	totalUsers, err := s.orgRepo.CountActiveMembersInTenant(c)
	if err != nil {
		return nil, err
	}
	since := time.Now().UTC().AddDate(0, 0, -30)
	activeUsers, err := s.auditLogRepo.DashboardDistinctUserCountSince(c, since)
	if err != nil {
		return nil, err
	}
	out.TenantUsage = &dto.DashboardTenantUsage{
		TokenUsed:      int64(tn.TokenUsed),
		TokenQuota:     int64(tn.TokenQuota),
		StorageUsedMB:  0,
		StorageQuotaMB: 0,
		ActiveUsers:    activeUsers,
		TotalUsers:     totalUsers,
	}

	rankRows, err := s.auditLogRepo.DashboardUserAuditRanking(c, 10)
	if err != nil {
		return nil, err
	}
	out.UserActivity = make([]dto.DashboardUserActivityRow, 0, len(rankRows))
	for _, u := range rankRows {
		out.UserActivity = append(out.UserActivity, dto.DashboardUserActivityRow{
			Username:    u.Username,
			DisplayName: u.DisplayName,
			Department:  u.Department,
			AuditCount:  u.AuditCount,
			LastActive:  u.LastActive.UTC().Format(time.RFC3339),
		})
	}

	return out, nil
}

// BuildPlatformOverview 系统管理员全平台仪表盘（不依赖 tenant_id）。
func (s *DashboardOverviewService) BuildPlatformOverview() (*dto.PlatformDashboardOverviewResponse, error) {
	out := &dto.PlatformDashboardOverviewResponse{PendingOACount: 0}

	totalTenants, activeTenants, err := s.tenantRepo.DashboardPlatformTenantCounts()
	if err != nil {
		return nil, err
	}
	out.TenantTotal = totalTenants
	out.TenantActive = activeTenants

	tokenUsed, tokenQuota, err := s.tenantRepo.DashboardPlatformTokenSum()
	if err != nil {
		return nil, err
	}
	out.TokenSummary = &dto.PlatformTokenSummary{TotalUsed: tokenUsed, TotalQuota: tokenQuota}

	stats, err := s.auditLogRepo.CountStatsGlobal()
	if err != nil {
		return nil, err
	}
	out.AuditSummary.Total = stats.Total
	out.AuditSummary.Approved = stats.ApproveCount
	out.AuditSummary.Returned = stats.ReturnCount
	out.AuditSummary.Review = stats.ReviewCount
	out.AuditSummary.PendingAI = stats.PendingAI

	archived, err := s.archiveLogRepo.CountCompletedArchiveLogsGlobal()
	if err != nil {
		return nil, err
	}
	out.AuditSummary.Archived = archived

	weekRows, err := s.auditLogRepo.DashboardWeeklyCompletedTrendGlobal(7)
	if err != nil {
		return nil, err
	}
	out.WeeklyTrend = make([]dto.DashboardDayCount, 0, len(weekRows))
	for _, row := range weekRows {
		out.WeeklyTrend = append(out.WeeklyTrend, dto.DashboardDayCount{Date: row.Date, Count: row.Count})
	}

	if err := s.fillRecentActivityPlatform(out); err != nil {
		return nil, err
	}

	archRows, err := s.archiveLogRepo.DashboardRecentArchiveLogsGlobal(6)
	if err != nil {
		return nil, err
	}
	out.ArchiveRecent = make([]dto.DashboardArchiveRow, 0, len(archRows))
	for _, r := range archRows {
		out.ArchiveRecent = append(out.ArchiveRecent, dto.DashboardArchiveRow{
			ID:         r.ID.String(),
			Title:      r.Title,
			Compliance: r.Compliance,
			UserName:   r.UserName,
			CreatedAt:  r.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	rankRows, err := s.auditLogRepo.DashboardTenantAuditRankingGlobal(10)
	if err != nil {
		return nil, err
	}
	out.TenantRanking = make([]dto.PlatformTenantRankRow, 0, len(rankRows))
	for _, r := range rankRows {
		out.TenantRanking = append(out.TenantRanking, dto.PlatformTenantRankRow{
			TenantID:   r.TenantID.String(),
			TenantName: r.TenantName,
			TenantCode: r.TenantCode,
			AuditCount: r.AuditCount,
		})
	}

	completed, failed, err := s.auditLogRepo.DashboardAuditOutcomeForAIStatsGlobal()
	if err != nil {
		return nil, err
	}
	llmCalls, llmAvgMs, err := s.llmLogRepo.DashboardLLMOverallStatsGlobal()
	if err != nil {
		return nil, err
	}
	llmWeek, err := s.llmLogRepo.DashboardLLMWeeklyTrendGlobal(7)
	if err != nil {
		return nil, err
	}
	successRate := 100.0
	if completed+failed > 0 {
		successRate = float64(completed) * 100.0 / float64(completed+failed)
	}
	daily := make([]dto.DashboardLLMDailyPoint, 0, len(llmWeek))
	for _, p := range llmWeek {
		daily = append(daily, dto.DashboardLLMDailyPoint{Date: p.Date, AvgMs: p.AvgMs, Calls: p.Calls})
	}
	out.AIPerformance = &dto.DashboardAIPerformance{
		AvgResponseMs: llmAvgMs,
		SuccessRate:   successRate,
		TotalCalls:    llmCalls,
		DailyStats:    daily,
	}

	return out, nil
}

func (s *DashboardOverviewService) fillRecentActivityPlatform(out *dto.PlatformDashboardOverviewResponse) error {
	audits, err := s.auditLogRepo.DashboardRecentAuditsGlobal(8)
	if err != nil {
		return err
	}
	cronLogs, err := s.cronLogRepo.ListRecentGlobal(8)
	if err != nil {
		return err
	}
	archives, err := s.archiveLogRepo.DashboardRecentArchiveLogsGlobal(6)
	if err != nil {
		return err
	}

	var buf []activitySort
	for _, a := range audits {
		kind := "audit_completed"
		if a.Status == model.AuditStatusFailed {
			kind = "audit_failed"
		}
		buf = append(buf, activitySort{
			at: a.CreatedAt,
			DashboardActivityItem: dto.DashboardActivityItem{
				ID:        "a-" + a.ID.String(),
				Kind:      kind,
				Title:     a.Title,
				UserName:  a.UserName,
				CreatedAt: a.CreatedAt.UTC().Format(time.RFC3339),
			},
		})
	}
	for _, cl := range cronLogs {
		title := cl.TaskLabel
		if title == "" {
			title = cl.TaskType
		}
		buf = append(buf, activitySort{
			at: cl.StartedAt,
			DashboardActivityItem: dto.DashboardActivityItem{
				ID:        "c-" + cl.ID.String(),
				Kind:      "cron_log",
				Title:     title,
				UserName:  cl.CreatedBy,
				CreatedAt: cl.StartedAt.UTC().Format(time.RFC3339),
			},
		})
	}
	for _, ar := range archives {
		buf = append(buf, activitySort{
			at: ar.CreatedAt,
			DashboardActivityItem: dto.DashboardActivityItem{
				ID:        "ar-" + ar.ID.String(),
				Kind:      "archive_reviewed",
				Title:     ar.Title,
				UserName:  ar.UserName,
				CreatedAt: ar.CreatedAt.UTC().Format(time.RFC3339),
			},
		})
	}

	sort.Slice(buf, func(i, j int) bool { return buf[i].at.After(buf[j].at) })
	if len(buf) > 18 {
		buf = buf[:18]
	}
	out.RecentActivity = make([]dto.DashboardActivityItem, 0, len(buf))
	for _, x := range buf {
		out.RecentActivity = append(out.RecentActivity, x.DashboardActivityItem)
	}
	return nil
}

type activitySort struct {
	at time.Time
	dto.DashboardActivityItem
}

func (s *DashboardOverviewService) fillRecentActivity(c *gin.Context, out *dto.DashboardOverviewResponse) error {
	audits, err := s.auditLogRepo.DashboardRecentAudits(c, 8)
	if err != nil {
		return err
	}
	tid, err := tenantUUIDFromContext(c)
	if err != nil {
		return err
	}
	cronLogs, err := s.cronLogRepo.ListByTenant(tid, 8)
	if err != nil {
		return err
	}
	archives, err := s.archiveLogRepo.DashboardRecentArchiveLogs(c, 6)
	if err != nil {
		return err
	}

	var buf []activitySort
	for _, a := range audits {
		kind := "audit_completed"
		if a.Status == model.AuditStatusFailed {
			kind = "audit_failed"
		}
		buf = append(buf, activitySort{
			at: a.CreatedAt,
			DashboardActivityItem: dto.DashboardActivityItem{
				ID:        "a-" + a.ID.String(),
				Kind:      kind,
				Title:     a.Title,
				UserName:  a.UserName,
				CreatedAt: a.CreatedAt.UTC().Format(time.RFC3339),
			},
		})
	}
	for _, cl := range cronLogs {
		title := cl.TaskLabel
		if title == "" {
			title = cl.TaskType
		}
		buf = append(buf, activitySort{
			at: cl.StartedAt,
			DashboardActivityItem: dto.DashboardActivityItem{
				ID:        "c-" + cl.ID.String(),
				Kind:      "cron_log",
				Title:     title,
				UserName:  cl.CreatedBy,
				CreatedAt: cl.StartedAt.UTC().Format(time.RFC3339),
			},
		})
	}
	for _, ar := range archives {
		buf = append(buf, activitySort{
			at: ar.CreatedAt,
			DashboardActivityItem: dto.DashboardActivityItem{
				ID:        "ar-" + ar.ID.String(),
				Kind:      "archive_reviewed",
				Title:     ar.Title,
				UserName:  ar.UserName,
				CreatedAt: ar.CreatedAt.UTC().Format(time.RFC3339),
			},
		})
	}

	sort.Slice(buf, func(i, j int) bool { return buf[i].at.After(buf[j].at) })
	if len(buf) > 18 {
		buf = buf[:18]
	}
	out.RecentActivity = make([]dto.DashboardActivityItem, 0, len(buf))
	for _, x := range buf {
		out.RecentActivity = append(out.RecentActivity, x.DashboardActivityItem)
	}
	return nil
}
