[Root](../CLAUDE.md) > **checker**

# checker -- 后台健康检查器

## 模块职责

定时批量验证池中代理的健康状态，移除失效代理，更新延迟和质量等级。健康状态下跳过 S 级代理以减少不必要的验证开销。

## 入口与启动

- `checker.NewHealthChecker(storage, validator, config, poolMgr)` -- 创建实例
- `(*HealthChecker).StartBackground()` -- 启动后台定时任务

## 对外接口

- `NewHealthChecker(s, v, cfg, pm) *HealthChecker` -- 构造函数
- `RunOnce()` -- 执行一次健康检查
- `StartBackground()` -- 启动后台定时检查（每 HealthCheckInterval 分钟）

旧接口（兼容）:
- `New(s, v, cfg) *Checker` -- 旧版构造函数
- `(*Checker).Start()` -- 旧版启动

## 检查策略

1. 获取池子状态，判断是否跳过 S 级代理
2. 按 `last_check` 升序获取一批代理（优先检查最久未检查的）
3. 流式验证，更新延迟和出口信息
4. 验证失败的代理 fail_count+1，累计 >= 3 次则删除

## 关键依赖与配置

- `storage` -- 代理数据读写
- `validator` -- 执行验证
- `config` -- HealthCheckInterval, HealthCheckBatchSize
- `pool` -- 获取池子状态

## 相关文件清单

- `checker/health_checker.go` -- 新版健康检查器（推荐使用）
- `checker/checker.go` -- 旧版检查器（兼容保留）

## Changelog

- 2026-04-01: 初始生成模块文档
