# QIM 后端架构

## 一、分层架构总览

```mermaid
graph TB
    subgraph Client["客户端"]
        Browser["浏览器 / 移动端"]
    end

    subgraph Transport["传输层 (internal/transport)"]
        HTTP["HTTP Server (Gin)\n/api/*  RESTful 接口"]
        WS["WebSocket Gateway\n/ws 实时长连接"]
    end

    subgraph Middleware["中间件 (internal/middleware)"]
        Auth["Auth\nJWT 鉴权"]
        LogID["LogID\n请求链路追踪"]
        CORS["CORS\n跨域处理"]
    end

    subgraph Service["服务门面层 (internal/service)"]
        ConvSvc["ConvService\n会话服务"]
        UserSvc["UserService\n用户服务"]
        MsgSvc["MsgService\n消息服务"]
        FriendSvc["FriendService\n好友服务"]
    end

    subgraph Domain["领域层 (internal/domain)"]
        subgraph Actors["Actor 模型"]
            ConvMgr["conv-manager\nConversationManagerActor\n会话列表、置顶、免打扰、已读"]
            ConvActor["conv:{id}\nConversationActor\n消息收发、撤回、成员管理"]
            UserMgr["user-manager\nUserManagerActor\n注册、登录、搜索、资料"]
            FriendMgr["friend-manager\nFriendManagerActor\n好友申请、列表、分组"]
            PresenceActor["presence\nPresenceActor\n在线状态管理"]
            MsgStoreActor["message-store\nMessageStoreActor\n消息持久化、分页、搜索"]
        end
    end

    subgraph Engine["Actor 引擎 (internal/actor)"]
        EngineCore["Engine\nSpawn / Stop / Lookup\nMailbox / Supervision / Metrics"]
        EventBusActor["eventbus\nEventBusActor\n事件订阅与发布"]
    end

    subgraph Push["推送层 (internal/transport/ws)"]
        PushHandler["message-push\nMessagePushActor\n订阅事件 → 推送给在线用户"]
    end

    subgraph Data["数据层"]
        DAL["DAL (internal/dal)\nGORM 数据访问"]
        SQLite[("SQLite\nqim.db\nWAL 模式")]
        FileStore[("文件存储\ndata/uploads/")]
    end

    subgraph Infra["基础设施 (internal/pkg)"]
        JWT["JWT\n令牌管理"]
        Logx["Logx\nZap 日志适配"]
        AppErr["AppErr\n错误码体系"]
    end

    Browser -->|"HTTPS"| HTTP
    Browser -->|"WSS"| WS
    HTTP --> Middleware
    WS --> Middleware
    Middleware --> Service
    Service -->|"Ask/Tell"| Domain
    Domain --> EngineCore
    Domain -->|"Publish"| EventBusActor
    EventBusActor -->|"Subscribe → Push"| PushHandler
    PushHandler -->|"WriteJSON"| WS
    Domain --> DAL
    DAL --> SQLite
    HTTP --> FileStore

    style Client fill:#e1f5fe
    style Transport fill:#fff3e0
    style Middleware fill:#f3e5f5
    style Service fill:#e8f5e9
    style Domain fill:#fce4ec
    style Engine fill:#e0f2f1
    style Push fill:#fce4ec
    style Data fill:#fff8e1
    style Infra fill:#eceff1
```

---

## 二、Actor 模型全景

