[Root](../CLAUDE.md) > **fetcher**

# fetcher -- 多源代理抓取器

## 模块职责

从多个公开代理源并发抓取代理列表，支持智能模式选择（紧急/补充/优化），内置断路器和 IP 查询限流。

## 入口与启动

- `fetcher.New(httpURL, socks5URL, sourceManager)` -- 创建 Fetcher 实例

## 对外接口

- `New(httpURL, socks5URL, sourceManager) *Fetcher` -- 构造函数
- `Fetch() ([]storage.Proxy, error)` -- 从所有源并发抓取（旧接口）
- `FetchSmart(mode, preferredProtocol) ([]storage.Proxy, error)` -- 智能抓取
  - `emergency`: 忽略断路器，使用所有源
  - `refill`: 使用快更新源（5-30 分钟更新频率）
  - `optimize`: 随机选 2-3 个慢更新源（每天更新）
- `NewSourceManager(db) *SourceManager` -- 创建断路器管理器
- `InitIPQueryLimiter(rps)` -- 初始化 IP 查询限流器
- `GetExitIPInfo(client) (ip, location)` -- 多源降级 IP 查询

## 关键依赖与配置

- `storage` -- 使用 `storage.Proxy` 数据类型
- `golang.org/x/time/rate` -- IP 查询限流
- 代理源分两组:
  - `fastUpdateSources` (7 个): proxifly, ProxyScraper, monosans (5-60 分钟更新)
  - `slowUpdateSources` (7 个): TheSpeedX, monosans SOCKS, databay-labs (每天更新)

## 数据模型

`Source` 结构体: URL + Protocol (http/socks5)

`SourceManager` 断路器:
- `CanUseSource(url)` -- 检查源是否可用
- `RecordSuccess(url)` -- 记录成功
- `RecordFail(url, failThreshold, disableThreshold, cooldownMinutes)` -- 记录失败

IP 查询降级链: ip-api.com -> ipapi.co -> ipinfo.io -> httpbin.org

## 测试与质量

无单元测试。

## 相关文件清单

- `fetcher/fetcher.go` -- 抓取逻辑和源定义
- `fetcher/source_manager.go` -- 断路器（SourceManager）
- `fetcher/ip_query.go` -- IP 查询限流和多源降级

## Changelog

- 2026-04-01: 初始生成模块文档
