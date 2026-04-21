# Requirements Document

## Introduction

对右上角消息通知系统进行全面优化，涵盖以下方面：后端通知内容中文化（合规性、建议字段）、前端评分展示统一与颜色标注、通知分类名称修改、新建独立消息页面（类微信布局）、一键已读功能增强、以及所有新增文案的国际化支持。整体改动需保持与现有代码风格一致的日志输出和注释风格。

## Glossary

- **Notification_System**: 用户通知系统，包含后端通知创建服务（Go）和前端通知展示组件（Vue/Nuxt），负责消息的生成、存储、展示和已读管理
- **AppHeader**: 前端顶部导航栏组件（`frontend/components/AppHeader.vue`），包含通知铃铛图标和通知下拉面板
- **Message_Page**: 新建的独立消息页面（`frontend/pages/messages.vue`），采用左右分栏布局展示消息列表和消息详情
- **Notification_Body**: 通知的正文内容（`body` 字段），由后端服务在审核/归档完成时生成，包含评分、建议/合规性等信息
- **Score_Color**: 根据评分数值范围对评分文本应用不同颜色的视觉标注机制
- **Category_Label**: 通知分类标签，通过 i18n 键映射将后端 category 值转换为用户可读的本地化名称
- **useNotifications**: 前端通知 composable（`frontend/composables/useNotifications.ts`），封装通知列表获取、已读标记等 API 调用
- **Audit_Review_Service**: 后端审核服务（`go-service/internal/service/audit_review_service.go`），在审核完成时生成通知
- **Archive_Review_Service**: 后端归档复盘服务（`go-service/internal/service/archive_review_service.go`），在归档复盘完成时生成通知
- **Locale_Files**: 前端国际化文件（`frontend/locales/zh-CN.ts` 和 `frontend/locales/en-US.ts`），存储所有 UI 文案的中英文翻译

## Requirements

### Requirement 1: 通知正文中建议字段中文化

**User Story:** 作为业务用户，我希望审核完成通知中的"建议"字段以中文显示（如"通过"、"退回"、"人工复核"），以便我能快速理解审核结论而无需翻译英文术语。

#### Acceptance Criteria

1. WHEN Audit_Review_Service 生成审核完成通知时，THE Notification_System SHALL 将 recommendation 值 `approve` 映射为中文"通过"、`return` 映射为"退回"、`review` 映射为"人工复核"后写入 Notification_Body
2. WHEN Notification_Body 中包含建议字段时，THE Notification_System SHALL 以"建议：{中文建议}"的格式展示，替代原有的"建议：approve"等英文格式
3. IF recommendation 值无法匹配已知映射时，THEN THE Notification_System SHALL 将原始值原样写入 Notification_Body 并记录警告日志

### Requirement 2: 通知正文中合规性字段中文化

**User Story:** 作为业务用户，我希望归档复盘通知中的"合规性"字段以中文显示（如"合规"、"不合规"、"部分合规"），以便我能直观了解复盘结论。

#### Acceptance Criteria

1. WHEN Archive_Review_Service 生成归档复盘完成通知时，THE Notification_System SHALL 将 overall_compliance 值 `compliant` 映射为"合规"、`non_compliant` 映射为"不合规"、`partially_compliant` 映射为"部分合规"后写入 Notification_Body
2. WHEN Notification_Body 中包含合规性字段时，THE Notification_System SHALL 以"合规性：{中文合规性}"的格式展示，替代原有的"合规性：non_compliant"等英文格式
3. IF overall_compliance 值无法匹配已知映射时，THEN THE Notification_System SHALL 将原始值原样写入 Notification_Body 并记录警告日志

### Requirement 3: 评分位置统一

**User Story:** 作为业务用户，我希望所有类型通知中的评分信息展示位置保持一致，以便我能快速定位和比较不同通知的评分。

#### Acceptance Criteria

1. THE Notification_System SHALL 在所有包含评分的 Notification_Body 中将评分统一放置在正文末尾位置，格式为"评分 {分数}"
2. WHEN Audit_Review_Service 生成通知时，THE Notification_Body SHALL 按照"建议：{建议}，评分 {分数}"的顺序组织内容
3. WHEN Archive_Review_Service 生成通知时，THE Notification_Body SHALL 按照"合规性：{合规性}，评分 {分数}"的顺序组织内容

### Requirement 4: 评分颜色标注

**User Story:** 作为业务用户，我希望通知中的评分根据分数高低以不同颜色展示，以便我能一眼识别评分等级。

#### Acceptance Criteria

1. WHEN 前端渲染 Notification_Body 中的评分时，THE Notification_System SHALL 对评分 80 分及以上应用绿色（成功色）标注
2. WHEN 前端渲染 Notification_Body 中的评分时，THE Notification_System SHALL 对评分 60 分至 79 分应用橙色（警告色）标注
3. WHEN 前端渲染 Notification_Body 中的评分时，THE Notification_System SHALL 对评分 60 分以下应用红色（危险色）标注
4. THE Notification_System SHALL 从 Notification_Body 文本中通过正则匹配提取"评分 {数字}"模式来识别评分值并应用颜色标注

