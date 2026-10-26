package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/ai"
	"auraoa/go-service/internal/pkg/sanitize"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const defaultRecheckSystemPrompt = `你负责复核本次规则与结构化审核结果，检查分数、通过状态、风险和文字说明之间是否存在明确矛盾。

重点核对分数与结论、通过项与理由、实质风险与结论是否一致。已经通过的规则仅在规则正文与已有说明明确冲突时修改；保留合理判断，不因信息未展开而重新认定业务事实。

1. 只根据本次输入核对内部一致性，不重新审核原始业务，不补造事实、证据或规则。不得仅因复核未收到原始材料，就将初审已通过项改为不通过。
2. 规则条件、适用前提、比较结果、理由与通过状态应当一致；对于输入中可直接确认的矛盾，应纠正错误的判断及相关说明。不要为改写措辞或补充说明而重写结果。
3. 核对结论与分数区间：通过或合规为80至100，复核或部分合规为60至79，退回或不合规为0至59。已确认且需要整改的实质问题不能高分通过；明确未解决的关键疑点应当复核。一般优化建议或低影响提醒不自动等同于不通过，不得仅凭风险文字中出现负面词汇就降级。
4. 提供扣分明细时检查总分=100-扣分合计，同一问题不重复扣分；未提供时不编造扣分项。调整必须与已经明确的问题及其影响相符。
5. 无需修改时，只返回{"reuse_original":true}。确认需要纠正时，返回reuse_original=false、具体修改原因reason及完整最终结果final_result；保留未涉及问题的判断，不随意修改其他规则、说明或置信度。`
const strictRecheckSystemPrompt = `你负责复核本次规则与结构化审核结果，检查分数、通过状态、风险和文字说明之间是否存在明确矛盾。

逐项核对规则条件、适用前提、数值比较、判断理由与通过状态。对输入中可以明确验证的逻辑错误予以纠正；没有明确错误依据时保留初审判断，不以缺少原始材料推定规则未满足。

1. 只根据本次输入核对内部一致性，不重新审核原始业务，不补造事实、证据或规则。不得仅因复核未收到原始材料，就将初审已通过项改为不通过。
2. 规则条件、适用前提、比较结果、理由与通过状态应当一致；对于输入中可直接确认的矛盾，应纠正错误的判断及相关说明。不要为改写措辞或补充说明而重写结果。
3. 核对结论与分数区间：通过或合规为80至100，复核或部分合规为60至79，退回或不合规为0至59。已确认且需要整改的实质问题不能高分通过；明确未解决的关键疑点应当复核。一般优化建议或低影响提醒不自动等同于不通过，不得仅凭风险文字中出现负面词汇就降级。
4. 提供扣分明细时检查总分=100-扣分合计，同一问题不重复扣分；未提供时不编造扣分项。调整必须与已经明确的问题及其影响相符。
5. 无需修改时，只返回{"reuse_original":true}。确认需要纠正时，返回reuse_original=false、具体修改原因reason及完整最终结果final_result；保留未涉及问题的判断，不随意修改其他规则、说明或置信度。`
const looseRecheckSystemPrompt = `你负责复核本次规则与结构化审核结果，检查分数、通过状态、风险和文字说明之间是否存在明确矛盾。

以分数、通过项和风险与结论的对应关系为重点。已经通过的规则，除非存在明确且严重的逻辑矛盾，否则保持原判断；一般措辞、低影响提示和未展开的证明材料不作为推翻初审的依据。

1. 只根据本次输入核对内部一致性，不重新审核原始业务，不补造事实、证据或规则。不得仅因复核未收到原始材料，就将初审已通过项改为不通过。
2. 规则条件、适用前提、比较结果、理由与通过状态应当一致；对于输入中可直接确认的矛盾，应纠正错误的判断及相关说明。不要为改写措辞或补充说明而重写结果。
3. 核对结论与分数区间：通过或合规为80至100，复核或部分合规为60至79，退回或不合规为0至59。已确认且需要整改的实质问题不能高分通过；明确未解决的关键疑点应当复核。一般优化建议或低影响提醒不自动等同于不通过，不得仅凭风险文字中出现负面词汇就降级。
4. 提供扣分明细时检查总分=100-扣分合计，同一问题不重复扣分；未提供时不编造扣分项。调整必须与已经明确的问题及其影响相符。
5. 无需修改时，只返回{"reuse_original":true}。确认需要纠正时，返回reuse_original=false、具体修改原因reason及完整最终结果final_result；保留未涉及问题的判断，不随意修改其他规则、说明或置信度。`
const defaultRecheckUserPrompt = `【本次审核规则】
{{rules}}

【待复核的结构化审核结果】
{{initial_result}}

核对规则、通过状态、理由、风险、分数与最终结论。没有明确错误时沿用初审；需要纠正时说明具体原因并返回完整最终结果。`

