# GoProxy Session-Sticky 实施计划

> 基于审计报告修订，解决所有 Critical/High 级问题
> 生成日期：2026-04-01

---

## 架构决策摘要

| 决策 | 选择 | 原因 |
|------|------|------|
| 会话隔离方案 | 固定端口 + 认证路由 | 废弃动态端口；复用现有 SOCKS5/HTTP 认证基础 |
| 会话状态存储 | 内存热状态 + SQLite 冷持久化 | 避免 SQLite 单写瓶颈 |
| 并发控制 | session.Manager 单 mutex | 单机单进程模型，复杂度最低 |
| API 鉴权 | X-API-Key Header | M2M 场景，简单有效 |

---

## 新增模块：`session/`

```
session/
├── manager.go      # SessionManager：租约管理核心（内存状态 + 原子 acquire/release）
├── store.go        # SQLite 持久化（sessions 表 + proxy_usage_daily 表）
├── proxy_server.go # Session-Sticky 代理服务器（固定端口 7781/7782，认证路由）
├── api.go          # REST API handlers（acquire/release/rotate/status）
└── types.go        # Session/AcquireRequest/ReleaseRequest 等类型定义
```

---

## Phase 1: 数据模型扩展（P0 前置）

### 1.1 扩展 `proxies` 表

在 `storage/storage.go` 的 `initSchema()` 中新增迁移：

```sql
ALTER TABLE proxies ADD COLUMN country_code TEXT NOT NULL DEFAULT '';
ALTER TABLE proxies ADD COLUMN timezone TEXT NOT NULL DEFAULT '';
```

同时更新 `Proxy` 结构体，新增 `CountryCode`、`Timezone` 字段。

修改 `validator/validator.go`：验证时从 ip-api.com 响应中解析 `countryCode` 和 `timezone`，写入 `UpdateExitInfo`。

### 1.2 新建 `sessions` 表

```sql
CREATE TABLE IF NOT EXISTS sessions (
    id              TEXT PRIMARY KEY,          -- sess-xxxx
    task_id         TEXT NOT NULL,             -- 调用方任务 ID（幂等键）
    proxy_id        INTEGER NOT NULL,          -- 关联 proxies.id
    proxy_address   TEXT NOT NULL,             -- 上游代理地址
    protocol        TEXT NOT NULL DEFAULT 'socks5',
    state           TEXT NOT NULL DEFAULT 'active',  -- active/releasing/released/expired
    version         INTEGER NOT NULL DEFAULT 1,      -- 乐观锁版本号
    leased_at       INTEGER NOT NULL,          -- Unix timestamp
    expires_at      INTEGER NOT NULL,          -- Unix timestamp
    released_at     INTEGER,                   -- Unix timestamp
    result          TEXT,                      -- success/failed/risk_blocked
    risk_detected   INTEGER NOT NULL DEFAULT 0,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_task_id_active
    ON sessions(task_id) WHERE state = 'active';
CREATE INDEX IF NOT EXISTS idx_sessions_state ON sessions(state);
CREATE INDEX IF NOT EXISTS idx_sessions_proxy_id ON sessions(proxy_id);
```

### 1.3 新建 `proxy_usage_daily` 表

```sql
CREATE TABLE IF NOT EXISTS proxy_usage_daily (
    proxy_id    INTEGER NOT NULL,
    usage_date  TEXT NOT NULL,    -- 'YYYY-MM-DD' UTC
    use_count   INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (proxy_id, usage_date)
);
```

---

## Phase 2: session.Manager 核心（P0）

### 2.1 `session/types.go`

```go
type Session struct {
    ID            string
    TaskID        string
    ProxyID       int64
    ProxyAddress  string
    Protocol      string
    ExitIP        string
    CountryCode   string
    Timezone      string
    Grade         string
    State         string    // active/releasing/released/expired
    Version       int
    LeasedAt      int64
    ExpiresAt     int64
    ReleasedAt    int64
    UsageToday    int
}

type AcquireRequest struct {
    TaskID          string `json:"task_id"`
    TTL             int    `json:"ttl"`              // 默认 600
    MinGrade        string `json:"min_grade"`         // 默认 "B"
    CooldownMinutes int    `json:"cooldown_minutes"`  // 客户端建议值
    MaxDailyUses    int    `json:"max_daily_uses"`    // 客户端建议值
    Country         string `json:"country"`
    Protocol        string `json:"protocol"`          // 默认 "socks5"
}

type AcquireResponse struct {
    SessionID    string `json:"session_id"`
    ProxyAddr    string `json:"proxy_addr"`
    UpstreamIP   string `json:"upstream_ip"`
    CountryCode  string `json:"country_code"`
    Timezone     string `json:"timezone"`
    Grade        string `json:"grade"`
    UsageToday   int    `json:"usage_today"`
    LastReleasedAt int64 `json:"last_released_at"`
    ExpiresAt    int64  `json:"expires_at"`
}
```

