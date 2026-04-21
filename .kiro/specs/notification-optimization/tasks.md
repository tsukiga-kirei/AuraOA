# Implementation Plan: Notification Optimization

## Overview

对通知系统进行全面优化，涵盖后端枚举中文映射、通知 body 格式化、前端评分颜色标注、AppHeader 铃铛行为变更、独立消息页面、一键已读、国际化支持。后端使用 Go，前端使用 Nuxt 3 / Vue 3 / TypeScript。

## Tasks

- [x] 1. 后端枚举映射模块与通知 body 格式化
  - [x] 1.1 新建 `go-service/internal/pkg/label/label.go`，实现 `RecommendationZh` 和 `ComplianceZh` 映射函数
    - 创建 `label` 包，提供 recommendation（approve→通过、return→退回、review→人工复核）和 compliance（compliant→合规、non_compliant→不合规、partially_compliant→部分合规）的中文映射
    - 未知值原样返回
    - 使用中文注释，与现有 Go 代码风格一致
    - _Requirements: 1.1, 1.3, 2.1, 2.3, 9.1, 9.3_

  - [ ]* 1.2 编写 `RecommendationZh` 和 `ComplianceZh` 的属性测试（Property 1: 已知值映射正确性）
    - **Property 1: Enum-to-Chinese mapping correctness**
    - 新建 `go-service/internal/pkg/label/label_test.go`
    - 使用 `testing/quick` 验证所有已知枚举值映射为正确的中文，且结果不等于原始英文输入
    - 标签：`Feature: notification-optimization, Property 1: Enum-to-Chinese mapping correctness`
    - **Validates: Requirements 1.1, 2.1**

  - [ ]* 1.3 编写未知值 fallback 的属性测试（Property 2: 未知值透传）
    - **Property 2: Unknown value passthrough**
    - 在 `go-service/internal/pkg/label/label_test.go` 中追加
    - 使用 `testing/quick` 验证任意非已知映射的字符串输入，函数返回原始值不变
    - 标签：`Feature: notification-optimization, Property 2: Unknown value passthrough`
    - **Validates: Requirements 1.3, 2.3**

  - [x] 1.4 修改 `go-service/internal/service/audit_review_service.go` 中审核完成通知 body 格式
    - 导入 `label` 包
    - 将 `fmt.Sprintf("评分 %d，建议：%s", parsed.OverallScore, parsed.Recommendation)` 改为 `fmt.Sprintf("建议：%s，评分 %d", label.RecommendationZh(parsed.Recommendation), parsed.OverallScore)`
    - 当 `RecommendationZh` 返回值等于原始值时（未知值），记录 `log.Warn`
    - _Requirements: 1.1, 1.2, 1.3, 3.1, 3.2, 9.1, 9.3_

  - [x] 1.5 修改 `go-service/internal/service/archive_review_service.go` 中归档复盘通知 body 格式
    - 导入 `label` 包
    - 将 `fmt.Sprintf("合规性：%s，评分 %d", parsed.OverallCompliance, parsed.OverallScore)` 改为 `fmt.Sprintf("合规性：%s，评分 %d", label.ComplianceZh(parsed.OverallCompliance), parsed.OverallScore)`
    - 当 `ComplianceZh` 返回值等于原始值时（未知值），记录 `log.Warn`
    - _Requirements: 2.1, 2.2, 2.3, 3.1, 3.3, 9.1, 9.3_

  - [ ]* 1.6 编写通知 body 格式的属性测试（Property 5: 评分在末尾）
    - **Property 5: Notification body format — score at end**
    - 在 `go-service/internal/pkg/label/label_test.go` 中追加
    - 验证使用 `fmt.Sprintf("建议：%s，评分 %d", ...)` 和 `fmt.Sprintf("合规性：%s，评分 %d", ...)` 生成的 body 字符串均以 `评分 {score}` 结尾
    - 标签：`Feature: notification-optimization, Property 5: Notification body format — score at end`
    - **Validates: Requirements 3.1, 3.2, 3.3**

- [x] 2. Checkpoint - 后端改动验证
  - Ensure all tests pass, ask the user if questions arise.

- [x] 3. 前端评分颜色工具与国际化基础
  - [x] 3.1 新建 `frontend/utils/scoreColor.ts`，实现 `extractScore` 和 `scoreColor` 函数
    - `extractScore(body: string): number | null` — 通过正则 `/评分\s*(\d+)/` 从 body 提取评分
    - `scoreColor(score: number): string` — ≥80 返回 `var(--color-success)`，≥60 返回 `var(--color-warning)`，<60 返回 `var(--color-danger)`
    - 使用中文注释，与现有 composable 风格一致
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 9.2_

  - [ ]* 3.2 编写 `scoreColor` 的属性测试（Property 3: 颜色分类完备性）
    - **Property 3: Score color classification completeness**
    - 新建 `frontend/utils/__tests__/scoreColor.test.ts`
    - 安装 `fast-check` 和 `vitest` 开发依赖（如尚未安装）
    - 使用 `fast-check` 验证任意整数 score，`scoreColor` 恰好返回三种颜色之一，且三个区间互斥完备
    - 标签：`Feature: notification-optimization, Property 3: Score color classification completeness`
    - **Validates: Requirements 4.1, 4.2, 4.3**

  - [ ]* 3.3 编写 `extractScore` 的属性测试（Property 4: 正则提取往返一致性）
    - **Property 4: Score regex extraction round-trip**
    - 在 `frontend/utils/__tests__/scoreColor.test.ts` 中追加
    - 使用 `fast-check` 验证：对包含 `评分 N` 模式的字符串，`extractScore` 返回 N；对不包含该模式的字符串，返回 null
    - 标签：`Feature: notification-optimization, Property 4: Score regex extraction round-trip`
    - **Validates: Requirements 4.4**

  - [x] 3.4 更新 `frontend/locales/zh-CN.ts`，新增/修改 i18n 键
    - 修改 `notifications.category.audit` 为 `审核工作台`
    - 新增 `messages.title`（消息中心）、`messages.empty`（暂无消息）、`messages.emptyDetail`（请从左侧列表选择一条消息查看详情）、`messages.markAllRead`（全部已读）、`messages.noMessages`（暂无消息记录）
    - 新增 `messages.score.high`（高分）、`messages.score.medium`（中等）、`messages.score.low`（低分）
    - _Requirements: 5.1, 8.1, 8.2, 8.4_

  - [x] 3.5 更新 `frontend/locales/en-US.ts`，新增/修改对应英文 i18n 键
    - 修改 `notifications.category.audit` 为 `Audit Workbench`
    - 新增与 zh-CN 对应的所有英文翻译键
    - _Requirements: 5.2, 8.1, 8.2, 8.4_

