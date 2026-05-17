# QIM API 文档

本文档描述当前代码已实现的 HTTP 与 WebSocket 协议。除注册、登录外，HTTP 接口均需要 JWT 鉴权。

## 通用约定

### 鉴权

登录成功后服务端返回 JWT，客户端后续请求使用以下任一方式携带：

```http
Authorization: Bearer <token>
```

WebSocket 也支持 query 参数：

```text
ws://host/ws?token=<token>
```

### HTTP 响应

成功：

```json
{
  "code": "ok",
  "data": {}
}
```

失败：

```json
{
  "code": "unauthorized",
  "message": "invalid token"
}
```

说明：

- `code` 使用字符串错误码，不再使用数字错误码。
- HTTP `log_id` 通过响应头 `X-Log-ID` 返回，不放在响应体。
- HTTP 业务错误当前统一返回 HTTP 200，业务成功失败看 `code`。

常见错误码：

| code | 说明 |
| --- | --- |
| `ok` | 成功 |
| `bad_request` | 请求参数错误 |
| `invalid_request` | 请求类型或 action 不支持 |
| `unauthorized` | 未登录、token 缺失或无效 |
| `rate_limit` | 发消息频率超限（每连接每秒最多 10 条） |
| `internal_error` | 服务端内部错误 |
| `user.invalid_credentials` | 用户名或密码错误 |
| `user.incorrect_password` | 旧密码错误 |
| `user.invalid_username` | 账号格式非法，账号只能是纯数字字符串 |
| `user.weak_password` | 密码强度不足，需 8-20 位且包含大小写字母和特殊字符 |
| `conversation.empty_message` | 消息内容为空 |
| `conversation.not_member` | 不是会话成员 |
| `conversation.member_exists` | 成员已存在 |
| `conversation.member_limit_reached` | 群成员数超限 |
| `conversation.member_not_found` | 成员不存在 |
| `conversation.owner_required` | 需要群主权限 |
| `conversation.admin_required` | 需要群主或管理员权限 |
| `conversation.invalid_role` | 成员角色非法 |
| `conversation.group_required` | 仅群聊支持该操作 |
| `friend.not_your_request` | 不能处理不属于自己的好友申请 |
| `friend.already_friends` | 已经是好友，不能重复发送申请 |
| `friend.pending_request` | 已存在待处理的好友申请 |

## 一、认证

| 方法 | 路径 | 说明 | 请求体 | 响应 data |
| --- | --- | --- | --- | --- |
| `POST` | `/api/auth/register` | 注册，`username` 只能是纯数字字符串且不可重复 | `{username, password, nickname}` | `UserDTO` |
| `POST` | `/api/auth/login` | 登录 | `{username, password}` | `{token, user}` |

注册响应示例：

```json
{
  "code": "ok",
  "data": {
    "id": 1,
    "username": "10001",
    "nickname": "Alice",
    "avatar": "",
    "sign": "",
    "status": 0,
    "created_at": 1710000000,
    "last_online_at": 1710000000
  }
}
```

登录响应示例：

```json
{
  "code": "ok",
  "data": {
    "token": "<jwt>",
    "user": {
      "id": 1,
      "username": "10001",
      "nickname": "Alice",
      "avatar": "",
      "sign": "",
      "status": 0,
      "created_at": 1710000000,
      "last_online_at": 1710000000
    }
  }
}
```

## 二、用户

| 方法 | 路径 | 说明 | 请求体 | 响应 data |
| --- | --- | --- | --- | --- |
| `GET` | `/api/user/profile` | 获取自己的资料 | - | `UserDTO` |
| `PUT` | `/api/user/profile` | 修改资料 | `{nickname?, avatar?, sign?}` | `true` |
| `PUT` | `/api/user/password` | 修改密码 | `{old_password, new_password}` | `true` |
| `GET` | `/api/users/search?keyword=xxx` | 搜索用户 | - | `UserDTO[]` |
| `GET` | `/api/users/:id` | 查看用户资料 | - | `UserDTO` |

`UserDTO` 字段：

```json
{
  "id": 1,
  "username": "10001",
  "nickname": "Alice",
  "avatar": "",
  "sign": "",
  "status": 0,
  "created_at": 1710000000,
  "last_online_at": 1710000000
}
```

## 三、好友

