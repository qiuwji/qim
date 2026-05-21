# WebSocket 协议

## 连接

```text
ws://host/ws?token=<jwt>
```

也可以在握手请求中使用：

```http
Authorization: Bearer <jwt>
```

## 通用格式

### 上行（客户端 → 服务端）

```json
{ "type": "msg", "action": "send", "data": {} }
```

### 下行（服务端 → 客户端）

请求成功：

```json
{ "type": "ack", "action": "send", "data": {}, "log_id": "..." }
```

请求失败：

```json
{
  "type": "error", "action": "send",
  "error": { "code": "bad_request", "message": "..." },
  "log_id": "..."
}
```

服务端主动推送：

```json
{ "type": "message", "data": {}, "log_id": "..." }
```

说明：

- WS 不是所有 action 都是强请求响应语义，部分操作只返回轻量 `ack`。
- `log_id` 是每条 WS 消息独立生成，不是连接级。
- WS 建连后服务端会注册 presence，并发布 `user.online` 事件更新在线时间。

## 用户类 action

| type | action | 说明 | data |
| --- | --- | --- | --- |
| `user` | `register` | 注册 | `{username, password, nickname}` |
| `user` | `login` | 登录 | `{username, password}` |
| `user` | `search` | 搜索用户 | `{keyword}` |
| `user` | `get` | 查看用户 | `{uid}` |
| `user` | `profile` | 获取自己的资料 | `{}` |
| `user` | `update_profile` | 修改资料 | `{nickname?, avatar?, sign?}` |
| `user` | `change_password` | 修改密码 | `{old_password, new_password}` |

说明：实际使用中推荐通过 HTTP 登录拿 token 后再连接 WS。当前 `/ws` 本身需要 token。

## 会话类 action

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

## 消息类 action

| type | action | 说明 | data |
| --- | --- | --- | --- |
| `msg` | `send` | 发送消息 | `{conversation_id, msg_type, content, reply_to?, client_id, mention_uids?, mention_all?}` |
| `msg` | `revoke` | 撤回消息 | `{conversation_id, message_id}` |
| `msg` | `typing` | 正在输入 | `{conversation_id}` |
| `msg` | `list` | 拉取历史消息 | `{conversation_id, before_seq?, limit?}` |
| `msg` | `search` | 搜索消息 | `{conversation_id, keyword, limit?}` |

发送消息成功响应 data：

```json
{
  "id": 1, "conversation_id": 1, "seq": 10, "sender_id": 1,
  "msg_type": 1, "content": "hello", "reply_to": 0,
  "client_id": "client-msg-id", "created_at": 1710000000,
  "mention_uids": [2], "mention_all": false
}
```

说明：

- 重复 `client_id` 的发送请求会幂等返回已有消息，不会重复写入、重复推送。
- 撤回消息会更新消息 `revoked=true`，发送者本人或群管理员/群主可撤回。
- 正在输入只实时推送，不落库。
- `msg_type` 枚举：`1=text`，`2=image`，`5=system`，`6=call_record`（通话记录，content 为 JSON）。

## 好友类 action

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

## 通话类 action

| type | action | 说明 | data |
| --- | --- | --- | --- |
| `call` | `initiate` | 发起通话 | `{callee_uid, call_type}` |
| `call` | `accept` | 接听 | `{call_id}` |
| `call` | `reject` | 拒绝 | `{call_id}` |
| `call` | `cancel` | 取消呼叫 | `{call_id}` |
| `call` | `end` | 挂断 | `{call_id}` |
| `call` | `offer` | WebRTC offer | `{call_id, sdp}` |
| `call` | `answer` | WebRTC answer | `{call_id, sdp}` |
| `call` | `ice` | ICE candidate | `{call_id, candidate, sdp_mid?, sdp_m_line_index?}` |

字段说明：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `call_type` | `1\|2` | 1=语音通话，2=视频通话 |
| `call_id` | `string` | 通话 ID，由服务端在 `initiate` 响应中返回 |
| `sdp` | `string` | SDP 描述文本 |
| `candidate` | `string` | ICE candidate 字符串 |

发起通话成功响应：

```json
{ "type": "ack", "action": "initiate", "data": { "call_id": "call_1700000000000_a3f8c2b1" } }
```

说明：

- 通话仅支持私聊（1对1），群聊不显示通话按钮。
- 同一用户同一时刻只能参与一通电话（忙线互斥）。
- 多设备同时来电，仅第一个接听生效。
- 呼叫超时时间为 30 秒。
- 信令（offer/answer/ice）走 Actor 直连，状态通知走 EventBus → PushHandler。
- NAT 穿透失败由前端提示，本期不做 TURN 中继。

## 实时推送

### 消息推送

新消息：

```json
{
  "type": "message", "action": "new",
  "data": {
    "MessageID": 1, "ConversationID": 1, "Seq": 10, "SenderID": 1,
    "MemberUIDs": [1, 2], "MsgType": 1, "Content": "hello",
    "ReplyTo": 0, "ClientID": "client-msg-id", "CreatedAt": 1710000000,
    "MentionUIDs": [2], "MentionAll": false
  }
}
```

通话记录（`MsgType=6`）消息的 `Content` 为 JSON 字符串：

