# Kanban 独立编排 Demo 设计

> 面向：验证任务状态、Agent 调度和多轮交互语义
> 
> 范围：独立模块，底层首个 Harness 接入 DSH
> 
> 状态：Demo 设计

## 1. 目标

这个 Demo 用来验证一套比当前七列 Kanban 更清晰的任务编排模型。它不依赖 VS Code、云端服务或当前 Kanban UI，能够单独启动、创建任务、调度 DSH、接收事件并展示完整时间线。

Demo 重点回答四个问题：

1. 看板列是否只表达 Task 的业务阶段；
2. Agent 的执行状态是否和 Task 状态分离；
3. Agent 需要信息时，用户能否在同一个 Run 中回答并继续；
4. 测试和代码审查是否可以作为 Review 阶段的检查，而不是固定列。

## 2. 非目标

第一版不实现以下内容：

- 团队账号、云端同步和权限系统；
- GitHub Issue 同步；
- 父子 Task、Stage 屏障和复杂依赖图；
- Worktree 创建、合并和清理；
- 多个 Runtime 的抢占和租约；
- 复刻 DSH 的内部协议。

这些能力以后可以接在模块边界之外，不应影响 Demo 的核心状态机。

## 3. 模块边界

```text
Demo UI / Demo CLI
        |
        v
Kanban Demo Core
  - Task 状态机
  - Run 调度
  - Check 门禁
  - 人工输入与恢复
  - 事件时间线
        |
        v
Harness Adapter
  - DSH Adapter（第一版）
        |
        v
DSH
```

Core 不直接执行子进程，也不依赖 DSH 的命令名、参数或输出格式。所有 Harness 都通过 Adapter 接入。

## 4. 核心语义

### 4.1 Task 是业务目标

Demo 默认使用五个阶段：

```text
Backlog -> To Do -> In Progress -> Review -> Done
                                  ^       |
                                  +-------+
                              Changes Requested
```

- `Backlog`：暂不安排；
- `To Do`：已确认，可以开始；
- `In Progress`：正在形成交付；
- `Review`：交付已经形成，等待检查和人工确认；
- `Done`：团队确认完成。

`Blocked` 不作为列，而是 Task 的附加状态。阻塞解除后回到阻塞前的阶段。

### 4.2 Run 是一次 Agent 工作

Run 不等于 Task，也不等于 DSH 进程。一个 Task 可以有多次 Run；一次 Run 因为等待回答或进程重启可以包含多次 Harness Attempt。

```text
queued -> running -> awaiting_input -> running -> completed
                     |                              |
                     +------------------------------+
                     failed / cancelled
```

Run 的完成只代表 Agent 交付了本次结果，不自动把 Task 标记为 `Done`。

### 4.3 Check 是验证活动

Review 阶段可以有多个 Check，例如：

```text
automated_test
code_review
product_acceptance
```

Check 独立记录 `pending / running / passed / changes_requested / failed`。所有必需 Check 通过后，Task 才具备进入 `Done` 的条件；需要修改时回到 `In Progress`。

## 5. 调度规则

Demo 只保留两种触发入口：

### 5.1 开始任务

用户点击“开始”或执行 Demo CLI 的 start 命令：

```text
Task: todo
  -> Core 创建 Run（queued）
  -> Scheduler 调用 DSH Adapter
  -> DSH 启动后 Run: running
  -> Task: in_progress
```

### 5.2 Review 介入

Agent 提交结构化完成结果后，Core：

1. 结束当前 Run；
2. 保存 Agent 结果到时间线；
3. 创建配置要求的 Check；
4. 将 Task 转为 `review`；
5. 为自动 Check 创建新的 Run。

同一个事件可以创建多个 Check/Run。调度器不能因为一个 Agent 完成就直接把 Task 设为 `done`。

## 6. 多轮人工交互

Agent 不能确定需求时，通过 Adapter 事件提出问题，而不是把问题写进普通日志：

```text
DSH -> request_input(question, options)
Core -> Run: awaiting_input
Core -> Task: blocked = human_input
UI  -> 展示问题和选项
User -> answer
Core -> 保存回答
Core -> 清除 Block
Core -> 恢复同一个 Run
DSH -> 继续执行
```

回答必须进入下一次 DSH 调用的上下文。若 DSH 支持原生 session resume，Adapter 使用原 session；若不支持，Adapter 用保存的任务上下文和历史回答重新启动一次 Attempt，但对 Core 仍表现为同一个 Run。

用户关闭 Demo 或刷新页面不应丢失问题、选项、回答和 Run 状态。

## 7. Harness Adapter 契约

Core 只依赖以下能力，不依赖具体传输方式：

```text
start(input) -> handle
resume(handle, input) -> handle       可选
sendInput(handle, answer)             可选
cancel(handle)
events(handle) -> progress / output / request_input / completed / failed
```