- [x] 4. AppHeader 铃铛行为变更
  - [x] 4.1 修改 `frontend/components/AppHeader.vue`，将铃铛点击行为从打开下拉面板改为 `navigateTo('/messages')`
    - 移除通知下拉面板（`<a-dropdown>` 包裹的通知列表部分）
    - 保留 `<a-badge>` 未读角标
    - 铃铛按钮点击直接调用 `navigateTo('/messages')`
    - 移除 `notifOpen`、`onNotifItemClick`、`handleMarkAllNotificationsRead` 等不再需要的逻辑
    - 保留 `categoryLabel` 函数（消息页面可能复用）
    - 移除通知面板相关的 `<style scoped>` 样式（`.notif-panel` 等）
    - _Requirements: 5.3, 6.1, 9.2_

- [x] 5. 独立消息页面
  - [x] 5.1 新建 `frontend/pages/messages.vue`，实现左右分栏消息页面
    - 路由 `/messages`（Nuxt 文件系统路由自动注册）
    - 复用 `useNotifications` composable 获取 `items`、`unreadCount`、`listLoading`、`refreshList`、`markOneRead`、`markAllRead`、`formatRelative`
    - 使用 `useI18n` 的 `t()` 获取所有 UI 文案
    - 左侧 30% 宽度展示消息列表：分类标签（通过 `categoryLabel` 转换）、标题、正文摘要、相对时间
    - 右侧 70% 宽度展示选中消息详情：标题、分类、时间、正文内容
    - 未读消息在左侧列表以加粗或未读标记点视觉区分
    - 未选中消息时右侧展示空状态提示文案（`messages.emptyDetail`）
    - 页面加载时调用 `refreshList()` 拉取消息列表
    - _Requirements: 6.1, 6.2, 6.3, 6.5, 6.6, 6.7, 6.8, 8.3, 9.2, 9.4_

  - [x] 5.2 实现消息选中与自动标记已读逻辑
    - 点击左侧消息条目时，`selectedId` 更新为该消息 ID
    - 若消息未读，自动调用 `markOneRead(item.id)` 标记已读
    - 右侧详情区域展示选中消息的完整内容
    - _Requirements: 6.3, 6.4_

  - [x] 5.3 在消息详情区域对评分文本应用颜色标注
    - 使用 `extractScore` 从 body 提取评分
    - 使用 `scoreColor` 获取对应颜色
    - 将 body 中的 `评分 {数字}` 部分渲染为带颜色的 `<span>` 标签
    - 无法匹配评分时 body 原样展示
    - 左侧列表的 body 摘要也应用相同的颜色标注
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 6.9_

  - [x] 5.4 实现"全部已读"功能
    - 消息列表区域顶部提供"全部已读"按钮，文案通过 `t('messages.markAllRead')` 获取
    - 点击调用 `markAllRead()` API
    - 成功后更新左侧列表所有消息已读状态，`unreadCount` 清零
    - 当前无未读消息时隐藏或禁用该按钮
    - _Requirements: 7.1, 7.2, 7.3, 7.4_

- [x] 6. Checkpoint - 前端功能验证
  - Ensure all tests pass, ask the user if questions arise.

- [x] 7. 集成与代码风格收尾
  - [x] 7.1 检查所有新增 Go 代码的注释和日志风格一致性
    - 确保 `label.go` 使用中文注释
    - 确保 `audit_review_service.go` 和 `archive_review_service.go` 中新增的 `log.Warn` 使用 `fmt.Sprintf` 格式化
    - _Requirements: 9.1, 9.3_

  - [x] 7.2 检查所有新增前端代码的注释和 composable 导出模式一致性
    - 确保 `scoreColor.ts` 使用中文注释（`/** ... */` 格式）
    - 确保 `messages.vue` 中的注释风格与现有组件一致
    - _Requirements: 9.2, 9.4_

  - [ ]* 7.3 编写消息页面的单元测试
    - 验证空状态渲染、全部已读按钮状态、分类标签 i18n 键值正确
    - _Requirements: 6.7, 7.4, 5.3_

- [x] 8. Final checkpoint - 全部验证
  - Ensure all tests pass, ask the user if questions arise.

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties (Properties 1–5 from design)
- 后端属性测试使用 Go `testing/quick`，前端属性测试使用 `fast-check` + `vitest`
- 数据库 schema 无变更，仅后端 body 生成逻辑和前端展示逻辑改动