| 方法 | 路径 | 说明 | 请求体 | 响应 data |
| --- | --- | --- | --- | --- |
| `POST` | `/api/friends/request` | 发送好友申请 | `{username, message?}`，兼容旧 `{to_uid, message?}` | `FriendRequestDTO` |
| `GET` | `/api/friends/requests/incoming` | 收到的好友申请 | - | `FriendRequestDTO[]` |
| `GET` | `/api/friends/requests/outgoing` | 发出的好友申请 | - | `FriendRequestDTO[]` |
| `PUT` | `/api/friends/requests/:req_id` | 同意或拒绝 | `{action: "accept"}` 或 `{action: "reject"}` | `true` |
| `DELETE` | `/api/friends/:friend_uid` | 删除好友（逻辑删除，重新加好友可恢复） | - | `true` |
| `GET` | `/api/friends` | 好友列表 | - | `FriendDTO[]` |
| `PUT` | `/api/friends/:friend_uid/remark` | 修改备注 | `{remark}` | `true` |
| `PUT` | `/api/friends/:friend_uid/group` | 移动分组 | `{group_id}` | `true` |

`FriendRequestDTO` 字段：

```json
{
  "id": 1,
  "from_uid": 1,
  "to_uid": 2,
  "message": "hello",
  "status": 0,
  "created_at": 1710000000
}
```

`FriendDTO` 字段：

```json
{
  "id": 1,
  "friend_uid": 2,
  "remark": "Bob",
  "group_id": 0,
  "status": 0,
  "created_at": 1710000000
}
```

说明：

- 当前好友列表和申请列表暂未 join 用户昵称、头像。
- `status`: `0=active`，`1=deleted`（逻辑删除，重新加好友时恢复为 0）。
- 好友申请 `status`: `0=pending`，`1=accepted`，`2=rejected`。

## 四、好友分组

| 方法 | 路径 | 说明 | 请求体 | 响应 data |
| --- | --- | --- | --- | --- |
| `GET` | `/api/friend/groups` | 分组列表 | - | `FriendGroupDTO[]` |
| `POST` | `/api/friend/groups` | 创建分组 | `{name}` | `FriendGroupDTO` |
| `PUT` | `/api/friend/groups/:group_id` | 重命名 | `{name}` | `true` |
| `DELETE` | `/api/friend/groups/:group_id` | 删除分组（逻辑删除） | - | `true` |
| `PUT` | `/api/friend/groups/sort` | 排序 | `{groups: [{group_id, sort_order}]}` | `true` |

`FriendGroupDTO` 字段：

```json
{
  "id": 1,
  "name": "default",
  "sort_order": 0
}
```

说明：

- 当前分组列表暂未返回 `count`。
- 删除分组为逻辑删除，分组内好友不受影响。

## 五、会话

| 方法 | 路径 | 说明 | 请求体 | 响应 data |
| --- | --- | --- | --- | --- |
| `GET` | `/api/conversations` | 会话列表 | - | `UserConvDTO[]` |
| `POST` | `/api/conversations/private` | 创建或获取单聊会话 | `{username}`，兼容旧 `{uid}` | `ConversationDTO` |
| `POST` | `/api/conversations/group` | 创建群聊 | `{name, avatar?, usernames: []}`，兼容旧 `{members: []}` | `ConversationDTO` |
| `PUT` | `/api/conversations/:id/pin` | 置顶或取消置顶 | `{pinned: true}` | `true` |
| `PUT` | `/api/conversations/:id/mute` | 免打扰或取消 | `{muted: true}` | `true` |
| `PUT` | `/api/conversations/:id/read` | 标记已读 | `{seq}` | `true` |
| `PUT` | `/api/conversations/read-all` | 全部标记已读 | - | `true` |

`UserConvDTO` 字段：

```json
{
  "conversation_id": 1,
  "is_pinned": false,
  "is_muted": false,
  "unread_count": 3,
  "last_msg_at": 1710000000,
  "conv": {
    "id": 1,
    "type": 2,
    "name": "group",
    "avatar": "",
    "owner_id": 1,
    "member_count": 3,
    "member_limit": 500,
    "max_seq": 10,
    "created_at": 1710000000
  }
}
```

`ConversationDTO` 字段：

```json
{
  "id": 1,
  "type": 2,
  "name": "group",
  "avatar": "",
  "owner_id": 1,
  "member_count": 3,
  "member_limit": 500,
  "max_seq": 10,
  "created_at": 1710000000
}
```

说明：

- `conv` 为会话元信息，前端应优先使用 `conv.type` 区分 `private/group`，不要通过成员数量推断会话类型。
- 当前后端不提供“删除会话”接口。前端“从聊天列表移除”属于本地隐藏视图语义，不删除后端消息；收到该会话新消息后可重新显示。
- `type`: `1=private`，`2=group`。
- 登录后未读同步建议复用会话列表摘要；后续可在 WS 建连后主动推 `sync.conversations`。

