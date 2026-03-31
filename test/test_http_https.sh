#!/bin/bash

# GoProxy HTTP 协议代理 HTTPS 访问测试脚本
# 随机访问多个 HTTPS 网站，验证 HTTP 代理的 CONNECT 隧道能力
# 用法: ./test_http_https.sh [端口号，默认7777] [测试次数，默认持续运行]
# 按 Ctrl+C 停止测试

PROXY_HOST="127.0.0.1"
PROXY_PORT="${1:-7777}"
MAX_COUNT="${2:-0}"  # 0 = 持续运行
DELAY=2

# 测试目标（HTTPS 网站）
TARGETS=(
    "https://www.google.com"
    "https://www.openai.com"
    "https://www.github.com"
    "https://www.cloudflare.com"
    "https://httpbin.org/ip"
)

# 统计变量
total=0
success=0
fail=0

# 获取毫秒时间戳
get_ms_time() {
    python3 -c 'import time; print(int(time.time() * 1000))'
}

# 捕获 Ctrl+C 信号
trap ctrl_c INT
function ctrl_c() {
    echo ""
    echo "---"
    if [ $total -gt 0 ]; then
        loss_rate=$(awk "BEGIN {printf \"%.1f\", ($total - $success)/$total*100}")
        success_rate=$(awk "BEGIN {printf \"%.1f\", $success/$total*100}")
        echo "$total requests transmitted, $success succeeded, $fail failed, ${loss_rate}% loss, ${success_rate}% success rate"
    fi
    exit 0
}

echo "HTTP PROXY HTTPS TEST — $PROXY_HOST:$PROXY_PORT"
echo "targets: ${#TARGETS[@]} HTTPS sites"
echo ""

while true; do
    # 随机选择目标
    idx=$((RANDOM % ${#TARGETS[@]}))
    target="${TARGETS[$idx]}"

    total=$((total + 1))

    start_time=$(get_ms_time)
    response=$(curl -x "http://${PROXY_HOST}:${PROXY_PORT}" \
                   -s -k \
                   -o /dev/null \
                   -w "%{http_code}" \
                   --connect-timeout 10 \
                   --max-time 15 \
                   "${target}" 2>&1)
    end_time=$(get_ms_time)
    elapsed=$((end_time - start_time))

    if [[ "$response" =~ ^[23] ]]; then
        echo "✅ seq=$total ${target} -> HTTP $response time=${elapsed}ms"
        success=$((success + 1))
    else
        echo "❌ seq=$total ${target} -> HTTP $response time=${elapsed}ms"
        fail=$((fail + 1))
    fi

    # 达到指定次数则停止
    if [ "$MAX_COUNT" -gt 0 ] && [ "$total" -ge "$MAX_COUNT" ]; then
        ctrl_c
    fi

    sleep $DELAY
done
