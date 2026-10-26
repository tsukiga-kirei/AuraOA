package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/ai"
	"auraoa/go-service/internal/pkg/sanitize"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const defaultRecheckSystemPrompt = `你负责复核结构化审核结果，检查规则判断、风险、扣分、分数与最终结论是否一致。

1. 核对每条规则的通过状态与理由。同一要求不能既判定通过，又说明尚未满足；归档的流程完整性、节点和字段判断也须一致。
2. 结果中存在未通过项或尚未排除的实际风险时，不能给出通过或合规结论，也不能给出80分及以上的分数。纯优化建议不作为风险或未通过项。
3. 结果已确认且需要整改的问题，对应退回或不合规，总分0至59；结果中的事实冲突、证据不足等待核实问题，对应复核或部分合规，总分60至79；无未通过项且无未解决风险时，才可通过或合规，总分80至100。
4. 提供了扣分明细时，检查总分是否等于100减去扣分合计，以及同一问题是否重复扣分；未提供明细时，不编造扣分项。
5. 依据输入内容纠正相互矛盾的判断、风险、分数和结论，说明具体修改原因，返回完整最终结果；没有发现矛盾时保留原结果。不臆测原始业务事实，不因结果未展开原始材料而新增问题。`
const defaultRecheckUserPrompt = `【待复核的结构化审核结果】
{{initial_result}}

请检查通过状态、判定理由、风险、分数与最终结论是否一致，说明复核发现，并返回完整最终结果。`

// normalizeAIRecheckConfig 仅在配置保存时补齐默认提示词，不给旧快照自动开启复核。
func normalizeAIRecheckConfig(cfg *model.AIRecheckConfig) {
	if cfg == nil {
		return
	}
	if strings.TrimSpace(cfg.SystemPrompt) == "" {
		cfg.SystemPrompt = defaultRecheckSystemPrompt
	}
	if strings.TrimSpace(cfg.UserPrompt) == "" {
		cfg.UserPrompt = defaultRecheckUserPrompt
	}
}

// normalizeSavedRecheckConfig 仅规范化明确提交的复核配置，缺省字段保持不存在。
func normalizeSavedRecheckConfig(raw []byte) (json.RawMessage, error) {
	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("AI配置必须为对象")
	}
	value, ok := values["recheck"]
	if !ok || string(value) == "null" {
		return raw, nil
	}
	var cfg model.AIRecheckConfig
	if err := json.Unmarshal(value, &cfg); err != nil {
		return nil, fmt.Errorf("AI复核配置格式无效")
	}
	if len([]rune(cfg.SystemPrompt)) > 100000 || len([]rune(cfg.UserPrompt)) > 100000 {
		return nil, fmt.Errorf("单段复核提示词不能超过100000个字符")
	}
	normalizeAIRecheckConfig(&cfg)
	values["recheck"], _ = json.Marshal(cfg)
	normalized, _ := json.Marshal(values)
	return normalized, nil
}

type recheckModelCaller interface {
	ChatWithFallback(*gin.Context, uuid.UUID, uuid.UUID, *model.AIModelConfig, *model.AIModelConfig, *ai.ChatRequest) (*ai.ChatResponse, error)
}

// buildAIRecheckRequest 默认只传结构化结果，其余初审内容仅在模板显式引用变量时传入。
func buildAIRecheckRequest(cfg *model.AIRecheckConfig, original *ai.ChatRequest, reasoning string, initial json.RawMessage) *ai.ChatRequest {
	values := []string{"{{original_system_prompt}}", original.SystemPrompt, "{{audit_context}}", original.UserPrompt, "{{reasoning_result}}", reasoning, "{{initial_result}}", string(initial)}
	replacer := strings.NewReplacer(values...)
	// 先替换模板中的日期变量，避免把业务内容里的同名文字当作模板再次展开。
	system := replacer.Replace(replaceSystemPromptVariables(cfg.SystemPrompt))
	user := replacer.Replace(replaceSystemPromptVariables(cfg.UserPrompt))
	// 结构化结果是复核必需输入，未引用时只补入这一项；其他材料不自动追加。
	if !strings.Contains(cfg.SystemPrompt, "{{initial_result}}") && !strings.Contains(cfg.UserPrompt, "{{initial_result}}") {
		user += "\n\n【初审结构化结果】\n" + string(initial)
	}
	system += `

【固定输出格式】仅返回JSON对象，结构为 {"reason":"基于证据的复核说明","final_result":完整最终结构化结果}。
reason为必填的复核说明。final_result必须是完整最终结果对象，保持本次初审结构化结果的字段、数组结构及规则项，不能只返回修改项，不能遗漏或新增规则。结论只能使用原枚举，分数和置信度为0至100整数，结论须与分数区间一致。
`
	return &ai.ChatRequest{SystemPrompt: system, UserPrompt: user, RequestType: original.RequestType, CallType: "recheck", Temperature: 0.1, MaxTokens: original.MaxTokens, ModelConfig: original.ModelConfig, ProcessID: original.ProcessID, ProcessTitle: original.ProcessTitle, BusinessLogID: original.BusinessLogID}
}

