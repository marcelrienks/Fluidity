#!/bin/bash
# End-to-End Docker Test Script
# Builds binaries, creates Docker images, runs containers, and verifies HTTP tunneling

set -e  # Exit on error

# Parse arguments
SKIP_BUILD=false
KEEP_CONTAINERS=false
TEST_URL="http://httpbin.org/get"

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --keep-containers)
            KEEP_CONTAINERS=true
            shift
            ;;
        --test-url)
            TEST_URL="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--skip-build] [--keep-containers] [--test-url URL]"
            exit 1
            ;;
    esac
done

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
SERVER_IMAGE="fluidity-server:test"
AGENT_IMAGE="fluidity-agent:test"
NETWORK_NAME="fluidity-test-net"
SERVER_CONTAINER="fluidity-server"  # Must match agent.docker.yaml server_ip
AGENT_CONTAINER="fluidity-agent"
PROXY_PORT=8081

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

echo -e "\n${MAGENTA}=== Fluidity End-to-End Docker Test ===${NC}"

cleanup() {
    if [ "$KEEP_CONTAINERS" = false ]; then
        echo -e "\n${YELLOW}Cleaning up test containers...${NC}"
        docker rm -f "$SERVER_CONTAINER" 2>/dev/null || true
        docker rm -f "$AGENT_CONTAINER" 2>/dev/null || true
        echo -e "  ${CYAN}[OK] Cleanup complete${NC}"
    else
        echo -e "\n${CYAN}Containers kept for manual testing:${NC}"
        echo "  docker logs -f $SERVER_CONTAINER"
        echo "  docker logs -f $AGENT_CONTAINER"
        echo "  curl -x http://127.0.0.1:$PROXY_PORT http://httpbin.org/get"
    fi
}

trap cleanup EXIT

handle_error() {
    echo -e "\n========================================"
    echo -e "${RED}  TEST FAILED: $1${NC}"
    echo -e "========================================"
    
    echo -e "\n${YELLOW}Server logs:${NC}"
    docker logs "$SERVER_CONTAINER" --tail 10 2>&1 || true
    
    echo -e "\n${YELLOW}Agent logs:${NC}"
    docker logs "$AGENT_CONTAINER" --tail 10 2>&1 || true
    
    exit 1
}

# Step 1: Build
if [ "$SKIP_BUILD" = false ]; then
    echo -e "\n${YELLOW}[Step 1] Building Linux binaries${NC}"
    cd "$PROJECT_ROOT"
    
    export GOOS=linux
    export GOARCH=amd64
    export CGO_ENABLED=0
    
    echo -e "  ${CYAN}Building server...${NC}"
    go build -o build/fluidity-server ./cmd/server || handle_error "Server build failed"
    
    echo -e "  ${CYAN}Building agent...${NC}"
    go build -o build/fluidity-agent ./cmd/agent || handle_error "Agent build failed"
    
    echo -e "  ${GREEN}[OK] Binaries built${NC}"
else
    echo -e "\n${YELLOW}[Step 1] Skipping build${NC}"
fi

# Step 2: Docker images
echo -e "\n${YELLOW}[Step 2] Building Docker images${NC}"
cd "$PROJECT_ROOT"
docker build -q -t "$SERVER_IMAGE" -f deployments/server/Dockerfile.local . || handle_error "Server image build failed"
docker build -q -t "$AGENT_IMAGE" -f deployments/agent/Dockerfile.local . || handle_error "Agent image build failed"
echo -e "  ${GREEN}[OK] Images built${NC}"

# Step 3: Network
echo -e "\n${YELLOW}[Step 3] Setting up network${NC}"
if ! docker network ls --filter "name=^${NETWORK_NAME}$" --format "{{.Name}}" | grep -q "^${NETWORK_NAME}$"; then
    docker network create "$NETWORK_NAME" >/dev/null 2>&1 || true
fi
echo -e "  ${GREEN}[OK] Network ready${NC}"

# Step 4: Cleanup old
echo -e "\n${YELLOW}[Step 4] Cleaning old containers${NC}"
docker ps -aq --filter "name=$SERVER_CONTAINER" | xargs -r docker rm -f >/dev/null 2>&1 || true
docker ps -aq --filter "name=$AGENT_CONTAINER" | xargs -r docker rm -f >/dev/null 2>&1 || true
echo -e "  ${GREEN}[OK] Cleaned${NC}"

# Step 5: Start server
echo -e "\n${YELLOW}[Step 5] Starting server${NC}"
docker run -d \
    --name "$SERVER_CONTAINER" \
    --network "$NETWORK_NAME" \
    -v "${PROJECT_ROOT}/configs:/root/configs:ro" \
    -v "${PROJECT_ROOT}/certs:/root/certs:ro" \
    "$SERVER_IMAGE" \
    --config configs/server.docker.yaml >/dev/null 2>&1 || handle_error "Failed to start server"

echo -e "  ${GREEN}[OK] Server started${NC}"

# Step 6: Start agent
echo -e "\n${YELLOW}[Step 6] Starting agent${NC}"
docker run -d \
    --name "$AGENT_CONTAINER" \
    --network "$NETWORK_NAME" \
    -p "${PROXY_PORT}:8080" \
    -v "${PROJECT_ROOT}/configs:/root/configs:ro" \
    -v "${PROJECT_ROOT}/certs:/root/certs:ro" \
    "$AGENT_IMAGE" \
    --config configs/agent.docker.yaml >/dev/null 2>&1 || handle_error "Failed to start agent"

echo -e "  ${GREEN}[OK] Agent started${NC}"

# Step 7: Wait
echo -e "\n${YELLOW}[Step 7] Waiting for initialization...${NC}"
sleep 3
echo -e "  ${GREEN}[OK] Ready${NC}"

# Step 8: Test
echo -e "\n${YELLOW}[Step 8] Testing HTTP tunnel${NC}"
echo -e "  ${CYAN}URL: $TEST_URL${NC}"

RESPONSE=$(curl -x "http://127.0.0.1:$PROXY_PORT" -s -w "\nHTTP_CODE:%{http_code}\n" "$TEST_URL" 2>&1)
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

# Step 9: Additional tests
echo -e "\n${YELLOW}[Step 9] Additional tests${NC}"

GH_RESPONSE=$(curl -x "http://127.0.0.1:$PROXY_PORT" -s -w "\nHTTP_CODE:%{http_code}\n" "https://api.github.com" 2>&1)
GH_CODE=$(echo "$GH_RESPONSE" | grep "HTTP_CODE:" | sed 's/HTTP_CODE://')
if [ "$GH_CODE" = "200" ]; then
    echo -e "  GitHub API: ${GREEN}HTTP $GH_CODE${NC}"
else
    echo -e "  GitHub API: ${YELLOW}HTTP $GH_CODE${NC}"
fi

EX_RESPONSE=$(curl -x "http://127.0.0.1:$PROXY_PORT" -s -w "\nHTTP_CODE:%{http_code}\n" "http://example.com" 2>&1)
EX_CODE=$(echo "$EX_RESPONSE" | grep "HTTP_CODE:" | sed 's/HTTP_CODE://')
if [ "$EX_CODE" = "200" ]; then
    echo -e "  example.com: ${GREEN}HTTP $EX_CODE${NC}"
else
    echo -e "  example.com: ${YELLOW}HTTP $EX_CODE${NC}"
fi

# Success
echo -e "\n========================================"
echo -e "${GREEN}  ALL TESTS PASSED!${NC}"
echo -e "========================================"
