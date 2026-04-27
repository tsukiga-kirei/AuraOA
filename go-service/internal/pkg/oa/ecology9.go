package oa

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"oa-smart-audit/go-service/internal/model"
	pkglogger "oa-smart-audit/go-service/internal/pkg/logger"
	"oa-smart-audit/go-service/internal/pkg/oa/dm"
	"oa-smart-audit/go-service/internal/pkg/oa/oracle"
)

// Ecology9Adapter 泛微 E9 OA 系统适配器。
// 支持 MySQL、Oracle 和 DM（达梦）三种底层数据库驱动。
type Ecology9Adapter struct {
	db     *gorm.DB
	driver string // "mysql" | "oracle" | "dm"
}

// isOracleCompatible 判断当前驱动是否为 Oracle 兼容模式（Oracle / DM）。
func (a *Ecology9Adapter) isOracleCompatible() bool {
	return a.driver == "oracle" || a.driver == "dm"
}

// tableName 根据驱动类型返回正确大小写的表名/列名。
// Oracle/DM 默认大写标识符，MySQL 不区分大小写。
func (a *Ecology9Adapter) tableName(name string) string {
	if a.isOracleCompatible() {
		return strings.ToUpper(name)
	}
	return name
}

// col 与 tableName 相同，用于列名场景，语义更清晰。
func (a *Ecology9Adapter) col(name string) string {
	return a.tableName(name)
}

// NewEcology9Adapter 根据 OA 数据库连接配置创建泛微 E9 适配器实例。
// 通过 conn.Driver 自动选择 MySQL 或 Oracle 驱动。
func NewEcology9Adapter(conn *model.OADatabaseConnection) (*Ecology9Adapter, error) {
	var dialector gorm.Dialector

	switch conn.Driver {
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			conn.Username, conn.Password, conn.Host, conn.Port, conn.DatabaseName)
		dialector = mysql.Open(dsn)
	case "oracle":
		dsn := oracle.BuildDSN(conn.Username, conn.Password, conn.Host, conn.Port, conn.DatabaseName)
		dialector = oracle.Open(dsn)
	case "dm":
		dsn := dm.BuildDSN(conn.Username, conn.Password, conn.Host, conn.Port, conn.DatabaseName)
		dialector = dm.Open(dsn)
	default:
		return nil, fmt.Errorf("泛微 E9 不支持数据库驱动: %s（仅支持 mysql、oracle、dm）", conn.Driver)
	}

	// Oracle/DM 默认将不加引号的标识符转为大写，
	// 泛微 E9 在 Oracle/DM 上的表名和列名均为大写。
	// 配置 NamingStrategy 使 GORM 不自动添加引号、不转小写。
	gormConfig := &gorm.Config{
		// 使用与主库相同的 zap logger，OA 慢查询也写入 app.log
		Logger: pkglogger.NewGormLogger(200*time.Millisecond, true),
	}
	if conn.Driver == "oracle" || conn.Driver == "dm" {
		gormConfig.NamingStrategy = schema.NamingStrategy{
			NoLowerCase: true,
		}
		gormConfig.DisableAutomaticPing = false
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接泛微 E9 数据库失败 (driver=%s): %w", conn.Driver, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接池失败: %w", err)
	}
	sqlDB.SetMaxOpenConns(conn.PoolSize)
	sqlDB.SetMaxIdleConns(conn.PoolSize / 2)

	return &Ecology9Adapter{db: db, driver: conn.Driver}, nil
}

// ── E9 表结构映射 ──────────────────────────────────────────

// e9WorkflowBillField 泛微 E9 workflow_billfield 表映射（流程表单字段定义）
// 注意：Oracle/DM 返回的列名为大写，通过 mapGet() 辅助函数不区分大小写取值。
type e9WorkflowBillField struct {
	FieldDBName   string
	FieldName     string
	FieldHTMLType string
	DetailTable   int
}

func (e9WorkflowBillField) TableName() string { return "workflow_billfield" }

// mapGet 从 map[string]interface{} 中不区分大小写地取字符串值。
func mapGet(m map[string]interface{}, key string) string {
	key = strings.ToLower(key)
	for k, v := range m {
		if strings.ToLower(k) == key {
			if v == nil {
				return ""
			}
			if s, ok := v.(string); ok {
				return s
			}
			return fmt.Sprintf("%v", v)
		}
	}
	return ""
}

// mapGetInt 从 map[string]interface{} 中不区分大小写地取整数值。
func mapGetInt(m map[string]interface{}, key string) int {
	key = strings.ToLower(key)
	for k, v := range m {
		if strings.ToLower(k) == key {
			switch n := v.(type) {
			case int:
				return n
			case int32:
				return int(n)
			case int64:
				return int(n)
			case float64:
				return int(n)
			}
		}
	}
	return 0
}

// ── ValidateProcess ────────────────────────────────────────

// ValidateProcess 验证流程类型是否存在于泛微 E9 系统中。
// 1. 查询 workflow_base，确认流程存在且 isvalid=1，获取 workflowtype
// 2. 查询 workflow_type，获取 typename
// 3. 通过 formid 关联 workflow_bill 获取真实主表名
//
// 使用 Row().Scan() 显式扫描列值，避免 GORM struct tag 大小写映射问题（Oracle/DM 列名大写）。
func (a *Ecology9Adapter) ValidateProcess(ctx context.Context, processType string) (*ProcessInfo, error) {
	// 查询 workflow_base：获取流程名称、formid 和 workflowtype
	var workflowName string
	var formID int
	var workflowTypeID int
	row := a.db.WithContext(ctx).
		Table(a.tableName("workflow_base")).
		Select(a.col("workflowname")+", "+a.col("formid")+", "+a.col("workflowtype")).
		Where(a.col("workflowname")+" = ? AND "+a.col("isvalid")+" = ?", processType, "1").
		Row()
	if err := row.Scan(&workflowName, &formID, &workflowTypeID); err != nil {
		return nil, fmt.Errorf("流程 '%s' 在泛微 E9 系统中不存在或已停用", processType)
	}

	// 查询 workflow_type：获取流类型名称(typename)
	var typeName string
	typeRow := a.db.WithContext(ctx).
		Table(a.tableName("workflow_type")).
		Select(a.col("typename")).
		Where(a.col("id")+" = ?", workflowTypeID).
		Row()
	if err := typeRow.Scan(&typeName); err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("查询流程类型定义失败 (workflowtype=%d): %w", workflowTypeID, err)
	}

	// 通过 formid 查询 workflow_bill，获取真实主表名
	var mainTable string
	billRow := a.db.WithContext(ctx).
		Table(a.tableName("workflow_bill")).
		Select(a.col("tablename")).
		Where(a.col("id")+" = ?", formID).
		Row()
	if err := billRow.Scan(&mainTable); err != nil {
		return nil, fmt.Errorf("查询流程表单定义失败 (formid=%d): %w", formID, err)
	}

	return &ProcessInfo{
		ProcessType:      processType,
		ProcessName:      workflowName,
		ProcessTypeLabel: typeName,
		MainTable:        mainTable,
	}, nil
}

