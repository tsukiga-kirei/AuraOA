# Requirements Document

## Introduction

本功能模块为 OA 智审平台新增和优化一套完整的 Excel 导入/导出能力，涵盖以下四个子模块：

1. **审核工作台 & 归档复盘 - 导出 Excel**：支持按当前筛选条件全量导出各页签数据为 Excel 文件，列名和枚举值根据用户语言动态转换。
2. **归档复盘 - 导出报告简化**：移除单流程的 CSV/Excel 导出入口，统一改为直接导出 JSON 格式报告。
3. **组织人员 - 导入 Excel**：支持通过 Excel 模板批量导入成员，使用系统默认密码，提供可下载的标准模板文件。
4. **用户偏好 - 导出优化**：将用户偏好页的导出方式统一为与"数据信息"模块相同的后端流式导出，移除前端 checkbox 选择逻辑和前端 XLSX 依赖。

技术栈：前端 Nuxt 3 / Vue 3（Ant Design Vue 4），后端 Go（Gin + GORM），数据库 PostgreSQL，缓存 Redis，已有国际化支持（zh-CN / en-US）。

---

## Glossary

- **Exporter**：后端导出服务，负责查询全量数据、构建 Excel 文件并通过 HTTP 响应流返回。
- **Importer**：后端导入服务，负责解析上传的 Excel 文件、校验数据并批量写入数据库。
- **AuditWorkbench**：审核工作台前端页面（`/dashboard`），展示待审核/已完成等多个页签的 OA 流程列表。
- **ArchiveReview**：归档复盘前端页面（`/archive`），展示已归档流程的复盘结果。
- **DataInfo**：数据信息模块（`/admin/tenant/data`），已有后端流式导出实现，作为本次导出的参考基准。
- **OrgAdmin**：组织人员管理页面（`/admin/tenant/org`），管理成员、角色、部门。
- **UserPrefs**：用户偏好页面（`/settings`），用户个人配置页。
- **AuditTab**：审核工作台的页签，包括"待审核（unaudited）"、"已完成（completed）"等。
- **ArchiveTab**：归档复盘的页签，包括"未复核（unaudited）"、"合规（compliant）"、"部分合规（partially_compliant）"、"不合规（non_compliant）"。
- **FilterParams**：当前页面生效的筛选条件集合，包括关键词、申请人、流程类型、部门、日期范围等。
- **EnumTranslator**：枚举值翻译器，将数据库存储的原始枚举值（如 `compliant`、`pending`）转换为对应语言的可读文本。
- **ImportTemplate**：Excel 导入模板文件，预定义列头和示例数据，长期存储于服务器静态资源目录。
- **DefaultPassword**：系统默认密码，由系统配置（`system_configs`）统一管理，批量导入成员时使用。
- **Locale**：用户语言偏好，取值为 `zh-CN`（中文）或 `en-US`（英文），从用户会话或请求头中读取。
- **ContentDisposition**：HTTP 响应头字段，用于指定下载文件名，支持 RFC 5987 UTF-8 编码。

---

## Requirements

### Requirement 1：审核工作台 - 按页签导出 Excel

**User Story：** As a tenant_admin or business user，I want to export all audit process records of the current tab as an Excel file based on active filter conditions，so that I can perform offline analysis and archiving.

#### Acceptance Criteria

1. WHEN a user clicks the export button on the AuditWorkbench page，THE Exporter SHALL query all records matching the current FilterParams for the active AuditTab without pagination limits.
2. WHEN the export request is received，THE Exporter SHALL generate an Excel file (.xlsx) containing all matching records as a single sheet.
3. WHEN the user's Locale is `zh-CN`，THE Exporter SHALL use Chinese column headers and translate enum field values to Chinese text.
4. WHEN the user's Locale is `en-US`，THE Exporter SHALL use English column headers and translate enum field values to English text.
5. THE Exporter SHALL apply EnumTranslator to all enum fields (such as audit_status, compliance level, process_type) so that raw database values are never written directly into the exported file.
6. WHEN the active AuditTab is "unaudited"，THE Exporter SHALL include only the columns relevant to unaudited records (e.g., process title, applicant, department, submit time, process type).
7. WHEN the active AuditTab is "completed"，THE Exporter SHALL include columns relevant to completed records (e.g., process title, applicant, audit result, score, audited time) in addition to base columns.
8. THE Exporter SHALL set the HTTP response Content-Type to `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`.
9. THE Exporter SHALL set the ContentDisposition header with a UTF-8 encoded filename containing the tab name and export timestamp (e.g., `audit_unaudited_20240101_120000.xlsx`).
10. IF the query returns zero records，THEN THE Exporter SHALL return an Excel file containing only the header row.
11. THE AuditWorkbench frontend SHALL trigger the export by calling the backend export API with current FilterParams and AuditTab，and SHALL use the same triggerDownload mechanism as DataInfo.
12. THE AuditWorkbench frontend SHALL NOT use any frontend XLSX library to generate the file.

