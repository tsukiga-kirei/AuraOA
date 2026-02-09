# 技术栈

## 架构：Go + Python 混合

- **Go** —— 业务中台：用户鉴权、动态菜单（RBAC）、OA 数据轮询、Cron 任务调度、历史记录持久化
- **Python** —— 智能引擎：LLM 交互、RAG 检索（向量库查询）、OCR 解析

## 前端

- Nuxt 3 + TypeScript（SSR 支持）
- Ant Design Vue
- 基于 RBAC 模型的动态路由

## 数据库与存储

- PostgreSQL —— 业务数据 + 向量数据（pgvector）
- MongoDB 或 Elasticsearch —— 审核历史记录、AI 对话原文、推理过程快照

## AI / ML

- LLM 集成（本地或云端模型）
- LangChain 链编排（Checklist Chain、Retrieval Chain、或并行混合）
- RAG 管线 + 向量存储
- 可配置项：模型选型、Prompt 模板、上下文窗口大小

## OA 集成

- JDBC 连接 OA 数据库
- 数据库表结构映射（首发适配泛微 E9）
- 多版本 OA 适配脚本

## 可观测性

- Go ↔ Python 跨服务链路追踪
- API 调用成功率监控
- 模型响应时间追踪

## 部署

- Docker Compose 统一编排所有服务（前端、Go、Python、PostgreSQL、MongoDB）
- 开发/生产环境通过 override 文件区分
- Git 版本控制，.gitignore 排除构建产物、依赖、环境配置

## 实现阶段

- 第一阶段：仅规则库模式（Rules_Only），结构化 Checklist 审核
- 第二阶段：制度库 RAG 模式、混合模式、OCR 解析

## Gemini Skills

仓库中包含 `.gemini/skills/` 目录，内含 10 个 AI 助手技能（PRD 撰写、版本规划、周报整理等）。这些是基于 Prompt 的工作流定义，不是应用代码。每个技能以 `SKILL.md` 为入口，可选包含 `stages/`、`templates/`、`references/` 子目录。
