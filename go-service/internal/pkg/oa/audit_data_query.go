package oa

import (
	"auraoa/go-service/internal/pkg/apptime"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	pkglogger "auraoa/go-service/internal/pkg/logger"
	"go.uber.org/zap"
)

// QuerySQLIssue 保存可定位的检查结果；AI 提示不参与只读校验。
type QuerySQLIssue struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Position int    `json:"position"`
}

// QuerySQLCheck 是不执行 SQL 的结构检查报告。
type QuerySQLCheck struct {
	Valid      bool            `json:"valid"`
	Issues     []QuerySQLIssue `json:"issues"`
	Parameters []string        `json:"parameters"`
	Columns    []string        `json:"columns"`
}
type queryToken struct {
	text              string
	kind              byte
	start, end, depth int
}

var queryIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
var queryDetailTableSuffix = regexp.MustCompile(`(?i)_dt[0-9]+$`)

// tokenizeQuerySQL 对引号、注释、括号与参数分词，避免将字符串中的 AS 或冒号误识别。
func tokenizeQuerySQL(input string) ([]queryToken, error) {
	var out []queryToken
	depth := 0
	for i := 0; i < len(input); {
		r, n := utf8.DecodeRuneInString(input[i:])
		if unicode.IsSpace(r) {
			i += n
			continue
		}
		start := i
		if i+1 < len(input) && (input[i:i+2] == "--" || input[i:i+2] == "/*") || input[i] == '#' {
			return nil, fmt.Errorf("不支持 SQL 注释，请将说明填写在数据说明中（位置 %d）", i+1)
		}
		ch := input[i]
		if ch == '\'' || ch == '"' || ch == '`' {
			i++
			closed := false
			for i < len(input) {
				if input[i] == '\\' {
					return nil, fmt.Errorf("请使用成对引号转义，不支持反斜杠转义（位置 %d）", i+1)
				}
				if input[i] == ch {
					i++
					if i < len(input) && input[i] == ch {
						i++
						continue
					}
					closed = true
					break
				}
				i++
			}
			if !closed {
				return nil, fmt.Errorf("引号未闭合（位置 %d）", start+1)
			}
			kind := byte('q')
			if ch == '\'' {
				kind = 's'
			}
			out = append(out, queryToken{input[start:i], kind, start, i, depth})
			continue
		}
		if ch == ':' {
			i++
			for i < len(input) && (input[i] == '_' || input[i] >= 'a' && input[i] <= 'z' || input[i] >= 'A' && input[i] <= 'Z' || input[i] >= '0' && input[i] <= '9') {
				i++
			}
			if i == start+1 {
				return nil, fmt.Errorf("命名参数无效（位置 %d）", start+1)
			}
			out = append(out, queryToken{input[start+1 : i], 'p', start, i, depth})
			continue
		}
		if unicode.IsLetter(r) || ch == '_' {
			i += n
			for i < len(input) {
				rr, nn := utf8.DecodeRuneInString(input[i:])
				if !unicode.IsLetter(rr) && !unicode.IsDigit(rr) && rr != '_' {
					break
				}
				i += nn
			}
			out = append(out, queryToken{input[start:i], 'w', start, i, depth})
			continue
		}
		if ch == ';' || ch == '@' || ch == '?' || ch == '$' || ch == '\\' {
			return nil, fmt.Errorf("不支持的 SQL 符号 %q（位置 %d），参数请使用 :name", ch, i+1)
		}
		if ch == ')' {
			depth--
			if depth < 0 {
				return nil, fmt.Errorf("括号不匹配")
			}
		}
		out = append(out, queryToken{string(ch), 'x', start, start + 1, depth})
		if ch == '(' {
			depth++
		}
		i++
	}
	if depth != 0 {
		return nil, fmt.Errorf("括号不匹配")
	}
	return out, nil
}
func tokenWord(t queryToken, s string) bool { return t.kind == 'w' && strings.EqualFold(t.text, s) }
func queryAlias(t queryToken) string {
	if t.kind == 'q' {
		q := t.text[:1]
		return strings.ReplaceAll(t.text[1:len(t.text)-1], q+q, q)
	}
	if t.kind == 'w' {
		return t.text
	}
	return ""
}
func hasChinese(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// CheckQuerySQL 检查单条 SELECT/WITH 查询、最终输出 AS 中文名和命名参数。
// 此检查限定支持的查询结构，数据库语法和字段存在性仍须通过受限预览确认。
func CheckQuerySQL(input string) QuerySQLCheck {
	result := QuerySQLCheck{Issues: []QuerySQLIssue{}, Parameters: []string{}, Columns: []string{}}
	add := func(code, msg string, pos int) {
		result.Issues = append(result.Issues, QuerySQLIssue{code, msg, len(utf16.Encode([]rune(input[:pos]))) + 1})
	}
	if len(input) > 30000 {
		add("sql_size", "SQL 超过 30000 字节", 0)
		return result
	}
	ts, err := tokenizeQuerySQL(input)
	if err != nil {
		add("syntax", err.Error(), 0)
		return result
	}
	if len(ts) == 0 || !(tokenWord(ts[0], "select") || tokenWord(ts[0], "with")) {
		add("readonly", "仅支持单条 SELECT 或只读 WITH 查询", 0)
		return result
	}
	blocked := map[string]bool{}
	for _, w := range strings.Fields(`
		INSERT UPDATE DELETE MERGE DROP ALTER TRUNCATE CREATE GRANT REVOKE CALL EXEC EXECUTE INTO
		OUTFILE DUMPFILE LOAD HANDLER LOCK UNLOCK SET COMMIT ROLLBACK SAVEPOINT NEXTVAL CURRVAL
		SLEEP BENCHMARK GET_LOCK RELEASE_LOCK DBMS_LOCK UTL_HTTP UTL_FILE DBMS_PIPE DBMS_XMLGEN
		DBMS_SCHEDULER CONNECT_BY_ROOT MODEL MATCH_RECOGNIZE
	`) {
		blocked[w] = true
	}
	// 只允许常见无副作用内置函数，防止只读 SELECT 调用自定义函数、外部网络或自治事务。
	functions := map[string]bool{}
	for _, name := range strings.Fields(`
		COUNT SUM AVG MIN MAX COALESCE NULLIF IFNULL NVL NVL2 IF DECODE CASE CAST CONVERT TO_DATE
		TO_CHAR TO_NUMBER DATE_FORMAT STR_TO_DATE DATEDIFF TIMESTAMPDIFF DATE_ADD DATE_SUB
		ADDDATE SUBDATE ADD_MONTHS LAST_DAY TRUNC ROUND CEIL CEILING FLOOR ABS MOD POWER SQRT
		SIGN CONCAT CONCAT_WS SUBSTR SUBSTRING LEFT RIGHT TRIM LTRIM RTRIM LOWER UPPER LENGTH
		CHAR_LENGTH REPLACE REGEXP_REPLACE REGEXP_LIKE REGEXP_SUBSTR INSTR LOCATE EXTRACT YEAR
		MONTH DAY DAYOFMONTH NOW SYSDATE CURRENT_DATE CURRENT_TIMESTAMP ROW_NUMBER RANK
		DENSE_RANK LAG LEAD FIRST_VALUE LAST_VALUE LISTAGG GROUP_CONCAT STRING_AGG JSON_VALUE
		JSON_EXTRACT JSON_UNQUOTE GREATEST LEAST STDDEV VARIANCE DATE INT INTEGER BIGINT DECIMAL
		NUMERIC NUMBER VARCHAR VARCHAR2 CHAR TIMESTAMP PARTITION ROLLUP CUBE GROUPING
	`) {
		functions[name] = true
	}
	for i, token := range ts {
		if i+1 >= len(ts) || ts[i+1].text != "(" || token.kind != 'w' && token.kind != 'q' {
			continue
		}
		upper := strings.ToUpper(token.text)
		// 这些括号属于查询语法，不是函数调用；CTE 列名列表暂不支持。
		if token.kind == 'w' {
			switch upper {
			case "AS", "IN", "EXISTS", "OVER", "USING", "SELECT", "FROM", "JOIN", "NOT",
				"AND", "OR", "ON", "WHEN", "THEN", "ELSE", "WHERE", "HAVING", "BY",
				"DISTINCT", "ALL", "BETWEEN", "FILTER":
				continue
			}
		}
		if token.kind == 'q' || !functions[upper] || i > 0 && ts[i-1].text == "." {
			add("function", "不支持的函数或限定调用："+token.text, token.start)
		}
	}
	params := map[string]bool{}
	sels := []int{}
	for i, t := range ts {
		if t.kind == 'w' && blocked[strings.ToUpper(t.text)] {
			add("readonly", "不允许的关键字或函数："+t.text, t.start)
		}
		if t.kind == 'p' {
			if !queryIdentifier.MatchString(t.text) {
				add("parameter", "参数名称只允许英文、数字和下划线", t.start)
			}
			params[t.text] = true
		}
		if tokenWord(t, "select") && t.depth == 0 {
			sels = append(sels, i)
		}
	}
	for p := range params {
		result.Parameters = append(result.Parameters, p)
	}
	sort.Strings(result.Parameters)
	if len(sels) == 0 {
		add("syntax", "未找到最终 SELECT", 0)
	}
	for si, sel := range sels {
		end := len(ts)
		for j := sel + 1; j < len(ts); j++ {
			if ts[j].depth == 0 && ts[j].kind == 'w' && strings.Contains(
				" FROM UNION INTERSECT EXCEPT MINUS ORDER LIMIT FETCH WHERE ",
				" "+strings.ToUpper(ts[j].text)+" ",
			) {
				end = j
				break
			}
		}
		first := sel + 1
		if first < end && (tokenWord(ts[first], "distinct") || tokenWord(ts[first], "all")) {
			first++
		}
		columns := []string{}
		seen := map[string]bool{}
		for j := first; j <= end; j++ {
			if j != end && !(ts[j].text == "," && ts[j].depth == 0) {
				continue
			}
			part := ts[first:j]
			first = j + 1
			if len(part) < 3 || !tokenWord(part[len(part)-2], "as") {
				pos := ts[sel].start
				if len(part) > 0 {
					pos = part[0].start
				}
				add("column_alias", "每个最终输出列必须显式 AS 中文别名", pos)
				continue
			}
			alias := queryAlias(part[len(part)-1])
			if !hasChinese(alias) || len([]rune(alias)) > 100 {
				add("column_alias", "输出别名须包含中文业务含义且不超过100字", part[len(part)-1].start)
			}
			if seen[alias] {
				add("duplicate_alias", "输出列名重复："+alias, part[len(part)-1].start)
			}
			seen[alias] = true
			columns = append(columns, alias)
			for k, t := range part[:len(part)-2] {
				if t.text == "*" && t.depth == 0 && (k == 0 || part[k-1].text == ".") {
					add("wildcard", "最终输出不能使用 *，请逐列 AS 中文别名", t.start)
				}
			}
		}
		if si == 0 {
			result.Columns = columns
		} else if len(columns) != len(result.Columns) {
			add("union_columns", "集合查询各分支输出列数量不一致", ts[sel].start)
		}
	}
	if len(result.Columns) > 40 {
		add("column_limit", "最多允许40个输出列", 0)
	}
	result.Valid = len(result.Issues) == 0
	return result
}

// QueryColumn 保存数据库列类型和显示名；结果行使用内部键，表结构使用物理字段键。
type QueryColumn struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	DataType string `json:"data_type"`
}

