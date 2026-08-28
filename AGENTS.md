# Kanban Demo 实现约束

## 文档定位

本文件是实现阶段的工作约束，配合以下两份基础设计文档阅读：

- `2026-08-28-kanban-dsh-demo-design.md`：产品语义、验收流程和 Demo 范围。
- `2026-08-28-kanban-dsh-demo-architecture.md`：分层架构、状态机和适配器契约。

如果实现细节与这两份文档冲突，以领域状态机、事件语义和验收标准为准；不得为了简化 UI 而改变领域含义。

## 产品范围

- 这是单机、单项目的独立 Kanban 编排 Demo，不依赖 VS Code、云端服务或现有 Kanban UI。
- 第一版以“小猫网站”单 Task 闭环为主要验收场景。
- 看板固定使用五个 Task 阶段：`backlog`、`todo`、`in_progress`、`review`、`done`。
- 不增加 `testing` 顶层列。测试和代码审查建模为 Review 阶段的 `Check`。
- `blocked` 是 Task 的附加状态，不是看板列；解除阻塞后回到阻塞前阶段。
- 第一版暂不实现账户、云端同步、GitHub Issue、父子 Task、Stage 屏障、Worktree、多 Runtime 抢占/租约和团队协作。

## 技术基线

- 前端使用 TypeScript + React + Vite，开启 TypeScript `strict`。
- 服务端使用 Go + Gin；使用 Go 原生 `net/http` 能力处理请求上下文和 SSE，不引入第二套服务端运行时。
- 保持单仓库，但划分 Go 服务端和 `web/` 前端两个构建单元；不引入 monorepo 工作区管理。
- 演示/生产模式由一个 Go 进程托管 API 和构建后的静态 UI；开发模式允许 Vite dev server 独立运行并代理到 Go 服务。
- 用户命令使用 REST；服务端事件使用 SSE。SSE 事件必须包含稳定的事件序号，并支持通过 `Last-Event-ID` 恢复；刷新页面不能丢失状态。
- 运行时使用本地 SQLite，通过 `database/sql` 和 `modernc.org/sqlite` 访问；Go 单元测试使用内存 Store。数据库文件放在 `.data/`，不得提交到 Git。
- 所有跨层通信使用显式接口和结构化类型，不传递 DSH 私有对象。

### 工程初始化记录

- Go module：`github.com/eternal-yellowduck/kanbandemo`。
- Go 默认端口：`8080`，可通过 `PORT` 环境变量覆盖。
- Vite 默认端口：`5173`，开发代理将 `/api` 和 `/events` 指向 Go 服务。
- Go 入口：`cmd/kanban-demo/main.go`；前端工程目录：`web/`。
- 当前初始化阶段只提供 `GET /api/health` 和 `GET /events` 占位，不提前冻结完整 Task API。
- 存在 `web/dist` 时，Go 服务托管前端构建产物；本地开发仍使用 Vite dev server。
- 根 `.gitignore` 必须忽略 `.data/`、`web/node_modules/`、`web/dist/`、Go 构建产物和测试报告。

推荐的目录边界：

```text
cmd/kanban-demo/main.go
internal/
  domain/
  application/
  infrastructure/
    memory/
    sqlite/
    harness/
  presentation/http/
web/
  src/
tests/e2e/
```

## 分层边界

依赖方向只能是 `Presentation -> Application -> Domain -> Infrastructure`。

### Presentation

- 只展示 Task、Run、Attempt、Check、Block 和事件时间线。
- 只提交领域命令：开始、取消、重试、回答问题、请求修改、人工验收。
- 不直接调用 DSH，不直接写 Store，不自行推断或修改状态。

### Application

- 负责事务边界、幂等和调度协调。
- 至少提供 `StartTask`、`AnswerInput`、`CancelRun`、`RetryRun`、`AcceptTask` 和 `HandleHarnessEvent`。
- 负责调用 Context Builder、Scheduler、Review Coordinator 和 Input Coordinator，但不拼接 DSH 命令行参数。

### Domain

- 是唯一允许定义业务状态转换的层。
- Domain 不执行 IO，不依赖 UI、文件系统、SQLite、DSH 或具体传输协议。
- 每次状态写入必须包含来源、原因和幂等键。

### Infrastructure

- 实现 SQLite/内存 Store、事件总线、Context Builder、进程管理和 Harness Adapter。
- DSH 的命令、参数、环境变量、输出字段和 session 细节只能出现在 DSH Adapter 内。

## 核心状态语义

### Task

```text
backlog -> todo -> in_progress -> review -> done
review -> in_progress  (changes_requested)
```

- Agent 不能直接把 Task 设为 `done`。
- Run 完成只代表本次 Agent 交付完成；必须先进入 `review`，创建必需 Check，并在所有必需 Check 通过后等待人工验收。
- Task 状态转换必须带原因，例如 `user_start`、`agent_delivery`、`review_changes_requested`。

### Run 和 Attempt

```text
queued -> dispatched -> running -> completed
                         |       |
                         |       +-> failed / cancelled
                         v
                   awaiting_input -> queued
```

