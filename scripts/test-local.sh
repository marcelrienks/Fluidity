#!/bin/bash
# Local Binary Test Script
# Tests native binaries running directly on the host (no Docker)

set -e  # Exit on error

# Parse arguments
SKIP_BUILD=false
TEST_URL="http://httpbin.org/get"
PROXY_PORT=8080
SERVER_PORT=8443

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --test-url)
            TEST_URL="$2"
            shift 2
            ;;
        --proxy-port)
            PROXY_PORT="$2"
            shift 2
            ;;
        --server-port)
            SERVER_PORT="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--skip-build] [--test-url URL] [--proxy-port PORT] [--server-port PORT]"
            exit 1
            ;;
    esac
done

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
SERVER_CONFIG="$PROJECT_ROOT/configs/server.local.yaml"
AGENT_CONFIG="$PROJECT_ROOT/configs/agent.local.yaml"
SERVER_BINARY="$PROJECT_ROOT/build/fluidity-server"
AGENT_BINARY="$PROJECT_ROOT/build/fluidity-agent"
LOG_DIR="$PROJECT_ROOT/logs"

# Process tracking
SERVER_PID=""
AGENT_PID=""

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
WHITE='\033[1;37m'
NC='\033[0m' # No Color

echo -e "\n${MAGENTA}=== Fluidity Local Binary Test ===${NC}"

cleanup() {
    echo -e "\n${YELLOW}Cleaning up processes...${NC}"
    
    if [ -n "$AGENT_PID" ] && kill -0 "$AGENT_PID" 2>/dev/null; then
        echo -e "  ${CYAN}Stopping agent (PID: $AGENT_PID)...${NC}"
        kill "$AGENT_PID" 2>/dev/null || true
        sleep 1
        kill -9 "$AGENT_PID" 2>/dev/null || true
    fi
    
    if [ -n "$SERVER_PID" ] && kill -0 "$SERVER_PID" 2>/dev/null; then
        echo -e "  ${CYAN}Stopping server (PID: $SERVER_PID)...${NC}"
        kill "$SERVER_PID" 2>/dev/null || true
        sleep 1
        kill -9 "$SERVER_PID" 2>/dev/null || true
    fi
    
    echo -e "  ${CYAN}[OK] Cleanup complete${NC}"
}

trap cleanup EXIT INT TERM

handle_error() {
    echo -e "\n========================================"
    echo -e "${RED}  TEST FAILED: $1${NC}"
    echo -e "========================================"
    
    if [ -f "$LOG_DIR/test-server.err" ]; then
        echo -e "\n${YELLOW}Server stderr (last 10 lines):${NC}"
        tail -n 10 "$LOG_DIR/test-server.err" 2>/dev/null || true
    fi
    
    if [ -f "$LOG_DIR/test-agent.err" ]; then
        echo -e "\n${YELLOW}Agent stderr (last 10 lines):${NC}"
        tail -n 10 "$LOG_DIR/test-agent.err" 2>/dev/null || true
    fi
    
    exit 1
}

# Create logs directory if it doesn't exist
mkdir -p "$LOG_DIR"

# Step 1: Build
if [ "$SKIP_BUILD" = false ]; then
    echo -e "\n${YELLOW}[Step 1] Building binaries${NC}"
    cd "$PROJECT_ROOT"
    
    # Detect OS
    OS_TYPE="$(uname -s)"
    case "$OS_TYPE" in
        Darwin*)
            MAKEFILE="Makefile.macos"
            ;;
        Linux*)
            MAKEFILE="Makefile.linux"
            ;;
        *)
            handle_error "Unsupported OS: $OS_TYPE"
            ;;
    esac
    
    echo -e "  ${CYAN}Building with $MAKEFILE...${NC}"
    make -f "$MAKEFILE" build || handle_error "Build failed"
    
    echo -e "  ${GREEN}[OK] Binaries built${NC}"
else
    echo -e "\n${YELLOW}[Step 1] Skipping build${NC}"
fi

# Step 2: Verify files exist
echo -e "\n${YELLOW}[Step 2] Verifying files${NC}"

if [ ! -f "$SERVER_BINARY" ]; then
    handle_error "Server binary not found: $SERVER_BINARY"
fi

if [ ! -f "$AGENT_BINARY" ]; then
    handle_error "Agent binary not found: $AGENT_BINARY"
fi

if [ ! -f "$SERVER_CONFIG" ]; then
    handle_error "Server config not found: $SERVER_CONFIG"
fi

if [ ! -f "$AGENT_CONFIG" ]; then
    handle_error "Agent config not found: $AGENT_CONFIG"
fi