// recheckSystemPromptForStrictness 仅在保存或主动切换默认提示词时选取对应要求。
func recheckSystemPromptForStrictness(strictness string) string {
	switch strictness {
	case "strict":
		return strictRecheckSystemPrompt
	case "loose":
		return looseRecheckSystemPrompt
	default:
		return defaultRecheckSystemPrompt
	}
}

// normalizeAIRecheckConfig 仅在配置保存时补齐默认提示词，不给旧快照自动开启复核。
func normalizeAIRecheckConfig(cfg *model.AIRecheckConfig, strictness ...string) {
	scale := "standard"
	if len(strictness) > 0 {
		scale = strictness[0]
	}
	if cfg == nil {
		return
	}
	if strings.TrimSpace(cfg.SystemPrompt) == "" {
		cfg.SystemPrompt = recheckSystemPromptForStrictness(scale)
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
	var scale string
	_ = json.Unmarshal(values["audit_strictness"], &scale)
	normalizeAIRecheckConfig(&cfg, scale)
	values["recheck"], _ = json.Marshal(cfg)
	normalized, _ := json.Marshal(values)
	return normalized, nil
}

type recheckModelCaller interface {
	ChatWithFallback(*gin.Context, uuid.UUID, uuid.UUID, *model.AIModelConfig, *model.AIModelConfig, *ai.ChatRequest) (*ai.ChatResponse, error)
}

// AIRecheckInput 携带当次已绑定规则与复核流式回调，不重新读取业务材料。
type AIRecheckInput struct {
	Rules           string
	StreamChunkFunc func(string)
	StreamResetFunc func()
}

// buildAIRecheckRequest 默认只传结构化结果，其余初审内容仅在模板显式引用变量时传入。
func buildAIRecheckRequest(cfg *model.AIRecheckConfig, original *ai.ChatRequest, reasoning string, initial json.RawMessage, inputs ...AIRecheckInput) *ai.ChatRequest {
	var input AIRecheckInput
	if len(inputs) > 0 {
		input = inputs[0]
	}
	values := []string{"{{original_system_prompt}}", original.SystemPrompt, "{{audit_context}}", original.UserPrompt, "{{reasoning_result}}", reasoning, "{{initial_result}}", string(initial), "{{rules}}", input.Rules}
	replacer := strings.NewReplacer(values...)
	// 先替换模板中的日期变量，避免把业务内容里的同名文字当作模板再次展开。
	system := replacer.Replace(replaceSystemPromptVariables(cfg.SystemPrompt))
	user := replacer.Replace(replaceSystemPromptVariables(cfg.UserPrompt))
	// 结构化结果是复核必需输入，未引用时只补入这一项；其他材料不自动追加。
	if !strings.Contains(cfg.SystemPrompt, "{{initial_result}}") && !strings.Contains(cfg.UserPrompt, "{{initial_result}}") {
		user += "\n\n【初审结构化结果】\n" + string(initial)
	}
	system += "\n\n" + aiRecheckOutputInstructions(original.RequestType)
	return &ai.ChatRequest{SystemPrompt: system, UserPrompt: user, RequestType: original.RequestType, CallType: "recheck", Temperature: 0.1, MaxTokens: original.MaxTokens, ModelConfig: original.ModelConfig, ProcessID: original.ProcessID, ProcessTitle: original.ProcessTitle, BusinessLogID: original.BusinessLogID, StreamChunkFunc: input.StreamChunkFunc, StreamResetFunc: input.StreamResetFunc}
}

// runAIRecheck 通过统一入口执行一次可选复核，持久化回调负责租户隔离和取消保护。
func runAIRecheck(c *gin.Context, caller recheckModelCaller, tenantID, userID uuid.UUID, primary, fallback *model.AIModelConfig, cfg *model.AIRecheckConfig, original *ai.ChatRequest, reasoning string, initial json.RawMessage, scrub bool, persist func(*model.AIRecheckRecord) error, inputs ...AIRecheckInput) (json.RawMessage, *model.AIRecheckRecord, error) {
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
	resp, err := caller.ChatWithFallback(c, tenantID, userID, primary, fallback, buildAIRecheckRequest(cfg, original, reasoning, initial, inputs...))
	record.DurationMs = time.Since(started).Milliseconds()
	if err == nil {
		err = c.Request.Context().Err()
	}
	if err == nil && resp == nil {
		err = fmt.Errorf("AI复核返回为空")
	}
	if err == nil {
		var envelope struct {
			ReuseOriginal bool            `json:"reuse_original"`
			Reason        string          `json:"reason"`
			FinalResult   json.RawMessage `json:"final_result"`
		}
		err = validateAIRecheckEnvelope([]byte(resp.Content), original.RequestType)
		if err == nil {
			err = json.Unmarshal([]byte(resp.Content), &envelope)
		}
		if err == nil && !envelope.ReuseOriginal && strings.TrimSpace(envelope.Reason) == "" {
			err = fmt.Errorf("复核缺少说明")
		}
		if err == nil && !envelope.ReuseOriginal {
			err = validateAIRecheckResult(envelope.FinalResult, initial, original.RequestType)
		}
		if err == nil && !envelope.ReuseOriginal {
			if scrub {
				envelope.Reason = sanitize.SanitizeText(envelope.Reason)
				envelope.FinalResult = json.RawMessage(sanitize.SanitizeText(string(envelope.FinalResult)))
				if !json.Valid(envelope.FinalResult) {
					err = fmt.Errorf("脱敏后的复核结果格式无效")
				} else {
					err = validateAIRecheckResult(envelope.FinalResult, initial, original.RequestType)
				}
			}
		}
		if err == nil {
			record.Status = "completed"
			record.Reason = envelope.Reason
			record.FinalResult = envelope.FinalResult
			record.ReuseOriginal = envelope.ReuseOriginal || reviewResultsEqual(initial, envelope.FinalResult)
			record.Changed = !record.ReuseOriginal
			if record.ReuseOriginal {
				record.FinalResult = nil
			}
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
	if record.ReuseOriginal {
		return initial, record, nil
	}
	return record.FinalResult, record, nil
}

// reviewResultsEqual 忽略对象键序、条目顺序和纯空白差异，只有最终业务内容改变才标记调整。
func reviewResultsEqual(before, after json.RawMessage) bool {
	var a, b interface{}
	if json.Unmarshal(before, &a) != nil || json.Unmarshal(after, &b) != nil {
		return false
	}
	var normalize func(interface{}) interface{}
	normalize = func(v interface{}) interface{} {
		switch x := v.(type) {
		case string:
			return strings.Join(strings.Fields(x), " ")
		case map[string]interface{}:
			for key, item := range x {
				x[key] = normalize(item)
			}
			return x
		case []interface{}:
			for i, item := range x {
				x[i] = normalize(item)
			}
			sort.Slice(x, func(i, j int) bool {
				a, _ := json.Marshal(x[i])
				b, _ := json.Marshal(x[j])
				return string(a) < string(b)
			})
			return x
		default:
			return v
		}
	}
	return reflect.DeepEqual(normalize(a), normalize(b))
}

// validateAIRecheckResult 只校验明确的结构、枚举、数值和规则覆盖，不解释文字的业务含义。
func validateAIRecheckResult(raw, initial json.RawMessage, module string) error {
	value, err := decodeRecheckJSON(raw)
	if err != nil {
		return err
	}
	if err := validateRecheckSchema(value, recheckOutputSchema(module).OneOf[1].Properties["final_result"], "$.final_result"); err != nil {
		return err
	}
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
		if err := validateRecheckRowCoverage(before["rule_audit"], out["rule_audit"], "rule_id", "rule_name"); err != nil {
			return err
		}
		return validateRecheckRowCoverage(before["field_audit"], out["field_audit"], "field_key", "field_name")
	}
	_, err = ParseAuditResult(string(raw))
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
			idRaw, hasID := row[idKey]
			if (hasID && (bytes.Equal(bytes.TrimSpace(idRaw), []byte("null")) || json.Unmarshal(idRaw, &id) != nil)) || json.Unmarshal(row[nameKey], &name) != nil {
				return nil, fmt.Errorf("复核核验明细缺少标识或名称")
			}
			counts[fmt.Sprintf("%t\x00%s\x00%s", hasID, id, name)]++
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
