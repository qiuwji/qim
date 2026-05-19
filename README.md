# QIM

QIM 是一个轻量级即时通讯系统，包含 React/Vite 前端、Go 后端、WebSocket 实时通信、SQLite 持久化、好友关系、群聊、消息撤回、未读数、在线状态和图片上传等能力。

## 功能概览

- 用户注册、登录、资料修改、头像上传、密码修改。
- 好友搜索、好友申请、同意/拒绝、好友分组、备注、删除好友。
- 单聊和群聊，支持创建群聊、从好友列表邀请成员、群管理、群主转让、解散群聊。
- WebSocket 实时消息、正在输入、消息撤回、在线状态、好友在线列表同步。
- 消息分页加载、聊天记录搜索、本地删除消息、未读红点、会话置顶和免打扰。
- 图片消息上传和访问。

## 技术栈

### 前端

- React + TypeScript + Vite。
- 原生 CSS 响应式布局，支持桌面端和移动端。
- HTTP API 封装在 `frontend/src/api/http.ts`。
- WebSocket 客户端封装在 `frontend/src/api/ws.ts`。
- 聊天状态主要由 `frontend/src/hooks/useChatStore.ts` 管理，消息/会话规则拆在 `frontend/src/hooks/chat/`。

### 后端

- Go 1.26。
- Gin 提供 HTTP API。
- Gorilla WebSocket 提供实时通信。
- GORM + SQLite 持久化。
- Zap 日志、JWT 鉴权。
- 自研 Actor Runtime，核心代码在 `backend/internal/actor/`。

## 架构设计

QIM 后端按分层和领域拆分：

```text
backend/
  cmd/server/                 # 服务启动、依赖初始化、数据库迁移
  internal/actor/             # Actor 引擎、邮箱、Future、Watch、Metrics
  internal/domain/            # 领域模型和 Actor
  internal/dal/               # GORM 数据访问层
  internal/eventbus/          # 领域事件总线
  internal/middleware/        # 鉴权、LogID 等中间件
  internal/service/           # HTTP/WS 到 Actor 的服务门面
  internal/transport/http/    # HTTP Handler
  internal/transport/ws/      # WebSocket Gateway、路由、推送
  internal/pkg/               # 通用错误、JWT、日志、响应
```

### Actor 模型

后端的核心业务使用 Actor 管理状态和并发：

- `ConversationActor`：负责单个会话的消息发送、撤回、成员管理、群信息更新。
- `ManagerActor`：负责用户维度的会话列表、置顶、免打扰、已读等聚合操作。
- `User Manager/Session Actor`：负责注册、登录、资料、密码、用户在线事件。
- `Friend ManagerActor`：负责好友申请、好友列表、好友分组。
- `PresenceActor`：负责在线连接、批量在线状态查询。
- `MessagePushActor`：订阅领域事件，并把消息、撤回、成员变化、在线状态推送给相关连接。

这样设计可以把高并发 IM 场景里的状态修改串行化到对应 Actor 内部，减少锁和竞态问题。

### 领域事件

核心业务动作会发布领域事件：

- `message.sent`
- `message.revoked`
- `conversation.updated`
- `member.joined`
- `member.left`
- `member.kicked`
- `friend.request_created`
- `presence.online`
- `presence.offline`

WebSocket 推送层通过事件订阅把领域事件转换为前端可消费的实时通知。

### 数据存储

当前使用 SQLite，默认数据库路径：

```text
backend/data/qim.db
```

启动时后端会自动：

- 创建数据库目录。
- 初始化 SQLite PRAGMA。
- 执行 GORM `AutoMigrate`。
- 创建消息幂等唯一索引。

因此本地开发和单机部署时不需要手动准备数据库文件。

上传图片默认保存在：

```text
backend/data/uploads/images
```

生产环境需要定期备份 `backend/data/`。

### 前端状态设计

前端按“服务端事实 + 本地投影 + UI 临时状态”组织：

- 服务端事实：会话、消息、好友、成员、用户资料。
- 本地投影：本地删除消息 ID、会话预览、当前用户缓存。
- UI 临时状态：弹窗、右键菜单、搜索框、移动端面板、悬浮卡片。

消息相关规则集中在 `frontend/src/hooks/chat/messageModel.ts`，会话列表规则集中在 `frontend/src/hooks/chat/conversationModel.ts`，避免删除、撤回、未读、预览互相覆盖。

