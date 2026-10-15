package oa

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestAuditQuerySQLBoundaries(t *testing.T) {
	cases := []struct {
		name, sql string
		valid     bool
	}{
		{"aggregate", "SELECT COALESCE(SUM(amount),0) AS `累计金额`, COUNT(*) AS `笔数` FROM expenses WHERE employee_id=:employee", true},
		{"CTE", "WITH history AS (SELECT amount FROM expenses WHERE id<>:current) SELECT SUM(amount) AS \"历史金额\" FROM history", true},
		{"subquery", "SELECT (SELECT COUNT(*) FROM expenses) AS 笔数", true},
		{"literal", "SELECT 'AS :unused; DROP TABLE' AS 中文", true},
		{"union", "SELECT a AS 金额 FROM x UNION ALL SELECT b AS 金额 FROM y", true},
		{"missing alias", "SELECT amount FROM expenses", false},
		{"English alias", "SELECT amount AS amount FROM expenses", false},
		{"duplicate alias", "SELECT a AS 金额,b AS 金额 FROM x", false},
		{"wildcard", "SELECT x.* AS 数据 FROM x", false},
		{"comment", "SELECT 1 AS 金额 /* unsafe */", false},
		{"write CTE", "WITH x AS (DELETE FROM expenses) SELECT 1 AS 金额", false},
		{"statement stacking", "SELECT 1 AS 金额; DELETE FROM x", false},
		{"lock", "SELECT id AS 编号 FROM expenses FOR UPDATE", false},
		{"outfile", "SELECT id AS 编号 INTO OUTFILE '/tmp/x' FROM expenses", false},
		{"sleep", "SELECT SLEEP(10) AS 耗时", false},
		{"external function", "SELECT evil_fn() AS 数据 FROM dual", false},
		{"qualified function", "SELECT pkg.SUM(1) AS 数据 FROM dual", false},
		{"quoted function", "SELECT `evil_fn`() AS 数据", false},
		{"escaped quote", "SELECT 'x\\' AS 中文", false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r := CheckQuerySQL(tt.sql)
			if r.Valid != tt.valid {
				t.Fatalf("valid=%v issues=%+v", r.Valid, r.Issues)
			}
		})
	}
}
func TestReferencedQueryTables(t *testing.T) {
	cases := []struct {
		sql  string
		want []string
		fail bool
	}{
		{"SELECT x.id AS 编号 FROM x JOIN y ON x.id=y.id WHERE x.a IN (SELECT a FROM z)", []string{"x", "y", "z"}, false},
		{"WITH h AS (SELECT a FROM expenses), j AS (SELECT b FROM budgets) SELECT SUM(a) AS 金额 FROM h JOIN j ON 1=1", []string{"budgets", "expenses"}, false},
		{"SELECT a.id AS 编号 FROM x a, y b WHERE a.id=b.id", []string{"x", "y"}, false},
		{"SELECT a AS 字段 FROM other_db.x", nil, true},
		{"SELECT a AS 字段 FROM table_fn()", nil, true},
		{"SELECT 1 AS 数值 FROM dual", []string{}, false},
	}
	for _, tt := range cases {
		got, err := ReferencedQueryTables(tt.sql)
		if (err != nil) != tt.fail || !tt.fail && !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("%s: got %v err %v", tt.sql, got, err)
		}
	}
}
func TestAuditQueryParameterBinding(t *testing.T) {
	source := "SELECT ':id' AS 说明 FROM expenses WHERE id=:id AND person IN (:people) OR id=:id"
	injection := "1' OR 1=1 --"
	values := map[string]interface{}{"id": injection, "people": []string{"a", "b"}}
	got, args, err := bindAuditQuery(source, "mysql", values)
	if err != nil || got != "SELECT ':id' AS 说明 FROM expenses WHERE id=? AND person IN (?,?) OR id=?" || len(args) != 4 || args[0] != injection || strings.Contains(got, injection) {
		t.Fatalf("%s %#v %v", got, args, err)
	}
	got, args, err = bindAuditQuery(source, "oracle", values)
	if err != nil || !strings.Contains(got, "IN (:p2,:p3)") || args[0].(sql.NamedArg).Value != injection {
		t.Fatalf("%s %#v %v", got, args, err)
	}
	if _, _, err = bindAuditQuery(source, "mysql", map[string]interface{}{}); err == nil {
		t.Fatal("missing parameter accepted")
	}
	values["people"] = []string{}
	if _, _, err = bindAuditQuery(source, "dm", values); err == nil {
		t.Fatal("empty list accepted")
	}
}

// auditQueryTestConnector 使用独立连接验证事务顺序，不接触真实 OA 数据库。
type auditQueryTestConnector struct{ conn *auditQueryTestConn }

func (c auditQueryTestConnector) Connect(context.Context) (driver.Conn, error) { return c.conn, nil }
func (c auditQueryTestConnector) Driver() driver.Driver                        { return auditQueryTestDriver{c.conn} }

