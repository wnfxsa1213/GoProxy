[Root](../CLAUDE.md) > **test**

# test -- 集成测试脚本

## 模块职责

提供多语言集成测试脚本，用于验证运行中的 GoProxy 实例的代理功能。类似 `ping` 命令的持续测试输出。

## 测试脚本

| 脚本 | 语言 | 用途 |
|------|------|------|
| `test_proxy.sh` | Bash | HTTP 代理持续测试（显示出口 IP + 国旗 emoji + 延迟） |
| `test_http_https.sh` | Bash | HTTP 代理 HTTPS CONNECT 隧道测试（随机访问 5 个 HTTPS 网站） |
| `test_socks5.sh` | Bash | SOCKS5 代理持续测试 |
| `test_proxy.go` | Go | HTTP 代理持续测试（Go 实现） |
| `test_proxy.py` | Python | HTTP 代理持续测试（Python 实现） |

## 使用方式

所有脚本需要 GoProxy 实例正在运行:

```bash
./test/test_proxy.sh [port]              # 默认 7777
./test/test_http_https.sh [port] [count] # 默认 7777, 持续运行
./test/test_socks5.sh [host] [port]      # 默认 127.0.0.1:7779
go run test/test_proxy.go [port]         # 默认 7777
python test/test_proxy.py [port]         # 默认 7777
```

按 Ctrl+C 停止，显示统计摘要（总请求数、成功数、失败率）。

## 关键依赖

- Bash 脚本依赖: `curl`, `python3`（用于毫秒时间戳和 emoji 转换）
- Go 脚本: 标准库
- Python 脚本: `requests` 库

## 相关文件清单

- `test/test_proxy.sh`
- `test/test_http_https.sh`
- `test/test_socks5.sh`
- `test/test_proxy.go`
- `test/test_proxy.py`
- `test/README.md`

## Changelog

- 2026-04-01: 初始生成模块文档
