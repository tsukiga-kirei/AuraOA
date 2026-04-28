// Package excel 提供 Excel 导入/导出的公共工具，包括国际化映射、文件构建和解析功能。
package excel

import (
	"strings"

	"github.com/gin-gonic/gin"

	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
)

// Locale 表示支持的语言代码。
type Locale string

const (
	// LocaleZH 表示简体中文。
	LocaleZH Locale = "zh-CN"
	// LocaleEN 表示英文。
	LocaleEN Locale = "en-US"
)

// ExportType 表示导出场景类型，决定导出文件的列结构。
type ExportType int

const (
	// ExportTypeAuditUnaudited 审核工作台 — 待审核页签（7 列）。
	ExportTypeAuditUnaudited ExportType = iota
	// ExportTypeAuditCompleted 审核工作台 — 已完成页签（11 列）。
	ExportTypeAuditCompleted
	// ExportTypeArchiveUnaudited 归档复盘 — 未复核页签（6 列）。
	ExportTypeArchiveUnaudited
	// ExportTypeArchiveReviewed 归档复盘 — 已复核页签（10 列）。
	ExportTypeArchiveReviewed
	// ExportTypeUserConfig 用户偏好导出（8 列）。
	ExportTypeUserConfig
)

// EnumType 表示需要翻译的枚举字段类型。
type EnumType int

const (
	// EnumAuditRecommendation 审核建议枚举（approve / reject / review）。
	EnumAuditRecommendation EnumType = iota
	// EnumAuditStatus 审核状态枚举（pending_ai / completed / failed）。
	EnumAuditStatus
	// EnumCompliance 合规性枚举（compliant / partially_compliant / non_compliant）。
	EnumCompliance
	// EnumMemberStatus 成员状态枚举（active / inactive）。
	EnumMemberStatus
)

// colHeadersMap 存储各导出类型在不同语言下的列头。
// 外层 key 为 ExportType，内层 key 为 Locale。
var colHeadersMap = map[ExportType]map[Locale][]string{
	ExportTypeAuditUnaudited: {
		LocaleZH: {"流程编号", "流程标题", "申请人", "部门", "流程类型", "提交时间", "当前节点"},
		LocaleEN: {"Process ID", "Title", "Applicant", "Department", "Process Type", "Submit Time", "Current Node"},
	},
	ExportTypeAuditCompleted: {
		LocaleZH: {"流程编号", "流程标题", "申请人", "部门", "流程类型", "提交时间", "当前节点", "审核建议", "评分", "审核状态", "审核时间"},
		LocaleEN: {"Process ID", "Title", "Applicant", "Department", "Process Type", "Submit Time", "Current Node", "Recommendation", "Score", "Audit Status", "Audit Time"},
	},
	ExportTypeArchiveUnaudited: {
		LocaleZH: {"流程编号", "流程标题", "申请人", "部门", "流程类型", "归档时间"},
		LocaleEN: {"Process ID", "Title", "Applicant", "Department", "Process Type", "Archive Time"},
	},
	ExportTypeArchiveReviewed: {
		LocaleZH: {"流程编号", "流程标题", "申请人", "部门", "流程类型", "归档时间", "合规性", "合规评分", "置信度", "复核时间"},
		LocaleEN: {"Process ID", "Title", "Applicant", "Department", "Process Type", "Archive Time", "Compliance", "Score", "Confidence", "Review Time"},
	},
	ExportTypeUserConfig: {
		LocaleZH: {"用户名", "显示名", "部门", "角色", "审核流程数", "归档流程数", "定时任务数", "最近修改时间"},
		LocaleEN: {"Username", "Display Name", "Department", "Roles", "Audit Processes", "Archive Processes", "Cron Tasks", "Last Modified"},
	},
}

// enumTranslationsMap 存储各枚举类型的翻译映射。
// 结构：enumType -> rawValue -> locale -> translatedText
var enumTranslationsMap = map[EnumType]map[string]map[Locale]string{
	EnumAuditRecommendation: {
		"approve": {LocaleZH: "通过", LocaleEN: "Approve"},
		"reject":  {LocaleZH: "拒绝", LocaleEN: "Reject"},
		"review":  {LocaleZH: "复核", LocaleEN: "Review"},
	},
	EnumAuditStatus: {
		"pending_ai": {LocaleZH: "待审核", LocaleEN: "Pending"},
		"completed":  {LocaleZH: "已完成", LocaleEN: "Completed"},
		"failed":     {LocaleZH: "失败", LocaleEN: "Failed"},
	},
	EnumCompliance: {
		"compliant":           {LocaleZH: "合规", LocaleEN: "Compliant"},
		"partially_compliant": {LocaleZH: "部分合规", LocaleEN: "Partially Compliant"},
		"non_compliant":       {LocaleZH: "不合规", LocaleEN: "Non-Compliant"},
	},
	EnumMemberStatus: {
		"active":   {LocaleZH: "启用", LocaleEN: "Active"},
		"inactive": {LocaleZH: "禁用", LocaleEN: "Inactive"},
	},
}

// ResolveLocale 从 gin.Context 中解析用户语言偏好。
// 解析优先级：
//  1. JWT claims.Locale（已认证用户的语言设置）
//  2. Accept-Language 请求头（取第一个语言标签）
//  3. 默认 fallback 到 zh-CN
func ResolveLocale(c *gin.Context) Locale {
	// 1. 优先读取 JWT claims 中的 Locale
	if claimsVal, exists := c.Get("jwt_claims"); exists {
		if claims, ok := claimsVal.(*jwtpkg.JWTClaims); ok && claims.Locale != "" {
			loc := Locale(claims.Locale)
			if loc == LocaleZH || loc == LocaleEN {
				return loc
			}
		}
	}

	// 2. 读取 Accept-Language 请求头，取第一个语言标签
	acceptLang := c.GetHeader("Accept-Language")
	if acceptLang != "" {
		// Accept-Language 可能形如 "en-US,en;q=0.9,zh-CN;q=0.8"
		// 取第一个逗号前的部分，再去掉权重后缀
		first := strings.SplitN(acceptLang, ",", 2)[0]
		tag := strings.SplitN(strings.TrimSpace(first), ";", 2)[0]
		tag = strings.TrimSpace(tag)
		loc := Locale(tag)
		if loc == LocaleZH || loc == LocaleEN {
			return loc
		}
	}

	// 3. 默认 fallback 到 zh-CN
	return LocaleZH
}

// ColHeaders 返回指定导出类型在给定语言下的列头切片。
// 若 locale 无对应映射，则 fallback 到 zh-CN。
// 若 exportType 未定义，则返回空切片。
func ColHeaders(exportType ExportType, locale Locale) []string {
	localeMap, ok := colHeadersMap[exportType]
	if !ok {
		return []string{}
	}

	headers, ok := localeMap[locale]
	if !ok {
		// fallback 到 zh-CN
		headers = localeMap[LocaleZH]
	}
	return headers
}

// TranslateEnum 将数据库枚举原始值转换为对应语言的可读文本。
// fallback 规则：
//  1. 若目标 locale 无映射，则尝试 zh-CN 翻译
//  2. 若 zh-CN 也无映射，则返回原始值 value
func TranslateEnum(enumType EnumType, value string, locale Locale) string {
	valueMap, ok := enumTranslationsMap[enumType]
	if !ok {
		return value
	}

	localeMap, ok := valueMap[value]
	if !ok {
		return value
	}

	if text, ok := localeMap[locale]; ok {
		return text
	}

	// fallback 到 zh-CN
	if text, ok := localeMap[LocaleZH]; ok {
		return text
	}

	return value
}