type auditQueryTestDriver struct{ conn *auditQueryTestConn }

func (d auditQueryTestDriver) Open(string) (driver.Conn, error) { return d.conn, nil }

type auditQueryTestConn struct {
	options        driver.TxOptions
	nativeReadOnly bool
	rejectReadOnly bool
	rejectNative   bool
	queries        int
	lastQuery      string
	rolledBack     bool
	committed      bool
	values         []driver.Value
}

func (c *auditQueryTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("未实现")
}
func (c *auditQueryTestConn) Close() error { return nil }
func (c *auditQueryTestConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}
func (c *auditQueryTestConn) BeginTx(_ context.Context, opts driver.TxOptions) (driver.Tx, error) {
	c.options = opts
	if c.rejectReadOnly {
		return nil, errors.New("只读事务不支持")
	}
	return auditQueryTestTx{c}, nil
}
func (c *auditQueryTestConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	if query != "SET TRANSACTION READ ONLY" || c.queries > 0 || c.rejectNative {
		return nil, errors.New("只读声明失败")
	}
	c.nativeReadOnly = true
	return driver.RowsAffected(0), nil
}
func (c *auditQueryTestConn) QueryContext(ctx context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	c.queries++
	c.lastQuery = query
	if !c.options.ReadOnly && !c.nativeReadOnly {
		return nil, errors.New("在只读声明之前执行了查询")
	}
	if _, ok := ctx.Deadline(); !ok {
		return nil, errors.New("查询缺少超时")
	}
	return &auditQueryTestRows{values: c.values}, nil
}

// 不限制行数时不应隐式截为20/100行；自定义上限仍须准确报告截断。
func TestAuditQueryOptionalRowLimit(t *testing.T) {
	for _, dialect := range []string{"mysql", "oracle", "dm"} {
		for _, limit := range []int{0, 101, 200, -1} {
			conn := &auditQueryTestConn{}
			for i := 0; i < 125; i++ {
				conn.values = append(conn.values, int64(i))
			}
			db := sql.OpenDB(auditQueryTestConnector{conn})
			gdb, err := gorm.Open(mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true})
			if err != nil {
				t.Fatal(err)
			}
			adapter := &Ecology9Adapter{db: gdb, driver: dialect}
			result, err := adapter.QueryAuditData(context.Background(), "SELECT amount AS 金额 FROM expenses", nil, limit, 0)
			db.Close()
			if limit < 0 {
				if err == nil || conn.queries != 0 {
					t.Fatalf("负数行数上限被接受：%v", err)
				}
				continue
			}
			wantRows, wantTruncated := 125, false
			if limit == 101 {
				wantRows, wantTruncated = 101, true
			}
			if err != nil || len(result.Rows) != wantRows || result.Truncated != wantTruncated {
				t.Fatalf("%s limit=%d result=%+v err=%v", dialect, limit, result, err)
			}
			wrapped := strings.Contains(conn.lastQuery, "auraoa_query")
			if wrapped != (limit > 0) {
				t.Fatalf("未按配置应用行数限制：%s", conn.lastQuery)
			}
		}
	}
}