### Requirement 5: 通知分类名称修改

**User Story:** 作为业务用户，我希望通知分类中的"智能审核"改为"审核工作台"，以便与系统菜单名称保持一致。

#### Acceptance Criteria

1. THE Locale_Files SHALL 将 `notifications.category.audit` 的中文翻译从"智能审核"修改为"审核工作台"
2. THE Locale_Files SHALL 将 `notifications.category.audit` 的英文翻译从"Smart audit"修改为"Audit Workbench"
3. WHEN AppHeader 渲染通知分类标签时，THE Category_Label SHALL 显示更新后的"审核工作台"（中文）或"Audit Workbench"（英文）

### Requirement 6: 独立消息页面

**User Story:** 作为业务用户，我希望点击通知铃铛后能打开一个独立的消息页面，采用类似微信的左右分栏布局，左侧展示消息列表，右侧展示选中消息的详细内容，以便我能更方便地浏览和管理消息。

#### Acceptance Criteria

1. WHEN 用户点击 AppHeader 中的通知铃铛图标时，THE Notification_System SHALL 导航到独立的 Message_Page（路由 `/messages`）
2. THE Message_Page SHALL 采用左右分栏布局：左侧占约 30% 宽度展示消息列表，右侧占约 70% 宽度展示选中消息的详细内容
3. WHEN 用户在左侧消息列表中点击某条消息时，THE Message_Page SHALL 在右侧区域展示该消息的完整标题、分类、时间和正文内容
4. WHEN 用户点击某条未读消息时，THE Message_Page SHALL 自动将该消息标记为已读
5. THE Message_Page 左侧列表 SHALL 展示每条消息的分类标签、标题、正文摘要和相对时间
6. THE Message_Page SHALL 对未读消息在左侧列表中以视觉区分方式（如加粗或未读标记点）标识
7. WHILE 未选中任何消息时，THE Message_Page 右侧区域 SHALL 展示空状态提示文案
8. THE Message_Page SHALL 复用 useNotifications composable 中的现有 API 调用逻辑
9. THE Message_Page SHALL 在右侧详情区域对评分文本应用与 Requirement 4 相同的颜色标注规则

### Requirement 7: 一键已读功能

**User Story:** 作为业务用户，我希望在消息页面中能一键将当前权限下的所有未读消息标记为已读，以便我能快速清理未读消息。

#### Acceptance Criteria

1. THE Message_Page SHALL 在消息列表区域顶部提供"全部已读"操作按钮
2. WHEN 用户点击"全部已读"按钮时，THE Notification_System SHALL 调用 `markAllRead` API 将当前角色下所有未读通知标记为已读
3. WHEN 全部已读操作成功后，THE Message_Page SHALL 立即更新左侧列表中所有消息的已读状态，并将 AppHeader 中的未读角标清零
4. WHILE 当前无未读消息时，THE Message_Page SHALL 隐藏或禁用"全部已读"按钮

### Requirement 8: 国际化支持

**User Story:** 作为使用英文界面的用户，我希望所有新增的 UI 文案都有对应的英文翻译，以便我能在英文环境下正常使用消息功能。

#### Acceptance Criteria

1. THE Locale_Files SHALL 为 Message_Page 中所有新增 UI 文案（页面标题、空状态提示、全部已读按钮等）同时提供中文和英文翻译
2. THE Locale_Files SHALL 为评分颜色标注相关的无障碍文案提供中英文翻译
3. THE Message_Page SHALL 通过 `useI18n` composable 的 `t()` 函数获取所有 UI 文案，确保语言切换时文案同步更新
4. THE Locale_Files SHALL 遵循现有的 i18n 键命名规范（使用点分隔的层级命名，如 `messages.title`、`messages.empty`）

### Requirement 9: 代码风格一致性

**User Story:** 作为开发团队成员，我希望所有新增代码的日志输出和注释风格与现有代码保持一致，以便维护代码库的统一性。

#### Acceptance Criteria

1. THE Notification_System 中所有新增的 Go 代码 SHALL 使用中文注释，格式与现有代码一致（如 `// 审核完成通知`）
2. THE Notification_System 中所有新增的 TypeScript/Vue 代码 SHALL 使用中文注释，格式与现有代码一致（如 `/** 通知列表（响应式，供模板直接绑定） */`）
3. THE Notification_System 中所有新增的 Go 代码 SHALL 使用 `fmt.Sprintf` 格式化日志消息，与现有服务代码风格一致
4. THE Notification_System 中所有新增的前端 composable SHALL 遵循现有 composable 的导出模式（返回响应式引用和异步函数）
