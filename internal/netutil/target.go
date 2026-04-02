package netutil

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// JoinTargetHostPort 生成合法的 host:port，自动处理 IPv6 bracket。
func JoinTargetHostPort(host string, port uint16) string {
	return net.JoinHostPort(host, strconv.Itoa(int(port)))
}

// SplitTargetHostPort 拆分 target，兼容缺少 bracket 的 IPv6 literal。
func SplitTargetHostPort(target string) (string, string, error) {
	host, port, err := net.SplitHostPort(target)
	if err == nil {
		return host, port, nil
	}

	if strings.Count(target, ":") > 1 && !strings.Contains(target, "]") {
		idx := strings.LastIndex(target, ":")
		if idx > 0 && idx < len(target)-1 {
			host = target[:idx]
			port = target[idx+1:]
			if parsed := net.ParseIP(host); parsed != nil {
				return host, port, nil
			}
		}
	}

	return "", "", fmt.Errorf("invalid target address %q: %w", target, err)
}