---

### Requirement 2：归档复盘 - 按页签导出 Excel

**User Story：** As a tenant_admin or business user，I want to export all archive review records of the current tab as an Excel file based on active filter conditions，so that I can review compliance results offline.

#### Acceptance Criteria

1. WHEN a user clicks the export button on the ArchiveReview page，THE Exporter SHALL query all records matching the current FilterParams for the active ArchiveTab without pagination limits.
2. WHEN the export request is received，THE Exporter SHALL generate an Excel file (.xlsx) containing all matching records as a single sheet.
3. WHEN the user's Locale is `zh-CN`，THE Exporter SHALL use Chinese column headers and translate enum field values to Chinese text.
4. WHEN the user's Locale is `en-US`，THE Exporter SHALL use English column headers and translate enum field values to English text.
5. THE Exporter SHALL apply EnumTranslator to all enum fields (such as overall_compliance, archive_status) so that raw database values are never written directly into the exported file.
6. WHEN the active ArchiveTab is "unaudited"，THE Exporter SHALL include only the columns relevant to unreviewed records (e.g., process title, applicant, department, archive time, process type).
7. WHEN the active ArchiveTab is one of "compliant"，"partially_compliant"，or "non_compliant"，THE Exporter SHALL include additional columns for review results (e.g., overall compliance, score, confidence, review time).
8. THE Exporter SHALL set the HTTP response Content-Type to `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`.
9. THE Exporter SHALL set the ContentDisposition header with a UTF-8 encoded filename containing the tab name and export timestamp.
10. IF the query returns zero records，THEN THE Exporter SHALL return an Excel file containing only the header row.
11. THE ArchiveReview frontend SHALL trigger the export by calling the backend export API with current FilterParams and ArchiveTab，and SHALL use the same triggerDownload mechanism as DataInfo.
12. THE ArchiveReview frontend SHALL NOT use any frontend XLSX library to generate the file.

---

### Requirement 3：归档复盘 - 移除单流程 CSV/Excel 导出，统一为 JSON 导出

**User Story：** As a business user，I want to export a single archive review result as a JSON file directly，so that I can obtain the complete structured report without choosing between formats.

#### Acceptance Criteria

1. THE ArchiveReview frontend SHALL remove the "Export CSV" menu item from the single-process export dropdown.
2. THE ArchiveReview frontend SHALL remove the "Export Excel" menu item from the single-process export dropdown.
3. WHEN a user triggers the export action for a single process，THE ArchiveReview frontend SHALL directly export the result as a JSON file without presenting a format selection menu.
4. WHEN the JSON export is triggered，THE ArchiveReview frontend SHALL fetch the full result via `GET /api/archive/result/:id` if the result ID is available，and SHALL serialize the combined process and result object to a `.json` file.
5. THE exported JSON filename SHALL follow the pattern `archive-review-{process_id}.json`.
6. IF the current result has no ID (not yet persisted)，THEN THE ArchiveReview frontend SHALL export the in-memory result object directly.

---

### Requirement 4：组织人员 - Excel 批量导入成员

**User Story：** As a tenant_admin，I want to import multiple members at once via an Excel file，so that I can efficiently onboard large numbers of users without manual entry.

#### Acceptance Criteria

