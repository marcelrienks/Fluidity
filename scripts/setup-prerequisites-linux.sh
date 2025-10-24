#!/bin/bash
# Setup Prerequisites Script for Linux
# This script checks for and installs required prerequisites for Fluidity
# Supports Ubuntu/Debian, RHEL/CentOS/Fedora, and Arch Linux

set -e

echo "========================================"
echo "Fluidity Prerequisites Setup (Linux)"
echo "========================================"
echo ""

HAS_ERRORS=false

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Detect Linux distribution
detect_distro() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        DISTRO=$ID
        DISTRO_VERSION=$VERSION_ID
    elif [ -f /etc/redhat-release ]; then
        DISTRO="rhel"
    else
        DISTRO="unknown"
    fi
}

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check if running as root
is_root() {
    [ "$(id -u)" -eq 0 ]
}

detect_distro
echo -e "${CYAN}Detected distribution: $DISTRO${NC}"
echo ""

# Check for root/sudo access
if ! is_root && ! command_exists sudo; then
    echo -e "${RED}Error: This script requires root access or sudo to install packages.${NC}"
    echo -e "${YELLOW}Please run as root or install sudo first.${NC}"
    exit 1
fi

SUDO=""
if ! is_root; then
    SUDO="sudo"
    echo -e "${YELLOW}Note: Some commands will require sudo password.${NC}"
    echo ""
fi

# 1. Update package manager
echo -e "${YELLOW}[1/7] Updating package manager...${NC}"
case $DISTRO in
    ubuntu|debian)
        $SUDO apt update
        echo -e "${GREEN}  ✓ Package lists updated${NC}"
        ;;
    fedora|rhel|centos)
        $SUDO yum update -y || $SUDO dnf update -y
        echo -e "${GREEN}  ✓ Package lists updated${NC}"
        ;;
    arch|manjaro)
        $SUDO pacman -Sy
        echo -e "${GREEN}  ✓ Package lists updated${NC}"
        ;;
    *)
        echo -e "${YELLOW}  ⚠ Unknown distribution, skipping package update${NC}"
        ;;
esac
echo ""

# 2. Check/Install Go
echo -e "${YELLOW}[2/7] Checking Go (1.21+)...${NC}"
if command_exists go; then
    GO_VERSION=$(go version)
    echo -e "${GREEN}  ✓ Go is installed: $GO_VERSION${NC}"
else
    echo -e "${RED}  ✗ Go is not installed${NC}"
    echo -e "${YELLOW}  Installing Go...${NC}"
    
    # Install Go from official source (package managers often have outdated versions)
    GO_VERSION="1.21.5"
    GO_ARCH="amd64"
    if [ "$(uname -m)" = "aarch64" ]; then
        GO_ARCH="arm64"
    fi
    
    cd /tmp
    wget "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    $SUDO rm -rf /usr/local/go
    $SUDO tar -C /usr/local -xzf "go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    rm "go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
    
    # Add to PATH
    if ! grep -q "/usr/local/go/bin" ~/.bashrc; then
        echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
        echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.bashrc
    fi
    
    export PATH=$PATH:/usr/local/go/bin
    export PATH=$PATH:$HOME/go/bin
    
    if command_exists go; then
        echo -e "${GREEN}  ✓ Go installed successfully${NC}"
        echo -e "${YELLOW}  ⚠ Please run: source ~/.bashrc (or reopen terminal)${NC}"
    else
        echo -e "${RED}  ✗ Go installation failed. Please install manually from https://golang.org/dl/${NC}"
        HAS_ERRORS=true
    fi
fi
echo ""

# 3. Check/Install Make
echo -e "${YELLOW}[3/7] Checking Make...${NC}"
if command_exists make; then
    MAKE_VERSION=$(make --version | head -n 1)
    echo -e "${GREEN}  ✓ Make is installed: $MAKE_VERSION${NC}"
