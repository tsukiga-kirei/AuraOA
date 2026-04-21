# 需求文档：仪表盘全面优化

## 简介

对 OA 智审平台的仪表盘（概览页 `overview.vue` 和审核工作台 `dashboard.vue`）进行全面优化，涵盖四个方面：使用 ECharts 增强图表视觉效果、优化全局布局使各组件拥有自适应高度比、在系统管理员界面新增系统运行监控模块、以及确保所有新增代码遵循国际化（i18n）、日志输出和注释规范。

## 术语表

- **Overview_Page**: 概览仪表盘页面（`frontend/pages/overview.vue`），展示各角色的统计概览小部件
- **Dashboard_Page**: 审核工作台页面（`frontend/pages/dashboard.vue`），展示待审核流程列表和审核结果
- **Widget_Grid**: 概览页中基于 CSS Grid 的小部件网格布局系统
- **ECharts_Component**: 基于 `vue-echarts` 封装的图表 Vue 组件（位于 `frontend/components/charts/`）
- **System_Admin_Overview**: 系统管理员角色在概览页中看到的仪表盘区域
- **System_Monitor_Widget**: 新增的系统运行监控小部件，展示后端服务和基础设施的运行状态
- **Widget_Registry**: 小部件注册表（`frontend/constants/overviewWidgets.ts`），定义所有可用小部件的元数据
- **I18n_Module**: 国际化模块（`frontend/composables/useI18n.ts`），提供 `t()` 翻译函数
- **Theme_System**: 主题系统，通过 CSS 自定义属性支持亮色/暗色主题切换
- **Dashboard_Overview_API**: 仪表盘概览数据 API（`frontend/composables/useDashboardOverviewApi.ts`）

## 需求

### 需求 1：ECharts 图表视觉增强

**用户故事：** 作为平台用户，我希望仪表盘中的图表更加美观和专业，以便更直观地理解数据趋势和分布。

#### 验收标准

1. WHEN Overview_Page 加载趋势图表数据时，THE ECharts_Component SHALL 使用渐变色填充柱状图，并配置圆角柱体、柔和阴影和平滑的动画过渡效果
2. WHEN Overview_Page 加载部门分布图表数据时，THE ECharts_Component SHALL 使用渐变色横向柱状图，并配置与趋势图一致的视觉风格
3. WHEN 用户将鼠标悬停在图表数据点上时，THE ECharts_Component SHALL 显示带有圆角、阴影和自定义配色的 tooltip 浮层
4. WHILE Theme_System 处于暗色主题模式时，THE ECharts_Component SHALL 自动适配暗色主题的配色方案，包括坐标轴颜色、文字颜色和背景色
5. THE ECharts_Component SHALL 在图表配置中使用 CSS 自定义属性（通过 JavaScript 读取 `getComputedStyle`）获取当前主题色值，确保与 Theme_System 保持一致
6. WHEN 浏览器窗口大小发生变化时，THE ECharts_Component SHALL 通过 `autoresize` 属性自动调整图表尺寸，无需手动触发

### 需求 2：概览页小部件布局自适应优化

**用户故事：** 作为平台用户，我希望概览页中每个小部件都有适合自身内容的高度，而不是被强制拉伸到统一高度，以获得更美观的视觉体验。

#### 验收标准

1. THE Widget_Grid SHALL 使用 CSS Grid 的 `masonry` 或 `auto` 行高策略，使每个小部件根据自身内容自然撑开高度，而非强制对齐到同一行高
2. WHEN Widget_Grid 中的小部件内容高度不一致时，THE Widget_Grid SHALL 允许同一行内的小部件拥有不同高度，避免出现大面积空白填充
3. THE Widget_Grid SHALL 为不同尺寸类别的小部件（sm/md/lg）设置合理的最小高度（min-height），确保空数据状态下不会过度塌缩
4. WHEN 浏览器视口宽度小于 768px 时，THE Widget_Grid SHALL 将所有小部件切换为单列全宽布局，并保持各小部件的自适应高度
5. THE Widget_Grid SHALL 为小部件之间设置统一的间距（gap），并确保间距在不同屏幕尺寸下保持视觉一致性
6. WHEN 用户在自定义模式下拖拽调整小部件顺序时，THE Widget_Grid SHALL 在拖拽过程中保持布局稳定，不出现闪烁或跳动

### 需求 3：系统管理员系统运行监控

**用户故事：** 作为系统管理员，我希望在概览仪表盘中看到系统运行状态的监控信息，以便及时发现和处理系统异常。

#### 验收标准