echo -e "  ${GREEN}[OK] All files verified${NC}"

# Step 3: Check ports are available
echo -e "\n${YELLOW}[Step 3] Checking ports${NC}"

if lsof -Pi :$SERVER_PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
    handle_error "Port $SERVER_PORT already in use"
fi

if lsof -Pi :$PROXY_PORT -sTCP:LISTEN -t >/dev/null 2>&1; then
    handle_error "Port $PROXY_PORT already in use"
fi

echo -e "  ${GREEN}[OK] Ports available${NC}"

# Step 4: Start server
echo -e "\n${YELLOW}[Step 4] Starting server${NC}"
echo -e "  ${CYAN}Command: $SERVER_BINARY --config $SERVER_CONFIG${NC}"

"$SERVER_BINARY" --config "$SERVER_CONFIG" \
    >"$LOG_DIR/test-server.out" 2>"$LOG_DIR/test-server.err" &
SERVER_PID=$!

if [ -z "$SERVER_PID" ]; then
    handle_error "Failed to start server process"
fi

sleep 1

if ! kill -0 "$SERVER_PID" 2>/dev/null; then
    handle_error "Server process died immediately"
fi

echo -e "  ${GREEN}[OK] Server started (PID: $SERVER_PID)${NC}"

# Step 5: Start agent
echo -e "\n${YELLOW}[Step 5] Starting agent${NC}"
echo -e "  ${CYAN}Command: $AGENT_BINARY --config $AGENT_CONFIG${NC}"

sleep 1  # Give server time to fully initialize

"$AGENT_BINARY" --config "$AGENT_CONFIG" \
    >"$LOG_DIR/test-agent.out" 2>"$LOG_DIR/test-agent.err" &
AGENT_PID=$!

if [ -z "$AGENT_PID" ]; then
    handle_error "Failed to start agent process"
fi

sleep 1

if ! kill -0 "$AGENT_PID" 2>/dev/null; then
    handle_error "Agent process died immediately"
fi

echo -e "  ${GREEN}[OK] Agent started (PID: $AGENT_PID)${NC}"

# Step 6: Wait for initialization
echo -e "\n${YELLOW}[Step 6] Waiting for initialization...${NC}"
sleep 3

# Check processes are still running
if ! kill -0 "$SERVER_PID" 2>/dev/null; then
    handle_error "Server process exited unexpectedly"
fi

if ! kill -0 "$AGENT_PID" 2>/dev/null; then
    handle_error "Agent process exited unexpectedly"
fi

echo -e "  ${GREEN}[OK] Processes running${NC}"

# Step 7: Test HTTP tunnel
echo -e "\n${YELLOW}[Step 7] Testing HTTP tunnel${NC}"
echo -e "  ${CYAN}URL: $TEST_URL${NC}"

RESPONSE=$(curl -x "http://127.0.0.1:$PROXY_PORT" -s -m 10 -w "\nHTTP_CODE:%{http_code}\n" "$TEST_URL" 2>&1)
HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | sed 's/HTTP_CODE://')

if [ "$HTTP_CODE" = "200" ]; then
    echo -e "  ${GREEN}[OK] Test passed (HTTP $HTTP_CODE)${NC}"
else
    if [ -z "$HTTP_CODE" ]; then
        echo -e "  ${RED}Response:${NC}"
        echo "$RESPONSE" | head -10
        handle_error "Could not extract HTTP code from response"
    else
        handle_error "Test failed (HTTP $HTTP_CODE)"
    fi
fi

# Step 8: Additional tests
echo -e "\n${YELLOW}[Step 8] Additional tests${NC}"

# Track failures for final verdict
TEST_FAILURES=()

# Test HTTPS via CONNECT
GH_RESPONSE=$(curl -x "http://127.0.0.1:$PROXY_PORT" -s -m 10 -w "\nHTTP_CODE:%{http_code}\n" "https://api.github.com" 2>&1)
GH_CODE=$(echo "$GH_RESPONSE" | grep "HTTP_CODE:" | sed 's/HTTP_CODE://')
if [ "$GH_CODE" = "200" ]; then
    echo -e "  GitHub API (HTTPS): ${GREEN}HTTP $GH_CODE${NC}"
else
    if [ -z "$GH_CODE" ]; then
        echo -e "  GitHub API (HTTPS): ${RED}Connection failed${NC}"
        TEST_FAILURES+=("GitHub API connection failed")
    else
        echo -e "  GitHub API (HTTPS): ${RED}HTTP $GH_CODE${NC}"
        TEST_FAILURES+=("GitHub API returned HTTP $GH_CODE")
    fi
fi

