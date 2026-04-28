# Implementation Tasks — excel-import-export

## Task Overview

基于需求文档和设计文档，以下任务按依赖顺序排列。后端基础包先行，再实现各模块 handler，最后完成前端改动。

---

## Tasks

- [x] 1. 后端：新增 `pkg/excel` 公共包
  - [x] 1.1 创建 `go-service/internal/pkg/excel/i18n.go`，定义 `Locale`、`ExportType`、`EnumType` 类型，实现 `ResolveLocale()`、`ColHeaders()`、`TranslateEnum()` 函数，包含完整的列名和枚举翻译映射表（zh-CN / en-US）
  - [x] 1.2 创建 `go-service/internal/pkg/excel/exporter.go`，定义 `ExportConfig` 结构体，实现 `WriteExcel()` 函数（基于 excelize，设置 Content-Type 和 RFC 5987 UTF-8 编码的 Content-Disposition）
  - [x] 1.3 创建 `go-service/internal/pkg/excel/importer.go`，定义 `MemberRow`、`ImportError` 类型，实现 `ParseMemberImport()` 函数（解析第一个 Sheet 从第二行开始，支持文件大小校验）
  - [x] 1.4 使用 excelize 生成两个导入模板文件（`member_import_zh.xlsx` 含中文列头和示例行，`member_import_en.xlsx` 含英文列头和示例行），存放至 `go-service/internal/pkg/excel/templates/`，并在包中添加 `//go:embed templates/*.xlsx` 嵌入声明
  - [x] 1.5 在 `go-service/go.mod` 中添加 `github.com/xuri/excelize/v2` 依赖，执行 `go mod tidy`

- [x] 2. 后端：审核工作台导出
  - [x] 2.1 在 `go-service/internal/service/audit_review_service.go` 中新增 `ListAllProcesses()` 方法（复用 `ListProcessesPaged` 的查询逻辑，去除分页限制，返回全量数据）
  - [x] 2.2 在 `go-service/internal/handler/audit_review_handler.go` 中新增 `ExportProcesses()` handler，调用 `ResolveLocale`、`ListAllProcesses`、`WriteExcel`，根据 tab 参数选择 `ExportTypeAuditUnaudited` 或 `ExportTypeAuditCompleted`
  - [x] 2.3 在 `go-service/internal/router/router.go` 的 `audit` 路由组中注册 `GET /processes/export` 路由

- [x] 3. 后端：归档复盘导出
  - [x] 3.1 在 `go-service/internal/service/archive_review_service.go` 中新增 `ListAllProcesses()` 方法（复用 `ListProcessesPaged` 的查询逻辑，去除分页限制）
  - [x] 3.2 在 `go-service/internal/handler/archive_review_handler.go` 中新增 `ExportProcesses()` handler，根据 `audit_status` 参数选择 `ExportTypeArchiveUnaudited` 或 `ExportTypeArchiveReviewed`
  - [x] 3.3 在 `go-service/internal/router/router.go` 的 `archive` 路由组中注册 `GET /processes/export` 路由

- [x] 4. 后端：组织人员导入
  - [x] 4.1 在 `go-service/internal/service/org_service.go` 中新增 `ImportMembers()` 方法，实现：查询系统默认密码、逐行校验（部门/角色存在性、用户名唯一性）、批量创建用户并分配部门角色、返回 `ImportMembersResult`
  - [x] 4.2 在 `go-service/internal/dto/org_dto.go` 中新增 `ImportMembersResult` 和 `ImportRowError` DTO 结构体
  - [x] 4.3 在 `go-service/internal/handler/org_handler.go` 中新增 `ImportMembers()` handler（multipart 文件接收、大小校验、调用 service、返回结果）和 `DownloadImportTemplate()` handler（读取 embed.FS 模板、按 locale 参数选择语言版本）
  - [x] 4.4 在 `go-service/internal/router/router.go` 的 `tenantOrg` 路由组中注册 `POST /members/import` 和 `GET /members/import-template` 路由

- [x] 5. 后端：用户偏好导出
  - [x] 5.1 在 `go-service/internal/handler/user_config_management_handler.go` 中新增 `ExportUserConfigs()` handler，复用 `ListUserConfigs` 的数据构建逻辑，调用 `WriteExcel` 输出 `ExportTypeUserConfig` 格式的 Excel
  - [x] 5.2 在 `go-service/internal/router/router.go` 的 `tenantUserConfigs` 路由组中注册 `GET /export` 路由

- [x] 6. 前端：移除 xlsx 依赖 & 公共工具
  - [x] 6.1 从 `frontend/package.json` 中移除 `"xlsx": "^0.18.5"` 依赖，执行 `pnpm remove xlsx`
  - [x] 6.2 在 `frontend/locales/zh-CN.ts` 和 `frontend/locales/en-US.ts` 中新增导出/导入相关 i18n key（`export.excel`、`export.exporting`、`export.success`、`export.failed`、`export.report`、`org.import.*`、`userConfig.export.button`）

- [x] 7. 前端：审核工作台导出
  - [x] 7.1 在 `frontend/composables/useAuditApi.ts` 中新增 `exportProcesses(tab, params)` 函数，复用 `buildExportUrl` 和 `triggerDownload` 模式（参考 `useAdminDataApi.ts`）
  - [x] 7.2 在 `frontend/pages/dashboard.vue` 中，在每个页签工具栏右侧新增「导出 Excel」按钮，绑定 loading 状态，点击时调用 `exportProcesses(currentTab, currentFilters)`

- [x] 8. 前端：归档复盘导出 & 单流程导出简化
  - [x] 8.1 在 `frontend/composables/useArchiveReviewApi.ts` 中新增 `exportProcesses(params)` 函数
  - [x] 8.2 在 `frontend/pages/archive.vue` 中，在页签工具栏新增「导出 Excel」按钮，调用 `exportProcesses`
  - [x] 8.3 在 `frontend/pages/archive.vue` 中，修改单流程导出逻辑：移除 CSV/Excel 格式选项，将下拉菜单改为单一「导出报告」按钮，直接触发 JSON 导出

- [x] 9. 前端：组织人员导入
  - [x] 9.1 在 `frontend/composables/useOrgApi.ts` 中新增 `importMembers(file)` 和 `downloadImportTemplate(locale)` 函数，定义 `ImportMembersResult` 和 `ImportRowError` 类型
  - [x] 9.2 在组织人员管理页面（`frontend/pages/admin/tenant/org.vue` 或对应组件）的成员列表工具栏新增「导入成员」按钮（隐藏 file input，触发文件选择）和「下载模板」按钮
  - [x] 9.3 实现导入结果弹窗（`a-modal`），展示成功数量和失败行列表（行号 + 原因），支持 i18n

- [x] 10. 前端：用户偏好导出优化
  - [x] 10.1 在 `frontend/composables/useAdminUserConfigApi.ts` 中新增 `exportUserConfigs()` 函数
  - [x] 10.2 在用户偏好页面（`frontend/pages/admin/tenant/user-configs.vue` 或对应组件）中，移除 checkbox 多选 UI 和前端 xlsx 导出逻辑，新增「导出 Excel」按钮调用 `exportUserConfigs()`
