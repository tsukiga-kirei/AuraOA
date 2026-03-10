# 泛微 Ecology E9 — OA 适配器 SQL 参考

> 对应代码：`go-service/internal/pkg/oa/ecology9.go`
>
> 泛微 E9 底层数据库支持 MySQL 和 Oracle，下面按功能列出所有 SQL，并标注两种数据库的差异。

---

## 1. ValidateProcess — 验证流程是否存在

### 1.1 查询流程定义

```sql
-- MySQL / Oracle 通用
SELECT * FROM workflow_base WHERE workflowname = ? LIMIT 1;
```

Oracle 等价写法（GORM 自动处理）：

```sql
SELECT * FROM workflow_base WHERE workflowname = ? AND ROWNUM <= 1;
```

### 1.2 统计明细表数量

```sql
-- MySQL / Oracle 通用
SELECT COUNT(DISTINCT detailtable)
  FROM workflow_billfield
 WHERE billid = ? AND detailtable > 0;
```

---

## 2. FetchFields — 拉取流程字段定义

### 2.1 查询流程定义（同 1.1）

```sql
SELECT * FROM workflow_base WHERE workflowname = ? LIMIT 1;
```

### 2.2 查询所有字段

```sql
-- MySQL / Oracle 通用
SELECT *
  FROM workflow_billfield
 WHERE billid = ?
 ORDER BY detailtable ASC, id ASC;
```

---

## 3. CheckUserPermission — 校验用户审批权限

### 3.1 查询流程定义（同 1.1）

```sql
SELECT * FROM workflow_base WHERE workflowname = ? LIMIT 1;
```

### 3.2 统计用户操作记录

```sql
-- MySQL / Oracle 通用
SELECT COUNT(*)
  FROM workflow_currentoperator
 WHERE workflowid = ? AND userid = ?;
```

---

## 4. FetchProcessData — 拉取流程实例业务数据

### 4.1 查询流程请求基本信息

```sql
-- MySQL / Oracle 通用
SELECT requestid, workflowid, requestname
  FROM workflow_requestbase
 WHERE requestid = ?
 LIMIT 1;
```

### 4.2 查询流程定义（按 ID）

```sql
SELECT * FROM workflow_base WHERE id = ? LIMIT 1;
```

### 4.3 查询主表数据

```sql
-- {tablename} 为 workflow_base.tablename 动态值
SELECT * FROM {tablename} WHERE requestid = ? LIMIT 1;
```

### 4.4 统计明细表数量（同 1.2）

```sql
SELECT COUNT(DISTINCT detailtable)
  FROM workflow_billfield
 WHERE billid = ? AND detailtable > 0;
```

### 4.5 查询明细表数据

```sql
-- MySQL / Oracle 通用（使用 EXISTS 子查询，兼容两种数据库）
SELECT *
  FROM {tablename}_dt{i}
 WHERE EXISTS (
    SELECT 1
      FROM {tablename} m
     WHERE m.id = {tablename}_dt{i}.mainid
       AND m.requestid = ?
 );
```

> 早期版本使用 `IN (SELECT ...)` 写法，Oracle 下存在隐式类型转换问题，
> 已统一改为 `EXISTS` 子查询。

---

## 涉及的 E9 表汇总

| 表名 | 用途 | 使用方法 |
|---|---|---|
| `workflow_base` | 流程定义（名称、表单ID、主表名） | ValidateProcess / FetchFields / CheckUserPermission / FetchProcessData |
| `workflow_billfield` | 表单字段定义（字段名、类型、明细表归属） | ValidateProcess / FetchFields / FetchProcessData |
| `workflow_currentoperator` | 流程当前操作人（待办/已办） | CheckUserPermission |
| `workflow_requestbase` | 流程请求实例（requestid ↔ workflowid） | FetchProcessData |
| `{tablename}` | 流程主表（动态表名，来自 workflow_base.tablename） | FetchProcessData |
| `{tablename}_dt{N}` | 流程明细表（动态表名，N 为明细表序号） | FetchProcessData |

---

## MySQL vs Oracle 差异备注

| 差异点 | MySQL | Oracle | 代码处理方式 |
|---|---|---|---|
| LIMIT 语法 | `LIMIT 1` | `ROWNUM <= 1` / `FETCH FIRST 1 ROWS ONLY` | GORM 自动适配 |
| 子查询 IN 隐式转换 | 正常 | 可能类型不匹配 | 统一使用 EXISTS |
| 字符串比较 | 大小写取决于 collation | 默认大小写敏感 | 暂未特殊处理，E9 字段名通常小写 |
| DSN 格式 | `user:pass@tcp(host:port)/db` | `oracle://user:pass@host:port/service` | `ecology9.go` 按 driver 分支构建 |
