# 实施计划：仪表盘全面优化

## 概述

基于需求文档和设计文档，将仪表盘优化拆分为增量式编码任务。每个任务构建在前一个任务之上，最终完成所有 6 个需求的实现。技术栈：Nuxt 3 + Vue 3 + TypeScript + vue-echarts + Ant Design Vue。

## 任务

- [x] 1. 新增类型定义和主题色工具
  - [x] 1.1 在 `frontend/types/dashboard-overview.ts` 中新增 `SystemMonitorData` 和 `ServiceStatus` 接口定义
    - 新增 `SystemMonitorData` 接口：`cpu_usage`、`memory_usage`、`disk_usage`、`services`、`uptime_seconds`
    - 新增 `ServiceStatus` 接口：`name`、`status`（'online' | 'offline' | 'degraded'）、`response_time_ms`
    - 在 `PlatformDashboardOverview` 接口中添加可选字段 `system_monitor?: SystemMonitorData`
    - _需求: 4.1, 4.2, 4.3_

  - [x] 1.2 创建 `frontend/composables/useThemeColors.ts` 主题色读取工具
    - 实现 `getCssVar(name)` 函数从 CSS 自定义属性读取当前计算值
    - 实现响应式 `chartColors` 计算属性，包含 primary、accent、success、warning、danger 等色值
    - 使用 `useTheme()` 的 `isDark` 实现主题感知
    - 提供 fallback 默认色值，确保 SSR 环境下不报错
    - _需求: 1.5, 6.4_

  - [x] 1.3 新增监控阈值常量
    - 在 `frontend/composables/useThemeColors.ts` 或独立文件中定义 `MONITOR_THRESHOLDS` 常量
    - 导出 `getMetricColor(value, thresholds)` 纯函数和 `getServiceStatusColor(status)` 纯函数
    - _需求: 3.4, 3.5_

  - [ ]* 1.4 为 `getMetricColor` 编写属性测试
    - **Property 4: 阈值颜色分类**
    - **验证: 需求 3.4**

  - [ ]* 1.5 为 `getServiceStatusColor` 编写属性测试
    - **Property 5: 服务状态指示器映射**
    - **验证: 需求 3.5**

- [x] 2. 检查点 - 确保所有测试通过
  - 确保所有测试通过，如有疑问请询问用户。

- [x] 3. ECharts 图表视觉增强
  - [x] 3.1 增强 `frontend/components/charts/StackedBarChart.vue` 趋势图
    - 引入 `useThemeColors()` 获取当前主题色
    - 使用 ECharts `LinearGradient` 为每个系列创建渐变色 `itemStyle`
    - 配置 `itemStyle.borderRadius: [4, 4, 0, 0]` 实现圆角柱体
    - 配置 tooltip 使用圆角、阴影和主题适配的背景色
    - 坐标轴颜色、文字颜色从 `chartColors` 读取
    - 保留 `autoresize` 属性
    - 添加中文注释说明关键逻辑
    - _需求: 1.1, 1.3, 1.4, 1.5, 1.6, 6.1_

  - [x] 3.2 增强 `frontend/components/charts/DeptDistributionChart.vue` 部门分布图
    - 引入 `useThemeColors()` 获取当前主题色
    - 横向柱状图使用 `LinearGradient` 渐变色
    - 配置 `itemStyle.borderRadius: [0, 4, 4, 0]`（横向柱体圆角方向）
    - tooltip、坐标轴配色与趋势图一致
    - 添加中文注释
    - _需求: 1.2, 1.3, 1.4, 1.5, 1.6, 6.1_

  - [x] 3.3 提取图表选项构建为纯函数以支持测试
    - 从 `StackedBarChart.vue` 提取 `buildStackedBarOption(data, themeColors)` 纯函数
    - 从 `DeptDistributionChart.vue` 提取 `buildDeptChartOption(data, themeColors)` 纯函数
    - 放置在 `frontend/utils/chart-builders.ts` 中
    - _需求: 1.1, 1.2_

  - [ ]* 3.4 为图表配置构建函数编写属性测试
    - **Property 1: 图表配置视觉一致性**
    - **验证: 需求 1.1, 1.2**

  - [ ]* 3.5 为主题感知配色编写属性测试
    - **Property 2: 主题感知配色适配**
    - **验证: 需求 1.4, 1.5**

