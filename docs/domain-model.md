# Domain Model

本文件记录 Phase 1 已实现的 Go 领域模型。HTTP、Gin、SQLite 和 Harness 不得进入 `internal/domain`。

## 对象

- `Task`：业务目标，拥有五列生命周期和当前 Block 引用。
- `Run`：一次逻辑 Agent 工作，使用 `RunKind` 区分 `delivery` 与 `check`。
- `Attempt`：一次实际 Harness 执行；同一 Run 可以有多个 Attempt。
- `Check`：Review 阶段的验证活动，保留 revision 和历史结论。
- `Block`：当前阻塞原因，Phase 1 只实现 `human_input`。
- `DomainEvent`：包含来源、原因、时间和幂等键的事实记录。

## 状态规则

### Task

```text
backlog -> todo -> in_progress -> review -> done
review -> in_progress (changes_requested)
```

`done` 需要结构化 `DeliveryResult`、所有必需 Check 通过和人工验收。

### Run

```text
queued -> dispatched -> running -> completed
                         |       |
                         |       +-> failed / cancelled
                         v
                   awaiting_input -> queued
```

一个 Task 最多一个活动交付 Run；Check Run 可以在 Review 阶段并行。

Attempt 初始为 `running`，结束时转换为 `completed`、`failed` 或 `cancelled`，并记录完成时间。

### Check

```text
pending -> running -> passed
                  -> changes_requested
                  -> failed / unable_to_run / cancelled
```

## 变更元数据

所有状态变更接收 `ChangeMeta`，必须提供：

- `EventID`
- `Source`
- `Reason`
- `IdempotencyKey`
- `OccurredAt`

这样可以在应用层实现事务边界、事件审计和重复事件幂等。

## Store

`internal/application.Store` 是应用层端口；`internal/infrastructure/memory.Store` 是 Phase 1 实现。内存 Store 会复制可变字段，并按 Event ID 和幂等键去重事件。SQLite Store 在后续 Phase 3 实现。