- `Run` 是一次逻辑 Agent 工作，`Attempt` 是一次实际 Harness 执行。
- 一个 Task 最多只能有一个活动中的交付 Run，避免多个 Agent 同时修改同一交付。
- Review 阶段的多个 Check 可以各自创建 Run 并行执行；这不违反交付 Run 的互斥限制。
- 用户回答后优先恢复同一个逻辑 Run；只有 Adapter 不支持原生输入/恢复时，才创建同一 Run 的新 Attempt，并在 UI 明确显示“已重新启动”。
- DSH 进程异常退出且没有结构化完成事件时，必须产生 `failed`，不能猜测为成功。

### Check

```text
pending -> running -> passed
                  -> changes_requested
                  -> failed / unable_to_run / cancelled
```

- 第一版至少创建 `automated_test` 和 `code_review` 两类 Check。
- `failed` 表示检查器或环境未成功完成；`changes_requested` 表示检查完成但交付需要修改，两者必须区分。
- Check 历史不能覆盖；重试必须创建新的 Attempt 或新的 Check revision。

### Block

- Agent 请求输入时创建结构化 `Block(kind=human_input)`，Run 进入 `awaiting_input`，Task 保持原阶段。
- 问题、选项、回答和恢复信息必须持久化，刷新或进程重启后仍可继续。

## Harness 规则

- 默认使用 Fake Harness 验证完整闭环；Fake Harness 与 DSH Adapter 必须通过同一套 Harness contract tests。
- DSH 通过配置/环境变量启用，并在启动时执行 capability probe，至少报告 `stream_events`、`request_input`、`send_input`、`session_resume`、`cancel`、`structured_completion` 和 `usage` 能力。
- Adapter 对 Application 层只发送统一事件：`started`、`progress`、`message`、`tool_activity`、`request_input`、`usage`、`completed`、`failed`、`cancelled`。
- 不支持原生交互时只能显式降级为新 Attempt，不能伪装成 session resume。
- Adapter 必须处理版本检查、启动失败、取消、异常退出、重复事件和原始 session/transcript 引用。
- 每个 Harness 事件至少包含 `runId`、`attemptId`、事件序号和幂等信息；重复事件必须被忽略或安全重放。

## Context Builder

每个 Attempt 启动前由 Core 生成上下文快照，顺序固定为：

1. 平台协议、允许操作和完成规则。
2. Agent 角色职责和输出要求。
3. Task 标题、描述和验收标准。
4. 当前阶段、Block 和相关 Check。
5. 历史评论、问题回答和前次 Attempt 结果。
6. 工作目录和可用工具说明。

保存 context revision，确保可以诊断 Agent 当时看到的内容；不得注入 access token 或无关本机信息，也不能让 Agent 通过文本修改授权范围。

## UI/UX 约束

- 默认浅色、扁平、信息密度适中的工作台风格；不使用渐变、重阴影或装饰性背景图形。
- 使用中性浅灰画布；深蓝只用于导航和主要操作，状态使用蓝、绿、琥珀、红等语义色，避免单一深蓝配色。
- 正文使用 `Fira Sans`，日志、ID、事件和代码使用 `Fira Code`。
- 使用 Lucide 或同类 SVG 图标，不使用 emoji 作为图标；图标按钮必须有可见焦点和 `aria-label`。
- 采用 8px 间距基线，卡片圆角最多 8px，触控目标至少 `44px`。
- 桌面展示五列；窄屏使用阶段 Tab + 单列任务列表，不出现横向滚动。Task 详情桌面使用右侧抽屉，移动端使用全屏面板。
- 所有操作提供 150–300ms 的状态反馈，支持键盘导航、可见 focus、足够的颜色对比度和 `prefers-reduced-motion`。
- 不提供任意拖拽改列或直接编辑状态的调试按钮。开发环境可以提供仅用于 Fake Harness 的事件模拟入口，但不能绕过领域命令。
- 暂不实现暗色模式，除非后续需求明确提出并补充对比度验收。

## 测试要求

- 先写 Domain 状态机测试，再实现 Application 和 Adapter。
- Go 原生 `testing` 覆盖 Task/Run/Check/Block 状态转换、重复事件幂等、Review 门禁、输入恢复和错误分类，运行 `go test ./...`。
- Adapter contract tests 同时运行 Fake Harness 和 DSH Adapter；替换 Harness 时不得修改 Domain/Application 测试。
- Playwright 只覆盖核心“小猫网站”闭环：开始、提问、刷新保留问题、回答恢复、结构化完成、Review Check 并行、失败回到 `in_progress`、修复后重新 Review、人工验收进入 `done`。第一版不为前端引入 Vitest，除非出现值得隔离测试的复杂前端逻辑。

## 完成定义

实现只有在以下路径可演示时才算完成：

```text
创建“小猫网站” Task
  -> 开始 Fake/DSH Run
  -> Agent 结构化提问
  -> 刷新后问题仍存在
  -> 用户回答，原 Run 恢复或明确创建新 Attempt
  -> Agent 提交结构化完成
  -> Task 进入 Review
  -> automated_test 和 code_review Check 并行
  -> 检查失败明确回到 In Progress
  -> 修复后重新 Review
  -> 所有 Check 通过
  -> 人工确认后进入 Done
```
