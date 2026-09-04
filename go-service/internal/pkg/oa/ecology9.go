package oa

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"auraoa/go-service/internal/model"
	pkglogger "auraoa/go-service/internal/pkg/logger"
	"auraoa/go-service/internal/pkg/oa/dm"
	"auraoa/go-service/internal/pkg/oa/oracle"
)

// Ecology9Adapter 泛微 E9 OA 系统适配器。
// 支持 MySQL、Oracle 和 DM（达梦）三种底层数据库驱动。
type Ecology9Adapter struct {
	db                       *gorm.DB
	driver                   string                       // "mysql" | "oracle" | "dm"
	attachmentRecognitionSvc AttachmentRecognitionService // 附件识别服务接口
	weaverAPIURL             string
	weaverAppID              string
	weaverLoginID            string
	httpClient               *http.Client
}

// AttachmentRecognitionService 附件识别服务接口（避免循环依赖）。
//
// adapter 层负责从 OA 拉取附件原始载荷，识别服务按配置执行代码解析或 MinerU 解析。
type AttachmentRecognitionService interface {
	RecognizeAttachments(ctx context.Context, files []AttachmentFilePayload, fieldKey string, fieldName string) ([]AttachmentInfo, error)
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

// openEcology9Database 创建并验证泛微 E9 底层数据库连接池。
// transient 为 true 时不保留空闲连接，用于连接测试等短生命周期场景。
func openEcology9Database(
	ctx context.Context,
	conn *model.OADatabaseConnection,
	transient bool,
) (*gorm.DB, *sql.DB, error) {
	if conn == nil {
		return nil, nil, fmt.Errorf("OA 数据库连接配置为空")
	}

	poolSize := normalizePoolSize(conn.PoolSize)
	connectionTimeout := normalizeConnectionTimeout(conn.ConnectionTimeout)

	var (
		dialector gorm.Dialector
		sqlDB     *sql.DB
		err       error
	)
	switch conn.Driver {
	case "mysql":
		dsn := fmt.Sprintf(
			"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=%s",
			conn.Username,
			conn.Password,
			conn.Host,
			conn.Port,
			conn.DatabaseName,
			connectionTimeout,
		)
		sqlDB, err = sql.Open(mysql.DefaultDriverName, dsn)
		if err == nil {
			dialector = mysql.New(mysql.Config{DSN: dsn, Conn: sqlDB})
		}
	case "oracle":
		dsn := oracle.BuildDSN(
			conn.Username,
			conn.Password,
			conn.Host,
			conn.Port,
			conn.DatabaseName,
			int(connectionTimeout/time.Second),
		)
		sqlDB, err = sql.Open("oracle", dsn)
		if err == nil {
			dialector = oracle.OpenWithConn(dsn, sqlDB)
		}
	case "dm":
		dsn := dm.BuildDSN(
			conn.Username,
			conn.Password,
			conn.Host,
			conn.Port,
			conn.DatabaseName,
			int(connectionTimeout/time.Second),
		)
		sqlDB, err = sql.Open("dm", dsn)
		if err == nil {
			dialector = dm.OpenWithConn(dsn, sqlDB)
		}
	default:
		return nil, nil, fmt.Errorf("泛微 E9 不支持数据库驱动: %s（仅支持 mysql、oracle、dm）", conn.Driver)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("创建泛微 E9 数据库连接池失败 (driver=%s): %w", conn.Driver, err)
	}

	// 必须在 GORM 初始化前设置连接池限制。MySQL 和 Oracle 驱动会在
	// Initialize 阶段查询数据库版本，晚于 gorm.Open 设置会留下无界连接窗口。
	sqlDB.SetMaxOpenConns(poolSize)
	if transient {
		sqlDB.SetMaxIdleConns(0)
	} else {
		sqlDB.SetMaxIdleConns(normalizeMaxIdleConns(poolSize))
	}
	sqlDB.SetConnMaxIdleTime(oaConnectionMaxIdleTime)
	sqlDB.SetConnMaxLifetime(oaConnectionMaxLifetime)

	// Oracle/DM 默认将不加引号的标识符转为大写，
	// 泛微 E9 在 Oracle/DM 上的表名和列名均为大写。
	// 配置 NamingStrategy 使 GORM 不自动添加引号、不转小写。
	gormConfig := &gorm.Config{
		// 使用与主库相同的 zap logger，OA 慢查询也写入 app.log
		Logger:               pkglogger.NewGormLogger(200*time.Millisecond, true),
		DisableAutomaticPing: true,
	}
	if conn.Driver == "oracle" || conn.Driver == "dm" {
		gormConfig.NamingStrategy = schema.NamingStrategy{
			NoLowerCase: true,
		}
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		_ = sqlDB.Close()
		return nil, nil, fmt.Errorf("初始化泛微 E9 数据库驱动失败 (driver=%s): %w", conn.Driver, err)
	}

	pingCtx := ctx
	if pingCtx == nil {
		pingCtx = context.Background()
	}
	pingCtx, cancel := context.WithTimeout(pingCtx, connectionTimeout)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		_ = sqlDB.Close()
		return nil, nil, fmt.Errorf("连接泛微 E9 数据库失败 (driver=%s): %w", conn.Driver, err)
	}

	return db, sqlDB, nil
}

// newEcology9AdapterWithDB 使用已建立的共享连接池创建轻量适配器。
func newEcology9AdapterWithDB(
	conn *model.OADatabaseConnection,
	db *gorm.DB,
	attachmentSvc AttachmentRecognitionService,
) *Ecology9Adapter {
	return &Ecology9Adapter{
		db:                       db,
		driver:                   conn.Driver,
		attachmentRecognitionSvc: attachmentSvc,
		weaverAPIURL:             strings.TrimSpace(conn.WeaverAPIURL),
		weaverAppID:              strings.TrimSpace(conn.WeaverAppID),
		weaverLoginID:            strings.TrimSpace(conn.WeaverDefaultUser),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
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
			if b, ok := v.([]byte); ok {
				return string(b)
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
			case string:
				var out int
				_, _ = fmt.Sscanf(strings.TrimSpace(n), "%d", &out)
				return out
			case []byte:
				var out int
				_, _ = fmt.Sscanf(strings.TrimSpace(string(n)), "%d", &out)
				return out
			}
		}
	}
	return 0
}

// mapFindKey 在 OA 原始行中按字段名做大小写不敏感匹配，兼容 Oracle/DM 返回大写列名。
func mapFindKey(m map[string]interface{}, key string) (string, bool) {
	for k := range m {
		if strings.EqualFold(k, key) {
			return k, true
		}
	}
	return "", false
}

