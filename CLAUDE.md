# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GoProxy is an intelligent proxy pool system written in Go. It automatically fetches HTTP/SOCKS5 proxies from public sources, validates them (exit IP + geolocation + latency), and serves them via 4 proxy ports (HTTP random/stable, SOCKS5 random/stable) plus a WebUI dashboard.

## Build & Run

```bash
go run .                        # requires Go 1.25, CGO_ENABLED=1
go build -o proxygo . && ./proxygo
docker compose up -d            # Docker deployment
```

CGO is required (`CGO_ENABLED=1`) because of the `github.com/mattn/go-sqlite3` dependency.

## Testing

No Go unit tests. Testing via shell scripts against a running instance:

```bash
./test/test_proxy.sh [port]           # HTTP proxy (default 7777)
./test/test_http_https.sh [port] [n]  # HTTPS via HTTP proxy
./test/test_socks5.sh [host] [port]   # SOCKS5 proxy
go run test/test_proxy.go 7777        # Go test client
python test/test_proxy.py 7776        # Python test client
```

## Architecture

Single binary, cooperating goroutines. Module name: `goproxy`.

### Module Structure (Mermaid)

````mermaid
graph TD
    ROOT["main.go"] --> CFG["config/"]
    ROOT --> STR["storage/"]
    ROOT --> FET["fetcher/"]
    ROOT --> VAL["validator/"]
    ROOT --> POOL["pool/"]
    ROOT --> CHK["checker/"]
    ROOT --> OPT["optimizer/"]
    ROOT --> PRX["proxy/"]
    ROOT --> WEB["webui/"]
    ROOT --> LOG["logger/"]
    FET --> STR
    VAL --> CFG
    POOL --> STR
    POOL --> CFG
    CHK --> STR
    CHK --> VAL
    OPT --> FET
    OPT --> VAL
    OPT --> POOL
    WEB --> STR
    WEB --> POOL
    WEB --> LOG
    PRX --> STR
    PRX --> CFG

    click CFG "./config/CLAUDE.md"
    click STR "./storage/CLAUDE.md"
    click FET "./fetcher/CLAUDE.md"
    click VAL "./validator/CLAUDE.md"
    click POOL "./pool/CLAUDE.md"
    click CHK "./checker/CLAUDE.md"
    click OPT "./optimizer/CLAUDE.md"
    click PRX "./proxy/CLAUDE.md"
    click WEB "./webui/CLAUDE.md"
    click LOG "./logger/CLAUDE.md"
````

### Module Index

| Module | Path | Description |
|--------|------|-------------|
| config | `config/` | 全局配置管理（环境变量 + config.json），线程安全单例 |
| storage | `storage/` | SQLite 持久化层（proxies + source_status 表） |
| fetcher | `fetcher/` | 多源代理抓取器，含断路器（SourceManager） |
| validator | `validator/` | 并发代理验证（连通性 + 出口 IP + 地理位置 + 延迟） |
| pool | `pool/` | 池子管理器（准入控制、槽位分配、替换逻辑） |
| checker | `checker/` | 后台健康检查器（批量验证，健康时跳过 S 级） |
| optimizer | `optimizer/` | 后台质量优化器（用更快代理替换慢代理） |
| proxy | `proxy/` | HTTP + SOCKS5 代理服务器（随机/最低延迟模式） |
| webui | `webui/` | WebUI 仪表盘（嵌入式 HTML + REST API + 双角色认证） |
| logger | `logger/` | 内存日志收集器，供 WebUI 实时展示 |
| test | `test/` | 集成测试脚本（Shell/Go/Python） |

### Key Design Patterns

- Pool state machine: healthy -> warning -> critical -> emergency
- Slot-based capacity: fixed size split by HTTP/SOCKS5 ratio (default 3:7)
- Smart admission: direct add if slots available, or replace worst proxy if 30%+ faster
- Protocol-parallel validation: SOCKS5 and HTTP validated concurrently
- Circuit breaker on sources: 3 fails -> degraded, 5 -> disabled for 30min
- Auto-retry on proxy failure: up to MaxRetry times, failed proxies deleted immediately
- SOCKS5 service only uses SOCKS5 upstreams; HTTP service can use either protocol

### Background Goroutines (started in main.go)

1. Status monitor -- every 30s, checks pool state, triggers smartFetchAndFill if needed
2. Health checker -- every HealthCheckInterval min, validates a batch of proxies
3. Optimizer -- every OptimizeInterval min, fetches and replaces B/C grade proxies
4. Config watcher -- listens for WebUI config changes, adjusts pool slots

### Ports

| Port | Service |
|------|---------|
| 7776 | HTTP proxy (lowest-latency mode) |
| 7777 | HTTP proxy (random rotation mode) |
| 7778 | WebUI dashboard |
| 7779 | SOCKS5 proxy (random rotation mode) |
| 7780 | SOCKS5 proxy (lowest-latency mode) |

### Configuration

- Environment variables: `WEBUI_PASSWORD`, `PROXY_AUTH_ENABLED`, `PROXY_AUTH_USERNAME`, `PROXY_AUTH_PASSWORD`, `BLOCKED_COUNTRIES`, `DATA_DIR`
- Persistent config: `config.json` (or `$DATA_DIR/config.json`), editable via WebUI
- Thread-safe via `sync.RWMutex`, loaded once at startup via `config.Load()`

### Storage

SQLite with `MaxOpenConns(1)` (single-writer). Two tables:
- `proxies`: quality grades S/A/B/C based on latency, exit IP/location, fail tracking
- `source_status`: circuit breaker state per source URL

Schema auto-migrates on startup.

### WebUI

Entire frontend embedded as Go string literals in `webui/html.go`. Dual-role auth: guest (read-only) and admin (full control via password).

## Code Conventions

- Log messages use `[module]` prefix: `[pool]`, `[fetch]`, `[health]`, `[optimize]`, `[monitor]`, `[socks5]`, `[proxy]`, `[tunnel]`, `[storage]`, `[source]`
- Comments and log messages are in Chinese
- Quality grades: S (<=500ms), A (501-1000ms), B (1001-2000ms), C (>2000ms)
- `storage.Proxy` is the shared data type across all modules

## AI Usage Guidelines

- This is a single-module Go project (no workspace/monorepo)
- All packages import via `goproxy/` prefix
- CGO must be enabled for sqlite3
- No unit tests exist; changes should be validated via integration test scripts
- `webui/html.go` is a large file (~49KB) containing embedded frontend HTML/CSS/JS

## Changelog

- 2026-04-01: Added module-level CLAUDE.md files, Mermaid architecture diagram, module index table, and .claude/index.json scan metadata

## .context 项目上下文

> 项目使用 `.context/` 管理开发决策上下文。

- 编码规范：`.context/prefs/coding-style.md`
- 工作流规则：`.context/prefs/workflow.md`
- 决策历史：`.context/history/commits.md`

**规则**：修改代码前必读 prefs/，做决策时按 workflow.md 规则记录日志。