else
    echo -e "${RED}  ✗ Make is not installed${NC}"
    echo -e "${YELLOW}  Installing Make...${NC}"
    case $DISTRO in
        ubuntu|debian)
            $SUDO apt install -y build-essential
            ;;
        fedora|rhel|centos)
            $SUDO yum groupinstall -y "Development Tools" || $SUDO dnf groupinstall -y "Development Tools"
            ;;
        arch|manjaro)
            $SUDO pacman -S --noconfirm base-devel
            ;;
        *)
            echo -e "${RED}  ✗ Cannot install Make automatically on this distribution${NC}"
            HAS_ERRORS=true
            ;;
    esac
    
    if command_exists make; then
        echo -e "${GREEN}  ✓ Make installed successfully${NC}"
    else
        echo -e "${YELLOW}  Note: Make is optional - you can run build commands manually.${NC}"
    fi
fi
echo ""

# 4. Check/Install Docker
echo -e "${YELLOW}[4/7] Checking Docker...${NC}"
if command_exists docker; then
    DOCKER_VERSION=$(docker --version)
    echo -e "${GREEN}  ✓ Docker is installed: $DOCKER_VERSION${NC}"
else
    echo -e "${RED}  ✗ Docker is not installed${NC}"
    echo -e "${YELLOW}  Installing Docker...${NC}"
    
    case $DISTRO in
        ubuntu|debian)
            # Install Docker using official Docker repository
            $SUDO apt install -y ca-certificates curl gnupg lsb-release
            $SUDO install -m 0755 -d /etc/apt/keyrings
            curl -fsSL https://download.docker.com/linux/$DISTRO/gpg | $SUDO gpg --dearmor -o /etc/apt/keyrings/docker.gpg
            $SUDO chmod a+r /etc/apt/keyrings/docker.gpg
            
            echo \
              "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/$DISTRO \
              $(lsb_release -cs) stable" | $SUDO tee /etc/apt/sources.list.d/docker.list > /dev/null
            
            $SUDO apt update
            $SUDO apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
            ;;
        fedora)
            $SUDO dnf -y install dnf-plugins-core
            $SUDO dnf config-manager --add-repo https://download.docker.com/linux/fedora/docker-ce.repo
            $SUDO dnf install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
            ;;
        centos|rhel)
            $SUDO yum install -y yum-utils
            $SUDO yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
            $SUDO yum install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
            ;;
        arch|manjaro)
            $SUDO pacman -S --noconfirm docker docker-compose
            ;;
        *)
            echo -e "${RED}  ✗ Cannot install Docker automatically on this distribution${NC}"
            echo -e "${YELLOW}    Please install manually from https://docs.docker.com/engine/install/${NC}"
            HAS_ERRORS=true
            ;;
    esac
    
    if command_exists docker; then
        echo -e "${GREEN}  ✓ Docker installed successfully${NC}"
        
        # Start Docker service
        $SUDO systemctl start docker
        $SUDO systemctl enable docker
        
        # Add user to docker group
        if ! is_root; then
            $SUDO usermod -aG docker $USER
            echo -e "${YELLOW}  ⚠ Added $USER to docker group. Please log out and back in for changes to take effect.${NC}"
        fi
    else
        echo -e "${RED}  ✗ Docker installation failed${NC}"
        HAS_ERRORS=true
    fi
fi
echo ""

# 5. Check/Install OpenSSL
echo -e "${YELLOW}[5/7] Checking OpenSSL...${NC}"
if command_exists openssl; then
    OPENSSL_VERSION=$(openssl version)
    echo -e "${GREEN}  ✓ OpenSSL is installed: $OPENSSL_VERSION${NC}"