// ── FetchFields ────────────────────────────────────────────

// FetchFields 从泛微 E9 拉取指定流程的全部字段定义。
func (a *Ecology9Adapter) FetchFields(ctx context.Context, processType string) (*ProcessFields, error) {
	// 显式扫描 formid，避免 struct tag 大小写映射问题
	var formID int
	row := a.db.WithContext(ctx).
		Table(a.tableName("workflow_base")).
		Select(a.col("formid")).
		Where(a.col("workflowname")+" = ?", processType).
		Row()
	if err := row.Scan(&formID); err != nil {
		return nil, fmt.Errorf("查询流程 '%s' 失败: %w", processType, err)
	}

	// 通过 formid 查询 workflow_bill，获取真实主表名
	var mainTableName string
	billRow := a.db.WithContext(ctx).
		Table(a.tableName("workflow_bill")).
		Select(a.col("tablename")).
		Where(a.col("id")+" = ?", formID).
		Row()
	if err := billRow.Scan(&mainTableName); err != nil {
		return nil, fmt.Errorf("查询流程表单定义失败 (formid=%d): %w", formID, err)
	}

	var rawFields []map[string]interface{}
	err := a.db.WithContext(ctx).
		Table(a.tableName("workflow_billfield")+" "+a.col("t1")).
		Select(a.col("t1.fieldname")+" AS fieldkey, "+a.col("t2.labelname")+" AS fieldname, "+a.col("t1.fieldhtmltype")+" AS fieldhtmltype, "+a.col("t1.detailtable")+" AS detailtable").
		Joins("JOIN "+a.tableName("htmllabelinfo")+" "+a.col("t2")+" ON "+a.col("t1.fieldlabel")+" = "+a.col("t2.indexid")).
		Where(a.col("t1.billid")+" = ? AND "+a.col("t2.languageid")+" = 7", formID).
		Order(a.col("t1.detailtable") + " ASC, " + a.col("t1.id") + " ASC").
		Find(&rawFields).Error
	if err != nil {
		return nil, fmt.Errorf("查询流程字段失败: %w", err)
	}

	result := &ProcessFields{
		MainFields:   make([]FieldDef, 0),
		DetailTables: make([]DetailTableDef, 0),
	}
	detailMap := make(map[string]*DetailTableDef)
	var detailTableKeys []string

	for _, row := range rawFields {
		fd := FieldDef{
			FieldKey:  mapGet(row, "fieldkey"),
			FieldName: mapGet(row, "fieldname"),
			FieldType: a.mapFieldType(mapGet(row, "fieldhtmltype")),
		}
		dt := strings.TrimSpace(mapGet(row, "detailtable"))

		// E9 中 detailtable 可能为 NULL(解析为空字符串)、"主表" 或对应主表表名
		if dt == "" || strings.EqualFold(dt, "主表") || strings.EqualFold(dt, mainTableName) {
			result.MainFields = append(result.MainFields, fd)
		} else {
			// 部分版本可能只存了一个数字(这算是老表结构)，这里做兼容拼接
			if len(dt) < 3 && !strings.Contains(strings.ToLower(dt), "dt") {
				dt = fmt.Sprintf("%s_dt%s", mainTableName, dt)
			}

			// 从形如 formtable_main_151_dt1 提取出 1 作为显示标签
			label := dt
			if idx := strings.LastIndex(dt, "_dt"); idx != -1 && idx+3 < len(dt) {
				label = "明细表" + dt[idx+3:]
			}

			dtDef, exists := detailMap[dt]
			if !exists {
				dtDef = &DetailTableDef{
					TableName:  dt,
					TableLabel: label,
					Fields:     make([]FieldDef, 0),
				}
				detailMap[dt] = dtDef
				detailTableKeys = append(detailTableKeys, dt)
			}
			dtDef.Fields = append(dtDef.Fields, fd)
		}
	}
	for _, k := range detailTableKeys {
		result.DetailTables = append(result.DetailTables, *detailMap[k])
	}
	return result, nil
}

// ── CheckUserPermission ────────────────────────────────────

// CheckUserPermission 检查用户在泛微 E9 中是否具有指定流程的审批权限。
func (a *Ecology9Adapter) CheckUserPermission(ctx context.Context, username string, processType string) (bool, error) {
	// 1. 通过 loginid 查询 OA 系统内部的数字 ID (id)
	var e9UserID int
	err := a.db.WithContext(ctx).
		Table(a.tableName("hrmresource")).
		Select(a.col("id")).
		Where(a.col("loginid")+" = ?", username).
		Row().Scan(&e9UserID)
	if err != nil {
		// 如果在 OA 中找不到对应用户，则直接返回无权限
		return false, nil
	}

	// 2. 查询流程 ID
	var workflowID int
	row := a.db.WithContext(ctx).
		Table(a.tableName("workflow_base")).
		Select(a.col("id")).
		Where(a.col("workflowname")+" = ?", processType).
		Row()
	if err := row.Scan(&workflowID); err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, fmt.Errorf("查询流程失败: %w", err)
	}

	// 3. 检查权限：workflow_currentoperator 没有 workflowid 列，
	//    需通过 requestid 关联 workflow_requestbase 匹配 workflowid
	var count int64
	coTable := a.tableName("workflow_currentoperator")
	rbTable := a.tableName("workflow_requestbase")
	joinSQL := fmt.Sprintf(
		"JOIN %s r ON %s.%s = r.%s",
		rbTable, coTable, a.col("requestid"), a.col("requestid"),
	)
	err = a.db.WithContext(ctx).
		Table(coTable).
		Joins(joinSQL).
		Where("r."+a.col("workflowid")+" = ? AND "+coTable+"."+a.col("userid")+" = ?", workflowID, e9UserID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("查询用户审批权限失败: %w", err)
	}
	return count > 0, nil
}

// ── FetchProcessData ───────────────────────────────────────

