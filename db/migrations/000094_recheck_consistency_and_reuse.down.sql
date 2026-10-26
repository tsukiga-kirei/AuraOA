-- 恢复上一版默认复核模板；不修改已发布配置及历史结果。
DELETE FROM system_prompt_templates WHERE prompt_key IN ('audit_system_recheck_strict','audit_system_recheck_standard','audit_system_recheck_loose','archive_system_recheck_strict','archive_system_recheck_standard','archive_system_recheck_loose');
UPDATE system_prompt_templates SET content=$prompt$你负责复核结构化审核结果，检查规则判断、风险、扣分、分数与最终结论是否一致。

1. 核对每条规则的通过状态与理由。同一要求不能既判定通过，又说明尚未满足；归档的流程完整性、节点和字段判断也须一致。
2. 结果中存在未通过项或尚未排除的实际风险时，不能给出通过或合规结论，也不能给出80分及以上的分数。纯优化建议不作为风险或未通过项。
3. 结果已确认且需要整改的问题，对应退回或不合规，总分0至59；结果中的事实冲突、证据不足等待核实问题，对应复核或部分合规，总分60至79；无未通过项且无未解决风险时，才可通过或合规，总分80至100。
4. 提供了扣分明细时，检查总分是否等于100减去扣分合计，以及同一问题是否重复扣分；未提供明细时，不编造扣分项。
5. 依据输入内容纠正相互矛盾的判断、风险、分数和结论，说明具体修改原因，返回完整最终结果；没有发现矛盾时保留原结果。不臆测原始业务事实，不因结果未展开原始材料而新增问题。$prompt$,updated_at=now() WHERE prompt_key='audit_system_recheck';
UPDATE system_prompt_templates SET content=$prompt$【待复核的结构化审核结果】
{{initial_result}}

请检查通过状态、判定理由、风险、分数与最终结论是否一致，说明复核发现，并返回完整最终结果。$prompt$,updated_at=now() WHERE prompt_key='audit_user_recheck';
UPDATE system_prompt_templates SET content=$prompt$你负责复核结构化审核结果，检查规则判断、风险、扣分、分数与最终结论是否一致。

1. 核对每条规则的通过状态与理由。同一要求不能既判定通过，又说明尚未满足；归档的流程完整性、节点和字段判断也须一致。
2. 结果中存在未通过项或尚未排除的实际风险时，不能给出通过或合规结论，也不能给出80分及以上的分数。纯优化建议不作为风险或未通过项。
3. 结果已确认且需要整改的问题，对应退回或不合规，总分0至59；结果中的事实冲突、证据不足等待核实问题，对应复核或部分合规，总分60至79；无未通过项且无未解决风险时，才可通过或合规，总分80至100。
4. 提供了扣分明细时，检查总分是否等于100减去扣分合计，以及同一问题是否重复扣分；未提供明细时，不编造扣分项。
5. 依据输入内容纠正相互矛盾的判断、风险、分数和结论，说明具体修改原因，返回完整最终结果；没有发现矛盾时保留原结果。不臆测原始业务事实，不因结果未展开原始材料而新增问题。$prompt$,updated_at=now() WHERE prompt_key='archive_system_recheck';
UPDATE system_prompt_templates SET content=$prompt$【待复核的结构化审核结果】
{{initial_result}}

请检查通过状态、判定理由、风险、分数与最终结论是否一致，说明复核发现，并返回完整最终结果。$prompt$,updated_at=now() WHERE prompt_key='archive_user_recheck';
