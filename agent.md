# QIM - 基于 Actor 模型的即时通讯系统

## 项目概述

QIM 是一个使用 Go 语言实现的即时通讯（IM）后端系统，核心设计基于 **Actor 模型**，自研了一套轻量级 Actor 框架作为并发与消息隔离的基础设施。系统采用分层架构，通过 Actor 之间的消息传递实现业务逻辑的解耦与并发安全。

## 技术栈

| 层面 | 技术 |
|------|------|
| 后端语言 | Go 1.26 |
| 后端 Web 框架 | Gin |
| 后端 WebSocket | gorilla/websocket |
| 后端 ORM | GORM + SQLite |
| 后端序列化 | bytedance/sonic |
| 前端 | React + TypeScript + Vite |
| 前端测试 | Vitest + V8 coverage |
| 前端质量 | ESLint + TypeScript strict |

## 项目结构

```
qim/
├── cmd/
│   ├── server/          # IM 服务端入口
│   └── im-demo/         # Actor 引擎交互式演示
├── internal/
│   ├── actor/           # 自研 Actor 框架（核心基础设施）
│   ├── dal/             # 数据访问层（Data Access Layer）
│   ├── domain/          # 业务领域层
│   │   ├── conversation/  # 会话域
│   │   ├── friend/        # 好友域
│   │   ├── message/       # 消息域
│   │   ├── presence/      # 在线状态域
│   │   └── user/          # 用户域
│   ├── gateway/         # 网关层（HTTP + WebSocket）
│   ├── middleware/       # 中间件（Auth、CORS）
│   ├── pkg/             # 通用工具包
│   │   ├── jwt/           # JWT 认证
│   │   └── resp/          # 统一响应格式
│   └── wal/             # 预写日志（WAL，桩实现）
```

## 核心架构：自研 Actor 框架

### 设计理念

Actor 框架是整个系统的基石，遵循 Actor 模型的核心原则：
- **消息隔离**：每个 Actor 独占一个 goroutine，状态无需加锁
- **消息传递**：Actor 之间仅通过异步消息通信，无共享内存
- **故障隔离**：单个 Actor panic 不影响其他 Actor
- **生命周期管理**：完整的 Spawn → Running → Stopping → Stopped 状态机

### 核心组件

#### Actor 接口
```go
type Actor interface {
    Receive(ctx Context)
}
type Lifecycle interface {
    OnStart(ctx Context)
    OnStop(ctx Context)
}
```
- `Receive`：消息处理入口，通过类型断言分发不同消息
- `Lifecycle`：可选的生命周期钩子

#### Engine（引擎）
- 管理所有 Actor 的注册、查找、创建与销毁
- 支持 `Spawn`、`Lookup`、`GetOrCreate`、`Stop`、`Shutdown` 操作
- 优雅关闭：`ShutdownWithContext` 等待所有 Actor 退出或超时
- 监督策略（Supervision）：支持 Resume / Stop / Restart 三种 panic 处理策略
- 重启节流：`MaxRestarts` + `RestartWindow` 限制重启频率

#### ActorRef（引用）
- `Tell(msg)`：异步发送，fire-and-forget
- `TellFrom(sender, msg)`：带发送者身份的异步发送
- `Ask(msg, timeout)`：同步请求-应答，基于 future 实现，支持超时

#### Mailbox（邮箱）
- 基于 `sync.Cond` 的有界队列实现（`ChanMailbox`）
- 可插拔的背压策略（`MailboxPolicy`）：
  - `Blocking`：邮箱满时阻塞发送方
  - `DropNewest`：丢弃新消息
  - `DropOldest`：淘汰最旧消息
- 死信处理：被丢弃的消息通过 `DeadLetterHandler` 回调

#### Context（上下文）
- `Self()` / `Sender()` / `Message()`：获取当前消息元信息
- `Reply(msg)`：应答 Ask 消息（防重复回复）
- `Spawn` / `Stop`：子 Actor 管理
- `Watch` / `Unwatch`：监听 Actor 终止事件
- `ScheduleAfter`：延迟消息调度

#### 中间件
- `Middleware func(next ReceiveFunc) ReceiveFunc`：洋葱模型
- 支持日志、指标、限流等横切关注点

