# QIM API 文档

## 目录

| 分类 | 文件 | 说明 |
|------|------|------|
| 通用 | [README.md](README.md) | 鉴权、响应格式、错误码汇总 |
| HTTP | [http-auth.md](http-auth.md) | 注册、登录 |
| HTTP | [http-user.md](http-user.md) | 用户资料、搜索 |
| HTTP | [http-friend.md](http-friend.md) | 好友申请、好友关系、好友分组 |
| HTTP | [http-conversation.md](http-conversation.md) | 会话列表、创建、群管理、消息查询 |
| HTTP | [http-file.md](http-file.md) | 图片上传 |
| WS | [ws.md](ws.md) | WebSocket 连接、上行/下行格式、各域 action、实时推送事件 |

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

### 错误码

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
| `conversation.mention_limit_exceeded` | @人数超过 50 |
| `conversation.mention_all_forbidden` | 非管理员 @全体成员 |
| `friend.not_your_request` | 不能处理不属于自己的好友申请 |
| `friend.already_friends` | 已经是好友，不能重复发送申请 |
| `friend.pending_request` | 已存在待处理的好友申请 |
| `call.invalid_type` | 无效的通话类型（必须为 1 或 2） |
| `call.self_busy` | 你正在通话中 |
| `call.offline` | 对方不在线 |
| `call.busy` | 对方忙线 |
| `call.self_call` | 不能给自己打电话 |
| `call.not_found` | 通话不存在 |
| `call.already_answered` | 已在其他设备接听 |
| `call.forbidden` | 无权操作此通话 |
| `call.invalid_state` | 当前通话状态不允许此操作 |
