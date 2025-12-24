# BaiduPCS-Go HTTP API

> 本项目是 [BaiduPCS-Go](https://github.com/qjfoidnh/BaiduPCS-Go) 的二次开发版本，旨在将其核心功能封装为 **RESTful HTTP API**，使其能够轻松集成到其他自动化流程（如 n8n, Web 界面, 脚本等）中。

![Swagger UI](https://img.shields.io/badge/API-Swagger-green) ![Platform](https://img.shields.io/badge/Platform-Linux%20%7C%20Windows%20%7C%20macOS-blue)

## 🚀 项目简介

原版 BaiduPCS-Go 是一个强大的百度网盘命令行客户端。本项目在其基础上增加了 Web Server 模块，提供了一套完整的 HTTP 接口，包括：

*   **文件管理**: 列出文件、创建目录、移动、复制、删除、搜索、重命名
*   **上传下载**: 文件上传、下载、获取直链
*   **账号管理**: 登录 (BDUSS/Cookie)、切换账号、查看配额
*   **离线下载**: 添加任务、查询进度、取消任务
*   **回收站**: 查看、恢复、清空
*   **分享管理**: 创建分享、列出分享、取消分享、**转存分享链接**

所有接口均提供 Swagger 在线文档。

## 🛠️ 快速开始

### 1. 编译构建

本项目基于 Go 语言开发，支持跨平台编译。

**依赖**: Go 1.18+

```bash
# 克隆项目
git clone https://github.com/YourUsername/BaiduPCS-Go-HTTP.git
cd BaiduPCS-Go-HTTP

# 编译 (根据你的系统选择)
# Windows
go build -o BaiduPCS-Go.exe

# Linux / macOS
go build -o BaiduPCS-Go
```

### 2. 启动服务

```bash
# 启动 API 服务，默认监听 5299 端口
# --verbose 参数用于开启详细日志 (推荐)
./BaiduPCS-Go server --verbose -p 5299
```

启动后，访问 **`http://localhost:5299/swagger/index.html`** 查看完整的 API 文档和调试接口。

---

## 💻 各平台使用指南

### Linux (推荐用于服务器/NAS)

通常用于部署在无头服务器上，作为后台服务运行。

1.  **编译**:
    ```bash
    export GOOS=linux
    export GOARCH=amd64
    go build -o BaiduPCS-Go
    chmod +x BaiduPCS-Go
    ```

2.  **运行**:
    ```bash
    # 后台运行，日志输出到 server.log
    nohup ./BaiduPCS-Go server --verbose -p 5299 > server.log 2>&1 &
    
    # 查看日志
    tail -f server.log
    ```

3.  **注意**:
    *   如果需要外网或局域网访问，请确保防火墙放行了 `5299` 端口。
    *   登录建议使用 API 接口 (`POST /api/auth/login`) 配合 BDUSS 进行登录。

### Windows

适合个人开发者或本地自动化任务。

1.  **编译**:
    在 PowerShell 中执行:
    ```powershell
    $Env:GOOS = "windows"; $Env:GOARCH = "amd64"; go build -o BaiduPCS-Go.exe
    ```

2.  **运行**:
    *   **方式 A (命令行)**: 打开 PowerShell，运行 `.\BaiduPCS-Go.exe server --verbose -p 5299`
    *   **方式 B (登录)**: 你也可以先使用命令行登录 `.\BaiduPCS-Go.exe login`，然后再启动 server，这样 API 会自动使用登录好的账号。

### macOS

1.  **编译**:
    ```bash
    export GOOS=darwin
    export GOARCH=amd64 # Intel core
    # export GOARCH=arm64 # Apple Silicon (M1/M2/M3)
    go build -o BaiduPCS-Go
    chmod +x BaiduPCS-Go
    ```

2.  **运行**:
    ```bash
    ./BaiduPCS-Go server --verbose -p 5299
    ```

---

## 🔑 账号登录

提供三种登录方式，满足不同使用场景：

### 方式 1: 扫码登录 (推荐 ⭐)

最简单安全的登录方式，无需手动获取 BDUSS：

```bash
# 运行扫码登录命令
./BaiduPCS-Go qrlogin

# 可选：自定义超时时间（秒）
./BaiduPCS-Go qrlogin --timeout 600
```

程序会在终端显示二维码，使用 **百度网盘 APP** 扫描即可完成登录。

### 方式 2: 通过 API 登录 (推荐远程部署使用)

调用 `/api/auth/login` 接口：

```bash
curl -X POST http://localhost:5299/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "bduss": "你的BDUSS......"
  }'
```

**扫码登录 API：**

```bash
# 1. 获取二维码
curl -X POST http://localhost:5299/api/auth/qrcode \
  -H "Content-Type: application/json" \
  -d '{"include_ascii": true}'

# 2. 查询扫码状态
curl "http://localhost:5299/api/auth/qrcode/status?sign=xxx"

# 3. 完成登录
curl -X POST http://localhost:5299/api/auth/qrcode/login \
  -H "Content-Type: application/json" \
  -d '{"sign": "xxx", "temp_bduss": "yyy"}'
```

> **如何获取 BDUSS?**
> 在浏览器登录百度网盘，打开开发者工具 (F12) -> Application -> Cookies，找到 `BDUSS` 的值。

### 方式 3: BDUSS/Cookies 登录

如果你已有 BDUSS 或完整 Cookies：

```bash
# 使用 BDUSS + STOKEN
./BaiduPCS-Go login -bduss=xxx -stoken=yyy

# 使用 Cookies
./BaiduPCS-Go login -cookies="BDUSS=xxx; STOKEN=yyy; ..."
```

---

## 🔒 隐私说明

*   **数据安全**: 本项目是一个开源的客户端工具，**不会**上传你的任何账号信息到第三方服务器。所有交互仅发生在你的服务器和百度网盘官方服务器之间。
*   **配置文件**: 登录凭据 (BDUSS/Cookies) 存储在用户目录下的配置文件中 (如 `~/.config/BaiduPCS-Go/pcs_config.json`)，**不包含**在项目源码中，因此推送到 Git 仓库是安全的。

## 📜 历史文档

原版 BaiduPCS-Go 的说明文档已归档至 [docs/README_legacy.md](docs/README_legacy.md)。