#### 其他组件
- **Future**：Ask 模式的异步等待实现，带超时控制
- **WatchManager**：Actor 终止信号分发，支持 Watch/Unwatch
- **TimerManager**：延迟消息调度，Actor 停止时自动清理
- **Metrics**：完整的运行时指标收集（消息量、队列深度、panic 次数、重启次数等）
- **DeadLetter**：无法投递的消息回调机制
- **Signal**：`PoisonPill`（毒丸停止）、`Terminated`（终止通知）

## 业务领域

### 会话域（conversation）
- **ManagerActor**（名称：`conv-manager`）：管理会话列表、创建私聊/群聊
- **ConversationActor**（名称：`conv:<convID>`）：单个会话的状态机
  - 按需创建：通过 `GetOrCreate` 懒加载
  - `OnStart` 时从 DB 加载会话信息和成员列表到内存
  - 支持：更新信息、删除、成员管理（增/删/退/角色/转让群主/解散）、置顶、免打扰、已读标记
  - 删除/解散后发送 `PoisonPill` 自杀

### 消息域（message）
- **MessageStoreActor**（名称：`msg-store`）：消息存储与查询
  - 消息写入、分页拉取（基于 seq 游标）、关键词搜索

### 用户域（user）
- **ManagerActor**（名称：`user-manager`）：注册、登录、搜索用户
- **SessionActor**（名称：`session:<uid>`）：用户会话状态
  - 按需创建：登录时通过 `GetOrCreate` 懒加载
  - 支持：查看资料、更新资料、修改密码

### 好友域（friend）
- **ManagerActor**（名称：`friend-manager`）：好友关系全生命周期
  - 好友请求（发送/接收/处理）、好友管理（删除/备注/分组移动）、好友分组（CRUD/排序）

### 在线状态域（presence）
- **PresenceActor**（名称：`presence`）：在线状态管理（桩实现）

## 网关层

### HTTP API
基于 Gin 框架，统一前缀 `/api`，认证路由组使用 JWT 中间件：

| 模块 | 路由 | 说明 |
|------|------|------|
| 认证 | POST /api/auth/register | 注册 |
| 认证 | POST /api/auth/login | 登录 |
| 会话 | GET /api/conversations | 会话列表 |
| 会话 | POST /api/conversations/private | 创建私聊 |
| 会话 | POST /api/conversations/group | 创建群聊 |
| 会话 | GET/POST/PUT/DELETE /api/conversations/:conv_id/... | 会话操作 |
| 消息 | GET /api/messages | 消息列表 |
| 消息 | GET /api/messages/search | 消息搜索 |
| 好友 | POST/GET/PUT/DELETE /api/friends/... | 好友管理 |
| 用户 | GET/PUT /api/users/... | 用户信息 |

### WebSocket
- 端点：`GET /ws`，需 Auth 中间件
- 每个连接创建一个 `GatewayActor`（名称：`gw:<timestamp>`）
- 协议格式：`{type, data}` 的 JSON 结构

## 数据访问层（DAL）

### 数据模型
| 模型 | 说明 |
|------|------|
| Conversation | 会话（私聊 type=1 / 群聊 type=2） |
| Member | 会话成员（联合唯一索引 conv+user） |
| UserConversation | 用户-会话关联（置顶/免打扰/未读数） |
| User | 用户 |
| Message | 消息（支持回复、@、撤回、编辑） |
| FriendRequest | 好友请求 |
| FriendGroup | 好友分组 |
| Friend | 好友关系（双向存储） |

### Store 接口
- `ConvStore`：会话 CRUD、成员管理、转让群主（事务）
- `MsgStore`：消息写入、分页查询、搜索
- `UserStore`：用户 CRUD、认证
- `FriendStore`：好友请求、好友关系、分组管理（接受请求/删除好友使用事务）

所有 Store 均基于 GORM 实现，通过接口抽象便于测试和替换。

## 请求处理流程

```
HTTP Request → Gin Handler → Handler.askManager/askConv → ActorRef.Ask → Mailbox.Push → processLoop → Actor.Receive → ctx.Reply → Future.wait → Handler → HTTP Response
```

1. HTTP 请求到达 Gin Handler
2. Handler 构造命令/查询消息，通过 `ActorRef.Ask` 发送给目标 Actor
3. 消息进入 Actor 的 Mailbox 排队
4. Actor 的 processLoop 取出消息，调用 `Receive` 处理
5. 处理完毕后通过 `ctx.Reply` 将结果写回 Future
6. Handler 从 Future 获取结果，返回 HTTP 响应

