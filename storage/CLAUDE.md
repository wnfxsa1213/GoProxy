[Root](../CLAUDE.md) > **storage**

# storage -- SQLite 持久化层

## 模块职责

提供 SQLite 数据库访问层，管理代理数据和源状态的 CRUD 操作，包含自动 schema 迁移。

## 入口与启动

- `storage.New(dbPath string)` -- 创建 Storage 实例，自动执行 schema 迁移

## 对外接口

- `New(dbPath) (*Storage, error)` -- 初始化数据库
- `AddProxy(address, protocol) error` -- 新增代理（已存在则忽略）
- `GetAll() ([]Proxy, error)` -- 获取所有可用代理（按延迟升序）
- `GetRandom() (*Proxy, error)` -- 随机取一个（优先 S/A 级）
- `GetRandomExclude(excludes) (*Proxy, error)` -- 排除指定地址随机取
- `GetLowestLatencyExclude(excludes) (*Proxy, error)` -- 排除后取最低延迟
- `GetRandomByProtocolExclude(protocol, excludes)` -- 按协议随机取
- `GetLowestLatencyByProtocolExclude(protocol, excludes)` -- 按协议取最低延迟
- `Delete(address) error` -- 删除代理
- `UpdateExitInfo(address, exitIP, exitLocation, latencyMs) error` -- 更新出口信息和质量等级
- `ReplaceProxy(oldAddress, newProxy) error` -- 事务性替换代理
- `GetWorstProxies(protocol, limit) ([]Proxy, error)` -- 获取延迟最高的代理
- `GetQualityDistribution() (map[string]int, error)` -- 质量分布统计
- `GetBatchForHealthCheck(batchSize, skipSGrade) ([]Proxy, error)` -- 健康检查批次
- `CalculateQualityGrade(latencyMs) string` -- 计算质量等级 S/A/B/C
- `GetDB() *sql.DB` -- 暴露底层 DB（供 SourceManager 使用）

## 数据模型

`proxies` 表:
- address (UNIQUE), protocol, exit_ip, exit_location
- latency, quality_grade (S/A/B/C)
- use_count, success_count, fail_count
- last_used, last_check, created_at, status

`source_status` 表:
- url (UNIQUE), success_count, fail_count, consecutive_fails
- last_success, last_fail, status (active/degraded/disabled), disabled_until

质量等级: S (<=500ms), A (501-1000ms), B (1001-2000ms), C (>2000ms)

## 关键依赖与配置

- `github.com/mattn/go-sqlite3` (CGO required)
- `MaxOpenConns(1)` -- SQLite 单写模式
- Schema 自动迁移（启动时检测并添加缺失字段）

## 测试与质量

无单元测试。

## 相关文件清单

- `storage/storage.go` -- 唯一源文件

## Changelog

- 2026-04-01: 初始生成模块文档
