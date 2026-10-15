package service

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/oa"
	"gorm.io/datatypes"
)

func TestEmptyPromptFieldValues(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value interface{}
		empty bool
	}{
		{"null", nil, true},
		{"空字符串", "", true},
		{"纯空白", " \t\n　", true},
		{"数据库空字节", []byte(" "), true},
		{"回显空值", map[string]interface{}{"value": nil, "display": ""}, true},
		{"回显非空值", map[string]interface{}{"value": "100", "display": ""}, false},
		{"零", 0, false},
		{"false", false, false},
		{"回显零", map[string]interface{}{"value": 0, "display": "未选择"}, false},
		{"字符串null", "null", false},
		{"普通文本", " 内容 ", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := isEmptyPromptField(tc.value); got != tc.empty {
				t.Fatalf("空值判断=%v，预期=%v", got, tc.empty)
			}
		})
	}
}

func TestEmptyFieldPolicyDoesNotMutateOADataOrDetailRows(t *testing.T) {
	data := &oa.ProcessData{
		MainData: map[string]interface{}{"REMARK": nil, "amount": 0},
		DetailTables: map[string][]map[string]interface{}{
			"FORM_DT1": {{"REMARK": ""}, {"REMARK": "明细内容"}},
			"form_dt2": {{"remark": nil}},
		},
	}
	before, _ := json.Marshal(data)
	refs := []string{"MAIN:remark", "form_dt1:remark"}
	filtered := filterEmptyProcessDataForPrompt(data, false, refs)
	if _, exists := filtered.MainData["REMARK"]; exists {
		t.Fatal("主表空字段不应保留")
	}
	if len(filtered.DetailTables["FORM_DT1"]) != 2 || len(filtered.DetailTables["FORM_DT1"][0]) != 0 || filtered.DetailTables["FORM_DT1"][1]["REMARK"] != "明细内容" {
		t.Fatalf("应逐行过滤而保留明细行位置: %+v", filtered.DetailTables)
	}
	if _, exists := filtered.DetailTables["form_dt2"][0]["remark"]; !exists {
		t.Fatal("不同明细表的同名字段不能互相影响")
	}
	all := filterEmptyProcessDataForPrompt(data, true, refs)
	if len(all.DetailTables["form_dt2"][0]) != 0 || all.MainData["amount"] != 0 {
		t.Fatal("统一开启应覆盖全部空值，保留零值")
	}
	after, _ := json.Marshal(data)
	if string(before) != string(after) {
		t.Fatal("生成提示词不得修改原始 OA 数据")
	}
	if filterEmptyProcessDataForPrompt(data, false, nil) != data {
		t.Fatal("旧配置应沿用原始数据")
	}
}

func TestAuditAndArchivePromptsApplyEmptyFieldPolicyWithinSelection(t *testing.T) {
	data := &oa.ProcessData{
		MainData: map[string]interface{}{"empty": nil, "zero": 0, "flag": false, "hidden": "不应传输"},
		DetailTables: map[string][]map[string]interface{}{
			"form_dt1": {{"detail_empty": "", "text": "有效明细"}},
		},
	}
	fields := SelectedFieldSet{"main": {"empty": true, "zero": true, "flag": true}, "form_dt1": {"detail_empty": true, "text": true}}
	for _, module := range []string{"audit", "archive"} {
		for _, tc := range []struct {
			name                 string
			all                  bool
			refs                 []string
			wantMain, wantDetail bool
		}{
			{"默认传空值", false, nil, true, true},
			{"仅主表字段开启", false, []string{"main:empty"}, false, true},
			{"仅明细字段开启", false, []string{"form_dt1:detail_empty"}, true, false},
			{"统一优先", true, []string{"main:empty"}, false, false},
		} {
			t.Run(module+"/"+tc.name, func(t *testing.T) {
				var prompt string
				if module == "audit" {
					prompt = BuildReasoningPrompt(&model.AIConfigData{SkipEmptyFields: tc.all, SkipEmptyFieldKeys: tc.refs, UserReasoningPrompt: "{{main_table}}\n{{detail_tables}}"}, "采购", data, "", "", fields, nil, "").UserPrompt
				} else {
					prompt = BuildArchiveReasoningPrompt(&model.ArchiveAIConfigData{SkipEmptyFields: tc.all, SkipEmptyFieldKeys: tc.refs, UserReasoningPrompt: "{{fields}}\n{{detail_tables}}"}, "采购", data, "", "", fields, nil, "").UserPrompt
				}
				if strings.Contains(prompt, `"empty"`) != tc.wantMain || strings.Contains(prompt, `"detail_empty"`) != tc.wantDetail {
					t.Fatalf("提示词未应用空值策略: %s", prompt)
				}
				for _, required := range []string{`"zero": 0`, `"flag": false`, "有效明细"} {
					if !strings.Contains(prompt, required) {
						t.Fatalf("非空值被误删: %s", prompt)
					}
				}
				if strings.Contains(prompt, "不应传输") {
					t.Fatal("空值开关不得扩大字段选择范围")
				}
			})
		}
	}
}

func TestEmptyFieldPolicyUsesExistingExecutionSnapshots(t *testing.T) {
	legacy := datatypes.JSON(`{"user_reasoning_prompt":"{{main_table}}"}`)
	policy := datatypes.JSON(`{"skip_empty_fields":true,"skip_empty_field_keys":["main:empty"],"user_reasoning_prompt":"{{main_table}}"}`)
	for _, module := range []string{"audit", "archive"} {
		t.Run(module, func(t *testing.T) {
			var snapshot interface{}
			if module == "audit" {
				snapshot = AuditExecutionConfigSnapshot{AIConfig: policy, BaseConfigVersionNo: 2}
			} else {
				snapshot = ArchiveExecutionConfigSnapshot{AIConfig: policy, BaseConfigVersionNo: 2}
			}
			raw, err := json.Marshal(snapshot)
			if err != nil {
				t.Fatal(err)
			}
			var decoded struct {
				AIConfig model.AIConfigData `json:"ai_config"`
				Version  int                `json:"base_config_version_no"`
			}
			if err := json.Unmarshal(raw, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.Version != 2 || !decoded.AIConfig.SkipEmptyFields || !reflect.DeepEqual(decoded.AIConfig.SkipEmptyFieldKeys, []string{"main:empty"}) {
				t.Fatalf("现有快照未保留策略及基础版本: %+v", decoded)
			}
			var old model.AIConfigData
			if err := json.Unmarshal(legacy, &old); err != nil || old.SkipEmptyFields || len(old.SkipEmptyFieldKeys) != 0 {
				t.Fatal("旧快照必须默认继续传输空值")
			}
			encodedOld, _ := json.Marshal(old)
			if strings.Contains(string(encodedOld), "skip_empty") {
				t.Fatal("旧配置序列化不得凭空增加空值开关，避免影响旧快照指纹")
			}
			// 缺省开关不进入旧内容指纹；开启策略仍属于正常的配置内容变更。
			oldRaw, _ := json.Marshal(map[string]interface{}{"ai_config": legacy})
			newRaw, _ := json.Marshal(map[string]interface{}{"ai_config": policy})
			if sourceFingerprintForTest(t, module, oldRaw) == sourceFingerprintForTest(t, module, newRaw) {
				t.Fatal("空值策略必须随配置内容保存与发布")
			}
		})
	}
}