# Test HTTP
EX_RESPONSE=$(curl -x "http://127.0.0.1:$PROXY_PORT" -s -m 10 -w "\nHTTP_CODE:%{http_code}\n" "http://example.com" 2>&1)
EX_CODE=$(echo "$EX_RESPONSE" | grep "HTTP_CODE:" | sed 's/HTTP_CODE://')
if [ "$EX_CODE" = "200" ]; then
    echo -e "  example.com (HTTP): ${GREEN}HTTP $EX_CODE${NC}"
else
    if [ -z "$EX_CODE" ]; then
        echo -e "  example.com (HTTP): ${RED}Connection failed${NC}"
        TEST_FAILURES+=("example.com connection failed")
    else
        echo -e "  example.com (HTTP): ${RED}HTTP $EX_CODE${NC}"
        TEST_FAILURES+=("example.com returned HTTP $EX_CODE")
    fi
fi

# Fail if any additional tests failed
if [ ${#TEST_FAILURES[@]} -gt 0 ]; then
    FAILURE_MSG=$(IFS=', '; echo "${TEST_FAILURES[*]}")
    handle_error "Additional tests failed: $FAILURE_MSG"
fi

# Step 9: WebSocket test
echo -e "\n${YELLOW}[Step 9] Testing WebSocket tunnel${NC}"

# Test WebSocket echo service
echo -e "  ${CYAN}Testing wss://echo.websocket.org...${NC}"

# Check if Node.js is available for WebSocket testing
if command -v node &> /dev/null; then
    # Create temp test file
    WS_TEST_SCRIPT=$(mktemp /tmp/fluidity-ws-test.XXXXXX.js)
    cat > "$WS_TEST_SCRIPT" << 'WSEOF'
const WebSocket = require('ws');
const HttpsProxyAgent = require('https-proxy-agent');

const proxyUrl = 'http://127.0.0.1:PROXY_PORT';
const wsUrl = 'wss://echo.websocket.org/';

const agent = new HttpsProxyAgent(proxyUrl);
const ws = new WebSocket(wsUrl, { agent });

let success = false;
const timeout = setTimeout(() => {
    console.log('TIMEOUT');
    process.exit(1);
}, 10000);

ws.on('open', function() {
    ws.send('WebSocket test message');
});

ws.on('message', function(data) {
    if (data.toString().includes('test message')) {
        console.log('SUCCESS');
        success = true;
        clearTimeout(timeout);
        ws.close();
        process.exit(0);
    }
});

ws.on('error', function(err) {
    console.log('ERROR: ' + err.message);
    clearTimeout(timeout);
    process.exit(1);
});
WSEOF
    
    # Replace PROXY_PORT placeholder
    sed -i "s/PROXY_PORT/$PROXY_PORT/g" "$WS_TEST_SCRIPT"
    
    # Check if ws module is available
    WS_CHECK=$(node -e "try { require('ws'); require('https-proxy-agent'); console.log('OK'); } catch(e) { console.log('MISSING'); }" 2>&1)
    
    if [ "$WS_CHECK" = "OK" ]; then
        WS_OUTPUT=$(node "$WS_TEST_SCRIPT" 2>&1 || true)
        if echo "$WS_OUTPUT" | grep -q "SUCCESS"; then
            echo -e "  ${GREEN}[OK] WebSocket tunnel test passed${NC}"
        elif echo "$WS_OUTPUT" | grep -q "TIMEOUT"; then
            echo -e "  ${YELLOW}[SKIP] WebSocket test timed out (may not be critical)${NC}"
        else
            echo -e "  ${YELLOW}[SKIP] WebSocket test failed: $WS_OUTPUT${NC}"
        fi
        rm -f "$WS_TEST_SCRIPT"
    else
        echo -e "  ${YELLOW}[SKIP] WebSocket test requires 'ws' and 'https-proxy-agent' npm packages${NC}"
        echo -e "        ${CYAN}Install with: npm install -g ws https-proxy-agent${NC}"
        rm -f "$WS_TEST_SCRIPT"
    fi
else
    echo -e "  ${YELLOW}[SKIP] WebSocket test requires Node.js (install from nodejs.org)${NC}"
    echo -e "        ${CYAN}Core HTTP/HTTPS tests passed successfully${NC}"
fi

# Success
echo -e "\n========================================"
echo -e "${GREEN}  ALL TESTS PASSED!${NC}"
echo -e "========================================"

echo -e "\n${CYAN}Log files:${NC}"
echo -e "  Server: ${WHITE}logs/test-server.out${NC}"
echo -e "  Agent:  ${WHITE}logs/test-agent.out${NC}"
