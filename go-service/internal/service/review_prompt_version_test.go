// 自定义标识和业务数据均为虚拟样例；厂商标准表结构按适配器约定保留。
package service

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	"gorm.io/datatypes"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/ai"
	"auraoa/go-service/internal/pkg/oa"
)

// TestReviewPromptsPreserveSelectedVersion 验证新旧快照均使用各自提示词，不在重审时自动追加新版评分要求。
func TestReviewPromptsPreserveSelectedVersion(t *testing.T) {
	const rules = "1. 主表与附件金额必须一致"
	versions := []struct {
		name, system string
	}{
		{"旧版本", "旧版业务说明：规则多时分值分散，可补正的笔误优先通过。"},
		{"新版本", "【问题定性与评分统一要求】关键金额冲突未排除时复核，按问题严重程度评分。"},
	}
	for _, version := range versions {
		for _, strictness := range []string{"strict", "standard", "loose"} {
			t.Run(version.name+"/"+strictness, func(t *testing.T) {
				auditConfig := model.AIConfigData{
					AuditStrictness: strictness, SystemReasoningPrompt: version.system, SystemExtractionPrompt: version.system,
					UserReasoningPrompt: "{{main_table}}\n{{rules}}", UserExtractionPrompt: "{{reasoning_result}}\n{{rules}}",
				}
				archiveConfig := model.ArchiveAIConfigData{
					AuditStrictness: strictness, SystemReasoningPrompt: version.system, SystemExtractionPrompt: version.system,
					UserReasoningPrompt: "{{main_table}}\n{{rules}}", UserExtractionPrompt: "{{reasoning_result}}\n{{rules}}",
				}
				auditJSON := mustMarshalReviewConfig(t, AuditExecutionConfigSnapshot{AIConfig: mustMarshalReviewConfig(t, auditConfig), MergedRules: rules})
				archiveJSON := mustMarshalReviewConfig(t, ArchiveExecutionConfigSnapshot{AIConfig: mustMarshalReviewConfig(t, archiveConfig), MergedRules: rules})
				auditSnapshot, err := decodeExecutionSnapshot[AuditExecutionConfigSnapshot](&model.ExecutionConfigVersion{ConfigSnapshot: auditJSON})
				if err != nil {
					t.Fatal(err)
				}
				archiveSnapshot, err := decodeExecutionSnapshot[ArchiveExecutionConfigSnapshot](&model.ExecutionConfigVersion{ConfigSnapshot: archiveJSON})
				if err != nil {
					t.Fatal(err)
				}
				var audit model.AIConfigData
				var archive model.ArchiveAIConfigData
				if err := json.Unmarshal(auditSnapshot.AIConfig, &audit); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(archiveSnapshot.AIConfig, &archive); err != nil {
					t.Fatal(err)
				}
				data := &oa.ProcessData{MainData: map[string]interface{}{"amount": "2400"}}
				requests := []struct {
					name    string
					request *ai.ChatRequest
				}{
					{"审核推理", BuildReasoningPrompt(&audit, "立项", data, auditSnapshot.MergedRules, "审批", nil, nil, "")},
					{"审核提取", BuildExtractionPrompt(&audit, "版本对应的推理结果", auditSnapshot.MergedRules)},
					{"归档推理", BuildArchiveReasoningPrompt(&archive, "立项", data, archiveSnapshot.MergedRules, "归档", nil, nil, "")},
					{"归档提取", BuildArchiveExtractionPrompt(&archive, "版本对应的推理结果", archiveSnapshot.MergedRules)},
				}
				for _, entry := range requests {
					t.Run(entry.name, func(t *testing.T) {
						// 保留本次改动前已有的适用前提说明，不允许额外注入评分政策。
						want := version.system + "\n\n" + ruleApplicabilityInstructions
						if entry.request.SystemPrompt != want {
							t.Fatal("请求必须保留所选快照的提示词，不得强制覆盖或追加新版评分政策")
						}
						if !strings.Contains(entry.request.UserPrompt, rules) {
							t.Fatal("所选版本的规则不能丢失")
						}
					})
				}
				if audit.SystemReasoningPrompt != auditConfig.SystemReasoningPrompt || archive.SystemReasoningPrompt != archiveConfig.SystemReasoningPrompt {
					t.Fatal("请求构建不能改写原配置")
				}
			})
		}
	}
}

func mustMarshalReviewConfig(t *testing.T, value interface{}) datatypes.JSON {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return datatypes.JSON(raw)
}

