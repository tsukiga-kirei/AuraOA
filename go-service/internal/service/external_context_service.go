package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/crypto"
	"auraoa/go-service/internal/pkg/oa"
	"auraoa/go-service/internal/repository"
)

const (
	externalContextDefaultSplitter = ","
	externalContextDefaultMaxRows  = 20
)

// ExternalContextService 解析规则或总结块级别的外部关联数据。
type ExternalContextService struct {
	oaConnRepo    *repository.OAConnectionRepo
	attachmentSvc *AttachmentRecognitionService
}

func NewExternalContextService(oaConnRepo *repository.OAConnectionRepo, attachmentSvc *AttachmentRecognitionService) *ExternalContextService {
	return &ExternalContextService{oaConnRepo: oaConnRepo, attachmentSvc: attachmentSvc}
}

type ExternalContextTestRequest struct {
	ProcessID string          `json:"process_id"`
	Mounts    json.RawMessage `json:"context_mounts" binding:"required"`
}

type ExternalContextTestResponse struct {
	ContextText string `json:"context_text"`
}

type ExternalWorkflowFieldsRequest struct {
	ProcessType string `json:"process_type"`
	WorkflowID  string `json:"workflow_id"`
}

type ExternalWorkflowSearchRequest struct {
	Keyword string `json:"keyword"`
}

func parseExternalContextMounts(raw []byte) []model.ExternalContextMount {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var mounts []model.ExternalContextMount
	_ = json.Unmarshal(raw, &mounts)
	return mounts
}

func (s *ExternalContextService) ResolveForPrompt(c *gin.Context, tenant *model.Tenant, processID string, processData *oa.ProcessData, raw []byte) string {
	return s.ResolveMountsForPrompt(c, tenant, processID, processData, parseExternalContextMounts(raw))
}

func (s *ExternalContextService) ResolveMountsForPrompt(c *gin.Context, tenant *model.Tenant, processID string, processData *oa.ProcessData, mounts []model.ExternalContextMount) string {
	if s == nil || tenant == nil || len(mounts) == 0 {
		return ""
	}
	adapter, err := s.getOAAdapter(tenant, false)
	if err != nil {
		return "外部关联数据：\n（创建 OA 查询连接失败：" + err.Error() + "）"
	}
	if processData == nil {
		pd, err := adapter.FetchProcessData(c.Request.Context(), processID)
		if err != nil {
			return "外部关联数据：\n（拉取当前流程数据失败：" + err.Error() + "）"
		}
		processData = pd
	}

	var sections []string
	for _, mount := range mounts {
		if !mount.Enabled {
			continue
		}
		sections = append(sections, s.resolveMount(c.Request.Context(), adapter, processData, mount))
	}
	if len(sections) == 0 {
		return ""
	}
	return "外部关联数据：\n" + strings.Join(sections, "\n\n")
}

func (s *ExternalContextService) Test(c *gin.Context, tenant *model.Tenant, req ExternalContextTestRequest) (*ExternalContextTestResponse, error) {
	mounts := parseExternalContextMounts(req.Mounts)
	if strings.TrimSpace(req.ProcessID) == "" {
		text := s.ValidateMounts(c, tenant, mounts)
		if strings.TrimSpace(text) == "" {
			text = "外部关联数据：\n（未配置启用的关联数据）"
		}
		return &ExternalContextTestResponse{ContextText: text}, nil
	}
	text := s.ResolveMountsForPrompt(c, tenant, req.ProcessID, nil, mounts)
	if strings.TrimSpace(text) == "" {
		text = "外部关联数据：\n（未配置启用的关联数据）"
	}
	return &ExternalContextTestResponse{ContextText: text}, nil
}

