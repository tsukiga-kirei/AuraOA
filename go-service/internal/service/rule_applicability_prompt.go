package service

import "strings"

// ruleApplicabilityInstructions 在推理与提取阶段统一适用前提语义，不改变通过/不通过结果结构。
const ruleApplicabilityInstructions = `【规则适用前提】
每条规则的“适用前提”和“规则内容”是一组，适用前提只决定是否需要校验该规则。
1. 适用前提为空或未提供：与原有规则一致，直接校验规则内容，不能因为前提为空就判定通过。
2. 适用前提满足：校验规则内容，按校验结果判定通过或不通过。多个条件按前提原文中的“且/或”等逻辑判断。
3. 适用前提明确不满足：无需校验规则内容，该规则直接通过（passed=true）；reason 或 reasoning 必须以“适用前提不满足，本规则直接通过”开头，并说明判断依据，不因此扣分、增加风险或建议驳回。强制执行规则也遵循此逻辑。
4. 判断前提所需的信息缺失时，不得擅自视为前提不满足；应说明缺失信息并按原有审核尺度处理。
5. 结构化提取必须保留推理阶段对前提的判断，前提不满足的规则仍须输出，保持 passed=true；结果中的 rule_content 或 rule_name 仅填写规则正文，不能拼入适用前提。保持既有 JSON 字段和通过/不通过两种结果，不新增状态。`

// reviewTemplateApplicabilityInstructions 为新版默认模板已有的说明，仅用于避免重复追加旧说明。
const reviewTemplateApplicabilityInstructions = `【规则适用前提】
1. 适用前提为空或已满足时，检查规则正文，按证据判断是否通过。
2. 适用前提明确不满足时，该规则输出 passed=true，理由以“适用前提不满足，本规则直接通过”开头，并说明依据；不因该规则扣分或增加风险。
3. 判断前提所需的信息缺失时，不得当作前提不满足；保留待核实问题，并说明需要补充的证据。
4. 每条输入规则均须保留，不能增删或合并；规则名称只填写规则正文，不拼接适用前提。`

// withRuleApplicabilityInstructions 为已有租户提示词补充统一说明，保留原文且避免重复追加。
func withRuleApplicabilityInstructions(prompt string) string {
	if strings.Contains(prompt, ruleApplicabilityInstructions) || strings.Contains(prompt, reviewTemplateApplicabilityInstructions) {
		return prompt
	}
	return strings.TrimSpace(prompt) + "\n\n" + ruleApplicabilityInstructions
}