// TestDefaultReviewPromptsMatchReviewedText 验证默认模板、审核稿与实际请求一致，避免隐藏追加或复核缺省值漂移。
func TestDefaultReviewPromptsMatchReviewedText(t *testing.T) {
	sql, err := os.ReadFile("../../../db/migrations/000093_review_prompt_scale_requirements.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	review, err := os.ReadFile("../../../docs/reviews/2026-10-26-audit-archive-prompts-for-review.md")
	if err != nil {
		t.Fatal(err)
	}
	// Windows 检出可能使用 CRLF，模板内容比较统一换行后再验证。
	reviewText := strings.ReplaceAll(string(review), "\r\n", "\n")
	rows := regexp.MustCompile(`(?s)\('([a-z_]+)', \$prompt\$(.*?)\$prompt\$\)`).FindAllStringSubmatch(strings.ReplaceAll(string(sql), "\r\n", "\n"), -1)
	filtered := rows[:0]
	for _, row := range rows {
		if !strings.HasSuffix(row[1], "_recheck") {
			filtered = append(filtered, row)
		}
	}
	newer, readErr := os.ReadFile("../../../db/migrations/000094_recheck_consistency_and_reuse.up.sql")
	if readErr != nil {
		t.Fatal(readErr)
	}
	rows = append(filtered, regexp.MustCompile(`(?s)\('([a-z_]+)'[^$]*\$prompt\$(.*?)\$prompt\$`).FindAllStringSubmatch(strings.ReplaceAll(string(newer), "\r\n", "\n"), -1)...)
	attachmentPolicySQL, err := os.ReadFile("../../../db/migrations/000095_unparsed_attachment_prompt_policy.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	policy := regexp.MustCompile(`(?s)\$policy\$(.*?)\$policy\$`).FindStringSubmatch(strings.ReplaceAll(string(attachmentPolicySQL), "\r\n", "\n"))
	policyKeys := regexp.MustCompile(`'((?:audit|archive)_system_(?:reasoning|extraction)_(?:standard|loose))'`).FindAllStringSubmatch(string(attachmentPolicySQL), -1)
	if len(policy) != 2 || len(policyKeys) != 8 {
		t.Fatal("附件解析失败政策应仅更新审核和归档标准、宽松尺度的八条系统模板")
	}
	updated := 0
	for _, row := range rows {
		for _, key := range policyKeys {
			if row[1] == key[1] {
				row[2] = strings.Replace(row[2], "【规则适用前提】", policy[1]+"\n\n【规则适用前提】", 1)
				updated++
			}
		}
	}
	if updated != 8 {
		t.Fatal("附件解析失败政策必须匹配八条现有模板，严格尺度不变")
	}
	if len(rows) != 22 {
		t.Fatalf("预期十二条系统模板和十条复核默认模板，实际%d条", len(rows))
	}
	for _, row := range rows {
		key, prompt := row[1], row[2]
		t.Run(key, func(t *testing.T) {
			if !strings.Contains(reviewText, "模板键：`"+key+"`\n\n~~~~text\n"+prompt+"\n~~~~") {
				t.Fatal("审核稿与迁移中的完整模板不一致")
			}
			if strings.Contains(key, "_recheck") {
				want := defaultRecheckSystemPrompt
				if strings.HasSuffix(key, "_strict") {
					want = strictRecheckSystemPrompt
				}
				if strings.HasSuffix(key, "_loose") {
					want = looseRecheckSystemPrompt
				}
				if strings.Contains(key, "_user_") {
					want = defaultRecheckUserPrompt
				}
				if prompt != want {
					t.Fatal("复核保存时的默认提示词与模板不一致")
				}
				return
			}
			audit := &model.AIConfigData{SystemReasoningPrompt: prompt, SystemExtractionPrompt: prompt}
			archive := &model.ArchiveAIConfigData{SystemReasoningPrompt: prompt, SystemExtractionPrompt: prompt}
			var req *ai.ChatRequest
			switch {
			case strings.HasPrefix(key, "audit_") && strings.Contains(key, "_reasoning_"):
				req = BuildReasoningPrompt(audit, "演示流程", &oa.ProcessData{}, "演示规则", "", nil, nil, "")
			case strings.HasPrefix(key, "audit_"):
				req = BuildExtractionPrompt(audit, "演示分析", "演示规则")
			case strings.Contains(key, "_reasoning_"):
				req = BuildArchiveReasoningPrompt(archive, "演示流程", &oa.ProcessData{}, "演示规则", "", nil, nil, "")
			default:
				req = BuildArchiveExtractionPrompt(archive, "演示分析", "演示规则")
			}
			if req.SystemPrompt != prompt {
				t.Fatal("新版模板已包含前提要求，实际请求不应再追加旧版说明")
			}
			if strings.Contains(key, "_extraction_") {
				start := strings.Index(prompt, "\n{\n")
				if start < 0 {
					t.Fatal("结构化模板缺少JSON格式示例")
				}
				if strings.HasPrefix(key, "audit_") {
					_, err = ParseAuditResult(prompt[start+1:])
				} else {
					_, err = ParseArchiveReviewResult(prompt[start+1:])
				}
				if err != nil {
					t.Fatalf("模板中的JSON示例与实际解析器不兼容：%v", err)
				}
			}
		})
	}
}
