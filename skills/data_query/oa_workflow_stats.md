---
name: OA 流程与人员统计查数指南
description: 指导大模型根据自然语言生成合规的只读 SELECT SQL
agent: oa_analyst
---

# 泛微 E9 常用数据表结构与查数规范

## 1. 核心业务表字典
- 人员表：`hrmresource`（id: 人员ID, lastname: 姓名, loginid: 工号, departmentid: 部门ID, status: 状态）
- 部门表：`hrmdepartment`（id: 部门ID, departmentname: 部门名称）
- 流程实例表：`workflow_requestbase`（requestid: 流程ID, requestname: 流程标题, workflowid: 流程类型ID, createdate: 创建日期）
- 待办节点表：`workflow_currentoperator`（requestid: 流程ID, userid: 处理人ID, isremark: 0为待办未读）

## 2. 查数原则
- 只能使用 SELECT 语句，严禁跨库或写入。
- 必须带限制条数（如 `LIMIT 20`）。
- 查询张三当前未读待办的 SQL 示例：
  ```sql
  SELECT count(1) AS todo_count 
  FROM workflow_currentoperator o 
  JOIN hrmresource h ON o.userid = h.id 
  WHERE h.lastname = '张三' AND o.isremark IN ('0','1','8','9');
  ```
- 查询部门流程发起排行的 SQL 示例：
  ```sql
  SELECT d.departmentname, count(r.requestid) AS process_count 
  FROM workflow_requestbase r 
  JOIN hrmdepartment d ON r.createrdepartmentid = d.id 
  GROUP BY d.departmentname ORDER BY process_count DESC LIMIT 10;
  ```
