# 用户 API

所有接口需 JWT 鉴权。

## 接口

| 方法 | 路径 | 说明 | 请求体 | 响应 data |
| --- | --- | --- | --- | --- |
| `GET` | `/api/user/profile` | 获取自己的资料 | - | `UserDTO` |
| `PUT` | `/api/user/profile` | 修改资料 | `{nickname?, avatar?, sign?}` | `true` |
| `PUT` | `/api/user/password` | 修改密码 | `{old_password, new_password}` | `true` |
| `GET` | `/api/users/search?keyword=xxx` | 搜索用户 | - | `UserDTO[]` |
| `GET` | `/api/users/:id` | 查看用户资料 | - | `UserDTO` |

## UserDTO

见 [http-auth.md](http-auth.md#userdto)。
