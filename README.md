# kitakami_hibiki e-library

自托管的个人电子书库，对标 Google Play 图书。支持 **EPUB / PDF** 的上传、管理、在线阅读与阅读进度同步。单文件二进制部署，前端资源内嵌于后端，开箱即用。

> 当前版本由项目版本环境变量 `E_LIBRARY_VERSION` 指定（未设置时读取根目录 `VERSION` 文件）。当前：v0.2.0

## 功能特性

| 功能 | 说明 |
|------|------|
| 图书管理 | 上传、删除、批量删除、列表、关键字搜索、标签/状态筛选、排序、分页、重新解析 |
| 格式支持 | EPUB / PDF |
| 元数据解析 | **EPUB**：书名、作者、封面、ISBN、出版社、语言、描述、章节数；**PDF**：元数据与首页封面（pdfcpu） |
| 在线阅读 | **EPUB** 基于 ePub.js（单/双栏、目录、字号、深色模式）；**PDF** 基于 PDF.js（缩放、单页/双页/连续滚动、页码跳转） |
| 阅读进度 | 自动保存、断点续读（EPUB 按 CFI、PDF 按页码）、阅读完成页显式提交 |
| 阅读完成 | 完成页可标记"已读完"、重新阅读、上滑/方向键返回原尾页 |
| 封面 | 自动从 EPUB / PDF 提取并展示 |
| 标签 | 标签管理（增删改），图书可关联多个标签 |
| 运行时设置 | Web 设置页配置主题、语言、排序、字号、阅读器视图、存储目录，立即生效 |
| 统计 | 书架概览统计 |
| 软件更新 | 设置页检查 GitHub Releases、下载、安装（优雅停机 + 更新脚本替换重启），全部操作需用户确认，不做自动更新 |