- [x] 4. 概览页小部件布局自适应优化
  - [x] 4.1 修改 `frontend/pages/overview.vue` 中的 Widget Grid CSS
    - 将 `.widget-grid` 改为 `grid-auto-rows: min-content` + `align-items: start`
    - 为 `.widget--sm`、`.widget--md`、`.widget--lg` 设置合理的 `min-height`
    - 添加 `@media (max-width: 768px)` 响应式断点，切换为单列布局
    - 确保 gap 间距统一为 20px
    - 确保拖拽过程中布局稳定
    - _需求: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6_

- [x] 5. 检查点 - 确保所有测试通过
  - 确保所有测试通过，如有疑问请询问用户。

- [x] 6. 系统管理员系统运行监控
  - [x] 6.1 在 `frontend/composables/useDashboardOverviewApi.ts` 中新增 `fetchSystemMonitorData()` 方法
    - 调用 `GET /api/admin/system-monitor` 接口
    - 添加 JSDoc 注释，说明方法用途、返回类型和对接的后端路由
    - _需求: 3.7, 6.3_

  - [x] 6.2 在 `frontend/constants/overviewWidgets.ts` 中注册 `system_monitor` 小部件
    - 扩展 `OverviewWidgetId` 联合类型新增 `'system_monitor'`
    - 在 `OVERVIEW_WIDGETS` 数组中添加 `system_monitor` 定义：`requiredPermissions: ['system_admin']`、`defaultEnabled: true`、`size: 'lg'`
    - 配置 `titleKey` 和 `descriptionKey` 指向 i18n 键
    - _需求: 3.1, 5.3_

  - [x] 6.3 创建 `frontend/components/charts/GaugeChart.vue` 仪表盘图表组件
    - 接收 `value`（0-100）、`label`、`thresholds`、`height` props
    - 使用 ECharts gauge 类型渲染
    - 根据 value 与 thresholds 动态设置指针和进度条颜色
    - 通过 `useThemeColors()` 适配主题
    - 提取 `buildGaugeOption` 为纯函数放入 `frontend/utils/chart-builders.ts`
    - 添加中文注释
    - _需求: 3.3, 6.1_

  - [ ]* 6.4 为 Gauge 图表数据映射编写属性测试
    - **Property 3: Gauge 图表数据映射**
    - **验证: 需求 3.3**

  - [x] 6.5 创建 `frontend/components/SystemMonitorWidget.vue` 系统监控小部件
    - 使用 `fetchSystemMonitorData()` 获取数据
    - 顶部：刷新按钮 + 系统运行时间显示
    - 中部：CPU / 内存 GaugeChart 并排 + 磁盘使用率进度条
    - 底部：关键服务状态列表（API 服务、数据库、Redis、AI 模型服务）
    - 使用 `getMetricColor` 和 `getServiceStatusColor` 进行颜色判断
    - 所有文案通过 `t()` 函数获取，不硬编码字符串
    - 使用 CSS 自定义属性定义样式，`<style scoped>`
    - 在关键数据获取和错误处添加 `console.warn` / `console.error` 日志
    - 请求失败时显示错误提示和重试按钮
    - _需求: 3.2, 3.3, 3.4, 3.5, 3.6, 3.8, 5.2, 6.2, 6.4, 6.5_

  - [x] 6.6 在 `frontend/pages/overview.vue` 中集成 SystemMonitorWidget
    - 导入 SystemMonitorWidget 组件
    - 在 widget-grid 中添加 `system_monitor` 小部件的渲染逻辑
    - 确保仅系统管理员角色可见
    - _需求: 3.1, 3.2_

- [x] 7. 国际化支持
  - [x] 7.1 在 `frontend/locales/zh-CN.ts` 中新增系统监控相关翻译键值对
    - 添加 `overview.widgetTitle.system_monitor`、`overview.widgetDesc.system_monitor` 等所有监控相关键
    - _需求: 5.1_

  - [x] 7.2 在 `frontend/locales/en-US.ts` 中新增系统监控相关翻译键值对
    - 添加与 zh-CN 对应的所有英文翻译
    - _需求: 5.1_

  - [ ]* 7.3 为 I18n 键完整性编写属性测试
    - **Property 6: I18n 键完整性**
    - **验证: 需求 5.1, 5.3**

- [x] 8. 最终检查点 - 确保所有测试通过
  - 确保所有测试通过，如有疑问请询问用户。

## 备注

- 标记 `*` 的任务为可选任务，可跳过以加速 MVP 交付
- 每个任务引用了具体的需求编号以确保可追溯性
- 检查点确保增量验证
- 属性测试验证设计文档中定义的 6 个正确性属性
- 单元测试验证具体示例和边界情况
