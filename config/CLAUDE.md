[Root](../CLAUDE.md) > **config**

# config -- 全局配置管理

## 模块职责

管理 GoProxy 的所有运行时配置，包括环境变量读取、config.json 持久化、线程安全的全局单例访问。

## 入口与启动

- `config.Load()` -- 启动时调用，从 config.json 加载配置，不存在则使用默认值
- `config.Get()` -- 获取当前配置（读锁保护）
- `config.Save()` -- 保存配置到文件并更新内存（写锁保护）

## 对外接口

- `Load() *Config` -- 加载配置（启动时调用一次）
- `Get() *Config` -- 获取当前全局配置
- `Save(cfg *Config) error` -- 保存配置到文件
- `DefaultConfig() *Config` -- 返回默认配置
- `(*Config).CalculateSlots() (httpSlots, socks5Slots int)` -- 计算各协议槽位数
- `(*Config).GetLatencyThreshold(poolStatus string) int` -- 根据池子状态返回延迟阈值

## 关键依赖与配置

- 环境变量: `WEBUI_PASSWORD`, `PROXY_AUTH_ENABLED`, `PROXY_AUTH_USERNAME`, `PROXY_AUTH_PASSWORD`, `BLOCKED_COUNTRIES`, `DATA_DIR`
- 持久化字段通过 `savedConfig` 结构体序列化为 JSON
- 线程安全: `sync.RWMutex` 保护 `globalCfg`

## 数据模型

`Config` 结构体包含:
- 服务端口配置 (WebUI/HTTP/SOCKS5, 随机/稳定模式)
- 池子容量配置 (PoolMaxSize, PoolHTTPRatio, PoolMinPerProtocol)
- 延迟标准配置 (标准/紧急/健康/降级 四档)
- 验证配置 (并发数、超时、验证 URL)
- 健康检查配置 (间隔、批次大小)
- 优化配置 (间隔、替换阈值)
- IP 查询限流配置
- 源管理配置 (降级/禁用阈值、冷却时间)

## 测试与质量

无单元测试。配置变更通过 WebUI API 测试。

## 相关文件清单

- `config/config.go` -- 唯一源文件，包含所有配置逻辑

## Changelog

- 2026-04-01: 初始生成模块文档
