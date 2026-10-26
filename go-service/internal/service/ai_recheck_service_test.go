package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/ai"
	"auraoa/go-service/internal/pkg/oa"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type fakeRecheckCaller struct {
	calls    int
	req      *ai.ChatRequest
	response string
	err      error
}

func (f *fakeRecheckCaller) ChatWithFallback(_ *gin.Context, _ uuid.UUID, _ uuid.UUID, _ *model.AIModelConfig, _ *model.AIModelConfig, req *ai.ChatRequest) (*ai.ChatResponse, error) {
	f.calls++
	f.req = req
	return &ai.ChatResponse{Content: f.response}, f.err
}

const initialAuditRecheck = `{"recommendation":"approve","overall_score":95,"confidence":95,"rule_results":[{"rule_content":"检查金额","passed":true,"reason":"笔误"}],"risk_points":["金额冲突"],"suggestions":["核实"]}`
const finalAuditRecheck = `{"recommendation":"review","overall_score":75,"confidence":90,"rule_results":[{"rule_content":"检查金额","passed":false,"reason":"金额单位未核实"}],"risk_points":["金额冲突未排除"],"suggestions":["核实金额单位"]}`

func TestAIRecheckCallAndTrace(t *testing.T) {
	for _, module := range []string{"audit", "archive"} {
		t.Run(module, func(t *testing.T) {
			initial, final := initialAuditRecheck, finalAuditRecheck
			if module == "archive" {
				initial = `{"overall_compliance":"compliant","overall_score":95,"confidence":95,"flow_audit":{"is_complete":true,"missing_nodes":[],"node_results":[]},"field_audit":[],"rule_audit":[{"rule_name":"检查金额","passed":true,"reasoning":"笔误"}],"risk_points":["金额冲突"],"suggestions":["核实"],"ai_summary":"初审"}`
				final = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(initial, "compliant", "partially_compliant"), "95", "75"), "笔误", "金额单位未核实")
			}
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/", nil)
			id := uuid.New()
			original := &ai.ChatRequest{SystemPrompt: "原版本审核要求", UserPrompt: "已选择字段、附件和规则证据", RequestType: module, ProcessID: "process-1", BusinessLogID: &id, MaxTokens: 2000}
			cfg := &model.AIRecheckConfig{Enabled: true, SystemPrompt: "复核原结果", UserPrompt: "自定义：{{initial_result}}"}
			caller := &fakeRecheckCaller{response: `{"reason":"金额冲突尚未核实，调整为人工复核","final_result":` + final + `}`}
			var records []model.AIRecheckRecord
			raw, record, err := runAIRecheck(c, caller, uuid.New(), uuid.New(), nil, nil, cfg, original, "原推理正文", json.RawMessage(initial), false, func(record *model.AIRecheckRecord) error { records = append(records, *record); return nil })
			if err != nil {
				t.Fatal(err)
			}
			if caller.calls != 1 || caller.req.CallType != "recheck" || caller.req.RequestType != module || caller.req.SkipQuotaCheck || caller.req.BusinessLogID != &id {
				t.Fatal("必须经统一调用入口、正确归集业务并独立检查配额")
			}
			for _, value := range []string{original.SystemPrompt, original.UserPrompt, "原推理正文", initial} {
				if !strings.Contains(caller.req.UserPrompt, value) {
					t.Fatal("自定义模板缺变量时不能丢失当次证据")
				}
			}
			if len(records) != 2 || records[0].Status != "running" || records[1].Status != "completed" || !record.Changed || string(record.OriginalResult) != initial || len(raw) == 0 {
				t.Fatal("应先保存初审，再保留最终结果与变更说明")
			}
		})
	}
}