// FetchProcessData 拉取指定流程实例的业务数据。
// 注意：明细表子查询在 Oracle 和 MySQL 中语法不同，需按 driver 分支处理。
func (a *Ecology9Adapter) FetchProcessData(ctx context.Context, processID string) (*ProcessData, error) {
	// 查询流程请求基本信息，显式扫描避免 struct tag 大小写问题
	var workflowID int
	reqRow := a.db.WithContext(ctx).
		Table(a.tableName("workflow_requestbase")).
		Select(a.col("workflowid")).
		Where(a.col("requestid")+" = ?", processID).
		Row()
	if err := reqRow.Scan(&workflowID); err != nil {
		return nil, fmt.Errorf("查询流程实例失败: %w", err)
	}

	// 查询 formid
	var formID int
	wfRow := a.db.WithContext(ctx).
		Table(a.tableName("workflow_base")).
		Select(a.col("formid")).
		Where(a.col("id")+" = ?", workflowID).
		Row()
	if err := wfRow.Scan(&formID); err != nil {
		return nil, fmt.Errorf("查询流程定义失败: %w", err)
	}

	// 通过 formid 关联 workflow_bill 获取真实主表名
	var tableDBName string
	billRow := a.db.WithContext(ctx).
		Table(a.tableName("workflow_bill")).
		Select(a.col("tablename")).
		Where(a.col("id")+" = ?", formID).
		Row()
	if err := billRow.Scan(&tableDBName); err != nil {
		return nil, fmt.Errorf("查询流程表单定义失败 (formid=%d): %w", formID, err)
	}

	// 查询主表数据
	mainTableName := a.tableName(tableDBName)
	var mainData map[string]interface{}
	err := a.db.WithContext(ctx).
		Table(mainTableName).
		Where(a.col("requestid")+" = ?", processID).
		Take(&mainData).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("查询主表数据失败: %w", err)
	}

	// 查询明细表数量
	var detailCount int64
	a.db.WithContext(ctx).
		Table(a.tableName("workflow_billfield")).
		Where(a.col("billid")+" = ? AND "+a.col("detailtable")+" > 0", formID).
		Distinct(a.col("detailtable")).
		Count(&detailCount)

	// 查询各明细表数据，按表名分组
	detailTables := make(map[string][]map[string]interface{})
	for i := 1; i <= int(detailCount); i++ {
		dtRawName := fmt.Sprintf("%s_dt%d", tableDBName, i)
		dtTableName := a.tableName(dtRawName)
		var rows []map[string]interface{}

		subQuery := fmt.Sprintf(
			"EXISTS (SELECT 1 FROM %s m WHERE m.%s = %s.%s AND m.%s = ?)",
			mainTableName,
			a.col("id"), dtTableName, a.col("mainid"),
			a.col("requestid"),
		)
		a.db.WithContext(ctx).
			Table(dtTableName).
			Where(subQuery, processID).
			Find(&rows)

		if len(rows) > 0 {
			detailTables[dtRawName] = rows
		}
	}

	return &ProcessData{
		ProcessID:    processID,
		MainData:     mainData,
		DetailTables: detailTables,
	}, nil
}

// ── FetchTodoList ──────────────────────────────────────────

// FetchTodoList 拉取用户在泛微 E9 中的待审批流程列表。
// 查询 workflow_currentoperator 获取用户待办，关联 workflow_requestbase 获取流程信息。
// 兼容 MySQL / Oracle / DM 三种驱动。
func (a *Ecology9Adapter) FetchTodoList(ctx context.Context, username string, filter TodoListFilter) ([]TodoItem, error) {
	var e9UserID int
	err := a.db.WithContext(ctx).
		Table(a.tableName("hrmresource")).
		Select(a.col("id")).
		Where(a.col("loginid")+" = ?", username).
		Row().Scan(&e9UserID)
	if err != nil {
		return nil, fmt.Errorf("OA 用户 '%s' 不存在", username)
	}

	createDateCol := "r." + a.col("createdate")
	var dateCond string
	var dateArgs []interface{}
	if filter.SubmitDateStart != nil {
		dateCond += fmt.Sprintf(" AND %s >= ?", createDateCol)
		dateArgs = append(dateArgs, *filter.SubmitDateStart)
	}
	if filter.SubmitDateEndExclusive != nil {
		dateCond += fmt.Sprintf(" AND %s < ?", createDateCol)
		dateArgs = append(dateArgs, *filter.SubmitDateEndExclusive)
	}

	// 查询待办：workflow_currentoperator + requestbase + base + bill + type + node
	// 使用 DISTINCT 避免同一流程多个审批节点导致重复
	query := fmt.Sprintf(`
		SELECT DISTINCT
			r.%s AS request_id,
			r.%s AS request_name,
			COALESCE(h.%s, '') AS applicant_name,
			COALESCE(d.%s, '') AS dept_name,
			COALESCE(wb.%s, '') AS workflow_name,
			COALESCE(wt.%s, '') AS type_name,
			COALESCE(n.%s, '') AS node_name,
			r.%s AS create_date,
			COALESCE(bill.%s, '') AS main_table_name
		FROM %s co
		JOIN %s r ON co.%s = r.%s
		LEFT JOIN %s wb ON r.%s = wb.%s
		LEFT JOIN %s wt ON wb.%s = wt.%s
		LEFT JOIN %s bill ON wb.%s = bill.%s
		LEFT JOIN %s h ON r.%s = h.%s
		LEFT JOIN %s d ON h.%s = d.%s
		LEFT JOIN %s n ON co.%s = n.%s
		WHERE co.%s = ? AND co.%s = 0%s
		ORDER BY r.%s DESC`,
		// SELECT
		a.col("requestid"), a.col("requestname"),
		a.col("lastname"), a.col("departmentname"),
		a.col("workflowname"), a.col("typename"),
		a.col("nodename"),
		a.col("createdate"),
		a.col("tablename"), // bill.tablename → 主表名
		// FROM
		a.tableName("workflow_currentoperator"), // co
		// JOINs
		a.tableName("workflow_requestbase"), // r
		a.col("requestid"), a.col("requestid"),
		a.tableName("workflow_base"), // wb
		a.col("workflowid"), a.col("id"),
		a.tableName("workflow_type"), // wt
		a.col("workflowtype"), a.col("id"),
		a.tableName("workflow_bill"), // bill (通过 formid 获取主表名)
		a.col("formid"), a.col("id"),
		a.tableName("hrmresource"), // h (applicant)
		a.col("creater"), a.col("id"),
		a.tableName("hrmdepartment"), // d
		a.col("departmentid"), a.col("id"),
		a.tableName("workflow_nodebase"), // n
		a.col("nodeid"), a.col("id"),
		// WHERE
		a.col("userid"), a.col("isremark"),
		dateCond,
		// ORDER BY
		a.col("createdate"),
	)

	args := []interface{}{e9UserID}
	args = append(args, dateArgs...)
	rows, err := a.db.WithContext(ctx).Raw(query, args...).Rows()
	if err != nil {
		return nil, fmt.Errorf("查询 OA 待办失败: %w", err)
	}
	defer rows.Close()

	var items []TodoItem
	for rows.Next() {
		var requestID, requestName, applicant, department, workflowName, typeName, nodeName, createDate, mainTableName string
		if err := rows.Scan(&requestID, &requestName, &applicant, &department, &workflowName, &typeName, &nodeName, &createDate, &mainTableName); err != nil {
			continue
		}
		items = append(items, TodoItem{
			ProcessID:        requestID,
			Title:            requestName,
			Applicant:        applicant,
			Department:       department,
			ProcessType:      workflowName,
			ProcessTypeLabel: typeName,
			CurrentNode:      nodeName,
			SubmitTime:       createDate,
			Urgency:          "medium",
			MainTableName:    mainTableName,
		})
	}
	return items, nil
}

