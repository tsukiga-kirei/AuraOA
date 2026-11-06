---
name: 办公用品领用申请填报指南
description: 引导员工通过自然语言对话收集领用要素并生成表单确认卡片
agent: oa_creator
---

# 办公用品领用流程 SOP

## 1. 主动推断与槽位收集
先用 list_creatable_workflows 确认当前用户能创建目标流程。名称明确时传流程名，含糊时查看全部可创建流程。没有权限就停止，不要读取表单。确认有权限后，再利用 get_workflow_schema 获取表单下拉选项：
- 当用户陈述具体品名（如“中性笔”、“记号笔”、“A4打印纸”）时，主动匹配至【办公文具】。
- 当用户陈述数码外设（如“无线鼠标”、“键盘”、“转接线”）时，主动匹配至【数码配件】。
- 在回复中主动告知：“已为您将【{物品品名}】自动归类为【{对应下拉分类}】类别”。

## 2. 表单确认规则
当上述信息收集完毕后，严禁直接调用接口，必须在回复末尾输出确认卡片。下拉选择框字段务必附带全部 options 候选列表：

:::form-confirm
{
  "variant": "summary_confirm",
  "workflow_id": "102",
  "workflow_name": "物品领用申请",
  "fields": [
    { "name": "sqrq", "label": "申请日期", "type": "date", "value": "{期望日期}" },
    { "name": "wp_type", "label": "物品类型", "type": "select", "value": "{推荐选项值}", "options": [
        { "label": "办公文具", "value": "0" },
        { "label": "数码配件", "value": "1" },
        { "label": "日常耗材", "value": "2" }
    ]},
    { "name": "reason", "label": "领用用途", "type": "textarea", "value": "{用途说明}" }
  ]
}
:::

## 3. 提交调用
仅当收到用户发送的以【用户已确认表单】开头的消息时，才调用 create_oa_draft 工具提交为草稿。