```mermaid
graph TB
    Engine["Actor Engine\n(mailbox / supervision / metrics / watch / timer)"]

    Engine --> EventBus
    Engine --> ConvMgr
    Engine --> UserMgr
    Engine --> FriendMgr
    Engine --> Presence
    Engine --> MsgStore
    Engine --> PushHandler
    Engine --> Gateway

    subgraph DomainActors["领域 Actor"]
        ConvMgr["conv-manager\nManagerActor\n──────\n会话列表查询\n创建私聊/群聊\n置顶/免打扰/已读"]
        ConvActor["conv:{id}\nConversationActor\n（每个会话一个实例）\n──────\n消息发送/撤回\n成员增删/角色\n群信息更新\n6h 闲置回收"]
        UserMgr["user-manager\nManagerActor\n──────\n注册/登录\n用户搜索\n资料修改\n密码校验"]
        FriendMgr["friend-manager\nManagerActor\n──────\n好友申请/审批\n好友列表/分组\n备注/删除"]
        Presence["presence\nPresenceActor\n──────\n注册在线连接\n批量查询在线状态\n6h 闲置回收"]
        MsgStore["message-store\nMessageStoreActor\n──────\n消息持久化\n分页加载\n聊天记录搜索"]
    end

    subgraph Bus["事件总线"]
        EventBus["eventbus\nEventBusActor\n──────\nSubscribe / Unsubscribe\nPublish / Watch"]
    end

    subgraph PushActors["推送 Actor"]
        PushHandler["message-push\nMessagePushActor\n──────\n订阅领域事件\n路由到在线用户\n批量推送"]
    end

    subgraph GatewayActors["网关 Actor"]
        Gateway["gateway:{uid}\nGatewayActor\n（每个连接一个实例）\n──────\nWS 消息收发\n频率限制(10/s)\nPresence 注册\n在线事件发布\n6h 闲置回收"]
    end

    ConvMgr -.->|"Ask: 创建会话"| ConvActor
    ConvMgr -->|"Publish: conversation.*"| EventBus
    ConvActor -->|"Publish: message.* / member.*"| EventBus
    UserMgr -->|"Publish: user.online"| EventBus
    FriendMgr -->|"Publish: friend.*"| EventBus
    Presence -->|"Publish: presence.*"| EventBus
    EventBus -->|"Subscribe"| PushHandler
    PushHandler -.->|"Ask: 查询好友列表"| FriendMgr
    PushHandler -.->|"Ask: 查询在线状态"| Presence
    PushHandler -->|"WriteJSON: 推送到网关"| Gateway

    style Engine fill:#e0f2f1
    style EventBus fill:#ffcc80
    style ConvMgr fill:#ef9a9a
    style ConvActor fill:#ef9a9a
    style UserMgr fill:#90caf9
    style FriendMgr fill:#a5d6a7
    style Presence fill:#ce93d8
    style MsgStore fill:#bcaaa4
    style PushHandler fill:#ffab91
    style Gateway fill:#80cbc4
```

---

## 三、消息发送全链路

```mermaid
sequenceDiagram
    participant Client as 客户端 A
    participant Gateway as GatewayActor<br/>(A 的连接)
    participant WS as WS Dispatcher
    participant ConvSvc as ConvService
    participant ConvActor as ConversationActor
    participant MsgStore as MessageStoreActor
    participant EventBus as EventBus
    participant PushHandler as MessagePushActor
    participant GatewayB as GatewayActor<br/>(B 的连接)
    participant ClientB as 客户端 B

    Client->>Gateway: {"type":"message","action":"send","data":{...}}
    Gateway->>Gateway: 频率限制检查 (10次/秒)
    Gateway->>WS: 路由到 Dispatcher
    WS->>ConvSvc: TellConv(convID, SendMessageCmd)
    ConvSvc->>ConvActor: Ask: SendMessageCmd
    ConvActor->>ConvActor: 权限校验 (是否为成员)
    ConvActor->>MsgStore: Ask: StoreMsgCmd
    MsgStore->>MsgStore: 写入 SQLite + 幂等去重
    MsgStore-->>ConvActor: MessageDTO
    ConvActor->>ConvActor: 更新 maxSeq
    ConvActor->>EventBus: Publish: MessageSentEvent
    ConvActor-->>ConvSvc: Result{Data: MessageDTO}
    ConvSvc-->>WS: MessageDTO
    WS-->>Gateway: WsResponse{type:"message", action:"sent"}
    Gateway-->>Client: 确认消息已发送
    EventBus->>PushHandler: EventEnvelope{MessageSentEvent}
    PushHandler->>PushHandler: 提取 memberUIDs 列表
    PushHandler->>GatewayB: PushCmd → WriteJSON
    GatewayB-->>ClientB: {"type":"message","action":"new",...}
```

---

## 四、事件驱动体系