// FetchArchivedList 拉取泛微 E9 中的已归档流程。
// 不同客户库对归档时间字段可能不一致，因此优先尝试 lastoperatedate，失败时回退到 createdate。
// filter 中的归档日期范围在 SQL WHERE 中生效，与 ORDER BY 使用同一归档时间表达式。
func (a *Ecology9Adapter) FetchArchivedList(ctx context.Context, username string, filter ArchivedListFilter) ([]ArchivedItem, error) {
	_ = username
	items, err := a.fetchArchivedListWithArchiveDate(ctx, true, filter)
	if err == nil {
		return items, nil
	}
	return a.fetchArchivedListWithArchiveDate(ctx, false, filter)
}

// FetchTodoListPaged 分页拉取待办列表，将 keyword/applicant/department/mainTableNames 筛选下推到 OA SQL，
// 同时使用 COUNT + LIMIT/OFFSET 实现真分页，避免全量拉取。
func (a *Ecology9Adapter) FetchTodoListPaged(ctx context.Context, username string, filter TodoListPagedFilter) (*PagedResult[TodoItem], error) {
	var e9UserID int
	err := a.db.WithContext(ctx).
		Table(a.tableName("hrmresource")).
		Select(a.col("id")).
		Where(a.col("loginid")+" = ?", username).
		Row().Scan(&e9UserID)
	if err != nil {
		return nil, fmt.Errorf("OA 用户 '%s' 不存在", username)
	}

	// 构建公共 FROM + JOIN + WHERE
	fromJoinWhere, args := a.buildTodoFromJoinWhere(e9UserID, filter)

	// 1. COUNT 查询（按 requestid 去重，避免同一流程多个审批节点导致重复计数）
	countSQL := "SELECT COUNT(DISTINCT r." + a.col("requestid") + ") " + fromJoinWhere
	var total int
	if err := a.db.WithContext(ctx).Raw(countSQL, args...).Row().Scan(&total); err != nil {
		return nil, fmt.Errorf("查询 OA 待办总数失败: %w", err)
	}

	page, pageSize := filter.Page, filter.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 1000 {
		pageSize = 20
	}

	if total == 0 {
		return &PagedResult[TodoItem]{Items: []TodoItem{}, Total: 0}, nil
	}

	// 2. 数据查询（带 LIMIT/OFFSET）
	selectCols := fmt.Sprintf(`
		r.%s AS request_id,
		r.%s AS request_name,
		COALESCE(h.%s, '') AS applicant_name,
		COALESCE(d.%s, '') AS dept_name,
		COALESCE(wb.%s, '') AS workflow_name,
		COALESCE(wt.%s, '') AS type_name,
		COALESCE(n.%s, '') AS node_name,
		r.%s AS create_date,
		COALESCE(bill.%s, '') AS main_table_name`,
		a.col("requestid"), a.col("requestname"),
		a.col("lastname"), a.col("departmentname"),
		a.col("workflowname"), a.col("typename"),
		a.col("nodename"),
		a.col("createdate"),
		a.col("tablename"),
	)

	offset := (page - 1) * pageSize
	dataSQL := "SELECT DISTINCT " + selectCols + " " + fromJoinWhere +
		fmt.Sprintf(" ORDER BY r.%s DESC", a.col("createdate")) +
		a.limitOffsetClause(pageSize, offset)

	rows, err := a.db.WithContext(ctx).Raw(dataSQL, args...).Rows()
	if err != nil {
		return nil, fmt.Errorf("查询 OA 待办失败: %w", err)
	}
	defer rows.Close()

	var items []TodoItem
	for rows.Next() {
		var requestID, requestName, applicant, department, workflowName, typeName, nodeName, createDate, mainTableName string
		if err := rows.Scan(&requestID, &requestName, &applicant, &department, &workflowName, &typeName, &nodeName, &createDate, &mainTableName); err != nil {
			continue
		}
		items = append(items, TodoItem{
			ProcessID:        requestID,
			Title:            requestName,
			Applicant:        applicant,
			Department:       department,
			ProcessType:      workflowName,
			ProcessTypeLabel: typeName,
			CurrentNode:      nodeName,
			SubmitTime:       createDate,
			Urgency:          "medium",
			MainTableName:    mainTableName,
		})
	}
	return &PagedResult[TodoItem]{Items: items, Total: total}, nil
}

// buildTodoFromJoinWhere 构建待办查询的 FROM + JOIN + WHERE 子句（不含 SELECT 和 ORDER BY），
// 供 COUNT 和数据查询共用。
func (a *Ecology9Adapter) buildTodoFromJoinWhere(e9UserID int, filter TodoListPagedFilter) (string, []interface{}) {
	var conds string
	var args []interface{}

	// 日期条件
	createDateCol := "r." + a.col("createdate")
	if filter.SubmitDateStart != nil {
		conds += fmt.Sprintf(" AND %s >= ?", createDateCol)
		args = append(args, *filter.SubmitDateStart)
	}
	if filter.SubmitDateEndExclusive != nil {
		conds += fmt.Sprintf(" AND %s < ?", createDateCol)
		args = append(args, *filter.SubmitDateEndExclusive)
	}

	// keyword → 模糊匹配 requestname
	if kw := strings.TrimSpace(filter.Keyword); kw != "" {
		conds += fmt.Sprintf(" AND %s(r.%s) LIKE ?", a.lowerFunc(), a.col("requestname"))
		args = append(args, "%"+strings.ToLower(kw)+"%")
	}

	// applicant → 模糊匹配 hrmresource.lastname
	if ap := strings.TrimSpace(filter.Applicant); ap != "" {
		conds += fmt.Sprintf(" AND %s(h.%s) LIKE ?", a.lowerFunc(), a.col("lastname"))
		args = append(args, "%"+strings.ToLower(ap)+"%")
	}

	// department → 精确匹配 hrmdepartment.departmentname
	if dept := strings.TrimSpace(filter.Department); dept != "" {
		conds += fmt.Sprintf(" AND d.%s = ?", a.col("departmentname"))
		args = append(args, dept)
	}

	// mainTableNames → 限制 bill.tablename
	if len(filter.MainTableNames) > 0 {
		placeholders := make([]string, len(filter.MainTableNames))
		for i, name := range filter.MainTableNames {
			placeholders[i] = "?"
			args = append(args, strings.ToLower(name))
		}
		conds += fmt.Sprintf(" AND %s(COALESCE(bill.%s, '')) IN (%s)",
			a.lowerFunc(), a.col("tablename"), strings.Join(placeholders, ","))
	}

	// processTypes → 限制 workflow_base.workflowname
	if len(filter.ProcessTypes) > 0 {
		placeholders := make([]string, len(filter.ProcessTypes))
		for i, pt := range filter.ProcessTypes {
			placeholders[i] = "?"
			args = append(args, strings.ToLower(pt))
		}
		conds += fmt.Sprintf(" AND %s(COALESCE(wb.%s, '')) IN (%s)",
			a.lowerFunc(), a.col("workflowname"), strings.Join(placeholders, ","))
	}

	fromJoinWhere := fmt.Sprintf(`FROM %s co
		JOIN %s r ON co.%s = r.%s
		LEFT JOIN %s wb ON r.%s = wb.%s
		LEFT JOIN %s wt ON wb.%s = wt.%s
		LEFT JOIN %s bill ON wb.%s = bill.%s
		LEFT JOIN %s h ON r.%s = h.%s
		LEFT JOIN %s d ON h.%s = d.%s
		LEFT JOIN %s n ON co.%s = n.%s
		WHERE co.%s = ? AND co.%s = 0%s`,
		a.tableName("workflow_currentoperator"),
		a.tableName("workflow_requestbase"),
		a.col("requestid"), a.col("requestid"),
		a.tableName("workflow_base"),
		a.col("workflowid"), a.col("id"),
		a.tableName("workflow_type"),
		a.col("workflowtype"), a.col("id"),
		a.tableName("workflow_bill"),
		a.col("formid"), a.col("id"),
		a.tableName("hrmresource"),
		a.col("creater"), a.col("id"),
		a.tableName("hrmdepartment"),
		a.col("departmentid"), a.col("id"),
		a.tableName("workflow_nodebase"),
		a.col("nodeid"), a.col("id"),
		a.col("userid"), a.col("isremark"),
		conds,
	)

	// e9UserID 放在最前面（对应 WHERE co.userid = ?）
	allArgs := []interface{}{e9UserID}
	allArgs = append(allArgs, args...)
	return fromJoinWhere, allArgs
}

