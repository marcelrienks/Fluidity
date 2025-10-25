#!/bin/bash
# Setup Prerequisites Script for macOS
# This script checks for and installs required prerequisites for Fluidity

set -e

echo "========================================"
echo "Fluidity Prerequisites Setup (macOS)"
echo "========================================"
echo ""

HAS_ERRORS=false

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# 1. Check/Install Homebrew
echo -e "${YELLOW}[1/6] Checking Homebrew...${NC}"
if command_exists brew; then
    BREW_VERSION=$(brew --version | head -n 1)
    echo -e "${GREEN}  ✓ Homebrew is installed: $BREW_VERSION${NC}"
else
    echo -e "${RED}  ✗ Homebrew is not installed${NC}"
    echo -e "${YELLOW}  Installing Homebrew...${NC}"
    if /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"; then
        # Add Homebrew to PATH for this session
        if [[ $(uname -m) == "arm64" ]]; then
            eval "$(/opt/homebrew/bin/brew shellenv)"
        else
            eval "$(/usr/local/bin/brew shellenv)"
        fi
        echo -e "${GREEN}  ✓ Homebrew installed successfully${NC}"
    else
        echo -e "${RED}  ✗ Homebrew installation failed${NC}"
        HAS_ERRORS=true
    fi
fi
echo ""

# 2. Check/Install Go
echo -e "${YELLOW}[2/6] Checking Go (1.21+)...${NC}"
if command_exists go; then
    GO_VERSION=$(go version)
    echo -e "${GREEN}  ✓ Go is installed: $GO_VERSION${NC}"
else
    echo -e "${RED}  ✗ Go is not installed${NC}"
    echo -e "${YELLOW}  Installing Go via Homebrew...${NC}"
    if brew install go; then
        echo -e "${GREEN}  ✓ Go installed successfully${NC}"
    else
        echo -e "${RED}  ✗ Go installation failed. Please install manually from https://golang.org/dl/${NC}"
        HAS_ERRORS=true
    fi
fi
echo ""

# 3. Check/Install Make
echo -e "${YELLOW}[3/6] Checking Make...${NC}"
if command_exists make; then
    MAKE_VERSION=$(make --version | head -n 1)
    echo -e "${GREEN}  ✓ Make is installed: $MAKE_VERSION${NC}"
else
    echo -e "${RED}  ✗ Make is not installed${NC}"
    echo -e "${YELLOW}  Installing Xcode Command Line Tools (includes Make)...${NC}"
    if xcode-select --install 2>/dev/null; then
        echo -e "${YELLOW}  ⚠ Please complete the Xcode Command Line Tools installation dialog${NC}"
        echo -e "${YELLOW}    Then run this script again${NC}"
        exit 0
    else
        # Already installed or error
        if command_exists make; then
            echo -e "${GREEN}  ✓ Make is available${NC}"
        else
            echo -e "${RED}  ✗ Make installation failed${NC}"
            echo -e "${YELLOW}    Note: Make is optional - you can run build commands manually.${NC}"
        fi
    fi
fi
echo ""

# 4. Check/Install Docker
echo -e "${YELLOW}[4/6] Checking Docker...${NC}"
if command_exists docker; then
    DOCKER_VERSION=$(docker --version)
    echo -e "${GREEN}  ✓ Docker is installed: $DOCKER_VERSION${NC}"
else
    echo -e "${RED}  ✗ Docker is not installed${NC}"
    echo -e "${YELLOW}  Installing Docker via Homebrew...${NC}"
    if brew install --cask docker; then
        echo -e "${YELLOW}  ⚠ Docker installed. Please:${NC}"
        echo -e "${YELLOW}    1. Launch Docker Desktop from Applications${NC}"
        echo -e "${YELLOW}    2. Complete the Docker Desktop setup wizard${NC}"
        echo -e "${YELLOW}    3. Ensure Docker is running before building containers${NC}"
    else
        echo -e "${RED}  ✗ Docker installation failed${NC}"
        echo -e "${YELLOW}    Please install manually from https://www.docker.com/products/docker-desktop${NC}"
        HAS_ERRORS=true
    fi
fi
echo ""

# 5. Check/Install OpenSSL
echo -e "${YELLOW}[5/6] Checking OpenSSL...${NC}"
if command_exists openssl; then
    OPENSSL_VERSION=$(openssl version)
    echo -e "${GREEN}  ✓ OpenSSL is installed: $OPENSSL_VERSION${NC}"
