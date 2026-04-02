# GoProxy Session-Sticky 规格审计报告

> 审计对象：`goproxy-secondary-dev-spec.md`
> 审计模型：Codex（后端架构） + Gemini（API 设计/集成）
> 日期：2026-04-01

---

## 综合评分

| 维度 | 评分 | 说明 |
|------|------|------|
| API 设计质量 | 14/20 | 偏离 RESTful 语义，缺少幂等性保障 |
| 协议与传输 | 8/20 | 动态端口方案存在严重安全/性能/运维风险 |
| 并发安全 | 10/20 | acquire 存在 TOCTOU，缺少状态机约束 |
| 数据模型 | 12/20 | 现有表结构不支持租约语义，需拆表 |
| 安全性 | 12/20 | 无 API 鉴权，冷却参数客户端可控 |
| 可观测性 | 16/20 | 有基础状态查询，缺标准化监控 |
| **总分** | **65/100** | **NEEDS_IMPROVEMENT — 不建议直接进入开发** |

---

## 一、Critical 级发现（必须修复）

### C1. 动态端口映射方案不可行（双模型一致）

**问题**：规格主方案是为每个 Session 分配 9000-9999 动态端口。但：
- 当前 GoProxy 启动时只创建固定 4 个监听器（`main.go:73`、`proxy/server.go:37`、`proxy/socks5_server.go:33`），动态端口与现有架构不连续
- 需要开放大段防火墙端口，Docker/NAT/1Panel 环境下端口映射灾难
- 频繁创建/销毁 TCP 监听器的内核开销高
- 端口范围 1000 个，但未定义 `max_concurrent_sessions` 与 `PoolMaxSize` 的关系

**建议**：**废弃动态端口，采用单端口 + 代理认证路由方案**。暴露一个固定 Session 代理端口（如 `7781`），CPA 侧将 `session_id` 作为代理认证的 Username（如 `socks5://sess-a1b2c3d4:token@goproxy:7781`）。现有代码已有 HTTP Basic Auth（`proxy/server.go:67`）和 SOCKS5 用户密码认证（`proxy/socks5_server.go:153`）基础，改造成本低。

### C2. acquire 伪代码存在 TOCTOU 竞态（Codex 发现）

**问题**：规格的分配逻辑先"筛选未占用"再"标记 leased"（`spec:159`），多个 goroutine 并发 acquire 时同一代理会被重复租出。当前存储层也是读后写分离（`storage/storage.go:265`、`pool/manager.go:133`）。

**建议**：acquire 必须由单点串行器负责，或用 SQLite `BEGIN IMMEDIATE` + 原子 claim 更新。推荐新增 `session.Manager` 单 goroutine/单 mutex 管理热状态。

### C3. API 无鉴权/限流/幂等（双模型一致）

**问题**：`POST /api/session/acquire` 直接开放，无 M2M 认证。现有 WebUI 认证是 cookie 登录式（`webui/server.go:78`），不适用于 API 调用。恶意调用方可耗尽全部代理。

**建议**：
- 引入 `X-API-Key` 或 Bearer Token 鉴权
- `task_id` 升级为幂等键：同一 `task_id` 重复 acquire 返回已有 session
- 对 acquire 做调用方级限速

---

## 二、High 级发现（强烈建议修复）

### H1. 现有失败策略与会话语义冲突（Codex 发现）

**问题**：现有 HTTP/SOCKS5 转发在单次请求失败时直接删代理（`proxy/server.go:137`、`proxy/socks5_server.go:103`）。对独占会话会导致"会话还活着，底层代理已被删"。

**建议**：新增 `broken/draining/cooling` 状态，活跃租约上的代理不能被直接物理删除。

### H2. leased/cooling 代理与 checker/optimizer 的交互未定义（Codex 发现）

**问题**：健康检查会批量更新/删除代理（`checker/health_checker.go:51`），优化器会替换非 S 级代理（`optimizer/optimizer.go:51`）。规格未说明 leased 代理如何豁免。