func stringifyDBValue(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case []byte:
		return strings.TrimSpace(string(t))
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
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
	var workflowID int
	var workflowName string
	var formID int
	var workflowTypeID int
	row := a.db.WithContext(ctx).
		Table(a.tableName("workflow_base")).
		Select(a.col("id")+", "+a.col("workflowname")+", "+a.col("formid")+", "+a.col("workflowtype")).
		Where(a.col("workflowname")+" = ? AND "+a.col("isvalid")+" = ?", processType, "1").
		Row()
	if err := row.Scan(&workflowID, &workflowName, &formID, &workflowTypeID); err != nil {
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
		WorkflowID:       fmt.Sprintf("%d", workflowID),
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

	return a.fetchFieldsByForm(ctx, formID, mainTableName)
}

// FetchFieldsByWorkflowID 按 OA workflow_base.id 拉取流程字段，避免同名流程取错表单。
func (a *Ecology9Adapter) FetchFieldsByWorkflowID(ctx context.Context, workflowID string) (*ProcessFields, error) {
	workflowID = strings.TrimSpace(workflowID)
	if workflowID == "" {
		return nil, fmt.Errorf("流程 ID 为空")
	}
	var formID int
	row := a.db.WithContext(ctx).
		Table(a.tableName("workflow_base")).
		Select(a.col("formid")).
		Where(a.col("id")+" = ? AND "+a.col("isvalid")+" = ?", workflowID, "1").
		Row()
	if err := row.Scan(&formID); err != nil {
		return nil, fmt.Errorf("查询流程定义失败 (workflowid=%s): %w", workflowID, err)
	}
	var mainTableName string
	billRow := a.db.WithContext(ctx).
		Table(a.tableName("workflow_bill")).
		Select(a.col("tablename")).
		Where(a.col("id")+" = ?", formID).
		Row()
	if err := billRow.Scan(&mainTableName); err != nil {
		return nil, fmt.Errorf("查询流程表单定义失败 (formid=%d): %w", formID, err)
	}
	return a.fetchFieldsByForm(ctx, formID, mainTableName)
}

// SearchWorkflowDefinitions 检索 OA 流程定义，供外部关联选择目标流程。
// 使用 Rows().Scan 显式扫描，避免 Oracle/DM 下 GORM struct tag 大小写映射导致名称/ID 全空。
func (a *Ecology9Adapter) SearchWorkflowDefinitions(ctx context.Context, keyword string) ([]ProcessInfo, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return []ProcessInfo{}, nil
	}

	selectSQL := strings.Join([]string{
		"wb." + a.col("id"),
		"COALESCE(wb." + a.col("workflowname") + ", '')",
		"COALESCE(wt." + a.col("typename") + ", '')",
		"COALESCE(b." + a.col("tablename") + ", '')",
	}, ", ")
	query := a.db.WithContext(ctx).
		Table(a.tableName("workflow_base")+" wb").
		Select(selectSQL).
		Joins("LEFT JOIN "+a.tableName("workflow_type")+" wt ON wb."+a.col("workflowtype")+" = wt."+a.col("id")).
		Joins("LEFT JOIN "+a.tableName("workflow_bill")+" b ON wb."+a.col("formid")+" = b."+a.col("id")).
		Where("wb."+a.col("isvalid")+" = ?", "1")
	like := "%" + keyword + "%"
	query = query.Where(
		"wb."+a.col("workflowname")+" LIKE ? OR wt."+a.col("typename")+" LIKE ? OR b."+a.col("tablename")+" LIKE ?",
		like, like, like,
	).Order("wb." + a.col("workflowname") + " ASC").Limit(30)

	rows, err := query.Rows()
	if err != nil {
		return nil, fmt.Errorf("检索流程定义失败: %w", err)
	}
	defer rows.Close()

	out := make([]ProcessInfo, 0, 30)
	for rows.Next() {
		var workflowID int
		var workflowName, typeName, mainTable string
		if err := rows.Scan(&workflowID, &workflowName, &typeName, &mainTable); err != nil {
			return nil, fmt.Errorf("解析流程定义失败: %w", err)
		}
		out = append(out, ProcessInfo{
			WorkflowID:       fmt.Sprintf("%d", workflowID),
			ProcessType:      workflowName,
			ProcessName:      workflowName,
			ProcessTypeLabel: typeName,
			MainTable:        mainTable,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("检索流程定义失败: %w", err)
	}
	return out, nil
}

func (a *Ecology9Adapter) fetchFieldsByForm(ctx context.Context, formID int, mainTableName string) (*ProcessFields, error) {
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
//
// 当注入了 attachmentRecognitionSvc 时，会顺带识别主表中的附件字段（fieldhtmltype=6），
// 把字段值（逗号分隔的 docId 列表）交给附件识别服务，结果填入 ProcessData.Attachments。
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

	// 查询各明细表数据（detailtable 在 E9 中为明细表物理表名或序号，不能按数字比较）
	var billFieldRows []map[string]interface{}
	if err := a.db.WithContext(ctx).
		Table(a.tableName("workflow_billfield")).
		Select(a.col("detailtable")+" AS detailtable").
		Where(a.col("billid")+" = ?", formID).
		Find(&billFieldRows).Error; err != nil {
		pkglogger.Global().Warn("查询明细表字段定义失败，跳过明细数据",
			zap.String("processID", processID),
			zap.Int("formID", formID),
			zap.Error(err))
		billFieldRows = nil
	}

	detailTables := make(map[string][]map[string]interface{})
	seenDetailTable := make(map[string]struct{})
	for _, row := range billFieldRows {
		dt := strings.TrimSpace(mapGet(row, "detailtable"))
		if dt == "" || dt == "0" || strings.EqualFold(dt, "主表") || strings.EqualFold(dt, tableDBName) {
			continue
		}
		// 部分版本仅存序号（如 "1"），拼成 formtable_main_x_dt1
		if len(dt) < 3 && !strings.Contains(strings.ToLower(dt), "dt") {
			dt = fmt.Sprintf("%s_dt%s", tableDBName, dt)
		}
		if _, exists := seenDetailTable[dt]; exists {
			continue
		}
		seenDetailTable[dt] = struct{}{}

		dtTableName := a.tableName(dt)
		var rows []map[string]interface{}
		subQuery := fmt.Sprintf(
			"EXISTS (SELECT 1 FROM %s m WHERE m.%s = %s.%s AND m.%s = ?)",
			mainTableName,
			a.col("id"), dtTableName, a.col("mainid"),
			a.col("requestid"),
		)
		if err := a.db.WithContext(ctx).
			Table(dtTableName).
			Where(subQuery, processID).
			Find(&rows).Error; err != nil {
			pkglogger.Global().Warn("查询明细表数据失败，跳过该表",
				zap.String("processID", processID),
				zap.String("detailTable", dt),
				zap.Error(err))
			continue
		}
		if len(rows) > 0 {
			detailTables[dt] = rows
		}
	}

	pd := &ProcessData{
		ProcessID:    processID,
		MainData:     mainData,
		DetailTables: detailTables,
		FieldLabels:  a.fetchFieldLabels(ctx, formID, tableDBName),
	}

	// 识别主表附件字段并提取内容（仅当注入了识别服务）
	if a.attachmentRecognitionSvc == nil {
		pkglogger.Global().Debug("附件识别：未注入识别服务，跳过",
			zap.String("processID", processID))
	} else if len(mainData) == 0 {
		pkglogger.Global().Debug("附件识别：主表无数据，跳过",
			zap.String("processID", processID))
	} else {
		pkglogger.Global().Info("附件识别：开始处理主表附件",
			zap.String("processID", processID),
			zap.Int("formID", formID),
			zap.Bool("weaverApiConfigured", a.weaverAPIURL != ""),
			zap.Bool("weaverAppidConfigured", a.weaverAppID != ""),
			zap.Bool("weaverLoginConfigured", a.weaverLoginID != ""))
		attachments, attachErr := a.recognizeMainAttachments(ctx, processID, formID, mainData)
		if attachErr != nil {
			pkglogger.Global().Warn("附件识别：整体失败，跳过附件内容",
				zap.String("processID", processID),
				zap.Error(attachErr))
		} else {
			pd.Attachments = attachments
			pkglogger.Global().Info("附件识别：主表处理完成",
				zap.String("processID", processID),
				zap.Int("attachmentCount", len(attachments)))
		}
	}

	return pd, nil
}

// QueryModelContext 对 E9 建模表执行受限只读查询。
// 这里会校验表名/字段名；自定义 SQL 仅允许单条 SELECT，且必须使用指定变量。
func (a *Ecology9Adapter) QueryModelContext(ctx context.Context, query ModelContextQuery) (*ModelContextQueryResult, error) {
	mode := strings.TrimSpace(query.Mode)
	if mode == "" {
		mode = "exists"
	}
	maxRows := query.MaxRows
	if maxRows == 0 {
		maxRows = 5
	}
	if maxRows > 50 {
		maxRows = 50
	}
	switch mode {
	case "exists", "count", "rows":
		return a.queryModelTableContext(ctx, query, mode, maxRows)
	case "custom_sql":
		return a.queryModelCustomSQLContext(ctx, query, maxRows)
	default:
		return nil, fmt.Errorf("不支持的建模查询方式: %s", mode)
	}
}

// FetchModelTableMetadata 查询建模表中文名称及主表字段显示名，查不到时由调用方回退为物理名称。
func (a *Ecology9Adapter) FetchModelTableMetadata(ctx context.Context, tableName string) (*ModelTableMetadata, error) {
	tableName = strings.TrimSpace(tableName)
	if !isSafeIdentifier(tableName) {
		return nil, fmt.Errorf("建模表名不合法")
	}
	var formID int
	var displayName string
	row := a.db.WithContext(ctx).
		Table(a.tableName("workflow_bill")+" wb").
		Select("wb."+a.col("id")+", COALESCE(mi."+a.col("modename")+", '')").
		Joins("LEFT JOIN "+a.tableName("modeinfo")+" mi ON mi."+a.col("formid")+" = wb."+a.col("id")).
		Where("LOWER(wb."+a.col("tablename")+") = LOWER(?)", tableName).
		Row()
	if err := row.Scan(&formID, &displayName); err != nil {
		return nil, fmt.Errorf("查询建模表字段定义失败: %w", err)
	}
	labelsByTable := a.fetchFieldLabels(ctx, formID, tableName)
	return &ModelTableMetadata{
		TableName:   tableName,
		DisplayName: strings.TrimSpace(displayName),
		FieldLabels: labelsByTable["main"],
	}, nil
}

func (a *Ecology9Adapter) queryModelTableContext(ctx context.Context, query ModelContextQuery, mode string, maxRows int) (*ModelContextQueryResult, error) {
	tableName := strings.TrimSpace(query.TableName)
	joinField := strings.TrimSpace(query.JoinField)
	if joinField == "" {
		joinField = "id"
	}
	if !isSafeIdentifier(tableName) || !isSafeIdentifier(joinField) {
		return nil, fmt.Errorf("建模表名或关联字段不合法")
	}
	db := a.db.WithContext(ctx).Table(a.tableName(tableName)).Where(a.col(joinField)+" = ?", query.SourceValue)
	switch mode {
	case "exists":
		var count int64
		if err := db.Count(&count).Error; err != nil {
			return nil, fmt.Errorf("查询建模记录失败: %w", err)
		}
		return &ModelContextQueryResult{Mode: mode, Exists: count > 0, Count: count}, nil
	case "count":
		var count int64
		if err := db.Count(&count).Error; err != nil {
			return nil, fmt.Errorf("统计建模记录失败: %w", err)
		}
		return &ModelContextQueryResult{Mode: mode, Count: count}, nil
	default:
		selectExpr := "*"
		if len(query.ReturnFields) > 0 {
			fields := make([]string, 0, len(query.ReturnFields))
			for _, field := range query.ReturnFields {
				field = strings.TrimSpace(field)
				if !isSafeIdentifier(field) {
					return nil, fmt.Errorf("返回字段不合法: %s", field)
				}
				fields = append(fields, a.col(field))
			}
			selectExpr = strings.Join(fields, ", ")
		}
		if orderBy := strings.TrimSpace(query.OrderBy); orderBy != "" {
			if !isSafeIdentifier(orderBy) {
				return nil, fmt.Errorf("排序字段不合法: %s", orderBy)
			}
			dir := strings.ToUpper(strings.TrimSpace(query.OrderDir))
			if dir != "ASC" && dir != "DESC" {
				dir = "ASC"
			}
			db = db.Order(a.col(orderBy) + " " + dir)
		}
		var rows []map[string]interface{}
		if maxRows > 0 {
			db = db.Limit(maxRows)
		}
		if err := db.Select(selectExpr).Find(&rows).Error; err != nil {
			return nil, fmt.Errorf("查询建模行数据失败: %w", err)
		}
		return &ModelContextQueryResult{Mode: mode, Rows: rows}, nil
	}
}

func (a *Ecology9Adapter) queryModelCustomSQLContext(ctx context.Context, query ModelContextQuery, maxRows int) (*ModelContextQueryResult, error) {
	sqlText := strings.TrimSpace(query.CustomSQL)
	tableName := strings.TrimSpace(query.TableName)
	joinField := strings.TrimSpace(query.JoinField)
	if joinField == "" {
		joinField = "id"
	}
	if !isSafeIdentifier(tableName) || !isSafeIdentifier(joinField) {
		return nil, fmt.Errorf("建模表名或关联字段不合法")
	}
	if !strings.Contains(sqlText, "{{table_name}}") || !strings.Contains(sqlText, "{{join_field}}") {
		return nil, fmt.Errorf("自定义 SQL 必须使用 {{table_name}} 和 {{join_field}} 变量")
	}
	sqlText = strings.ReplaceAll(sqlText, "{{table_name}}", a.tableName(tableName))
	sqlText = strings.ReplaceAll(sqlText, "{{join_field}}", a.col(joinField))
	if !isReadonlySingleSelect(sqlText) {
		return nil, fmt.Errorf("自定义 SQL 仅允许单条 SELECT 查询")
	}
	if !strings.Contains(sqlText, ":source_value") {
		return nil, fmt.Errorf("自定义 SQL 必须使用 :source_value 参数")
	}
	if maxRows > 0 && !hasSQLLimit(sqlText) {
		sqlText += a.limitOffsetClause(maxRows, 0)
	}
	var rows []map[string]interface{}
	if err := a.db.WithContext(ctx).Raw(sqlText, sql.Named("source_value", query.SourceValue)).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("执行自定义 SQL 失败: %w", err)
	}
	if maxRows > 0 && len(rows) > maxRows {
		rows = rows[:maxRows]
	}
	return &ModelContextQueryResult{Mode: "custom_sql", Rows: rows}, nil
}

func isReadonlySingleSelect(sqlText string) bool {
	trimmed := strings.TrimSpace(sqlText)
	if trimmed == "" {
		return false
	}
	lower := strings.ToLower(trimmed)
	if !strings.HasPrefix(lower, "select ") {
		return false
	}
	if strings.Contains(lower, ";") || strings.Contains(lower, "--") || strings.Contains(lower, "/*") || strings.Contains(lower, "*/") {
		return false
	}
	blocked := []string{" insert ", " update ", " delete ", " drop ", " alter ", " truncate ", " create ", " call ", " exec ", " merge "}
	padded := " " + lower + " "
	for _, token := range blocked {
		if strings.Contains(padded, token) {
			return false
		}
	}
	return true
}

func hasSQLLimit(sqlText string) bool {
	lower := strings.ToLower(sqlText)
	return strings.Contains(lower, " limit ") || strings.Contains(lower, " fetch next ")
}

func (a *Ecology9Adapter) fetchFieldLabels(ctx context.Context, formID int, mainTable string) map[string]map[string]string {
	labels := map[string]map[string]string{"main": {}}
	var rawFields []map[string]interface{}
	err := a.db.WithContext(ctx).
		Table(a.tableName("workflow_billfield")+" "+a.col("t1")).
		Select(a.col("t1.fieldname")+" AS fieldkey, "+a.col("t2.labelname")+" AS fieldname, "+a.col("t1.detailtable")+" AS detailtable").
		Joins("JOIN "+a.tableName("htmllabelinfo")+" "+a.col("t2")+" ON "+a.col("t1.fieldlabel")+" = "+a.col("t2.indexid")).
		Where(a.col("t1.billid")+" = ? AND "+a.col("t2.languageid")+" = 7", formID).
		Find(&rawFields).Error
	if err != nil {
		pkglogger.Global().Warn("查询流程字段中文标签失败，AI prompt 将使用数据库字段名",
			zap.Int("formID", formID),
			zap.Error(err))
		return labels
	}
	for _, row := range rawFields {
		fieldKey := strings.TrimSpace(mapGet(row, "fieldkey"))
		fieldName := strings.TrimSpace(mapGet(row, "fieldname"))
		if fieldKey == "" || fieldName == "" {
			continue
		}
		tableKey := normalizeDetailTableKey(mainTable, strings.TrimSpace(mapGet(row, "detailtable")))
		if _, ok := labels[tableKey]; !ok {
			labels[tableKey] = map[string]string{}
		}
		labels[tableKey][fieldKey] = fieldName
		labels[tableKey][strings.ToLower(fieldKey)] = fieldName
	}
	return labels
}

func normalizeDetailTableKey(mainTable, detailTable string) string {
	dt := strings.TrimSpace(detailTable)
	if dt == "" || dt == "0" || strings.EqualFold(dt, "主表") || strings.EqualFold(dt, mainTable) {
		return "main"
	}
	if len(dt) < 3 && !strings.Contains(strings.ToLower(dt), "dt") {
		dt = fmt.Sprintf("%s_dt%s", mainTable, dt)
	}
	return dt
}

type e9BrowseField struct {
	FieldKey    string
	DetailTable string
	Type        int
	FieldDBType string
}

type e9ChoiceField struct {
	FieldID     int
	FieldKey    string
	DetailTable string
}

type e9BrowseTarget struct {
	Table             string
	IDColumn          string
	DisplayColumn     string
	DisplayExpression bool
	Multiple          bool
	Source            string
	NumericID         bool
}

type e9BrowserURLDef struct {
	ID          int
	BrowserName string
	BrowserURL  string
	FieldDBType string
	TableName   string
	ColumnName  string
	KeyColumn   string
}

type e9ModeBrowserDef struct {
	ID         int
	ShowName   string
	Name       string
	SQLText    string
	SearchByID string
	SQLText1   string
}

// ResolveBrowseDisplayValues 仅解析字段选择集会发送给 AI 的浏览按钮字段。
func (a *Ecology9Adapter) ResolveBrowseDisplayValues(ctx context.Context, processID string, pd *ProcessData, fieldSet map[string]map[string]bool) error {
	if pd == nil {
		return nil
	}
	var workflowID int
	reqRow := a.db.WithContext(ctx).
		Table(a.tableName("workflow_requestbase")).
		Select(a.col("workflowid")).
		Where(a.col("requestid")+" = ?", processID).
		Row()
	if err := reqRow.Scan(&workflowID); err != nil {
		return fmt.Errorf("查询流程实例失败: %w", err)
	}

	var formID int
	wfRow := a.db.WithContext(ctx).
		Table(a.tableName("workflow_base")).
		Select(a.col("formid")).
		Where(a.col("id")+" = ?", workflowID).
		Row()
	if err := wfRow.Scan(&formID); err != nil {
		return fmt.Errorf("查询流程定义失败: %w", err)
	}

	var tableDBName string
	billRow := a.db.WithContext(ctx).
		Table(a.tableName("workflow_bill")).
		Select(a.col("tablename")).
		Where(a.col("id")+" = ?", formID).
		Row()
	if err := billRow.Scan(&tableDBName); err != nil {
		return fmt.Errorf("查询流程表单定义失败 (formid=%d): %w", formID, err)
	}

	if err := a.resolveBrowseDisplayValuesForFields(ctx, processID, formID, tableDBName, pd, fieldSet); err != nil {
		return err
	}
	return a.resolveChoiceDisplayValuesForFields(ctx, processID, formID, tableDBName, pd, fieldSet)
}

func (a *Ecology9Adapter) resolveBrowseDisplayValuesForFields(ctx context.Context, processID string, formID int, mainTable string, pd *ProcessData, fieldSet map[string]map[string]bool) error {
	if pd == nil {
		return nil
	}
	fields, err := a.fetchBrowseFields(ctx, formID)
	if err != nil {
		pkglogger.Global().Warn("浏览按钮解析：查询字段定义失败，保留原始值",
			zap.String("processID", processID),
			zap.Int("formID", formID),
			zap.Error(err))
		return err
	}
	if len(fields) == 0 {
		return nil
	}

	targetsByKey := map[string]e9BrowseTarget{}
	valuesByKey := map[string]map[string]struct{}{}
	type fieldBinding struct {
		field  e9BrowseField
		target e9BrowseTarget
		row    map[string]interface{}
		key    string
	}
	var bindings []fieldBinding

	for _, field := range fields {
		if !browseFieldSelected(mainTable, field, fieldSet) {
			continue
		}
		target, ok := a.resolveBrowseTarget(ctx, field)
		if !ok {
			continue
		}
		targetKey := target.cacheKey()
		targetsByKey[targetKey] = target
		if _, ok := valuesByKey[targetKey]; !ok {
			valuesByKey[targetKey] = map[string]struct{}{}
		}

		rows := browseRowsForField(pd, mainTable, field)
		for _, row := range rows {
			actualKey, ok := mapFindKey(row, field.FieldKey)
			if !ok {
				continue
			}
			raw := stringifyDBValue(row[actualKey])
			if raw == "" {
				continue
			}
			for _, id := range splitBrowseIDs(raw) {
				if !target.validID(id) {
					continue
				}
				valuesByKey[targetKey][id] = struct{}{}
			}
			bindings = append(bindings, fieldBinding{field: field, target: target, row: row, key: actualKey})
		}
	}

	if len(bindings) == 0 {
		return nil
	}

	displayByTarget := map[string]map[string]string{}
	for key, idSet := range valuesByKey {
		target := targetsByKey[key]
		ids := make([]string, 0, len(idSet))
		for id := range idSet {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		displayByTarget[key] = a.fetchBrowseDisplayMap(ctx, target, ids)
	}

	var resolvedCount int
	for _, binding := range bindings {
		raw := stringifyDBValue(binding.row[binding.key])
		if raw == "" {
			continue
		}
		displayMap := displayByTarget[binding.target.cacheKey()]
		resolved := buildBrowseResolvedValue(raw, binding.target.Multiple, displayMap)
		if resolved != nil {
			binding.row[binding.key] = resolved
			resolvedCount++
		}
	}
	if resolvedCount > 0 {
		pkglogger.Global().Debug("浏览按钮解析：已增补显示值",
			zap.String("processID", processID),
			zap.Int("fieldValueCount", resolvedCount))
	}
	return nil
}

func (a *Ecology9Adapter) resolveChoiceDisplayValuesForFields(ctx context.Context, processID string, formID int, mainTable string, pd *ProcessData, fieldSet map[string]map[string]bool) error {
	if pd == nil {
		return nil
	}
	fields, err := a.fetchChoiceFields(ctx, formID)
	if err != nil {
		pkglogger.Global().Warn("选择框解析：查询字段定义失败，保留原始值",
			zap.String("processID", processID),
			zap.Int("formID", formID),
			zap.Error(err))
		return nil
	}
	if len(fields) == 0 {
		return nil
	}

	selectedFields := make([]e9ChoiceField, 0, len(fields))
	fieldIDs := make([]int, 0, len(fields))
	for _, field := range fields {
		if !browseFieldSelected(mainTable, e9BrowseField{FieldKey: field.FieldKey, DetailTable: field.DetailTable}, fieldSet) {
			continue
		}
		selectedFields = append(selectedFields, field)
		fieldIDs = append(fieldIDs, field.FieldID)
	}
	if len(selectedFields) == 0 {
		return nil
	}

	optionsByField := a.fetchChoiceOptionMaps(ctx, fieldIDs)
	if len(optionsByField) == 0 {
		return nil
	}

	var resolvedCount int
	for _, field := range selectedFields {
		optionMap := optionsByField[field.FieldID]
		if len(optionMap) == 0 {
			continue
		}
		rows := browseRowsForField(pd, mainTable, e9BrowseField{FieldKey: field.FieldKey, DetailTable: field.DetailTable})
		for _, row := range rows {
			actualKey, ok := mapFindKey(row, field.FieldKey)
			if !ok {
				continue
			}
			raw := stringifyDBValue(row[actualKey])
			if raw == "" {
				continue
			}
			if resolved := buildChoiceResolvedValue(raw, optionMap); resolved != nil {
				row[actualKey] = resolved
				resolvedCount++
			}
		}
	}
	if resolvedCount > 0 {
		pkglogger.Global().Debug("选择框解析：已增补显示值",
			zap.String("processID", processID),
			zap.Int("fieldValueCount", resolvedCount))
	}
	return nil
}

func (a *Ecology9Adapter) fetchBrowseFields(ctx context.Context, formID int) ([]e9BrowseField, error) {
	var rows []map[string]interface{}
	err := a.db.WithContext(ctx).
		Table(a.tableName("workflow_billfield")).
		Select(strings.Join([]string{
			a.col("fieldname") + " AS fieldkey",
			a.col("detailtable") + " AS detailtable",
			a.col("type") + " AS type",
			a.col("fielddbtype") + " AS fielddbtype",
		}, ", ")).
		Where(a.col("billid")+" = ? AND "+a.col("fieldhtmltype")+" = ?", formID, "3").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	fields := make([]e9BrowseField, 0, len(rows))
	for _, row := range rows {
		fieldKey := strings.TrimSpace(mapGet(row, "fieldkey"))
		if fieldKey == "" {
			continue
		}
		fields = append(fields, e9BrowseField{
			FieldKey:    fieldKey,
			DetailTable: strings.TrimSpace(mapGet(row, "detailtable")),
			Type:        mapGetInt(row, "type"),
			FieldDBType: strings.TrimSpace(mapGet(row, "fielddbtype")),
		})
	}
	return fields, nil
}

func (a *Ecology9Adapter) fetchChoiceFields(ctx context.Context, formID int) ([]e9ChoiceField, error) {
	var rows []map[string]interface{}
	err := a.db.WithContext(ctx).
		Table(a.tableName("workflow_billfield")).
		Select(strings.Join([]string{
			a.col("id") + " AS fieldid",
			a.col("fieldname") + " AS fieldkey",
			a.col("detailtable") + " AS detailtable",
		}, ", ")).
		Where(a.col("billid")+" = ? AND "+a.col("fieldhtmltype")+" = ?", formID, "5").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	fields := make([]e9ChoiceField, 0, len(rows))
	for _, row := range rows {
		fieldID := mapGetInt(row, "fieldid")
		fieldKey := strings.TrimSpace(mapGet(row, "fieldkey"))
		if fieldID == 0 || fieldKey == "" {
			continue
		}
		fields = append(fields, e9ChoiceField{
			FieldID:     fieldID,
			FieldKey:    fieldKey,
			DetailTable: strings.TrimSpace(mapGet(row, "detailtable")),
		})
	}
	return fields, nil
}

func (a *Ecology9Adapter) fetchChoiceOptionMaps(ctx context.Context, fieldIDs []int) map[int]map[string]string {
	result := map[int]map[string]string{}
	if len(fieldIDs) == 0 {
		return result
	}
	seen := map[int]struct{}{}
	ids := make([]int, 0, len(fieldIDs))
	for _, id := range fieldIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return result
	}

	var rows []map[string]interface{}
	err := a.db.WithContext(ctx).
		Table(a.tableName("workflow_selectitem")).
		Select(strings.Join([]string{
			a.col("fieldid") + " AS fieldid",
			a.col("selectvalue") + " AS option_value",
			a.col("selectname") + " AS option_display",
		}, ", ")).
		Where(a.col("fieldid")+" IN ?", ids).
		Find(&rows).Error
	if err != nil {
		pkglogger.Global().Warn("选择框解析：查询选项定义失败，保留原始值",
			zap.Error(err))
		return result
	}
	for _, row := range rows {
		fieldID := mapGetInt(row, "fieldid")
		value := strings.TrimSpace(mapGet(row, "option_value"))
		display := normalizeChoiceDisplayName(mapGet(row, "option_display"))
		if fieldID == 0 || value == "" || display == "" {
			continue
		}
		if _, ok := result[fieldID]; !ok {
			result[fieldID] = map[string]string{}
		}
		result[fieldID][value] = display
	}
	return result
}

func browseRowsForField(pd *ProcessData, mainTable string, field e9BrowseField) []map[string]interface{} {
	dt := strings.TrimSpace(field.DetailTable)
	if dt == "" || dt == "0" || strings.EqualFold(dt, "主表") || strings.EqualFold(dt, mainTable) {
		if len(pd.MainData) == 0 {
			return nil
		}
		return []map[string]interface{}{pd.MainData}
	}
	if len(dt) < 3 && !strings.Contains(strings.ToLower(dt), "dt") {
		dt = fmt.Sprintf("%s_dt%s", mainTable, dt)
	}
	for tableName, rows := range pd.DetailTables {
		if strings.EqualFold(tableName, dt) {
			return rows
		}
	}
	return nil
}

func browseFieldSelected(mainTable string, field e9BrowseField, fieldSet map[string]map[string]bool) bool {
	if fieldSet == nil {
		return true
	}
	tableKey := browseFieldTableKey(mainTable, field)
	allowedKeys, ok := fieldSet[tableKey]
	if !ok && tableKey != "main" {
		for key, value := range fieldSet {
			if strings.EqualFold(key, tableKey) {
				allowedKeys = value
				ok = true
				break
			}
		}
	}
	if !ok || allowedKeys == nil {
		return true
	}
	if len(allowedKeys) == 0 {
		return false
	}
	return allowedKeys[field.FieldKey] || allowedKeys[strings.ToLower(field.FieldKey)]
}

func browseFieldTableKey(mainTable string, field e9BrowseField) string {
	dt := strings.TrimSpace(field.DetailTable)
	if dt == "" || dt == "0" || strings.EqualFold(dt, "主表") || strings.EqualFold(dt, mainTable) {
		return "main"
	}
	if len(dt) < 3 && !strings.Contains(strings.ToLower(dt), "dt") {
		dt = fmt.Sprintf("%s_dt%s", mainTable, dt)
	}
	return dt
}

func (a *Ecology9Adapter) resolveBrowseTarget(ctx context.Context, field e9BrowseField) (e9BrowseTarget, bool) {
	if def, ok := a.fetchBrowserURLDefByType(ctx, field); ok {
		if target, ok := browseTargetFromBrowserURLDef(def); ok {
			target.Multiple = isMultipleBrowseField(field)
			target.Source = "workflow_browserurl"
			return target, true
		}
	}
	if target, ok := builtinBrowseTarget(field.Type); ok {
		return target, true
	}
	if isCustomBrowseType(field.Type) || strings.HasPrefix(strings.ToLower(field.FieldDBType), "browser.") {
		browserName := browserNameFromFieldDBType(field.FieldDBType)
		def, ok := a.fetchCustomBrowserURLDef(ctx, field)
		if !ok {
			if target, ok := a.resolveModeBrowserTarget(ctx, browserName); ok {
				target.Multiple = isMultipleBrowseField(field)
				target.Source = "mode_browser"
				return target, true
			}
			return e9BrowseTarget{}, false
		}
		if target, ok := browseTargetFromBrowserURLDef(def); ok {
			target.Multiple = isMultipleBrowseField(field)
			target.Source = "workflow_browserurl_custom"
			return target, true
		}
		if target, ok := a.resolveModeBrowserTarget(ctx, browserName); ok {
			target.Multiple = isMultipleBrowseField(field)
			target.Source = "mode_browser"
			return target, true
		}
		pkglogger.Global().Warn("浏览按钮解析：未找到可用显示值配置，保留原始值",
			zap.String("fieldKey", field.FieldKey),
			zap.String("fieldDBType", field.FieldDBType),
			zap.String("browserName", def.BrowserName))
		return e9BrowseTarget{}, false
	}
	return e9BrowseTarget{}, false
}

func (a *Ecology9Adapter) fetchBrowserURLDefByType(ctx context.Context, field e9BrowseField) (e9BrowserURLDef, bool) {
	if field.Type <= 0 {
		return e9BrowserURLDef{}, false
	}
	rows, err := a.queryBrowserURLDefs(ctx, a.col("id")+" = ?", field.Type, true)
	if err != nil || len(rows) == 0 {
		return e9BrowserURLDef{}, false
	}
	return browserURLDefFromRow(rows[0], browserNameFromFieldDBType(field.FieldDBType)), true
}

func isCustomBrowseType(typeID int) bool {
	switch typeID {
	case 161, 162, 226, 256, 257:
		return true
	default:
		return false
	}
}

func isMultipleBrowseField(field e9BrowseField) bool {
	if isMultipleCustomBrowseType(field.Type) {
		return true
	}
	dbType := strings.ToLower(strings.TrimSpace(field.FieldDBType))
	switch dbType {
	case "text":
		return true
	}
	switch field.Type {
	case 17, 37, 57, 135, 152, 166, 168, 170, 184, 194, 278, 293, 314, 315, 317, 321, 322:
		return true
	default:
		return false
	}
}

func isMultipleCustomBrowseType(typeID int) bool {
	switch typeID {
	case 162, 257:
		return true
	default:
		return false
	}
}

func builtinBrowseTarget(typeID int) (e9BrowseTarget, bool) {
	targets := map[int]e9BrowseTarget{
		1:   {Table: "hrmresource", IDColumn: "id", DisplayColumn: "lastname", Source: "hrm", NumericID: true},
		4:   {Table: "hrmdepartment", IDColumn: "id", DisplayColumn: "departmentname", Source: "department", NumericID: true},
		16:  {Table: "workflow_requestbase", IDColumn: "requestid", DisplayColumn: "requestname", Source: "workflow", NumericID: true},
		17:  {Table: "hrmresource", IDColumn: "id", DisplayColumn: "lastname", Multiple: true, Source: "hrm_multi", NumericID: true},
		57:  {Table: "hrmdepartment", IDColumn: "id", DisplayColumn: "departmentname", Multiple: true, Source: "department_multi", NumericID: true},
		152: {Table: "workflow_requestbase", IDColumn: "requestid", DisplayColumn: "requestname", Multiple: true, Source: "workflow_multi", NumericID: true},
		164: {Table: "hrmsubcompany", IDColumn: "id", DisplayColumn: "subcompanyname", Source: "subcompany", NumericID: true},
		165: {Table: "hrmresource", IDColumn: "id", DisplayColumn: "lastname", Source: "hrm_authorized", NumericID: true},
		166: {Table: "hrmresource", IDColumn: "id", DisplayColumn: "lastname", Multiple: true, Source: "hrm_authorized_multi", NumericID: true},
		167: {Table: "hrmdepartment", IDColumn: "id", DisplayColumn: "departmentname", Source: "department_authorized", NumericID: true},
		168: {Table: "hrmdepartment", IDColumn: "id", DisplayColumn: "departmentname", Multiple: true, Source: "department_authorized_multi", NumericID: true},
		169: {Table: "hrmsubcompany", IDColumn: "id", DisplayColumn: "subcompanyname", Source: "subcompany_authorized", NumericID: true},
		170: {Table: "hrmsubcompany", IDColumn: "id", DisplayColumn: "subcompanyname", Multiple: true, Source: "subcompany_authorized_multi", NumericID: true},
		171: {Table: "workflow_requestbase", IDColumn: "requestid", DisplayColumn: "requestname", Source: "workflow_archive", NumericID: true},
		194: {Table: "hrmsubcompany", IDColumn: "id", DisplayColumn: "subcompanyname", Multiple: true, Source: "subcompany_multi", NumericID: true},
	}
	target, ok := targets[typeID]
	return target, ok
}

func (a *Ecology9Adapter) fetchCustomBrowserURLDef(ctx context.Context, field e9BrowseField) (e9BrowserURLDef, bool) {
	dbType := strings.TrimSpace(field.FieldDBType)
	browserName := browserNameFromFieldDBType(dbType)

	if dbType != "" {
		rows, err := a.queryBrowserURLDefs(ctx, a.col("fielddbtype")+" = ?", dbType, true)
		if err == nil && len(rows) > 0 {
			return browserURLDefFromRow(rows[0], browserName), true
		}
		if err != nil {
			pkglogger.Global().Debug("浏览按钮解析：按 fielddbtype 查询自定义浏览框失败，尝试降级",
				zap.String("fieldKey", field.FieldKey),
				zap.String("fieldDBType", field.FieldDBType),
				zap.Error(err))
		}
	}

	if field.Type > 0 && field.Type != 161 && field.Type != 162 {
		rows, err := a.queryBrowserURLDefs(ctx, a.col("id")+" = ?", field.Type, true)
		if err == nil && len(rows) > 0 {
			return browserURLDefFromRow(rows[0], browserName), true
		}
	}

	return e9BrowserURLDef{}, false
}

func (a *Ecology9Adapter) resolveModeBrowserTarget(ctx context.Context, browserName string) (e9BrowseTarget, bool) {
	def, ok := a.fetchModeBrowserDef(ctx, browserName)
	if !ok {
		return e9BrowseTarget{}, false
	}
	if target, ok := parseModeBrowserSQLTextTarget(def.SQLText); ok {
		return target, true
	}
	for _, sqlText := range []string{def.SearchByID, def.SQLText1} {
		if target, ok := parseModeBrowserSearchByIDTarget(sqlText); ok {
			return target, true
		}
	}
	return e9BrowseTarget{}, false
}

func (a *Ecology9Adapter) fetchModeBrowserDef(ctx context.Context, browserName string) (e9ModeBrowserDef, bool) {
	browserName = strings.TrimSpace(browserName)
	if browserName == "" {
		return e9ModeBrowserDef{}, false
	}
	var rows []map[string]interface{}
	err := a.db.WithContext(ctx).
		Table(a.tableName("mode_browser")).
		Select(strings.Join([]string{
			a.col("id") + " AS id",
			a.col("showname") + " AS showname",
			a.col("name") + " AS name",
			a.col("sqltext") + " AS sqltext",
			a.col("searchbyid") + " AS searchbyid",
			a.col("sqltext1") + " AS sqltext1",
		}, ", ")).
		Where(a.col("showname")+" = ?", browserName).
		Find(&rows).Error
	if err != nil {
		pkglogger.Global().Debug("浏览按钮解析：查询建模浏览框失败，保留原始值",
			zap.String("browserName", browserName),
			zap.Error(err))
		return e9ModeBrowserDef{}, false
	}
	if len(rows) == 0 {
		return e9ModeBrowserDef{}, false
	}
	row := rows[0]
	return e9ModeBrowserDef{
		ID:         mapGetInt(row, "id"),
		ShowName:   mapGet(row, "showname"),
		Name:       mapGet(row, "name"),
		SQLText:    mapGet(row, "sqltext"),
		SearchByID: mapGet(row, "searchbyid"),
		SQLText1:   mapGet(row, "sqltext1"),
	}, true
}

func (a *Ecology9Adapter) queryBrowserURLDefs(ctx context.Context, where string, arg interface{}, includeFieldDBType bool) ([]map[string]interface{}, error) {
	selectParts := []string{
		a.col("id") + " AS id",
		a.col("browserurl") + " AS browserurl",
		a.col("tablename") + " AS tablename",
		a.col("columname") + " AS columname",
		a.col("keycolumname") + " AS keycolumname",
	}
	if includeFieldDBType {
		selectParts = append(selectParts, a.col("fielddbtype")+" AS fielddbtype")
	}
	query := a.db.WithContext(ctx).
		Table(a.tableName("workflow_browserurl")).
		Select(strings.Join(selectParts, ", "))
	if strings.TrimSpace(where) != "" {
		query = query.Where(where, arg)
	}
	var rows []map[string]interface{}
	err := query.Find(&rows).Error
	if err != nil {
		if includeFieldDBType {
			return a.queryBrowserURLDefsLegacy(ctx, where, arg, false)
		}
		return a.queryBrowserURLDefsLegacy(ctx, where, arg, includeFieldDBType)
	}
	return rows, err
}

func (a *Ecology9Adapter) queryBrowserURLDefsLegacy(ctx context.Context, where string, arg interface{}, includeFieldDBType bool) ([]map[string]interface{}, error) {
	selectParts := []string{
		a.col("id") + " AS id",
		a.col("browserurl") + " AS browserurl",
	}
	if includeFieldDBType {
		selectParts = append(selectParts, a.col("fielddbtype")+" AS fielddbtype")
	}
	query := a.db.WithContext(ctx).
		Table(a.tableName("workflow_browserurl")).
		Select(strings.Join(selectParts, ", "))
	if strings.TrimSpace(where) != "" {
		query = query.Where(where, arg)
	}
	var rows []map[string]interface{}
	err := query.Find(&rows).Error
	if err != nil && includeFieldDBType {
		return a.queryBrowserURLDefsLegacy(ctx, where, arg, false)
	}
	return rows, err
}

func browserURLDefFromRow(row map[string]interface{}, browserName string) e9BrowserURLDef {
	return e9BrowserURLDef{
		ID:          mapGetInt(row, "id"),
		BrowserName: browserName,
		BrowserURL:  mapGet(row, "browserurl"),
		FieldDBType: mapGet(row, "fielddbtype"),
		TableName:   mapGet(row, "tablename"),
		ColumnName:  mapGet(row, "columname"),
		KeyColumn:   mapGet(row, "keycolumname"),
	}
}

func browseTargetFromBrowserURLDef(def e9BrowserURLDef) (e9BrowseTarget, bool) {
	table := strings.TrimSpace(def.TableName)
	idColumn := strings.TrimSpace(def.KeyColumn)
	displayColumn := strings.TrimSpace(def.ColumnName)
	if table == "" || idColumn == "" || displayColumn == "" {
		return e9BrowseTarget{}, false
	}
	if !isSafeIdentifier(table) || !isSafeIdentifier(idColumn) {
		return e9BrowseTarget{}, false
	}
	if isSafeIdentifier(displayColumn) {
		return e9BrowseTarget{
			Table:         table,
			IDColumn:      idColumn,
			DisplayColumn: displayColumn,
			NumericID:     isLikelyNumericBrowseKey(idColumn),
		}, true
	}
	if !isSafeDisplayExpression(displayColumn) {
		return e9BrowseTarget{}, false
	}
	return e9BrowseTarget{
		Table:             table,
		IDColumn:          idColumn,
		DisplayColumn:     displayColumn,
		DisplayExpression: true,
		NumericID:         isLikelyNumericBrowseKey(idColumn),
	}, true
}

func isLikelyNumericBrowseKey(column string) bool {
	col := strings.ToLower(strings.TrimSpace(column))
	return col == "id" || strings.HasSuffix(col, "id")
}

func browserNameFromFieldDBType(fieldDBType string) string {
	s := strings.TrimSpace(fieldDBType)
	if strings.HasPrefix(strings.ToLower(s), "browser.") {
		return s[len("browser."):]
	}
	return s
}

func (a *Ecology9Adapter) fetchBrowseDisplayMap(ctx context.Context, target e9BrowseTarget, ids []string) map[string]string {
	result := map[string]string{}
	if len(ids) == 0 || !isSafeIdentifier(target.Table) || !isSafeIdentifier(target.IDColumn) || !target.hasSafeDisplaySelector() {
		return result
	}
	validIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		if target.validID(id) {
			validIDs = append(validIDs, id)
		}
	}
	if len(validIDs) == 0 {
		return result
	}
	var rows []map[string]interface{}
	displaySelector := target.DisplayColumn
	if !target.DisplayExpression {
		displaySelector = a.col(displaySelector)
	}
	err := a.db.WithContext(ctx).
		Table(a.tableName(target.Table)).
		Select(a.col(target.IDColumn)+" AS browse_id, "+displaySelector+" AS browse_display").
		Where(a.col(target.IDColumn)+" IN ?", validIDs).
		Find(&rows).Error
	if err != nil {
		pkglogger.Global().Warn("浏览按钮解析：查询显示值失败，保留原始值",
			zap.String("table", target.Table),
			zap.String("idColumn", target.IDColumn),
			zap.String("displayColumn", target.DisplayColumn),
			zap.Error(err))
		return result
	}
	for _, row := range rows {
		id := mapGet(row, "browse_id")
		display := mapGet(row, "browse_display")
		if id != "" && display != "" {
			result[id] = display
		}
	}
	return result
}

