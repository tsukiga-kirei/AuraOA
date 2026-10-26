package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// getToolModelName 将包含特殊字符的工具键转换为 ASCII 函数别名（与 ExecuteMessageStream 中 toolAliases 规则一致）
func getToolModelName(toolCode string) string {
	if strings.Contains(toolCode, ":") {
		sum := sha256.Sum256([]byte(toolCode))
		return "tool_" + hex.EncodeToString(sum[:16])
	}
	return toolCode
}

// CompactToolPayload 将工具执行结果进行确定性精简与字段投影，供后续多轮对话上下文复用。
// 仅保留核心业务字段与关键事实，剔除冗余展示配置与超长格式，防止多轮对话 Token 膨胀并避免幻觉。
func CompactToolPayload(toolCode string, payload interface{}) string {
	if payload == nil {
		return "{}"
	}

	// 1. 若本身已是字符串
	if str, ok := payload.(string); ok {
		str = strings.TrimSpace(str)
		if len([]rune(str)) > 600 {
			return string([]rune(str)[:600]) + "...(已截断)"
		}
		return str
	}

	// 2. 将复杂结构统一转为字节序列方便通用剪裁
	dataBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf("%v", payload)
	}

	// 尝试解析为 map
	var rawMap map[string]interface{}
	if err := json.Unmarshal(dataBytes, &rawMap); err == nil {
		return compactMapPayload(toolCode, rawMap)
	}

	// 尝试解析为 slice
	var rawSlice []interface{}
	if err := json.Unmarshal(dataBytes, &rawSlice); err == nil {
		cleaned := compactSlicePayload(rawSlice, 1)
		out, _ := json.Marshal(cleaned)
		return string(out)
	}

	// 兜底长度限制
	res := string(dataBytes)
	if len([]rune(res)) > 800 {
		return string([]rune(res)[:800]) + "...(已截断)"
	}
	return res
}

