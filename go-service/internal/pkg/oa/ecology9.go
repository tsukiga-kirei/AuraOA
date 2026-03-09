package oa

import (
	"context"
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// Ecology9Adapter 泛微 E9 OA 系统适配器。
// 通过 GORM + MySQL 驱动连接 E9 数据库，封装 E9 特有的表结构查询。
type Ecology9Adapter struct {
	db *gorm.DB
}

// NewEcology9Adapter 根据 OA 数据库连接配置创建泛微 E9 适配器实例。
func NewEcology9Adapter(conn *model.OADatabaseConnection) (*Ecology9Adapter, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		conn.Username, conn.Password, conn.Host, conn.Port, conn.DatabaseName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("连接泛微 E9 数据库失败: %w", err)
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接池失败: %w", err)
	}
	sqlDB.SetMaxOpenConns(conn.PoolSize)
	sqlDB.SetMaxIdleConns(conn.PoolSize / 2)

	return &Ecology9Adapter{db: db}, nil
}

// e9WorkflowBase 泛微 E9 workflow_base 表映射
type e9WorkflowBase struct {
	ID           int    `gorm:"column:id"`
	WorkflowName string `gorm:"column:workflowname"`
	FormID       int    `gorm:"column:formid"`
	TableDBName  string `gorm:"column:tablename"`
}

func (e9WorkflowBase) TableName() string { return "workflow_base" }

// e9WorkflowBillField 泛微 E9 workflow_billfield 表映射（流程表单字段定义）
type e9WorkflowBillField struct {
	ID        int    `gorm:"column:id"`
	FieldDBName string `gorm:"column:fielddbname"`
	FieldName string `gorm:"column:fieldname"`
	FieldHTMLType string `gorm:"column:fieldhtmltype"`
	DetailTable int  `gorm:"column:detailtable"` // 0=主表, >0=明细表序号
	FormID    int    `gorm:"column:billid"`
}

func (e9WorkflowBillField) TableName() string { return "workflow_billfield" }

// e9WorkflowDetailTable 泛微 E9 明细表定义
type e9WorkflowDetailTable struct {
	ID          int    `gorm:"column:id"`
	TableDBName string `gorm:"column:tablename"`
	TableName   string `gorm:"column:tablename_view"` // 显示名称
	OrderID     int    `gorm:"column:orderid"`
}

// e9NodeInfo 泛微 E9 流程节点信息
type e9NodeInfo struct {
	ID       int `gorm:"column:id"`
	NodeType int `gorm:"column:nodetype"`
}

// ValidateProcess 验证流程类型是否存在于泛微 E9 系统中。
// 通过 workflow_base 表查询流程名称匹配的记录。
func (a *Ecology9Adapter) ValidateProcess(ctx context.Context, processType string) (*ProcessInfo, error) {
	var wf e9WorkflowBase
	err := a.db.WithContext(ctx).
		Where("workflowname = ?", processType).
		First(&wf).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("流程 '%s' 在泛微 E9 系统中不存在", processType)
		}
		return nil, fmt.Errorf("查询泛微 E9 流程失败: %w", err)
	}

	// 查询明细表数量
	var detailCount int64
	a.db.WithContext(ctx).
		Table("workflow_billfield").
		Where("billid = ? AND detailtable > 0", wf.FormID).
		Distinct("detailtable").
		Count(&detailCount)

	return &ProcessInfo{
		ProcessType: processType,
		ProcessName: wf.WorkflowName,
		MainTable:   wf.TableDBName,
		DetailCount: int(detailCount),
	}, nil
}

// FetchFields 从泛微 E9 拉取指定流程的全部字段定义。
// 通过 workflow_billfield 表查询，按主表和明细表分组返回。
func (a *Ecology9Adapter) FetchFields(ctx context.Context, processType string) (*ProcessFields, error) {
	// 先查询流程获取 formID
	var wf e9WorkflowBase
	err := a.db.WithContext(ctx).
		Where("workflowname = ?", processType).
		First(&wf).Error
	if err != nil {
		return nil, fmt.Errorf("查询流程 '%s' 失败: %w", processType, err)
	}

	// 查询所有字段
	var fields []e9WorkflowBillField
	err = a.db.WithContext(ctx).
		Where("billid = ?", wf.FormID).
		Order("detailtable ASC, id ASC").
		Find(&fields).Error
	if err != nil {
		return nil, fmt.Errorf("查询流程字段失败: %w", err)
	}

	// 按主表和明细表分组
	result := &ProcessFields{
		MainFields:   make([]FieldDef, 0),
		DetailTables: make([]DetailTableDef, 0),
	}

	// 用 map 收集明细表字段
	detailMap := make(map[int]*DetailTableDef)

	for _, f := range fields {
		fd := FieldDef{
			FieldKey:  f.FieldDBName,
			FieldName: f.FieldName,
			FieldType: a.mapFieldType(f.FieldHTMLType),
		}

		if f.DetailTable == 0 {
			// 主表字段
			result.MainFields = append(result.MainFields, fd)
		} else {
			// 明细表字段
			dt, exists := detailMap[f.DetailTable]
			if !exists {
				dt = &DetailTableDef{
					TableName:  fmt.Sprintf("%s_dt%d", wf.TableDBName, f.DetailTable),
					TableLabel: fmt.Sprintf("明细表%d", f.DetailTable),
					Fields:     make([]FieldDef, 0),
				}
				detailMap[f.DetailTable] = dt
			}
			dt.Fields = append(dt.Fields, fd)
		}
	}

	// 将明细表 map 转为有序切片
	for i := 1; i <= len(detailMap); i++ {
		if dt, ok := detailMap[i]; ok {
			result.DetailTables = append(result.DetailTables, *dt)
		}
	}

	return result, nil
}

