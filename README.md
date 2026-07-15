# kitakami_hibiki e-library

对标 Google Play 图书的个人网络云端书库。书籍可存储于本地或百度网盘。

## 概述

自托管的 EPUB 电子书管理平台。通过 Web 界面管理藏书、在线阅读，存储后端可自由选择本地文件系统或百度网盘，数据主权归用户所有。

## 技术栈

| 层级 | 选型 |
|------|------|
| 后端 | Go + Gin |
| 前端 | Vue 3 + TypeScript + Element Plus |
| 数据库 | SQLite (via GORM) |
| 存储 | 本地文件系统 / 百度网盘 Open API (可插拔) |
| 阅读器 | 基于 Web 的 EPUB.js |
| 部署 | 单文件二进制或 Docker |

## 架构

`
+----------------------------+
|        Web Browser          |
|  (书架 / 阅读器 / 管理面板)  |
+-------------+--------------+
              | HTTP
+-------------v--------------+
|         Go Backend          |
|  +---------+ +------+ +--+ |
|  | 管理 API | | 阅读  | |存| |
|  |         | | 服务  | |抽| |
|  +---------+ +------+ +-+ |
+----------------------------+---+
              |                |
    +---------+--------+------+------+
    |                  |             |
+---v----+      +-----v----+  +-----v----+
| 本地文件系统 |   | 百度网盘  |  | 其他(预留) |
+----------+      +----------+  +----------+
`

## 功能

| 功能 | 说明 |
|------|------|
| 书籍管理 | 上传、删除、列表展示 |
| EPUB 阅读 | 基于浏览器的阅读器，支持书签、笔记、进度同步 |
| 书架展示 | 网格/列表视图，搜索、筛选、排序 |
| 存储后端 | 本地文件系统 / 百度网盘可切换 |

## 快速开始

### 本地编译

`ash
VERSION=v0.0.1
cd backend
go build -ldflags="-s -w -X main.Version=" -o e-library ./cmd/server
./e-library
`

### Docker 编译部署

`ash
docker build -f docker/Dockerfile.build \
  --build-arg E_LIBRARY_VERSION=v0.0.1 \
  -t e-library:build .
`

### Docker 下载部署

`ash
docker build -f docker/Dockerfile.download \
  --build-arg E_LIBRARY_VERSION=v0.0.1 \
  -t e-library:download .
`

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| SERVER_HOST | 0.0.0.0 | 监听地址 |
| SERVER_PORT | 14325 | 监听端口 |
| DB_PATH | ~/.kh/e-library/library.db | SQLite 数据库路径 |
| STORAGE_DRIVER | local | 存储驱动 (local / baidu) |
| STORAGE_LOCAL_BOOKS_DIR | ~/.kh/e-library/books | 本地书籍存储目录 |

## 开发路线

- [x] Phase 0：项目脚手架，后端基础框架，数据库 schema
- [ ] Phase 1：本地存储后端，EPUB 上传解析，书架展示，基础阅读器
- [ ] Phase 2：阅读进度同步，书签、笔记
- [ ] Phase 3：百度网盘存储后端集成
- [ ] Phase 4：批量导入、Calibre 集成、OPDS
- [ ] Phase 5：移动端适配、PWA

## License

MIT