### 2.2 `session/manager.go`

核心设计：
- 单 `sync.Mutex` 保护所有租约操作（消除 TOCTOU）
- 内存 `map[string]*Session`（sessionID → Session）
- 内存 `map[int64]string`（proxyID → sessionID，快速判断是否被租用）
- 内存 `map[string]string`（taskID → sessionID，幂等键）

关键方法：

```go
func (m *Manager) Acquire(req AcquireRequest) (*AcquireResponse, error)
func (m *Manager) Release(sessionID string, result string, riskDetected bool) error
func (m *Manager) Rotate(sessionID string, reason string) (*AcquireResponse, error)
func (m *Manager) IsLeased(proxyID int64) bool        // checker/optimizer 调用
func (m *Manager) IsCooling(proxyID int64) bool        // acquire 过滤调用
func (m *Manager) GetActiveSessions() []Session
func (m *Manager) ExpireCheck()                        // 后台 goroutine 每 30s 调用
```

Acquire 流程（mutex 内原子执行）：

```
1. 幂等检查：taskID 已有 active session → 直接返回
2. 从 storage 查询候选代理（SQL 过滤 grade/country/protocol）
3. 内存过滤：排除 leased + cooling + 日使用超限
4. 冷却参数强制下限：max(客户端值, 服务端最小值)
5. 评分优先 + 同评分随机选择
6. 标记 leased，写入 sessions 表，更新 proxy_usage_daily
7. 返回响应
```

### 2.3 服务端冷却参数强制

在 `config/config.go` 新增：

```go
// Session 配置
SessionAPIKey           string // API Key（环境变量 SESSION_API_KEY）
SessionStickyPort       string // Session SOCKS5 端口（默认 :7781）
SessionStickyHTTPPort   string // Session HTTP 端口（默认 :7782）
SessionMinCooldownMin   int    // 最小冷却时间（默认 30 分钟）
SessionMinDailyUses     int    // 最小日使用上限（默认 3）
SessionMaxConcurrent    int    // 最大并发会话数（默认 50）
SessionAdvertiseHost    string // 对外宣告地址（默认空=自动检测）
SessionRiskCooldownMin  int    // 风控触发后冷却（默认 120 分钟）
```

---

## Phase 3: Session-Sticky 代理服务器（P0）

### 3.1 `session/proxy_server.go`

固定端口方案（两个端口：7781 SOCKS5, 7782 HTTP）：

- SOCKS5 端口 7781：客户端用 `session_id` 作为 username 认证
- HTTP 端口 7782：客户端用 `session_id` 作为 Proxy-Auth username

路由逻辑：
```
1. 解析认证中的 username → session_id
2. 从 Manager 查找 active session
3. 如果 session 不存在或已过期 → 返回认证失败
4. 获取 session 绑定的上游代理
5. 通过上游代理转发请求
```

连接管理：
- 维护 `map[string][]net.Conn`（sessionID → 活跃连接列表）
- release/expire 时主动关闭所有关联连接

### 3.2 proxy_addr 返回格式

```
socks5://sess-a1b2c3d4:token@{advertise_host}:7781
http://sess-a1b2c3d4:token@{advertise_host}:7782
```

`advertise_host` 从配置读取，未配置时尝试自动检测本机外网 IP。

---

## Phase 4: REST API（P0）

### 4.1 `session/api.go`

挂载到现有 WebUI 的 HTTP mux 上（复用 7778 端口）：

```
POST   /api/session/acquire   → handleAcquire
POST   /api/session/release   → handleRelease
POST   /api/session/rotate    → handleRotate（P2）
GET    /api/session/status     → handleSessionStatus（管理员）
GET    /api/pool/session-stats → handlePoolSessionStats（管理员）
```

所有 `/api/session/*` 接口要求 `X-API-Key` 头匹配 `SESSION_API_KEY`。

### 4.2 错误响应规范

