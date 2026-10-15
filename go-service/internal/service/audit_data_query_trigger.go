package service

import (
	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
	"auraoa/go-service/internal/pkg/oa"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var queryTriggerField = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*:[A-Za-z_][A-Za-z0-9_]*$`)
var queryTriggerNumber = regexp.MustCompile(`^[+-]?(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+-]?\d+)?$`)
var queryTriggerGroupedNumber = regexp.MustCompile(`^[+-]?\d{1,3}(?:,\d{3})+(?:\.\d+)?$`)

// validateQueryTrigger 校验单字段条件，不接收表达式或 SQL，关闭时沿用原执行行为。
func validateQueryTrigger(trigger *model.AuditDataQueryTrigger) error {
	if trigger == nil {
		return nil
	}
	if !queryTriggerField.MatchString(trigger.Field) || len(trigger.Field) > 200 || len([]rune(trigger.FieldLabel)) > 200 {
		return fmt.Errorf("请选择有效的查询条件字段")
	}
	allowed := map[string]string{
		"text":    "eq ne contains not_contains is_empty not_empty",
		"number":  "eq ne gt gte lt lte is_empty not_empty",
		"date":    "eq ne gt gte lt lte is_empty not_empty",
		"boolean": "eq ne is_empty not_empty",
		"choice":  "contains not_contains is_empty not_empty",
		"file":    "is_empty not_empty",
	}
	ops, ok := allowed[trigger.ValueType]
	if !ok || !strings.Contains(" "+ops+" ", " "+trigger.Operator+" ") {
		return fmt.Errorf("查询条件的字段类型与比较方式不匹配")
	}
	if trigger.Operator == "is_empty" || trigger.Operator == "not_empty" {
		return nil
	}
	if strings.TrimSpace(trigger.Value) == "" || len(trigger.Value) > 4000 {
		return fmt.Errorf("请填写查询条件的比较值，最多4000字节")
	}
	_, err := queryTriggerComparable(trigger.ValueType, trigger.Value)
	return err
}

// queryTriggerValues 区分字段缺失与正常空值，保留明细行和多值边界。
func queryTriggerValues(pd *oa.ProcessData, fieldRef string) ([]interface{}, error) {
	if pd == nil {
		return nil, fmt.Errorf("无法读取查询条件的流程字段")
	}
	table, field := parseExternalFieldRef(fieldRef)
	if strings.EqualFold(table, "main") {
		value, ok := findValueCaseInsensitive(pd.MainData, field)
		if !ok {
			return nil, fmt.Errorf("查询条件字段不存在：%s", fieldRef)
		}
		return []interface{}{value}, nil
	}
	for name, rows := range pd.DetailTables {
		if !strings.EqualFold(name, table) {
			continue
		}
		values := make([]interface{}, 0, len(rows))
		for _, row := range rows {
			value, ok := findValueCaseInsensitive(row, field)
			if !ok {
				return nil, fmt.Errorf("查询条件明细字段不存在：%s", fieldRef)
			}
			values = append(values, value)
		}
		return values, nil
	}
	return nil, fmt.Errorf("查询条件明细表不存在：%s", table)
}

// flattenQueryTriggerValue 只拆数组；普通文本中的逗号不被误认为多选分隔符。
func flattenQueryTriggerValue(value interface{}, choice bool) []interface{} {
	if value == nil {
		return nil
	}
	if raw, ok := value.([]byte); ok {
		value = string(raw)
	}
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		var out []interface{}
		for i := 0; i < v.Len(); i++ {
			out = append(out, flattenQueryTriggerValue(v.Index(i).Interface(), choice)...)
		}
		return out
	}
	if choice {
		if text, ok := value.(string); ok {
			var out []interface{}
			for _, item := range strings.Split(text, ",") {
				if strings.TrimSpace(item) != "" {
					out = append(out, strings.TrimSpace(item))
				}
			}
			return out
		}
	}
	return []interface{}{value}
}

func queryTriggerEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text) == ""
	}
	return false // 0 和 false 是有效值。
}

func queryTriggerComparable(valueType string, value interface{}) (interface{}, error) {
	if date, ok := value.(time.Time); ok {
		value = date.In(apptime.Location()).Format("2006-01-02")
	}
	text := strings.TrimSpace(fmt.Sprint(value))
	switch valueType {
	case "number":
		if len(text) > 1000 {
			return nil, fmt.Errorf("查询条件数字过长")
		}
		if queryTriggerGroupedNumber.MatchString(text) {
			text = strings.ReplaceAll(text, ",", "")
		}
		if !queryTriggerNumber.MatchString(text) {
			return nil, fmt.Errorf("查询条件要求有效数字")
		}
		if index := strings.IndexAny(text, "eE"); index >= 0 {
			exponent, err := strconv.Atoi(text[index+1:])
			if err != nil || exponent < -1000 || exponent > 1000 {
				return nil, fmt.Errorf("查询条件数字指数超出范围")
			}
		}
		number, ok := new(big.Rat).SetString(text)
		if !ok {
			return nil, fmt.Errorf("查询条件要求有效数字")
		}
		return number, nil
	case "date":
		for _, layout := range []string{"2006-01-02", "2006/01/02", time.RFC3339Nano, "2006-01-02 15:04:05", "2006/01/02 15:04:05"} {
			date, err := time.ParseInLocation(layout, text, apptime.Location())
			if err == nil {
				return date.In(apptime.Location()).Format("2006-01-02"), nil
			}
		}
		return nil, fmt.Errorf("查询条件要求有效日期，格式为 YYYY-MM-DD")
	case "boolean":
		switch strings.ToLower(text) {
		case "1", "true", "yes", "是":
			return true, nil
		case "0", "false", "no", "否":
			return false, nil
		}
		return nil, fmt.Errorf("查询条件要求有效的是/否值")
	case "text", "choice":
		return text, nil
	}
	return nil, fmt.Errorf("查询条件字段类型无效")
}

// evaluateQueryTrigger 数字精确比较，日期按业务自然日，多选按完整编码匹配。
// 正向条件任一值命中即可；否定条件要求所有非空值均不命中，空集合不满足值比较。
func evaluateQueryTrigger(trigger *model.AuditDataQueryTrigger, pd *oa.ProcessData) (bool, error) {
	if err := validateQueryTrigger(trigger); err != nil {
		return false, err
	}
	if trigger == nil {
		return true, nil
	}
	raw, err := queryTriggerValues(pd, trigger.Field)
	if err != nil {
		return false, err
	}
	var values []interface{}
	for _, row := range raw {
		for _, value := range flattenQueryTriggerValue(row, trigger.ValueType == "choice") {
			if !queryTriggerEmpty(value) {
				values = append(values, value)
			}
		}
	}
	if trigger.Operator == "is_empty" {
		return len(values) == 0, nil
	}
	if trigger.Operator == "not_empty" {
		return len(values) > 0, nil
	}
	if len(values) == 0 {
		return false, nil
	}
	expected, err := queryTriggerComparable(trigger.ValueType, trigger.Value)
	if err != nil {
		return false, err
	}
	matched := false
	for _, value := range values {
		actual, err := queryTriggerComparable(trigger.ValueType, value)
		if err != nil {
			return false, fmt.Errorf("字段 %s 无法按所选类型判断：%w", trigger.Field, err)
		}
		cmp := 0
		switch item := actual.(type) {
		case *big.Rat:
			cmp = item.Cmp(expected.(*big.Rat))
		case string:
			cmp = strings.Compare(item, expected.(string))
		case bool:
			if item != expected.(bool) {
				cmp = 1
			}
		}
		hit := false
		switch trigger.Operator {
		case "eq", "ne":
			hit = cmp == 0
		case "gt":
			hit = cmp > 0
		case "gte":
			hit = cmp >= 0
		case "lt":
			hit = cmp < 0
		case "lte":
			hit = cmp <= 0
		case "contains", "not_contains":
			if trigger.ValueType == "choice" {
				hit = cmp == 0
			} else {
				hit = strings.Contains(actual.(string), expected.(string))
			}
		}
		matched = matched || hit
	}
	if trigger.Operator == "ne" || trigger.Operator == "not_contains" {
		return !matched, nil
	}
	return matched, nil
}

// prepareTriggeredQueryInputs 在取数绑定之前判断条件，未命中时不要求查询参数。
func prepareTriggeredQueryInputs(def model.AuditDataQueryDefinition, bindings map[string]model.AuditDataQueryBinding, trigger *model.AuditDataQueryTrigger, pd *oa.ProcessData, processID string) (map[string]interface{}, *bool, error) {
	var matched *bool
	if trigger != nil {
		value, err := evaluateQueryTrigger(trigger, pd)
		if err != nil {
			return nil, nil, err
		}
		matched = &value
		if !value {
			return map[string]interface{}{}, matched, nil
		}
	}
	values, err := resolveQueryBindings(def, bindings, pd, processID)
	return values, matched, err
}

func queryEvidenceSkipped(e QueryEvidence) bool {
	return e.Status == "skipped" && e.Trigger != nil && e.TriggerMatched != nil && !*e.TriggerMatched
}

func formatQueryTrigger(trigger *model.AuditDataQueryTrigger) string {
	if trigger == nil {
		return ""
	}
	raw, _ := json.Marshal(trigger)
	return "查询触发条件（业务配置）：" + string(raw) + "\n"
}