**建议**：leased 代理不参与优化替换；健康检查只做只读探测，不删除活跃租约上的代理。

### H3. TTL/release/rotate 三者竞态无状态机（双模型一致）

**问题**：TTL 扫描、手动 release、rotate 都会触发"释放当前租约"，没有 `active → releasing → released` 单向状态机，重复释放和过期释放会互相覆盖。

**建议**：会话表必须带 `state/version/released_at`，所有释放路径走同一原子转换。

### H4. 冷却参数由客户端控制存在越权风险（Gemini 发现）

**问题**：`cooldown_minutes` 和 `max_daily_uses` 由客户端请求体指定。恶意客户端传 `cooldown_minutes: 0` 可绕过冷却保护。

**建议**：这两个参数作为服务端全局配置硬性校验。客户端只能要求"更严格"的限制（如服务端默认 30 分钟，客户端可要求 60 分钟，但传 10 分钟会被强制重置为 30）。

### H5. proxy_addr 返回内部地址（双模型一致）

**问题**：`proxy_addr` 返回 `10.0.0.1:9001`，在 NAT/Docker 下不可用。当前配置无"对外宣告地址"字段。

**建议**：新增 `AdvertiseHost` 配置项，返回 `advertise_host:port` 或统一入口地址 + 会话凭证。

### H6. 现有数据模型不支持规格需求（Codex 发现）

**问题**：
- `proxies` 表只有 `exit_location` 拼接串，无结构化 `country_code`/`timezone`
- `ReplaceProxy` 物理删除旧记录再插入，会话历史/评分历史/日使用次数全部丢失
- `usage_today` 无存储方案

**建议**：
- 新增 `country_code`、`city`、`timezone` 规范字段
- 拆表：`sessions`（租约）、`proxy_usage_daily`（日使用计数）、`proxy_grade_events`（评分历史）
- `usage_today` 用 `proxy_usage_daily(proxy_id, usage_date_utc, count)` 按天插入/更新

### H7. SQLite 单写连接与高频租约状态冲突（Codex 发现）

**问题**：当前 DB `MaxOpenConns(1)`，checker/optimizer/WebUI 已在竞争写连接。再加 acquire/release 高频写入会放大延迟。

**建议**：热路径状态放内存（`sync.Map` 或单 goroutine），SQLite 只持久化冷却和历史。

### H8. 缺少 503 场景的 Retry-After 协同（Gemini 发现）

**问题**：所有代理冷却/占用时返回 503，未指导调用方等待时间，易导致高频轮询。

**建议**：503 响应中包含 `Retry-After: <seconds>` HTTP 头，GoProxy 预估最短冷却结束时间。

### H9. TTL 过期时长连接逃逸（Gemini 发现）

**问题**：TTL 超时关闭端口/释放代理时，如果调用方已建立 Keep-Alive 连接，底层代理可能被持续占用。

**建议**：release/TTL 过期时必须主动中断与该 session_id 关联的所有存量 TCP 连接。

---

## 三、Medium 级发现

