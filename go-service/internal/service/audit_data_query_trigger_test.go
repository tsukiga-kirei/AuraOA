package service

import (
	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
	"auraoa/go-service/internal/pkg/oa"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestQueryTriggerFieldTypes(t *testing.T) {
	for _, tc := range []struct {
		name, kind, operator, expected string
		actual                         interface{}
		want, invalid                  bool
	}{
		{"数字字符串", "number", "gte", "5000", "5000.00", true, false},
		{"数字原值", "number", "lt", "5000", 4999.99, true, false},
		{"千分位金额", "number", "eq", "12345.60", []byte("12,345.60"), true, false},
		{"不丢失大整数精度", "number", "gt", "9007199254740992", "9007199254740993", true, false},
		{"不丢失小数精度", "number", "lt", "0.100000000000000002", json.Number("0.100000000000000001"), true, false},
		{"非法千分位", "number", "eq", "123", "1,23", false, true},
		{"无效数字", "number", "eq", "1", "NaN", false, true},
		{"指数过大", "number", "eq", "1e9999999999", "1", false, true},
		{"日期自然日", "date", "eq", "2026-10-15", "2026-10-15 23:59:59", true, false},
		{"斜线日期", "date", "gte", "2026-10-15", "2026/10/16", true, false},
		{"原生日期", "date", "eq", "2026-10-15", time.Date(2026, 10, 15, 12, 0, 0, 0, apptime.Location()), true, false},
		{"日期无效", "date", "eq", "2026-10-15", "2026-02-30", false, true},
		{"复选框数字真", "boolean", "eq", "true", 1, true, false},
		{"复选框字符串假", "boolean", "eq", "false", "0", true, false},
		{"布尔原值", "boolean", "ne", "true", false, true, false},
		{"不将未知布尔视为否", "boolean", "eq", "false", "未知", false, true},
		{"中文文字", "text", "contains", "招待", "客户招待费", true, false},
		{"文本逗号保留", "text", "eq", "甲,乙", "甲,乙", true, false},
		{"单选编码", "choice", "contains", "1", 1, true, false},
		{"多选编码", "choice", "contains", "1", "11,1,21", true, false},
		{"编码完整匹配", "choice", "contains", "1", "11,21", false, false},
		{"数组多选", "choice", "contains", "1", []interface{}{11, "1"}, true, false},
		{"否定多选", "choice", "not_contains", "1", []string{"11", "1"}, false, false},
		{"空附件", "file", "is_empty", "", []interface{}{}, true, false},
		{"附件数组", "file", "not_empty", "", []string{"123", "456"}, true, false},
		{"空字符串", "text", "is_empty", "", "  ", true, false},
		{"空值不命中不等于", "number", "ne", "1", nil, false, false},
		{"零不是空", "number", "not_empty", "", 0, true, false},
		{"否不是空", "boolean", "not_empty", "", false, true, false},
		{"类型与比较方式不匹配", "file", "gt", "1", "2", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			trigger := &model.AuditDataQueryTrigger{Field: "MAIN:amount", ValueType: tc.kind, Operator: tc.operator, Value: tc.expected}
			matched, err := evaluateQueryTrigger(trigger, &oa.ProcessData{MainData: map[string]interface{}{"AMOUNT": tc.actual}})
			if (err != nil) != tc.invalid || matched != tc.want {
				t.Fatalf("matched=%t err=%v", matched, err)
			}
		})
	}
}

