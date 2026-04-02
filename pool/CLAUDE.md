[Root](../CLAUDE.md) > **pool**

# pool -- 池子管理器

## 模块职责

管理代理池的容量控制、状态判断、准入决策和替换逻辑。实现基于状态机的智能池子管理。

## 入口与启动

- `pool.NewManager(storage, config)` -- 创建 Manager 实例

## 对外接口

- `NewManager(s, cfg) *Manager` -- 构造函数
- `GetStatus() (*PoolStatus, error)` -- 获取池子状态（总数、各协议数、状态、平均延迟）
- `NeedsFetch(status) (needFetch, mode, preferredProtocol)` -- 判断是否需要抓取
- `NeedsFetchQuick(status) bool` -- 快速判断（用于提前终止验证）
- `TryAddProxy(proxy) (added, reason)` -- 尝试将代理加入池子
- `AdjustForConfigChange(oldSize, oldRatio)` -- 配置变更后调整池子

## 状态机

- `healthy`: 总数 >= 95% 容量
- `warning`: 总数 < 95% 但各协议 >= 20% 槽位
- `critical`: 任一协议 < 20% 槽位
- `emergency`: 总数 < 10% 或任一协议为 0

## 准入逻辑

1. 槽位未满 -> 直接入池
2. 槽位满但总池未满（10% 浮动） -> 浮动入池
3. 池子满 -> 尝试替换延迟最高的代理（新代理需快 30%+）

## 关键依赖与配置

- `storage` -- 代理数据读写
- `config` -- 池子容量、比例、替换阈值

## 数据模型

`PoolStatus`: Total, HTTP, SOCKS5, HTTPSlots, SOCKS5Slots, State, AvgLatencyHTTP, AvgLatencySocks5

## 相关文件清单

- `pool/manager.go` -- 唯一源文件

## Changelog

- 2026-04-01: 初始生成模块文档
