package service

import (
	"time"

	"github.com/gin-gonic/gin"
	"oa-smart-audit/go-service/internal/repository"
)

// ReportCalculatorService 提供日报、周报所需的数据计算公共方法。
type ReportCalculatorService struct {
	auditLogRepo    *repository.AuditLogRepo
	archiveLogRepo  *repository.ArchiveLogRepo
	tenantRepo      *repository.TenantRepo
}

func NewReportCalculatorService(
	auditLogRepo *repository.AuditLogRepo,
	archiveLogRepo *repository.ArchiveLogRepo,
	tenantRepo *repository.TenantRepo,
) *ReportCalculatorService {
	return &ReportCalculatorService{
		auditLogRepo:    auditLogRepo,
		archiveLogRepo:  archiveLogRepo,
		tenantRepo:      tenantRepo,
	}
}

// Stats 包含报告所需的基础变量
type ReportStats struct {
	AuditStats   *repository.AuditLogStats
	ArchiveStats *repository.ArchiveLogStats
	TimeRange    string
	TenantName   string
}

// CalculateAuditStats 计算指定时间段内的审核统计
func (s *ReportCalculatorService) CalculateAuditStats(c *gin.Context, start, end time.Time) (*repository.AuditLogStats, error) {
	return s.auditLogRepo.CountStatsByTimeRange(c, start, end)
}

// CalculateArchiveStats 计算指定时间段内的归档统计
func (s *ReportCalculatorService) CalculateArchiveStats(c *gin.Context, start, end time.Time) (*repository.ArchiveLogStats, error) {
	return s.archiveLogRepo.CountStatsByTimeRange(c, start, end)
}

// GetSummaryVariables 获取汇总后的文字变量（供邮件模板使用）
func (s *ReportCalculatorService) GetSummaryVariables(stats *ReportStats) map[string]interface{} {
	vars := make(map[string]interface{})
	vars["tenant_name"] = stats.TenantName
	vars["time_range"] = stats.TimeRange

	if stats.AuditStats != nil {
		vars["audit_total"] = stats.AuditStats.Total
		vars["audit_approve"] = stats.AuditStats.ApproveCount
		vars["audit_return"] = stats.AuditStats.ReturnCount
		vars["audit_review"] = stats.AuditStats.ReviewCount
		vars["audit_pass_rate"] = 0.0
		if stats.AuditStats.Total > 0 {
			vars["audit_pass_rate"] = float64(stats.AuditStats.ApproveCount) * 100 / float64(stats.AuditStats.Total)
		}
	}

	if stats.ArchiveStats != nil {
		vars["archive_total"] = stats.ArchiveStats.Total
		vars["archive_compliant"] = stats.ArchiveStats.Compliant
		vars["archive_partial"] = stats.ArchiveStats.Partial
		vars["archive_non_compliant"] = stats.ArchiveStats.NonCompliant
		vars["archive_pass_rate"] = 0.0
		if stats.ArchiveStats.Total > 0 {
			vars["archive_pass_rate"] = float64(stats.ArchiveStats.Compliant) * 100 / float64(stats.ArchiveStats.Total)
		}
	}

	return vars
}