## 当前状态与待完善

| 模块 | 状态 | 说明 |
|------|------|------|
| Actor 框架 | ✅ 完整 | 包含测试（单元/基准/稳定性） |
| 会话域 | ✅ 完整 | 含 Manager + 实例 Actor |
| 消息域 | ✅ 基本完成 | 存储与查询，缺少实时推送 |
| 用户域 | ✅ 基本完成 | 注册/登录/资料管理 |
| 好友域 | ✅ 完整 | 请求/关系/分组 |
| 在线状态域 | ✅ 基本完成 | PresenceActor 支持在线连接注册、离线、批量在线状态查询 |
| GatewayActor | ✅ 基本完成 | WebSocket 连接管理、消息路由、频率限制、Presence 注册 |
| 实时推送 | ✅ 基本完成 | MessagePushActor 订阅领域事件并推送消息、撤回、成员、好友、在线状态 |
| JWT 认证 | ✅ 完整 | Generate/Parse 已实现并有测试 |
| Auth 中间件 | ✅ 完整 | HTTP/WS 鉴权已接入 |
| DB 初始化 | ✅ 完整 | 服务启动初始化 SQLite、PRAGMA、AutoMigrate、索引和上传目录 |
| 密码安全 | ✅ 完整 | 使用 bcrypt 哈希与校验 |
| 前端测试 | ✅ 基本完成 | Vitest 覆盖聊天核心 model/view model/realtime，CI 运行 lint/test/coverage/build |
| WAL | ⚠️ 保留 | 当前系统主要依赖 SQLite WAL，`internal/wal` 为轻量接口/测试保留 |

## 编码规范

- 消息类型定义在各域的 `messages.go` 中，命名遵循 `XxxCmd`（写操作）/ `XxxQuery`（读操作）模式
- 统一返回 `Result{Data any, Err error}` 结构
- HTTP 响应使用 `resp.OK` / `resp.Fail` 统一格式 `{code, message, data}`
- Actor 命名约定：`conv:<id>`、`session:<uid>`、`gw:<connID>` 等
- Store 接口与实现分离，便于 mock 测试
- 时间统一使用 Unix 时间戳（`int64`）
- 修改 HTTP API、WebSocket payload、DTO 字段或错误码时，必须同步更新 `backend/docs/api.md`，必要时同步 `README.md` 和相关前端类型。
- 会话类型约定固定为 `1=private`、`2=group`，前后端不得使用成员数量推断会话类型。

## CI/CD 门禁

### 后端门禁

- GitHub Actions 在 PR 和 main push 上执行 `go test ./... -coverprofile=coverage.out -covermode=atomic`。
- 非测试 Go 源文件的增量 per-file 覆盖率必须 `>=80%`。
- 修改后端生产代码时，必须补充或更新对应测试；无法覆盖时需要在 PR 中说明原因。
- 部署前执行 `go mod tidy` 和 `go build ./...`。

### 前端门禁

- GitHub Actions 在 PR 和 main push 上执行 `npm ci`、`npm run lint`、`npm test`、`npm run test:coverage`、`npm run build`。
- `npm run lint` 使用 ESLint 检查 TypeScript、React Hooks 规则和未使用代码；`react-hooks/rules-of-hooks` 与 `react-hooks/exhaustive-deps` 均为 error，禁止遗漏 effect/callback 依赖。
- `npm run test:coverage` 使用 Vitest + V8 coverage，当前覆盖范围为 `reducers/chatReducer.ts`、`models/contactViewModel.ts`、`models/conversationModel.ts`、`models/conversationViewModel.ts`、`models/friendModel.ts`、`models/messageModel.ts`、`models/realtimeModel.ts` 这些聊天核心逻辑文件。
- 前端覆盖率阈值：statements `>=80%`、lines `>=80%`、functions `>=80%`、branches `>=70%`。
- 组件层优先通过复用组件、TypeScript strict、ESLint 和 build 守门；复杂组件逻辑必须先下沉到 `hooks/chat/models/*Model.ts` 或 `hooks/chat/models/*ViewModel.ts` 再补单测。

### 本地 CI