// FetchArchivedListPaged 分页拉取已归档流程列表，将筛选条件下推到 OA SQL。
func (a *Ecology9Adapter) FetchArchivedListPaged(ctx context.Context, username string, filter ArchivedListPagedFilter) (*PagedResult[ArchivedItem], error) {
	_ = username
	result, err := a.fetchArchivedListPagedWithArchiveDate(ctx, true, filter)
	if err == nil {
		return result, nil
	}
	return a.fetchArchivedListPagedWithArchiveDate(ctx, false, filter)
}

// fetchArchivedListPagedWithArchiveDate 分页查询已归档流程，支持 COUNT + LIMIT/OFFSET 真分页。
func (a *Ecology9Adapter) fetchArchivedListPagedWithArchiveDate(ctx context.Context, useLastOperateDate bool, filter ArchivedListPagedFilter) (*PagedResult[ArchivedItem], error) {
	archiveDateExpr := "r." + a.col("createdate")
	if useLastOperateDate {
		archiveDateExpr = fmt.Sprintf("COALESCE(r.%s, r.%s)", a.col("lastoperatedate"), a.col("createdate"))
	}

	// 构建公共 FROM + JOIN + WHERE
	fromJoinWhere, args := a.buildArchivedFromJoinWhere(archiveDateExpr, filter)

	// 1. COUNT 查询
	countSQL := "SELECT COUNT(*) " + fromJoinWhere
	var total int
	if err := a.db.WithContext(ctx).Raw(countSQL, args...).Row().Scan(&total); err != nil {
		return nil, fmt.Errorf("查询 OA 已归档流程总数失败: %w", err)
	}

	page, pageSize := filter.Page, filter.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 1000 {
		pageSize = 20
	}

	if total == 0 {
		return &PagedResult[ArchivedItem]{Items: []ArchivedItem{}, Total: 0}, nil
	}

	// 2. 数据查询
	selectCols := fmt.Sprintf(`
		r.%s AS request_id,
		r.%s AS request_name,
		COALESCE(h.%s, '') AS applicant_name,
		COALESCE(d.%s, '') AS dept_name,
		COALESCE(wb.%s, '') AS workflow_name,
		COALESCE(wt.%s, '') AS type_name,
		COALESCE(n.%s, '已归档') AS node_name,
		r.%s AS create_date,
		%s AS archive_date,
		COALESCE(bill.%s, '') AS main_table_name`,
		a.col("requestid"), a.col("requestname"),
		a.col("lastname"), a.col("departmentname"),
		a.col("workflowname"), a.col("typename"),
		a.col("nodename"),
		a.col("createdate"),
		archiveDateExpr,
		a.col("tablename"),
	)

	offset := (page - 1) * pageSize
	dataSQL := "SELECT " + selectCols + " " + fromJoinWhere +
		fmt.Sprintf(" ORDER BY %s DESC", archiveDateExpr) +
		a.limitOffsetClause(pageSize, offset)

	rows, err := a.db.WithContext(ctx).Raw(dataSQL, args...).Rows()
	if err != nil {
		return nil, fmt.Errorf("查询 OA 已归档流程失败: %w", err)
	}
	defer rows.Close()

	var items []ArchivedItem
	for rows.Next() {
		var requestID, requestName, applicant, department, workflowName, typeName, nodeName, createDate, archiveDate, mainTableName string
		if err := rows.Scan(&requestID, &requestName, &applicant, &department, &workflowName, &typeName, &nodeName, &createDate, &archiveDate, &mainTableName); err != nil {
			continue
		}
		items = append(items, ArchivedItem{
			ProcessID:        requestID,
			Title:            requestName,
			Applicant:        applicant,
			Department:       department,
			ProcessType:      workflowName,
			ProcessTypeLabel: typeName,
			CurrentNode:      nodeName,
			SubmitTime:       createDate,
			ArchiveTime:      archiveDate,
			MainTableName:    mainTableName,
		})
	}
	return &PagedResult[ArchivedItem]{Items: items, Total: total}, nil
}

