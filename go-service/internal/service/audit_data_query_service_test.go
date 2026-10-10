package service

import (
	"encoding/json"
	"strings"
	"testing"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/oa"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

func TestQueryDefinitionScope(t *testing.T) {
	def := model.AuditDataQueryDefinition{
		Name: "年度累计", Description: "按项目统计，已完成单据，人民币元", Dialect: "mysql", MaxRows: 20,
		SQL:    "SELECT SUM(amount) AS 累计金额 FROM expenses WHERE person=:person",
		Tables: []string{"expenses"}, Parameters: []model.AuditDataQueryParameter{{Name: "person", Label: "人员", Type: "string"}},
	}
	if result := queryCheck(def); !result.Valid {
		t.Fatalf("unexpected issues: %+v", result.Issues)
	}
	for _, limit := range []int{0, 101, 1000, -1} {
		def.MaxRows = limit
		if result := queryCheck(def); result.Valid != (limit >= 0) {
			t.Fatalf("行数限制%d的检查结果不正确：%+v", limit, result.Issues)
		}
	}
	def.MaxRows = 20
	def.Tables = []string{"budgets"}
	if queryCheck(def).Valid {
		t.Fatal("undeclared table accepted")
	}
	def.Tables = []string{"expenses"}
	def.Parameters[0].Label = "person"
	if queryCheck(def).Valid {
		t.Fatal("non-Chinese parameter accepted")
	}
}

func TestQueryBindingsTypedAndOffset(t *testing.T) {
	def := model.AuditDataQueryDefinition{Parameters: []model.AuditDataQueryParameter{
		{Name: "person", Label: "人员", Type: "string"}, {Name: "ids", Label: "明细项目", Type: "string_list"},
		{Name: "date", Label: "起始日期", Type: "date"}, {Name: "current", Label: "当前流程", Type: "string"},
	}}
	pd := &oa.ProcessData{MainData: map[string]interface{}{"EMPLOYEE": "42"}, DetailTables: map[string][]map[string]interface{}{"detail_1": {{"project": "A"}, {"project": "B"}}}}
	bindings := map[string]model.AuditDataQueryBinding{
		"person": {Source: "field", Field: "main:employee"}, "ids": {Source: "field", Field: "detail_1:project"},
		"date": {Source: "constant", Value: "2024-03-01", DaysOffset: -1}, "current": {Source: "system", Value: "process_id"},
	}
	got, err := resolveQueryBindings(def, bindings, pd, "request-7")
	if err != nil || got["person"] != "42" || got["date"] != "2024-02-29" || got["current"] != "request-7" || len(got["ids"].([]string)) != 2 {
		t.Fatalf("%#v %v", got, err)
	}
	delete(bindings, "person")
	if _, err = resolveQueryBindings(def, bindings, pd, "request-7"); err == nil {
		t.Fatal("missing field binding accepted")
	}
	number := model.AuditDataQueryParameter{Label: "金额", Type: "number"}
	for _, v := range []string{"NaN", "+Inf", "-Inf", "", "a"} {
		if _, err := normalizeQueryParameter(number, v, 0); err == nil {
			t.Fatalf("invalid number %s accepted", v)
		}
	}
	if _, err := normalizeQueryParameter(def.Parameters[2], "2024-01-01", 9999); err == nil {
		t.Fatal("unbounded offset accepted")
	}
	if _, err := normalizeQueryParameter(def.Parameters[0], "42", 1); err == nil {
		t.Fatal("offset on non-date accepted")
	}
}

func TestQueryEvidenceAndGuards(t *testing.T) {
	e := QueryEvidence{Name: "历史报销", RuleContent: "不得拆单", Status: "success", Result: &oa.AuditQueryResult{Columns: []oa.QueryColumn{{Key: "col_1", Label: "金额"}}, Rows: []map[string]interface{}{{"col_1": 100}}, Truncated: true}}
	text := formatQueryEvidence(e)
	if !strings.Contains(text, `"金额":100`) || !strings.Contains(text, "不能据此计算全量累计") {
		t.Fatalf("missing evidence semantics: %s", text)
	}
	audit := &model.AuditResultJSON{Recommendation: "approve", RuleResults: []model.RuleResultJSON{{RuleContent: "不得拆单", Passed: true}}}
	GuardQueryEvidence(audit, []QueryEvidence{e})
	if audit.Recommendation != "review" || audit.RuleResults[0].Passed {
		t.Fatal("truncated evidence approved")
	}
	archive := &model.ArchiveResultJSON{OverallCompliance: "compliant"}
	GuardArchiveQueryEvidence(archive, []QueryEvidence{e})
	if archive.OverallCompliance == "compliant" {
		t.Fatal("archive approved incomplete evidence")
	}
	e.Status = "failed"
	e.Error = "数据库超时"
	e.Result = nil
	if text = formatQueryEvidence(e); !strings.Contains(text, "不得将查询失败解释为没有记录") {
		t.Fatalf("failure evidence: %s", text)
	}
	e.Status = "success"
	e.Result = &oa.AuditQueryResult{Rows: []map[string]interface{}{}}
	if text = formatQueryEvidence(e); !strings.Contains(text, "查询成功，无匹配记录") {
		t.Fatal("empty success is not explicit")
	}
}

func TestQueryReferenceSurvivesEffectiveConfigSnapshot(t *testing.T) {
	raw := datatypes.JSON(`[{"query_id":"11111111-1111-4111-8111-111111111111","version_no":2,"enabled":true,"bindings":{"person":{"source":"field","field":"main:person"}}}]`)
	rules := []model.AuditRule{{ID: uuid.New(), RuleScope: "mandatory", DataQueries: raw}}
	effective := resolveEffectiveAuditRules(rules, nil)
	encoded, err := json.Marshal(AuditExecutionConfigSnapshot{EffectiveRules: effective})
	if err != nil {
		t.Fatal(err)
	}
	var restored AuditExecutionConfigSnapshot
	if err = json.Unmarshal(encoded, &restored); err != nil || string(restored.EffectiveRules[0].DataQueries) != string(raw) {
		t.Fatalf("query version lost in execution snapshot: %s %v", encoded, err)
	}
}

// 配置元数据经历 PostgreSQL JSONB 重排后，成功预览的指纹仍然匹配。
func TestQueryBuilderFingerprintCanonicalJSON(t *testing.T) {
	first := model.AuditDataQueryDefinition{SQL: "SELECT 1 AS 笔数", Builder: json.RawMessage(`{"table":"formtable_main_1","columns":[{"field":"amount","alias":"金额"}]}`)}
	second := first
	second.Builder = json.RawMessage(`{"columns": [{"alias": "金额", "field": "amount"}], "table": "formtable_main_1"}`)
	a, err := canonicalQueryJSON(first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := canonicalQueryJSON(second)
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatalf("JSONB 重排改变了查询指纹: %s != %s", a, b)
	}
}