1. THE Widget_Registry SHALL 注册一个新的 `system_monitor` 小部件，其 `requiredPermissions` 设置为 `['system_admin']`，默认启用，尺寸为 `lg`
2. WHEN System_Admin_Overview 加载时，THE System_Monitor_Widget SHALL 从后端 API 获取系统运行状态数据，包括 CPU 使用率、内存使用率、磁盘使用率和服务在线状态
3. WHEN 系统运行状态数据加载成功时，THE System_Monitor_Widget SHALL 使用 ECharts 仪表盘（gauge）图表展示 CPU 和内存使用率，使用进度条展示磁盘使用率
4. WHEN 任一监控指标超过预设阈值（CPU > 80%、内存 > 85%、磁盘 > 90%）时，THE System_Monitor_Widget SHALL 将对应指标的颜色变为警告色（`--color-warning`）或危险色（`--color-danger`）
5. THE System_Monitor_Widget SHALL 展示后端关键服务（API 服务、数据库、Redis、AI 模型服务）的在线/离线状态，每个服务用绿色圆点（在线）或红色圆点（离线）标识
6. WHEN 用户点击 System_Monitor_Widget 中的刷新按钮时，THE System_Monitor_Widget SHALL 重新从后端获取最新的系统运行状态数据
7. THE Dashboard_Overview_API SHALL 新增 `fetchSystemMonitorData()` 方法，调用后端 `GET /api/admin/system-monitor` 接口获取系统监控数据
8. IF 系统监控数据接口请求失败，THEN THE System_Monitor_Widget SHALL 显示友好的错误提示信息，并提供重试按钮

### 需求 4：系统监控数据类型定义

**用户故事：** 作为开发者，我希望系统监控数据有清晰的 TypeScript 类型定义，以便在开发过程中获得类型安全和代码提示。

#### 验收标准

1. THE Dashboard_Overview_API SHALL 在 `frontend/types/dashboard-overview.ts` 中定义 `SystemMonitorData` 接口，包含 `cpu_usage`（number）、`memory_usage`（number）、`disk_usage`（number）、`services`（ServiceStatus 数组）和 `uptime_seconds`（number）字段
2. THE Dashboard_Overview_API SHALL 在 `frontend/types/dashboard-overview.ts` 中定义 `ServiceStatus` 接口，包含 `name`（string）、`status`（'online' | 'offline' | 'degraded'）和 `response_time_ms`（number）字段
3. THE Dashboard_Overview_API SHALL 将 `SystemMonitorData` 类型添加到 `PlatformDashboardOverview` 接口中作为可选字段 `system_monitor`

### 需求 5：国际化支持

**用户故事：** 作为多语言环境的用户，我希望所有新增的仪表盘功能都支持中英文切换，以便在不同语言环境下正常使用。

#### 验收标准

1. THE I18n_Module SHALL 在 `frontend/locales/zh-CN.ts` 和 `frontend/locales/en-US.ts` 中新增所有系统监控相关的翻译键值对，包括小部件标题、状态标签、指标名称和错误提示
2. THE System_Monitor_Widget SHALL 通过 `t()` 函数获取所有用户可见的文案，不在模板中硬编码任何中文或英文字符串
3. THE Widget_Registry SHALL 为 `system_monitor` 小部件配置 `titleKey` 和 `descriptionKey`，指向对应的 i18n 翻译键
4. WHEN 用户切换语言时，THE System_Monitor_Widget SHALL 立即更新所有文案为目标语言，无需刷新页面

### 需求 6：代码规范一致性

**用户故事：** 作为开发者，我希望所有新增代码遵循项目现有的编码规范，以便维护代码库的一致性和可读性。

#### 验收标准

1. THE ECharts_Component SHALL 在 `<script setup>` 中使用中文注释说明组件的 props 定义、计算属性和关键逻辑，与现有 `StackedBarChart.vue` 和 `DeptDistributionChart.vue` 的注释风格保持一致
2. THE System_Monitor_Widget SHALL 在关键数据获取和状态变更处添加 `console.warn` 或 `console.error` 日志输出，用于开发调试和生产环境问题排查
3. THE Dashboard_Overview_API SHALL 在新增的 `fetchSystemMonitorData()` 方法上方添加 JSDoc 注释，说明方法用途、返回类型和对接的后端路由，与现有 `fetchDashboardOverview()` 的注释风格一致
4. THE System_Monitor_Widget SHALL 使用项目已有的 CSS 自定义属性（`--color-*`、`--radius-*`、`--shadow-*`）进行样式定义，不引入新的硬编码颜色值或尺寸值
5. THE System_Monitor_Widget SHALL 使用 `<style scoped>` 限定组件样式作用域，避免全局样式污染