```mermaid
graph LR
    subgraph Publishers["事件发布者"]
        ConvActor_P["ConversationActor"]
        ConvMgr_P["ConversationManager"]
        UserMgr_P["UserManager"]
        Presence_P["PresenceActor"]
        FriendMgr_P["FriendManager"]
        Gateway_P["GatewayActor"]
    end

    subgraph Events["领域事件"]
        E1["conversation.message_sent"]
        E2["conversation.message_revoked"]
        E3["conversation.member_joined"]
        E4["conversation.member_left"]
        E5["conversation.member_kicked"]
        E6["conversation.owner_transferred"]
        E7["conversation.group_dissolved"]
        E8["conversation.updated"]
        E9["presence.online"]
        E10["presence.offline"]
        E11["user.online"]
        E12["friend.request_created"]
    end

    subgraph EventBus_Core["EventBusActor"]
        SubTable["订阅表\nmap[eventName]map[subscriberName]*ActorRef"]
        Watch["死亡监控\n订阅者退出 → 自动清理"]
    end

    subgraph Subscribers["事件订阅者"]
        PushHandler_S["MessagePushActor\n推送消息给在线用户"]
        UserMgr_S["UserManagerActor\n处理好友在线通知"]
    end

    ConvActor_P --> E1
    ConvActor_P --> E2
    ConvActor_P --> E3
    ConvActor_P --> E4
    ConvActor_P --> E5
    ConvActor_P --> E6
    ConvActor_P --> E7
    ConvActor_P --> E8
    Presence_P --> E9
    Presence_P --> E10
    Gateway_P --> E11
    FriendMgr_P --> E12
    E1 --> EventBus_Core
    E2 --> EventBus_Core
    E3 --> EventBus_Core
    E4 --> EventBus_Core
    E5 --> EventBus_Core
    E6 --> EventBus_Core
    E7 --> EventBus_Core
    E8 --> EventBus_Core
    E9 --> EventBus_Core
    E10 --> EventBus_Core
    E11 --> EventBus_Core
    E12 --> EventBus_Core
    EventBus_Core --> PushHandler_S
    EventBus_Core --> UserMgr_S

    style EventBus_Core fill:#ffcc80
    style PushHandler_S fill:#ffab91
    style UserMgr_S fill:#90caf9
```

---

## 五、Actor 引擎内部结构

```mermaid
graph TB
    subgraph EngineAPI["Engine API"]
        Spawn["Spawn(name, Actor)"]
        Stop["Stop(name)"]
        Lookup["Lookup(name)"]
        GetOrCreate["GetOrCreate(name, factory)"]
    end

    subgraph Instance["actorInstance"]
        ActorObj["Actor\n(Receive 方法)"]
        ActorRef["ActorRef\n(Tell / Ask / Done)"]
        ActorCtx["actorContext\n(Self / Sender / Watch / Spawn)"]
        Lifecycle["Lifecycle\n(OnStart / OnStop)"]
    end

    subgraph Mailbox["邮箱系统"]
        ChanQueue["有界通道队列\n(默认 256)"]
        Policy["饱和策略\nBlock / DropNewest\nDropOldest / Reject"]
        Push["Push(envelope)"]
        Pull["Pull() (envelope, ok)"]
    end

    subgraph Dispatch["消息分发"]
        MiddlewareChain["中间件链\nMiddleware → Middleware → Receive"]
        PanicRecover["Panic 恢复\nSupervision 策略"]
        MetricsCollect["Metrics 收集\n消息数 / 邮箱深度 / Panic 数"]
    end

    subgraph Supervision["监督策略"]
        Resume["Resume\n恢复并继续"]
        Stop["Stop\n停止 Actor"]
        Restart["Restart\n新实例重启 + 节流"]
    end

    subgraph Extras["扩展能力"]
        WatchMgr["WatchManager\nActor 死亡监控"]
        TimerMgr["TimerManager\nScheduleAfter 定时器"]
        DeadLetter["DeadLetterHandler\n死信处理"]
    end

    Spawn --> Instance
    Instance --> Mailbox
    Mailbox --> Dispatch
    Dispatch --> Supervision
    EngineAPI --> Extras

    style EngineAPI fill:#e0f2f1
    style Instance fill:#b2dfdb
    style Mailbox fill:#80cbc4
    style Dispatch fill:#4db6ac
    style Supervision fill:#26a69a
    style Extras fill:#009688
```

---

## 六、目录结构与分层映射

