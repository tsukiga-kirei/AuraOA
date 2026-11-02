-- 000096_disable_oa_query_and_batch_audit_tools.up.sql
-- 1. 将系统种子智能体 oa_query (OA 查询助手) 设为默认禁用，避免与 oa_assist 发生重复
UPDATE agent_definitions
SET enabled = FALSE,
    updated_at = NOW()
WHERE agent_code = 'oa_query' AND is_system = TRUE;

-- 2. 优化系统默认旗舰智能体 oa_assist (OA 辅助办理助手) 的系统提示词：明确要求遇到待办列表多条审核时使用批量查询
UPDATE agent_definitions
SET system_prompt = '你是 AuraOA 的官方 OA 辅助办理助手。当前时间：{{current_datetime}}（{{weekday}}）。你的职责是协助用户高效检索 OA 待办与申请流程、查看表单详情与审批轨迹、针对流程内容起草专业的同意或退回审批意见、触发 AuraOA 的智能审核和流程总结任务，并为用户生成跳转至 OA 系统的直接办理链接。

【核心规则】
1. 当用户查询待办/流程并询问审核或总结结果时，若有多条单据，严禁针对每条流程单条循环调用工具！必须将所有相关的 process_id 组装到 process_ids 数组中一次性调用 get_latest_audit 或 get_latest_summary 批量获取。
2. 数据必须客观真实，严禁编造流程内容。你只具有辅助办理能力，不能代替用户在 OA 中最终点击审批提交。',
    updated_at = NOW()
WHERE agent_code = 'oa_assist' AND is_system = TRUE;

-- 3. 将租户默认分配的智能体列表收敛为默认仅包含 oa_assist
ALTER TABLE tenant_chat_allocations 
    ALTER COLUMN agent_codes SET DEFAULT '["oa_assist"]'::jsonb;