> 存储层预留了可插拔的 `StorageDriver` 接口，目前仅实现本地文件系统驱动（百度网盘等后端规划中，见[开发路线](#开发路线)）。

## 技术栈

| 层 | 选型 |
|---|------|
| 后端 | Go 1.25 + Gin + GORM |
| 数据库 | SQLite（纯 Go 驱动 `glebarez/sqlite`，无需 CGO，启用 WAL 模式） |
| 前端 | Vue 3 + TypeScript + Element Plus + vue-i18n |
| 阅读器 | ePub.js（EPUB）、pdf.js（PDF） |
| PDF 处理 | pdfcpu（元数据 / 封面提取） |
| 部署 | 单文件二进制（内嵌前端）或 Docker |

## 项目结构

```
.
├── backend/                  # Go 后端
│   ├── cmd/server/           # 入口 main.go
│   ├── internal/
│   │   ├── config/           # 配置加载（application.yml + 环境变量）
│   │   ├── epub/             # EPUB 校验 / 元数据 / 封面 / XHTML 净化
│   │   ├── handler/          # HTTP 处理器
│   │   ├── middleware/       # CORS / 日志 / 请求体限制
│   │   ├── model/            # 数据模型
│   │   ├── repository/       # 数据库访问（GORM）
│   │   ├── service/          # 业务逻辑（图书、更新、PDF 辅助）
│   │   └── storage/          # 存储驱动接口 + 本地实现
│   ├── web/dist/             # 前端构建产物（go:embed 内嵌）
│   └── application.yml       # 服务配置
├── client/web/               # Vue 3 前端
├── docker/                   # Dockerfile.build / Dockerfile.download
├── docs/                     # 设计文档（数据迁移、软件更新）
├── build.sh                  # Linux / macOS 构建脚本
├── build.bat                 # Windows 构建脚本
└── VERSION                   # 版本号（构建时注入）
```

## 快速开始

### 环境要求

- Go 1.25+
- Node.js 22+

### 方式一：开发模式（前后端分离）

> `backend/web/dist` 仅跟踪一个 `.gitkeep` 占位，保证 `//go:embed` 在全新克隆时即可编译；前端构建产物不纳入版本管理。开发模式用 Vite 前端服务器（5173），后端（14325）仅提供 API。如需后端直接托管前端界面，先执行一次 `build.sh` / `build.bat`（会自动构建并拷贝 `client/web/dist` 到 `backend/web/dist`）。

```bash
# 终端 1：启动后端（默认监听 14325 端口）
cd backend
go run ./cmd/server
```

```bash
# 终端 2：启动前端开发服务器（监听 5173 端口，API 代理到后端）
cd client/web
npm install
npm run dev
```

访问 http://localhost:5173

### 方式二：一键构建（生产模式）

**Windows：**

```bash
build.bat
```

运行 `backend\kh-e-library.exe`，访问 http://localhost:14325

**Linux / macOS：**

```bash
chmod +x build.sh
./build.sh
```

解压生成的 `backend/kh-e-library-<版本>-<系统>-<架构>.tar.gz`，运行 `./kh-e-library`

构建脚本自动完成：前端构建 → 复制到后端 → 编译单文件二进制（注入版本号）。

### 方式三：Docker

```bash
# 从源码编译镜像（版本号自动取自根目录 VERSION 文件）
docker build -f docker/Dockerfile.build \
  --build-arg E_LIBRARY_VERSION="$(cat VERSION)" \
  -t e-library:latest .

# 运行容器（数据目录挂载）
docker run -d -p 14325:14325 -v ./data:/app/data --name e-library e-library:latest
```

```bash
# 或从 GitHub Release 下载预编译产物构建镜像
docker build -f docker/Dockerfile.download \
  --build-arg E_LIBRARY_VERSION="$(cat VERSION)" \
  -t e-library:download .
```

## 配置

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SERVER_HOST` | `0.0.0.0` | 监听地址 |
| `SERVER_PORT` | `14325` | 监听端口 |
| `CONFIG_PATH` | `application.yml` | 自定义配置文件路径 |
| `E_LIBRARY_VERSION` | 读取根目录 `VERSION` 文件 | 项目版本号（构建时注入二进制，可覆盖默认值） |

### application.yml

只含服务监听配置，修改后需重启生效：

```yaml
server:
  host: "0.0.0.0"
  port: 14325
```

### 数据目录

- 数据库：`./data/library.db`（路径固定，不可配置）
- 图书文件：`./data/books/{book_id}/`（默认路径，可在设置页修改 `storage.local.books_dir`）
- 运行时设置（主题、语言、排序、字号、视图模式等）通过 Web 设置页修改，立即生效

## API 概览

| 分组 | 端点 | 说明 |
|------|------|------|
| 健康 | `GET /health` | 健康检查 |
| 图书 | `GET /api/v1/books/list` · `/detail` · `/read` · `/download` · `/cover` | 列表、详情、读取、下载、封面 |
| 图书 | `POST /api/v1/books/create` · `/update` · `/reprocess` · `/delete` · `/batch_delete` | 上传、更新、重新解析、删除、批量删除 |
| 进度 | `GET /api/v1/books/progress` · `POST /api/v1/books/progress/save` | 读取 / 保存阅读进度 |
| 标签 | `GET /api/v1/tags/list` · `POST /api/v1/tags/create` · `/update` · `/delete` | 标签管理 |
| 图书标签 | `POST /api/v1/books/tags/add` · `/remove` | 图书-标签关联 |
| 设置 | `GET /api/v1/settings/list` · `POST /api/v1/settings/update` | 运行时设置 |
| 统计 | `GET /api/v1/stats/overview` | 书架概览 |
| 系统 | `GET /api/v1/system/status` | 版本与运行状态 |
| 更新 | `GET /api/v1/system/check-update` · `POST /api/v1/system/download-update` · `/install-update` | 检查 / 下载 / 安装更新 |

## 开发路线

- [x] Phase 0：项目脚手架、后端框架、数据库 Schema
- [x] Phase 1：EPUB 上传解析、书架展示、基础阅读器
- [x] Phase 2：阅读进度同步
- [x] 扩展：PDF 阅读支持、阅读完成页、软件更新、运行时设置
- [ ] Phase 3：其他存储后端集成（百度网盘等）
- [ ] Phase 4：移动端适配、PWA

## License

MIT