func (t e9BrowseTarget) cacheKey() string {
	return strings.ToLower(strings.Join([]string{t.Table, t.IDColumn, t.DisplayColumn, fmt.Sprintf("%t", t.DisplayExpression), fmt.Sprintf("%t", t.Multiple), fmt.Sprintf("%t", t.NumericID), t.Source}, "|"))
}

func (t e9BrowseTarget) hasSafeDisplaySelector() bool {
	if t.DisplayExpression {
		return isSafeDisplayExpression(t.DisplayColumn)
	}
	return isSafeIdentifier(t.DisplayColumn)
}

func (t e9BrowseTarget) validID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	if !t.NumericID {
		return true
	}
	for _, r := range id {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func splitBrowseIDs(raw string) []string {
	parts := strings.Split(raw, ",")
	ids := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		id := strings.TrimSpace(part)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func buildBrowseResolvedValue(raw string, multiple bool, displayMap map[string]string) map[string]interface{} {
	ids := splitBrowseIDs(raw)
	if len(ids) == 0 {
		return nil
	}
	if !multiple && len(ids) == 1 {
		display := displayMap[ids[0]]
		if display == "" {
			return nil
		}
		return map[string]interface{}{
			"value":   ids[0],
			"display": display,
		}
	}
	items := make([]map[string]interface{}, 0, len(ids))
	displays := make([]string, 0, len(ids))
	hasDisplay := false
	for _, id := range ids {
		display := displayMap[id]
		if display != "" {
			hasDisplay = true
			displays = append(displays, display)
		} else {
			displays = append(displays, id)
		}
		items = append(items, map[string]interface{}{
			"value":   id,
			"display": display,
		})
	}
	if !hasDisplay {
		return nil
	}
	return map[string]interface{}{
		"value":   raw,
		"display": strings.Join(displays, ", "),
		"items":   items,
	}
}

func buildChoiceResolvedValue(raw string, optionMap map[string]string) map[string]interface{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if display := optionMap[raw]; display != "" {
		return map[string]interface{}{
			"value":   raw,
			"display": display,
		}
	}

	values := splitBrowseIDs(raw)
	if len(values) <= 1 {
		return nil
	}
	items := make([]map[string]interface{}, 0, len(values))
	displays := make([]string, 0, len(values))
	hasDisplay := false
	for _, value := range values {
		display := optionMap[value]
		if display != "" {
			hasDisplay = true
			displays = append(displays, display)
		} else {
			displays = append(displays, value)
		}
		items = append(items, map[string]interface{}{
			"value":   value,
			"display": display,
		})
	}
	if !hasDisplay {
		return nil
	}
	return map[string]interface{}{
		"value":   raw,
		"display": strings.Join(displays, ", "),
		"items":   items,
	}
}

func normalizeChoiceDisplayName(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	labels := parseE9MultiLangText(s)
	if len(labels) > 0 {
		for _, langID := range []string{"7", "9", "8"} {
			if label := strings.TrimSpace(labels[langID]); label != "" {
				return label
			}
		}
	}
	return strings.Trim(s, "`~ \t\r\n")
}

func parseE9MultiLangText(raw string) map[string]string {
	s := strings.TrimSpace(raw)
	labels := map[string]string{}
	for i := 0; i < len(s); {
		for i < len(s) && (isE9SelectNameDelimiter(s[i]) || isASCIISpace(s[i])) {
			i++
		}
		if i >= len(s) || s[i] < '0' || s[i] > '9' {
			break
		}

		langStart := i
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		langID := strings.TrimSpace(s[langStart:i])
		for i < len(s) && isASCIISpace(s[i]) {
			i++
		}

		labelStart := i
		labelEnd := len(s)
		for j := i; j < len(s); j++ {
			if !isE9SelectNameDelimiter(s[j]) {
				continue
			}
			k := j
			for k < len(s) && (isE9SelectNameDelimiter(s[k]) || isASCIISpace(s[k])) {
				k++
			}
			if k >= len(s) {
				labelEnd = j
				break
			}
			if s[k] >= '0' && s[k] <= '9' {
				labelEnd = j
				break
			}
		}

		label := strings.Trim(s[labelStart:labelEnd], "`~ \t\r\n")
		if langID != "" && label != "" {
			labels[langID] = label
		}
		if labelEnd >= len(s) {
			break
		}
		i = labelEnd
	}
	if len(labels) == 0 {
		return nil
	}
	return labels
}

func isE9SelectNameDelimiter(b byte) bool {
	return b == '`' || b == '~'
}

func isASCIISpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n'
}

func parseModeBrowserSQLTextTarget(sqlText string) (e9BrowseTarget, bool) {
	sqlText = strings.TrimSpace(sqlText)
	if sqlText == "" {
		return e9BrowseTarget{}, false
	}
	re := regexp.MustCompile(`(?is)\bselect\s+(.+?)\s+\bfrom\s+([a-zA-Z_][a-zA-Z0-9_]*)\b`)
	matches := re.FindStringSubmatch(sqlText)
	if len(matches) < 3 {
		return e9BrowseTarget{}, false
	}
	columns := splitSelectColumns(matches[1])
	if len(columns) < 2 {
		return e9BrowseTarget{}, false
	}
	idColumn, ok := cleanSelectColumn(columns[0])
	if !ok {
		return e9BrowseTarget{}, false
	}
	displayColumn, ok := cleanSelectColumn(columns[1])
	if !ok {
		return e9BrowseTarget{}, false
	}
	table := strings.TrimSpace(matches[2])
	if !isSafeIdentifier(table) || !isSafeIdentifier(idColumn) || !isSafeIdentifier(displayColumn) {
		return e9BrowseTarget{}, false
	}
	return e9BrowseTarget{Table: table, IDColumn: idColumn, DisplayColumn: displayColumn}, true
}

func parseModeBrowserSearchByIDTarget(sqlText string) (e9BrowseTarget, bool) {
	sqlText = strings.TrimSpace(sqlText)
	if sqlText == "" {
		return e9BrowseTarget{}, false
	}
	re := regexp.MustCompile(`(?is)\bselect\s+(.+?)\s+\bfrom\s+([a-zA-Z_][a-zA-Z0-9_]*)\b.+?\bwhere\s+([a-zA-Z_][a-zA-Z0-9_\.]*)\s*=\s*\?`)
	matches := re.FindStringSubmatch(sqlText)
	if len(matches) < 4 {
		return e9BrowseTarget{}, false
	}
	columns := splitSelectColumns(matches[1])
	if len(columns) == 0 {
		return e9BrowseTarget{}, false
	}
	displayColumn, ok := cleanSelectColumn(columns[0])
	if !ok {
		return e9BrowseTarget{}, false
	}
	idColumn, ok := cleanSelectColumn(matches[3])
	if !ok {
		return e9BrowseTarget{}, false
	}
	table := strings.TrimSpace(matches[2])
	if !isSafeIdentifier(table) || !isSafeIdentifier(idColumn) || !isSafeIdentifier(displayColumn) {
		return e9BrowseTarget{}, false
	}
	return e9BrowseTarget{Table: table, IDColumn: idColumn, DisplayColumn: displayColumn, NumericID: isLikelyNumericBrowseKey(idColumn)}, true
}

func splitSelectColumns(selectPart string) []string {
	var cols []string
	start := 0
	depth := 0
	inQuote := rune(0)
	for i, r := range selectPart {
		switch {
		case inQuote != 0:
			if r == inQuote {
				inQuote = 0
			}
		case r == '\'' || r == '"':
			inQuote = r
		case r == '(':
			depth++
		case r == ')' && depth > 0:
			depth--
		case r == ',' && depth == 0:
			cols = append(cols, strings.TrimSpace(selectPart[start:i]))
			start = i + len(string(r))
		}
	}
	cols = append(cols, strings.TrimSpace(selectPart[start:]))
	return cols
}

func cleanSelectColumn(expr string) (string, bool) {
	s := strings.TrimSpace(expr)
	if s == "" {
		return "", false
	}
	lower := strings.ToLower(s)
	if idx := strings.LastIndex(lower, " as "); idx >= 0 {
		s = strings.TrimSpace(s[:idx])
	}
	fields := strings.Fields(s)
	if len(fields) >= 1 {
		s = fields[0]
	}
	if dot := strings.LastIndex(s, "."); dot >= 0 {
		s = s[dot+1:]
	}
	s = strings.Trim(s, "`\"[] ")
	if !isSafeIdentifier(s) {
		return "", false
	}
	return s, true
}

func isSafeIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || (i > 0 && r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func isSafeDisplayExpression(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	lower := strings.ToLower(s)
	if strings.Contains(lower, ";") || strings.Contains(lower, "--") || strings.Contains(lower, "/*") || strings.Contains(lower, "*/") {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '.' || r == '(' || r == ')' || r == '+' || r == '-' || r == '*' || r == '/' || r == '\'' || r == '"' || r == '|' || r == ',':
		case r == ' ' || r == '\t' || r == '\r' || r == '\n':
		default:
			return false
		}
	}
	return true
}

// recognizeMainAttachments 识别主表中的附件字段（fieldhtmltype=6），
// 调用注入的识别服务获取每个 docId 的解析文本。
func (a *Ecology9Adapter) recognizeMainAttachments(
	ctx context.Context,
	processID string,
	formID int,
	mainData map[string]interface{},
) ([]AttachmentInfo, error) {
	// 查询主表中所有附件类型字段（detailtable 为空 / 0 / 主表名都视为主表）
	var rawFields []map[string]interface{}
	err := a.db.WithContext(ctx).
		Table(a.tableName("workflow_billfield")+" "+a.col("t1")).
		Select(a.col("t1.fieldname")+" AS fieldkey, "+a.col("t2.labelname")+" AS fieldname, "+a.col("t1.detailtable")+" AS detailtable").
		Joins("JOIN "+a.tableName("htmllabelinfo")+" "+a.col("t2")+" ON "+a.col("t1.fieldlabel")+" = "+a.col("t2.indexid")).
		Where(a.col("t1.billid")+" = ? AND "+a.col("t1.fieldhtmltype")+" = ? AND "+a.col("t2.languageid")+" = 7",
			formID, "6").
		Find(&rawFields).Error
	if err != nil {
		return nil, fmt.Errorf("查询附件字段定义失败: %w", err)
	}

	pkglogger.Global().Info("附件识别：查询到附件类字段定义",
		zap.String("processID", processID),
		zap.Int("formID", formID),
		zap.Int("fieldDefinitionCount", len(rawFields)))

	var all []AttachmentInfo
	var skippedDetail, skippedEmpty, fetchFailed, recogFailed int
	for _, row := range rawFields {
		dt := strings.TrimSpace(mapGet(row, "detailtable"))
		// 仅识别主表附件；明细表附件如有需要后续扩展
		if dt != "" && dt != "0" {
			skippedDetail++
			pkglogger.Global().Debug("附件识别：跳过明细表附件字段",
				zap.String("processID", processID),
				zap.String("field", mapGet(row, "fieldkey")),
				zap.String("detailTable", dt))
			continue
		}
		fieldKey := mapGet(row, "fieldkey")
		fieldName := mapGet(row, "fieldname")
		if fieldKey == "" {
			continue
		}
		if !attachmentFieldAllowed(ctx, fieldKey) {
			pkglogger.Global().Debug("附件识别：字段未被业务配置选中，跳过",
				zap.String("processID", processID),
				zap.String("field", fieldKey),
				zap.String("fieldName", fieldName))
			continue
		}
		// 主表数据里取附件 docId 列表（逗号分隔）
		docIds := strings.TrimSpace(mapGet(mainData, fieldKey))
		if docIds == "" {
			skippedEmpty++
			pkglogger.Global().Debug("附件识别：主表字段无 docId，跳过",
				zap.String("processID", processID),
				zap.String("field", fieldKey),
				zap.String("fieldName", fieldName))
			continue
		}
		pkglogger.Global().Info("附件识别：准备拉取泛微附件",
			zap.String("processID", processID),
			zap.String("field", fieldKey),
			zap.String("fieldName", fieldName),
			zap.String("docIds", docIds))
		files, fetchErr := a.fetchWeaverAttachmentsByDocIDs(ctx, processID, fieldKey, docIds)
		if fetchErr != nil {
			fetchFailed++
			pkglogger.Global().Warn("附件识别：拉取泛微附件失败，跳过该字段",
				zap.String("processID", processID),
				zap.String("field", fieldKey),
				zap.Error(fetchErr))
			continue
		}
		pkglogger.Global().Info("附件识别：泛微附件拉取成功，开始按格式解析",
			zap.String("processID", processID),
			zap.String("field", fieldKey),
			zap.Int("fileCount", len(files)))
		infos, recogErr := a.attachmentRecognitionSvc.RecognizeAttachments(ctx, files, fieldKey, fieldName)
		if recogErr != nil {
			recogFailed++
			pkglogger.Global().Warn("附件识别：附件解析失败，跳过该字段",
				zap.String("processID", processID),
				zap.String("field", fieldKey),
				zap.Error(recogErr))
			continue
		}
		var withContent, withError int
		for _, info := range infos {
			if info.Error != "" {
				withError++
			} else if info.Content != "" {
				withContent++
			}
		}
		pkglogger.Global().Info("附件识别：字段处理完成",
			zap.String("processID", processID),
			zap.String("field", fieldKey),
			zap.Int("resultCount", len(infos)),
			zap.Int("withContent", withContent),
			zap.Int("withError", withError))
		all = append(all, infos...)
	}
	pkglogger.Global().Info("附件识别：主表字段遍历结束",
		zap.String("processID", processID),
		zap.Int("attachmentCount", len(all)),
		zap.Int("skippedDetailField", skippedDetail),
		zap.Int("skippedEmptyDocId", skippedEmpty),
		zap.Int("fetchFailedField", fetchFailed),
		zap.Int("recognizeFailedField", recogFailed))
	return all, nil
}

func (a *Ecology9Adapter) fetchWeaverAttachmentsByDocIDs(
	ctx context.Context,
	processID, fieldKey, docIDs string,
) ([]AttachmentFilePayload, error) {
	if a.weaverAPIURL == "" {
		return nil, fmt.Errorf("未配置泛微附件接口 URL（weaver_api_url）")
	}
	if a.weaverAppID == "" {
		return nil, fmt.Errorf("未配置泛微 appid（weaver_appid）")
	}
	if a.weaverLoginID == "" {
		return nil, fmt.Errorf("未配置泛微 loginid（weaver_default_user）")
	}

	reqURL, err := url.Parse(a.weaverAPIURL)
	if err != nil {
		return nil, fmt.Errorf("泛微附件接口 URL 非法: %w", err)
	}
	query := reqURL.Query()
	query.Set("docIds", docIDs)
	query.Set("appid", a.weaverAppID)
	query.Set("loginid", a.weaverLoginID)
	reqURL.RawQuery = query.Encode()

	pkglogger.Global().Info("附件识别：请求泛微附件接口",
		zap.String("processID", processID),
		zap.String("field", fieldKey),
		zap.String("docIds", docIDs),
		zap.String("apiBase", fmt.Sprintf("%s://%s%s", reqURL.Scheme, reqURL.Host, reqURL.Path)))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("创建泛微附件请求失败: %w", err)
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用泛微附件接口失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errBody := string(body)
		if len(errBody) > 200 {
			errBody = errBody[:200] + "..."
		}
		pkglogger.Global().Warn("附件识别：泛微附件接口 HTTP 异常",
			zap.String("processID", processID),
			zap.String("field", fieldKey),
			zap.Int("status", resp.StatusCode),
			zap.String("body", errBody))
		return nil, fmt.Errorf("泛微附件接口 HTTP %d: %s", resp.StatusCode, errBody)
	}

	// 泛微桥接接口的对外契约使用 camelCase：
	// { docId, fileName, fileSize, fileData }。
	// AuraOA 内部模型则统一保留 snake_case JSON tag，故在边界层显式转换，
	// 避免外部协议细节泄漏到内部结构。
	var result struct {
		Code int `json:"code"`
		Data []struct {
			DocID    string `json:"docId"`
			FileName string `json:"fileName"`
			FileSize int64  `json:"fileSize"`
			FileData string `json:"fileData"`
		} `json:"data"`
		Msg string `json:"msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析泛微附件接口响应失败: %w", err)
	}
	if result.Code != 0 {
		pkglogger.Global().Warn("附件识别：泛微附件接口业务错误",
			zap.String("processID", processID),
			zap.String("field", fieldKey),
			zap.Int("code", result.Code),
			zap.String("msg", result.Msg))
		return nil, fmt.Errorf("泛微附件接口返回错误: %s", result.Msg)
	}
	files := make([]AttachmentFilePayload, 0, len(result.Data))
	fileNames := make([]string, 0, len(result.Data))
	for _, f := range result.Data {
		files = append(files, AttachmentFilePayload{
			DocID:    f.DocID,
			FileName: f.FileName,
			FileSize: f.FileSize,
			FileData: f.FileData,
		})
		fileNames = append(fileNames, f.FileName)
	}
	pkglogger.Global().Info("附件识别：泛微附件接口返回成功",
		zap.String("processID", processID),
		zap.String("field", fieldKey),
		zap.Int("fileCount", len(result.Data)),
		zap.Strings("fileNames", fileNames))
	return files, nil
}

// ResolveUsernameByOAUserID 根据泛微 E9 人员数值 ID 反查登录账号（loginid）。
func (a *Ecology9Adapter) ResolveUsernameByOAUserID(ctx context.Context, oaUserID string) (string, error) {
	trimmed := strings.TrimSpace(oaUserID)
	if trimmed == "" {
		return "", fmt.Errorf("oa_user_id 不能为空")
	}
	intID, err := strconv.Atoi(trimmed)
	if err != nil {
		return "", fmt.Errorf("无效的泛微人员ID: %s", trimmed)
	}

	var loginid string
	err = a.db.WithContext(ctx).
		Table(a.tableName("hrmresource")).
		Select(a.col("loginid")).
		Where(a.col("id")+" = ?", intID).
		Row().Scan(&loginid)
	if err != nil {
		return "", fmt.Errorf("未在 OA 中找到人员 ID 为 '%d' 的用户: %w", intID, err)
	}
	return strings.TrimSpace(loginid), nil
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

// castToTextExpr 将字段表达式转换为文本，用于处理不同数据库的类型转换方言。
func (a *Ecology9Adapter) castToTextExpr(expr string) string {
	if a.isOracleCompatible() {
		return "TO_CHAR(" + expr + ")"
	}
	return "CAST(" + expr + " AS CHAR)"
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
// 包含完整审批历史（带操作类型映射）和流程定义路由图（带出口条件）。
// 路由图基于 workflowid 查询流程定义，与实例是否有审批记录无关。
// 若历史日志表结构不兼容，审批历史部分退化为当前节点，但路由图仍正常返回。
func (a *Ecology9Adapter) FetchProcessFlow(ctx context.Context, processID string) (*ProcessFlowSnapshot, error) {
	// ── 1. 查询 OA 系统中的真实当前节点（workflow_requestbase.currentnodeid） ──
	currentNodeName := a.fetchCurrentNodeName(ctx, processID)

	// ── 2. 获取流程定义路由图（节点连接 + 出口条件） ──
	// 路由图基于 workflowid，是流程定义级别的信息，与当前实例走了多少步无关。
	graphText := a.fetchFlowRouteGraph(ctx, processID)

	// ── 2. 获取审批历史（仅最后一次退回之后的有效路径） ──
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
		// 审批历史查询失败，退化为当前节点，但路由图仍保留
		snapshot, _ := a.fetchCurrentNodeSnapshot(ctx, processID)
		if snapshot != nil && graphText != "" {
			snapshot.GraphText = graphText
		}
		return snapshot, nil
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
		// 审批历史为空（流程还在第一个节点），退化为当前节点，但路由图仍保留
		snapshot, _ := a.fetchCurrentNodeSnapshot(ctx, processID)
		if snapshot != nil {
			snapshot.HistoryText = "（暂无审批流转记录）"
			if graphText != "" {
				snapshot.GraphText = graphText
			}
		}
		return snapshot, nil
	}

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
		IsComplete:      true,
		MissingNodes:    []string{},
		CurrentNodeName: currentNodeName,
		Nodes:           nodes,
		HistoryText:     strings.Join(historyLines, "\n"),
		GraphText:       graphText,
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

	// 查询节点连接和出口条件。规则 ID 的文本转换按数据库方言生成。
	ruleIDTextExpr := a.castToTextExpr("RB." + a.col("id"))
	query := fmt.Sprintf(`
		SELECT
			COALESCE(WN1.%s, '') AS src_node_name,
			COALESCE(WN2.%s, '') AS dest_node_name,
			COALESCE(WN.%s, '') AS link_name,
			COALESCE(RB.%s, '') AS condition_text
		FROM %s WN
		LEFT JOIN %s WN1 ON WN1.%s = WN.%s
		LEFT JOIN %s WN2 ON WN2.%s = WN.%s
		LEFT JOIN %s RB ON %s = WN.%s
		WHERE WN.%s = ?
		ORDER BY WN.%s, WN.%s`,
		a.col("nodename"),
		a.col("nodename"),
		a.col("linkname"),
		a.col("condit"),
		a.tableName("workflow_nodelink"),
		a.tableName("workflow_nodebase"), a.col("id"), a.col("nodeid"),
		a.tableName("workflow_nodebase"), a.col("id"), a.col("destnodeid"),
		a.tableName("rule_base"), ruleIDTextExpr, a.col("newrule"),
		a.col("workflowid"),
		a.col("nodeid"), a.col("destnodeid"),
	)

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

// fetchCurrentNodeName 查询 OA 系统中流程实例的真实当前节点名称（基于 workflow_requestbase.currentnodeid）。
func (a *Ecology9Adapter) fetchCurrentNodeName(ctx context.Context, processID string) string {
	query := fmt.Sprintf(`
		SELECT
			COALESCE(n.%s, '')
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
		return ""
	}
	return nodeName
}

