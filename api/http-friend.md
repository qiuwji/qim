# 好友 API

所有接口需 JWT 鉴权。

## 接口

### 好友申请

| 方法 | 路径 | 说明 | 请求体 | 响应 data |
| --- | --- | --- | --- | --- |
| `POST` | `/api/friends/request` | 发送好友申请 | `{username, message?}`，兼容旧 `{to_uid, message?}` | `FriendRequestDTO` |
| `GET` | `/api/friends/requests/incoming` | 收到的好友申请 | - | `FriendRequestDTO[]` |
| `GET` | `/api/friends/requests/outgoing` | 发出的好友申请 | - | `FriendRequestDTO[]` |
| `PUT` | `/api/friends/requests/:req_id` | 同意或拒绝 | `{action: "accept"}` 或 `{action: "reject"}` | `true` |

### 好友管理

| 方法 | 路径 | 说明 | 请求体 | 响应 data |
| --- | --- | --- | --- | --- |
| `DELETE` | `/api/friends/:friend_uid` | 删除好友（逻辑删除，重新加好友可恢复） | - | `true` |
| `GET` | `/api/friends` | 好友列表 | - | `FriendDTO[]` |
| `PUT` | `/api/friends/:friend_uid/remark` | 修改备注 | `{remark}` | `true` |
| `PUT` | `/api/friends/:friend_uid/group` | 移动分组 | `{group_id}` | `true` |

### 好友分组

| 方法 | 路径 | 说明 | 请求体 | 响应 data |
| --- | --- | --- | --- | --- |
| `GET` | `/api/friend/groups` | 分组列表 | - | `FriendGroupDTO[]` |
| `POST` | `/api/friend/groups` | 创建分组 | `{name}` | `FriendGroupDTO` |
| `PUT` | `/api/friend/groups/:group_id` | 重命名 | `{name}` | `true` |
| `DELETE` | `/api/friend/groups/:group_id` | 删除分组（逻辑删除） | - | `true` |
| `PUT` | `/api/friend/groups/sort` | 排序 | `{groups: [{group_id, sort_order}]}` | `true` |

## DTO

### FriendRequestDTO

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

- `status`: `0=pending`，`1=accepted`，`2=rejected`

### FriendDTO

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

- `status`: `0=active`，`1=deleted`（逻辑删除，重新加好友时恢复为 0）

### FriendGroupDTO

```json
{
  "id": 1,
  "name": "default",
  "sort_order": 0
}
```

## 说明

- 好友列表和申请列表暂未 join 用户昵称、头像。
- 删除分组为逻辑删除，分组内好友不受影响。
- 当前分组列表暂未返回 `count`。