// runAIRecheck 通过统一入口执行一次可选复核，持久化回调负责租户隔离和取消保护。
func runAIRecheck(c *gin.Context, caller recheckModelCaller, tenantID, userID uuid.UUID, primary, fallback *model.AIModelConfig, cfg *model.AIRecheckConfig, original *ai.ChatRequest, reasoning string, initial json.RawMessage, scrub bool, persist func(*model.AIRecheckRecord) error) (json.RawMessage, *model.AIRecheckRecord, error) {
	if cfg == nil || !cfg.Enabled {
		return initial, nil, nil
	}
	record := &model.AIRecheckRecord{Status: "running", OriginalResult: initial}
	if err := c.Request.Context().Err(); err != nil {
		return nil, record, err
	}
	if err := persist(record); err != nil {
		return nil, record, err
	}
	started := time.Now()
	resp, err := caller.ChatWithFallback(c, tenantID, userID, primary, fallback, buildAIRecheckRequest(cfg, original, reasoning, initial))
	record.DurationMs = time.Since(started).Milliseconds()
	if err == nil {
		err = c.Request.Context().Err()
	}
	if err == nil && resp == nil {
		err = fmt.Errorf("AI复核返回为空")
	}
	if err == nil {
		var envelope struct {
			Reason      string          `json:"reason"`
			FinalResult json.RawMessage `json:"final_result"`
		}
		err = json.Unmarshal([]byte(cleanJSONResponse(resp.Content)), &envelope)
		if err == nil && strings.TrimSpace(envelope.Reason) == "" {
			err = fmt.Errorf("复核缺少说明")
		}
		if err == nil {
			err = validateAIRecheckResult(envelope.FinalResult, initial, original.RequestType)
		}
		if err == nil {
			if scrub {
				envelope.Reason = sanitize.SanitizeText(envelope.Reason)
				envelope.FinalResult = json.RawMessage(sanitize.SanitizeText(string(envelope.FinalResult)))
				if !json.Valid(envelope.FinalResult) {
					err = fmt.Errorf("脱敏后的复核结果格式无效")
				}
			}
		}
		if err == nil {
			record.Status = "completed"
			record.Reason = envelope.Reason
			record.FinalResult = envelope.FinalResult
			var before, after interface{}
			_ = json.Unmarshal(initial, &before)
			_ = json.Unmarshal(envelope.FinalResult, &after)
			record.Changed = !reflect.DeepEqual(before, after)
		}
	}
	if err != nil {
		record.Status = "failed"
		record.Error = "AI复核未完成：" + err.Error()
		if c.Request.Context().Err() != nil {
			record.Status = "cancelled"
			record.Error = "AI复核已取消或超时"
		}
		if scrub {
			record.Error = sanitize.SanitizeText(record.Error)
		}
	}
	if saveErr := persist(record); saveErr != nil {
		return nil, record, saveErr
	}
	if err != nil {
		return nil, record, fmt.Errorf("AI复核未完成，请重新执行：%w", err)
	}
	return record.FinalResult, record, nil
}