func (s *ExternalContextService) ValidateMounts(c *gin.Context, tenant *model.Tenant, mounts []model.ExternalContextMount) string {
	if s == nil || tenant == nil || len(mounts) == 0 {
		return ""
	}
	adapter, err := s.getOAAdapter(tenant, false)
	if err != nil {
		return "外部关联数据测试：\n（创建 OA 查询连接失败：" + err.Error() + "）"
	}
	var sections []string
	for _, mount := range mounts {
		if !mount.Enabled {
			continue
		}
		name := firstNonEmpty(mount.Name, contextMountTypeLabel(mount.Type))
		switch mount.Type {
		case "workflow":
			if mount.Workflow == nil || strings.TrimSpace(mount.Workflow.TargetProcessType) == "" {
				sections = append(sections, fmt.Sprintf("【%s】\n未指定目标流程：运行时将按 requestid 自动读取引用流程全部字段。", name))
				continue
			}
			if _, err := adapter.ValidateProcess(c.Request.Context(), mount.Workflow.TargetProcessType); err != nil {
				sections = append(sections, fmt.Sprintf("【%s】\n目标流程校验失败：%s", name, err.Error()))
			} else {
				sections = append(sections, fmt.Sprintf("【%s】\n目标流程「%s」存在，配置可用于指定字段引用。", name, mount.Workflow.TargetProcessType))
			}
		case "model":
			if mount.Model == nil {
				sections = append(sections, fmt.Sprintf("【%s】\n建模表配置为空。", name))
				continue
			}
			querier, ok := adapter.(oa.ModelContextQuerier)
			if !ok {
				sections = append(sections, fmt.Sprintf("【%s】\n当前 OA 类型暂不支持建模表关联查询。", name))
				continue
			}
			_, err := querier.QueryModelContext(c.Request.Context(), oa.ModelContextQuery{
				TableName:    mount.Model.TableName,
				JoinField:    firstNonEmpty(mount.Model.JoinField, "id"),
				SourceValue:  "__auraoa_config_probe__",
				Mode:         firstNonEmpty(mount.Model.Mode, "exists"),
				ReturnFields: mount.Model.ReturnFields,
				MaxRows:      1,
				CustomSQL:    mount.Model.CustomSQL,
			})
			if err != nil {
				sections = append(sections, fmt.Sprintf("【%s】\n建模查询校验失败：%s", name, err.Error()))
			} else {
				sections = append(sections, fmt.Sprintf("【%s】\n建模表「%s」和关联字段「%s」可查询。", name, mount.Model.TableName, firstNonEmpty(mount.Model.JoinField, "id")))
			}
		}
	}
	if len(sections) == 0 {
		return ""
	}
	return "外部关联数据测试：\n" + strings.Join(sections, "\n\n")
}

func (s *ExternalContextService) FetchWorkflowFields(c *gin.Context, tenant *model.Tenant, req ExternalWorkflowFieldsRequest) (*oa.ProcessFields, error) {
	adapter, err := s.getOAAdapter(tenant, false)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.WorkflowID) != "" {
		if selector, ok := adapter.(oa.WorkflowDefinitionSelector); ok {
			return selector.FetchFieldsByWorkflowID(c.Request.Context(), req.WorkflowID)
		}
	}
	if strings.TrimSpace(req.ProcessType) == "" {
		return nil, fmt.Errorf("目标流程为空")
	}
	return adapter.FetchFields(c.Request.Context(), req.ProcessType)
}

func (s *ExternalContextService) SearchWorkflows(c *gin.Context, tenant *model.Tenant, keyword string) ([]oa.ProcessInfo, error) {
	adapter, err := s.getOAAdapter(tenant, false)
	if err != nil {
		return nil, err
	}
	if selector, ok := adapter.(oa.WorkflowDefinitionSelector); ok {
		return selector.SearchWorkflowDefinitions(c.Request.Context(), keyword)
	}
	if strings.TrimSpace(keyword) == "" {
		return []oa.ProcessInfo{}, nil
	}
	info, err := adapter.ValidateProcess(c.Request.Context(), keyword)
	if err != nil {
		return nil, err
	}
	return []oa.ProcessInfo{*info}, nil
}

func (s *AuditExecuteService) resolveAuditRulesExternalContext(c *gin.Context, tenant *model.Tenant, processID string, processData *oa.ProcessData, rules []model.AuditRule) string {
	if s.externalCtx == nil || len(rules) == 0 {
		return ""
	}
	var sections []string
	for _, rule := range rules {
		if !isRuleEnabled(&rule) || !rule.ContextEnabled || len(rule.ContextMounts) == 0 {
			continue
		}
		text := s.externalCtx.ResolveForPrompt(c, tenant, processID, processData, rule.ContextMounts)
		if strings.TrimSpace(text) == "" {
			continue
		}
		sections = append(sections, fmt.Sprintf("规则：%s\n%s", rule.RuleContent, text))
	}
	if len(sections) == 0 {
		return ""
	}
	return "规则关联外部数据：\n" + strings.Join(sections, "\n\n")
}

func (s *ArchiveReviewService) resolveArchiveRulesExternalContext(c *gin.Context, tenant *model.Tenant, processID string, processData *oa.ProcessData, rules []model.ArchiveRule) string {
	if s.externalCtx == nil || len(rules) == 0 {
		return ""
	}
	var sections []string
	for _, rule := range rules {
		if !rule.IsEnabled() || !rule.ContextEnabled || len(rule.ContextMounts) == 0 {
			continue
		}
		text := s.externalCtx.ResolveForPrompt(c, tenant, processID, processData, rule.ContextMounts)
		if strings.TrimSpace(text) == "" {
			continue
		}
		sections = append(sections, fmt.Sprintf("规则：%s\n%s", rule.RuleContent, text))
	}
	if len(sections) == 0 {
		return ""
	}
	return "规则关联外部数据：\n" + strings.Join(sections, "\n\n")
}

