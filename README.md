# OTA-Center

[English](./README.md) | [简体中文](./README.zh-CN.md)

An OTA file and version management service for distributing firmware across multiple projects.

## Overview

- Manage OTA firmware files for multiple `appName` projects
- Query latest version metadata and specific version metadata
- Download firmware by version
- Upload `.bin` files from CI with an admin key
- Automatically calculate and store `sha256` for each uploaded file

## Version Format

- Fixed format: `v0.0-yyyyMMddHHmm`
- Example: `v1.2-202604051230`
- Comparison order: `major.minor`, then revision timestamp

## Docker Image

- Image: `ghcr.io/real-uangi/ota-center:latest`

Example:

```bash
docker run --rm -p 8765:8765 \
  -e PORT=8765 \
  -e OTA_DATA_DIR=/data \
  -e OTA_ADMIN_KEY=replace-with-your-key \
  -v $(pwd)/data:/data \
  ghcr.io/real-uangi/ota-center:latest
```

## Environment Variables

| Name | Default | Description |
| --- | --- | --- |
| `PORT` | `8765` | HTTP listening port |
| `OTA_DATA_DIR` | `data` | Directory for OTA files and index metadata |
| `OTA_ADMIN_KEY` | `admin12345` | Admin key for upload API; change this in production |

## API

### 1. Get latest version metadata

`GET /ota/{appName}/versions/latest`

Example response:

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

### 2. Get specific version metadata

`GET /ota/{appName}/versions/{version}`

### 3. Download a versioned firmware file

`GET /ota/{appName}/download/{version}`

Returns the `.bin` file as binary content.

### 4. Upload a firmware file

`POST /admin/ota/upload`

Requirements:

- Header: `X-OTA-Admin-Key: <your-key>`
- Content-Type: `multipart/form-data`
- Form fields: `app_name`, `version`, `file`

Example:

```bash
curl -X POST "http://localhost:8765/admin/ota/upload" \
  -H "X-OTA-Admin-Key: replace-with-your-key" \
  -F "app_name=demo-app" \
  -F "version=v1.2-202604051230" \
  -F "file=@firmware.bin"
```

Successful response:

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

## Release and GHCR

- Pushing a `v*` tag triggers GitHub Actions
- GoReleaser will:
  - run tests
  - build release archives
  - generate `checksums.txt`
  - publish a GitHub Release
  - push container images to GHCR

## Local Development

```bash
go test ./...
go run .
```
