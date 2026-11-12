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
	selected := attachmentFieldFingerprintsForFieldSet(anchor, fieldSet)
	for key := range selected {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+selected[key])
	}
	if len(parts) == 0 {
		return ""
	}
	return stableJSONFingerprint(parts)
}

func attachmentFieldFingerprintsForFieldSet(anchor oa.OAContextAnchor, fieldSet SelectedFieldSet) map[string]string {
	if fieldSet == nil {
		return anchor.AttachmentFieldFingerprints
	}
	selected := make(map[string]string)
	for key, fingerprint := range anchor.AttachmentFieldFingerprints {
		// 主表保留历史字段键，明细使用「表名:字段名」，与实际下载和总结块提示词一致。
		table, field := "main", key
		if parts := strings.SplitN(key, ":", 2); len(parts) == 2 {
			table, field = parts[0], parts[1]
		}
		if oa.AttachmentFieldSelected(fieldSet, table, field) {
			selected[key] = fingerprint
		}
	}
	return selected
}

// auditOAAnchorForFields 让上下文查询与任务执行按同一字段范围保存完整锚点和失败拦截指纹。
func auditOAAnchorForFields(anchor oa.OAContextAnchor, pd *oa.ProcessData, fieldSet SelectedFieldSet, executionFingerprint string) oa.OAContextAnchor {
	anchor.ContentFingerprint = computeSelectedProcessFingerprint(pd, fieldSet, true, true)
	anchor.AttachmentFingerprint = attachmentFingerprintForFieldSet(anchor, fieldSet)
	anchor.AttachmentFieldFingerprints = attachmentFieldFingerprintsForFieldSet(anchor, fieldSet)
	anchor.ExecutionFingerprint = executionFingerprint
	return anchor
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
			// 总结块配置版本变化不属于 OA 业务数据变化。旧结果继续使用已绑定版本，
			// 只有用户显式选择升级配置时才重新生成。
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
	if cfg.AutoAuditOnDataChange && (changes.DataChanged || changes.AttachmentChanged) {
		return true
	}
	if cfg.AutoAuditOnReturnResubmit && changes.ReturnResubmitChanged {
		return true
	}
	return cfg.AutoAuditOnFlowChange && (changes.FlowChanged || changes.CurrentNodeChanged)
}