## 目录结构

```text
qim/
  backend/                    # Go 后端
    cmd/server/               # 服务入口
  api/                        # API 文档（按域拆分）
    README.md                   # 鉴权/响应格式/错误码
    http-auth.md                # 认证
    http-user.md                # 用户
    http-friend.md              # 好友
    http-conversation.md        # 会话/消息/群组
    http-file.md                # 文件上传
    ws.md                       # WebSocket 协议
    internal/                 # 后端核心代码
  frontend/                   # React 前端
    src/api/                  # HTTP / WebSocket API
    src/components/           # UI 组件
    src/hooks/                # 状态和业务 Hook
    src/styles/global.css     # 全局样式
```

## 本地开发

### 环境要求

- Go 1.26+
- Node.js 20+
- npm

### 启动后端

```bash
cd backend
go mod download
go run ./cmd/server
```

默认监听：

```text
http://localhost:8080
```

常用环境变量：

```bash
export QIM_JWT_SECRET="your-secret"
export QIM_DB_PATH="./data/qim.db"
```

如果不设置，开发环境会使用默认值。

### 启动前端

```bash
cd frontend
npm install
npm run dev
```

Vite 开发服务会把以下路径代理到后端：

- `/api`
- `/ws`
- `/uploads`

### 构建前端

```bash
cd frontend
npm run build
```

构建产物：

```text
frontend/dist
```

### 测试后端

```bash
cd backend
go test ./...
```

## 使用说明

1. 打开前端页面。
2. 注册账号，账号只能是纯数字字符串，且不能重复。
3. 登录后在 `+` 菜单中添加好友或创建群聊。
4. 添加好友时使用对方注册账号。
5. 创建群聊或邀请群成员时，可以从好友列表中多选。
6. 进入会话后可以发送文字和图片消息。
7. 右键消息可以回复、复制、转发、撤回或本地删除。
8. 群主和管理员可以在群详情中管理群成员和群信息。

## 单机部署

推荐部署方式：

```text
Nginx
  /              -> frontend/dist
  /api/...       -> 127.0.0.1:8080
  /ws            -> 127.0.0.1:8080
  /uploads/...   -> 127.0.0.1:8080/uploads/...

Go Backend
  SQLite: backend/data/qim.db
  Uploads: backend/data/uploads/images
```

### 构建

```bash
cd frontend
npm install
npm run build

cd ../backend
go mod download
go build -o qim-server ./cmd/server
```

### systemd 示例

```ini
[Unit]
Description=QIM Server
After=network.target

[Service]
WorkingDirectory=/opt/qim/qim/backend
ExecStart=/opt/qim/qim/backend/qim-server
Restart=always
RestartSec=3
Environment=QIM_JWT_SECRET=replace-with-random-secret
Environment=QIM_DB_PATH=/opt/qim/qim/backend/data/qim.db

[Install]
WantedBy=multi-user.target
```

### Nginx 示例

```nginx
server {
    listen 80;
    server_name your-domain.com;

    root /opt/qim/qim/frontend/dist;
    index index.html;

    client_max_body_size 10m;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8080/api/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location /ws {
        proxy_pass http://127.0.0.1:8080/ws;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_read_timeout 3600s;
    }

    location /uploads/ {
        proxy_pass http://127.0.0.1:8080/uploads/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
    }
}
```

## API 文档

HTTP 和 WebSocket 协议见：

```text
api/
```

修改或新增 API 时需要同步更新该文档。

## 生产注意事项

- 必须设置强随机 `QIM_JWT_SECRET`。
- 不要把 `backend/data/` 提交到 Git。
- 定期备份 `backend/data/qim.db` 和 `backend/data/uploads/`。
- 线上建议启用 HTTPS，WebSocket 使用 `wss://`。
- SQLite 适合单机和早期阶段；用户量增长后可以迁移到 MySQL/PostgreSQL。
- 后端的 WebSocket Origin 策略上线后应限制为可信域名。

## 常用命令

```bash
# 前端构建
cd frontend && npm run build

# 后端测试
cd backend && go test ./...

# 后端构建
cd backend && go build -o qim-server ./cmd/server

# 查看服务状态
systemctl status qim

# 查看服务日志
journalctl -u qim -f
```
