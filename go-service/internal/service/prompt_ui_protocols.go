package service

// 智能体模块（Generative UI）卡片协议的系统变量定义。
//
// 这两段协议是 ::form-confirm 与 ::chart 的唯一权威结构说明，由系统变量在运行时
// 注入智能体系统提示词，供 oa_creator、oa_analyst 以及后续新增的任意智能体复用，
// 避免模型凭“常见写法”猜测键名导致前端卡片渲染异常。
//
// 前端 ChatThread 的解析器与 FormConfirmCard/ChartCard 的归一化逻辑必须与本文件
// 保持同步：协议里承诺的键，渲染端必须支持；渲染端兼容的别名，协议里可只写推荐值。
const (
	// SystemVariableFormConfirmProtocol 表单确认卡片协议占位符
	SystemVariableFormConfirmProtocol = "{{form_confirm_protocol}}"
	// SystemVariableChartProtocol 图表卡片协议占位符
	SystemVariableChartProtocol = "{{chart_protocol}}"
)

// uiFormConfirmProtocol 表单确认卡片（:::form-confirm）的权威结构说明
const uiFormConfirmProtocol = `【:::form-confirm 表单确认卡片协议（系统内置，输出后逐条自检）】
1. 何时使用：仅当需要用户核对并确认一份表单数据时使用；纯文字说明不要套卡片。
2. 语法：开始标记 :::form-confirm 与结束标记 ::: 必须各自独占一行；块内必须是合法的裸 JSON 对象；禁止用 Markdown 代码块围栏（三个反引号）包裹，禁止注释与尾随逗号。
3. 顶层键只允许："variant"、"workflow_id"、"workflow_name"、"description"、"fields"、"columns"、"rows"、"details"（键名小写）。严禁自造键名，前端会静默丢弃不认识的键（自造键会导致卡片缺标题、字段串值甚至无法确认）。
4. variant 取值：summary_confirm（默认，摘要确认卡片）| compact_form（紧凑表单）| data_table（只读数据表格，用于查数结果展示与导出）。
5. fields 每项：
   - name（必填）：OA 真实字段名，必须与 get_workflow_schema 返回一致（如 sqrq、wp_type、reason）；缺失或写错会导致用户点击确认后创建草稿失败。
   - label（必填）：中文显示名。
   - type：text | number | date | select | textarea | browser（人员、部门等浏览按钮字段）。
   - value：预填值；date 使用 YYYY-MM-DD；browser 使用存储 ID，同时用 display_value 提供展示名称。
   - options：select 必须给出全部候选，推荐对象数组 [{"label":"办公文具","value":"0"}]，也接受 ["办公文具"]。
   - disabled：可选，true 表示只读（如申请人、申请部门）。
6. 明细表：优先使用 data_table 变体（columns + rows）；若需在确认卡片内附带只读明细，使用 details（对象或数组），形如 {"title":"费用明细","columns":["费用类型","金额"],"rows":[{"费用类型":"交通费","金额":120}]}。
7. 交互闭环：用户点击【确认创建】或【取消】后，前端会回传以【用户已确认表单】或【用户已取消表单创建】开头的消息；在收到确认消息之前，严禁调用任何提交类工具。
8. 示例：
:::form-confirm
{
  "variant": "summary_confirm",
  "workflow_id": "102",
  "workflow_name": "物品领用申请",
  "fields": [
    { "name": "sqrq", "label": "申请日期", "type": "date", "value": "2026-11-05" },
    { "name": "wp_type", "label": "物品类型", "type": "select", "value": "0", "options": [
        { "label": "办公文具", "value": "0" },
        { "label": "数码配件", "value": "1" }
    ]},
    { "name": "reason", "label": "领用用途", "type": "textarea", "value": "部门日常办公文具补充" }
  ]
}
:::`

// uiChartProtocol 图表卡片（:::chart）的权威结构说明
const uiChartProtocol = `【:::chart 图表卡片协议（系统内置，输出后逐条自检）】
1. 何时使用：查询结果涉及排行对比（bar）、时间走势（line）、占比结构（pie）或多维画像（radar）时，在回复末尾输出。
2. 语法：开始标记 :::chart 与结束标记 ::: 各自独占一行；块内为裸 JSON；禁止 Markdown 代码块围栏与注释；一个块只画一张图，多图请输出多个块。
3. 顶层键只允许："type"、"title"、"subtitle"、"categories"、"series"、"indicators"、"columns"、"rows"（键名小写，雷达图指标是复数 indicators，不是 indicator）。
4. type 取值：bar | line | pie | radar。
5. categories 是维度名称数组；series 是数组，每项 {"name":"系列名","data":[数值,...]}，data 与 categories 等长且顺序一致。
6. pie：categories 表示扇区名称、series[0].data 表示同序数值；不要只给 [{"name":"A","value":1}] 这类对象数组而缺少 categories。
7. radar：必须提供 indicators，形如 [{"name":"维度名","max":800}]，与 series[0].data 等长，max 取该维度的合理上限。
8. 数值必须来自真实查询结果，严禁编造；维度超过 10 项时先聚合或取 Top N。
9. 示例（bar）：
:::chart
{
  "type": "bar",
  "title": "各部门流程发起量 Top5",
  "categories": ["市场部", "研发部", "供应链", "财务部", "人事部"],
  "series": [{ "name": "流程数量", "data": [142, 98, 76, 55, 32] }]
}
:::
10. 示例（pie）：
:::chart
{
  "type": "pie",
  "title": "流程办结状态占比",
  "categories": ["已归档", "流转中"],
  "series": [{ "name": "状态占比", "data": [660141, 17092] }]
}
:::`