// validateAIRecheckResult 只校验明确的结构、枚举、数值和规则覆盖，不解释文字的业务含义。
func validateAIRecheckResult(raw, initial json.RawMessage, module string) error {
	var out, before map[string]json.RawMessage
	if json.Unmarshal(raw, &out) != nil || out == nil {
		return fmt.Errorf("复核最终结果不是有效对象")
	}
	_ = json.Unmarshal(initial, &before)
	for key := range before {
		if _, ok := out[key]; !ok {
			return fmt.Errorf("复核缺少字段：%s", key)
		}
	}
	for _, key := range []string{"overall_score", "confidence"} {
		var v int
		if err := json.Unmarshal(out[key], &v); err != nil || string(bytes.TrimSpace(out[key])) == "null" || v < 0 || v > 100 {
			return fmt.Errorf("复核数值字段不合法：%s", key)
		}
	}
	var score int
	_ = json.Unmarshal(out["overall_score"], &score)
	conclusion, ruleKey, nameKey := "recommendation", "rule_results", "rule_content"
	expected := CalibrateAuditRecommendationByScore("", score)
	if module == "archive" {
		conclusion, ruleKey, nameKey = "overall_compliance", "rule_audit", "rule_name"
		expected = CalibrateArchiveComplianceByScore("", score)
	}
	var actual string
	_ = json.Unmarshal(out[conclusion], &actual)
	if actual != expected {
		return fmt.Errorf("复核结论与分数区间不一致")
	}
	for _, key := range []string{"risk_points", "suggestions", ruleKey} {
		if len(bytes.TrimSpace(out[key])) == 0 || bytes.TrimSpace(out[key])[0] != '[' {
			return fmt.Errorf("复核数组字段不合法：%s", key)
		}
	}
	var oldRules, newRules []map[string]json.RawMessage
	if json.Unmarshal(before[ruleKey], &oldRules) != nil || json.Unmarshal(out[ruleKey], &newRules) != nil || len(oldRules) != len(newRules) {
		return fmt.Errorf("复核规则覆盖不完整")
	}
	counts := map[string]int{}
	for _, r := range oldRules {
		var name string
		if json.Unmarshal(r[nameKey], &name) != nil {
			return fmt.Errorf("初审规则名称格式错误")
		}
		counts[name]++
	}
	for _, r := range newRules {
		var passed bool
		var reason string
		reasonKey := "reason"
		if module == "archive" {
			reasonKey = "reasoning"
		}
		if len(r["passed"]) == 0 || string(r["passed"]) == "null" || json.Unmarshal(r["passed"], &passed) != nil || len(r[reasonKey]) == 0 || string(r[reasonKey]) == "null" || json.Unmarshal(r[reasonKey], &reason) != nil {
			return fmt.Errorf("复核规则字段不完整或类型错误")
		}
		var name string
		if json.Unmarshal(r[nameKey], &name) != nil {
			return fmt.Errorf("复核规则名称格式错误")
		}
		counts[name]--
	}
	if deductions, ok := out["deductions"]; ok {
		var items []deductionPayload
		if json.Unmarshal(deductions, &items) != nil {
			return fmt.Errorf("复核扣分明细格式错误")
		}
		total := 0.0
		for _, item := range items {
			if item.Points < 0 || item.Points > 100 {
				return fmt.Errorf("复核扣分范围错误")
			}
			total += item.Points
		}
		if total != float64(100-score) {
			return fmt.Errorf("复核总分与扣分合计不一致")
		}
	}
	for _, n := range counts {
		if n != 0 {
			return fmt.Errorf("复核不能增删规则")
		}
	}
	if module == "archive" {
		if _, err := ParseArchiveReviewResult(string(raw)); err != nil {
			return err
		}
		var oldFlow, newFlow map[string]json.RawMessage
		_ = json.Unmarshal(before["flow_audit"], &oldFlow)
		_ = json.Unmarshal(out["flow_audit"], &newFlow)
		if err := validateRecheckRowCoverage(oldFlow["node_results"], newFlow["node_results"], "node_id", "node_name"); err != nil {
			return err
		}
		return validateRecheckRowCoverage(before["field_audit"], out["field_audit"], "field_key", "field_name")
	}
	_, err := ParseAuditResult(string(raw))
	return err
}

// validateRecheckRowCoverage 允许调整判断与说明、重排记录，但不允许删除或替换已核验的节点与字段。
func validateRecheckRowCoverage(before, after json.RawMessage, idKey, nameKey string) error {
	identities := func(raw json.RawMessage) (map[string]int, error) {
		var rows []map[string]json.RawMessage
		if json.Unmarshal(raw, &rows) != nil || rows == nil {
			return nil, fmt.Errorf("复核核验明细结构无效")
		}
		counts := map[string]int{}
		for _, row := range rows {
			var id, name string
			if json.Unmarshal(row[idKey], &id) != nil || json.Unmarshal(row[nameKey], &name) != nil {
				return nil, fmt.Errorf("复核核验明细缺少标识或名称")
			}
			counts[id+"\x00"+name]++
		}
		return counts, nil
	}
	oldIDs, err := identities(before)
	if err != nil {
		return err
	}
	newIDs, err := identities(after)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(oldIDs, newIDs) {
		return fmt.Errorf("复核不能增删或替换原有核验节点与字段")
	}
	return nil
}

// persistRecheckJSON 保留合法 JSON，供业务日志和结果详情共用。
func persistRecheckJSON(record *model.AIRecheckRecord) []byte {
	raw, _ := json.Marshal(record)
	return raw
}