func TestAIRecheckFailuresKeepInitial(t *testing.T) {
	for name, caller := range map[string]*fakeRecheckCaller{
		"调用失败":   {err: errors.New("模型不可用")},
		"格式错误":   {response: "不是JSON"},
		"缺少完整结果": {response: `{"reason":"维持","final_result":{}}`},
		"分数结论矛盾": {response: `{"reason":"调整","final_result":` + strings.Replace(finalAuditRecheck, `"review"`, `"approve"`, 1) + `}`},
		"丢失规则":   {response: `{"reason":"调整","final_result":` + strings.Replace(finalAuditRecheck, `"检查金额"`, `"其他规则"`, 1) + `}`},
	} {
		t.Run(name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/", nil)
			var saved *model.AIRecheckRecord
			out, record, err := runAIRecheck(c, caller, uuid.New(), uuid.New(), nil, nil, &model.AIRecheckConfig{Enabled: true}, &ai.ChatRequest{RequestType: "audit"}, "分析", json.RawMessage(initialAuditRecheck), false, func(r *model.AIRecheckRecord) error { saved = r; return nil })
			if err == nil || out != nil || saved.Status != "failed" || len(record.FinalResult) != 0 || string(record.OriginalResult) != initialAuditRecheck {
				t.Fatal("复核失败不能发布最终结论或丢失初审")
			}
		})
	}
}

func TestAIRecheckCancelledBeforeCall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", nil).WithContext(ctx)
	caller := &fakeRecheckCaller{}
	_, _, err := runAIRecheck(c, caller, uuid.New(), uuid.New(), nil, nil, &model.AIRecheckConfig{Enabled: true}, &ai.ChatRequest{}, "", json.RawMessage(initialAuditRecheck), false, func(*model.AIRecheckRecord) error { t.Fatal("取消后不应进入复核"); return nil })
	if !errors.Is(err, context.Canceled) || caller.calls != 0 {
		t.Fatal("取消后不得继续调用模型")
	}
}

func TestAIRecheckDisabledAndPersistenceFailure(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", nil)
	caller := &fakeRecheckCaller{}
	for _, cfg := range []*model.AIRecheckConfig{nil, {Enabled: false}} {
		out, record, err := runAIRecheck(c, caller, uuid.New(), uuid.New(), nil, nil, cfg, nil, "", json.RawMessage(initialAuditRecheck), false, func(*model.AIRecheckRecord) error { t.Fatal("关闭时不能保存复核记录"); return nil })
		if err != nil || record != nil || string(out) != initialAuditRecheck || caller.calls != 0 {
			t.Fatal("旧配置及关闭配置必须保持两阶段")
		}
	}
	_, _, err := runAIRecheck(c, caller, uuid.New(), uuid.New(), nil, nil, &model.AIRecheckConfig{Enabled: true}, nil, "", json.RawMessage(initialAuditRecheck), false, func(*model.AIRecheckRecord) error { return errors.New("无法保存初审") })
	if err == nil || caller.calls != 0 {
		t.Fatal("初审留痕失败时不能继续调用模型")
	}
}