// buildArchivedFromJoinWhere 构建已归档查询的 FROM + JOIN + WHERE 子句。
func (a *Ecology9Adapter) buildArchivedFromJoinWhere(archiveDateExpr string, filter ArchivedListPagedFilter) (string, []interface{}) {
	var conds string
	var args []interface{}

	// 日期条件
	if filter.ArchiveDateStart != nil {
		conds += fmt.Sprintf(" AND (%s) >= ?", archiveDateExpr)
		args = append(args, *filter.ArchiveDateStart)
	}
	if filter.ArchiveDateEndExclusive != nil {
		conds += fmt.Sprintf(" AND (%s) < ?", archiveDateExpr)
		args = append(args, *filter.ArchiveDateEndExclusive)
	}

	// keyword → 模糊匹配 requestname
	if kw := strings.TrimSpace(filter.Keyword); kw != "" {
		conds += fmt.Sprintf(" AND %s(r.%s) LIKE ?", a.lowerFunc(), a.col("requestname"))
		args = append(args, "%"+strings.ToLower(kw)+"%")
	}

	// applicant → 模糊匹配 hrmresource.lastname
	if ap := strings.TrimSpace(filter.Applicant); ap != "" {
		conds += fmt.Sprintf(" AND %s(h.%s) LIKE ?", a.lowerFunc(), a.col("lastname"))
		args = append(args, "%"+strings.ToLower(ap)+"%")
	}

	// department → 精确匹配 hrmdepartment.departmentname
	if dept := strings.TrimSpace(filter.Department); dept != "" {
		conds += fmt.Sprintf(" AND d.%s = ?", a.col("departmentname"))
		args = append(args, dept)
	}

	// mainTableNames 和 processTypes 必须同时满足（AND 关系）
	if len(filter.MainTableNames) > 0 {
		placeholders := make([]string, len(filter.MainTableNames))
		for i, name := range filter.MainTableNames {
			placeholders[i] = "?"
			args = append(args, strings.ToLower(name))
		}
		conds += fmt.Sprintf(" AND %s(COALESCE(bill.%s, '')) IN (%s)",
			a.lowerFunc(), a.col("tablename"), strings.Join(placeholders, ","))
	}
	if len(filter.ProcessTypes) > 0 {
		placeholders := make([]string, len(filter.ProcessTypes))
		for i, pt := range filter.ProcessTypes {
			placeholders[i] = "?"
			args = append(args, strings.ToLower(pt))
		}
		conds += fmt.Sprintf(" AND %s(COALESCE(wb.%s, '')) IN (%s)",
			a.lowerFunc(), a.col("workflowname"), strings.Join(placeholders, ","))
	}

	fromJoinWhere := fmt.Sprintf(`FROM %s r
		LEFT JOIN %s wb ON r.%s = wb.%s
		LEFT JOIN %s wt ON wb.%s = wt.%s
		LEFT JOIN %s bill ON wb.%s = bill.%s
		LEFT JOIN %s h ON r.%s = h.%s
		LEFT JOIN %s d ON h.%s = d.%s
		LEFT JOIN %s n ON r.%s = n.%s
		WHERE r.%s = 3%s`,
		a.tableName("workflow_requestbase"),
		a.tableName("workflow_base"),
		a.col("workflowid"), a.col("id"),
		a.tableName("workflow_type"),
		a.col("workflowtype"), a.col("id"),
		a.tableName("workflow_bill"),
		a.col("formid"), a.col("id"),
		a.tableName("hrmresource"),
		a.col("creater"), a.col("id"),
		a.tableName("hrmdepartment"),
		a.col("departmentid"), a.col("id"),
		a.tableName("workflow_nodebase"),
		a.col("currentnodeid"), a.col("id"),
		a.col("currentnodetype"),
		conds,
	)

	return fromJoinWhere, args
}

// lowerFunc 返回当前数据库驱动的小写函数名。
func (a *Ecology9Adapter) lowerFunc() string {
	return "LOWER"
}

// limitOffsetClause 根据数据库驱动生成分页子句。
// MySQL/DM: LIMIT n OFFSET m
// Oracle 12c+: OFFSET m ROWS FETCH NEXT n ROWS ONLY
func (a *Ecology9Adapter) limitOffsetClause(limit, offset int) string {
	if a.driver == "oracle" {
		return fmt.Sprintf(" OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", offset, limit)
	}
	// MySQL 和 DM 都支持 LIMIT/OFFSET
	return fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
}

// FetchAllTodoItems 拉取所有待审批流程（不过滤用户，供调度器批处理使用）。
// 与 FetchTodoList 相比，去掉了 WHERE co.userid = ? 条件，并对结果去重（同一流程可能出现在多个审批人的待办中）。
func (a *Ecology9Adapter) FetchAllTodoItems(ctx context.Context, limit int) ([]TodoItem, error) {
	query := fmt.Sprintf(`
		SELECT DISTINCT
			r.%s AS request_id,
			r.%s AS request_name,
			COALESCE(h.%s, '') AS applicant_name,
			COALESCE(d.%s, '') AS dept_name,
			COALESCE(wb.%s, '') AS workflow_name,
			COALESCE(wt.%s, '') AS type_name,
			COALESCE(n.%s, '') AS node_name,
			r.%s AS create_date,
			COALESCE(bill.%s, '') AS main_table_name
		FROM %s co
		JOIN %s r ON co.%s = r.%s
		LEFT JOIN %s wb ON r.%s = wb.%s
		LEFT JOIN %s wt ON wb.%s = wt.%s
		LEFT JOIN %s bill ON wb.%s = bill.%s
		LEFT JOIN %s h ON r.%s = h.%s
		LEFT JOIN %s d ON h.%s = d.%s
		LEFT JOIN %s n ON co.%s = n.%s
		WHERE co.%s = 0
		ORDER BY r.%s DESC`,
		// SELECT
		a.col("requestid"), a.col("requestname"),
		a.col("lastname"), a.col("departmentname"),
		a.col("workflowname"), a.col("typename"),
		a.col("nodename"),
		a.col("createdate"),
		a.col("tablename"),
		// FROM
		a.tableName("workflow_currentoperator"),
		a.tableName("workflow_requestbase"),
		a.col("requestid"), a.col("requestid"),
		a.tableName("workflow_base"),
		a.col("workflowid"), a.col("id"),
		a.tableName("workflow_type"),
		a.col("workflowtype"), a.col("id"),
		a.tableName("workflow_bill"),
		a.col("formid"), a.col("id"),
		a.tableName("hrmresource"),
		a.col("creater"), a.col("id"),
		a.tableName("hrmdepartment"),
		a.col("departmentid"), a.col("id"),
		a.tableName("workflow_nodebase"),
		a.col("nodeid"), a.col("id"),
		// WHERE
		a.col("isremark"),
		// ORDER BY
		a.col("createdate"),
	)

	db := a.db.WithContext(ctx)
	if limit > 0 {
		db = db.Limit(limit)
	}
	rows, err := db.Raw(query).Rows()
	if err != nil {
		return nil, fmt.Errorf("查询 OA 全量待办失败: %w", err)
	}
	defer rows.Close()

	var items []TodoItem
	seen := make(map[string]struct{})
	for rows.Next() {
		var requestID, requestName, applicant, department, workflowName, typeName, nodeName, createDate, mainTableName string
		if err := rows.Scan(&requestID, &requestName, &applicant, &department, &workflowName, &typeName, &nodeName, &createDate, &mainTableName); err != nil {
			continue
		}
		if _, dup := seen[requestID]; dup {
			continue
		}
		seen[requestID] = struct{}{}
		items = append(items, TodoItem{
			ProcessID:        requestID,
			Title:            requestName,
			Applicant:        applicant,
			Department:       department,
			ProcessType:      workflowName,
			ProcessTypeLabel: typeName,
			CurrentNode:      nodeName,
			SubmitTime:       createDate,
			Urgency:          "medium",
			MainTableName:    mainTableName,
		})
	}
	return items, nil
}

