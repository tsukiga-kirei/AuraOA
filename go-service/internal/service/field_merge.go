package service

import (
	"encoding/json"
	"strings"

	"gorm.io/datatypes"

	"oa-smart-audit/go-service/internal/dto"
)

// ── 字段合并输入结构 ──

// rawFieldItem 租户配置中的原始字段定义。
type rawFieldItem struct {
	FieldKey  string `json:"field_key"`
	FieldName string `json:"field_name"`
	FieldType string `json:"field_type"`
	Selected  bool   `json:"selected"`
}

// rawDetailTable 租户配置中的原始明细表定义。
type rawDetailTable struct {
	TableName  string         `json:"table_name"`
	TableLabel string         `json:"table_label"`
	Fields     []rawFieldItem `json:"fields"`
}

// FieldMergeInput 字段合并所需的全部输入参数。
type FieldMergeInput struct {
	FieldMode         string         // "all" | "selected"
	MainFieldsJSON    datatypes.JSON // 租户主表字段 JSON
	DetailTablesJSON  datatypes.JSON // 租户明细表 JSON
	UserOverrides     []string       // 用户额外选中的字段列表（格式 "table:field_key" 或 "field_key"）
	AllowCustomFields bool           // 用户是否有权自定义字段
}

// FieldMergeResult 字段合并的输出结果。
type FieldMergeResult struct {
	MainFields   []dto.TenantFieldDTO
	DetailTables []dto.DetailTableDTO
}

// ── 字段合并核心逻辑 ──

// MergeFields 执行字段合并，将租户配置与用户个性化覆盖合并为最终生效的字段列表。
//
// 合并规则：
//   - field_mode = "all"：所有字段强制选中且锁定，用户不可取消
//   - field_mode = "selected"：租户选中的字段锁定，用户只能在此基础上新增（不能减少）
func MergeFields(input FieldMergeInput) FieldMergeResult {
	var mainFields []rawFieldItem
	var detailTables []rawDetailTable
	_ = json.Unmarshal(input.MainFieldsJSON, &mainFields)
	_ = json.Unmarshal(input.DetailTablesJSON, &detailTables)

	// 构建用户额外选中字段索引
	userAddedMap := buildUserFieldMap(input.UserOverrides, input.AllowCustomFields)

	// 合并主表字段
	mergedMain := make([]dto.TenantFieldDTO, len(mainFields))
	for i, f := range mainFields {
		locked := input.FieldMode == "all" || f.Selected
		selected := locked || (userAddedMap["main"] != nil && userAddedMap["main"][f.FieldKey])
		mergedMain[i] = dto.TenantFieldDTO{
			FieldKey:  f.FieldKey,
			FieldName: f.FieldName,
			FieldType: f.FieldType,
			Selected:  selected,
			Locked:    locked,
		}
	}

	// 合并明细表字段
	mergedDetails := make([]dto.DetailTableDTO, len(detailTables))
	for i, dt := range detailTables {
		fields := make([]dto.TenantFieldDTO, len(dt.Fields))
		for j, f := range dt.Fields {
			locked := input.FieldMode == "all" || f.Selected
			selected := locked || (userAddedMap[dt.TableName] != nil && userAddedMap[dt.TableName][f.FieldKey])
			fields[j] = dto.TenantFieldDTO{
				FieldKey:  f.FieldKey,
				FieldName: f.FieldName,
				FieldType: f.FieldType,
				Selected:  selected,
				Locked:    locked,
			}
		}
		mergedDetails[i] = dto.DetailTableDTO{
			TableName:  dt.TableName,
			TableLabel: dt.TableLabel,
			Fields:     fields,
		}
	}

	return FieldMergeResult{
		MainFields:   mergedMain,
		DetailTables: mergedDetails,
	}
}

// buildUserFieldMap 将用户字段覆盖列表解析为 table -> fieldKey -> true 的索引。
// allowCustomFields 为 false 时返回空 map（用户无权自定义）。
func buildUserFieldMap(overrides []string, allowCustomFields bool) map[string]map[string]bool {
	result := make(map[string]map[string]bool)
	if !allowCustomFields {
		return result
	}
	for _, fo := range overrides {
		table, key := parseFieldOverrideKey(fo)
		if result[table] == nil {
			result[table] = make(map[string]bool)
		}
		result[table][key] = true
	}
	return result
}

// parseFieldOverrideKey 解析字段覆盖键，格式为 "table:field_key" 或 "field_key"（默认 main 表）。
func parseFieldOverrideKey(fo string) (string, string) {
	if strings.Contains(fo, ":") {
		parts := strings.SplitN(fo, ":", 2)
		return parts[0], parts[1]
	}
	return "main", fo
}