1. THE OrgAdmin page SHALL provide an "Import Members" button on the members tab toolbar.
2. WHEN a user clicks "Import Members"，THE OrgAdmin frontend SHALL open a file picker accepting only `.xlsx` and `.xls` files.
3. WHEN a valid Excel file is selected，THE OrgAdmin frontend SHALL upload the file to the backend import API via multipart/form-data POST.
4. THE Importer SHALL parse the uploaded Excel file and extract member records from the first sheet starting from the second row (row 1 is the header).
5. THE Importer SHALL validate each row for required fields: name (姓名/Name)，username (用户名/Username)，department_name (部门/Department)，role_names (角色/Roles).
6. IF a row is missing any required field，THEN THE Importer SHALL record that row as a validation error and continue processing remaining rows.
7. IF a username already exists in the system，THEN THE Importer SHALL record that row as a duplicate error and skip insertion.
8. THE Importer SHALL assign the DefaultPassword to all successfully imported members.
9. WHEN import is complete，THE Importer SHALL return a summary response containing: total rows processed，success count，and a list of failed rows with row number and error reason.
10. WHEN the user's Locale is `zh-CN`，THE OrgAdmin frontend SHALL display the import result summary in Chinese.
11. WHEN the user's Locale is `en-US`，THE OrgAdmin frontend SHALL display the import result summary in English.
12. THE OrgAdmin page SHALL provide a "Download Template" button that triggers download of the ImportTemplate file.
13. THE ImportTemplate file SHALL be stored as a static file on the server (not generated on-the-fly) and SHALL be accessible via a dedicated download endpoint.
14. WHEN the user's Locale is `zh-CN`，THE ImportTemplate SHALL use Chinese column headers.
15. WHEN the user's Locale is `en-US`，THE ImportTemplate SHALL use English column headers.
16. THE ImportTemplate SHALL include at least one example data row to guide users on the expected format.
17. IF the uploaded file is not a valid Excel format，THEN THE Importer SHALL return a 400 error with a descriptive message.
18. IF the uploaded file exceeds the maximum allowed size (5 MB)，THEN THE Importer SHALL return a 413 error.

---

### Requirement 5：用户偏好 - 导出方式统一优化

**User Story：** As a business user，I want to export my preference configuration data using the same backend-driven export as the DataInfo module，so that the export is consistent and does not depend on frontend libraries.

#### Acceptance Criteria

1. THE UserPrefs frontend SHALL remove all checkbox (multi-select) UI elements previously used for selecting records to export.
2. THE UserPrefs frontend SHALL add an export button that triggers a backend export API call with the current FilterParams.
3. WHEN the export button is clicked，THE Exporter SHALL query all records matching the current FilterParams without pagination limits.
4. WHEN the export request is received，THE Exporter SHALL generate an Excel file (.xlsx) and return it as a streaming HTTP response.
5. WHEN the user's Locale is `zh-CN`，THE Exporter SHALL use Chinese column headers and translate enum field values to Chinese text.
6. WHEN the user's Locale is `en-US`，THE Exporter SHALL use English column headers and translate enum field values to English text.
7. THE UserPrefs frontend SHALL use the same triggerDownload mechanism as DataInfo to initiate the file download.
8. THE UserPrefs frontend SHALL NOT use any frontend XLSX library (e.g., the `xlsx` npm package) to generate the export file.
9. THE Exporter SHALL set the ContentDisposition header with a UTF-8 encoded filename.

---

### Requirement 6：国际化通用要求

**User Story：** As a user，I want all exported files and import interfaces to respect my language preference，so that I can work in my preferred language without encountering raw system values.

#### Acceptance Criteria

1. THE Exporter SHALL read the user's Locale from the authenticated session or the `Accept-Language` request header.
2. WHEN generating any Excel export，THE Exporter SHALL use the Locale to select the appropriate column header translation from the i18n resource.
3. THE EnumTranslator SHALL maintain a mapping table for each enum type (audit_status, compliance level, process_type, member status, etc.) in both `zh-CN` and `en-US`.
4. WHEN translating an enum value，IF no translation is found for the given Locale，THEN THE EnumTranslator SHALL fall back to the `zh-CN` translation.
5. WHEN translating an enum value，IF no `zh-CN` translation exists either，THEN THE EnumTranslator SHALL use the raw database value as-is.
6. THE ImportTemplate SHALL be provided in two language variants (zh-CN and en-US)，and THE Importer SHALL serve the appropriate variant based on the requested Locale.

---

### Requirement 7：代码规范与文档

**User Story：** As a developer，I want all new code to follow consistent comment standards and be accompanied by documentation，so that the codebase remains maintainable.

#### Acceptance Criteria

1. THE Exporter backend code SHALL include Go doc comments on all exported functions and types，following the existing project comment style (as seen in handler and service files).
2. THE Importer backend code SHALL include Go doc comments on all exported functions and types.
3. THE frontend composable functions added for export/import SHALL include JSDoc-style block comments describing parameters and return values，consistent with the existing composable comment style (e.g., `useAdminDataApi.ts`).
4. THE feature SHALL include an updated or new API documentation section describing all new endpoints，their request parameters，and response formats.