func (a *Ecology9Adapter) fetchArchivedListWithArchiveDate(ctx context.Context, useLastOperateDate bool, filter ArchivedListFilter) ([]ArchivedItem, error) {
	archiveDateExpr := "r." + a.col("createdate")
	if useLastOperateDate {
		archiveDateExpr = fmt.Sprintf("COALESCE(r.%s, r.%s)", a.col("lastoperatedate"), a.col("createdate"))
	}

	var dateCond string
	var dateArgs []interface{}
	if filter.ArchiveDateStart != nil {
		dateCond += fmt.Sprintf(" AND (%s) >= ?", archiveDateExpr)
		dateArgs = append(dateArgs, *filter.ArchiveDateStart)
	}
	if filter.ArchiveDateEndExclusive != nil {
		dateCond += fmt.Sprintf(" AND (%s) < ?", archiveDateExpr)
		dateArgs = append(dateArgs, *filter.ArchiveDateEndExclusive)
	}

	query := fmt.Sprintf(`
		SELECT
			r.%s AS request_id,
			r.%s AS request_name,
			COALESCE(h.%s, '') AS applicant_name,
			COALESCE(d.%s, '') AS dept_name,
			COALESCE(wb.%s, '') AS workflow_name,
			COALESCE(wt.%s, '') AS type_name,
			COALESCE(n.%s, '已归档') AS node_name,
			r.%s AS create_date,
			%s AS archive_date,
			COALESCE(bill.%s, '') AS main_table_name
		FROM %s r
		LEFT JOIN %s wb ON r.%s = wb.%s
		LEFT JOIN %s wt ON wb.%s = wt.%s
		LEFT JOIN %s bill ON wb.%s = bill.%s
		LEFT JOIN %s h ON r.%s = h.%s
		LEFT JOIN %s d ON h.%s = d.%s
		LEFT JOIN %s n ON r.%s = n.%s
		WHERE r.%s = 3%s
		ORDER BY %s DESC`,
		a.col("requestid"), a.col("requestname"),
		a.col("lastname"), a.col("departmentname"),
		a.col("workflowname"), a.col("typename"),
		a.col("nodename"),
		a.col("createdate"),
		archiveDateExpr,
		a.col("tablename"),
		a.tableName("workflow_requestbase"),
		a.tableName("workflow_base"),
		a.col("workflowid"), a.col("id"),
		a.tableName("workflow_type"),
		a.col("workflowtype"), a.col("id"),
		a.tableName("workflow_bill"),
		a.col("formid"), a.col("id"),
		a.tableName("hrmresource"),
		a.col("creater"), a.col("id"),
		a.tableName("hrmdepartment"),
		a.col("departmentid"), a.col("id"),
		a.tableName("workflow_nodebase"),
		a.col("currentnodeid"), a.col("id"),
		a.col("currentnodetype"),
		dateCond,
		archiveDateExpr,
	)

	rows, err := a.db.WithContext(ctx).Raw(query, dateArgs...).Rows()
	if err != nil {
		return nil, fmt.Errorf("查询 OA 已归档流程失败: %w", err)
	}
	defer rows.Close()

	var items []ArchivedItem
	for rows.Next() {
		var requestID, requestName, applicant, department, workflowName, typeName, nodeName, createDate, archiveDate, mainTableName string
		if err := rows.Scan(&requestID, &requestName, &applicant, &department, &workflowName, &typeName, &nodeName, &createDate, &archiveDate, &mainTableName); err != nil {
			continue
		}
		items = append(items, ArchivedItem{
			ProcessID:        requestID,
			Title:            requestName,
			Applicant:        applicant,
			Department:       department,
			ProcessType:      workflowName,
			ProcessTypeLabel: typeName,
			CurrentNode:      nodeName,
			SubmitTime:       createDate,
			ArchiveTime:      archiveDate,
			MainTableName:    mainTableName,
		})
	}
	return items, nil
}

// FetchProcessFlow 拉取流程审批流快照。
// 包含完整审批历史（带操作类型映射）和流程路由图（带出口条件）。
// 若历史日志表结构不兼容，则退化为仅返回当前节点快照，避免阻塞主链路。
func (a *Ecology9Adapter) FetchProcessFlow(ctx context.Context, processID string) (*ProcessFlowSnapshot, error) {
	// ── 1. 获取审批历史（仅最后一次退回之后的有效路径） ──
	historyQuery := fmt.Sprintf(`
		SELECT
			WRL.%s AS log_id,
			COALESCE(WNB.%s, '') AS node_name,
			WRL.%s AS log_type,
			COALESCE(HR.%s, '') AS operator_name,
			COALESCE(WRL.%s, '') AS remark,
			COALESCE(WRL.%s, '') AS operate_date,
			COALESCE(WRL.%s, '') AS operate_time
		FROM %s WRL
		LEFT JOIN %s WNB ON WRL.%s = WNB.%s
		LEFT JOIN %s HR ON WRL.%s = HR.%s
		WHERE WRL.%s = ?
		  AND WRL.%s > (
		    SELECT COALESCE(MAX(%s), 0) FROM %s WHERE %s = WRL.%s AND %s = '3'
		  )
		ORDER BY WRL.%s ASC`,
		a.col("logid"),
		a.col("nodename"),
		a.col("logtype"),
		a.col("lastname"),
		a.col("remark"),
		a.col("operatedate"),
		a.col("operatetime"),
		a.tableName("workflow_requestlog"),
		a.tableName("workflow_nodebase"), a.col("nodeid"), a.col("id"),
		a.tableName("hrmresource"), a.col("operator"), a.col("id"),
		a.col("requestid"),
		a.col("logid"),
		a.col("logid"), a.tableName("workflow_requestlog"), a.col("requestid"), a.col("requestid"), a.col("logtype"),
		a.col("logid"),
	)

	rows, err := a.db.WithContext(ctx).Raw(historyQuery, processID).Rows()
	if err != nil {
		return a.fetchCurrentNodeSnapshot(ctx, processID)
	}
	defer rows.Close()

	var nodes []ProcessFlowNode
	var historyLines []string
	for rows.Next() {
		var logID int
		var nodeName, logType, operator, remark, operateDate, operateTime string
		if err := rows.Scan(&logID, &nodeName, &logType, &operator, &remark, &operateDate, &operateTime); err != nil {
			continue
		}
		action := mapLogType(logType)
		actionTime := strings.TrimSpace(operateDate + " " + operateTime)
		nodes = append(nodes, ProcessFlowNode{
			NodeID:     nodeName,
			NodeName:   nodeName,
			Approver:   operator,
			Action:     action,
			ActionTime: actionTime,
			Opinion:    remark,
		})
		historyLines = append(historyLines, fmt.Sprintf("%s | %s | %s | %s | %s", actionTime, nodeName, operator, action, remark))
	}

	if len(nodes) == 0 {
		return a.fetchCurrentNodeSnapshot(ctx, processID)
	}

	// ── 2. 获取流程路由图（节点连接 + 出口条件） ──
	graphText := a.fetchFlowRouteGraph(ctx, processID)

	// 如果路由图为空，退化为简单节点路径
	if graphText == "" {
		nodeNames := make([]string, 0, len(nodes))
		seen := make(map[string]bool)
		for _, node := range nodes {
			if !seen[node.NodeName] {
				nodeNames = append(nodeNames, node.NodeName)
				seen[node.NodeName] = true
			}
		}
		graphText = strings.Join(nodeNames, " → ")
	}

	return &ProcessFlowSnapshot{
		IsComplete:   true,
		MissingNodes: []string{},
		Nodes:        nodes,
		HistoryText:  strings.Join(historyLines, "\n"),
		GraphText:    graphText,
	}, nil
}

