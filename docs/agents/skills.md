# Skills

Skills 是给智能体的**可加载指令包**，格式对齐常见 `SKILL.md`（YAML frontmatter + Markdown 正文），可选 `scripts/`。用于沉淀 OA 话术、查询口径、辅助办步骤，而不把所有说明写进智能体系统提示。

## 1. 包结构

```text
skill-code/
  SKILL.md          # 必填
  reference.md      # 可选，按需加载
  scripts/          # 可选，声明后可升为工具
```

`SKILL.md` 文首：

```yaml
---
name: oa-todo-triage
description: 在用户询问待办优先级时，按节点与提交时间整理待办并建议处理顺序
---
```

`description` 用于：管理端展示、以及运行时是否把该 Skill 纳入提示（模型只先看到短描述，正文按匹配加载，避免提示词过长）。

## 2. 两级分配

| 层级 | 能力 |
|------|------|
| 系统管理员 | 维护内置 Skill 目录；按租户勾选可用 `skill_code`；是否允许租户上传自定义 Skill |
| 租户管理员 | 启用配额内 Skill；在允许时新增租户 Skill；把 Skill 绑到智能体 |
| 业务用户 | 无独立 Skill 授权；随智能体生效 |

自定义 Skill 视为租户数据：随 `chat_retention` 或单独保留策略；禁止执行未声明的任意脚本路径。

## 3. 加载时机

1. 会话使用某智能体时，列出其绑定 Skills（∩ 租户配额）。
2. 将各 Skill 的 `name` + `description` 放入系统提示的「可用技能」段。
3. 用户问题与某 Skill 描述匹配（或模型请求加载）时，再注入该 `SKILL.md` 正文。
4. `reference.md` 默认不注入，除非正文要求或实现做显式加载工具。

## 4. 脚本升为工具

若 `SKILL.md` 声明脚本入口（名称、参数 schema、说明）：

- 生成 `tool_code`：`skill:{skill_code}:{script_name}`
- 进入工具目录后，**仍需**出现在系统管理员配额与角色再分配中才能调用（可随 Skill 配额批量授予，实现时在管理端提供「授予该 Skill 下全部脚本工具」）。
- `ui_kind`：`skill_script`
- 执行沙箱：无网络或仅允许列明出口；超时与输出大小限制。一期若来不及做安全沙箱，则 **只加载文档、不执行脚本**，并在管理端标明。

## 5. 与系统工具的关系

Skill 教模型**何时调用** `list_my_todos` 等系统工具，不替代适配器。不要在 Skill 正文中写死某客户的数据库表名或账号。
