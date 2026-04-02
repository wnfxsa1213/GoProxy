[Root](../CLAUDE.md) > **optimizer**

# optimizer -- 后台质量优化器

## 模块职责

定时从慢更新源抓取新代理，验证后用优质候选替换池中延迟较高的 B/C 级代理，持续提升池子整体质量。

## 入口与启动

- `optimizer.NewOptimizer(storage, fetcher, validator, poolMgr, config)` -- 创建实例
- `(*Optimizer).StartBackground()` -- 启动后台定时任务

## 对外接口

- `NewOptimizer(s, f, v, pm, cfg) *Optimizer` -- 构造函数
- `RunOnce()` -- 执行一次优化轮换
- `StartBackground()` -- 启动后台定时优化（每 OptimizeInterval 分钟）

## 优化流程

1. 检查池子状态，仅在 `healthy` 状态下执行
2. 使用 `optimize` 模式抓取候选代理（随机选 2-3 个慢更新源）
3. 流式验证候选代理，仅保留延迟 < MaxLatencyHealthy 的
4. 通过 `poolMgr.TryAddProxy()` 尝试替换延迟最高的代理

## 关键依赖与配置

- `fetcher` -- 抓取候选代理
- `validator` -- 验证候选代理
- `pool` -- 尝试入池/替换
- `config` -- OptimizeInterval, MaxLatencyHealthy

## 相关文件清单

- `optimizer/optimizer.go` -- 唯一源文件

## Changelog

- 2026-04-01: 初始生成模块文档