func (a *Ecology9Adapter) fetchCurrentNodeSnapshot(ctx context.Context, processID string) (*ProcessFlowSnapshot, error) {
	nodeName := a.fetchCurrentNodeName(ctx, processID)
	if nodeName == "" {
		nodeName = "已归档"
	}

	node := ProcessFlowNode{
		NodeID:   nodeName,
		NodeName: nodeName,
		Action:   "approve",
	}

	return &ProcessFlowSnapshot{
		IsComplete:      true,
		MissingNodes:    []string{},
		CurrentNodeName: nodeName,
		Nodes:           []ProcessFlowNode{node},
		HistoryText:     "",
		GraphText:       nodeName,
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

// CheckProcessVisibility 检查指定用户对流程实例是否具有可见性（待办人、申请人或历史审批人）。
func (a *Ecology9Adapter) CheckProcessVisibility(ctx context.Context, username string, processID string) (bool, error) {
	var e9UserID int
	err := a.db.WithContext(ctx).
		Table(a.tableName("hrmresource")).
		Select(a.col("id")).
		Where(a.col("loginid")+" = ?", username).
		Row().Scan(&e9UserID)
	if err != nil {
		return false, nil
	}

	// 1. 判断是否在待办中
	inTodo, err := a.IsProcessInTodo(ctx, username, processID)
	if err == nil && inTodo {
		return true, nil
	}

	// 2. 判断是否是流程创建/申请人
	var createrCount int64
	err = a.db.WithContext(ctx).
		Table(a.tableName("workflow_requestbase")).
		Where(a.col("requestid")+" = ? AND "+a.col("creater")+" = ?", processID, e9UserID).
		Count(&createrCount).Error
	if err == nil && createrCount > 0 {
		return true, nil
	}

	// 3. 判断是否在审批历史操作人中
	var logCount int64
	err = a.db.WithContext(ctx).
		Table(a.tableName("workflow_requestlog")).
		Where(a.col("requestid")+" = ? AND "+a.col("operator")+" = ?", processID, e9UserID).
		Count(&logCount).Error
	if err == nil && logCount > 0 {
		return true, nil
	}

	return false, nil
}

// FetchProcessRequestSummary 按 requestid 拉取流程实例摘要。
func (a *Ecology9Adapter) FetchProcessRequestSummary(ctx context.Context, processID string) (*ProcessRequestSummary, error) {
	query := fmt.Sprintf(`
		SELECT
			r.%s AS request_id,
			COALESCE(r.%s, '') AS request_name,
			COALESCE(h.%s, '') AS applicant_name,
			COALESCE(d.%s, '') AS dept_name,
			COALESCE(wb.%s, '') AS workflow_name,
			COALESCE(wt.%s, '') AS type_name,
			COALESCE(n.%s, '') AS node_name,
			COALESCE(r.%s, '') AS create_date
		FROM %s r
		LEFT JOIN %s wb ON r.%s = wb.%s
		LEFT JOIN %s wt ON wb.%s = wt.%s
		LEFT JOIN %s h ON r.%s = h.%s
		LEFT JOIN %s d ON h.%s = d.%s
		LEFT JOIN %s n ON r.%s = n.%s
		WHERE r.%s = ?`,
		a.col("requestid"), a.col("requestname"),
		a.col("lastname"), a.col("departmentname"),
		a.col("workflowname"), a.col("typename"),
		a.col("nodename"), a.col("createdate"),
		a.tableName("workflow_requestbase"),
		a.tableName("workflow_base"), a.col("workflowid"), a.col("id"),
		a.tableName("workflow_type"), a.col("workflowtype"), a.col("id"),
		a.tableName("hrmresource"), a.col("creater"), a.col("id"),
		a.tableName("hrmdepartment"), a.col("departmentid"), a.col("id"),
		a.tableName("workflow_nodebase"), a.col("currentnodeid"), a.col("id"),
		a.col("requestid"),
	)

	var requestID, requestName, applicant, department, workflowName, typeName, nodeName, createDate string
	err := a.db.WithContext(ctx).Raw(query, processID).Row().Scan(
		&requestID, &requestName, &applicant, &department, &workflowName, &typeName, &nodeName, &createDate,
	)
	if err != nil {
		return nil, fmt.Errorf("查询流程实例失败: %w", err)
	}
	return &ProcessRequestSummary{
		ProcessID:        requestID,
		Title:            requestName,
		Applicant:        applicant,
		Department:       department,
		ProcessType:      workflowName,
		ProcessTypeLabel: typeName,
		CurrentNode:      nodeName,
		SubmitTime:       createDate,
	}, nil
}

// CaptureProcessRequestHighWatermark 读取 OA 流程实例表当前最大的 requestid。
func (a *Ecology9Adapter) CaptureProcessRequestHighWatermark(ctx context.Context) (int64, error) {
	query := fmt.Sprintf(
		"SELECT COALESCE(MAX(%s), 0) FROM %s",
		a.col("requestid"),
		a.tableName("workflow_requestbase"),
	)
	var watermark int64
	if err := a.db.WithContext(ctx).Raw(query).Row().Scan(&watermark); err != nil {
		return 0, fmt.Errorf("读取 OA 流程 requestid 高水位失败: %w", err)
	}
	return watermark, nil
}

// FindCreatedProcessRequestsAfter 查询高水位之后的同一流程定义实例。
func (a *Ecology9Adapter) FindCreatedProcessRequestsAfter(
	ctx context.Context,
	workflowID string,
	afterRequestID int64,
	limit int,
) ([]ProcessRequestCandidate, error) {
	workflowID = strings.TrimSpace(workflowID)
	if workflowID == "" {
		return nil, fmt.Errorf("workflow_id 不能为空")
	}
	if limit < 1 || limit > 20 {
		limit = 3
	}

	query := fmt.Sprintf(`
		SELECT
			r.%s AS request_id,
			COALESCE(r.%s, '') AS request_name,
			COALESCE(r.%s, 0) AS creator_id
		FROM %s r
		WHERE r.%s > ?
		  AND r.%s = ?
		ORDER BY r.%s ASC%s`,
		a.col("requestid"), a.col("requestname"), a.col("creater"),
		a.tableName("workflow_requestbase"),
		a.col("requestid"),
		a.col("workflowid"),
		a.col("requestid"), a.limitOffsetClause(limit, 0),
	)
	rows, err := a.db.WithContext(ctx).Raw(query, afterRequestID, workflowID).Rows()
	if err != nil {
		return nil, fmt.Errorf("解析 OA 新建流程 requestid 失败: %w", err)
	}
	defer rows.Close()

	items := make([]ProcessRequestCandidate, 0, limit)
	for rows.Next() {
		var item ProcessRequestCandidate
		if scanErr := rows.Scan(&item.ProcessID, &item.Title, &item.CreatorID); scanErr != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

// FetchRecentProcessSummaries 拉取指定流程类型在时间窗口内创建的流程实例。
// 该查询只用于发现定时检查候选，是否调用 AI 仍由 AuraOA 的上下文指纹决定。
func (a *Ecology9Adapter) FetchRecentProcessSummaries(
	ctx context.Context,
	processType string,
	since time.Time,
	limit int,
) ([]ProcessRequestSummary, error) {
	processType = strings.TrimSpace(processType)
	if processType == "" {
		return nil, fmt.Errorf("流程类型不能为空")
	}
	if limit < 1 || limit > 1000 {
		limit = 200
	}

	query := fmt.Sprintf(`
		SELECT
			r.%s AS request_id,
			COALESCE(r.%s, '') AS request_name,
			COALESCE(h.%s, '') AS applicant_name,
			COALESCE(d.%s, '') AS dept_name,
			COALESCE(wb.%s, '') AS workflow_name,
			COALESCE(wt.%s, '') AS type_name,
			COALESCE(n.%s, '') AS node_name,
			COALESCE(r.%s, '') AS create_date
		FROM %s r
		LEFT JOIN %s wb ON r.%s = wb.%s
		LEFT JOIN %s wt ON wb.%s = wt.%s
		LEFT JOIN %s h ON r.%s = h.%s
		LEFT JOIN %s d ON h.%s = d.%s
		LEFT JOIN %s n ON r.%s = n.%s
		WHERE %s(COALESCE(wb.%s, '')) = %s(?)
		  AND r.%s >= ?
		ORDER BY r.%s DESC%s`,
		a.col("requestid"), a.col("requestname"),
		a.col("lastname"), a.col("departmentname"),
		a.col("workflowname"), a.col("typename"),
		a.col("nodename"), a.col("createdate"),
		a.tableName("workflow_requestbase"),
		a.tableName("workflow_base"), a.col("workflowid"), a.col("id"),
		a.tableName("workflow_type"), a.col("workflowtype"), a.col("id"),
		a.tableName("hrmresource"), a.col("creater"), a.col("id"),
		a.tableName("hrmdepartment"), a.col("departmentid"), a.col("id"),
		a.tableName("workflow_nodebase"), a.col("currentnodeid"), a.col("id"),
		a.lowerFunc(), a.col("workflowname"), a.lowerFunc(),
		a.col("createdate"),
		a.col("createdate"),
		a.limitOffsetClause(limit, 0),
	)

	rows, err := a.db.WithContext(ctx).Raw(query, processType, since).Rows()
	if err != nil {
		return nil, fmt.Errorf("查询 OA 近期流程失败: %w", err)
	}
	defer rows.Close()

	items := make([]ProcessRequestSummary, 0, limit)
	for rows.Next() {
		var item ProcessRequestSummary
		if err := rows.Scan(
			&item.ProcessID,
			&item.Title,
			&item.Applicant,
			&item.Department,
			&item.ProcessType,
			&item.ProcessTypeLabel,
			&item.CurrentNode,
			&item.SubmitTime,
		); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

// FetchProcessContextAnchor 拉取 OA 流程上下文锚点。
func (a *Ecology9Adapter) FetchProcessContextAnchor(ctx context.Context, processID string, pd *ProcessData) (*OAContextAnchor, error) {
	var currentNodeID int
	err := a.db.WithContext(ctx).
		Table(a.tableName("workflow_requestbase")).
		Select(a.col("currentnodeid")).
		Where(a.col("requestid")+" = ?", processID).
		Row().Scan(&currentNodeID)
	if err != nil {
		return nil, fmt.Errorf("查询流程当前节点失败: %w", err)
	}

	var lastReturnLogID int64
	returnQuery := fmt.Sprintf(`
		SELECT COALESCE(MAX(%s), 0) FROM %s WHERE %s = ? AND %s = '3'`,
		a.col("logid"), a.tableName("workflow_requestlog"), a.col("requestid"), a.col("logtype"),
	)
	if err := a.db.WithContext(ctx).Raw(returnQuery, processID).Row().Scan(&lastReturnLogID); err != nil {
		return nil, fmt.Errorf("查询退回日志失败: %w", err)
	}

	var flowRevision int64
	revisionQuery := fmt.Sprintf(`
		SELECT COALESCE(MAX(%s), 0) FROM %s WHERE %s = ? AND %s > ?`,
		a.col("logid"), a.tableName("workflow_requestlog"), a.col("requestid"), a.col("logid"),
	)
	if err := a.db.WithContext(ctx).Raw(revisionQuery, processID, lastReturnLogID).Row().Scan(&flowRevision); err != nil {
		return nil, fmt.Errorf("查询流程版本失败: %w", err)
	}

	var lastLogID int64
	lastLogQuery := fmt.Sprintf(
		"SELECT COALESCE(MAX(%s), 0) FROM %s WHERE %s = ?",
		a.col("logid"), a.tableName("workflow_requestlog"), a.col("requestid"),
	)
	if err := a.db.WithContext(ctx).Raw(lastLogQuery, processID).Row().Scan(&lastLogID); err != nil {
		return nil, fmt.Errorf("查询最新流程日志失败: %w", err)
	}
	var lastLogType string
	if lastLogID > 0 {
		lastLogTypeQuery := fmt.Sprintf(
			"SELECT COALESCE(%s, '') FROM %s WHERE %s = ? AND %s = ?",
			a.col("logtype"), a.tableName("workflow_requestlog"), a.col("requestid"), a.col("logid"),
		)
		_ = a.db.WithContext(ctx).Raw(lastLogTypeQuery, processID, lastLogID).Row().Scan(&lastLogType)
	}

	var lastResubmitLogID int64
	if lastReturnLogID > 0 {
		resubmitQuery := fmt.Sprintf(`
			SELECT COALESCE(MAX(%s), 0) FROM %s WHERE %s = ? AND %s = '2' AND %s > ?`,
			a.col("logid"), a.tableName("workflow_requestlog"), a.col("requestid"), a.col("logtype"), a.col("logid"),
		)
		_ = a.db.WithContext(ctx).Raw(resubmitQuery, processID, lastReturnLogID).Row().Scan(&lastResubmitLogID)
	}

	anchor := &OAContextAnchor{
		LastReturnLogID:   lastReturnLogID,
		FlowRevision:      flowRevision,
		LastResubmitLogID: lastResubmitLogID,
		LastLogID:         lastLogID,
		LastLogType:       strings.TrimSpace(lastLogType),
		CurrentNodeID:     currentNodeID,
	}
	if pd != nil {
		anchor.ContentFingerprint = ComputeProcessDataFingerprint(pd)
		attachmentVersions, versionErr := a.fetchAttachmentVersionAnchors(ctx, processID, pd)
		if versionErr != nil {
			pkglogger.Global().Warn("查询附件版本元数据失败，退化为附件字段值判断",
				zap.String("processID", processID),
				zap.Error(versionErr))
		} else {
			anchor.AttachmentFingerprint, anchor.AttachmentFieldFingerprints =
				ComputeAttachmentFingerprints(attachmentVersions)
		}
	}
	return anchor, nil
}

// fetchAttachmentVersionAnchors 读取主表附件的最新 DocImageFile 版本，不下载附件正文。
func (a *Ecology9Adapter) fetchAttachmentVersionAnchors(
	ctx context.Context,
	processID string,
	pd *ProcessData,
) ([]AttachmentVersionAnchor, error) {
	if pd == nil || len(pd.MainData) == 0 {
		return nil, nil
	}
	var workflowID int
	if err := a.db.WithContext(ctx).
		Table(a.tableName("workflow_requestbase")).
		Select(a.col("workflowid")).
		Where(a.col("requestid")+" = ?", processID).
		Row().Scan(&workflowID); err != nil {
		return nil, fmt.Errorf("查询附件流程定义失败: %w", err)
	}
	var formID int
	if err := a.db.WithContext(ctx).
		Table(a.tableName("workflow_base")).
		Select(a.col("formid")).
		Where(a.col("id")+" = ?", workflowID).
		Row().Scan(&formID); err != nil {
		return nil, fmt.Errorf("查询附件表单定义失败: %w", err)
	}

	var rawFields []map[string]interface{}
	if err := a.db.WithContext(ctx).
		Table(a.tableName("workflow_billfield")).
		Select(a.col("fieldname")+" AS field_key, "+a.col("detailtable")+" AS detail_table").
		Where(a.col("billid")+" = ? AND "+a.col("fieldhtmltype")+" = ?", formID, "6").
		Find(&rawFields).Error; err != nil {
		return nil, fmt.Errorf("查询附件字段定义失败: %w", err)
	}

	items := make([]AttachmentVersionAnchor, 0)
	docIDsSet := make(map[string]struct{})
	for _, field := range rawFields {
		detailTable := strings.TrimSpace(mapGet(field, "detail_table"))
		if detailTable != "" && detailTable != "0" {
			continue
		}
		fieldKey := strings.TrimSpace(mapGet(field, "field_key"))
		if fieldKey == "" {
			continue
		}
		docIDs := splitAttachmentDocIDs(mapGet(pd.MainData, fieldKey))
		for _, docID := range docIDs {
			docIDsSet[docID] = struct{}{}
			items = append(items, AttachmentVersionAnchor{FieldKey: fieldKey, DocID: docID})
		}
	}
	docIDs := make([]string, 0, len(docIDsSet))
	for docID := range docIDsSet {
		docIDs = append(docIDs, docID)
	}
	sort.Strings(docIDs)
	versions, err := a.fetchLatestAttachmentVersions(ctx, docIDs)
	if err != nil {
		pkglogger.Global().Warn("批量查询附件版本失败，退化为 docId 判断",
			zap.String("processID", processID),
			zap.Int("docCount", len(docIDs)),
			zap.Error(err))
		return items, nil
	}
	for i := range items {
		if row := versions[items[i].DocID]; row != nil {
			items[i].VersionID = int64(mapGetInt(row, "version_id"))
			items[i].ImageFileID = mapGet(row, "image_file_id")
			items[i].FileName = mapGet(row, "file_name")
		}
	}
	return items, nil
}

func splitAttachmentDocIDs(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；'
	})
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		id := strings.TrimSpace(part)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (a *Ecology9Adapter) fetchLatestAttachmentVersions(ctx context.Context, docIDs []string) (map[string]map[string]interface{}, error) {
	out := make(map[string]map[string]interface{}, len(docIDs))
	if len(docIDs) == 0 {
		return out, nil
	}
	docTable := a.tableName("DocImageFile")
	imageTable := a.tableName("ImageFile")
	docIDCol := a.col("docid")
	versionIDCol := a.col("versionid")
	imageFileIDCol := a.col("imagefileid")
	imageFileNameCol := a.col("imagefilename")
	query := fmt.Sprintf(`
		SELECT
			DIF.%s AS doc_id,
			DIF.%s AS version_id,
			DIF.%s AS image_file_id,
			COALESCE(IMG.%s, '') AS file_name
		FROM %s DIF
		LEFT JOIN %s IMG ON DIF.%s = IMG.%s
		WHERE DIF.%s IN ?
		  AND DIF.%s = (
			SELECT MAX(D2.%s) FROM %s D2 WHERE D2.%s = DIF.%s
		  )`,
		docIDCol,
		versionIDCol,
		imageFileIDCol,
		imageFileNameCol,
		docTable,
		imageTable, imageFileIDCol, imageFileIDCol,
		docIDCol,
		versionIDCol,
		versionIDCol, docTable, docIDCol, docIDCol,
	)
	var rows []map[string]interface{}
	if err := a.db.WithContext(ctx).Raw(query, docIDs).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("批量查询附件最新版本失败: %w", err)
	}
	for _, row := range rows {
		docID := strings.TrimSpace(mapGet(row, "doc_id"))
		if docID != "" {
			out[docID] = row
		}
	}
	return out, nil
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