## 六、消息

| 方法 | 路径 | 说明 | 请求体 | 响应 data |
| --- | --- | --- | --- | --- |
| `GET` | `/api/conversations/:id/messages?before_seq=0&limit=20` | 历史消息，按 seq 向前翻 | - | `MessageDTO[]` |
| `GET` | `/api/messages/search?keyword=xxx&conversation_id=1&limit=20` | 搜索消息 | - | `MessageDTO[]` |

兼容参数：

- `GET /api/conversations/:id/messages?seq=100` 等价于 `before_seq=100`。

`MessageDTO` 字段：

```json
{
  "id": 1,
  "conversation_id": 1,
  "seq": 10,
  "sender_id": 1,
  "msg_type": 1,
  "content": "hello",
  "reply_to": 0,
  "revoked": false,
  "edited": false,
  "client_id": "client-msg-id",
  "created_at": 1710000000
}
```

说明：

- 发送消息走 WebSocket `type=msg action=send`。
- `client_id` 用于发送消息幂等，服务端按 `conversation_id + sender_id + client_id` 去重。
- 当前消息查询侧仍需补“当前用户是否是会话成员”的权限校验。

## 七、群组

| 方法 | 路径 | 说明 | 请求体 | 响应 data |
| --- | --- | --- | --- | --- |
| `GET` | `/api/conversations/:id/members` | 群成员列表 | - | `MemberDTO[]` |
| `POST` | `/api/conversations/:id/members` | 邀请单个成员入群 | `{username, role}`，兼容旧 `{uid, role}` | `true` |
| `DELETE` | `/api/conversations/:id/members/:uid` | 踢人（逻辑删除） | - | `true` |
| `DELETE` | `/api/conversations/:id/leave` | 退群（逻辑删除） | - | `true` |
| `PUT` | `/api/conversations/:id/members/:uid/role` | 设置成员角色 | `{role}` | `true` |
| `PUT` | `/api/conversations/:id/owner` | 转让群主 | `{new_owner_id}` | `true` |
| `DELETE` | `/api/conversations/:id/dissolve` | 解散群聊（逻辑删除） | - | `true` |
| `PUT` | `/api/conversations/:id/info` | 修改群信息 | `{name?, avatar?, member_limit?}` | `true` |

`MemberDTO` 字段：

```json
{
  "uid": 1,
  "role": 2,
  "last_read_seq": 10,
  "join_time": 1710000000
}
```

说明：

- 当前邀请入群一次只支持一个账号 `username`，不是 `usernames: []`。
- 当前成员列表暂未 join 用户昵称、头像。
- `role`: `0=普通成员`，`1=管理员`，`2=群主`。

## 八、图片上传

当前已实现图片上传接口，文件会保存到 `data/uploads/images`，并通过 `/uploads/images/<filename>` 访问。

| 方法 | 路径 | 说明 | 请求体 | 响应 data |
| --- | --- | --- | --- | --- |
| `POST` | `/api/files/upload` | 上传图片 | `multipart/form-data`，字段名 `file` | `{url, filename, size, mime_type}` |

说明：

- 当前仅支持图片上传，文件消息、视频消息后续再做。
- 单图大小限制为 10MB。
- 非图片 MIME 类型会返回 `bad_request`。

## 九、WebSocket

连接：

```text
ws://host/ws?token=<jwt>
```

也可以在握手请求中使用：

```http
Authorization: Bearer <jwt>
```

### 通用上行格式

客户端发送：

```json
{
  "type": "msg",
  "action": "send",
  "data": {}
}
```

### 通用下行格式

请求成功响应：

```json
{
  "type": "ack",
  "action": "send",
  "data": {},
  "log_id": "..."
}
```

请求失败响应：

```json
{
  "type": "error",
  "action": "send",
  "error": {
    "code": "bad_request",
    "message": "..."
  },
  "log_id": "..."
}
```

服务端主动推送：

```json
{
  "type": "message",
  "data": {},
  "log_id": "..."
}
```

说明：

- WS 不是所有 action 都是强请求响应语义，部分操作只返回轻量 `ack`。
- `log_id` 是每条 WS 消息独立生成，不是连接级。
- WS 建连后服务端会注册 presence，并发布 `user.online` 事件更新在线时间。

### 用户类 action