else
    echo -e "${RED}  ✗ OpenSSL is not installed${NC}"
    echo -e "${YELLOW}  Installing OpenSSL...${NC}"
    case $DISTRO in
        ubuntu|debian)
            $SUDO apt install -y openssl
            ;;
        fedora|rhel|centos)
            $SUDO yum install -y openssl || $SUDO dnf install -y openssl
            ;;
        arch|manjaro)
            $SUDO pacman -S --noconfirm openssl
            ;;
        *)
            echo -e "${RED}  ✗ Cannot install OpenSSL automatically${NC}"
            HAS_ERRORS=true
            ;;
    esac
    
    if command_exists openssl; then
        echo -e "${GREEN}  ✓ OpenSSL installed successfully${NC}"
    else
        echo -e "${RED}  ✗ OpenSSL installation failed${NC}"
        HAS_ERRORS=true
    fi
fi
echo ""

# 6. Check/Install curl (needed for Node.js installation)
echo -e "${YELLOW}[6/7] Checking curl...${NC}"
if command_exists curl; then
    echo -e "${GREEN}  ✓ curl is installed${NC}"
else
    echo -e "${RED}  ✗ curl is not installed${NC}"
    echo -e "${YELLOW}  Installing curl...${NC}"
    case $DISTRO in
        ubuntu|debian)
            $SUDO apt install -y curl
            ;;
        fedora|rhel|centos)
            $SUDO yum install -y curl || $SUDO dnf install -y curl
            ;;
        arch|manjaro)
            $SUDO pacman -S --noconfirm curl
            ;;
    esac
fi
echo ""

# 7. Check/Install Node.js and npm packages
echo -e "${YELLOW}[7/7] Checking Node.js (18+) and npm packages...${NC}"
if command_exists node; then
    NODE_VERSION=$(node --version)
    echo -e "${GREEN}  ✓ Node.js is installed: $NODE_VERSION${NC}"
    
    # Check npm packages
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
        if $SUDO npm install -g ws https-proxy-agent; then
            echo -e "${GREEN}  ✓ npm packages installed successfully${NC}"
        else
            echo -e "${RED}  ✗ Error installing npm packages${NC}"
            HAS_ERRORS=true
        fi
    else
        echo -e "${GREEN}  ✓ Required npm packages are installed${NC}"
    fi
else
    echo -e "${RED}  ✗ Node.js is not installed${NC}"
    echo -e "${YELLOW}  Installing Node.js LTS...${NC}"
    
    case $DISTRO in
        ubuntu|debian)
            curl -fsSL https://deb.nodesource.com/setup_lts.x | $SUDO -E bash -
            $SUDO apt install -y nodejs
            ;;
        fedora)
            curl -fsSL https://rpm.nodesource.com/setup_lts.x | $SUDO bash -
            $SUDO dnf install -y nodejs
            ;;
        centos|rhel)
            curl -fsSL https://rpm.nodesource.com/setup_lts.x | $SUDO bash -
            $SUDO yum install -y nodejs
            ;;
        arch|manjaro)
            $SUDO pacman -S --noconfirm nodejs npm
            ;;
        *)
            echo -e "${RED}  ✗ Cannot install Node.js automatically${NC}"
            echo -e "${YELLOW}    Please install manually from https://nodejs.org/${NC}"
            HAS_ERRORS=true
            ;;
    esac
    
    if command_exists node; then
        echo -e "${GREEN}  ✓ Node.js installed successfully${NC}"
        
        # Install npm packages
        echo -e "${YELLOW}  Installing required npm packages...${NC}"
        if $SUDO npm install -g ws https-proxy-agent; then
            echo -e "${GREEN}  ✓ npm packages installed successfully${NC}"
        else
            echo -e "${RED}  ✗ Error installing npm packages${NC}"
            HAS_ERRORS=true
        fi
    else
        echo -e "${RED}  ✗ Node.js installation failed${NC}"
        HAS_ERRORS=true
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
    echo "  2. If Docker was installed, log out and back in to apply docker group membership"
    echo "  3. Generate certificates: cd scripts && ./generate-certs.sh"
    echo "  4. Build the project: make build-linux"
    echo "  5. Run tests: ./scripts/test-local.sh"
    exit 0
fi