// 字符限制按 Unicode 计数，末行完整保留；关闭限制后不再暗中截断大结果。
func TestAuditQueryCharacterLimit(t *testing.T) {
	for _, dialect := range []string{"mysql", "oracle", "dm"} {
		for _, tc := range []struct {
			name                  string
			values                []driver.Value
			rows, chars, wantRows int
			truncated, invalid    bool
		}{
			{name: "无隐藏体积上限", values: []driver.Value{strings.Repeat("中", 70000), strings.Repeat("文", 70000)}, wantRows: 2},
			{name: "行数模式无隐藏体积上限", values: []driver.Value{strings.Repeat("中", 70000), strings.Repeat("文", 70000)}, rows: 2, wantRows: 2},
			{name: "字符计数不按UTF8字节", values: []driver.Value{"中😀", "完整末行", "其他行"}, chars: 15, wantRows: 2, truncated: true},
			{name: "字符上限小于首行", values: []driver.Value{"完整首行😀", "第二行"}, chars: 1, wantRows: 1, truncated: true},
			{name: "恰好完整不标截断", values: []driver.Value{"中😀"}, chars: 13, wantRows: 1},
			{name: "空结果", chars: 1},
			{name: "负数字符上限", chars: -1, invalid: true},
			{name: "两种上限互斥", rows: 1, chars: 10, invalid: true},
		} {
			t.Run(dialect+"/"+tc.name, func(t *testing.T) {
				conn := &auditQueryTestConn{values: tc.values}
				db := sql.OpenDB(auditQueryTestConnector{conn})
				defer db.Close()
				gdb, err := gorm.Open(mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true})
				if err != nil {
					t.Fatal(err)
				}
				adapter := &Ecology9Adapter{db: gdb, driver: dialect}
				result, err := adapter.QueryAuditData(context.Background(), "SELECT amount AS 金额 FROM expenses", nil, tc.rows, tc.chars)
				if tc.invalid {
					if err == nil || conn.queries != 0 {
						t.Fatalf("非法上限执行了查询：%v", err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if len(result.Rows) != tc.wantRows || result.Truncated != tc.truncated {
					t.Fatalf("result=%+v wantRows=%d truncated=%t", result, tc.wantRows, tc.truncated)
				}
				for i, row := range result.Rows {
					if row["col_1"] != tc.values[i] {
						t.Fatal("末行数据被切断")
					}
				}
				if tc.rows == 0 && strings.Contains(conn.lastQuery, "auraoa_query") {
					t.Fatal("字符限制不应注入 SQL 行数限制")
				}
			})
		}
	}
}

type auditQueryTestTx struct{ conn *auditQueryTestConn }

func (t auditQueryTestTx) Commit() error   { t.conn.committed = true; return nil }
func (t auditQueryTestTx) Rollback() error { t.conn.rolledBack = true; return nil }

type auditQueryTestRows struct {
	values []driver.Value
	index  int
}

func (*auditQueryTestRows) Columns() []string                     { return []string{"金额"} }
func (*auditQueryTestRows) ColumnTypeDatabaseTypeName(int) string { return "DECIMAL" }
func (*auditQueryTestRows) Close() error                          { return nil }
func (r *auditQueryTestRows) Next(values []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	values[0] = r.values[r.index]
	r.index++
	return nil
}
func TestAuditQueryReadOnlyTransaction(t *testing.T) {
	for _, dialect := range []string{"mysql", "oracle", "dm"} {
		t.Run(dialect, func(t *testing.T) {
			conn := &auditQueryTestConn{values: []driver.Value{int64(10), int64(20)}}
			db := sql.OpenDB(auditQueryTestConnector{conn})
			defer db.Close()
			gdb, err := gorm.Open(mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true})
			if err != nil {
				t.Fatal(err)
			}
			adapter := &Ecology9Adapter{db: gdb, driver: dialect}
			result, err := adapter.QueryAuditData(context.Background(), "SELECT amount AS 金额 FROM expenses", nil, 1, 0)
			if err != nil || !result.Truncated || len(result.Rows) != 1 || !conn.rolledBack || conn.committed {
				t.Fatalf("result=%+v conn=%+v err=%v", result, conn, err)
			}
			if dialect == "oracle" && !conn.nativeReadOnly || dialect != "oracle" && !conn.options.ReadOnly {
				t.Fatal("missing read-only declaration")
			}
		})
	}
	for _, native := range []bool{false, true} {
		conn := &auditQueryTestConn{rejectReadOnly: !native, rejectNative: native}
		db := sql.OpenDB(auditQueryTestConnector{conn})
		defer db.Close()
		gdb, err := gorm.Open(mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true})
		if err != nil {
			t.Fatal(err)
		}
		dialect := "mysql"
		if native {
			dialect = "oracle"
		}
		adapter := &Ecology9Adapter{db: gdb, driver: dialect}
		if _, err = adapter.QueryAuditData(context.Background(), "SELECT 1 AS 金额", nil, 1, 0); err == nil || conn.queries != 0 {
			t.Fatalf("writable fallback: %+v %v", conn, err)
		}
	}
}

// 防止表结构返回预览行内部键，导致可视化配置生成不存在的列。
func TestAuditQuerySchemaUsesPhysicalColumnKeys(t *testing.T) {
	conn := &auditQueryTestConn{}
	db := sql.OpenDB(auditQueryTestConnector{conn})
	defer db.Close()
	gdb, err := gorm.Open(mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	adapter := &Ecology9Adapter{db: gdb, driver: "mysql"}
	columns, err := adapter.AuditQuerySchema(context.Background(), "expenses")
	if err != nil {
		t.Fatal(err)
	}
	if len(columns) != 1 || columns[0].Key != "金额" {
		t.Fatalf("表结构字段键不正确：%+v", columns)
	}
}

func TestAuditQuerySchemaEnrichesSystemTableLabels(t *testing.T) {
	adapter := &Ecology9Adapter{}
	cols := []QueryColumn{
		{Key: "REQUESTID", Label: "REQUESTID", DataType: "NUMBER"},
		{Key: "requestname", Label: "requestname", DataType: "VARCHAR2"},
		{Key: "unknown_col", Label: "unknown_col", DataType: "VARCHAR2"},
	}
	adapter.enrichQueryColumnLabels(context.Background(), "workflow_requestbase", cols)
	if cols[0].Key != "REQUESTID" || cols[0].Label != "请求ID" {
		t.Fatalf("REQUESTID 未正确补全中文标签：%+v", cols[0])
	}
	if cols[1].Key != "requestname" || cols[1].Label != "请求标题" {
		t.Fatalf("requestname 未正确补全中文标签：%+v", cols[1])
	}
	if cols[2].Key != "unknown_col" || cols[2].Label != "unknown_col" {
		t.Fatalf("未匹配字段应保持原值：%+v", cols[2])
	}
}
