#!/bin/bash
# GoProxy Session-Sticky API 集成测试
# 用法: ./test_session.sh [webui_port] [api_key]
# 需要 GoProxy 正在运行且配置了 SESSION_API_KEY

WEBUI_PORT="${1:-7778}"
API_KEY="${2:-${SESSION_API_KEY}}"
BASE_URL="http://127.0.0.1:${WEBUI_PORT}"

if [ -z "$API_KEY" ]; then
    echo "错误: 请设置 SESSION_API_KEY 环境变量或传入第二个参数"
    echo "用法: ./test_session.sh [port] [api_key]"
    exit 1
fi

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass=0
fail=0

check() {
    local desc="$1"
    local expected_code="$2"
    local actual_code="$3"
    local body="$4"

    if [ "$actual_code" = "$expected_code" ]; then
        echo -e "${GREEN}[PASS]${NC} $desc (HTTP $actual_code)"
        pass=$((pass + 1))
    else
        echo -e "${RED}[FAIL]${NC} $desc (expected $expected_code, got $actual_code)"
        echo "  Response: $body"
        fail=$((fail + 1))
    fi
}

echo "=== Session-Sticky API 集成测试 ==="
echo "目标: ${BASE_URL}"
echo ""

# 1. 无 API Key 应返回 401
echo "--- 认证测试 ---"
resp=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}/api/session/acquire" \
    -H "Content-Type: application/json" \
    -d '{"task_id":"test-no-auth","ttl":60}')
code=$(echo "$resp" | tail -1)
body=$(echo "$resp" | sed '$d')
check "无 API Key 返回 401" "401" "$code" "$body"

# 错误 API Key
resp=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}/api/session/acquire" \
    -H "Content-Type: application/json" \
    -H "X-API-Key: wrong-key" \
    -d '{"task_id":"test-bad-key","ttl":60}')
code=$(echo "$resp" | tail -1)
body=$(echo "$resp" | sed '$d')
check "错误 API Key 返回 401" "401" "$code" "$body"

echo ""
echo "--- Acquire 测试 ---"

# 2. 正常 Acquire
resp=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}/api/session/acquire" \
    -H "Content-Type: application/json" \
    -H "X-API-Key: ${API_KEY}" \
    -d '{"task_id":"test-session-1","ttl":120,"min_grade":"C","protocol":"socks5"}')
code=$(echo "$resp" | tail -1)
body=$(echo "$resp" | sed '$d')
check "Acquire 会话" "200" "$code" "$body"

SESSION_ID=""
PROXY_ADDR=""
if [ "$code" = "200" ]; then
    SESSION_ID=$(echo "$body" | python3 -c "import sys,json; print(json.load(sys.stdin).get('session_id',''))" 2>/dev/null)
    PROXY_ADDR=$(echo "$body" | python3 -c "import sys,json; print(json.load(sys.stdin).get('proxy_addr',''))" 2>/dev/null)
    echo "  session_id: $SESSION_ID"
    echo "  proxy_addr: $PROXY_ADDR"
fi

# 3. 幂等检查：相同 task_id 返回相同 session
if [ -n "$SESSION_ID" ]; then
    resp=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}/api/session/acquire" \
        -H "Content-Type: application/json" \
        -H "X-API-Key: ${API_KEY}" \
        -d '{"task_id":"test-session-1","ttl":120,"min_grade":"C","protocol":"socks5"}')
    code=$(echo "$resp" | tail -1)
    body=$(echo "$resp" | sed '$d')
    SESSION_ID_2=$(echo "$body" | python3 -c "import sys,json; print(json.load(sys.stdin).get('session_id',''))" 2>/dev/null)

    if [ "$SESSION_ID" = "$SESSION_ID_2" ] && [ "$code" = "200" ]; then
        echo -e "${GREEN}[PASS]${NC} 幂等: 相同 task_id 返回相同 session_id"
        pass=$((pass + 1))
    else
        echo -e "${RED}[FAIL]${NC} 幂等: 期望 session_id=$SESSION_ID, 实际=$SESSION_ID_2"
        fail=$((fail + 1))
    fi
