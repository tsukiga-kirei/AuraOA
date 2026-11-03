package oa

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestE9DB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite db: %v", err)
	}

	// 初始化模拟泛微 E9 字典与组织架构表
	queries := []string{
		`CREATE TABLE workflow_base (id INTEGER PRIMARY KEY, formid INTEGER, workflowname TEXT, isvalid TEXT);`,
		`CREATE TABLE workflow_billfield (id INTEGER PRIMARY KEY, billid INTEGER, fieldname TEXT, fieldhtmltype TEXT, type INTEGER, fielddbtype TEXT, detailtable TEXT);`,
		`CREATE TABLE workflow_selectitem (id INTEGER PRIMARY KEY, fieldid INTEGER, selectvalue TEXT, selectname TEXT);`,
		`CREATE TABLE hrmresource (id INTEGER PRIMARY KEY, lastname TEXT, loginid TEXT, workcode TEXT, status INTEGER);`,
		`CREATE TABLE hrmdepartment (id INTEGER PRIMARY KEY, departmentname TEXT, departmentcode TEXT);`,
	}

	for _, q := range queries {
		if err := db.Exec(q).Error; err != nil {
			t.Fatalf("failed to create table: %v", err)
		}
	}

	// 插入模拟数据
	db.Exec(`INSERT INTO workflow_base (id, formid, workflowname, isvalid) VALUES (101, 201, '办公用品领用申请', '1');`)

	db.Exec(`INSERT INTO workflow_billfield (id, billid, fieldname, fieldhtmltype, type, fielddbtype, detailtable) VALUES 
		(1, 201, 'sqr', '3', 1, 'int', ''),
		(2, 201, 'dept', '3', 4, 'int', ''),
		(3, 201, 'wplx', '5', 0, 'int', ''),
		(4, 201, 'approvers', '3', 17, 'text', '');`)

	db.Exec(`INSERT INTO workflow_selectitem (id, fieldid, selectvalue, selectname) VALUES 
		(1, 3, '0', '办公文具'),
		(2, 3, '1', '电子耗材');`)

	db.Exec(`INSERT INTO hrmresource (id, lastname, loginid, workcode, status) VALUES 
		(1001, '张三', 'zhangsan', 'A001', 1),
		(1002, '李四', 'lisi', 'A002', 1);`)

	db.Exec(`INSERT INTO hrmdepartment (id, departmentname, departmentcode) VALUES 
		(301, '技术研发部', 'DEV01');`)

	return db
}

func TestNormalizeMainDataForE9(t *testing.T) {
	db := setupTestE9DB(t)

	adapter := &Ecology9Adapter{
		db:     db,
		driver: "mysql", // sqlite 行为与 mysql 相同（不强制全大写标识符）
	}

	ctx := context.Background()

	// 1. 测试文本反查 ID
	input := map[string]interface{}{
		"sqr":       "张三",
		"dept":      "技术研发部",
		"wplx":      "办公文具",
		"approvers": "张三,李四",
	}

	normalized := adapter.normalizeMainDataForE9(ctx, "101", input)

	if got := normalized["sqr"]; got != "1001" {
		t.Errorf("sqr: expected '1001', got '%v'", got)
	}
	if got := normalized["dept"]; got != "301" {
		t.Errorf("dept: expected '301', got '%v'", got)
	}
	if got := normalized["wplx"]; got != "0" {
		t.Errorf("wplx: expected '0', got '%v'", got)
	}
	if got := normalized["approvers"]; got != "1001,1002" {
		t.Errorf("approvers: expected '1001,1002', got '%v'", got)
	}

	// 2. 测试直接传合法 ID 与编码（应保持原值不被覆盖）
	inputAlreadyID := map[string]interface{}{
		"sqr":       "1001",
		"dept":      "301",
		"wplx":      "1",
		"approvers": "1001,1002",
	}

	normalized2 := adapter.normalizeMainDataForE9(ctx, "101", inputAlreadyID)

	if got := normalized2["sqr"]; got != "1001" {
		t.Errorf("sqr already ID: expected '1001', got '%v'", got)
	}
	if got := normalized2["dept"]; got != "301" {
		t.Errorf("dept already ID: expected '301', got '%v'", got)
	}
	if got := normalized2["wplx"]; got != "1" {
		t.Errorf("wplx already value: expected '1', got '%v'", got)
	}
	if got := normalized2["approvers"]; got != "1001,1002" {
		t.Errorf("approvers already ID: expected '1001,1002', got '%v'", got)
	}

	// 3. 测试模糊包含与近义词推断匹配（针对普通用户不知道确切下拉框名称的场景）
	inputFuzzyAndSynonym := map[string]interface{}{
		"wplx": "文具", // 子串包含匹配 "办公文具" -> "0"
	}
	normalizedFuzzy := adapter.normalizeMainDataForE9(ctx, "101", inputFuzzyAndSynonym)
	if got := normalizedFuzzy["wplx"]; got != "0" {
		t.Errorf("wplx fuzzy substring: expected '0', got '%v'", got)
	}

	inputSynonym := map[string]interface{}{
		"wplx": "中性笔", // 语义词根推断 "笔" -> "办公文具" -> "0"
	}
	normalizedSyn := adapter.normalizeMainDataForE9(ctx, "101", inputSynonym)
	if got := normalizedSyn["wplx"]; got != "0" {
		t.Errorf("wplx synonym: expected '0', got '%v'", got)
	}

	// 4. 测试 DB 为空时的优雅降级
	nilDBAdapter := &Ecology9Adapter{}
	fallback := nilDBAdapter.normalizeMainDataForE9(ctx, "101", input)
	if got := fallback["sqr"]; got != "张三" {
		t.Errorf("fallback: expected '张三', got '%v'", got)
	}
}