| type | action | 说明 | data |
| --- | --- | --- | --- |
| `user` | `register` | 注册 | `{username, password, nickname}` |
| `user` | `login` | 登录 | `{username, password}` |
| `user` | `search` | 搜索用户 | `{keyword}` |
| `user` | `get` | 查看用户 | `{uid}` |
| `user` | `profile` | 获取自己的资料 | `{}` |
| `user` | `update_profile` | 修改资料 | `{nickname?, avatar?, sign?}` |
| `user` | `change_password` | 修改密码 | `{old_password, new_password}` |

说明：

- 实际使用中推荐通过 HTTP 登录拿 token 后再连接 WS。
- 当前 `/ws` 本身需要 token，因此 WS 内 `user.register` 和 `user.login` 仅保留为路由能力，不推荐作为主登录方式。

### 会话类 action

| type | action | 说明 | data |
| --- | --- | --- | --- |
| `conv` | `list` | 会话列表 | `{}` |
| `conv` | `create_private` | 创建或获取单聊 | `{uid}` |
| `conv` | `create_group` | 创建群聊 | `{name, avatar?, members: []}` |
| `conv` | `read_all` | 全部标记已读 | `{}` |
| `conv` | `update_info` | 修改群信息 | `{conv_id, name?, avatar?, member_limit?}` |
| `conv` | `pin` | 置顶或取消 | `{conv_id, pinned}` |
| `conv` | `mute` | 免打扰或取消 | `{conv_id, muted}` |
| `conv` | `read` | 标记已读 | `{conv_id, seq}` |
| `conv` | `members` | 成员列表 | `{conv_id}` |
| `conv` | `add_member` | 邀请成员 | `{conv_id, uid, role}` |
| `conv` | `remove_member` | 移除成员 | `{conv_id, uid}` |
| `conv` | `leave` | 退群 | `{conv_id}` |
| `conv` | `set_role` | 设置角色 | `{conv_id, uid, role}` |
| `conv` | `transfer_owner` | 转让群主 | `{conv_id, new_owner_id}` |
| `conv` | `dissolve` | 解散群聊 | `{conv_id}` |

### 消息类 action

| type | action | 说明 | data |
| --- | --- | --- | --- |
| `msg` | `send` | 发送消息 | `{conversation_id, msg_type, content, reply_to?, client_id}` |
| `msg` | `revoke` | 撤回消息 | `{conversation_id, message_id}` |
| `msg` | `typing` | 正在输入 | `{conversation_id}` |
| `msg` | `list` | 拉取历史消息 | `{conversation_id, before_seq?, limit?}` |
| `msg` | `search` | 搜索消息 | `{conversation_id, keyword, limit?}` |

发送消息成功响应 data：

```json
{
  "id": 1,
  "conversation_id": 1,
  "seq": 10,
  "sender_id": 1,
  "msg_type": 1,
  "content": "hello",
  "reply_to": 0,
  "client_id": "client-msg-id",
  "created_at": 1710000000
}
```

服务端新消息推送：

```json
{
  "type": "message",
  "action": "new",
  "data": {
    "MessageID": 1,
    "ConversationID": 1,
    "Seq": 10,
    "SenderID": 1,
    "MemberUIDs": [1, 2],
    "MsgType": 1,
    "Content": "hello",
    "ReplyTo": 0,
    "ClientID": "client-msg-id",
    "CreatedAt": 1710000000
  },
  "log_id": "..."
}
```

服务端消息撤回推送：

```json
{
  "type": "message",
  "action": "revoked",
  "data": {
    "conversation_id": 1,
    "message_id": 1,
    "seq": 10,
    "sender_id": 2,
    "operator_id": 1,
    "is_latest": true,
    "member_uids": [1, 2]
  },
  "log_id": "..."
}
```

服务端正在输入推送：

```json
{
  "type": "typing",
  "action": "indicator",
  "data": {
    "conversation_id": 1,
    "user_id": 2,
    "member_uids": [1]
  },
  "log_id": "..."
}
```

说明：

- 当前服务端推送直接使用 Go 事件结构体，字段为大驼峰。后续建议改为稳定的 snake_case DTO。
- 重复 `client_id` 的发送请求会幂等返回已有消息，不会重复写入，不会重复推送。
- 撤回消息会更新消息 `revoked=true`，发送者本人或群管理员/群主可撤回。
- 正在输入只实时推送，不落库。

### 好友类 action

