# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- 新增附件信息数据模型 `AttachmentInfo`，包含附件 ID、文件名、文件类型、文件大小、所属字段、提取内容等字段
- 在 `ProcessData` 结构体中新增 `Attachments` 字段，用于存储流程附件信息
- 在 `Ecology9Adapter` 中集成 `AttachmentRecognitionService` 接口，支持附件识别服务注入

### Changed

### Fixed

### In Progress
- [#001] 流程附件未被 AI 识别 - 数据模型扩展已完成，E9 适配器接口集成已完成，待实现附件查询逻辑与 Prompt 注入
- [#002] 审核数据保留天数未实现自动清理 - 配置项已存在，待实现按 `data_retention_days` 清理库表审核/归档数据，详见 [docs/known-issues/002-audit-data-retention-cleanup.md](docs/known-issues/002-audit-data-retention-cleanup.md)

---

## 版本说明

- **Unreleased**: 开发中的变更，尚未发布
- 版本号格式：`[主版本.次版本.修订号] - YYYY-MM-DD`
