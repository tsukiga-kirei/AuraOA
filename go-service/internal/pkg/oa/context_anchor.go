package oa

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// OAContextAnchor 审核完成时 OA 流程上下文锚点，用于判断结论是否过期。
type OAContextAnchor struct {
	LastReturnLogID    int64  `json:"last_return_log_id"`
	FlowRevision       int64  `json:"flow_revision"`
	LastResubmitLogID  int64  `json:"last_resubmit_log_id,omitempty"`
	CurrentNodeID      int    `json:"current_node_id"`
	ContentFingerprint string `json:"content_fingerprint,omitempty"`
}

// IsEmpty 历史数据或未写入锚点时视为空。
func (a OAContextAnchor) IsEmpty() bool {
	return a.FlowRevision == 0 && a.LastReturnLogID == 0 && a.CurrentNodeID == 0 && a.ContentFingerprint == ""
}

// IsStale 对比当前 OA 锚点与审核完成时存储的锚点。
func IsAnchorStale(stored, current OAContextAnchor) bool {
	if stored.IsEmpty() {
		return true
	}
	if current.LastReturnLogID > stored.LastReturnLogID {
		return true
	}
	if current.FlowRevision > stored.FlowRevision {
		return true
	}
	if stored.ContentFingerprint != "" && current.ContentFingerprint != "" &&
		stored.ContentFingerprint != current.ContentFingerprint {
		return true
	}
	return false
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