// CheckUserPermission 检查用户在泛微 E9 中是否具有指定流程的审批权限。
// 通过 workflow_currentoperator 表查询用户是否有该流程的待办或已办记录。
func (a *Ecology9Adapter) CheckUserPermission(ctx context.Context, userID string, processType string) (bool, error) {
	// 先查询流程 ID
	var wf e9WorkflowBase
	err := a.db.WithContext(ctx).
		Where("workflowname = ?", processType).
		First(&wf).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, fmt.Errorf("查询流程失败: %w", err)
	}

	// 查询用户是否在该流程的操作人列表中（包含待办和已办）
	var count int64
	err = a.db.WithContext(ctx).
		Table("workflow_currentoperator").
		Where("workflowid = ? AND userid = ?", wf.ID, userID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("查询用户审批权限失败: %w", err)
	}

	return count > 0, nil
}

// FetchProcessData 拉取指定流程实例的业务数据。
// 通过 workflow_requestbase 关联表单主表和明细表查询。
func (a *Ecology9Adapter) FetchProcessData(ctx context.Context, processID string) (*ProcessData, error) {
	// 查询流程请求基本信息
	var requestInfo struct {
		RequestID    int    `gorm:"column:requestid"`
		WorkflowID   int    `gorm:"column:workflowid"`
		RequestName  string `gorm:"column:requestname"`
	}
	err := a.db.WithContext(ctx).
		Table("workflow_requestbase").
		Where("requestid = ?", processID).
		First(&requestInfo).Error
	if err != nil {
		return nil, fmt.Errorf("查询流程实例失败: %w", err)
	}

	// 查询流程对应的表单表名
	var wf e9WorkflowBase
	err = a.db.WithContext(ctx).
		Where("id = ?", requestInfo.WorkflowID).
		First(&wf).Error
	if err != nil {
		return nil, fmt.Errorf("查询流程定义失败: %w", err)
	}

	// 查询主表数据
	var mainData map[string]interface{}
	err = a.db.WithContext(ctx).
		Table(wf.TableDBName).
		Where("requestid = ?", processID).
		Take(&mainData).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("查询主表数据失败: %w", err)
	}

	// 查询明细表数据
	var detailData []map[string]interface{}
	// 获取明细表数量
	var detailCount int64
	a.db.WithContext(ctx).
		Table("workflow_billfield").
		Where("billid = ? AND detailtable > 0", wf.FormID).
		Distinct("detailtable").
		Count(&detailCount)

	for i := 1; i <= int(detailCount); i++ {
		dtTableName := fmt.Sprintf("%s_dt%d", wf.TableDBName, i)
		var rows []map[string]interface{}
		a.db.WithContext(ctx).
			Table(dtTableName).
			Where("mainid IN (SELECT id FROM "+wf.TableDBName+" WHERE requestid = ?)", processID).
			Find(&rows)
		for _, row := range rows {
			detailData = append(detailData, row)
		}
	}

	return &ProcessData{
		ProcessID:  processID,
		MainData:   mainData,
		DetailData: detailData,
	}, nil
}

// mapFieldType 将泛微 E9 的字段 HTML 类型映射为通用字段类型。
func (a *Ecology9Adapter) mapFieldType(htmlType string) string {
	switch htmlType {
	case "1":
		return "text"       // 单行文本
	case "2":
		return "textarea"   // 多行文本
	case "3":
		return "select"     // 下拉框
	case "4":
		return "checkbox"   // 复选框
	case "5":
		return "date"       // 日期
	case "6":
		return "number"     // 数字
	case "7":
		return "attachment" // 附件
	default:
		return "text"
	}
}