func (s *ExternalContextService) resolveMount(ctx context.Context, adapter oa.OAAdapter, current *oa.ProcessData, mount model.ExternalContextMount) string {
	name := firstNonEmpty(mount.Name, contextMountTypeLabel(mount.Type))
	sourceValue := extractContextSourceValue(current, mount.SourceField)
	sourceField := formatExternalContextSourceField(current, mount.SourceField)
	if strings.TrimSpace(sourceValue) == "" {
		return fmt.Sprintf("【%s】\n来源字段 %s 为空，未执行查询。", name, sourceField)
	}
	switch mount.Type {
	case "workflow":
		return s.resolveWorkflowMount(ctx, adapter, name, sourceValue, sourceField, mount)
	case "model":
		return s.resolveModelMount(ctx, adapter, name, sourceValue, sourceField, mount)
	default:
		return fmt.Sprintf("【%s】\n不支持的关联类型：%s", name, mount.Type)
	}
}

func (s *ExternalContextService) resolveWorkflowMount(ctx context.Context, adapter oa.OAAdapter, name, sourceValue, sourceField string, mount model.ExternalContextMount) string {
	cfg := mount.Workflow
	if cfg == nil {
		cfg = &model.ExternalWorkflowContextConfig{}
	}
	splitter := firstNonEmpty(mount.Splitter, externalContextDefaultSplitter)
	ids := splitExternalValues(sourceValue, splitter, 0)
	if len(ids) == 0 {
		return fmt.Sprintf("【%s】\n未解析到有效 requestid。", name)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("【%s】\n来源字段：%s\n解析到 %d 个 requestid。", name, sourceField, len(ids)))
	for i, id := range ids {
		sb.WriteString(fmt.Sprintf("\n\n%d. requestid：%s", i+1, id))
		summary, err := adapter.FetchProcessRequestSummary(ctx, id)
		if err != nil {
			sb.WriteString("\n查询流程基础信息失败：" + err.Error())
			continue
		}
		sb.WriteString(formatReferencedWorkflowBasic(summary, cfg.BasicFields))
		dataMode := firstNonEmpty(cfg.DataMode, "none")
		if strings.TrimSpace(cfg.TargetProcessType) == "" {
			dataMode = "all_fields"
		}
		if dataMode == "none" {
			continue
		}
		if cfg.TargetProcessType != "" && summary.ProcessType != "" && !strings.EqualFold(cfg.TargetProcessType, summary.ProcessType) {
			sb.WriteString(fmt.Sprintf("\n提示：实际流程类型为「%s」，与配置目标流程「%s」不一致。", summary.ProcessType, cfg.TargetProcessType))
			if firstNonEmpty(cfg.FallbackStrategy, "basic_with_notice") != "all_fields" {
				sb.WriteString("\n已按兜底策略仅提供流程基础信息。")
				continue
			}
		}
		pd, err := adapter.FetchProcessData(ctx, id)
		if err != nil {
			sb.WriteString("\n查询引用流程表单数据失败：" + err.Error())
			continue
		}
		if dataMode == "selected_fields" && len(cfg.SelectedFields) == 0 {
			sb.WriteString("\n引用流程表单数据：未选择字段，已跳过。")
			continue
		}
		fieldSet := SelectedFieldSet(nil)
		if dataMode == "selected_fields" {
			fieldSet = selectedFieldSetFromRefs(cfg.SelectedFields)
		}
		sb.WriteString("\n引用流程表单数据：\n")
		sb.WriteString(formatProcessDataForExternalContext(pd, fieldSet, cfg.MaxRows))
	}
	return sb.String()
}

