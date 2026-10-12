package service

import (
	"strings"

	"auraoa/go-service/internal/model"
)

// ruleApplicabilityInstructions 在推理与提取阶段统一适用前提语义，不改变通过/不通过结果结构。
const ruleApplicabilityInstructions = `【规则适用前提】
每条规则的“适用前提”和“规则内容”是一组，适用前提只决定是否需要校验该规则。
1. 适用前提为空或未提供：与原有规则一致，直接校验规则内容，不能因为前提为空就判定通过。
2. 适用前提满足：校验规则内容，按校验结果判定通过或不通过。多个条件按前提原文中的“且/或”等逻辑判断。
3. 适用前提明确不满足：无需校验规则内容，该规则直接通过（passed=true）；reason 或 reasoning 必须以“适用前提不满足，本规则直接通过”开头，并说明判断依据，不因此扣分、增加风险或建议驳回。强制执行规则也遵循此逻辑。
4. 判断前提所需的信息缺失时，不得擅自视为前提不满足；应说明缺失信息并按原有审核尺度处理。
5. 结构化提取必须保留推理阶段对前提的判断，前提不满足的规则仍须输出，保持 passed=true；结果中的 rule_content 或 rule_name 仅填写规则正文，不能拼入适用前提。保持既有 JSON 字段和通过/不通过两种结果，不新增状态。`

// withRuleApplicabilityInstructions 为已有租户提示词补充统一说明，保留原文且避免重复追加。
func withRuleApplicabilityInstructions(prompt string) string {
	if strings.Contains(prompt, ruleApplicabilityInstructions) {
		return prompt
	}
	return strings.TrimSpace(prompt) + "\n\n" + ruleApplicabilityInstructions
}

// auditRulePassedForUnmetCondition 仅豁免已明确不适用的规则，普通通过不能绕过查询证据检查。
// 审核结果没有规则 ID；同正文结果存在歧义时要求全部明确不适用，保留保守兜底。
func auditRulePassedForUnmetCondition(result *model.AuditResultJSON, evidence QueryEvidence) bool {
	if strings.TrimSpace(evidence.ApplicabilityCondition) == "" {
		return false
	}
	matched := false
	for _, rule := range result.RuleResults {
		if rule.RuleContent != evidence.RuleContent {
			continue
		}
		matched = true
		if !rule.Passed || !strings.HasPrefix(strings.TrimSpace(rule.Reason), "适用前提不满足，本规则直接通过") {
			return false
		}
	}
	return matched
}

// archiveRulePassedForUnmetCondition 与审核侧保持一致，仅豁免明确不适用的归档规则。
func archiveRulePassedForUnmetCondition(result *model.ArchiveResultJSON, evidence QueryEvidence) bool {
	if strings.TrimSpace(evidence.ApplicabilityCondition) == "" {
		return false
	}
	matched := false
	for _, rule := range result.RuleAudit {
		if rule.RuleID != evidence.RuleID && rule.RuleName != evidence.RuleContent {
			continue
		}
		matched = true
		if !rule.Passed || !strings.HasPrefix(strings.TrimSpace(rule.Reasoning), "适用前提不满足，本规则直接通过") {
			return false
		}
	}
	return matched
}
