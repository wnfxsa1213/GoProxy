[Root](../CLAUDE.md) > **webui**

# webui -- WebUI 仪表盘

## 模块职责

提供 Web 管理界面，包括代理列表查看、池子状态监控、配置管理、日志查看等功能。支持访客（只读）和管理员（完整控制）双角色。

## 入口与启动

- `webui.New(storage, config, poolMgr, fetchTrigger, configChanged)` -- 创建实例
- `(*Server).Start()` -- 启动 HTTP 服务器（默认端口 :7778）

## 对外接口

- `New(s, cfg, pm, ft, cc) *Server` -- 构造函数
- `Start()` -- 启动 WebUI 服务

## API 端点

只读 API（访客可访问）:
- `GET /api/stats` -- 代理统计（总数、HTTP/SOCKS5 数量）
- `GET /api/proxies?protocol=` -- 代理列表
- `GET /api/logs` -- 最近 100 条日志
- `GET /api/pool/status` -- 池子状态
- `GET /api/pool/quality` -- 质量分布
- `GET /api/config` -- 当前配置
- `GET /api/auth/check` -- 检查登录状态

管理员 API（需登录）:
- `POST /api/proxy/delete` -- 删除代理
- `POST /api/proxy/refresh` -- 刷新单个代理（异步验证）
- `POST /api/fetch` -- 触发抓取
- `POST /api/refresh-latency` -- 刷新所有代理延迟（异步）
- `POST /api/config/save` -- 保存配置

## 认证机制

- 内存 session（SHA256 token，24 小时过期）
- 密码通过 SHA256 哈希比对
- Cookie-based session 管理

## 关键依赖与配置

- `storage` -- 代理数据
- `pool` -- 池子状态和管理
- `logger` -- 日志获取
- `config` -- 配置读写
- `validator` -- 单代理刷新验证

## 相关文件清单

- `webui/server.go` -- HTTP 服务器、路由、认证中间件、API 处理器
- `webui/dashboard.go` -- 已合并到 server.go（历史遗留命名）
- `webui/html.go` -- 嵌入式前端 HTML/CSS/JS（~49KB，赛博朋克风格 UI）

## Changelog

- 2026-04-01: 初始生成模块文档
