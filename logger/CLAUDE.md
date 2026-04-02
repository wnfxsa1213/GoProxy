[Root](../CLAUDE.md) > **logger**

# logger -- 内存日志收集器

## 模块职责

替换标准 `log` 输出，将日志同时写入内存环形缓冲区和控制台。供 WebUI 实时展示最近日志。

## 入口与启动

- `logger.Init()` -- 替换标准 log 输出（启动时调用一次）

## 对外接口

- `Init()` -- 初始化日志收集器，替换 `log.SetOutput`
- `GetLines(n int) []string` -- 返回最近 N 条日志

## 实现细节

- 内存缓冲区最大 500 条（`maxLines = 500`）
- 每条日志自动添加 `[HH:MM:SS]` 时间戳前缀
- 线程安全: `sync.RWMutex` 保护日志切片
- 同时输出到控制台（`fmt.Println`）

## 关键依赖与配置

无外部依赖。

## 相关文件清单

- `logger/logger.go` -- 唯一源文件（54 行）

## Changelog

- 2026-04-01: 初始生成模块文档
