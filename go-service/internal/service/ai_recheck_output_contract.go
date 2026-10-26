package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"auraoa/go-service/internal/model"
)

// recheckSchemaNode 只定义复核固定输出使用的结构；同一份定义用于展示、提示词和服务端校验。
type recheckSchemaNode struct {
	Dialect              string                        `json:"$schema,omitempty"`
	Type                 string                        `json:"type"`
	Properties           map[string]*recheckSchemaNode `json:"properties,omitempty"`
	Required             []string                      `json:"required,omitempty"`
	AdditionalProperties *bool                         `json:"additionalProperties,omitempty"`
	Items                *recheckSchemaNode            `json:"items,omitempty"`
	Enum                 []string                      `json:"enum,omitempty"`
	Minimum              *int                          `json:"minimum,omitempty"`
	Maximum              *int                          `json:"maximum,omitempty"`
	MinLength            int                           `json:"minLength,omitempty"`
	OneOf                []*recheckSchemaNode          `json:"oneOf,omitempty"`
	Const                *bool                         `json:"const,omitempty"`
}

func recheckOutputSchema(module string) *recheckSchemaNode {
	text := &recheckSchemaNode{Type: "string"}
	name := &recheckSchemaNode{Type: "string", MinLength: 1}
	boolean := &recheckSchemaNode{Type: "boolean"}
	zero, hundred := 0, 100
	percent := &recheckSchemaNode{Type: "integer", Minimum: &zero, Maximum: &hundred}
	array := func(item *recheckSchemaNode) *recheckSchemaNode {
		return &recheckSchemaNode{Type: "array", Items: item}
	}
	object := func(properties map[string]*recheckSchemaNode, required ...string) *recheckSchemaNode {
		if len(required) == 0 {
			for key := range properties {
				required = append(required, key)
			}
			sort.Strings(required)
		}
		additional := false
		return &recheckSchemaNode{Type: "object", Properties: properties, Required: required, AdditionalProperties: &additional}
	}
	properties := map[string]*recheckSchemaNode{
		"overall_score": percent, "confidence": percent,
		"risk_points": array(text), "suggestions": array(text),
		"deductions": array(object(map[string]*recheckSchemaNode{"item": name, "points": percent, "reason": text})),
	}
	required := []string{"overall_score", "confidence", "risk_points", "suggestions"}
	if module == "archive" {
		properties["overall_compliance"] = &recheckSchemaNode{Type: "string", Enum: []string{"compliant", "partially_compliant", "non_compliant"}}
		properties["flow_audit"] = object(map[string]*recheckSchemaNode{
			"is_complete": boolean, "missing_nodes": array(text),
			"node_results": array(object(map[string]*recheckSchemaNode{"node_id": text, "node_name": name, "compliant": boolean, "reasoning": text}, "node_name", "compliant", "reasoning")),
		})
		properties["field_audit"] = array(object(map[string]*recheckSchemaNode{"field_key": text, "field_name": name, "passed": boolean, "reasoning": text}, "field_name", "passed", "reasoning"))
		properties["rule_audit"] = array(object(map[string]*recheckSchemaNode{"rule_id": text, "rule_name": name, "passed": boolean, "reasoning": text}, "rule_name", "passed", "reasoning"))
		properties["ai_summary"] = text
		required = append(required, "overall_compliance", "flow_audit", "field_audit", "rule_audit", "ai_summary")
	} else {
		properties["recommendation"] = &recheckSchemaNode{Type: "string", Enum: []string{"approve", "review", "return"}}
		properties["rule_results"] = array(object(map[string]*recheckSchemaNode{"rule_content": name, "passed": boolean, "reason": text}))
		required = append(required, "recommendation", "rule_results")
	}
	result := object(properties, required...)
	yes, no := true, false
	root := &recheckSchemaNode{Type: "object", OneOf: []*recheckSchemaNode{
		object(map[string]*recheckSchemaNode{"reuse_original": {Type: "boolean", Const: &yes}}),
		object(map[string]*recheckSchemaNode{"reuse_original": {Type: "boolean", Const: &no}, "reason": name, "final_result": result}),
	}}
	root.Dialect = "https://json-schema.org/draft/2020-12/schema"
	return root
}

// aiRecheckOutputInstructions 是实际追加的完整原文，不包含仅供界面展示的解释。
func aiRecheckOutputInstructions(module string) string {
	schema, _ := json.MarshalIndent(recheckOutputSchema(module), "", "  ")
	conclusions := "recommendation只能为approve（80至100分）、review（60至79分）或return（0至59分）。"
	if module == "archive" {
		conclusions = "overall_compliance只能为compliant（80至100分）、partially_compliant（60至79分）或non_compliant（0至59分）。"
	}
	return `【固定返回格式】
仅返回一个符合下列JSON Schema的JSON对象。不要输出Markdown代码围栏、解释性前后缀、注释、占位符或省略号；禁止重复JSON键。
没有发现需要纠正的明确问题时，只返回{"reuse_original":true}，不输出reason或final_result。
确认需要修改时，返回reuse_original=false、非空reason及完整final_result三个字段；final_result不能是JSON字符串或只含修改项的对象。不得为补充说明或改写措辞而重写结果。
保留本次初审的所有结果字段、规则正文和已有节点、字段标识，不得增删、替换或合并已核验条目；可根据复核调整判断、说明、风险、建议、分数及结论。没有内容的数组使用[]，不得使用null。
分数、置信度和扣分写成0至100的整数，不使用字符串、小数或科学计数法。` + conclusions + `
deductions为可选扣分数组；初审已含该字段时必须保留。每项必须包含item、points、reason；提供时总扣分必须等于100减去overall_score。
【JSON Schema】
` + string(schema)
}