| type | action | 说明 | data |
| --- | --- | --- | --- |
| `friend` | `send_request` | 发送好友申请 | `{to_uid, message?}` |
| `friend` | `list_incoming` | 收到的好友申请 | `{}` |
| `friend` | `list_outgoing` | 发出的好友申请 | `{}` |
| `friend` | `handle_request` | 处理申请 | `{req_id, accept}` |
| `friend` | `delete` | 删除好友 | `{friend_uid}` |
| `friend` | `list` | 好友列表 | `{}` |
| `friend` | `update_remark` | 修改备注 | `{friend_uid, remark}` |
| `friend` | `move_group` | 移动分组 | `{friend_uid, group_id}` |
| `friend` | `list_groups` | 分组列表 | `{}` |
| `friend` | `create_group` | 创建分组 | `{name}` |
| `friend` | `rename_group` | 重命名分组 | `{group_id, name}` |
| `friend` | `delete_group` | 删除分组 | `{group_id}` |
| `friend` | `sort_groups` | 分组排序 | `{groups: [{group_id, sort_order}]}` |

### 好友实时推送

收到好友申请：

```json
{
  "type": "friend",
  "action": "request",
  "data": {
    "request_id": 1,
    "from_uid": 1,
    "to_uid": 2,
    "message": "hello",
    "created_at": 1710000000
  },
  "log_id": "..."
}
```

好友申请被处理：

```json
{
  "type": "friend",
  "action": "accepted",
  "data": {
    "request_id": 1,
    "from_uid": 1,
    "to_uid": 2,
    "accepted": true
  },
  "log_id": "..."
}
```

说明：

- `action=accepted` 表示好友申请被同意。
- `action=rejected` 表示好友申请被拒绝。
- 用户重新登录后仍可通过 `friend.list_incoming`、`friend.list_outgoing` 或对应 HTTP 接口拉取好友申请列表。

### 群和会话实时推送

会话信息变更：

```json
{
  "type": "conversation",
  "action": "updated",
  "data": {
    "conversation_id": 1,
    "name": "new name",
    "avatar": "avatar",
    "member_uids": [1, 2]
  },
  "log_id": "..."
}
```

成员入群：

```json
{
  "type": "member",
  "action": "joined",
  "data": {
    "conversation_id": 1,
    "uid": 3,
    "role": 0,
    "operator_id": 1,
    "member_uids": [1, 2, 3]
  },
  "log_id": "..."
}
```

成员退群或被踢：

```json
{
  "type": "member",
  "action": "kicked",
  "data": {
    "conversation_id": 1,
    "uid": 3,
    "operator_id": 1,
    "member_uids": [1, 2, 3]
  },
  "log_id": "..."
}
```

群主转让：

```json
{
  "type": "member",
  "action": "owner_transferred",
  "data": {
    "conversation_id": 1,
    "old_owner_id": 1,
    "new_owner_id": 2,
    "member_uids": [1, 2]
  },
  "log_id": "..."
}
```

群解散：

```json
{
  "type": "conversation",
  "action": "group_dissolved",
  "data": {
    "conversation_id": 1,
    "operator_id": 1,
    "member_uids": [1, 2]
  },
  "log_id": "..."
}
```

### 同步类推送

当前规划在 WS 建连后服务端主动推送会话同步摘要：

```json
{
  "type": "sync",
  "action": "conversations",
  "data": [
    {
      "conversation_id": 1,
      "unread_count": 3,
      "last_msg_at": 1710000000,
      "is_pinned": false,
      "is_muted": false
    }
  ],
  "log_id": "..."
}
```

说明：

- 该推送用于前端感知未读数和最后消息时间。
- 当前代码还未自动发送 `sync.conversations`，可后续在 `GatewayActor.OnStart` 中实现。
- 具体消息仍由客户端按需调用 `msg.list` 或 HTTP 历史消息接口拉取。

### 暂未实现的 WS 能力

以下能力在旧设计中出现，但当前代码暂未实现：

| type/action | 说明 |
| --- | --- |
| `msg.edit` | 编辑消息 |
| `heartbeat` | 心跳与心跳回复 |
| `presence_change` | 上下线广播 |
| `read_receipt` | 已读回执广播 |

## 十、当前限制与后续补齐

- 当前仅实现图片上传，文件消息和视频消息后续再做。
- 前端删除会话当前不走后端接口，后端只提供群聊解散。
- 消息查询侧需要补会话成员权限校验。
- 好友列表、申请列表、成员列表暂未 join 展示昵称和头像。
- 会话列表当前未返回最后一条消息内容 `last_msg`。
- 会话同步推送 `sync.conversations` 仍是规划，还未在 `GatewayActor.OnStart` 自动下发。
- 禁言、封禁、退群后读边界、群解散软状态等生产级权限细节仍待补齐。
