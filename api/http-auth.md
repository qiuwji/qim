# 认证 API

## 接口

| 方法 | 路径 | 说明 | 请求体 | 响应 data |
| --- | --- | --- | --- | --- |
| `POST` | `/api/auth/register` | 注册，`username` 只能是纯数字字符串且不可重复 | `{username, password, nickname}` | `UserDTO` |
| `POST` | `/api/auth/login` | 登录 | `{username, password}` | `{token, user}` |

## UserDTO

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

## 示例

### 注册

```json
// Request: POST /api/auth/register
{ "username": "10001", "password": "Abc123!@", "nickname": "Alice" }

// Response
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

### 登录

```json
// Request: POST /api/auth/login
{ "username": "10001", "password": "Abc123!@" }

// Response
{
  "code": "ok",
  "data": {
    "token": "<jwt>",
    "user": { "id": 1, "username": "10001", "nickname": "Alice", "avatar": "", "sign": "", "status": 0, "created_at": 1710000000, "last_online_at": 1710000000 }
  }
}
```
