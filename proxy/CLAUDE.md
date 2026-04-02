[Root](../CLAUDE.md) > **proxy**

# proxy -- 对外代理服务器

## 模块职责

提供 HTTP 和 SOCKS5 代理服务，支持随机轮换和最低延迟两种模式，内置自动重试和失败代理清除机制。支持可选的代理认证。

## 入口与启动

- `proxy.New(storage, config, mode, port)` -- 创建 HTTP 代理服务器
- `proxy.NewSOCKS5(storage, config, mode, port)` -- 创建 SOCKS5 代理服务器
- `(*Server).Start()` / `(*SOCKS5Server).Start()` -- 启动监听

## 对外接口

HTTP Server:
- `New(s, cfg, mode, port) *Server` -- 构造函数
- `Start() error` -- 启动 HTTP 代理（实现 http.Handler）
- `ServeHTTP(w, r)` -- 处理请求（CONNECT 隧道 + 普通 HTTP）

SOCKS5 Server:
- `NewSOCKS5(s, cfg, mode, port) *SOCKS5Server` -- 构造函数
- `Start() error` -- 启动 SOCKS5 代理（原始 TCP 监听）

## 代理模式

- `random`: 随机轮换（优先 S/A 级）
- `lowest-latency`: 选择延迟最低的代理

## 重试机制

- HTTP: 失败后换代理重试，最多 MaxRetry 次，失败代理立即删除
- SOCKS5: 同上，但重试次数为 MaxRetry+2（应对质量差的代理）
- SOCKS5 服务仅使用 SOCKS5 上游代理

## 认证支持

- HTTP: Proxy-Authorization Basic Auth（SHA256 密码比对，constant-time）
- SOCKS5: RFC 1929 用户名/密码认证（明文比对）
- 通过 `PROXY_AUTH_ENABLED` 环境变量启用

## 关键依赖与配置

- `storage` -- 获取代理
- `config` -- 端口、认证、超时、重试次数
- `golang.org/x/net/proxy` -- SOCKS5 拨号器

## 相关文件清单

- `proxy/server.go` -- HTTP 代理服务器
- `proxy/socks5_server.go` -- SOCKS5 代理服务器（手动实现协议）

## Changelog

- 2026-04-01: 初始生成模块文档
