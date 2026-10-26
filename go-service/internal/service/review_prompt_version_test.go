// 自定义标识和业务数据均为虚拟样例；厂商标准表结构按适配器约定保留。
package service

import (
	"encoding/json"
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
