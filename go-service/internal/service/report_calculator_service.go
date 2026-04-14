package service

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"oa-smart-audit/go-service/internal/repository"
)

// ReportCalculatorService 提供日报、周报所需的数据计算公共方法。
type ReportCalculatorService struct {
	auditLogRepo   *repository.AuditLogRepo
	archiveLogRepo *repository.ArchiveLogRepo
	tenantRepo     *repository.TenantRepo
}

// NewReportCalculatorService 创建 ReportCalculatorService，注入审核、归档日志仓储和租户仓储。
func NewReportCalculatorService(
	auditLogRepo *repository.AuditLogRepo,
	archiveLogRepo *repository.ArchiveLogRepo,
	tenantRepo *repository.TenantRepo,
) *ReportCalculatorService {
	return &ReportCalculatorService{
		auditLogRepo:   auditLogRepo,
		archiveLogRepo: archiveLogRepo,
		tenantRepo:     tenantRepo,
	}
}

// ReportStats 报告所需的基础统计数据，包含审核统计、归档统计、时间范围和租户名称。
type ReportStats struct {
	AuditStats   *repository.AuditLogStats
	ArchiveStats *repository.ArchiveLogStats
	TimeRange    string
	TenantName   string
}

// CalculateAuditStats 计算指定时间段内的审核统计数据。
func (s *ReportCalculatorService) CalculateAuditStats(c *gin.Context, start, end time.Time) (*repository.AuditLogStats, error) {
	return s.auditLogRepo.CountStatsByTimeRange(c, start, end)
}

// CalculateArchiveStats 计算指定时间段内的归档统计数据。
func (s *ReportCalculatorService) CalculateArchiveStats(c *gin.Context, start, end time.Time) (*repository.ArchiveLogStats, error) {
	return s.archiveLogRepo.CountStatsByTimeRange(c, start, end)
}

// GetSummaryVariables 将统计数据转换为邮件模板变量映射，供模板渲染使用。
func (s *ReportCalculatorService) GetSummaryVariables(stats *ReportStats) map[string]interface{} {
	vars := make(map[string]interface{})
	vars["tenant_name"] = stats.TenantName
	vars["time_range"] = stats.TimeRange
	vars["date"] = time.Now().Format("2006-01-02")
	vars["time"] = time.Now().Format("2006-01-02 15:04:05")

	// 默认统计占位符（后续可根据业务逻辑生成详情列表）
	vars["detail_list"] = "（各流程详情请点击进入系统查看）"
	vars["statistics"] = "（完整统计报表请在系统管理页下载）"

	if stats.AuditStats != nil {
		vars["audit_total"] = stats.AuditStats.Total
		vars["audit_approve"] = stats.AuditStats.ApproveCount
		vars["audit_return"] = stats.AuditStats.ReturnCount
		vars["audit_review"] = stats.AuditStats.ReviewCount
		vars["audit_pass_rate"] = 0.0
		if stats.AuditStats.Total > 0 {
			vars["audit_pass_rate"] = float64(stats.AuditStats.ApproveCount) * 100 / float64(stats.AuditStats.Total)
		}

		// 注入别名以兼容默认模板
		vars["total"] = vars["audit_total"]
		vars["approved"] = vars["audit_approve"]
		vars["rejected"] = vars["audit_return"]
		vars["revised"] = vars["audit_review"]
		vars["pass_rate"] = fmt.Sprintf("%.2f", vars["audit_pass_rate"])

		// 周报额外变量占位
		vars["week"] = time.Now().Format("02") // 简单用日期占位或后续计算周数
		vars["trend"] = "持平"
		vars["compliance_rate"] = vars["pass_rate"]
		vars["compliance_trend"] = "稳定"
		vars["date_range"] = stats.TimeRange
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

		// 如果是归档任务，别名映射到归档统计
		if stats.AuditStats == nil || stats.AuditStats.Total == 0 {
			vars["total"] = vars["archive_total"]
			vars["approved"] = vars["archive_compliant"]
			vars["rejected"] = vars["archive_non_compliant"]
			vars["revised"] = vars["archive_partial"]
			vars["pass_rate"] = fmt.Sprintf("%.2f", vars["archive_pass_rate"])

			vars["week"] = time.Now().Format("02")
			vars["trend"] = "持平"
			vars["compliance_rate"] = vars["pass_rate"]
			vars["compliance_trend"] = "稳定"
			vars["date_range"] = stats.TimeRange
		}
	}

	return vars
}
