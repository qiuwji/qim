# 会话 + 消息 + 群组 API

所有接口需 JWT 鉴权。

## 会话

| 方法 | 路径 | 说明 | 请求体 | 响应 data |
| --- | --- | --- | --- | --- |
| `GET` | `/api/conversations` | 会话列表 | - | `UserConvDTO[]` |
| `POST` | `/api/conversations/private` | 创建或获取单聊会话 | `{username}`，兼容旧 `{uid}` | `ConversationDTO` |
| `POST` | `/api/conversations/group` | 创建群聊 | `{name, avatar?, usernames: []}`，兼容旧 `{members: []}` | `ConversationDTO` |
| `PUT` | `/api/conversations/:id/pin` | 置顶或取消置顶 | `{pinned: true}` | `true` |
| `PUT` | `/api/conversations/:id/mute` | 免打扰或取消 | `{muted: true}` | `true` |
| `PUT` | `/api/conversations/:id/read` | 标记已读 | `{seq}` | `true` |
| `PUT` | `/api/conversations/read-all` | 全部标记已读 | - | `true` |

### UserConvDTO

```json
{
  "conversation_id": 1,
  "is_pinned": false,
  "is_muted": false,
  "unread_count": 3,
  "last_msg_at": 1710000000,
  "conv": {
    "id": 1, "type": 2, "name": "group", "avatar": "",
    "owner_id": 1, "member_count": 3, "member_limit": 500,
    "max_seq": 10, "created_at": 1710000000
  }
}
```

### ConversationDTO

```json
{
  "id": 1, "type": 2, "name": "group", "avatar": "",
  "owner_id": 1, "member_count": 3, "member_limit": 500,
  "max_seq": 10, "created_at": 1710000000
}
```

- `type`: `1=private`，`2=group`。前端应优先使用 `conv.type` 区分会话类型，不要通过成员数量推断。
- 当前后端不提供"删除会话"接口，前端"移除"属于本地隐藏视图语义。

## 消息

| 方法 | 路径 | 说明 | 请求体 | 响应 data |
| --- | --- | --- | --- | --- |
| `GET` | `/api/conversations/:id/messages?before_seq=0&limit=20` | 历史消息，按 seq 向前翻 | - | `MessageDTO[]` |
| `GET` | `/api/messages/search?keyword=xxx&conversation_id=1&limit=20` | 搜索消息 | - | `MessageDTO[]` |

兼容参数：`GET /api/conversations/:id/messages?seq=100` 等价于 `before_seq=100`。

### MessageDTO

```json
{
  "id": 1, "conversation_id": 1, "seq": 10, "sender_id": 1,
  "msg_type": 1, "content": "hello", "reply_to": 0,
  "revoked": false, "edited": false,
  "client_id": "client-msg-id", "created_at": 1710000000,
  "mention_uids": [2], "mention_all": false
}
```

- 发送消息走 WebSocket `type=msg action=send`。
- `client_id` 用于发送消息幂等，服务端按 `conversation_id + sender_id + client_id` 去重。
- `msg_type` 枚举：`1=text`，`2=image`，`5=system`，`6=call_record`（通话记录卡片，content 为 JSON）。

## 群组

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

### MemberDTO

```json
{
  "uid": 1, "role": 2, "last_read_seq": 10, "join_time": 1710000000
}
```

- `role`: `0=普通成员`，`1=管理员`，`2=群主`。
- 当前邀请入群一次只支持一个账号。
- 当前成员列表暂未 join 用户昵称、头像。
