package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/oa"
)

// SummaryBlockDependencyFingerprint 保存单个总结块实际依赖的数据指纹。
type SummaryBlockDependencyFingerprint struct {
	Config      string `json:"config"`
	Data        string `json:"data,omitempty"`
	Attachments string `json:"attachments,omitempty"`
	Flow        string `json:"flow,omitempty"`
	ProcessMeta string `json:"process_meta,omitempty"`
}

func stableJSONFingerprint(value interface{}) string {
	b, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func sortedStringSetKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key, enabled := range values {
		if enabled {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

// computeSelectedProcessFingerprint 只对最终传给 AI 的主表和明细字段计算指纹。
func computeSelectedProcessFingerprint(pd *oa.ProcessData, fieldSet SelectedFieldSet, includeMain, includeDetails bool) string {
	if pd == nil {
		return ""
	}
	selected := &oa.ProcessData{
		MainData:     map[string]interface{}{},
		DetailTables: map[string][]map[string]interface{}{},
	}
	if includeMain {
		var allowed map[string]bool
		if fieldSet != nil {
			allowed = fieldSet["main"]
		}
		selected.MainData = filterFields(pd.MainData, allowed)
	}
	if includeDetails {
		tableNames := make([]string, 0, len(pd.DetailTables))
		for table := range pd.DetailTables {
			tableNames = append(tableNames, table)
		}
		sort.Strings(tableNames)
		for _, table := range tableNames {
			var allowed map[string]bool
			if fieldSet != nil {
				allowed = fieldSet[table]
			}
			rows := filterRowFields(pd.DetailTables[table], allowed)
			if len(rows) > 0 {
				selected.DetailTables[table] = rows
			}
		}
	}
	return oa.ComputeProcessDataFingerprint(selected)
}

func summaryBlockUsesVariable(block model.SummaryBlockConfig, variable string) bool {
	if summaryBlockIncludeAllData(block) {
		return true
	}
	for _, enabled := range block.EnabledDataVariables {
		if strings.TrimSpace(enabled) == variable {
			return true
		}
	}
	return false
}

func attachmentFingerprintForFieldSet(anchor oa.OAContextAnchor, fieldSet SelectedFieldSet) string {
	if fieldSet == nil {
		return anchor.AttachmentFingerprint
	}
	keys := make([]string, 0)
	seen := make(map[string]struct{})
	for _, fields := range fieldSet {
		for field := range fields {
			key := strings.ToLower(strings.TrimSpace(field))
			if _, ok := anchor.AttachmentFieldFingerprints[key]; !ok {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+anchor.AttachmentFieldFingerprints[key])
	}
	if len(parts) == 0 {
		return ""
	}
	return stableJSONFingerprint(parts)
}

func buildSummaryBlockDependencyFingerprints(
	blocks []model.SummaryBlockConfig,
	pd *oa.ProcessData,
	anchor oa.OAContextAnchor,
	process *oa.ProcessRequestSummary,
) map[string]SummaryBlockDependencyFingerprint {
	out := make(map[string]SummaryBlockDependencyFingerprint)
	for _, block := range blocks {
		if !block.Enabled || strings.TrimSpace(block.ID) == "" {
			continue
		}
		fieldSet := buildSummaryBlockFieldSet(block)
		usesMain := summaryBlockUsesVariable(block, "{{main_table}}")
		usesDetails := summaryBlockUsesVariable(block, "{{detail_tables}}")
		dep := SummaryBlockDependencyFingerprint{
			Config: stableJSONFingerprint(block),
		}
		if usesMain || usesDetails {
			dep.Data = computeSelectedProcessFingerprint(pd, fieldSet, usesMain, usesDetails)
		}
		if summaryBlockUsesVariable(block, "{{attachments}}") {
			dep.Attachments = attachmentFingerprintForFieldSet(anchor, fieldSet)
		}
		usesProcessMeta := summaryBlockUsesVariable(block, "{{process_meta}}")
		if summaryBlockUsesVariable(block, "{{flow_history}}") ||
			summaryBlockUsesVariable(block, "{{flow_graph}}") ||
			usesProcessMeta {
			dep.Flow = stableJSONFingerprint(map[string]interface{}{
				"last_log_id":          anchor.LastLogID,
				"last_return_log_id":   anchor.LastReturnLogID,
				"last_resubmit_log_id": anchor.LastResubmitLogID,
				"current_node_id":      anchor.CurrentNodeID,
			})
		}
		if usesProcessMeta && process != nil {
			dep.ProcessMeta = stableJSONFingerprint(map[string]string{
				"process_id":         process.ProcessID,
				"title":              process.Title,
				"applicant":          process.Applicant,
				"department":         process.Department,
				"process_type":       process.ProcessType,
				"process_type_label": process.ProcessTypeLabel,
				"submit_time":        process.SubmitTime,
			})
		}
		out[block.ID] = dep
	}
	return out
}

func changedSummaryBlockIDs(
	blocks []model.SummaryBlockConfig,
	stored, current map[string]SummaryBlockDependencyFingerprint,
	changes oa.OAContextChanges,
	cfg model.SummaryEmbedConfigData,
) []string {
	out := make([]string, 0)
	for _, block := range blocks {
		if !block.Enabled {
			continue
		}
		before, exists := stored[block.ID]
		after := current[block.ID]
		if !exists || before.Config == "" || before.Config != after.Config {
			if cfg.AutoSummaryOnDataChange {
				out = append(out, block.ID)
			}
			continue
		}
		dataChanged := before.Data != after.Data || before.Attachments != after.Attachments
		processMetaChanged := before.ProcessMeta != after.ProcessMeta
		if cfg.AutoSummaryOnDataChange && (dataChanged || processMetaChanged) {
			out = append(out, block.ID)
			continue
		}
		flowDependencyChanged := before.Flow != after.Flow
		if !flowDependencyChanged {
			continue
		}
		if cfg.AutoSummaryOnReturnResubmit && changes.ReturnResubmitChanged {
			out = append(out, block.ID)
			continue
		}
		if cfg.AutoSummaryOnFlowChange && (changes.FlowChanged || changes.CurrentNodeChanged) {
			out = append(out, block.ID)
		}
	}
	return out
}

func parseSummaryBlockDependencies(raw []byte) map[string]SummaryBlockDependencyFingerprint {
	var snapshot struct {
		BlockDependencies map[string]SummaryBlockDependencyFingerprint `json:"block_dependencies"`
	}
	_ = json.Unmarshal(raw, &snapshot)
	if snapshot.BlockDependencies == nil {
		return map[string]SummaryBlockDependencyFingerprint{}
	}
	return snapshot.BlockDependencies
}

func auditRefreshRequired(changes oa.OAContextChanges, cfg model.EmbedConfigData) bool {
	if changes.LegacyAnchor {
		return cfg.AutoAuditOnDataChange
	}
	if changes.ExecutionConfigChanged {
		return cfg.AutoAuditOnDataChange
	}
	if cfg.AutoAuditOnDataChange && (changes.DataChanged || changes.AttachmentChanged) {
		return true
	}
	if cfg.AutoAuditOnReturnResubmit && changes.ReturnResubmitChanged {
		return true
	}
	return cfg.AutoAuditOnFlowChange && (changes.FlowChanged || changes.CurrentNodeChanged)
}