// AuditQueryResult 包含完整性标记，行数上限不会被当作完整汇总。
type AuditQueryResult struct {
	Columns    []QueryColumn            `json:"columns"`
	Rows       []map[string]interface{} `json:"rows"`
	Truncated  bool                     `json:"truncated"`
	DurationMs int64                    `json:"duration_ms"`
}

// AuditQueryExecutor 是独立于关联流程及建模挂载的 OA 只读查询能力。
type AuditQueryExecutor interface {
	QueryAuditData(context.Context, string, map[string]interface{}, int, int) (*AuditQueryResult, error)
	AuditQuerySchema(context.Context, string) ([]QueryColumn, error)
	AuditQueryDialect() string
}

func (a *Ecology9Adapter) AuditQueryDialect() string { return a.driver }

// bindAuditQuery 仅替换词法识别到的值参数，列表展开也逐项绑定。
func bindAuditQuery(input, driver string, values map[string]interface{}) (string, []interface{}, error) {
	ts, err := tokenizeQuerySQL(input)
	if err != nil {
		return "", nil, err
	}
	var b strings.Builder
	args := []interface{}{}
	last := 0
	appendValue := func(v interface{}) {
		if driver == "oracle" {
			name := fmt.Sprintf("p%d", len(args)+1)
			b.WriteString(":" + name)
			args = append(args, sql.Named(name, v))
		} else {
			b.WriteString("?")
			args = append(args, v)
		}
	}
	for _, t := range ts {
		if t.kind != 'p' {
			continue
		}
		v, ok := values[t.text]
		if !ok || v == nil {
			return "", nil, fmt.Errorf("参数 %s 缺少值", t.text)
		}
		b.WriteString(input[last:t.start])
		switch list := v.(type) {
		case []string:
			if len(list) == 0 || len(list) > 100 {
				return "", nil, fmt.Errorf("列表参数 %s 需要1至100个值", t.text)
			}
			for i, item := range list {
				if i > 0 {
					b.WriteString(",")
				}
				appendValue(item)
			}
		default:
			appendValue(v)
		}
		last = t.end
	}
	b.WriteString(input[last:])
	return b.String(), args, nil
}

