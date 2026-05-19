# 文件上传 API

需 JWT 鉴权。

## 接口

| 方法 | 路径 | 说明 | 请求体 | 响应 data |
| --- | --- | --- | --- | --- |
| `POST` | `/api/files/upload` | 上传图片 | `multipart/form-data`，字段名 `file` | `{url, filename, size, mime_type}` |

## 说明

- 当前仅支持图片上传，文件、视频消息后续再做。
- 单图大小限制为 10MB。
- 非图片 MIME 类型会返回 `bad_request`。
- 文件保存到 `data/uploads/images`，通过 `/uploads/images/<filename>` 访问。