func (s *ExternalContextService) resolveModelMount(ctx context.Context, adapter oa.OAAdapter, name, sourceValue, sourceField string, mount model.ExternalContextMount) string {
	cfg := mount.Model
	if cfg == nil {
		cfg = &model.ExternalModelContextConfig{}
	}
	querier, ok := adapter.(oa.ModelContextQuerier)
	if !ok {
		return fmt.Sprintf("【%s】\n当前 OA 类型暂不支持建模表关联查询。", name)
	}
	res, err := querier.QueryModelContext(ctx, oa.ModelContextQuery{
		TableName:    cfg.TableName,
		JoinField:    firstNonEmpty(cfg.JoinField, "id"),
		SourceValue:  sourceValue,
		Mode:         firstNonEmpty(cfg.Mode, "exists"),
		ReturnFields: cfg.ReturnFields,
		MaxRows:      cfg.MaxRows,
		OrderBy:      cfg.OrderBy,
		OrderDir:     cfg.OrderDir,
		CustomSQL:    cfg.CustomSQL,
	})
	if err != nil {
		return fmt.Sprintf("【%s】\n建模表查询失败：%s", name, err.Error())
	}
	var fieldLabels map[string]string
	if res.Mode == "rows" {
		if resolver, ok := adapter.(oa.ModelFieldLabelResolver); ok {
			fieldLabels, _ = resolver.FetchModelFieldLabels(ctx, cfg.TableName)
		}
	}
	return formatModelContextResult(name, sourceField, cfg, res, fieldLabels)
}

func (s *ExternalContextService) getOAAdapter(tenant *model.Tenant, withAttachments bool) (oa.OAAdapter, error) {
	if tenant.OADBConnectionID == nil {
		return nil, fmt.Errorf("租户未配置 OA 数据库连接")
	}
	conn, err := s.oaConnRepo.FindByID(*tenant.OADBConnectionID)
	if err != nil {
		return nil, fmt.Errorf("OA 数据库连接配置不存在")
	}
	if conn.Password != "" {
		password, err := crypto.Decrypt(conn.Password)
		if err != nil {
			return nil, fmt.Errorf("OA 数据库密码解密失败")
		}
		conn.Password = password
	}
	var attachmentSvc oa.AttachmentRecognitionService
	if withAttachments && s.attachmentSvc != nil {
		attachmentSvc = s.attachmentSvc
	}
	return oa.NewOAAdapter(conn.OAType, conn, attachmentSvc)
}

func contextMountTypeLabel(t string) string {
	switch t {
	case "workflow":
		return "关联流程"
	case "model":
		return "关联建模表"
	default:
		return "外部关联数据"
	}
}

func extractContextSourceValue(pd *oa.ProcessData, ref string) string {
	if pd == nil {
		return ""
	}
	table, field := parseExternalFieldRef(ref)
	var values []string
	if table == "main" {
		if v, ok := findValueCaseInsensitive(pd.MainData, field); ok {
			return stringifyContextValue(v)
		}
		return ""
	}
	for name, rows := range pd.DetailTables {
		if !strings.EqualFold(name, table) {
			continue
		}
		for _, row := range rows {
			if v, ok := findValueCaseInsensitive(row, field); ok {
				if s := stringifyContextValue(v); s != "" {
					values = append(values, s)
				}
			}
		}
	}
	return strings.Join(values, ",")
}

// formatExternalContextSourceField 为外部关联数据中的来源字段补充 OA 字段显示名。
func formatExternalContextSourceField(pd *oa.ProcessData, ref string) string {
	ref = strings.TrimSpace(ref)
	if pd == nil || ref == "" {
		return ref
	}
	table, field := parseExternalFieldRef(ref)
	if field == "" {
		return ref
	}
	labels := pd.FieldLabels[table]
	if labels == nil {
		for tableName, tableLabels := range pd.FieldLabels {
			if strings.EqualFold(tableName, table) {
				labels = tableLabels
				break
			}
		}
	}
	label := strings.TrimSpace(promptFieldLabel(field, labels))
	if label == "" || label == field {
		return ref
	}
	return fmt.Sprintf("%s（%s）", label, ref)
}

func parseExternalFieldRef(ref string) (string, string) {
	ref = strings.TrimSpace(ref)
	if strings.Contains(ref, ":") {
		parts := strings.SplitN(ref, ":", 2)
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return "main", ref
}

func findValueCaseInsensitive(row map[string]interface{}, field string) (interface{}, bool) {
	for k, v := range row {
		if strings.EqualFold(k, field) {
			return v, true
		}
	}
	return nil, false
}

func stringifyContextValue(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case []byte:
		return strings.TrimSpace(string(t))
	case map[string]interface{}:
		if raw, ok := t["value"]; ok {
			return stringifyContextValue(raw)
		}
		b, _ := json.Marshal(t)
		return string(b)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
}

func splitExternalValues(raw, splitter string, max int) []string {
	if splitter == "" {
		splitter = externalContextDefaultSplitter
	}
	parts := strings.Split(raw, splitter)
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		v := strings.TrimSpace(part)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
		if max > 0 && len(out) >= max {
			break
		}
	}
	return out
}