// QueryAuditData 在数据库只读事务中执行，失败关闭，不降级为可写事务。
func (a *Ecology9Adapter) QueryAuditData(ctx context.Context, input string, values map[string]interface{}, maxRows, maxChars int) (*AuditQueryResult, error) {
	check := CheckQuerySQL(input)
	if !check.Valid {
		return nil, fmt.Errorf("%s", check.Issues[0].Message)
	}
	query, args, err := bindAuditQuery(input, a.driver, values)
	if err != nil {
		return nil, err
	}
	if maxRows < 0 {
		return nil, fmt.Errorf("最大返回行数不能为负数")
	}
	if maxChars < 0 || maxRows > 0 && maxChars > 0 {
		return nil, fmt.Errorf("最大字符数不能为负数，且不能与行数上限同时设置")
	}
	// 0 表示不限制行数；指定上限时在外层多读取一行判定截断，不改变内部聚合或排序。
	if maxRows > 0 {
		limit := fmt.Sprint(uint64(maxRows) + 1)
		if a.driver == "mysql" {
			query = "SELECT * FROM (" + query + ") auraoa_query LIMIT " + limit
		} else {
			query = "SELECT * FROM (" + query + ") auraoa_query WHERE ROWNUM <= " + limit
		}
	}
	return a.runAuditReadOnly(ctx, query, args, maxRows, maxChars)
}
func (a *Ecology9Adapter) runAuditReadOnly(ctx context.Context, query string, args []interface{}, maxRows, maxChars int) (*AuditQueryResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	started := time.Now()
	db, err := a.db.DB()
	if err != nil {
		return nil, err
	}
	options := &sql.TxOptions{ReadOnly: a.driver != "oracle"}
	tx, err := db.BeginTx(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("无法开启只读事务：%w", err)
	}
	defer tx.Rollback()
	// go-ora 不支持 ReadOnly 选项，使用 Oracle 原生事务语句作为首条语句。
	if a.driver == "oracle" {
		if _, err = tx.ExecContext(ctx, "SET TRANSACTION READ ONLY"); err != nil {
			return nil, fmt.Errorf("无法设置只读事务：%w", err)
		}
	}
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	types, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	out := &AuditQueryResult{Columns: []QueryColumn{}, Rows: []map[string]interface{}{}}
	seen := map[string]bool{}
	for i, t := range types {
		label := t.Name()
		if seen[label] {
			return nil, fmt.Errorf("数据库返回重复列名：%s", label)
		}
		seen[label] = true
		out.Columns = append(out.Columns, QueryColumn{fmt.Sprintf("col_%d", i+1), label, t.DatabaseTypeName()})
	}
	// 按传给模型的中文列名 JSON 计算 Unicode 字符，包含数组括号和行间逗号。
	size := 2
	for rows.Next() {
		if maxRows > 0 && len(out.Rows) >= maxRows || maxChars > 0 && len(out.Rows) > 0 && size >= maxChars {
			out.Truncated = true
			break
		}
		vals := make([]interface{}, len(types))
		ptrs := make([]interface{}, len(types))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err = rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := map[string]interface{}{}
		for i, v := range vals {
			switch vv := v.(type) {
			case []byte:
				v = string(vv)
			case time.Time:
				v = apptime.FormatRFC3339(vv)
			}
			row[out.Columns[i].Key] = v
		}
		if maxChars > 0 {
			named := map[string]interface{}{}
			for _, col := range out.Columns {
				named[col.Label] = row[col.Key]
			}
			raw, e := json.Marshal(named)
			if e != nil {
				return nil, e
			}
			size += utf8.RuneCount(raw)
			if len(out.Rows) > 0 {
				size++
			}
		}
		out.Rows = append(out.Rows, row)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	out.DurationMs = time.Since(started).Milliseconds()
	return out, nil
}

// AuditQuerySchema 读取用户指定表的零行列元数据，不扫描业务记录。
func (a *Ecology9Adapter) AuditQuerySchema(ctx context.Context, table string) ([]QueryColumn, error) {
	if !queryIdentifier.MatchString(table) {
		return nil, fmt.Errorf("表名只允许英文、数字和下划线")
	}
	result, err := a.runAuditReadOnly(ctx, "SELECT * FROM "+a.tableName(table)+" WHERE 1=0", nil, 1, 0)
	if err != nil {
		return nil, err
	}
	if len(result.Columns) > 300 {
		return nil, fmt.Errorf("表字段超过300个，请缩小数据范围")
	}
	// 表结构必须返回物理字段名，不能复用结果预览中的 col_1 等内部行键。
	for i := range result.Columns {
		result.Columns[i].Key = result.Columns[i].Label
	}
	a.enrichQueryColumnLabels(ctx, table, result.Columns)
	return result.Columns, nil
}

// enrichQueryColumnLabels 为物理列补充泛微流程表单、明细表、建模表或系统表的中文业务标签。
func (a *Ecology9Adapter) enrichQueryColumnLabels(ctx context.Context, table string, columns []QueryColumn) {
	if a == nil || len(columns) == 0 {
		return
	}
	labels := make(map[string]string)

	// 1. 系统核心表常用字段静态兜底（无论数据库驱动，命中即生效）
	switch strings.ToLower(table) {
	case "workflow_requestbase":
		labels["requestid"] = "请求ID"
		labels["requestname"] = "请求标题"
		labels["workflowid"] = "工作流ID"
		labels["lastnodeid"] = "上一节点ID"
		labels["currentnodeid"] = "当前节点ID"
		labels["currentnodetype"] = "当前节点类型"
		labels["status"] = "流程状态"
		labels["creater"] = "创建人"
		labels["createdate"] = "创建日期"
		labels["createtime"] = "创建时间"
		labels["lastoperator"] = "最后操作人"
		labels["lastoperatedate"] = "最后操作日期"
		labels["lastoperatetime"] = "最后操作时间"
		labels["requestlevel"] = "紧急程度"
		labels["requestmark"] = "流程编号"
	case "hrmresource":
		labels["id"] = "人员ID"
		labels["workcode"] = "工号"
		labels["lastname"] = "姓名"
		labels["loginid"] = "登录账号"
		labels["departmentid"] = "部门ID"
		labels["subcompanyid1"] = "分部ID"
		labels["jobtitle"] = "岗位ID"
		labels["managerid"] = "直接上级ID"
		labels["status"] = "状态"
		labels["mobile"] = "手机号"
		labels["email"] = "邮箱"
		labels["telephone"] = "办公电话"
	case "hrmdepartment":
		labels["id"] = "部门ID"
		labels["departmentname"] = "部门名称"
		labels["departmentmark"] = "部门编号"
		labels["subcompanyid1"] = "分部ID"
		labels["supdepid"] = "上级部门ID"
	case "workflow_currentoperator":
		labels["requestid"] = "请求ID"
		labels["userid"] = "操作人ID"
		labels["groupdetailid"] = "节点操作组ID"
		labels["nodeid"] = "节点ID"
		labels["isremark"] = "操作类型"
		labels["usertype"] = "用户类型"
		labels["operatedate"] = "操作日期"
		labels["operatetime"] = "操作时间"
		labels["viewtype"] = "查看状态"
	}

	// 2. 先按主表查询字典，再按字段归属区分主表与明细表，兼容明细表只存序号。
	if a.db != nil {
		mainTable := table
		if suffix := queryDetailTableSuffix.FindStringIndex(table); suffix != nil {
			mainTable = table[:suffix[0]]
		}
		// Oracle/达梦返回大写列名，沿用 OA 适配器的 mapGet，避免结构体小写标签漏接值。
		var rows []map[string]interface{}
		err := a.db.WithContext(ctx).
			Table(a.tableName("workflow_billfield")+" "+a.col("t1")).
			Select(a.col("t1.fieldname")+" AS fieldkey, "+a.col("t2.labelname")+" AS fieldname, "+a.col("t1.detailtable")+" AS detailtable, "+a.col("b.tablename")+" AS maintable").
			Joins("JOIN "+a.tableName("htmllabelinfo")+" "+a.col("t2")+" ON "+a.col("t1.fieldlabel")+" = "+a.col("t2.indexid")+" AND "+a.col("t2.languageid")+" = 7").
			Joins("LEFT JOIN "+a.tableName("workflow_bill")+" "+a.col("b")+" ON "+a.col("t1.billid")+" = "+a.col("b.id")).
			Where("LOWER("+a.col("b.tablename")+") = LOWER(?) OR LOWER("+a.col("t1.detailtable")+") = LOWER(?)", mainTable, table).
			Find(&rows).Error

		if err != nil {
			pkglogger.Global().Warn("读取查询表字段中文标签失败，保留物理字段名", zap.String("table", table), zap.Error(err))
		} else {
			for _, r := range rows {
				rowMain := strings.TrimSpace(mapGet(r, "maintable"))
				if rowMain == "" {
					rowMain = mainTable
				}
				owner := normalizeDetailTableKey(rowMain, mapGet(r, "detailtable"))
				if owner == "main" {
					owner = rowMain
				}
				if !strings.EqualFold(owner, table) {
					continue
				}
				k := strings.ToLower(strings.TrimSpace(mapGet(r, "fieldkey")))
				v := strings.TrimSpace(mapGet(r, "fieldname"))
				if k != "" && v != "" {
					labels[k] = v
				}
			}
		}
	}

	for i := range columns {
		k := strings.ToLower(columns[i].Key)
		if label, ok := labels[k]; ok && label != "" {
			columns[i].Label = label
		}
	}
}

// ReferencedQueryTables 提取各层 FROM/JOIN 表；限制同一连接下的未限定表名，避免跨库绕过表范围。
func ReferencedQueryTables(input string) ([]string, error) {
	ts, err := tokenizeQuerySQL(input)
	if err != nil {
		return nil, err
	}
	ctes := map[string]bool{}
	tables := map[string]bool{}
	if len(ts) > 0 && tokenWord(ts[0], "with") {
		expect := true
		for i := 1; i < len(ts); i++ {
			t := ts[i]
			if t.depth != 0 {
				continue
			}
			if tokenWord(t, "select") {
				break
			}
			if expect {
				if t.kind != 'w' && t.kind != 'q' {
					return nil, fmt.Errorf("CTE 名称无效")
				}
				ctes[strings.ToLower(queryAlias(t))] = true
				expect = false
			}
			if t.text == "," {
				expect = true
			}
		}
	}
	inFrom := map[int]bool{}
	want := map[int]bool{}
	for i, t := range ts {
		d := t.depth
		if tokenWord(t, "from") || tokenWord(t, "join") {
			inFrom[d] = true
			want[d] = true
			continue
		}
		if t.kind == 'w' && strings.Contains(" WHERE GROUP ORDER HAVING UNION INTERSECT EXCEPT MINUS LIMIT FETCH CONNECT START ", " "+strings.ToUpper(t.text)+" ") {
			inFrom[d] = false
			want[d] = false
		}
		if t.text == "," && inFrom[d] {
			want[d] = true
			continue
		}
		if !want[d] {
			continue
		}
		want[d] = false
		if t.text == "(" {
			continue
		}
		if t.kind != 'w' && t.kind != 'q' {
			return nil, fmt.Errorf("FROM/JOIN 后必须为固定表名或子查询")
		}
		name := queryAlias(t)
		if !queryIdentifier.MatchString(name) {
			return nil, fmt.Errorf("表名无效：%s", name)
		}
		if i+1 < len(ts) && (ts[i+1].text == "." || ts[i+1].text == "(") {
			return nil, fmt.Errorf("不支持跨库表名或表函数：%s", name)
		}
		lower := strings.ToLower(name)
		if !ctes[lower] && lower != "dual" {
			tables[lower] = true
		}
	}
	out := []string{}
	for table := range tables {
		out = append(out, table)
	}
	sort.Strings(out)
	return out, nil
}