```json
// 400 Bad Request
{"error": "invalid_request", "message": "task_id is required"}

// 401 Unauthorized
{"error": "unauthorized", "message": "invalid or missing API key"}

// 404 Not Found
{"error": "session_not_found", "message": "session sess-xxx not found or already released"}

// 409 Conflict
{"error": "session_exists", "message": "active session already exists for task_id xxx"}

// 503 Service Unavailable + Retry-After header
{"error": "no_available_proxy", "message": "...", "total": 50, "leased": 35, "cooling": 12, "retry_after": 120}
```

---

## Phase 5: 与现有模块集成（P1）

### 5.1 checker 集成

修改 `checker/health_checker.go`：
- 健康检查前调用 `sessionMgr.IsLeased(proxyID)` 过滤
- leased 代理跳过健康检查（不删除、不降级）
- cooling 代理可以做只读探测但不删除

### 5.2 optimizer 集成

修改 `optimizer/optimizer.go`：
- `GetWorstProxies` 结果过滤掉 leased 和 cooling 的代理
- 替换操作前二次检查 `IsLeased`

### 5.3 proxy 集成

修改 `proxy/server.go` 和 `proxy/socks5_server.go`：
- 现有 4 端口的"失败即删"逻辑增加 lease 检查
- 如果代理被 leased，失败时不删除，只记录

### 5.4 评分反馈

release 时根据 result/risk_detected 调整评分：
- `success + !risk` → 不变（连续 5 次成功升一级）
- `success + risk` → 降一级 + 冷却 120 分钟
- `risk_blocked` → 降一级 + 冷却 120 分钟
- `failed` → 不变

### 5.5 WebUI 集成

在 `webui/server.go` 注册 session API 路由。
在 `webui/html.go` 新增"会话管理"面板（仅管理员可见）：
- 活跃会话列表
- Kill Session 按钮
- 冷却队列
- 按国家/评分的可用代理统计

---

## Phase 6: 运维保障（P1）

### 6.1 优雅停机

在 `main.go` 新增信号处理：

```go
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
go func() {
    <-sigCh
    log.Println("[main] 收到停机信号，开始优雅关闭...")
    sessionMgr.ReleaseAll("server_shutdown")  // 释放所有会话
    // 关闭监听器...
    os.Exit(0)
}()
```

### 6.2 日志

所有 session 操作使用 `[session]` 前缀：
```
[session] acquire: task=uuid-xxx proxy=1.2.3.4:1080 session=sess-abc grade=S country=US
[session] release: session=sess-abc result=success duration=45s
[session] expired: session=sess-abc (TTL 600s)
[session] rotate: session=sess-abc old=1.2.3.4 new=5.6.7.8 reason=risk_triggered
```

### 6.3 集成测试

新增 `test/test_session.sh`：
```bash
# 1. acquire → 验证返回 session_id + proxy_addr
# 2. 通过 proxy_addr 发请求 → 验证 IP 一致
# 3. release → 验证冷却生效
# 4. 并发 acquire × 5 → 验证 IP 隔离
# 5. 重复 acquire 同一 task_id → 验证幂等
# 6. 等待 TTL 过期 → 验证自动释放
```

---

## 实施顺序

```
Phase 1 (数据模型) ──→ Phase 2 (Manager 核心) ──→ Phase 3 (代理服务器)
                                                        │
                                                        ▼
                                              Phase 4 (REST API)
                                                        │
                                                        ▼
                                              Phase 5 (模块集成)
                                                        │
                                                        ▼
                                              Phase 6 (运维保障)
```

预计新增/修改文件：
- 新增：`session/manager.go`, `session/store.go`, `session/proxy_server.go`, `session/api.go`, `session/types.go`, `test/test_session.sh`
- 修改：`storage/storage.go`, `config/config.go`, `main.go`, `checker/health_checker.go`, `optimizer/optimizer.go`, `proxy/server.go`, `proxy/socks5_server.go`, `webui/server.go`, `webui/html.go`, `validator/validator.go`

---

## 端口总览（实施后）

| 端口 | 服务 | 变更 |
|------|------|------|
| 7776 | HTTP 代理（最低延迟） | 不变 |
| 7777 | HTTP 代理（随机轮换） | 不变 |
| 7778 | WebUI + Session API | 新增 API 路由 |
| 7779 | SOCKS5 代理（随机轮换） | 不变 |
| 7780 | SOCKS5 代理（最低延迟） | 不变 |
| 7781 | **Session SOCKS5**（认证路由） | **新增** |
| 7782 | **Session HTTP**（认证路由） | **新增** |