func formatReferencedWorkflowBasic(summary *oa.ProcessRequestSummary, fields []string) string {
	if summary == nil || len(fields) == 0 {
		// 未勾选任何基础信息字段时不注入，避免空配置被当成「全选」
		return ""
	}
	enabled := map[string]bool{}
	for _, f := range fields {
		enabled[f] = true
	}
	lines := []string{}
	add := func(key, label, val string) {
		if enabled[key] && strings.TrimSpace(val) != "" {
			lines = append(lines, fmt.Sprintf("%s：%s", label, val))
		}
	}
	add("title", "流程标题", summary.Title)
	if enabled["archived"] {
		archived := "否"
		if strings.Contains(summary.CurrentNode, "归档") {
			archived = "是"
		}
		lines = append(lines, "是否归档："+archived)
	}
	add("applicant", "发起人", summary.Applicant)
	add("department", "发起部门", summary.Department)
	add("process_type", "流程类型", firstNonEmpty(summary.ProcessTypeLabel, summary.ProcessType))
	add("current_node", "当前节点", summary.CurrentNode)
	add("submit_time", "提交时间", summary.SubmitTime)
	if len(lines) == 0 {
		return ""
	}
	return "\n" + strings.Join(lines, "\n")
}

func selectedFieldSetFromRefs(refs []string) SelectedFieldSet {
	fs := SelectedFieldSet{"main": map[string]bool{}}
	for _, ref := range refs {
		table, field := parseExternalFieldRef(ref)
		if table == "" || field == "" {
			continue
		}
		if fs[table] == nil {
			fs[table] = map[string]bool{}
		}
		fs[table][strings.ToLower(field)] = true
	}
	return fs
}

func formatProcessDataForExternalContext(pd *oa.ProcessData, fieldSet SelectedFieldSet, maxRows int) string {
	if maxRows <= 0 {
		maxRows = externalContextDefaultMaxRows
	}
	if maxRows > 100 {
		maxRows = 100
	}
	main := formatMainData(filterFields(pd.MainData, selectedKeysForTable(fieldSet, "main")), pd.FieldLabels["main"])
	details := limitDetailRows(pd.DetailTables, maxRows)
	detailText := formatGroupedDetailData(details, fieldSet, pd.FieldLabels)
	return "主表字段：\n" + main + "\n\n明细表字段：\n" + detailText
}

func limitDetailRows(input map[string][]map[string]interface{}, maxRows int) map[string][]map[string]interface{} {
	out := make(map[string][]map[string]interface{}, len(input))
	for table, rows := range input {
		if len(rows) > maxRows {
			out[table] = rows[:maxRows]
		} else {
			out[table] = rows
		}
	}
	return out
}

func formatModelContextResult(name, sourceField string, cfg *model.ExternalModelContextConfig, res *oa.ModelContextQueryResult, fieldLabels map[string]string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("【%s】\n来源字段：%s\n查询方式：%s", name, sourceField, firstNonEmpty(cfg.Mode, "exists")))
	switch res.Mode {
	case "exists":
		if res.Exists {
			sb.WriteString(fmt.Sprintf("\n结果：存在匹配记录（共 %d 条）。", res.Count))
		} else {
			sb.WriteString("\n结果：不存在匹配记录。")
		}
	case "count":
		sb.WriteString(fmt.Sprintf("\n结果：匹配记录数 %d 条。", res.Count))
	default:
		sb.WriteString(fmt.Sprintf("\n结果：返回 %d 行。", len(res.Rows)))
		if len(res.Rows) == 0 {
			break
		}
		sb.WriteString("\n行数据：")
		for i, row := range res.Rows {
			sb.WriteString(fmt.Sprintf("\n%d. %s", i+1, formatContextRow(row, fieldLabels)))
		}
	}
	if strings.TrimSpace(res.Notice) != "" {
		sb.WriteString("\n提示：" + res.Notice)
	}
	return sb.String()
}

func formatContextRow(row map[string]interface{}, fieldLabels map[string]string) string {
	if len(row) == 0 {
		return "（空行）"
	}
	parts := make([]string, 0, len(row))
	keys := make([]string, 0, len(row))
	for key := range row {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	usedLabels := map[string]int{}
	for _, key := range keys {
		label := promptFieldLabel(key, fieldLabels)
		usedLabels[label]++
		if usedLabels[label] > 1 {
			label = fmt.Sprintf("%s_%d", label, usedLabels[label])
		}
		parts = append(parts, fmt.Sprintf("%s=%s", label, truncateContextValue(stringifyContextValue(row[key]), 500)))
	}
	return strings.Join(parts, "；")
}

func truncateContextValue(s string, max int) string {
	if max <= 0 || len([]rune(s)) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max]) + "..."
}