- 提交或推送前优先运行 `./scripts/local-ci.sh`，它按远程 CI 的主要检查执行后端 `go test`、增量 per-file 覆盖率、`go build ./...`，以及前端 `npm ci`、lint、unit test、coverage、build。
- `./scripts/local-ci.sh` 默认使用 `HEAD~1..HEAD` 作为后端增量覆盖率比较范围；需要模拟 PR 或指定基线时，用 `CI_BASE=origin/main ./scripts/local-ci.sh` 或 `CI_BASE=<commit> ./scripts/local-ci.sh`。

## 前端架构原则

### 目标

前端应优先保证聊天核心路径稳定，避免会话类型、系统消息、未读数、隐藏会话、头像展示等高频业务规则散落在组件中导致回归。任何新增业务规则都要先找到可测试的纯逻辑落点，再接入 UI。

### 分层约束

- `components/` 只负责展示、交互触发和组合，不直接承载复杂业务判断。
- `components/ui/` 放无业务语义的函数式基础组件，例如 `Avatar`、`Badge`、`PanelHeader`、`SwitchRow`。
- 业务复用组件放在 `components/`，例如 `ConversationAvatar`、`ConversationSummaryRow`、`MessageList`、`MessageComposer`、`ContactFriendItem`，用于统一会话头像、标题、列表行、消息列表、输入区和联系人行等展示规则。
- `hooks/chat/models/` 放纯函数、视图模型和状态计算，禁止直接调用 API、WebSocket、DOM。
- `hooks/chat/actions/` 负责组织副作用和 API 编排，副作用执行后只通过明确的状态更新函数落库到 store。
- `hooks/chat/reducers/` 放 reducer 与 action/state 类型，核心聊天状态变更必须走 reducer。
- `hooks/chat/effects/` 放生命周期 effect，例如 WebSocket 连接、ref 同步、轮询、通知自动消失。
- `hooks/chat/__tests__/` 放聊天核心逻辑单测，测试文件按被测模块命名。
- `useChatStore` 只作为状态编排层，不继续堆积新的业务分支；新增逻辑优先下沉到 `chat/models/`、`chat/actions/` 或 `chat/effects/`。

### 短期原则

- 继续把 `useChatStore` 里的纯逻辑下沉到 `hooks/chat/models/`。
- 每次修改会话、消息、成员、实时事件的业务规则，都必须补对应 model/realtime 单测。
- 会话类型必须以服务端 `conv.type` 为准，不允许用成员数推断私聊/群聊。
- 系统消息展示必须走统一判断函数，兼容历史类型但新增逻辑使用当前协议类型。

### 中期原则

- 核心聊天状态已引入 `chatReducer`，`baseLoaded`、`hydrateChatMeta`、`hideConversation`、`openConversation`、`markConversationRead`、`incomingMessage`、`messagesLoaded`、`deleteLocalMessage` 等高风险状态变更必须继续走 reducer。
- reducer 输入是明确 action，输出是新 state，副作用只放 `useChatStore` 或 `hooks/chat/*Actions.ts`，不得在组件中直接拼业务状态。
- `useChatStore` 的数据加载编排已拆到 `actions/chatDataActions.ts`，搜索/导航编排已拆到 `actions/navigationActions.ts`，生命周期副作用已拆到 `effects/useChatLifecycleEffects.ts`；后续新增 API 加载、搜索、页面切换、WebSocket/effect 逻辑不得直接塞回 store 主体。
- 本地状态语义要明确区分：hide 是前端隐藏，不删消息；delete message 是本地删除，不影响服务端历史。

### 长期原则

- 组件只展示状态和触发事件，业务规则不散落在 `ConversationList`、`RightPane`、`ChatDetailPanel` 等页面级组件里。
- 头像、列表行、面板头部、空状态、开关行等可复用结构必须抽函数式组件复用。
- 大组件必须持续拆分：`ChatWindow` 已拆出 `useChatScroll`、`MessageList`、`MessageComposer`；`ContactsPanel` 已拆出 `ContactFriendItem`、`ContactSearchResults`、`FriendListSection`、`GroupListSection`。
- 优先维护小而稳定的组件：基础 UI 组件无业务依赖，业务组件只依赖视图模型，不直接复制业务判断。
- 架构重构必须保持测试先行或测试同步，`npm test` 和 `npm run build` 必须通过后再交付。
