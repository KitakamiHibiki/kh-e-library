# kitakami_hibiki e-library

自托管的 EPUB 个人网络书库，对标 Google Play 图书。支持本地文件系统与百度网盘双存储后端，可在运行时热切换。

## 功能

| 功能 | 说明 |
|------|------|
| EPUB 管理 | 上传、删除、列表、搜索、分页展示 |
| 元数据解析 | 上传时自动提取书名、作者、封面、ISBN、出版社、语言、描述、章节数 |
| 在线阅读 | 基于 ePub.js 的浏览器阅读器 |
| 阅读进度 | 自动保存同步，断点续读 |
| 书签 | 添加、删除、跳转阅读 |
| 封面 | 自动从 EPUB 提取并展示 |
| 存储后端 | 本地文件系统 / 百度网盘可切换，运行时热加载生效 |
| 运行时设置 | Web 页面配置界面主题、排序、字号、存储驱动 |

## 技术栈

| 层 | 选型 |
|---|------|
| 后端 | Go + Gin + GORM |
| 前端 | Vue 3 + TypeScript + Element Plus |
| 数据库 | SQLite |
| 存储 | 本地文件系统 / 百度网盘 Open API（可插拔 Driver 接口） |
| 阅读器 | ePub.js |
| 部署 | 单文件二进制 或 Docker |

## 架构

```
+---------------------------+
|      Web Browser           |
|  (书架 / 阅读器 / 设置)    |
+------------+--------------+
             | HTTP
+------------v--------------+
|        Go Backend          |
|  +--------+ +------+ +--+ |
|  | 管理 API| | 阅读  | |存| |
|  |        | | 服务  | |抽| |
|  +--------+ +------+ +-+ |
+---------------------------+---+
             |                |
   +---------+--------+------+------+
   |                  |             |
+--v----+      +-----v----+  +-----v----+
| 本地文件系统 |   | 百度网盘  |  | 其他(预留) |
+---------+      +----------+  +----------+
```

## 快速开始

### 环境要求

- Go 1.25+
- Node.js 22+
	
### 方式一：开发模式（前后端分离，推荐开发时使用）

```bash
# 终端 1：启动后端（监听 14325 端口）
cd backend
go run ./cmd/server
```

```bash
# 终端 2：启动前端开发服务器（监听 5173 端口）
cd client/web
npm install
npm run dev
```

前端开发服务器自带热更新，API 请求自动代理到后端 `localhost:14325`。访问 http://localhost:5173

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
解压生成的 `kh-e-library-*.tar.gz`，运行 `./kh-e-library`

构建脚本会自动完成：前端构建 → 复制到后端 → 编译二进制。

### 方式三：Docker

```bash
# 从源码编译镜像
docker build -f docker/Dockerfile.build \
  --build-arg E_LIBRARY_VERSION=v0.0.3 \
  -t e-library:latest .

# 运行容器
docker run -d -p 14325:14325 -v ./data:/app/data --name e-library e-library:latest
```

```bash
# 或从 GitHub Release 下载预编译镜像
docker build -f docker/Dockerfile.download \
  --build-arg E_LIBRARY_VERSION=v0.0.3 \
  -t e-library:download .
```

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| SERVER_HOST | 0.0.0.0 | 监听地址 |
| SERVER_PORT | 14325 | 监听端口 |
| CONFIG_PATH | application.yml | 自定义配置文件路径 |

### 配置说明

- application.yml 只含服务端口和监听地址，修改后需重启
- 数据库路径固定在 ./data/library.db，不可配置
- 运行时设置（主题、排序、字号、存储后端）通过 Web 设置页修改，立即生效

## 开发路线

- [x] Phase 0：项目脚手架、后端框架、数据库 schema
- [x] Phase 1：EPUB 上传解析、书架展示、基础阅读器
- [x] Phase 2：阅读进度同步、书签
- [ ] Phase 3：百度网盘存储后端集成
- [ ] Phase 4：移动端适配、PWA

## License

MIT