else
    echo -e "${RED}  ✗ OpenSSL is not installed${NC}"
    echo -e "${YELLOW}  Installing OpenSSL via Homebrew...${NC}"
    if brew install openssl; then
        echo -e "${GREEN}  ✓ OpenSSL installed successfully${NC}"
        echo -e "${YELLOW}  ⚠ You may need to add OpenSSL to your PATH:${NC}"
        echo -e "${YELLOW}    export PATH=\"/opt/homebrew/opt/openssl@3/bin:\$PATH\"${NC}"
    else
        echo -e "${RED}  ✗ OpenSSL installation failed${NC}"
        HAS_ERRORS=true
    fi
fi
echo ""

# 6. Check/Install Node.js, npm, and npm packages
echo -e "${YELLOW}[6/6] Checking Node.js (18+), npm, and npm packages...${NC}"
NODE_INSTALLED=false
NPM_INSTALLED=false
if command_exists node; then
    NODE_INSTALLED=true
    NODE_VERSION=$(node --version)
    echo -e "${GREEN}  ✓ Node.js is installed: $NODE_VERSION${NC}"
else
    echo -e "${RED}  ✗ Node.js is not installed${NC}"
fi
if command_exists npm; then
    NPM_INSTALLED=true
    NPM_VERSION=$(npm --version)
    echo -e "${GREEN}  ✓ npm is installed: $NPM_VERSION${NC}"
else
    echo -e "${RED}  ✗ npm is not installed${NC}"
fi

if [ "$NODE_INSTALLED" = false ] || [ "$NPM_INSTALLED" = false ]; then
    echo -e "${YELLOW}  Installing Node.js and npm via Homebrew...${NC}"
    if brew install node; then
        if command_exists node; then
            NODE_INSTALLED=true
            NODE_VERSION=$(node --version)
            echo -e "${GREEN}  ✓ Node.js installed successfully: $NODE_VERSION${NC}"
        fi
        if command_exists npm; then
            NPM_INSTALLED=true
            NPM_VERSION=$(npm --version)
            echo -e "${GREEN}  ✓ npm installed successfully: $NPM_VERSION${NC}"
        fi
    else
        echo -e "${RED}  ✗ Node.js/npm installation failed. Please install manually from https://nodejs.org/${NC}"
        HAS_ERRORS=true
    fi
fi

if [ "$NODE_INSTALLED" = true ] && [ "$NPM_INSTALLED" = true ]; then
    echo -e "${YELLOW}  Checking npm packages (ws, https-proxy-agent)...${NC}"
    WS_INSTALLED=false
    PROXY_INSTALLED=false
    if npm list -g ws 2>/dev/null | grep -q "ws@"; then
        WS_INSTALLED=true
    fi
    if npm list -g https-proxy-agent 2>/dev/null | grep -q "https-proxy-agent@"; then
        PROXY_INSTALLED=true
    fi
    if [ "$WS_INSTALLED" = false ] || [ "$PROXY_INSTALLED" = false ]; then
        echo -e "${YELLOW}  Installing required npm packages...${NC}"
        if npm install -g ws https-proxy-agent; then
            echo -e "${GREEN}  ✓ npm packages installed successfully${NC}"
        else
            echo -e "${RED}  ✗ Error installing npm packages${NC}"
            HAS_ERRORS=true
        fi
    else
        echo -e "${GREEN}  ✓ Required npm packages are installed${NC}"
    fi
fi
echo ""

# Summary
echo "========================================"
echo "Setup Summary"
echo "========================================"

if [ "$HAS_ERRORS" = true ]; then
    echo -e "${YELLOW}⚠ Setup completed with errors. Please review the output above.${NC}"
    echo -e "${YELLOW}  Some prerequisites may need to be installed manually.${NC}"
    exit 1
else
    echo -e "${GREEN}✓ All prerequisites are installed!${NC}"
    echo ""
    echo -e "${CYAN}Next steps:${NC}"
    echo "  1. Close and reopen your terminal to refresh environment variables"
    echo "  2. Generate certificates: cd scripts && ./generate-certs.sh"
    echo "  3. Build the project: make build-macos"
    echo "  4. Run tests: ./scripts/test-local.sh"
    exit 0
fi
