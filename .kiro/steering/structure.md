# 项目结构

```
/
├── OA智审（流程智能审核平台）.md   # 主产品方案/架构文档（中文）
├── .gemini/
│   └── skills/                     # Gemini AI 助手技能定义
│       ├── image-assistant/        # 配图/信息图提示词生成
│       ├── lesson-builder/         # 课程/教程构建
│       ├── prd-doc-writer/         # PRD 需求文档撰写流程
│       ├── priority-judge/         # 优先级判断助手
│       ├── project-map-builder/    # 项目结构梳理
│       ├── req-change-workflow/    # 需求变更管理流程
│       ├── thought-mining/         # 对话洞察提炼
│       ├── version-planner/        # 版本路线规划
│       ├── weekly-report/          # 周报撰写
│       └── writing-assistant/      # 通用写作助手
├── .kiro/
│   ├── hooks/                      # Kiro agent hooks
│   └── steering/                   # Kiro steering 规则（本目录）
```

## Gemini Skill 约定

每个技能遵循统一模式：
- `SKILL.md`（或 `skill.md`）—— 入口文件，包含 frontmatter（`name`、`description`）、触发条件、工作流阶段、核心原则
- `stages/` —— 编号步骤文件（`01-xxx.md`、`02-xxx.md`……），定义各工作流阶段
- `templates/` —— 可复用的输出模板
- `references/` —— 示例文件与参考资料
- `scripts/` —— 可选的自动化脚本（Python、Shell）

## 当前状态

这是一个**实现前的规划仓库**，目前包含：
- 一份产品架构方案文档（根目录 `.md` 文件）
- 面向产品/项目管理的 AI 助手技能定义
- 尚无应用源代码

当应用代码加入后，预期目录结构如下：

```
/
├── frontend/                       # Nuxt 3 前端应用
├── go-service/                     # Go 业务中台
├── ai-service/                     # Python AI 智能引擎
├── db/
│   └── init/                       # 数据库初始化脚本
├── docker-compose.yml              # Docker Compose 编排
├── .gitignore                      # Git 忽略规则
├── .env.example                    # 环境变量模板
```

采用 `tech.md` 中描述的 Nuxt 3 + Go + Python 技术栈，Docker Compose 统一部署。
