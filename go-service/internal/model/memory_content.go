package model

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"
)

// CanonicalQueryDefinition 将查询定义补齐兼容默认值后规范化；SQL 原文与 builder 扩展字段保持原样。
func CanonicalQueryDefinition(raw []byte) ([]byte, error) {
	var def AuditDataQueryDefinition
	if err := json.Unmarshal(raw, &def); err != nil {
		return nil, err
	}
	var builder map[string]interface{}
	if len(def.Builder) > 0 && string(bytes.TrimSpace(def.Builder)) != "null" {
		if err := decodeMemoryJSON(def.Builder, &builder); err != nil {
			return nil, err
		}
	}
	if def.Mode == "" {
		def.Mode = "sql"
		if builder != nil && builder["sql_mode"] != true {
			def.Mode = "config"
		}
	}
	if def.Mode == "sql" {
		def.Builder = nil
		builder = nil
	} else if builder != nil {
		delete(builder, "sql_mode")
		fillMemoryDefaults(builder, map[string]interface{}{"source_type": "table", "table": "", "columns": []interface{}{}, "filters": []interface{}{}, "order_alias": "", "order_dir": "DESC"})
		archived := builder["source_type"] == "workflow"
		fillMemoryDefaults(builder, map[string]interface{}{"archived_only": archived})
		for _, key := range []string{"columns", "filters"} {
			if rows, ok := builder[key].([]interface{}); ok {
				for _, row := range rows {
					if fields, ok := row.(map[string]interface{}); ok && key == "columns" {
						fillMemoryDefaults(fields, map[string]interface{}{"field": "", "alias": "", "aggregate": ""})
					}
				}
			}
		}
		encoded, err := json.Marshal(builder)
		if err != nil {
			return nil, err
		}
		def.Builder = encoded
	}
	if def.Tables == nil {
		def.Tables = []string{}
	}
	if def.Parameters == nil {
		def.Parameters = []AuditDataQueryParameter{}
	}
	if def.AIContext == nil {
		module := "general"
		if builder["source_type"] == "workflow" {
			module = "workflow"
		}
		def.AIContext = &AuditDataQueryAIContext{OAModule: module}
	}
	encoded, err := json.Marshal(def)
	if err != nil {
		return nil, err
	}
	var content interface{}
	if err := decodeMemoryJSON(encoded, &content); err != nil {
		return nil, err
	}
	return json.Marshal(content)
}

// EqualQueryDefinitions 只比较定义内容，不含分类、预览时间、版本号等条目元数据。
func EqualQueryDefinitions(a, b []byte) bool {
	left, err := CanonicalQueryDefinition(a)
	if err != nil {
		return false
	}
	right, err := CanonicalQueryDefinition(b)
	return err == nil && bytes.Equal(left, right)
}

// CanonicalFeedbackMemoryItems 保留条目顺序，来源引用按集合比较；空条件与未配置条件等价。
func CanonicalFeedbackMemoryItems(raw []byte) ([]byte, error) {
	var items []FeedbackMemoryItem
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, err
		}
	}
	if items == nil {
		items = []FeedbackMemoryItem{}
	}
	for i := range items {
		items[i].Condition = strings.TrimSpace(items[i].Condition)
		items[i].Content = strings.TrimSpace(items[i].Content)
		items[i].SourceRefs = append([]string{}, items[i].SourceRefs...)
		sort.Strings(items[i].SourceRefs)
	}
	return json.Marshal(items)
}

// EqualFeedbackMemoryItems 比较管理员可编辑的记忆正文及来源，不比较生成和编辑时间。
func EqualFeedbackMemoryItems(a, b []byte) bool {
	left, err := CanonicalFeedbackMemoryItems(a)
	if err != nil {
		return false
	}
	right, err := CanonicalFeedbackMemoryItems(b)
	return err == nil && bytes.Equal(left, right)
}

func decodeMemoryJSON(raw []byte, value interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	return decoder.Decode(value)
}

func fillMemoryDefaults(value, defaults map[string]interface{}) {
	for key, fallback := range defaults {
		if value[key] == nil {
			value[key] = fallback
		}
	}
}
