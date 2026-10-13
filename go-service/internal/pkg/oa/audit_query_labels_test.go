package oa

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// queryLabelConnector 模拟真实字典返回的列名大小写及 NULL、字节值，不连接 OA。
type queryLabelConnector struct {
	conn *queryLabelConn
}

func (c queryLabelConnector) Connect(context.Context) (driver.Conn, error) { return c.conn, nil }
func (c queryLabelConnector) Driver() driver.Driver                        { return auditQueryTestDriver{} }

type queryLabelConn struct {
	auditQueryTestConn
	columnNames []string
	data        [][]driver.Value
	err         error
	args        []driver.NamedValue
}

func (c *queryLabelConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.lastQuery, c.args = query, args
	return &queryLabelRows{names: c.columnNames, data: c.data}, c.err
}

type queryLabelRows struct {
	names []string
	data  [][]driver.Value
	index int
}

func (r *queryLabelRows) Columns() []string { return r.names }
func (r *queryLabelRows) Close() error      { return nil }
func (r *queryLabelRows) Next(values []driver.Value) error {
	if r.index == len(r.data) {
		return io.EOF
	}
	copy(values, r.data[r.index])
	r.index++
	return nil
}

func TestAuditQuerySchemaDictionaryLabels(t *testing.T) {
	for _, dbDriver := range []string{"mysql", "oracle", "dm"} {
		for _, table := range []string{"formtable_main_199", "formtable_main_199_dt1", "uf_contract", "uf_contract_dt1"} {
			t.Run(dbDriver+"/"+table, func(t *testing.T) {
				main := strings.TrimSuffix(table, "_dt1")
				names := []string{"fieldkey", "fieldname", "detailtable", "maintable"}
				if dbDriver != "mysql" {
					for i := range names {
						names[i] = strings.ToUpper(names[i])
					}
				}
				conn := &queryLabelConn{columnNames: names, data: [][]driver.Value{
					{[]byte("AMOUNT"), []byte("主表金额"), nil, strings.ToUpper(main)},
					{"PERSON", "申请人", "0", main},
					{"TITLE", "标题", "主表", main},
					{"NAME", "名称", main, main},
					{"AMOUNT", "明细金额", "1", main},
					{"PERSON", "明细人员", strings.ToUpper(main) + "_DT1", main},
					{"AMOUNT", "其他明细金额", main + "_dt2", main},
				}}
				db := sql.OpenDB(queryLabelConnector{conn})
				defer db.Close()
				gdb, err := gorm.Open(mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true}), &gorm.Config{
					DisableAutomaticPing: true, NamingStrategy: schema.NamingStrategy{NoLowerCase: dbDriver != "mysql"},
				})
				if err != nil {
					t.Fatal(err)
				}
				adapter := &Ecology9Adapter{db: gdb, driver: dbDriver}
				columns := []QueryColumn{{"AMOUNT", "AMOUNT", "NUMBER"}, {"person", "person", "VARCHAR"}, {"TITLE", "TITLE", "VARCHAR"}, {"NAME", "NAME", "VARCHAR"}, {"unknown", "unknown", "INT"}}
				adapter.enrichQueryColumnLabels(context.Background(), strings.ToUpper(table), columns)
				want := []string{"主表金额", "申请人", "标题", "名称", "unknown"}
				if table != main {
					want = []string{"明细金额", "明细人员", "TITLE", "NAME", "unknown"}
				}
				for i, label := range want {
					if columns[i].Label != label {
						t.Fatalf("字段 %s: got %q, want %q", columns[i].Key, columns[i].Label, label)
					}
				}
				if len(conn.args) != 2 || !strings.EqualFold(conn.args[0].Value.(string), main) {
					t.Fatalf("明细表未按主表查字典: %+v", conn.args)
				}
				if columns[0].Key != "AMOUNT" || columns[0].DataType != "NUMBER" {
					t.Fatal("中文标签不应改变物理字段与类型")
				}
			})
		}
	}
}

func TestAuditQuerySchemaDictionaryFailureKeepsPhysicalColumns(t *testing.T) {
	conn := &queryLabelConn{err: errors.New("字典不可访问")}
	db := sql.OpenDB(queryLabelConnector{conn})
	defer db.Close()
	gdb, err := gorm.Open(mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	columns := []QueryColumn{{"UNKNOWN", "UNKNOWN", "NUMBER"}, {"REQUESTID", "REQUESTID", "NUMBER"}}
	adapter := &Ecology9Adapter{db: gdb, driver: "mysql"}
	adapter.enrichQueryColumnLabels(context.Background(), "workflow_requestbase", columns)
	if columns[0].Label != "UNKNOWN" || columns[1].Label != "请求ID" {
		t.Fatalf("字典失败时兜底被破坏: %+v", columns)
	}
}
