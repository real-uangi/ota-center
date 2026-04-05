# OTA-Center

[English](./README.md) | [简体中文](./README.zh-CN.md)

统一 OTA 文件与版本管理中心，面向多项目固件分发场景。

## 功能概览

- 统一管理多个 `appName` 的 OTA 固件文件
- 提供“最新版本元数据”与“指定版本元数据”查询接口
- 提供指定版本文件下载接口
- 支持 CI 通过管理 Key 上传 `.bin` 文件
- 自动计算并保存每个版本文件的 `sha256`

## 版本规则

- 版本格式固定为 `v0.0-yyyyMMddHHmm`
- 示例：`v1.2-202604051230`
- 比较规则：先比较 `major.minor`，再比较后缀时间

## Docker 镜像

- 镜像地址：`ghcr.io/real-uangi/ota-center:latest`

运行示例：

```bash
docker run --rm -p 8765:8765 \
  -e PORT=8765 \
  -e OTA_DATA_DIR=/data \
  -e OTA_ADMIN_KEY=replace-with-your-key \
  -v $(pwd)/data:/data \
  ghcr.io/real-uangi/ota-center:latest
```

## 环境变量

| 变量名 | 默认值 | 说明 |
| --- | --- | --- |
| `PORT` | `8765` | HTTP 服务监听端口 |
| `OTA_DATA_DIR` | `data` | OTA 文件与索引目录 |
| `OTA_ADMIN_KEY` | `admin12345` | 上传接口管理 Key，生产环境请务必修改 |

## API

### 1. 获取最新版本元数据

`GET /ota/{appName}/versions/latest`

响应示例：

```json
{
  "app_name": "demo-app",
  "version": "v1.2-202604051230",
  "file_name": "v1.2-202604051230.bin",
  "file_size": 123456,
  "sha256": "7b0a5b...",
  "download_url": "http://localhost:8765/ota/demo-app/download/v1.2-202604051230",
  "uploaded_at": "2026-04-05T12:30:00Z"
}
```

### 2. 获取指定版本元数据

`GET /ota/{appName}/versions/{version}`

### 3. 下载指定版本文件

`GET /ota/{appName}/download/{version}`

返回 `.bin` 二进制文件。

### 4. 上传 OTA 文件

`POST /admin/ota/upload`

请求要求：

- Header: `X-OTA-Admin-Key: <your-key>`
- Content-Type: `multipart/form-data`
- Form 字段：`app_name`、`version`、`file`

`curl` 示例：

```bash
curl -X POST "http://localhost:8765/admin/ota/upload" \
  -H "X-OTA-Admin-Key: replace-with-your-key" \
  -F "app_name=demo-app" \
  -F "version=v1.2-202604051230" \
  -F "file=@firmware.bin"
```

成功响应示例：

```json
{
  "app_name": "demo-app",
  "version": "v1.2-202604051230",
  "file_name": "v1.2-202604051230.bin",
  "file_size": 123456,
  "sha256": "7b0a5b...",
  "uploaded_at": "2026-04-05T12:30:00Z"
}
```

## GitHub Release 与 GHCR

- 推送 `v*` tag 时会触发 GitHub Actions
- 使用 GoReleaser 自动：
  - 运行测试
  - 生成 Release 归档
  - 生成 `checksums.txt`
  - 发布 GitHub Release
  - 推送 GHCR 镜像

## 本地开发

```bash
go test ./...
go run .
```
