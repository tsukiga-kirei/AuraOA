package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

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
	for _, limits := range [][2]int{{0, 0}, {0, 64000}, {10, 0}, {10, 64000}, {0, -1}, {-1, 0}} {
		def.MaxRows, def.MaxChars = limits[0], limits[1]
		valid := limits[0] >= 0 && limits[1] >= 0 && !(limits[0] > 0 && limits[1] > 0)
		if result := queryCheck(def); result.Valid != valid {
			t.Fatalf("数据量限制%v检查不正确：%+v", limits, result.Issues)
		}
		copy := def
		if err := normalizeQueryAuthoring(&copy); (err == nil) != valid {
			t.Fatalf("草稿接受非法数据量限制%v：%v", limits, err)
		}
	}
	def.MaxRows, def.MaxChars = 20, 0
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

func TestQueryAssistFieldContextDoesNotRepeatNotes(t *testing.T) {
	def := model.AuditDataQueryDefinition{AIContext: &model.AuditDataQueryAIContext{
		OAModule: "workflow", Notes: "统计业务口径",
		FieldNotes: map[string]string{"FORMTABLE_MAIN_199.sqr": "申请人", "formtable_main_199.AMOUNT": "实付金额，单位元"},
	}}
	schemas := map[string][]oa.QueryColumn{"formtable_main_199": {
		{Key: "SQR", Label: "申请人", DataType: "INT"},
		{Key: "AMOUNT", Label: "金额", DataType: "NUMBER"},
	}}
	definition, schema := queryAssistFieldContext(def, schemas)
	raw, err := json.Marshal(map[string]interface{}{"definition": definition, "schema": schema})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "field_notes") || strings.Count(string(raw), "申请人") != 1 {
		t.Fatalf("AI 字段字典仍然重复: %s", raw)
	}
	if schema["formtable_main_199"][1].Description != "实付金额，单位元" || schema["formtable_main_199"][1].Label != "金额" {
		t.Fatalf("补充业务说明或原始中文名丢失: %+v", schema)
	}
	if len(def.AIContext.FieldNotes) != 2 || definition.AIContext.Notes != def.AIContext.Notes {
		t.Fatal("合并 AI 上下文不应修改原草稿或移除业务口径")
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
	yearParam := model.AuditDataQueryParameter{Name: "year", Label: "年份", Type: "number"}
	yearBinding := model.AuditDataQueryBinding{Source: "system", Value: "current_year"}
	resolvedYear, err := resolveQueryBinding(yearParam, yearBinding, nil, "")
	if err != nil || resolvedYear.(float64) < 2020 {
		t.Fatalf("current_year resolution failed: %v, %v", resolvedYear, err)
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
	raw := datatypes.JSON(`[{"query_id":"11111111-1111-4111-8111-111111111111","version_no":2,"enabled":true,"bindings":{"person":{"source":"field","field":"main:person"}},"trigger":{"field":"main:amount","value_type":"number","operator":"gte","value":"5000"}}]`)
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

// TestQueryPreviewFingerprintKeepsContentAndConnectionBound 验证兼容默认值不会解绑数据源校验。
func TestQueryPreviewFingerprintKeepsContentAndConnectionBound(t *testing.T) {
	legacy := model.AuditDataQueryDefinition{SQL: "SELECT 1 AS 数量", Dialect: "mysql"}
	current := legacy
	current.Mode = "sql"
	current.Tables = []string{}
	current.Parameters = []model.AuditDataQueryParameter{}
	current.AIContext = &model.AuditDataQueryAIContext{OAModule: "general", FieldNotes: map[string]string{}}
	connectionID := uuid.New()
	now := time.Date(2026, 10, 15, 1, 0, 0, 0, time.UTC)
	first, oldHash, err := queryContentFingerprints(legacy, connectionID, now)
	if err != nil {
		t.Fatal(err)
	}
	second, _, err := queryContentFingerprints(current, connectionID, now)
	if err != nil || first != second {
		t.Fatal("补默认字段不能改变内容预览指纹")
	}
	raw, _ := json.Marshal([]interface{}{legacy, connectionID, now})
	var old interface{}
	_ = json.Unmarshal(raw, &old)
	raw, _ = json.Marshal(old)
	sum := sha256.Sum256(raw)
	if oldHash != hex.EncodeToString(sum[:]) {
		t.Fatal("升级前验证指纹兼容失败")
	}
	changed, changedOld, err := queryContentFingerprints(current, connectionID, now.Add(time.Second))
	if err != nil || changed == first || changedOld == oldHash {
		t.Fatal("数据源修改后必须使旧预览失效")
	}
}
