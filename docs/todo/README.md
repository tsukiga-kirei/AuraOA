# TODO 文档目录

本目录用于存放待办事项、改进点及技术债务相关的说明文档，便于后续排期与实现参考。

## 文档列表

| 文档 | 说明 |
|------|------|
| [business-todo.md](./business-todo.md) | **业务功能待办**：审核工作台、定时任务执行、归档复盘、仪表盘、消息通知等核心业务模块的后端集成计划 |
| [detail-todo.md](./detail-todo.md) | **细节改进待办**：默认管理员账号、提示词模板格式约束、前端分页性能、国际化完善、UI/UX 优化、安全相关 |
| [technical-todo.md](./technical-todo.md) | **技术改造计划**：后端分页、Redis 扩展、消息队列、OCR 附件识别、Python AI 服务、监控可观测性、多实例部署等 |
| [personal-config-dirty-data-cleanup.md](./personal-config-dirty-data-cleanup.md) | **脏数据清理**：个人配置中因租户修改而产生的无效数据现状、改进点及实现注意事项 |
| [../config-data-flow.md](../config-data-flow.md) | **配置数据流转全景**：租户配置、用户个人配置在数据库存储、后端处理、前端传输三层的完整数据结构和流转逻辑 |

## 优先级总览

| 优先级 | 内容 | 文档 |
|--------|------|------|
| 🔴 P0 | 后端分页（日志/成员列表）、Redis API 限流 | [技术改造](./technical-todo.md) |
| 🟠 P1 | 审核执行完整链路、异步任务队列 | [业务待办](./business-todo.md) + [技术改造](./technical-todo.md) |
| 🟡 P2 | 定时任务引擎、邮件发送、仪表盘数据、归档执行 | [业务待办](./business-todo.md) |
| 🟡 P2 | 默认账号安全、提示词模板校验 | [细节改进](./detail-todo.md) |
| 🟡 P2 | Redis Token 缓存、Python AI 服务 | [技术改造](./technical-todo.md) |
| 🟢 P3 | OCR 附件、监控、备份、多实例 | [技术改造](./technical-todo.md) |
| 🟢 P3 | 脏数据清理工具 | [脏数据清理](./personal-config-dirty-data-cleanup.md) |

## 使用说明

- 每个 TODO 文档应独立成文，描述当前问题、改进点及潜在风险。
- 实现完成后可移至 `docs/` 归档或标记为已完成。
- 优先级说明：🔴 P0（立即）、🟠 P1（近期）、🟡 P2（中期）、🟢 P3（后期）。
