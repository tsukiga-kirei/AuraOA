package oa

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

// OAContextAnchor 审核完成时 OA 流程上下文锚点，用于判断结论是否过期。
type OAContextAnchor struct {
	LastReturnLogID             int64             `json:"last_return_log_id"`
	FlowRevision                int64             `json:"flow_revision"`
	LastResubmitLogID           int64             `json:"last_resubmit_log_id,omitempty"`
	LastLogID                   int64             `json:"last_log_id,omitempty"`
	LastLogType                 string            `json:"last_log_type,omitempty"`
	CurrentNodeID               int               `json:"current_node_id"`
	ContentFingerprint          string            `json:"content_fingerprint,omitempty"`
	AttachmentFingerprint       string            `json:"attachment_fingerprint,omitempty"`
	AttachmentFieldFingerprints map[string]string `json:"attachment_field_fingerprints,omitempty"`
	ExecutionFingerprint        string            `json:"execution_fingerprint,omitempty"`
}

// IsEmpty 历史数据或未写入锚点时视为空。
func (a OAContextAnchor) IsEmpty() bool {
	return a.FlowRevision == 0 &&
		a.LastReturnLogID == 0 &&
		a.LastLogID == 0 &&
		a.CurrentNodeID == 0 &&
		a.ContentFingerprint == "" &&
		a.AttachmentFingerprint == ""
}

// OAContextChanges 描述两次 OA 上下文之间的变化类型。
type OAContextChanges struct {
	LegacyAnchor           bool `json:"legacy_anchor"`
	ExecutionConfigChanged bool `json:"execution_config_changed"`
	DataChanged            bool `json:"data_changed"`
	AttachmentChanged      bool `json:"attachment_changed"`
	ReturnResubmitChanged  bool `json:"return_resubmit_changed"`
	FlowChanged            bool `json:"flow_changed"`
	CurrentNodeChanged     bool `json:"current_node_changed"`
}

// Any 返回是否存在任意需要关注的 OA 上下文变化。
func (c OAContextChanges) Any() bool {
	return c.LegacyAnchor ||
		c.ExecutionConfigChanged ||
		c.DataChanged ||
		c.AttachmentChanged ||
		c.ReturnResubmitChanged ||
		c.FlowChanged ||
		c.CurrentNodeChanged
}

// CompareContextAnchors 将业务数据、附件版本、退回重提和普通审批流变化分开比较。
func CompareContextAnchors(stored, current OAContextAnchor) OAContextChanges {
	if stored.IsEmpty() {
		return OAContextChanges{LegacyAnchor: true}
	}
	changes := OAContextChanges{
		ReturnResubmitChanged: current.LastReturnLogID > stored.LastReturnLogID ||
			current.LastResubmitLogID > stored.LastResubmitLogID,
		CurrentNodeChanged: stored.CurrentNodeID != 0 &&
			current.CurrentNodeID != 0 &&
			stored.CurrentNodeID != current.CurrentNodeID,
	}
	if stored.ContentFingerprint != "" && current.ContentFingerprint != "" {
		changes.DataChanged = stored.ContentFingerprint != current.ContentFingerprint
	}
	if stored.AttachmentFingerprint != "" || current.AttachmentFingerprint != "" {
		changes.AttachmentChanged = stored.AttachmentFingerprint != current.AttachmentFingerprint
	}
	if stored.ExecutionFingerprint != "" || current.ExecutionFingerprint != "" {
		changes.ExecutionConfigChanged = stored.ExecutionFingerprint != current.ExecutionFingerprint
	}
	lastStoredLogID := stored.LastLogID
	if lastStoredLogID == 0 {
		lastStoredLogID = stored.FlowRevision
	}
	lastCurrentLogID := current.LastLogID
	if lastCurrentLogID == 0 {
		lastCurrentLogID = current.FlowRevision
	}
	changes.FlowChanged = lastCurrentLogID > lastStoredLogID
	return changes
}

// IsAnchorStale 保留通用兼容判断；具体自动刷新策略应使用 CompareContextAnchors 分类决策。
func IsAnchorStale(stored, current OAContextAnchor) bool {
	return CompareContextAnchors(stored, current).Any()
}

// ComputeProcessDataFingerprint 对流程主表+明细数据做稳定 hash。
func ComputeProcessDataFingerprint(pd *ProcessData) string {
	if pd == nil {
		return ""
	}
	payload := map[string]interface{}{
		"main_data":     pd.MainData,
		"detail_tables": pd.DetailTables,
	}
	b, err := json.Marshal(canonicalize(payload))
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// AttachmentVersionAnchor 是无需下载文件即可读取的附件版本元数据。
type AttachmentVersionAnchor struct {
	FieldKey    string `json:"field_key"`
	DocID       string `json:"doc_id"`
	VersionID   int64  `json:"version_id"`
	ImageFileID string `json:"image_file_id,omitempty"`
	FileName    string `json:"file_name,omitempty"`
}

// ComputeAttachmentFingerprints 计算全局及按附件字段分组的稳定版本指纹。
func ComputeAttachmentFingerprints(items []AttachmentVersionAnchor) (string, map[string]string) {
	grouped := make(map[string][]AttachmentVersionAnchor)
	for _, item := range items {
		key := strings.ToLower(strings.TrimSpace(item.FieldKey))
		if key == "" {
			continue
		}
		item.FieldKey = key
		grouped[key] = append(grouped[key], item)
	}
	fieldFingerprints := make(map[string]string, len(grouped))
	fieldKeys := make([]string, 0, len(grouped))
	for key, values := range grouped {
		sort.Slice(values, func(i, j int) bool {
			if values[i].DocID != values[j].DocID {
				return values[i].DocID < values[j].DocID
			}
			if values[i].VersionID != values[j].VersionID {
				return values[i].VersionID < values[j].VersionID
			}
			return values[i].ImageFileID < values[j].ImageFileID
		})
		b, _ := json.Marshal(values)
		sum := sha256.Sum256(b)
		fieldFingerprints[key] = "sha256:" + hex.EncodeToString(sum[:])
		fieldKeys = append(fieldKeys, key)
	}
	sort.Strings(fieldKeys)
	combined := make([]string, 0, len(fieldKeys))
	for _, key := range fieldKeys {
		combined = append(combined, key+"="+fieldFingerprints[key])
	}
	if len(combined) == 0 {
		return "", fieldFingerprints
	}
	sum := sha256.Sum256([]byte(strings.Join(combined, "\n")))
	return "sha256:" + hex.EncodeToString(sum[:]), fieldFingerprints
}

func canonicalize(v interface{}) interface{} {
	switch x := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(x))
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			out[k] = canonicalize(x[k])
		}
		return out
	case []map[string]interface{}:
		items := make([]interface{}, len(x))
		for i, item := range x {
			items[i] = canonicalize(item)
		}
		return items
	case []interface{}:
		items := make([]interface{}, len(x))
		for i, item := range x {
			items[i] = canonicalize(item)
		}
		return items
	default:
		return v
	}
}