fi

echo ""
echo "--- Status 测试 ---"

# 4. 查询所有活跃会话
resp=$(curl -s -w "\n%{http_code}" "${BASE_URL}/api/session/status" \
    -H "X-API-Key: ${API_KEY}")
code=$(echo "$resp" | tail -1)
body=$(echo "$resp" | sed '$d')
check "查询活跃会话" "200" "$code" "$body"

# 5. 查询单个会话
if [ -n "$SESSION_ID" ]; then
    resp=$(curl -s -w "\n%{http_code}" "${BASE_URL}/api/session/status?session_id=${SESSION_ID}" \
        -H "X-API-Key: ${API_KEY}")
    code=$(echo "$resp" | tail -1)
    body=$(echo "$resp" | sed '$d')
    check "查询单个会话" "200" "$code" "$body"
fi

# 6. Pool stats
resp=$(curl -s -w "\n%{http_code}" "${BASE_URL}/api/session/pool-stats" \
    -H "X-API-Key: ${API_KEY}")
code=$(echo "$resp" | tail -1)
body=$(echo "$resp" | sed '$d')
check "获取池子统计" "200" "$code" "$body"
echo "  Stats: $body"

echo ""
echo "--- Proxy 连接测试 ---"

# 7. 通过 Session-Sticky 代理发起请求
if [ -n "$PROXY_ADDR" ]; then
    resp=$(curl -s -k --max-time 15 -x "$PROXY_ADDR" "https://httpbin.org/ip" 2>&1)
    if echo "$resp" | grep -q '"origin"'; then
        origin=$(echo "$resp" | grep -o '"origin":"[^"]*"' | cut -d'"' -f4 | cut -d',' -f1)
        echo -e "${GREEN}[PASS]${NC} Session 代理连通: exit_ip=$origin"
        pass=$((pass + 1))

        # 第二次请求验证 IP 不变
        resp2=$(curl -s -k --max-time 15 -x "$PROXY_ADDR" "https://httpbin.org/ip" 2>&1)
        origin2=$(echo "$resp2" | grep -o '"origin":"[^"]*"' | cut -d'"' -f4 | cut -d',' -f1)
        if [ "$origin" = "$origin2" ]; then
            echo -e "${GREEN}[PASS]${NC} IP 一致性: $origin == $origin2"
            pass=$((pass + 1))
        else
            echo -e "${YELLOW}[WARN]${NC} IP 不一致: $origin != $origin2（可能上游代理 IP 变化）"
        fi
    else
        echo -e "${RED}[FAIL]${NC} Session 代理连通失败"
        echo "  Response: $resp"
        fail=$((fail + 1))
    fi
fi

echo ""
echo "--- Release 测试 ---"

# 8. Release 会话
if [ -n "$SESSION_ID" ]; then
    resp=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}/api/session/release" \
        -H "Content-Type: application/json" \
        -H "X-API-Key: ${API_KEY}" \
        -d "{\"session_id\":\"${SESSION_ID}\",\"result\":\"success\",\"risk_detected\":false}")
    code=$(echo "$resp" | tail -1)
    body=$(echo "$resp" | sed '$d')
    check "Release 会话" "200" "$code" "$body"

    # 幂等 release
    resp=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}/api/session/release" \
        -H "Content-Type: application/json" \
        -H "X-API-Key: ${API_KEY}" \
        -d "{\"session_id\":\"${SESSION_ID}\",\"result\":\"success\",\"risk_detected\":false}")
    code=$(echo "$resp" | tail -1)
    body=$(echo "$resp" | sed '$d')
    check "幂等 Release (已释放)" "200" "$code" "$body"
fi

echo ""
echo "=== 测试完成 ==="
echo -e "通过: ${GREEN}${pass}${NC}  失败: ${RED}${fail}${NC}  总计: $((pass + fail))"
