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
// 1. 查询 workflow_base，确认流程存在且 isvalid=1
// 2. 通过 formid 关联 workflow_bill 获取真实主表名
//
// 使用 Row().Scan() 显式扫描列值，避免 GORM struct tag 大小写映射问题（Oracle/DM 列名大写）。
func (a *Ecology9Adapter) ValidateProcess(ctx context.Context, processType string) (*ProcessInfo, error) {
	// 查询 workflow_base：获取流程名称和 formid
	var workflowName string
	var formID int
	row := a.db.WithContext(ctx).
		Table(a.tableName("workflow_base")).
		Select(a.col("workflowname")+", "+a.col("formid")).
		Where(a.col("workflowname")+" = ? AND "+a.col("isvalid")+" = ?", processType, "1").
		Row()
	if err := row.Scan(&workflowName, &formID); err != nil {
		return nil, fmt.Errorf("流程 '%s' 在泛微 E9 系统中不存在或已停用", processType)
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
		ProcessType: processType,
		ProcessName: workflowName,
		MainTable:   mainTable,
	}, nil
}

// ── FetchFields ────────────────────────────────────────────

// FetchFields 从泛微 E9 拉取指定流程的全部字段定义。
func (a *Ecology9Adapter) FetchFields(ctx context.Context, processType string) (*ProcessFields, error) {
	// 显式扫描 formid 和 tablename，避免 struct tag 大小写映射问题
	var formID int
	var mainTableName string
	row := a.db.WithContext(ctx).
		Table(a.tableName("workflow_base")).
		Select(a.col("formid")+", "+a.col("tablename")).
		Where(a.col("workflowname")+" = ?", processType).
		Row()
	if err := row.Scan(&formID, &mainTableName); err != nil {
		return nil, fmt.Errorf("查询流程 '%s' 失败: %w", processType, err)
	}

	var rawFields []map[string]interface{}
	err := a.db.WithContext(ctx).
		Table(a.tableName("workflow_billfield")).
		Select(a.col("fielddbname")+", "+a.col("fieldname")+", "+a.col("fieldhtmltype")+", "+a.col("detailtable")).
		Where(a.col("billid")+" = ?", formID).
		Order(a.col("detailtable") + " ASC, " + a.col("id") + " ASC").
		Find(&rawFields).Error
	if err != nil {
		return nil, fmt.Errorf("查询流程字段失败: %w", err)
	}

	result := &ProcessFields{
		MainFields:   make([]FieldDef, 0),
		DetailTables: make([]DetailTableDef, 0),
	}
	detailMap := make(map[int]*DetailTableDef)

	for _, row := range rawFields {
		fd := FieldDef{
			FieldKey:  mapGet(row, "fielddbname"),
			FieldName: mapGet(row, "fieldname"),
			FieldType: a.mapFieldType(mapGet(row, "fieldhtmltype")),
		}
		dt := mapGetInt(row, "detailtable")
		if dt == 0 {
			result.MainFields = append(result.MainFields, fd)
		} else {
			dtDef, exists := detailMap[dt]
			if !exists {
				dtDef = &DetailTableDef{
					TableName:  fmt.Sprintf("%s_dt%d", mainTableName, dt),
					TableLabel: fmt.Sprintf("明细表%d", dt),
					Fields:     make([]FieldDef, 0),
				}
				detailMap[dt] = dtDef
			}
			dtDef.Fields = append(dtDef.Fields, fd)
		}
	}
	for i := 1; i <= len(detailMap); i++ {
		if dt, ok := detailMap[i]; ok {
			result.DetailTables = append(result.DetailTables, *dt)
		}
	}
	return result, nil
}

// ── CheckUserPermission ────────────────────────────────────

// CheckUserPermission 检查用户在泛微 E9 中是否具有指定流程的审批权限。
func (a *Ecology9Adapter) CheckUserPermission(ctx context.Context, userID string, processType string) (bool, error) {
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

	var count int64
	err := a.db.WithContext(ctx).
		Table(a.tableName("workflow_currentoperator")).
		Where(a.col("workflowid")+" = ? AND "+a.col("userid")+" = ?", workflowID, userID).
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

// ── mapFieldType ───────────────────────────────────────────

// mapFieldType 将泛微 E9 的字段 HTML 类型映射为通用字段类型。
func (a *Ecology9Adapter) mapFieldType(htmlType string) string {
	switch htmlType {
	case "1":
		return "text"
	case "2":
		return "textarea"
	case "3":
		return "select"
	case "4":
		return "checkbox"
	case "5":
		return "date"
	case "6":
		return "number"
	case "7":
		return "attachment"
	default:
		return "text"
	}
}
