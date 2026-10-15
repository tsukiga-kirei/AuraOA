package service

import (
	"auraoa/go-service/internal/model"
	"bytes"
	"encoding/json"
	"gorm.io/datatypes"
)

// sourceContentFingerprint 仅规范化内容 JSON，不改变执行快照和 OA 数据变化指纹的既有语义。
func sourceContentFingerprint(value interface{}) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	var normalized interface{}
	if decodeContentJSON(raw, &normalized) != nil {
		return ""
	}
	return stableJSONFingerprint(normalized)
}

// decodeContentJSON 保留查询固定参数等 JSON 数值精度，避免规范化时经过 float64 丢失内容。
func decodeContentJSON(raw []byte, target interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	return decoder.Decode(target)
}

// normalizedAIConfigContent 补齐旧配置缺省字段，保留未知扩展字段以免漏报真实修改。
func normalizedAIConfigContent(raw datatypes.JSON) map[string]interface{} {
	return normalizedObjectContent(raw, model.AIConfigData{})
}

func normalizedPermissionContent(raw datatypes.JSON) map[string]interface{} {
	return normalizedObjectContent(raw, model.ArchiveUserPermissionsData{})
}

func normalizedObjectContent(raw datatypes.JSON, defaults interface{}) map[string]interface{} {
	encoded, _ := json.Marshal(defaults)
	result := map[string]interface{}{}
	_ = json.Unmarshal(encoded, &result)
	var values map[string]interface{}
	if decodeContentJSON(raw, &values) == nil {
		for key, value := range values {
			if value != nil {
				result[key] = value
			}
		}
	}
	return result
}

// executionConfigVersionChanged 只比较执行所依据的租户发布版本号，不比较内容指纹或版本大小。
func executionConfigVersionChanged(execution *model.ExecutionConfigVersion, published *model.TenantConfigVersion) bool {
	if execution == nil || published == nil {
		return false
	}
	used := executionVersionNumber(execution)
	return used != nil && *used != published.VersionNo
}
