package oa

// 本文件只说明泛微 E10 以后怎么接，不提供可调用的实现。
//
// 当前：
//   - 档案在 profile.go，编码 weaver_e10，Implemented 为 false。工厂会拒绝建连。
//   - 不写入 oa_type_options，系统设置的 OA 类型下拉先不出现 E10。
//   - 默认跳转模板、附件凭证、数据记忆归档规则、SQL 助手说明都留空。
//
// 跳转地址按 OA 类型区分，和 E9 无关。E9 留空模板时使用
// /workflow/request/ViewRequestForwardSPA.jsp?requestid={process_id}。
// E10 确认自己的流程页之后，只填写 weaver_e10 档案的 DefaultProcessURLTemplate。
// 工作台和对话读同一份档案，不要再各写一条兜底。
//
// 表结构确认并开始实现时：
//   - 在本文件实现 OAAdapter 的全部方法，按需实现可选接口。
//   - 在 factory.go 的 openOADatabase、newOAAdapterWithDB 中接入，连接池仍走 ConnectionManager。
//   - 禁止调用 openEcology9Database，也禁止复用 ecology9.go 的 SQL。
//   - 档案改为 Implemented=true，并补上 Drivers。
//   - 附件接口若与 E9 不同，新增自己的连接字段，不要复用 weaver_api_url、weaver_appid、weaver_default_user。
//   - 归档判断若不是「状态列等于一个字面量」，先扩展 WorkflowArchiveRule。
//   - QueryAssistNote 只写 E10 自己的表和字段。
//   - 最后增加迁移，向 oa_type_options 插入 weaver_e10。