// mapLogType 将泛微 E9 的 LOGTYPE 代码转换为可读的操作类型文本。
func mapLogType(logType string) string {
	switch strings.TrimSpace(logType) {
	case "0":
		return "批准"
	case "1":
		return "保存"
	case "2":
		return "提交"
	case "3":
		return "退回"
	case "4":
		return "重新打开"
	case "5":
		return "删除"
	case "6":
		return "激活"
	case "7":
		return "转发"
	case "9":
		return "批注"
	case "e":
		return "强制归档"
	case "t":
		return "抄送"
	case "i":
		return "干预"
	default:
		return "其他(" + logType + ")"
	}
}

// fetchFlowRouteGraph 获取流程定义的路由图（节点连接关系和出口条件）。
// 通过 requestid 关联 workflow_requestbase 获取 workflowid，再查询 workflow_nodelink。
func (a *Ecology9Adapter) fetchFlowRouteGraph(ctx context.Context, processID string) string {
	// 获取 workflowid
	var workflowID int
	err := a.db.WithContext(ctx).
		Table(a.tableName("workflow_requestbase")).
		Select(a.col("workflowid")).
		Where(a.col("requestid")+" = ?", processID).
		Row().Scan(&workflowID)
	if err != nil {
		return ""
	}

	// 查询节点连接和出口条件
	query := fmt.Sprintf(`
		SELECT
			COALESCE(WN1.%s, '') AS src_node_name,
			COALESCE(WN2.%s, '') AS dest_node_name,
			COALESCE(WN.%s, '') AS link_name,
			COALESCE(RB.%s, '') AS condition_text
		FROM %s WN
		LEFT JOIN %s WN1 ON WN1.%s = WN.%s
		LEFT JOIN %s WN2 ON WN2.%s = WN.%s
		LEFT JOIN %s RB ON TO_CHAR(RB.%s) = WN.%s
		WHERE WN.%s = ?
		ORDER BY WN.%s, WN.%s`,
		a.col("nodename"),
		a.col("nodename"),
		a.col("linkname"),
		a.col("condit"),
		a.tableName("workflow_nodelink"),
		a.tableName("workflow_nodebase"), a.col("id"), a.col("nodeid"),
		a.tableName("workflow_nodebase"), a.col("id"), a.col("destnodeid"),
		a.tableName("rule_base"), a.col("id"), a.col("newrule"),
		a.col("workflowid"),
		a.col("nodeid"), a.col("destnodeid"),
	)

	// Oracle/DM 使用 TO_CHAR，MySQL 需要 CAST
	if !a.isOracleCompatible() {
		query = fmt.Sprintf(`
			SELECT
				COALESCE(WN1.%s, '') AS src_node_name,
				COALESCE(WN2.%s, '') AS dest_node_name,
				COALESCE(WN.%s, '') AS link_name,
				COALESCE(RB.%s, '') AS condition_text
			FROM %s WN
			LEFT JOIN %s WN1 ON WN1.%s = WN.%s
			LEFT JOIN %s WN2 ON WN2.%s = WN.%s
			LEFT JOIN %s RB ON CAST(RB.%s AS CHAR) = WN.%s
			WHERE WN.%s = ?
			ORDER BY WN.%s, WN.%s`,
			a.col("nodename"),
			a.col("nodename"),
			a.col("linkname"),
			a.col("condit"),
			a.tableName("workflow_nodelink"),
			a.tableName("workflow_nodebase"), a.col("id"), a.col("nodeid"),
			a.tableName("workflow_nodebase"), a.col("id"), a.col("destnodeid"),
			a.tableName("rule_base"), a.col("id"), a.col("newrule"),
			a.col("workflowid"),
			a.col("nodeid"), a.col("destnodeid"),
		)
	}

	rows, err := a.db.WithContext(ctx).Raw(query, workflowID).Rows()
	if err != nil {
		return ""
	}
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var srcNode, destNode, linkName, condText string
		if err := rows.Scan(&srcNode, &destNode, &linkName, &condText); err != nil {
			continue
		}
		line := srcNode + " → " + destNode
		if linkName != "" {
			line += " [" + linkName + "]"
		}
		if condText != "" {
			line += " 条件: " + condText
		}
		lines = append(lines, line)
	}

	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}

func (a *Ecology9Adapter) fetchCurrentNodeSnapshot(ctx context.Context, processID string) (*ProcessFlowSnapshot, error) {
	query := fmt.Sprintf(`
		SELECT
			COALESCE(n.%s, '已归档')
		FROM %s r
		LEFT JOIN %s n ON r.%s = n.%s
		WHERE r.%s = ?`,
		a.col("nodename"),
		a.tableName("workflow_requestbase"),
		a.tableName("workflow_nodebase"),
		a.col("currentnodeid"), a.col("id"),
		a.col("requestid"),
	)

	var nodeName string
	if err := a.db.WithContext(ctx).Raw(query, processID).Row().Scan(&nodeName); err != nil {
		return &ProcessFlowSnapshot{
			IsComplete:   true,
			MissingNodes: []string{},
			Nodes:        []ProcessFlowNode{},
			HistoryText:  "",
			GraphText:    "",
		}, nil
	}

	node := ProcessFlowNode{
		NodeID:   nodeName,
		NodeName: nodeName,
		Action:   "approve",
	}

	return &ProcessFlowSnapshot{
		IsComplete:   true,
		MissingNodes: []string{},
		Nodes:        []ProcessFlowNode{node},
		HistoryText:  nodeName,
		GraphText:    nodeName,
	}, nil
}

// IsProcessInTodo 判断指定流程是否仍在用户待办中。
func (a *Ecology9Adapter) IsProcessInTodo(ctx context.Context, username string, processID string) (bool, error) {
	var e9UserID int
	err := a.db.WithContext(ctx).
		Table(a.tableName("hrmresource")).
		Select(a.col("id")).
		Where(a.col("loginid")+" = ?", username).
		Row().Scan(&e9UserID)
	if err != nil {
		return false, nil
	}

	var count int64
	err = a.db.WithContext(ctx).
		Table(a.tableName("workflow_currentoperator")).
		Where(a.col("userid")+" = ? AND "+a.col("requestid")+" = ? AND "+a.col("isremark")+" = 0",
			e9UserID, processID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("查询待办状态失败: %w", err)
	}
	return count > 0, nil
}

// ── mapFieldType ───────────────────────────────────────────

// mapFieldType 将泛微 E9 的字段 HTML 类型映射为通用字段类型。
func (a *Ecology9Adapter) mapFieldType(htmlType string) string {
	switch htmlType {
	case "1": // 单行文本框
		return "text"
	case "2": // 多行文本框
		return "textarea"
	case "3": // 浏览按钮
		return "select"
	case "4": // check框
		return "checkbox"
	case "5": // 选择框
		return "select"
	case "6": // 附件上传 (泛微 E9 附件通常是 6)
		return "file"
	default:
		return "text"
	}
}
