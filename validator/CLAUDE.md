[Root](../CLAUDE.md) > **validator**

# validator -- 并发代理验证器

## 模块职责

并发验证代理的可用性，包括连通性测试、出口 IP 获取、地理位置识别、延迟测量，以及 HTTP 代理的 HTTPS CONNECT 隧道验证。

## 入口与启动

- `validator.New(concurrency, timeoutSec, validateURL)` -- 创建 Validator 实例

## 对外接口

- `New(concurrency, timeoutSec, validateURL) *Validator` -- 构造函数
- `ValidateAll(proxies) []Result` -- 并发验证所有代理，返回结果切片
- `ValidateStream(proxies) <-chan Result` -- 流式验证，边验证边返回
- `ValidateOne(proxy) (valid, latency, exitIP, exitLocation)` -- 验证单个代理

## 验证流程

1. 通过代理访问 `validateURL`（默认 gstatic generate_204）测试连通性和延迟
2. 检查响应状态码（200 或 204）
3. 检查响应时间是否超过 `maxResponseMs`
4. 通过代理访问 ip-api.com 获取出口 IP 和地理位置
5. 过滤屏蔽国家出口（根据 BlockedCountries 配置）
6. 可分配代理额外检测: 随机访问真实 HTTPS 站点验证连通性
   - 测试目标: Google / OpenAI / GitHub / Cloudflare / httpbin
   - 要求 TLS 握手和证书链校验通过

## 关键依赖与配置

- `config` -- 获取 BlockedCountries、MaxResponseMs
- `storage` -- 使用 `storage.Proxy` 数据类型
- `golang.org/x/net/proxy` -- SOCKS5 拨号器

## 数据模型

`Result` 结构体: Proxy, Valid, Latency, ExitIP, ExitLocation

## 测试与质量

无单元测试。

## 相关文件清单

- `validator/validator.go` -- 唯一源文件

## Changelog

- 2026-04-01: 初始生成模块文档