| # | 发现 | 来源 | 建议 |
|---|------|------|------|
| M1 | 接口命名偏离 RESTful（POST /session/acquire → POST /sessions） | Gemini | 分配用 `POST /api/sessions`(201)，释放用 `DELETE /api/sessions/{id}` |
| M2 | `/api/pool/status` 的 `active_sessions` 无分页，大并发时报文过大 | Gemini | 拆出 `GET /api/sessions?limit=50&offset=0` |
| M3 | 现有 `/api/pool/status` 是 guest 可读，塞入 task_id/upstream_ip 会泄露业务信息 | Codex | 新增 `/api/admin/sessions`，仅管理员可访问 |
| M4 | `GET /api/proxy/{upstream_ip}` 用出口 IP 作主键不稳定（多上游可能共用同一出口） | Codex | 改为 `/api/proxy/by-id/{id}` |
| M5 | `rotate` 的连接语义不完整（旧连接如何处理） | Codex | 明确 rotate 只影响新连接，旧连接强制切断 |
| M6 | 上游代理会话中途失效时无错误契约 | Codex | 返回 `session_broken`，规定客户端调 rotate 或重新 acquire |
| M7 | `protocol`/`country`/`min_grade` 缺少非法值行为定义 | Codex | 补 400 错误码、枚举校验 |
| M8 | 缺少标准化监控指标（Prometheus `/metrics`） | Gemini | 暴露 `goproxy_session_active`、`goproxy_acquire_latency_seconds` 等 |
| M9 | 缺少 WebUI 会话监控视图 | 双模型 | P1 阶段新增"活跃会话大屏" + Kill Session 按钮 |
| M10 | 缺少优雅停机方案 | Codex | 补 SIGTERM/SIGINT 处理：停止新分配 → 标记终止 → 超时释放 |
| M11 | 缺少验收测试方案 | Codex | 补 5 组集成测试：并发 acquire 去重、TTL 自动释放、checker 与 lease 共存、optimizer 不替换活跃会话、Docker/NAT 地址可连通 |
| M12 | timezone 字段可能为空（IP 库不一定有） | Gemini | CPA 侧内置时区 Fallback 表，基于 country_code 回退到首都时区 |

---

## 四、改进建议（双模型交叉验证后的共识）

### 架构层

1. **废弃动态端口，主方案改为固定入口 + 会话凭证路由**。现有 HTTP Basic Auth 和 SOCKS5 用户密码认证已有基础，改造成本最低。
2. **新增独立 `session/` 模块**，包含 `session.Manager` 作为租约唯一真源。热状态在内存，持久化写入 SQLite。`proxy`、`checker`、`optimizer` 都通过此管理器判断代理是否可删/可换/可分配。
3. **引入代理状态机**：`available → leased → releasing → cooling → available`，替代当前的"失败即删"模型。

### 数据层

4. **拆表**：`sessions`（租约）、`proxy_usage_daily`（日使用计数）、`proxy_grade_events`（评分历史）。`proxies` 只保留长期属性。
5. **新增结构化地理字段**：`country_code`、`timezone` 独立列，不再复用 `exit_location` 拼接串。

### API 层

6. **幂等 acquire**：`task_id` 作为幂等键，重复提交返回已有 session。
7. **服务端强制冷却下限**：客户端只能要求更严格的限制。
8. **503 响应包含 `Retry-After`**。
9. **会话状态接口独立于现有 pool/status**，避免 guest 信息泄露。

### 运维层

10. **补 SIGTERM 优雅停机**。
11. **补集成测试脚本**（并发 acquire、TTL、release 幂等、rotate、异常停机恢复）。
12. **引入 /24 网段打散算法**（Gemini 建议）：避免同时给两个并发任务分配同一 C 段 IP。
13. **指数退避的动态风控惩罚**（Gemini 建议）：连续 risk_detected 时冷却时间指数增长。

---

## 五、建议的实施优先级调整

| 原优先级 | 功能 | 调整建议 |
|----------|------|----------|
| P0 | 动态端口方案 | **降级为备选**，主方案改为固定入口 + 认证路由 |
| — | session.Manager 模块 | **新增 P0**，作为所有租约操作的唯一入口 |
| — | API 鉴权 + 幂等 | **新增 P0**，acquire 必须幂等 |
| — | 数据模型拆表 | **新增 P0**，sessions + proxy_usage_daily |
| P1 | TTL 自动过期 | **升级为 P0**，防止泄漏是核心安全需求 |
| — | checker/optimizer lease-aware | **新增 P1**，防止活跃会话被后台任务破坏 |
| — | 优雅停机 | **新增 P1** |
| — | 集成测试 | **新增 P1** |