```
backend/
├── cmd/server/                入口 → app.go: 依赖初始化、启动服务
│
├── internal/
│   ├── transport/             传输层
│   │   ├── http/              ── Gin Handler，参数解析，调用 Service
│   │   ├── ws/                ── WebSocket Gateway、Dispatcher、PushHandler
│   │   └── server.go          ── 路由注册、中间件挂载
│   │
│   ├── service/               服务门面层
│   │   ├── conv_service.go    ── AskConv / AskManager / TellConv
│   │   ├── user_service.go    ── 用户 Actor 的 Ask 封装
│   │   ├── msg_service.go     ── 消息 Actor 的 Ask 封装
│   │   └── friend_service.go  ── 好友 Actor 的 Ask 封装
│   │
│   ├── domain/                领域层 (Actor 实现)
│   │   ├── conversation/      ── ConversationActor + ManagerActor + Store + Events
│   │   ├── friend/            ── FriendManagerActor + Events
│   │   ├── message/           ── MessageStoreActor
│   │   ├── presence/          ── PresenceActor
│   │   └── user/              ── UserManagerActor + Events
│   │
│   ├── eventbus/              事件总线
│   │   └── actor_bus.go       ── EventBusActor (订阅/发布/Watch)
│   │
│   ├── actor/                 Actor 引擎 (自研)
│   │   ├── engine.go          ── Spawn / Stop / Lookup / GetOrCreate
│   │   ├── actor.go           ── Actor 接口 + ActorRef (Tell/Ask)
│   │   ├── context.go         ── Context 接口 (Self/Sender/Watch/Spawn)
│   │   ├── mailbox.go         ── 有界队列 + 4 种饱和策略
│   │   ├── option.go          ── EngineOption 函数式配置
│   │   ├── metrics.go         ── Metrics 接口 + 默认实现
│   │   ├── middleware.go      ── 消息中间件链
│   │   ├── timer.go           ── ScheduleAfter 定时器
│   │   ├── watch_manager.go   ── Actor 死亡监控
│   │   ├── dead_letter.go     ── 死信处理
│   │   ├── signal.go          ── PoisonPill / Terminated
│   │   ├── future.go          ── Ask 模式的 Future 实现
│   │   └── errors.go          ── 引擎级错误定义
│   │
│   ├── dal/                   数据访问层
│   │   ├── models.go          ── GORM Model 定义
│   │   ├── user_store.go      ── 用户 CRUD
│   │   ├── conv_store.go      ── 会话/成员 CRUD
│   │   ├── msg_store.go       ── 消息 CRUD + 幂等索引
│   │   └── friend_store.go    ── 好友 CRUD
│   │
│   ├── middleware/             HTTP 中间件
│   │   ├── auth.go            ── JWT 鉴权 + 上下文注入
│   │   └── logid.go           ── LogID 生成与响应头注入
│   │
│   └── pkg/                   通用基础设施
│       ├── jwt/               ── JWT 签发与校验
│       ├── logx/              ── Zap 日志 + Actor Logger 适配
│       ├── apperr/            ── 错误码体系 (code + message)
│       └── resp/              ── HTTP 响应封装
```

---

## 七、关键设计决策

| 决策 | 说明 |
|------|------|
| Actor 模型管理状态 | 每个会话/用户/好友的状态串行化到对应 Actor，消除锁竞争 |
| Ask/Tell 模式 | 同步请求用 Ask（带 5s 超时），异步通知用 Tell |
| 6 小时闲置回收 | ConversationActor / PresenceActor / GatewayActor 超时自动 PoisonPill |
| EventBus 解耦推送 | 领域 Actor 只发布事件，MessagePushActor 订阅后负责推送 |
| Service 门面层 | HTTP/WS Handler 不直接操作 Actor，通过 Service 层的 Ask/Tell 封装 |
| 函数注入解耦 | Service 通过 `newConvFn` 工厂函数获取 Actor，不直接 import domain 包 |
| SQLite WAL 模式 | 提升并发读性能，`busy_timeout=5000` 避免锁冲突 |
| 消息幂等去重 | `(conversation_id, client_id)` 联合唯一索引，防止重复提交 |
| Logger 接口解耦 | Actor 引擎只依赖 `actor.Logger` 接口（一个 Printf 方法），通过适配器桥接 Zap |