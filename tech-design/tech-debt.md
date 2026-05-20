# 技术债记录

> 记录已知的性能优化和架构改进项，按优先级排序，等待后续迭代偿还。

---

## Phase 2 — Mailbox 框架增强

**问题**: 当前所有 actor 统一使用 256 大小的邮箱和 Blocking 背压策略。高并发场景下，读 actor（如 msg-reader）阻塞即可丢弃旧请求（幂等读），而写 actor（如 conv、call-manager）不应丢弃消息。

**目标改动**:

1. **`actor/option.go`** — `MailboxFactory` 签名改为 `func(name string) Mailbox`，支持按 actor 名定制邮箱参数
2. **`actor/engine.go`** — `newMailbox(name string)` 传递 actor 名给工厂
3. **`cmd/server/app.go`** — 配置不同 actor 的 mailbox size/policy：
   - `msg-reader-N`: size=2048, policy=DropOldest（读请求幂等，丢弃旧的接受新的）
   - `conv:<id>`: size=512, policy=Blocking（写请求不能丢）
   - `call-manager` 等管理类: size=512, policy=Blocking

**收益**: 读路径在峰值负载下不会阻塞上游调用方，自动丢弃过期请求。

---

## Phase 3 — friend-manager / user-manager 读写分离

**问题**: `friend-manager` 和 `user-manager` 是单 actor，所有操作（读写）串行化。虽然这些域操作频率远低于消息，但在好友列表加载、用户搜索等场景仍可能成为瓶颈。

**目标改动**:

1. **friend-manager** 拆分为 N 个 shard，按 `requester_uid % N` 路由
2. **user-manager** 拆分为 N 个 shard，按 `uid % N` 路由
3. 或者只拆读路径（类似 msg-store 的处理），写路径保持单 actor 保证一致性

**收益**: 好友/用户读操作并发度提升 N 倍。

---

## Phase 4 — 异步 DB 回调（可选，需框架改造）

**问题**: `ctx.Reply()` 不是 goroutine-safe 的（`c.replied` 无锁，`processLoop` 在 dispatch 后检查 `replied` 并自动回复 `ErrNoReply`）。这导致 DB 操作必须在 actor 的 goroutine 内同步执行，无法利用协程池并发。

**目标改动**:

1. `actorContext.Reply()` 加 `sync.Mutex` 保护 `replied` 和 `current` 字段
2. 增加 `ReplyAsync(msg)` 方法，允许从其他 goroutine 安全回复
3. actor 内部可使用 `go func() { db.Query(); ctx.ReplyAsync(result) }()` 模式

**注意**: 此改动影响框架核心，需要充分的并发测试和 Review。

---

## 改动记录

| 日期 | 内容 |
|------|------|
| 2026-05-21 | Phase 1 完成：msg-store 拆分为 4 个 msg-reader-N，按 convID % 4 路由。记录 Phase 2-4 为技术债。 |