func compactMapPayload(toolCode string, m map[string]interface{}) string {
	// 针对列表类工具 (待办、我发起的流程、搜索结果等)
	if itemsRaw, ok := m["items"]; ok {
		if itemsList, ok := itemsRaw.([]interface{}); ok {
			compactedItems := make([]map[string]interface{}, 0, 5)
			limit := 5
			if len(itemsList) < limit {
				limit = len(itemsList)
			}
			for i := 0; i < limit; i++ {
				if itemMap, ok := itemsList[i].(map[string]interface{}); ok {
					item := make(map[string]interface{})
					for _, k := range []string{
						"request_id", "process_id", "request_name", "title",
						"applicant", "creator_name", "created_at", "created_date", "submit_time", "department", "workflow_code",
						"status", "current_node", "node_name", "amount", "total_amount",
					} {
						if v, exists := itemMap[k]; exists && v != nil && v != "" {
							item[k] = v
						}
					}
					if len(item) == 0 {
						// 若无预设字段，做通用清理取前 5 个非空键
						count := 0
						for _, k := range sortedPayloadKeys(itemMap) {
							v := itemMap[k]
							if count >= 5 {
								break
							}
							if v != nil && v != "" {
								item[k] = cleanGenericValue(v, 1)
								count++
							}
						}
					}
					compactedItems = append(compactedItems, item)
				}
			}
			resMap := map[string]interface{}{
				"items": compactedItems,
				"total": m["total"],
			}
			if len(itemsList) > limit {
				resMap["_note"] = fmt.Sprintf("共查询到 %v 条记录，为控制上下文仅保留前 %d 条", m["total"], limit)
			}
			out, _ := json.Marshal(resMap)
			return string(out)
		}
	}

	// 针对获取流程详情 get_process
	if toolCode == "get_process" {
		resMap := make(map[string]interface{})
		for _, k := range []string{"process_id", "request_id", "request_name", "creator_name", "title", "applicant", "department", "process_type", "workflow_code", "status", "current_node", "node_name", "form_title"} {
			if v, ok := m[k]; ok && v != nil && v != "" {
				resMap[k] = v
			}
		}
		// 真实工具以 main_fields 数组返回主表字段，保留字段和值的对应关系。
		if fields, ok := m["main_fields"].([]interface{}); ok {
			kept := make([]interface{}, 0, len(fields))
			for i, field := range fields {
				if i >= 30 {
					kept = append(kept, fmt.Sprintf("其余字段已折叠，共%d项", len(fields)))
					break
				}
				kept = append(kept, cleanGenericValue(field, 0))
			}
			resMap["main_fields"] = kept
		}
		// 兼容旧工具的 form_data 返回格式。
		if formData, ok := m["form_data"].(map[string]interface{}); ok {
			cleanedForm := make(map[string]interface{})
			fieldCount := 0
			for _, k := range sortedPayloadKeys(formData) {
				v := formData[k]
				if fieldCount >= 30 {
					cleanedForm["_more_fields"] = "更多表单字段已折叠"
					break
				}
				if v != nil && v != "" {
					cleanedForm[k] = cleanGenericValue(v, 1)
					fieldCount++
				}
			}
			resMap["form_data"] = cleanedForm
		}
		// 明细表瘦身 (每个明细表最多取 3 行)
		if detailTables, ok := m["detail_tables"].(map[string]interface{}); ok {
			cleanedDetails := make(map[string]interface{})
			for tblName, tblData := range detailTables {
				if rowList, ok := tblData.([]interface{}); ok {
					subLimit := 3
					if len(rowList) < subLimit {
						subLimit = len(rowList)
					}
					cleanedRows := make([]interface{}, 0, subLimit)
					for r := 0; r < subLimit; r++ {
						cleanedRows = append(cleanedRows, cleanGenericValue(rowList[r], 1))
					}
					cleanedDetails[tblName] = cleanedRows
				}
			}
			resMap["detail_tables"] = cleanedDetails
		}
		// 附件列表只保留名称
		if attachList, ok := m["attachments"].([]interface{}); ok {
			names := make([]string, 0, 5)
			for idx, at := range attachList {
				if idx >= 5 {
					names = append(names, fmt.Sprintf("...等共 %d 个附件", len(attachList)))
					break
				}
				if name, ok := at.(string); ok {
					names = append(names, name)
				} else if atMap, ok := at.(map[string]interface{}); ok {
					if n, ok := atMap["name"].(string); ok && n != "" {
						names = append(names, n)
					} else if n, ok := atMap["filename"].(string); ok && n != "" {
						names = append(names, n)
					}
				}
			}
			resMap["attachments"] = names
		}
		out, _ := json.Marshal(resMap)
		return string(out)
	}

	// 针对审批流 get_approval_flow
	if toolCode == "get_approval_flow" {
		resMap := make(map[string]interface{})
		for _, k := range []string{"process_id", "request_name", "workflow_code", "status"} {
			if v, ok := m[k]; ok && v != nil && v != "" {
				resMap[k] = v
			}
		}
		// 处理 nodes / history
		var steps []interface{}
		if nodes, ok := m["nodes"].([]interface{}); ok {
			steps = nodes
		} else if history, ok := m["history"].([]interface{}); ok {
			steps = history
		}
		if len(steps) > 0 {
			cleanedSteps := make([]map[string]interface{}, 0, 8)
			stepLimit := 8
			if len(steps) < stepLimit {
				stepLimit = len(steps)
			}
			for i := 0; i < stepLimit; i++ {
				if sMap, ok := steps[i].(map[string]interface{}); ok {
					step := make(map[string]interface{})
					for _, k := range []string{"node_name", "operator_name", "action", "operate_time", "operated_at"} {
						if v, ok := sMap[k]; ok && v != nil && v != "" {
							step[k] = v
						}
					}
					if op, ok := sMap["opinion"].(string); ok && op != "" {
						opRunes := []rune(op)
						if len(opRunes) > 60 {
							op = string(opRunes[:60]) + "..."
						}
						step["opinion"] = op
					}
					cleanedSteps = append(cleanedSteps, step)
				}
			}
			resMap["approval_history"] = cleanedSteps
		}
		out, _ := json.Marshal(resMap)
		return string(out)
	}

	// 针对审核/总结结果
	if toolCode == "get_latest_audit" || toolCode == "get_latest_summary" {
		resMap := make(map[string]interface{})
		for _, k := range []string{"process_id", "has_audit", "has_summary", "message", "recommendation", "score", "created_at", "conclusion", "status", "summary", "risk_level", "risk_count"} {
			if v, ok := m[k]; ok && v != nil && v != "" {
				resMap[k] = v
			}
		}
		// 审核与总结正文位于独立结果对象内，保留评估依据及分块正文。
		for _, key := range []string{"audit_result", "summary_result"} {
			if value, ok := m[key]; ok {
				resMap[key] = cleanGenericValue(value, 0)
			}
		}
		out, _ := json.Marshal(resMap)
		return string(out)
	}

	// 通用兜底清理 (适用于 MCP 工具与未知工具)
	cleaned := cleanGenericMap(m, 0)
	out, _ := json.Marshal(cleaned)
	if len(out) > 1200 {
		return truncateUTF8Bytes(string(out), 1200) + "...(已截断)"
	}
	return string(out)
}

func cleanGenericValue(v interface{}, depth int) interface{} {
	if depth > 3 {
		return "..."
	}
	switch val := v.(type) {
	case string:
		runes := []rune(val)
		if len(runes) > 200 {
			return string(runes[:200]) + "...[截断]"
		}
		return val
	case map[string]interface{}:
		return cleanGenericMap(val, depth+1)
	case []interface{}:
		return compactSlicePayload(val, depth+1)
	default:
		return val
	}
}

func cleanGenericMap(m map[string]interface{}, depth int) map[string]interface{} {
	if depth > 3 {
		return map[string]interface{}{"...": "..."}
	}
	cleaned := make(map[string]interface{})
	count := 0
	for _, k := range sortedPayloadKeys(m) {
		v := m[k]
		if count >= 20 {
			cleaned["_more"] = "..."
			break
		}
		if v == nil || v == "" {
			continue
		}
		cleaned[k] = cleanGenericValue(v, depth)
		count++
	}
	return cleaned
}

func compactSlicePayload(slice []interface{}, depth int) []interface{} {
	limit := 5
	if len(slice) < limit {
		limit = len(slice)
	}
	cleaned := make([]interface{}, 0, limit+1)
	for i := 0; i < limit; i++ {
		cleaned = append(cleaned, cleanGenericValue(slice[i], depth))
	}
	if len(slice) > limit {
		cleaned = append(cleaned, fmt.Sprintf("...等共 %d 项", len(slice)))
	}
	return cleaned
}

// sortedPayloadKeys 保证通用字段裁剪顺序可重复。
func sortedPayloadKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
