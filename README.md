# GoProxy

> **智能代理池系统** — 基于 Go 的轻量级、低资源消耗、自适应的代理池服务

[![Docker Hub](https://img.shields.io/docker/v/isboyjc/goproxy?label=Docker%20Hub&logo=docker)](https://hub.docker.com/r/isboyjc/goproxy)
[![GitHub Container Registry](https://img.shields.io/badge/GHCR-latest-blue?logo=github)](https://github.com/isboyjc/GoProxy/pkgs/container/goproxy)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://go.dev/)

GoProxy 从多个公开代理源自动抓取 HTTP/SOCKS5 代理，通过严格验证（出口 IP + 位置 + 延迟）后加入智能代理池，对外提供 **HTTP 和 SOCKS5 双协议**代理服务。系统采用质量分级、智能补充、自动优化等机制，确保代理池始终保持高质量和稳定性。

**GitHub**：[github.com/isboyjc/GoProxy](https://github.com/isboyjc/GoProxy)

![](https://cdn.isboyjc.com/img/Xnip2026-03-29_03-35-06.png)

## 📋 快速导航

- [✨ 核心特性](#-核心特性) - 智能池子、按需抓取、健康管理、双协议支持
- [🔌 HTTP vs SOCKS5 协议对比](#-http-vs-socks5-协议对比) - 协议区别、使用建议
- [🔒 Session-Sticky 会话代理](#-session-sticky-会话代理) - 独占代理租约、IP 一致性保障
- [🚀 快速开始](#-快速开始) - 本地运行、端口说明、代理使用
- [🐳 Docker 部署](#-docker-部署) - 容器化部署、环境配置、安全建议
- [⚙️ 配置说明](#️-配置说明) - 完整参数详解、性能优化
- [🛠️ 开发与调试](#️-开发与调试) - 日志查看、数据库操作
- [🧪 测试代理服务](#-测试代理服务) - 测试脚本、持续测试、认证测试
- [❓ 常见问题](#-常见问题) - SOCKS5 使用、浏览器配置、故障排查

## ✨ 核心特性

### 🎯 智能池子机制
- **固定容量管理**：可配置池子大小和 HTTP/SOCKS5 协议比例（默认 3:7）
- **质量分级**：S/A/B/C 四级评分（基于延迟），智能选择高质量代理
- **动态状态感知**：Healthy → Warning → Critical → Emergency 四级状态自适应
- **严格准入标准**：必须通过出口 IP、地理位置、延迟三重验证才可入池
- **HTTPS 可用性验证**：HTTP 协议代理入池前额外验证 HTTPS CONNECT 隧道能力，随机访问真实 HTTPS 网站确认可用（失败自动换站重试），确保入池的 HTTP 代理都能正常访问 HTTPS 网站
- **智能替换**：新代理必须显著优于现有代理（默认快 30%）才触发替换

### 🚀 按需抓取
- **源分组策略**：快更新源（5-30min）用于紧急补充，慢更新源（每天）用于优化轮换
- **断路器保护**：连续失败的源自动降级/禁用，冷却后恢复
- **协议并发验证**：抓取到的候选代理按协议分组，SOCKS5 和 HTTP 各自并发验证入池。SOCKS5 无额外检测，天然更快优先填充；HTTP 带 HTTPS CONNECT 检测较慢但不阻塞 SOCKS5 入池
- **多模式抓取**：
  - **Emergency**：单协议缺失或池子 <10%，使用所有可用源
  - **Refill**：池子 <80%，使用快更新源
  - **Optimize**：池子健康时，随机抽取少量慢源优化替换

### 🏥 分层健康管理
- **轻量批次检查**：每次仅检查 20 个代理，避免资源浪费
- **智能跳过 S 级**：池子健康时跳过 S 级代理检查
- **定时优化轮换**：健康状态下，定期抓取优质代理替换池中延迟高的

### 🔄 智能重试机制
- **自动故障切换**：代理请求失败时立即切换到另一个代理重试（最多 3 次）
- **失败即删除**：连接失败或请求超时的代理立即从池子中移除
- **用户无感知**：自动重试在服务端完成，用户只会收到成功响应或最终失败提示
- **防重复尝试**：已尝试过的失败代理不会在同一请求中再次使用

### 🔒 Session-Sticky 独占代理
- **独占租约**：通过 REST API 分配独占代理会话，保证同一 session 始终使用同一出口 IP
- **双协议接入**：SOCKS5（端口 7781）和 HTTP（端口 7782），通过 session_id 作为认证用户名路由
- **智能选择**：支持按协议、评分等级、国家、冷却时间等条件筛选代理
- **安全机制**：X-API-Key 认证、冷却防滥用、日使用量上限、风控触发降级
- **租约保护**：被租用的代理不会被健康检查删除、优化替换或分配给普通端口用户
- **会话生命周期**：acquire → 使用 → release/rotate/expire，支持幂等操作和优雅停机

### 🚪 多端口多协议支持
- **双协议支持**：同时提供 HTTP 和 SOCKS5 协议服务，满足不同应用需求
- **双模式策略**：每种协议都提供随机轮换和最低延迟两种模式
- **4 个服务端口**：
  - `7777` - HTTP 随机轮换（IP 多样性，适合爬虫）
  - `7776` - HTTP 最低延迟（稳定连接，适合流媒体）
  - `7779` - SOCKS5 随机轮换（IP 多样性，适合浏览器/游戏）
  - `7780` - SOCKS5 最低延迟（稳定连接，适合固定应用）
- **2 个 Session-Sticky 端口**：
  - `7781` - Session SOCKS5（独占代理，IP 一致性）
  - `7782` - Session HTTP（独占代理，IP 一致性）
- **自动切换**：所有端口都支持失败自动重试，智能切换备用代理
- **共享池子**：四个端口使用同一个代理池，统一管理和优化
- **可选认证**：支持 Basic Auth（HTTP）和用户名/密码认证（SOCKS5），对外开放时可启用

### 🎨 黑客风格 WebUI
- **Matrix 美学**：荧光绿 + 纯黑背景，CRT 扫描线效果，JetBrains Mono 等宽字体
- **双角色权限**：访客模式（只读）+ 管理员模式（完全控制），可安全公网开放
- **实时仪表盘**：池子状态、质量分布可视化、协议统计，带荧光光晕效果
- **完整配置界面**：池子容量、延迟标准、验证参数、优化策略均可在线调整（管理员）
- **代理注册表**：详细展示地址、出口 IP、位置、延迟、质量等级、使用统计
- **中英文切换**：支持中文/英文界面切换，默认中文
- **交互优化**：点击地址复制、单个代理刷新、实时日志倒计时

### 📊 适用场景
- **Web 开发测试**：HTTP 代理测试 API、爬虫开发、数据采集
- **浏览器代理**：SOCKS5 协议配置浏览器代理，访问受限网站
- **命令行工具**：curl、wget、git 等工具使用 HTTP 代理
- **应用代理**：需要 SOCKS5 协议的应用（SSH、游戏、聊天工具）
- **小型 VPS**：低资源消耗（固定池子 + 按需抓取 + 限流查询）
- **稳定需求**：自动剔除失败代理，始终保持健康池子
- **质量优先**：S/A 级代理优先使用，自动优化延迟

### 🔌 HTTP vs SOCKS5 协议对比

| 特性 | HTTP 代理 | SOCKS5 代理 |
|------|----------|-------------|
| **协议支持** | 仅 HTTP/HTTPS | 所有 TCP 协议（HTTP/HTTPS/SSH/FTP/游戏等） |
| **工作层级** | 应用层（Layer 7） | 会话层（Layer 5） |
| **浏览器支持** | ✅ 良好 | ✅ 更好（无协议限制） |
| **命令行工具** | ✅ curl/wget 原生支持 | ⚠️ 部分工具需要额外配置 |
| **非 HTTP 应用** | ❌ 不支持 | ✅ 完全支持（SSH/游戏/聊天） |
| **UDP 支持** | ❌ 不支持 | ✅ 支持（SOCKS5 协议特性） |
| **性能开销** | 较低 | 稍高 |
| **认证方式** | Basic Auth | 用户名/密码 |

**快速选择建议**：
- 🌐 **HTTP 代理**（端口 7776/7777）：适合 Web 开发、API 测试、爬虫、数据采集
- 🔒 **SOCKS5 代理**（端口 7779/7780）：适合浏览器、SSH 隧道、游戏、聊天应用、需要完整协议支持的场景

**架构设计**：
- HTTP 代理服务：可使用池中的 HTTP 或 SOCKS5 上游代理
- SOCKS5 代理服务：**仅使用 SOCKS5 上游代理**（因为许多免费 HTTP 代理不支持 HTTPS CONNECT 方法）

### 📝 扩展文档

- [地理过滤配置指南](GEO_FILTER.md) - 国家代码、使用场景、测试方法
- [数据目录说明](DATA_DIRECTORY.md) - 数据库、配置文件、备份恢复
- [测试脚本使用](test/README.md) - HTTP + SOCKS5 测试脚本详细说明
- [架构设计文档](POOL_DESIGN.md) - 完整的系统设计和实现细节

## 🔒 Session-Sticky 会话代理

Session-Sticky 模块提供**独占代理租约**功能，适用于需要在整个流程中保持同一出口 IP 的场景（如注册流程、多步骤表单提交、账号操作等）。

### 工作原理

```text
客户端 → POST /api/session/acquire → 获得 session_id + proxy_addr
           ↓
客户端 → 使用 proxy_addr 发起请求（session_id 作为认证用户名）
           ↓                         ↑ 始终同一出口 IP
客户端 → POST /api/session/release → 释放代理，进入冷却期
```

### 端口说明

| 端口 | 协议 | 说明 |
|------|------|------|
| **7781** | SOCKS5 | Session-Sticky SOCKS5 代理（用户名 = session_id） |
| **7782** | HTTP | Session-Sticky HTTP 代理（Proxy-Authorization 用户名 = session_id） |

### REST API

所有 API 需要 `X-API-Key` 请求头认证（通过 `SESSION_API_KEY` 环境变量配置）。

#### 分配代理会话

```bash
curl -X POST http://localhost:7778/api/session/acquire \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "task_id": "reg-task-001",
    "ttl": 600,
    "protocol": "socks5",
    "min_grade": "B",
    "country": "US"
  }'
```

**请求参数**：

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `task_id` | string | 空 | 任务标识（相同 task_id 幂等返回同一会话） |
| `ttl` | int | 600 | 会话有效期（秒） |
| `protocol` | string | `socks5` | 代理协议（`socks5` / `http`） |
| `min_grade` | string | `B` | 最低质量等级（`S`/`A`/`B`/`C`） |
| `country` | string | 空 | 指定出口国家代码（如 `US`） |
| `cooldown_minutes` | int | 30 | 释放后冷却时间（分钟） |
| `max_daily_uses` | int | 3 | 单 IP 日使用上限 |

**响应示例**：

```json
{
  "session_id": "sess-a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4",
  "proxy_addr": "socks5://sess-a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4:x@127.0.0.1:7781",
  "upstream_ip": "203.0.113.45",
  "country_code": "US",
  "timezone": "America/New_York",
  "grade": "A",
  "usage_today": 1,
  "expires_at": 1711876800
}
```

#### 使用会话代理

```bash
# SOCKS5 方式（使用返回的 proxy_addr）
curl --socks5 "sess-xxx:x@127.0.0.1:7781" https://httpbin.org/ip

# HTTP 方式
curl -x "http://sess-xxx:x@127.0.0.1:7782" https://httpbin.org/ip

# Python 示例
import requests
proxies = {"https": "socks5://sess-xxx:x@127.0.0.1:7781"}
response = requests.get("https://httpbin.org/ip", proxies=proxies)
```

#### 释放会话

```bash
curl -X POST http://localhost:7778/api/session/release \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "session_id": "sess-xxx",
    "result": "success",
    "risk_detected": false
  }'
```

`result` 可选值：`success` / `failed` / `risk_blocked`。`risk_detected: true` 会触发更长的冷却期和代理降级。

#### 会话内 IP 轮换

```bash
curl -X POST http://localhost:7778/api/session/rotate \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{"session_id": "sess-xxx", "reason": "ip_blocked"}'
```

#### 查询状态

```bash
# 所有活跃会话
curl -H "X-API-Key: your-api-key" http://localhost:7778/api/session/status

# 单个会话
curl -H "X-API-Key: your-api-key" "http://localhost:7778/api/session/status?session_id=sess-xxx"

# 池子统计
curl -H "X-API-Key: your-api-key" http://localhost:7778/api/session/pool-stats
```

### 安全机制

- **API Key 认证**：所有 Session API 请求需携带 `X-API-Key`
- **冷却防滥用**：释放后代理进入冷却期（默认 30 分钟），风控触发后更长（默认 120 分钟）
- **日使用量上限**：单个代理每日最多使用 3 次（可配置）
- **并发上限**：最多 50 个并发会话（可配置）
- **租约保护**：被租用的代理不会被健康检查删除、优化替换或分配给普通代理端口用户
- **优雅停机**：收到 SIGINT/SIGTERM 信号时自动释放所有会话

## 📦 项目结构

```text
.
├── main.go                        # 程序入口，协调所有模块
├── config/                        # 配置系统（池子容量、延迟标准、验证参数等）
├── pool/                          # 🆕 池子管理器（入池判断、替换逻辑、状态计算）
├── fetcher/                       # 🆕 智能抓取器（源分组、断路器、按需抓取）
│   ├── fetcher.go                 # 多模式抓取逻辑
│   ├── source_manager.go          # 源状态管理和断路器
│   └── ip_query.go                # IP查询限流和多源降级
├── validator/                     # 代理验证（连接测试 + 出口IP检测 + 地理过滤）
├── checker/                       # 🆕 分批健康检查器
├── optimizer/                     # 🆕 优化轮换器（定时优化池子质量）
├── storage/                       # 🆕 扩展存储层（质量等级、使用统计、源状态表）
├── proxy/                         # 🆕 对外代理服务（HTTP + SOCKS5 双协议，4 端口 + 可选认证）
│   ├── server.go                  # HTTP 代理服务器
│   └── socks5_server.go           # SOCKS5 代理服务器
├── session/                       # 🆕 Session-Sticky 独占代理租约（REST API + 双协议代理）
│   ├── api.go                     # Session REST API 处理器
│   ├── manager.go                 # 会话管理器（acquire/release/rotate）
│   ├── proxy_server.go            # Session SOCKS5 + HTTP 代理服务器
│   ├── store.go                   # 会话持久化层
│   └── types.go                   # 数据类型定义
├── webui/                         # 🆕 黑客风格 WebUI（健康仪表盘、配置界面、RBAC）
├── logger/                        # 内存日志收集
├── test/                          # 🧪 测试脚本与文档
│   ├── test_proxy.sh              # HTTP 代理测试脚本（Bash）
│   ├── test_socks5.sh             # SOCKS5 代理测试脚本（Bash）
│   ├── test_http_https.sh         # HTTP 代理 HTTPS 访问测试脚本（Bash）
│   ├── test_proxy.go              # Go 测试脚本
│   ├── test_proxy.py              # Python 测试脚本
│   └── README.md                  # 测试脚本使用说明
├── .github/workflows/
│   └── docker-image.yml           # 🆕 GitHub Actions 自动构建（多平台镜像）
├── .env.example                   # 🆕 环境变量配置模板
├── docker-compose.yml             # 🆕 Docker Compose 配置（使用 Named Volume）
├── Dockerfile                     # Docker 构建文件
├── data/                          # 🆕 数据目录（SQLite 数据库、配置文件）
│   └── .gitkeep                   # 目录占位符和说明
├── GEO_FILTER.md                  # 🆕 地理过滤配置指南
├── DATA_DIRECTORY.md              # 🆕 数据目录说明文档
├── POOL_DESIGN.md                 # 完整架构设计文档
└── README.md                      # 本文件
```

## 🚀 快速开始

### 运行要求
- Go `1.25`
- CGO 编译环境（依赖 `github.com/mattn/go-sqlite3`）

### 本地运行

```bash
go run .
```

或先编译再启动：

```bash
go build -o proxygo .
./proxygo
```

程序启动后会：
1. 加载配置（环境变量 + `config.json`）
2. 初始化数据库和限流器
3. 清理不符合条件的代理（屏蔽国家出口、无地理信息）
4. 启动 WebUI（`:7778`）
5. 立即执行智能填充（按需抓取 + 严格验证）
6. 启动后台协程：
   - 状态监控（每 30 秒）
   - 健康检查（默认 5 分钟）
   - 优化轮换（默认 30 分钟）
7. 启动四个代理服务（支持可选认证）：
   - `:7776` - HTTP 最低延迟模式（稳定连接）
   - `:7777` - HTTP 随机轮换模式（IP 多样性）
   - `:7779` - SOCKS5 随机轮换模式（IP 多样性）
   - `:7780` - SOCKS5 最低延迟模式（稳定连接）
8. 启动 Session-Sticky 代理服务：
   - `:7781` - Session SOCKS5（独占代理）
   - `:7782` - Session HTTP（独占代理）

### 默认端口

#### HTTP 代理端口
- **7777 端口（HTTP 随机轮换）**：每次请求随机选择代理，IP 多样性高
- **7776 端口（HTTP 最低延迟）**：固定使用延迟最低的代理，性能优先

#### SOCKS5 代理端口
- **7779 端口（SOCKS5 随机轮换）**：每次连接随机选择代理，IP 多样性高
- **7780 端口（SOCKS5 最低延迟）**：固定使用延迟最低的代理，性能优先

#### 管理端口
- **WebUI**：`7778`
- **默认密码**：`goproxy`（可通过 `WEBUI_PASSWORD` 环境变量自定义）
- **访问方式**：本地使用 `localhost`，远程使用服务器 IP 地址

#### Session-Sticky 端口
- **7781 端口（Session SOCKS5）**：独占代理，通过 session_id 认证路由到绑定的上游代理
- **7782 端口（Session HTTP）**：独占代理，通过 Proxy-Authorization 中的 session_id 路由

### 使用代理

GoProxy 提供**四个代理端口**，支持 HTTP 和 SOCKS5 两种协议，满足不同场景需求：

#### 🌐 HTTP 协议代理

##### 🎲 7777 端口 - HTTP 随机轮换模式

适合需要 **IP 多样性** 的场景（爬虫、数据采集、负载均衡）：

```bash
# 本地使用
curl -x http://localhost:7777 https://httpbin.org/ip

# 远程使用
curl -x http://your-server-ip:7777 https://httpbin.org/ip
```

**特点**：
- 每次请求随机选择一个代理
- 优先使用高质量（S/A 级）代理
- IP 地址高度分散

##### ⚡ 7776 端口 - HTTP 最低延迟模式

适合需要 **稳定连接** 的场景（长连接、流媒体、实时通信）：

```bash
# 本地使用
curl -x http://localhost:7776 https://httpbin.org/ip

# 远程使用
curl -x http://your-server-ip:7776 https://httpbin.org/ip
```

**特点**：
- 固定使用池中延迟最低的代理
- 除非该代理失败，否则不会切换
- 失败时自动删除并切换到下一个最低延迟代理
- 性能和稳定性最优

#### 🔌 SOCKS5 协议代理

##### 🎲 7779 端口 - SOCKS5 随机轮换模式

适合需要 **原生 SOCKS5** 和 **IP 多样性** 的场景：

```bash
# 使用 curl（需要 7.21.7+）
curl --socks5 localhost:7779 https://httpbin.org/ip

# 远程使用
curl --socks5 your-server-ip:7779 https://httpbin.org/ip

# 使用 proxychains
echo "socks5 127.0.0.1 7779" > ~/.proxychains.conf
proxychains4 curl https://httpbin.org/ip
```

**特点**：
- 原生 SOCKS5 协议，更广泛的应用支持
- 每次连接随机选择代理
- 支持 TCP 和 UDP（如果上游代理支持）

##### ⚡ 7780 端口 - SOCKS5 最低延迟模式

适合需要 **SOCKS5 协议** 和 **稳定连接** 的场景：

```bash
# 本地使用
curl --socks5 localhost:7780 https://httpbin.org/ip

# 远程使用
curl --socks5 your-server-ip:7780 https://httpbin.org/ip
```

**特点**：
- 固定使用延迟最低的代理
- 适合需要 SOCKS5 协议的应用（如浏览器、游戏客户端）
- 最佳性能和稳定性

#### 环境变量配置

**HTTP 代理**：
```bash
# 使用随机模式
export http_proxy=http://localhost:7777
export https_proxy=http://localhost:7777

# 或使用稳定模式
export http_proxy=http://localhost:7776
export https_proxy=http://localhost:7776

# 远程使用（带认证）
export http_proxy=http://proxy:your_password@your-server-ip:7777
export https_proxy=http://proxy:your_password@your-server-ip:7777
```

**SOCKS5 代理**（更多应用支持）：
```bash
# 使用随机模式
export ALL_PROXY=socks5://localhost:7779

# 或使用稳定模式
export ALL_PROXY=socks5://localhost:7780

# 远程使用（带认证）
export ALL_PROXY=socks5://proxy:your_password@your-server-ip:7779
```

#### 端口对比

| 端口 | 协议 | 模式 | IP 多样性 | 稳定性 | 性能 | 适用场景 |
|------|------|------|----------|--------|------|---------|
| **7777** | HTTP | 随机轮换 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | 爬虫、数据采集 |
| **7776** | HTTP | 最低延迟 | ⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 长连接、流媒体 |
| **7779** | SOCKS5 | 随机轮换 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ | 浏览器、游戏、SSH |
| **7780** | SOCKS5 | 最低延迟 | ⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | 稳定应用连接 |
| **7781** | SOCKS5 | Session-Sticky | ⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | 注册流程、独占 IP |
| **7782** | HTTP | Session-Sticky | ⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | 注册流程、独占 IP |

#### 协议选择建议

**何时使用 HTTP 代理（7776/7777）**：
- 简单的 HTTP/HTTPS 请求
- curl、wget 等命令行工具
- 大多数编程语言的 HTTP 客户端

**何时使用 SOCKS5 代理（7779/7780）**：
- 需要代理非 HTTP 协议（如 SSH、FTP、游戏）
- 浏览器代理设置（SOCKS5 支持更好）
- 需要 UDP 支持的应用
- 某些应用只支持 SOCKS5 协议

#### SOCKS5 使用示例

**Python**：
```python
import requests

proxies = {
    'http': 'socks5://localhost:7779',
    'https': 'socks5://localhost:7779'
}
response = requests.get('https://httpbin.org/ip', proxies=proxies)
```

**Node.js**：
```javascript
const SocksProxyAgent = require('socks-proxy-agent');
const fetch = require('node-fetch');

const agent = new SocksProxyAgent('socks5://localhost:7779');
fetch('https://httpbin.org/ip', { agent }).then(res => res.json());
```

**浏览器配置**：
- 类型：SOCKS5
- 地址：`localhost` 或服务器 IP
- 端口：`7779`（随机）或 `7780`（稳定）

**SSH 隧道**：
```bash
ssh -o ProxyCommand='nc -X 5 -x localhost:7779 %h %p' user@remote-server
```

#### 自动重试机制说明

当你通过 GoProxy 发送请求时，如果上游代理失败，系统会**自动处理**：

1. **立即删除失败代理**：从池子中移除不可用的代理
2. **自动切换重试**：随机选择另一个可用代理重新发送请求（最多重试 3 次）
3. **用户完全无感知**：整个过程在服务端完成，你的应用只会收到成功响应或最终失败提示
4. **防止重复尝试**：同一请求中不会重复使用已失败的代理

**示例流程**：
```
用户请求 → 代理A失败(删除) → 自动切换代理B → 代理B成功 → 返回响应
```

这意味着即使池子中有部分失效代理，你的应用依然可以正常工作，系统会自动保持池子质量。

## 🐳 Docker 部署

> 💡 **自动构建**：GitHub Actions 自动构建多架构镜像（linux/amd64、linux/arm64），默认推送到 GHCR，可选推送到 Docker Hub  
> 💾 **数据持久化**：必须挂载 `data/` 目录以保存代理池数据和配置，详见 [`DATA_DIRECTORY.md`](DATA_DIRECTORY.md)

### 🔄 GitHub Actions 自动构建

项目配置了自动化 CI/CD 流程：

**触发条件**：
- 推送到 `main` 分支 → 构建 `latest` 标签
- 推送版本标签（如 `v1.0.0`）→ 构建多个版本标签（`1.0.0`, `1.0`, `1`, `latest`）
- 手动触发（workflow_dispatch）

**镜像仓库**：
- **GHCR**（默认）：`ghcr.io/isboyjc/goproxy` - 零配置，自动推送
- **Docker Hub**（可选）：`docker.io/isboyjc/goproxy` - 需配置 secrets：
  - `DOCKERHUB_USERNAME` - Docker Hub 用户名
  - `DOCKERHUB_TOKEN` - Docker Hub Access Token

**工作流程文件**：[`.github/workflows/docker-image.yml`](.github/workflows/docker-image.yml)

### 快速启动（推荐）

使用 docker-compose 一键部署：

```bash
# 1. 复制环境变量模板（可选，使用默认配置也可直接启动）
cp .env.example .env

# 2. 编辑 .env 设置密码等（可选）
vim .env

# 3. 启动服务（自动拉取最新镜像）
docker compose up -d

# 4. 访问 WebUI
# http://localhost:7778（默认密码：goproxy）
```

**数据持久化**：
- ✅ 使用 Docker Named Volume `goproxy-data`
- ✅ 容器重启/更新不会丢失数据
- ✅ 数据独立存储，不受项目目录影响

### docker run 方式部署

```bash
docker run -d --name proxygo \
  -p 7776:7776 \
  -p 7777:7777 \
  -p 7778:7778 \
  -p 7779:7779 \
  -p 7780:7780 \
  -e WEBUI_PASSWORD=your_password \
  -v goproxy-data:/app/data \
  ghcr.io/isboyjc/goproxy:latest
```

> 💡 **数据卷说明**：使用 Named Volume `goproxy-data` 确保数据持久化。如需本地开发调试，可改用 `-v "$(pwd)/data:/app/data"`。

### 数据备份与恢复

**导出数据**：
```bash
# 导出 Named Volume 数据
docker run --rm \
  -v goproxy-data:/data \
  -v $(pwd):/backup \
  alpine tar czf /backup/goproxy-backup-$(date +%Y%m%d).tar.gz -C /data .
```

**恢复数据**：
```bash
# 停止服务
docker compose down

# 恢复备份
docker run --rm \
  -v goproxy-data:/data \
  -v $(pwd):/backup \
  alpine sh -c "cd /data && tar xzf /backup/goproxy-backup-20260328.tar.gz"

# 重启服务
docker compose up -d
```

### 环境变量配置

**核心配置**（`.env` 文件）：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `BLOCKED_COUNTRIES` | `CN` | 屏蔽的国家代码（逗号分隔，如 `CN,RU`，留空=不屏蔽） |
| `PROXY_AUTH_ENABLED` | `false` | 是否启用代理认证（对外开放时强烈建议启用） |
| `PROXY_AUTH_USERNAME` | `proxy` | 代理认证用户名 |
| `PROXY_AUTH_PASSWORD` | 空 | 代理认证密码 |
| `WEBUI_PASSWORD` | `goproxy` | WebUI 登录密码 |
| `STABLE_PORT` | `7776` | HTTP 最低延迟代理端口 |
| `RANDOM_PORT` | `7777` | HTTP 随机轮换代理端口 |
| `WEBUI_PORT` | `7778` | WebUI 管理端口 |
| `SOCKS5_STABLE_PORT` | `7780` | SOCKS5 最低延迟代理端口 |
| `SOCKS5_RANDOM_PORT` | `7779` | SOCKS5 随机轮换代理端口 |

完整环境变量列表请查看 `.env.example` 文件。

**⚠️ 生产部署注意事项**：
- 如使用 Dokploy、Coolify 等平台部署，确保 `docker-compose.yml` 中配置了平台网络（如 `dokploy-network`）
- WebUI 端口可通过平台的域名功能访问，无需手动配置端口绑定
- 代理端口（7776/7777/7779/7780）通过 `IP:端口` 直接访问，**强烈建议启用认证**

**常用配置示例**：

```bash
# 场景 1：本地使用（默认配置）
# 直接运行 docker compose up -d 即可

# 场景 2：启用代理认证（推荐）
cat > .env << EOF
PROXY_AUTH_ENABLED=true
PROXY_AUTH_USERNAME=myuser
PROXY_AUTH_PASSWORD=secure_pass_123
WEBUI_PASSWORD=admin_pass_456
BLOCKED_COUNTRIES=CN
EOF
docker compose up -d

# 场景 3：屏蔽多个国家
cat > .env << EOF
BLOCKED_COUNTRIES=CN,RU,KP,IR
WEBUI_PASSWORD=admin_pass
EOF
docker compose up -d

# 场景 4：不屏蔽任何国家
cat > .env << EOF
BLOCKED_COUNTRIES=
WEBUI_PASSWORD=admin_pass
EOF
docker compose up -d
```

### 安全部署配置

**默认配置**：代理服务对外开放（端口 7776、7777、7779、7780），WebUI 对外开放（端口 7778）。

**⚠️ 重要提示**：

| 使用场景 | 配置建议 | 安全级别 |
|---------|---------|---------|
| **公网部署** | 启用代理认证 + 防火墙限制 | ⭐⭐⭐⭐⭐ 推荐 |
| **内网部署** | 启用代理认证 或 防火墙白名单 | ⭐⭐⭐⭐ 安全 |
| **本地测试** | 无需认证 | ⭐⭐⭐ 仅测试 |

**启用代理认证**（编辑 `.env`）：

```bash
PROXY_AUTH_ENABLED=true
PROXY_AUTH_USERNAME=myuser
PROXY_AUTH_PASSWORD=secure_pass_123
WEBUI_PASSWORD=admin_pass
```

**客户端使用（带认证）**：

```bash
# HTTP 代理 - 环境变量方式
export http_proxy=http://myuser:secure_pass_123@server-ip:7777
export https_proxy=http://myuser:secure_pass_123@server-ip:7777

# HTTP 代理 - curl 直接指定
curl -x http://myuser:secure_pass_123@server-ip:7777 https://httpbin.org/ip

# SOCKS5 代理 - curl 使用
curl --socks5 myuser:secure_pass_123@server-ip:7779 https://httpbin.org/ip

# Python - HTTP 代理
proxies = {'http': 'http://myuser:secure_pass_123@server-ip:7777', 'https': 'http://myuser:secure_pass_123@server-ip:7777'}

# Python - SOCKS5 代理
proxies = {'http': 'socks5://myuser:secure_pass_123@server-ip:7779', 'https': 'socks5://myuser:secure_pass_123@server-ip:7779'}
```

## ⚙️ 配置说明

### 配置文件示例

所有配置均可通过 WebUI 的 **Configure Pool** 界面在线调整，也可以手动编辑 `config.json`：

```json
{
  "pool_max_size": 100,
  "pool_http_ratio": 0.3,
  "pool_min_per_protocol": 10,
  "max_latency_ms": 2000,
  "max_latency_healthy": 1500,
  "max_latency_emergency": 3000,
  "validate_concurrency": 300,
  "validate_timeout": 8,
  "health_check_interval": 5,
  "health_check_batch_size": 20,
  "optimize_interval": 30,
  "replace_threshold": 0.7
}
```

### 配置参数详解

**服务端口配置**

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `proxy_port` | `:7777` | HTTP 随机轮换代理端口 |
| `stable_proxy_port` | `:7776` | HTTP 最低延迟代理端口 |
| `socks5_port` | `:7779` | SOCKS5 随机轮换代理端口 |
| `stable_socks5_port` | `:7780` | SOCKS5 最低延迟代理端口 |
| `webui_port` | `:7778` | WebUI 端口 |

**代理认证配置**

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `proxy_auth_enabled` | `false` | 是否启用代理认证（对外开放时建议启用） |
| `proxy_auth_username` | `proxy` | 代理认证用户名 |
| `proxy_auth_password_hash` | 空 | 代理认证密码 SHA256 哈希（HTTP Basic Auth） |

> 💡 **注意**：
> - 代理认证配置通过**环境变量**设置，不在 `config.json` 中
> - 启动时从 `PROXY_AUTH_ENABLED`、`PROXY_AUTH_USERNAME`、`PROXY_AUTH_PASSWORD` 环境变量读取
> - **HTTP 代理**使用 Basic Auth（密码哈希），**SOCKS5 代理**使用用户名/密码认证（明文传输，建议内网使用）

**池子容量配置**

| 参数 | 默认值 | 说明 | 推荐范围 |
| --- | --- | --- | --- |
| `pool_max_size` | `100` | 代理池总容量 | 50-150 ⚠️ |
| `pool_http_ratio` | `0.3` | HTTP 协议占比 | 0.2-0.5 |
| `pool_min_per_protocol` | `10` | 每协议最少保证数量 | 5-50 |

> ⚠️ **容量限制说明**：公开代理源质量有限，验证通过率通常只有 1-3%。受地理过滤、延迟标准、出口检测等因素影响，**实际填充率约为 70-90%**。如设置 150 容量，实际可能稳定在 105-135 个。建议根据实际需求设置合理容量。

**延迟标准配置** ⚡

| 参数 | 默认值 | 说明 | 推荐范围 |
| --- | --- | --- | --- |
| `max_latency_ms` | `2500` | 标准模式最大延迟（毫秒） | 2000-3500 |
| `max_latency_healthy` | `2000` | 健康模式严格延迟（毫秒） | 1500-2500 |
| `max_latency_emergency` | `4000` | 紧急/补充模式放宽延迟（毫秒） | 3000-5000 |

> 💡 **状态与延迟**：`emergency/critical/warning` 状态下使用 `max_latency_emergency`（4000ms），`healthy` 状态使用 `max_latency_healthy`（2000ms）。这确保在池子容量不足时能快速补充。

**验证配置**

| 参数 | 默认值 | 说明 | 推荐范围 |
| --- | --- | --- | --- |
| `validate_concurrency` | `300` | 验证并发数 | 200-500 |
| `validate_timeout` | `10` | 验证超时（秒） | 8-15 |

**健康检查配置**

| 参数 | 默认值 | 说明 | 推荐范围 |
| --- | --- | --- | --- |
| `health_check_interval` | `5` | 检查间隔（分钟） | 3-15 |
| `health_check_batch_size` | `20` | 每批检查数量 | 10-50 |

**优化配置**

| 参数 | 默认值 | 说明 | 推荐范围 |
| --- | --- | --- | --- |
| `optimize_interval` | `30` | 优化轮换间隔（分钟） | 15-120 |
| `replace_threshold` | `0.7` | 替换阈值（新代理需快 30%） | 0.5-0.9 |

### 不同场景配置建议

**小型 VPS（1C2G）**
```json
{
  "pool_max_size": 50,
  "pool_http_ratio": 0.3,
  "validate_concurrency": 100,
  "health_check_interval": 10,
  "health_check_batch_size": 10,
  "optimize_interval": 60
}
```

**中型服务器（2C4G+）**
```json
{
  "pool_max_size": 200,
  "pool_http_ratio": 0.6,
  "validate_concurrency": 300,
  "health_check_interval": 5,
  "health_check_batch_size": 30,
  "optimize_interval": 30
}
```

**低延迟优先**
```json
{
  "pool_max_size": 100,
  "max_latency_ms": 1000,
  "max_latency_healthy": 800,
  "optimize_interval": 15,
  "replace_threshold": 0.8
}
```

**高可用优先（需要更多代理）**
```json
{
  "pool_max_size": 300,
  "pool_http_ratio": 0.7,
  "pool_min_per_protocol": 20,
  "max_latency_ms": 3000
}
```

### 固定配置

以下配置在代码中固定或通过环境变量设置，无需在 `config.json` 中调整：

| 配置项 | 值 | 说明 |
| --- | --- | --- |
| `WebUIPort` | `:7778` | WebUI 端口 |
| `ProxyPort` | `:7777` | 随机轮换代理端口 |
| `StableProxyPort` | `:7776` | 最低延迟代理端口 |
| `ValidateURL` | `http://www.gstatic.com/generate_204` | 验证目标地址 |
| `IPQueryRateLimit` | `10 次/秒` | IP 查询限流 |
| `SourceFailThreshold` | `3` | 源降级阈值（连续失败） |
| `SourceDisableThreshold` | `5` | 源禁用阈值（连续失败） |
| `SourceCooldownMinutes` | `30` | 源禁用冷却时间 |
| `MaxRetry` | `3` | 代理请求失败重试次数 |

**环境变量配置**（启动时读取）：

| 环境变量 | 默认值 | 说明 |
| --- | --- | --- |
| `PROXY_AUTH_ENABLED` | `false` | 是否启用代理认证 |
| `PROXY_AUTH_USERNAME` | `proxy` | 代理认证用户名 |
| `PROXY_AUTH_PASSWORD` | 空 | 代理认证密码（原始密码，自动哈希） |
| `BLOCKED_COUNTRIES` | `CN` | 屏蔽的国家代码（逗号分隔，如 `CN,RU,KP`，留空=不屏蔽） |
| `SESSION_API_KEY` | 空 | Session API 鉴权密钥（不设置则 Session API 不可用） |
| `SESSION_ADVERTISE_HOST` | 空 | Session 代理对外宣告地址（默认 `127.0.0.1`） |

## 🎨 WebUI 使用指南

访问地址：`http://localhost:7778`（本地）或 `http://your-server-ip:7778`（远程）

### 👥 双角色权限系统

GoProxy WebUI 支持**访客模式**和**管理员模式**：

#### 访客模式（只读）

**无需登录**即可访问，可以查看所有数据但不能操作：

- ✅ 查看池子状态和质量分布
- ✅ 查看代理列表和详细信息
- ✅ 查看系统日志
- ✅ 点击复制代理地址
- ❌ 不能抓取代理
- ❌ 不能刷新延迟
- ❌ 不能删除代理
- ❌ 不能修改配置

**适用场景**：
- 团队成员监控代理池状态
- 展示给客户或第三方查看
- 公网开放访问（只读数据安全）

#### ⚡ 管理员模式（完全控制）

**登录后**拥有所有操作权限：

- ✅ 所有访客模式的查看功能
- ✅ 手动触发代理抓取
- ✅ 刷新所有代理延迟
- ✅ 刷新单个代理信息
- ✅ 删除指定代理
- ✅ 修改池子配置（容量、延迟标准、检查间隔等）

**默认密码**：`goproxy`（通过环境变量 `WEBUI_PASSWORD` 自定义）

### 健康仪表盘

**四宫格指标卡**
- **Pool Status**：当前池子状态（HEALTHY/WARNING/CRITICAL/EMERGENCY）
- **Total Proxies**：总代理数 / 池子容量
- **HTTP**：HTTP 代理数 / HTTP 槽位数 + 平均延迟
- **SOCKS5**：SOCKS5 代理数 / SOCKS5 槽位数 + 平均延迟

**质量分布可视化**
- 横向条形图展示 S/A/B/C 四级质量分布
- 实时显示各级别代理数量

### 代理注册表

**表格字段**
- **Grade**：质量等级（S/A/B/C，由综合质量分映射）
- **Score**：综合质量分（0-100，综合考虑延迟、成功率、失败记录、风险惩罚）
- **Protocol**：协议类型（HTTP/SOCKS5）
- **Address**：代理地址（host:port），点击可复制
- **Exit IP**：代理的出口 IP 地址
- **Location**：出口地理位置（国旗 emoji + 国家代码 + 城市）
- **Latency**：连接延迟（毫秒，动态颜色编码：绿/黄/橙/红）
- **Usage**：使用统计（使用总次数 / 成功次数，成功率指标）
- **Action**：操作按钮（刷新单个代理、删除代理，管理员可见）

**操作功能**

**所有用户可用**：
- **协议筛选**：下拉选择协议类型（全部/HTTP/SOCKS5）
- **国家筛选**：下拉选择出口国家（全部/动态国家列表，带国旗 emoji）
- **点击复制地址**：点击代理地址单元格直接复制到剪贴板
- **查看数据**：池子状态、质量分布、系统日志

**管理员专属**（需登录）：
- **Fetch Proxies**：手动触发智能抓取
- **Refresh Latency**：重新验证所有代理并更新延迟
- **刷新单个代理**：点击行内刷新按钮验证单个代理
- **删除代理**：点击行内删除按钮移除指定代理
- **Configure Pool**：打开配置界面修改池子参数

### 配置界面（⚡ 管理员专属）

点击 **Configure Pool** 打开配置模态框，包含：

**Pool Capacity 部分**
- Max Size：池子总容量
- HTTP Ratio：HTTP 协议占比（0.5 = 50%）
- Min Per Protocol：每协议最小保证

**Latency Standards 部分**
- Standard：标准模式延迟阈值
- Healthy：健康模式严格延迟
- Emergency：紧急模式放宽延迟

**Validation & Health Check 部分**
- Validate Concurrency：并发验证数
- Validate Timeout：验证超时
- Health Check Interval：健康检查间隔
- Health Check Batch Size：每批检查数量

**Optimization 部分**
- Optimize Interval：优化轮换间隔
- Replace Threshold：替换阈值（0.7 = 新代理需快 30%）

保存后立即生效，系统会自动调整池子策略。

## 🏗️ 核心架构

### 智能池子生命周期

```text
[启动] → 状态监控 → 判断池子健康度
              ↓
         需要补充？
          ↙     ↘
        是        否
         ↓         ↓
    智能抓取    保持监控
    (多模式)       ↓
         ↓      优化轮换
    严格验证   (定时执行)
         ↓         ↓
    智能入池    替换劣质代理
    (替换逻辑)     ↓
         ↓    分批健康检查
         ↓    (剔除失败)
         ↓         ↓
         └─────────┘
              ↓
         持续优化循环
```

### 状态转换机制

```text
Healthy (总数≥95% 且 各协议≥80%槽位)
   ↓ 代理失效
Warning (总数<95% 或 任一协议<80%)
   ↓ 继续失效
Critical (总数<50% 或 任一协议<20%槽位)
   ↓ 继续失效
Emergency (总数<10% 或 单协议缺失)
   ↑
   └─ 自动触发紧急抓取 ─┘
```

> 💡 **自动补充阈值**：当总数低于 95% 时进入 Warning 状态并触发自动补充，确保池子始终接近满容量运行。

### 抓取模式选择

| 池子状态 | 抓取模式 | 使用源 | 触发条件 |
| --- | --- | --- | --- |
| Emergency | 紧急模式 | 所有可用源 | 单协议缺失或总数<10% |
| Critical/Warning | 补充模式 | 快更新源 | 总数<95%或协议不均 |
| Healthy | 优化模式 | 慢更新源（随机2-3个） | 定时触发（30分钟） |

### 质量分级标准

| 等级 | 综合质量分 | 说明 | 主导因子 |
| --- | --- | --- | --- |
| S | 85-100 | 综合质量最佳，优先使用，健康状态可跳过检查 | 延迟低且稳定性高 |
| A | 70-84 | 质量良好，适合稳定使用 | 延迟与成功率平衡较好 |
| B | 50-69 | 可用，但会被优化轮换重点关注 | 质量一般或历史样本不足 |
| C | 0-49 | 淘汰候选，优先替换 | 延迟差、失败多或风险惩罚高 |

> 综合质量分是系统内部评分，不依赖外部信誉站点；当前会综合 `latency`、`success_count`、`fail_count`、`risk_count` 等指标后，再映射为 `S/A/B/C`。

## 🔧 数据库 Schema

### proxies 表

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | INTEGER | 主键 |
| `address` | TEXT | 代理地址（UNIQUE） |
| `protocol` | TEXT | 协议类型（http/socks5） |
| `exit_ip` | TEXT | 出口 IP |
| `exit_location` | TEXT | 出口位置 |
| `latency` | INTEGER | 延迟（毫秒） |
| `quality_grade` | TEXT | 质量等级（S/A/B/C） |
| `quality_score` | INTEGER | 综合质量分（0-100） |
| `risk_count` | INTEGER | 风险事件累计次数 |
| `use_count` | INTEGER | 使用次数 |
| `success_count` | INTEGER | 成功次数 |
| `fail_count` | INTEGER | 失败次数 |
| `last_used` | DATETIME | 最后使用时间 |
| `last_check` | DATETIME | 最后检查时间 |
| `created_at` | DATETIME | 创建时间 |
| `status` | TEXT | 状态（active/degraded/candidate_replace） |

### source_status 表

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | INTEGER | 主键 |
| `url` | TEXT | 源地址（UNIQUE） |
| `success_count` | INTEGER | 成功次数 |
| `fail_count` | INTEGER | 失败次数 |
| `consecutive_fails` | INTEGER | 连续失败次数 |
| `last_success` | DATETIME | 最后成功时间 |
| `last_fail` | DATETIME | 最后失败时间 |
| `status` | TEXT | 状态（active/degraded/disabled） |
| `disabled_until` | DATETIME | 禁用到期时间 |

## 🔍 代理源

系统内置 16 个代理源，分为快更新和慢更新两组：

**快更新源（5-30分钟更新）**
- proxifly/free-proxy-list (HTTP/SOCKS4/SOCKS5)
- ProxyScraper/ProxyScraper (HTTP/SOCKS4/SOCKS5)
- monosans/proxy-list (HTTP)

**慢更新源（每天更新）**
- TheSpeedX/SOCKS-List (HTTP/SOCKS4/SOCKS5)
- monosans/proxy-list (SOCKS4/SOCKS5)
- databay-labs/free-proxy-list (HTTP/SOCKS5)

系统会根据池子状态自动选择合适的源组：
- 紧急/补充模式：使用快更新源，快速填充
- 优化模式：随机选择慢更新源，精细优化

## 🚦 核心机制详解

### 1. 智能入池机制

每个代理在入池前需通过：
1. **连接验证**：能否成功连接 `http://www.gstatic.com/generate_204`
2. **出口 IP 检测**：获取代理的出口 IP
3. **地理位置查询**：获取出口 IP 的国家/城市
4. **延迟测试**：测量连接延迟
5. **质量评估**：根据延迟、成功率、失败记录、风险惩罚计算综合质量分，并映射为质量等级
6. **HTTPS 隧道验证**（仅 HTTP 协议）：通过代理实际访问随机 HTTPS 网站（Google/OpenAI/GitHub/Cloudflare/httpbin），验证 CONNECT 隧道可用性，首次失败自动换站重试

**入池判断逻辑**
- ✅ 协议槽位未满：直接加入
- ✅ 槽位满但总量允许10%浮动：浮动加入
- 🔄 池子满且质量更优：优先替换综合质量最低的现有代理，同分时仍要求显著更低延迟
- ❌ 池子满且质量不足：拒绝

### 2. 健康检查机制

**批次检查策略**
- 每次检查 20 个代理（可配置）
- 优先检查长时间未检查的
- 池子健康时跳过 S 级代理（降低资源消耗）

**检查结果处理**
- ✅ 验证通过：更新延迟、出口 IP、综合质量分与质量等级
- ❌ 验证失败：失败计数 +1，≥3次自动删除

### 3. 优化轮换机制

**触发条件**
- 池子状态：Healthy
- 定时触发：默认 30 分钟

**优化流程**
1. 从慢更新源随机抽取 2-3 个源
2. 抓取候选代理并验证
3. 筛选出延迟 ≤1500ms 的优质代理
4. 尝试替换池中 B/C 级代理（需快30%+）

**资源控制**
- 仅在池子健康时执行
- 抽取少量源，避免浪费
- 严格质量标准（≤1500ms）

### 4. 源管理与断路器

**状态跟踪**
- 记录每个源的成功/失败次数
- 连续失败 3 次：降级（Degraded）
- 连续失败 5 次：禁用 30 分钟（Disabled）
- 冷却期结束：自动恢复为 Active

**好处**
- 避免浪费资源在失效源上
- 自动恢复，无需人工干预
- 保护系统免受源故障影响

## 📖 常见问题

### Q: 为什么池子容量是固定的？
A: 固定容量可以：
- **可预测资源消耗**：内存、CPU、网络带宽均可控
- **提升代理质量**：通过严格准入和替换保持高质量
- **简化管理逻辑**：避免无限增长和复杂的淘汰策略

### Q: 如何调整池子大小和协议比例？
A: 
1. 访问 WebUI → 点击 **Configure Pool**
2. 修改 **Max Size** 和 **HTTP Ratio**
3. 点击 **Save Configuration**
4. 系统会自动调整槽位分配

示例：
- 池子大小 200，HTTP 比例 0.7 → HTTP 槽位 140，SOCKS5 槽位 60
- 池子大小 50，HTTP 比例 0.3 → HTTP 槽位 15，SOCKS5 槽位 35

### Q: 池子状态如何计算？
A: 
- **Healthy**：总数 ≥95% 且各协议 ≥80% 槽位
- **Warning**：总数 <95% 或任一协议 <80% 槽位
- **Critical**：总数 <50% 或任一协议 <20% 槽位
- **Emergency**：总数 <10% 或单协议缺失

### Q: 如何优化延迟？
A: 系统会自动优化，也可以手动调整：
1. 降低 `max_latency_healthy`（严格模式）
2. 增加 `optimize_interval` 频率（更频繁优化）
3. 调高 `replace_threshold`（要求新代理更快）
4. 点击 **Refresh Latency** 立即重新验证

### Q: 为什么有的代理没有出口 IP？
A: 
- IP 查询有限流（10 次/秒）
- 部分代理可能不支持 IP 查询
- 系统会在后续健康检查中补全信息
- 没有出口信息的代理会在启动时被自动清理

### Q: 如何配置地理过滤？
A: 
通过 `BLOCKED_COUNTRIES` 环境变量配置需要屏蔽的国家：

```bash
# 默认屏蔽中国大陆（CN）
BLOCKED_COUNTRIES=CN

# 屏蔽多个国家（逗号分隔）
BLOCKED_COUNTRIES=CN,RU,KP

# 不屏蔽任何国家（留空）
BLOCKED_COUNTRIES=
```

**工作机制**：
- **验证阶段**：检测到屏蔽国家出口直接拒绝入池
- **启动清理**：自动删除数据库中屏蔽国家的代理
- **精确匹配**：使用 ISO 3166-1 alpha-2 国家代码（CN、HK、US 等）

**常用国家代码**：`CN`=中国大陆 | `HK`=香港 | `RU`=俄罗斯 | `US`=美国 | `JP`=日本 | `SG`=新加坡

> 📖 **详细配置指南**：更多国家代码、使用场景、测试方法，请查看 [`GEO_FILTER.md`](./GEO_FILTER.md)

### Q: 资源消耗如何？
A: 
- **内存**：池子 100 个约 50MB，200 个约 100MB
- **CPU**：空闲时 <1%，验证时 10-30%（取决于并发数）
- **网络**：
  - IP 查询限流 10 次/秒
  - 按需抓取，避免无效流量
  - 健康检查批次小（20 个）

### Q: 代理服务如何启用认证？
A: 
1. 编辑 `.env` 文件：
   ```bash
   PROXY_AUTH_ENABLED=true
   PROXY_AUTH_USERNAME=myuser
   PROXY_AUTH_PASSWORD=mypass
   ```
2. 重启服务：`docker compose up -d`
3. 客户端使用：
   - HTTP：`http://myuser:mypass@server-ip:7777`
   - SOCKS5：`socks5://myuser:mypass@server-ip:7779`

### Q: 代理认证和 WebUI 认证有什么区别？
A: 
- **代理认证**：保护代理服务端口（7776/7777/7779/7780），防止代理被滥用
- **WebUI 认证**：保护 7778 管理后台，区分访客和管理员权限
- 两者独立配置，互不影响
- 启用代理认证时，HTTP 和 SOCKS5 代理都需要认证

## 🛠️ 开发与调试

### 查看日志

日志会输出到 stdout，同时在 WebUI 的 **System Log** 部分实时展示。

关键日志标识：
- `[pool]`：池子管理器
- `[fetch]`：抓取器
- `[source]`：源管理器
- `[health]`：健康检查器
- `[optimize]`：优化器
- `[monitor]`：状态监控器
- `[socks5]`：SOCKS5 代理服务器（握手、认证、连接建立）

### 数据库操作

```bash
# 查看当前代理
sqlite3 data/proxy.db "SELECT address, protocol, latency, quality_score, quality_grade, status FROM proxies LIMIT 10;"

# 查看质量分布
sqlite3 data/proxy.db "SELECT quality_grade, COUNT(*) FROM proxies WHERE status='active' GROUP BY quality_grade;"

# 查看源状态
sqlite3 data/proxy.db "SELECT url, status, consecutive_fails FROM source_status;"

# 清空池子（慎用）
sqlite3 data/proxy.db "DELETE FROM proxies;"
```

## 🧪 测试代理服务

项目提供了多种测试脚本，用于验证 HTTP 和 SOCKS5 代理服务功能和性能（位于 `test/` 目录）。

### 快速测试

**HTTP 代理测试**：
```bash
# 测试 HTTP 随机轮换模式（7777 端口）
./test/test_proxy.sh

# 测试 HTTP 最低延迟模式（7776 端口）
./test/test_proxy.sh 7776

# 使用 Go/Python 脚本
go run test/test_proxy.go 7777
python test/test_proxy.py 7776
```

**HTTP 代理 HTTPS 访问测试**：
```bash
# 持续测试 HTTP 代理访问 HTTPS 网站（随机访问 Google/OpenAI/GitHub 等）
./test/test_http_https.sh

# 指定端口
./test/test_http_https.sh 7776

# 指定端口 + 测试次数
./test/test_http_https.sh 7777 20
```

**SOCKS5 代理测试**：
```bash
# 测试 SOCKS5 随机轮换模式（7779 端口）
./test/test_socks5.sh localhost 7779

# 测试 SOCKS5 最低延迟模式（7780 端口）
./test/test_socks5.sh localhost 7780

# 持续测试 50 次
./test/test_socks5.sh localhost 7779 50

# 按 Ctrl+C 停止测试并查看统计
```

**HTTP 测试脚本**（`test_proxy.sh`）特点：
- **持续运行模式**：类似 `ping` 命令，持续发送请求
- 实时显示每次请求的出口 IP、国家和延迟
- 动态更新成功率统计
- 验证 HTTP 代理轮换机制
- 按 `Ctrl+C` 停止并显示完整统计报告

**SOCKS5 测试脚本**（`test_socks5.sh`）特点：
- 使用 `curl --socks5-hostname -k` 测试 SOCKS5 协议
- 显示出口 IP、国家、延迟和时间戳
- 验证 SOCKS5 代理轮换和稳定性
- 支持指定测试次数或持续测试
- 实时统计成功率和平均延迟
- 自动跳过 SSL 证书验证（免费代理常有证书问题）

### 测试输出示例

```
PROXY localhost:7777 (http://ip-api.com/json/?fields=countryCode,query): continuous mode

proxy from 🇺🇸 203.0.113.45: seq=1 time=1234ms
proxy from 🇩🇪 198.51.100.78: seq=2 time=987ms
proxy from 🇬🇧 192.0.2.123: seq=3 time=1567ms
proxy #4: request failed (timeout)
proxy from 🇯🇵 198.51.100.12: seq=5 time=890ms
...（持续运行，按 Ctrl+C 停止）

^C
---
50 requests transmitted, 47 received, 3 failed, 6.0% packet loss
```

**测试认证功能**：
```bash
# HTTP 代理 - 带认证
curl -x http://myuser:mypass@localhost:7777 https://httpbin.org/ip

# HTTP 代理 - 无认证（应该返回 407 错误）
curl -x http://localhost:7777 https://httpbin.org/ip

# SOCKS5 代理 - 带认证
curl --socks5 myuser:mypass@localhost:7779 https://httpbin.org/ip

# SOCKS5 代理 - 无认证（应该连接失败）
curl --socks5 localhost:7779 https://httpbin.org/ip
```

**测试脚本使用**：[`test/README.md`](./test/README.md)

## ❓ 常见问题

### Q1: HTTP 和 SOCKS5 代理有什么区别？

**简单理解**：
- **HTTP 代理**：专门为 HTTP/HTTPS 设计，简单易用，命令行工具支持好
- **SOCKS5 代理**：通用代理协议，支持所有 TCP/UDP 协议，功能更强大

**详细对比**见上方 [HTTP vs SOCKS5 协议对比](#-http-vs-socks5-协议对比) 章节。

### Q2: 我应该使用哪个端口？

根据你的需求：

| 需求 | 推荐端口 | 理由 |
|------|---------|------|
| 🕷️ **爬虫/数据采集** | 7777（HTTP 随机） | IP 高度分散，降低封禁风险 |
| 🎥 **流媒体/长连接** | 7776（HTTP 稳定） | 延迟最低，连接稳定 |
| 🌐 **浏览器代理** | 7779（SOCKS5 随机） | 协议支持好，IP 多样性高 |
| 🎮 **游戏/SSH/聊天** | 7780（SOCKS5 稳定） | 非 HTTP 协议，稳定优先 |

### Q3: SOCKS5 代理支持认证吗？

支持。当 `PROXY_AUTH_ENABLED=true` 时，所有代理端口（包括 SOCKS5）都需要认证：

```bash
# SOCKS5 带认证
curl --socks5 username:password@server-ip:7779 https://httpbin.org/ip

# 浏览器配置
# SOCKS5 Host: server-ip
# SOCKS5 Port: 7779
# Username: username
# Password: password
```

### Q4: 为什么推荐"域名:端口"访问而非域名直接配置代理？

**技术限制**：
- Cloudflare 等 DNS 托管服务的 HTTP 代理（橙云）**不支持非标准端口**（80/443 以外）
- 即使关闭橙云，DNS 记录也无法直接指向容器内部端口

**正确做法**：
```bash
# ✅ 推荐方式：IP:端口 或 域名:端口
curl -x http://proxy.example.com:7777 https://httpbin.org/ip
curl -x http://123.45.67.89:7777 https://httpbin.org/ip

# ❌ 不支持：直接域名（除非反向代理到 80/443）
curl -x http://proxy.example.com https://httpbin.org/ip
```

### Q5: 如何测试 SOCKS5 代理是否正常工作？

使用提供的测试脚本：

```bash
# 测试 SOCKS5 随机端口（7779）
./test/test_socks5.sh localhost 7779

# 持续测试 50 次
./test/test_socks5.sh localhost 7779 50
```

或手动测试：
```bash
# curl 测试（-k 跳过 SSL 证书验证）
curl -k --socks5-hostname localhost:7779 https://httpbin.org/ip

# 或测试 HTTP（不需要 SSL）
curl --socks5-hostname localhost:7779 http://httpbin.org/ip

# Python 测试（需要安装 requests[socks]）
python3 -c "import requests; print(requests.get('https://httpbin.org/ip', proxies={'http': 'socks5://localhost:7779', 'https': 'socks5://localhost:7779'}).json())"
```

### Q6: SOCKS5 代理失败率是否比 HTTP 高？

不会。两种服务质量和成功率相近，但**使用不同的上游代理**：
- **HTTP 代理服务**（7776/7777）：可使用池中的 HTTP 或 SOCKS5 上游代理
- **SOCKS5 代理服务**（7779/7780）：仅使用 SOCKS5 上游代理（因为许多免费 HTTP 代理不支持 HTTPS CONNECT）
- 两种服务都支持自动重试和故障切换

### Q7: 如何在浏览器中配置 SOCKS5 代理？

**Chrome/Edge**（使用插件如 SwitchyOmega）：
1. 安装 Proxy SwitchyOmega 插件
2. 新建情景模式 → SOCKS5
3. 代理服务器：`server-ip` 或 `localhost`
4. 代理端口：`7779`（随机）或 `7780`（稳定）
5. 如启用认证，填写用户名和密码

**Firefox**：
1. 设置 → 网络设置 → 手动代理配置
2. SOCKS Host：`server-ip` 或 `localhost`
3. Port：`7779` 或 `7780`
4. 选择：SOCKS v5
5. 勾选"通过 SOCKS 代理 DNS 查询"（可选）

### Q8: 如何查看 SOCKS5 服务的运行日志？

**本地运行**：
```bash
# 直接查看终端输出，SOCKS5 日志前缀为 [socks5]
# 启动日志示例：
# socks5 server listening on :7779 [随机轮换] [需认证 (用户: proxy)]
# socks5 server listening on :7780 [最低延迟] [无认证]

# 连接日志示例：
# [socks5] google.com:443 via 203.0.113.45:1080 established
```

**Docker 部署**：
```bash
# 查看最近 50 行日志
docker logs proxygo --tail 50

# 实时跟踪日志（含 SOCKS5）
docker logs proxygo -f

# 过滤 SOCKS5 相关日志
docker logs proxygo -f | grep socks5

# 查看 SOCKS5 错误日志
docker logs proxygo --tail 200 | grep -i "socks5.*failed"
```

**WebUI 查看**：
- 访问 WebUI（端口 7778）→ 点击右上角 **Logs** 按钮
- 实时日志面板会显示所有协议的连接和错误信息
- 包括 SOCKS5 握手、认证、连接建立等详细日志

## 🙏 致谢与声明

本项目基于 [jonasen1988/proxygo](https://github.com/jonasen1988/proxygo) 进行魔改和增强。

### 原项目
- **项目地址**：https://github.com/jonasen1988/proxygo
- **作者**：jonasen1988
- **基础功能**：代理抓取、验证、存储、HTTP代理服务、WebUI管理

### 本项目增强功能
在原项目基础上，我们进行了大量改进和功能增强：

- 🆕 **智能池子机制**：固定容量管理、质量分级（S/A/B/C）、智能替换逻辑、HTTP/SOCKS5 默认 3:7 比例
- 🆕 **HTTPS 可用性验证**：HTTP 协议代理入池/刷新时额外验证 HTTPS CONNECT 隧道，随机访问真实网站确认可用
- 🆕 **按需抓取策略**：源分组、断路器保护、Emergency/Refill/Optimize 多模式
- 🆕 **分层健康管理**：批次检查、智能跳过 S 级、定时优化轮换
- 🆕 **智能重试机制**：自动故障切换、失败即删除、防重复尝试
- 🆕 **双协议支持**：HTTP + SOCKS5 双协议，4 个服务端口（随机/稳定各 2 个）
- 🆕 **双模式策略**：每种协议都支持随机轮换（IP 多样性）和最低延迟（稳定连接）
- 🆕 **代理认证保护**：HTTP Basic Auth + SOCKS5 用户名/密码认证，对外开放时保护服务
- 🆕 **黑客风格 WebUI**：Matrix 美学、实时仪表盘、完整配置界面、中英文切换
- 🆕 **双角色权限**：访客模式（只读）+ 管理员模式（完全控制），可安全公网开放
- 🆕 **扩展存储层**：质量等级、使用统计、源状态管理
- 🆕 **测试套件**：HTTP + SOCKS5 + HTTPS 访问测试脚本，持续运行模式，显示国旗 emoji
- 🆕 **CI/CD 自动化**：GitHub Actions 自动构建多架构镜像（amd64/arm64），双仓库发布
- 🆕 **环境变量配置**：docker-compose + .env 文件，灵活配置各种部署场景

感谢原作者提供的基础实现，让我们能够在此之上构建更强大的代理池系统。

同时感谢 [LINUX DO](https://linux.do/) 社区的支持。

## 📝 License

MIT License