```json
{"call_id":"call_xxx","call_type":1,"duration":120,"end_reason":"hangup","status":2}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `call_type` | `1\|2` | 1=语音通话，2=视频通话 |
| `duration` | `int` | 通话时长（秒），未接通为 0 |
| `end_reason` | `string` | `hangup`/`rejected`/`timeout`/`cancelled`/`disconnect` |
| `status` | `1\|2` | 1=未接通，2=已接通 |

通话记录消息 `SenderID` 固定为 `0`（系统消息）。

消息撤回：

```json
{
  "type": "message", "action": "revoked",
  "data": {
    "conversation_id": 1, "message_id": 1, "seq": 10,
    "sender_id": 2, "operator_id": 1, "is_latest": true,
    "member_uids": [1, 2]
  }
}
```

@消息通知（仅被 @者收到，穿透免打扰）：

```json
{ "type": "message", "action": "mention", "data": { "conversation_id": 1, "message_id": 123 } }
```

正在输入：

```json
{
  "type": "typing", "action": "indicator",
  "data": { "conversation_id": 1, "user_id": 2 }
}
```

### 好友实时推送

收到好友申请：

```json
{
  "type": "friend", "action": "request",
  "data": { "request_id": 1, "from_uid": 1, "to_uid": 2, "message": "hello", "created_at": 1710000000 }
}
```

好友申请被处理：

```json
{
  "type": "friend", "action": "accepted",
  "data": { "request_id": 1, "from_uid": 1, "to_uid": 2, "accepted": true }
}
```

- `action=rejected` 表示被拒绝。

### 群和会话实时推送

会话信息变更：

```json
{
  "type": "conversation", "action": "updated",
  "data": { "conversation_id": 1, "name": "new name", "avatar": "avatar", "member_uids": [1, 2] }
}
```

成员入群：

```json
{
  "type": "member", "action": "joined",
  "data": { "conversation_id": 1, "uid": 3, "role": 0, "operator_id": 1, "member_uids": [1, 2, 3] }
}
```

成员退群或被踢（`action=left` / `action=kicked`）：

```json
{
  "type": "member", "action": "kicked",
  "data": { "conversation_id": 1, "uid": 3, "operator_id": 1, "member_uids": [1, 2, 3] }
}
```

群主转让（`action=owner_transferred`）：

```json
{
  "type": "member", "action": "owner_transferred",
  "data": { "conversation_id": 1, "old_owner_id": 1, "new_owner_id": 2, "member_uids": [1, 2] }
}
```

群解散（`action=group_dissolved`）：

```json
{
  "type": "conversation", "action": "group_dissolved",
  "data": { "conversation_id": 1, "operator_id": 1, "member_uids": [1, 2] }
}
```

在线状态：

```json
{ "type": "presence", "action": "online", "data": { "uid": 2 } }
{ "type": "presence", "action": "offline", "data": { "uid": 2 } }
```

### 通话实时推送

被叫方来电通知（推送到被叫所有设备）：

```json
{
  "type": "call", "action": "incoming",
  "data": { "call_id": "call_xxx", "caller_uid": 100, "call_type": 1, "caller_nickname": "张三", "caller_avatar": "xxx" }
}
```

主叫方呼叫中确认（推送到主叫所有设备）：

```json
{ "type": "call", "action": "calling", "data": { "call_id": "call_xxx", "callee_uid": 200 } }
```

被叫接听后推送给主叫方：

```json
{ "type": "call", "action": "accepted", "data": { "call_id": "call_xxx" } }
```

被叫拒绝后推送给主叫方：

```json
{ "type": "call", "action": "rejected", "data": { "call_id": "call_xxx" } }
```

主叫取消后推送给被叫方：

```json
{ "type": "call", "action": "cancelled", "data": { "call_id": "call_xxx" } }
```

通话结束后推送给双方：

```json
{
  "type": "call", "action": "ended",
  "data": { "call_id": "call_xxx", "started_at": 1700000000, "duration": 120, "end_reason": "hangup" }
}
```

`end_reason` 枚举：`hangup` | `rejected` | `timeout` | `cancelled` | `disconnect`

呼叫超时推送给双方：

```json
{ "type": "call", "action": "timeout", "data": { "call_id": "call_xxx" } }
```

多设备互斥（推送到被叫方非接听设备）：

```json
{ "type": "call", "action": "answered_elsewhere", "data": { "call_id": "call_xxx" } }
```

WebRTC 信令推送（Actor 直连，非 EventBus 广播）：

```json
{ "type": "call", "action": "offer", "data": { "call_id": "call_xxx", "sdp": "..." } }
{ "type": "call", "action": "answer", "data": { "call_id": "call_xxx", "sdp": "..." } }
{ "type": "call", "action": "ice", "data": { "call_id": "call_xxx", "candidate": "...", "sdp_mid": "...", "sdp_m_line_index": 0 } }
```

### 同步类推送（规划中）

```json
{
  "type": "sync", "action": "conversations",
  "data": [{ "conversation_id": 1, "unread_count": 3, "last_msg_at": 1710000000, "is_pinned": false, "is_muted": false }]
}
```

说明：该推送用于前端感知未读数和最后消息时间，当前代码还未自动发送，可后续在 `GatewayActor.OnStart` 中实现。

## 暂未实现

| type/action | 说明 |
| --- | --- |
| `msg.edit` | 编辑消息 |
| `heartbeat` | 心跳与心跳回复 |
| `read_receipt` | 已读回执广播 |

## 当前限制与后续补齐

- 当前仅实现图片上传，文件消息和视频消息后续再做。
- 前端删除会话当前不走后端接口，后端只提供群聊解散。
- 消息查询侧需要补会话成员权限校验。
- 好友列表、申请列表、成员列表暂未 join 展示昵称和头像。
- 会话列表当前未返回最后一条消息内容 `last_msg`。
- 同步推送 `sync.conversations` 仍是规划。
- 禁言、封禁、退群后读边界等生产级权限细节仍待补齐。