### 7.1 DSH Adapter 职责

DSH Adapter 负责：

- 将 Core 生成的任务上下文、角色指令和工作目录传给 DSH；
- 启动并监控 DSH；
- 将 DSH 的文本、工具调用、问题、完成和错误转换成 Core 事件；
- 保存 DSH session 标识，支持恢复或重启；
- 在 DSH 退出但没有完成事件时，上报 `failed`，不能猜测为成功。

DSH 的实际启动命令、参数和事件格式属于 Adapter 内部配置。集成第一步是对 DSH 做一个 capability probe，确认它是否支持：流式事件、原生输入、session resume、取消和结构化完成信号。

### 7.2 交互能力降级

如果 DSH 只有一次性 stdout，没有输入通道：

- Demo 仍可以展示问题；
- 回答后必须创建新的 Attempt；
- 新 Attempt 需要重新注入完整上下文；
- UI 明确显示“已重新启动”，不能伪装成真正的原生续聊。

这只是兼容降级路径，验收重点仍是 DSH 原生交互是否可用。

## 8. 上下文组成

每次启动或恢复时，Core 组装四类上下文：

1. 平台协议：当前 Task、Run、Attempt、允许的操作和完成规则；
2. Agent 角色：职责、输出要求和项目约束；
3. Task 内容：标题、描述、验收标准、当前阶段；
4. 会话增量：历史评论、已回答问题、前一次 Attempt 结果。

上下文由 Core 生成，Adapter 只负责传输。Agent 不能通过修改提示词自行改变 Task 阶段或其他 Task 的数据。

## 9. “制作小猫网站”验收流程

Demo 使用一个 Task，不拆父子任务，以便先验证单 Task 语义：

```text
To Do
  -> 用户开始
In Progress
  -> DSH Agent 询问：小猫网站需要哪些页面、风格和素材？
  -> Run awaiting_input，Task 保持 In Progress + human_input
  -> 用户选择/回答
  -> 同一 Run 恢复，Agent 产出网站实现
Review
  -> automated_test Check + code_review Check 并行运行
  -> 全部通过，等待人工确认
Done
```

如果测试失败：

```text
Review -> Changes Requested -> In Progress
       -> 创建修复 Run
       -> 修复完成后重新进入 Review
```

这个流程中没有 `Testing` 列，但测试仍然是可追踪的 Check；没有把 Agent 的 `running/completed` 显示成 Task 列。

## 10. 事件和持久化要求

Demo 使用本地 SQLite 或 JSON 持久化当前状态，并把关键变化建模为事件；内存实现只用于单元测试：

```text
task_created
task_started
run_queued
run_started
input_requested
input_answered
run_completed
check_created
check_passed
changes_requested
task_completed
```

事件用于驱动 UI 时间线和恢复，不要求第一版实现完整事件溯源。持久化层至少要能在进程重启后恢复 Task、Run、问题和回答；DSH transcript 可以只保存引用或文件路径。

## 11. Demo 组成和顺序

### 阶段一：Core 状态机

- 创建 Task、手动开始和状态转换；
- 创建 Run、完成 Run、创建 Review Check；
- 测试失败回到 `In Progress`。

### 阶段二：模拟 Harness

- 用 Fake Harness 产生进度、问题、完成和失败事件；
- 先验证多轮交互，不受 DSH 安装和协议不确定性影响。

### 阶段三：DSH Adapter

- 完成 capability probe；
- 接入真实启动、流式事件、输入和取消；
- 验证 session resume，确认断线后的行为。

### 阶段四：最小 UI

- 看板五列；
- Task 时间线；
- 当前 Run 状态；
- Check 结果；
- 问题回答和恢复按钮。

## 12. 验收标准

Demo 完成必须满足：

1. Task 状态不会因为 DSH Run 的 `completed` 自动跳过 Review；
2. Agent 提问时，Run 进入 `awaiting_input`，页面刷新后问题仍存在；
3. 用户回答后能恢复同一逻辑 Run，并看到回答进入上下文；
4. 自动测试和代码审查可以并行存在为两个 Check；
5. Check 失败能明确回到 `In Progress`，而不是停在含义不清的 Testing 列；
6. DSH 进程异常退出会产生可见的失败事件；
7. 替换为 Fake Harness 时，Core 和 UI 无需修改。

## 13. 后续扩展

在 Demo 证明单 Task 模型后，再增加：

- 父子 Task 和 Stage 屏障；
- 多 Agent 并行调度；
- 远程服务端和团队协作；
- Runtime、Worktree 和租约；
- 更多 Harness Adapter。

这些扩展都应复用本设计中的 Task、Run、Check、Block 和 Adapter 边界，不再把新的 Agent 活动直接增加为看板列。
