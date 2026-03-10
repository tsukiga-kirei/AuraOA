# 泛微 Ecology E9 — OA 适配器 SQL 参考

> 对应代码：`go-service/internal/pkg/oa/ecology9.go`
>
> 泛微 E9 底层数据库支持 MySQL、Oracle 和 DM（达梦），下面按功能列出所有 SQL，并标注数据库差异。
>
> **标识符大小写**：Oracle / DM 默认将未加引号的标识符转为大写，泛微 E9 在 Oracle / DM 上的表名和列名均为大写。Oracle 驱动通过 `IgnoreCase=true` + `NamingCaseSensitive=false` 配置使 GORM 不给标识符加双引号，Oracle 自动将标识符转为大写匹配；业务层另有 `tableName()` 辅助方法在 Oracle / DM 下将表名列名显式转大写，MySQL 保持原样。

---

## 1. ValidateProcess — 验证流程是否存在

### 1.1 查询流程定义

使用 `Row().Scan()` 显式扫描列值，避免 GORM struct tag 大小写映射问题（Oracle/DM 列名大写）。

```sql
-- MySQL
SELECT workflowname, formid, workflowtype FROM workflow_base WHERE workflowname = ? AND isvalid = '1' LIMIT 1;

-- Oracle / DM（tableName() / col() 自动转大写）
SELECT WORKFLOWNAME, FORMID, WORKFLOWTYPE FROM WORKFLOW_BASE WHERE WORKFLOWNAME = ? AND ISVALID = '1' AND ROWNUM <= 1;
```

> 直接在查询条件中加 `isvalid = '1'`，查不到记录即表示流程不存在或已停用。

### 1.2 查询流程类型分类名称

通过 `workflow_base.workflowtype` 关联 `workflow_type.id`，获取 `workflow_type.typename` 作为流程类型显示名称。

```sql
-- MySQL
SELECT typename FROM workflow_type WHERE id = ? LIMIT 1;

-- Oracle / DM
SELECT TYPENAME FROM WORKFLOW_TYPE WHERE ID = ? AND ROWNUM <= 1;
```

### 1.3 查询表单定义（获取真实主表名）

通过 `workflow_base.formid` 关联 `workflow_bill.id`，获取 `workflow_bill.tablename` 作为真实主表名。

```sql
-- MySQL
SELECT tablename FROM workflow_bill WHERE id = ? LIMIT 1;

-- Oracle / DM
SELECT TABLENAME FROM WORKFLOW_BILL WHERE ID = ? AND ROWNUM <= 1;
```

> `workflow_bill` 查询失败时直接返回错误，不再降级使用 `workflow_base.tablename`。
>
> 当前端传入 `main_table_name` 或 `process_type_label` 时，Service 层会将其与实际系统里的数据做忽略大小写比较：
> - 一致：正常返回
> - 不一致：返回 mismatch=true 加上 expected_xxx（正确名称），前端自动纠正

---

## 2. FetchFields — 拉取流程字段定义

### 2.1 查询流程定义

> 与 1.1 不同，此处不过滤 `isvalid`，仅按流程名称查询。

```sql
-- MySQL
SELECT * FROM workflow_base WHERE workflowname = ? LIMIT 1;

-- Oracle / DM
SELECT * FROM WORKFLOW_BASE WHERE WORKFLOWNAME = ? AND ROWNUM <= 1;
```

### 2.2 查询所有字段及国际化显示名

泛微 E9 将实际展示的中文字段名存放在 `htmllabelinfo` 中，需要通过 `fieldlabel` 关联查询（`languageid = 7` 代表简体中文）。

```sql
-- MySQL / Oracle 通用
SELECT
    T1.FIELDNAME AS FIELDKEY,
    T2.LABELNAME AS FIELDNAME,
    T1.FIELDHTMLTYPE AS FIELDHTMLTYPE,
    T1.DETAILTABLE AS DETAILTABLE
FROM WORKFLOW_BILLFIELD T1
JOIN HTMLLABELINFO T2 ON T1.FIELDLABEL = T2.INDEXID
WHERE T1.BILLID = ? AND T2.LANGUAGEID = 7
ORDER BY T1.DETAILTABLE ASC, T1.ID ASC;
```

---

## 3. CheckUserPermission — 校验用户审批权限

### 3.1 查询流程定义

> 与 1.1 不同，此处不过滤 `isvalid`，仅按流程名称查询。
>
> 使用 `Row().Scan()` 显式扫描 `id` 列，避免 GORM struct tag 大小写映射问题（Oracle/DM 列名大写）。

```sql
-- MySQL
SELECT id FROM workflow_base WHERE workflowname = ? LIMIT 1;

-- Oracle / DM（tableName() / col() 自动转大写）
SELECT ID FROM WORKFLOW_BASE WHERE WORKFLOWNAME = ? AND ROWNUM <= 1;
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
| `workflow_base` | 流程定义（名称、表单ID、流程类型ID、主表名、isvalid 启停状态） | ValidateProcess / FetchFields / CheckUserPermission / FetchProcessData |
| `workflow_type` | 流程类型定义（类型ID → 分类名称 typename） | ValidateProcess（获取流程类型显示名称） |
| `workflow_bill` | 表单定义（表单ID → 关联主表名 tablename） | ValidateProcess（获取真实主表名） |
| `workflow_billfield` | 表单字段定义（字段名、类型、明细表归属） | FetchFields / FetchProcessData |
| `workflow_currentoperator` | 流程当前操作人（待办/已办） | CheckUserPermission |
| `workflow_requestbase` | 流程请求实例（requestid ↔ workflowid） | FetchProcessData |
| `{tablename}` | 流程主表（动态表名，来自 workflow_base.tablename） | FetchProcessData |
| `{tablename}_dt{N}` | 流程明细表（动态表名，N 为明细表序号） | FetchProcessData |

---

## MySQL vs Oracle / DM 差异备注

| 差异点 | MySQL | Oracle / DM | 代码处理方式 |
|---|---|---|---|
| 标识符大小写 | 不区分大小写 | 默认大写 | Oracle 驱动 `IgnoreCase=true` + `NamingCaseSensitive=false` 使 GORM 不加双引号，Oracle 自动转大写；业务层 `tableName()` 方法辅助显式转大写 |
| LIMIT 语法 | `LIMIT 1` | `ROWNUM <= 1` / `FETCH FIRST 1 ROWS ONLY` | GORM 自动适配 |
| 子查询 IN 隐式转换 | 正常 | 可能类型不匹配 | 统一使用 EXISTS |
| 字符串比较 | 大小写取决于 collation | 默认大小写敏感 | 业务层暂未特殊处理 |
| DSN 格式 | `user:pass@tcp(host:port)/db` | `oracle://user:pass@host:port/service` | `ecology9.go` 按 driver 分支构建 |
| DM（达梦） | — | 使用 Oracle 兼容模式 | `isOracleCompatible()` 统一判断 oracle / dm |