// TestAIRecheckQueryEvidenceDoesNotForceDowngrade 验证查询异常仅作为资料传递，复核开关不增加额外调用或强制改判。
func TestAIRecheckQueryEvidenceDoesNotForceDowngrade(t *testing.T) {
	for _, module := range []string{"audit", "archive"} {
		for _, mode := range []struct {
			name    string
			enabled bool
		}{{"disabled", false}, {"enabled", true}} {
			for _, status := range []string{"failed", "truncated"} {
				t.Run(module+"/"+mode.name+"/"+status, func(t *testing.T) {
					initial := initialAuditRecheck
					if module == "archive" {
						initial = `{"overall_compliance":"compliant","overall_score":95,"confidence":95,"flow_audit":{"is_complete":true,"missing_nodes":[],"node_results":[]},"field_audit":[],"rule_audit":[{"rule_name":"检查金额","passed":true,"reasoning":"已核实"}],"risk_points":[],"suggestions":[],"ai_summary":"初审"}`
					}
					evidence := QueryEvidence{Name: "金额依据", RuleContent: "检查金额", Status: "failed", Error: "数据库超时"}
					if status == "truncated" {
						evidence.Status, evidence.Error = "success", ""
						evidence.Result = &oa.AuditQueryResult{Truncated: true}
					}
					contextText := formatQueryEvidence(evidence)
					c, _ := gin.CreateTestContext(httptest.NewRecorder())
					c.Request = httptest.NewRequest("POST", "/", nil)
					caller := &fakeRecheckCaller{response: `{"reason":"维持原结论","final_result":` + initial + `}`}
					saves := 0
					out, record, err := runAIRecheck(c, caller, uuid.New(), uuid.New(), nil, nil, &model.AIRecheckConfig{Enabled: mode.enabled}, &ai.ChatRequest{RequestType: module, UserPrompt: contextText}, "原分析", json.RawMessage(initial), false, func(*model.AIRecheckRecord) error { saves++; return nil })
					if err != nil || string(out) != initial {
						t.Fatalf("查询异常不应自行修改 AI 结果：%s, %v", out, err)
					}
					if mode.enabled {
						if caller.calls != 1 || saves != 2 || record == nil || record.Changed || !strings.Contains(caller.req.UserPrompt, contextText) {
							t.Fatal("开启时只执行已有复核并完整保留查询资料与原结果")
						}
					} else if caller.calls != 0 || saves != 0 || record != nil {
						t.Fatal("关闭时不能新增 AI 调用或复核记录")
					}
					if module == "archive" {
						parsed, err := ParseArchiveReviewResult(string(out))
						if err != nil || parsed.OverallScore != 95 || parsed.OverallCompliance != "compliant" || !parsed.RuleAudit[0].Passed {
							t.Fatalf("归档结果不应被查询状态改判：%+v, %v", parsed, err)
						}
					} else {
						parsed, err := ParseAuditResult(string(out))
						if err != nil || parsed.OverallScore != 95 || parsed.Recommendation != "approve" || !parsed.RuleResults[0].Passed {
							t.Fatalf("审核结果不应被查询状态改判：%+v, %v", parsed, err)
						}
					}
				})
			}
		}
	}
}

func TestAIRecheckConfigVersionIsolation(t *testing.T) {
	legacy := json.RawMessage(`{"audit_strictness":"standard","custom":"保留"}`)
	normalized, err := normalizeSavedRecheckConfig(legacy)
	if err != nil || string(normalized) != string(legacy) {
		t.Fatal("旧配置缺省不得新增复核设置")
	}
	current := json.RawMessage(`{"audit_strictness":"standard","recheck":{"enabled":true,"system_prompt":"自定义复核","user_prompt":"{{initial_result}}"}}`)
	var base model.AIConfigData
	_ = json.Unmarshal(current, &base)
	effective, err := auditAIConfigFromTemplates("strict", nil, base)
	if err != nil {
		t.Fatal(err)
	}
	var next model.AIConfigData
	_ = json.Unmarshal(effective, &next)
	if next.Recheck == nil || next.Recheck.SystemPrompt != "自定义复核" {
		t.Fatal("尺度切换不能丢失独立复核配置")
	}
	var archive model.ArchiveAIConfigData
	_ = json.Unmarshal(current, &archive)
	strict := "strict"
	templates := []model.SystemPromptTemplate{{PromptKey: "archive_system_reasoning_strict", Strictness: &strict, PromptType: "system", Phase: "reasoning"}}
	effective, err = archiveAIConfigFromTemplates("strict", templates, archive)
	if err != nil {
		t.Fatal(err)
	}
	var nextArchive model.ArchiveAIConfigData
	_ = json.Unmarshal(effective, &nextArchive)
	if nextArchive.Recheck == nil || nextArchive.Recheck.UserPrompt != "{{initial_result}}" {
		t.Fatal("归档尺度切换不能丢失独立复核配置")
	}
	if normalizedAIConfigContent(datatypes.JSON(legacy))["recheck"] != nil {
		t.Fatal("默认结构不能使旧配置指纹变化")
	}
	if len(auditProgressSteps("completed")) != 5 || len(auditProgressSteps("completed", true)) != 6 || len(archiveProgressSteps("rechecking")) != 5 {
		t.Fatal("进度必须按配置区分两阶段与三阶段")
	}
}