func withAIRecheckOutputInstructions(template model.SystemPromptTemplate, module string) model.SystemPromptTemplate {
	if template.Phase == "recheck" && template.PromptType == "system" {
		template.OutputInstructions = aiRecheckOutputInstructions(module)
	}
	return template
}

// decodeRecheckJSON 使用逐层解码拒绝重复键、非JSON前后缀及多个根值。
func decodeRecheckJSON(raw []byte) (interface{}, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var read func(int) (interface{}, error)
	read = func(depth int) (interface{}, error) {
		if depth > 32 {
			return nil, fmt.Errorf("复核JSON嵌套过深")
		}
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		switch token {
		case json.Delim('{'):
			object := map[string]interface{}{}
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyToken.(string)
				if !ok {
					return nil, fmt.Errorf("复核JSON键不是字符串")
				}
				if _, exists := object[key]; exists {
					return nil, fmt.Errorf("复核JSON存在重复键：%s", key)
				}
				value, err := read(depth + 1)
				if err != nil {
					return nil, err
				}
				object[key] = value
			}
			_, err = decoder.Token()
			return object, err
		case json.Delim('['):
			array := []interface{}{}
			for decoder.More() {
				value, err := read(depth + 1)
				if err != nil {
					return nil, err
				}
				array = append(array, value)
			}
			_, err = decoder.Token()
			return array, err
		default:
			return token, nil
		}
	}
	value, err := read(0)
	if err != nil {
		return nil, fmt.Errorf("复核JSON格式无效：%w", err)
	}
	if _, err = decoder.Token(); err != io.EOF {
		return nil, fmt.Errorf("复核仅允许返回一个JSON对象")
	}
	return value, nil
}

// validateRecheckSchema 按固定结构中实际使用的类型、必填项、枚举与范围逐层校验。
func validateRecheckSchema(value interface{}, schema *recheckSchemaNode, path string) error {
	invalid := func() error { return fmt.Errorf("复核返回字段不符合固定格式：%s", path) }
	if len(schema.OneOf) > 0 {
		matched := 0
		for _, candidate := range schema.OneOf {
			if validateRecheckSchema(value, candidate, path) == nil {
				matched++
			}
		}
		if matched != 1 {
			return fmt.Errorf("复核返回必须明确选择沿用初审或提供完整修改结果")
		}
	}
	switch schema.Type {
	case "object":
		object, ok := value.(map[string]interface{})
		if !ok {
			return invalid()
		}
		for _, key := range schema.Required {
			if _, exists := object[key]; !exists {
				return fmt.Errorf("复核缺少必填字段：%s.%s", path, key)
			}
		}
		for key, item := range object {
			child, known := schema.Properties[key]
			if !known {
				if schema.AdditionalProperties != nil && !*schema.AdditionalProperties {
					return fmt.Errorf("复核包含未定义字段：%s.%s", path, key)
				}
				continue
			}
			if err := validateRecheckSchema(item, child, path+"."+key); err != nil {
				return err
			}
		}
	case "array":
		array, ok := value.([]interface{})
		if !ok {
			return invalid()
		}
		for index, item := range array {
			if err := validateRecheckSchema(item, schema.Items, fmt.Sprintf("%s[%d]", path, index)); err != nil {
				return err
			}
		}
	case "string":
		text, ok := value.(string)
		if !ok || utf8.RuneCountInString(text) < schema.MinLength {
			return invalid()
		}
		if len(schema.Enum) > 0 {
			for _, allowed := range schema.Enum {
				if text == allowed {
					return nil
				}
			}
			return invalid()
		}
	case "boolean":
		boolean, ok := value.(bool)
		if !ok || (schema.Const != nil && boolean != *schema.Const) {
			return invalid()
		}
	case "integer":
		number, ok := value.(json.Number)
		if !ok {
			return invalid()
		}
		integer, err := strconv.ParseInt(number.String(), 10, 64)
		if err != nil || (schema.Minimum != nil && integer < int64(*schema.Minimum)) || (schema.Maximum != nil && integer > int64(*schema.Maximum)) {
			return invalid()
		}
	}
	return nil
}

func validateAIRecheckEnvelope(raw []byte, module string) error {
	value, err := decodeRecheckJSON(raw)
	if err != nil {
		return err
	}
	if err := validateRecheckSchema(value, recheckOutputSchema(module), "$"); err != nil {
		return err
	}
	object := value.(map[string]interface{})
	if !object["reuse_original"].(bool) && strings.TrimSpace(object["reason"].(string)) == "" {
		return fmt.Errorf("复核缺少说明")
	}
	return nil
}
