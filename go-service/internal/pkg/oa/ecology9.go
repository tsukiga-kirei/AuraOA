package oa

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"oa-smart-audit/go-service/internal/model"
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
	gormConfig := &gorm.Config{}
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

	// 3. 检查权限 (userid 在 E9 中是数字类型)
	var count int64
	err = a.db.WithContext(ctx).
		Table(a.tableName("workflow_currentoperator")).
		Where(a.col("workflowid")+" = ? AND "+a.col("userid")+" = ?", workflowID, e9UserID).
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

	// 查询流程对应的主表名和 formid
	var tableDBName string
	var formID int
	wfRow := a.db.WithContext(ctx).
		Table(a.tableName("workflow_base")).
		Select(a.col("tablename")+", "+a.col("formid")).
		Where(a.col("id")+" = ?", workflowID).
		Row()
	if err := wfRow.Scan(&tableDBName, &formID); err != nil {
		return nil, fmt.Errorf("查询流程定义失败: %w", err)
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

	// 查询各明细表数据
	var detailData []map[string]interface{}
	for i := 1; i <= int(detailCount); i++ {
		dtTableName := a.tableName(fmt.Sprintf("%s_dt%d", tableDBName, i))
		var rows []map[string]interface{}

		// 统一使用 EXISTS 子查询，兼容 MySQL / Oracle / DM
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

		detailData = append(detailData, rows...)
	}

	return &ProcessData{
		ProcessID:  processID,
		MainData:   mainData,
		DetailData: detailData,
	}, nil
}

// ── FetchTodoList ──────────────────────────────────────────

// FetchTodoList 拉取用户在泛微 E9 中的待审批流程列表。
// 查询 workflow_currentoperator 获取用户待办，关联 workflow_requestbase 获取流程信息。
// 兼容 MySQL / Oracle / DM 三种驱动。
func (a *Ecology9Adapter) FetchTodoList(ctx context.Context, username string) ([]TodoItem, error) {
	var e9UserID int
	err := a.db.WithContext(ctx).
		Table(a.tableName("hrmresource")).
		Select(a.col("id")).
		Where(a.col("loginid")+" = ?", username).
		Row().Scan(&e9UserID)
	if err != nil {
		return nil, fmt.Errorf("OA 用户 '%s' 不存在", username)
	}

	// 查询待办：workflow_currentoperator + workflow_requestbase + workflow_base
	query := fmt.Sprintf(`
		SELECT
			r.%s AS request_id,
			r.%s AS request_name,
			COALESCE(h.%s, '') AS applicant_name,
			COALESCE(d.%s, '') AS dept_name,
			COALESCE(wb.%s, '') AS workflow_name,
			COALESCE(wt.%s, '') AS type_name,
			COALESCE(n.%s, '') AS node_name,
			r.%s AS create_date
		FROM %s co
		JOIN %s r ON co.%s = r.%s
		LEFT JOIN %s wb ON r.%s = wb.%s
		LEFT JOIN %s wt ON wb.%s = wt.%s
		LEFT JOIN %s h ON r.%s = h.%s
		LEFT JOIN %s d ON h.%s = d.%s
		LEFT JOIN %s n ON co.%s = n.%s
		WHERE co.%s = ? AND co.%s = 0
		ORDER BY r.%s DESC`,
		a.col("requestid"), a.col("requestname"),
		a.col("lastname"), a.col("departmentname"),
		a.col("workflowname"), a.col("typename"),
		a.col("nodename"),
		a.col("createdate"),
		a.tableName("workflow_currentoperator"), // co
		a.tableName("workflow_requestbase"),      // r
		a.col("requestid"), a.col("requestid"),
		a.tableName("workflow_base"),  // wb
		a.col("workflowid"), a.col("id"),
		a.tableName("workflow_type"),  // wt
		a.col("workflowtype"), a.col("id"),
		a.tableName("hrmresource"),    // h (applicant)
		a.col("creater"), a.col("id"),
		a.tableName("hrmdepartment"),  // d
		a.col("departmentid"), a.col("id"),
		a.tableName("workflow_nodebase"), // n
		a.col("nownodeid"), a.col("id"),
		a.col("userid"), a.col("isremark"),
		a.col("createdate"),
	)

	rows, err := a.db.WithContext(ctx).Raw(query, e9UserID).Rows()
	if err != nil {
		return nil, fmt.Errorf("查询 OA 待办失败: %w", err)
	}
	defer rows.Close()

	var items []TodoItem
	for rows.Next() {
		var requestID, requestName, applicant, department, workflowName, typeName, nodeName, createDate string
		if err := rows.Scan(&requestID, &requestName, &applicant, &department, &workflowName, &typeName, &nodeName, &createDate); err != nil {
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
		})
	}
	return items, nil
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