func TestQueryTriggerDetailsAndMissingFields(t *testing.T) {
	trigger := &model.AuditDataQueryTrigger{Field: "dt1:amount", ValueType: "number", Operator: "gte", Value: "5000"}
	pd := &oa.ProcessData{DetailTables: map[string][]map[string]interface{}{"DT1": {{"AMOUNT": 1}, {"AMOUNT": 6000}}}}
	if matched, err := evaluateQueryTrigger(trigger, pd); !matched || err != nil {
		t.Fatalf("任一明细行未命中：%t %v", matched, err)
	}
	trigger.Operator, trigger.Value = "ne", "6000"
	if matched, err := evaluateQueryTrigger(trigger, pd); matched || err != nil {
		t.Fatalf("否定条件忽略了匹配行：%t %v", matched, err)
	}
	pd.DetailTables["DT1"] = nil
	trigger.Operator = "is_empty"
	if matched, err := evaluateQueryTrigger(trigger, pd); !matched || err != nil {
		t.Fatal("无明细行应为空")
	}
	delete(pd.DetailTables, "DT1")
	if _, err := evaluateQueryTrigger(trigger, pd); err == nil {
		t.Fatal("缺失明细表被视为空表")
	}
	trigger.Field = "main:missing"
	if _, err := evaluateQueryTrigger(trigger, pd); err == nil {
		t.Fatal("缺失主表字段被视为空值")
	}
	trigger.Field = "main:amount;DROP"
	if err := validateQueryTrigger(trigger); err == nil {
		t.Fatal("非法字段引用被接受")
	}
}

func TestQueryTriggerDatesUseConfiguredZone(t *testing.T) {
	previous := apptime.Name()
	defer apptime.Configure(previous)
	if err := apptime.Configure("America/New_York"); err != nil {
		t.Fatal(err)
	}
	trigger := &model.AuditDataQueryTrigger{Field: "main:date", ValueType: "date", Operator: "eq", Value: "2026-10-14"}
	matched, err := evaluateQueryTrigger(trigger, &oa.ProcessData{MainData: map[string]interface{}{"date": "2026-10-15T01:00:00Z"}})
	if !matched || err != nil {
		t.Fatalf("未按应用时区计算自然日：%t %v", matched, err)
	}
}

func TestQueryTriggerBeforeBindingsAndEvidenceGuards(t *testing.T) {
	def := model.AuditDataQueryDefinition{Parameters: []model.AuditDataQueryParameter{{Name: "company", Label: "公司", Type: "string"}}}
	trigger := &model.AuditDataQueryTrigger{Field: "main:amount", ValueType: "number", Operator: "gte", Value: "5000"}
	pd := &oa.ProcessData{MainData: map[string]interface{}{"amount": 1}}
	values, matched, err := prepareTriggeredQueryInputs(def, nil, trigger, pd, "123")
	if err != nil || matched == nil || *matched || len(values) != 0 {
		t.Fatal("未命中条件不应继续解析必填查询参数")
	}
	evidence := QueryEvidence{Status: "skipped", Trigger: trigger, TriggerMatched: matched}
	text := formatQueryEvidence(evidence)
	if !strings.Contains(text, "不代表数据库没有记录") || strings.Contains(text, "数据不可用") {
		t.Fatalf("跳过语义错误：%s", text)
	}
	audit := &model.AuditResultJSON{Recommendation: "approve"}
	archive := &model.ArchiveResultJSON{OverallCompliance: "compliant"}
	GuardQueryEvidence(audit, []QueryEvidence{evidence})
	GuardArchiveQueryEvidence(archive, []QueryEvidence{evidence})
	if audit.Recommendation != "approve" || archive.OverallCompliance != "compliant" {
		t.Fatal("主动跳过不应强制降级结论")
	}
	pd.MainData["amount"] = 6000
	_, matched, err = prepareTriggeredQueryInputs(def, nil, trigger, pd, "123")
	if err == nil || matched == nil || !*matched {
		t.Fatal("命中后必须验证查询参数")
	}
	_, matched, err = prepareTriggeredQueryInputs(def, nil, nil, pd, "123")
	if err == nil || matched != nil {
		t.Fatal("旧引用应继续正常绑定参数")
	}
	evidence.Status, evidence.TriggerMatched = "failed", nil
	GuardQueryEvidence(audit, []QueryEvidence{evidence})
	GuardArchiveQueryEvidence(archive, []QueryEvidence{evidence})
	if audit.Recommendation != "review" || archive.OverallCompliance == "compliant" {
		t.Fatal("条件判断错误仍须人工复核")
	}
}
