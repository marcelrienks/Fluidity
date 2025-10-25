# Setup Prerequisites Script for Windows
# This script checks for and installs required prerequisites for Fluidity

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Fluidity Prerequisites Setup (Windows)" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$hasErrors = $false

# Function to check if running as administrator
function Test-Administrator {
    $currentUser = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
    return $currentUser.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

# Function to check if a command exists
function Test-Command {
    param($Command)
    try {
        if (Get-Command $Command -ErrorAction Stop) {
            return $true
        }
    }
    catch {
        return $false
    }
    return $false
}

# Check for Administrator privileges
if (-not (Test-Administrator)) {
    Write-Host "WARNING: Not running as Administrator. Some installations may fail." -ForegroundColor Yellow
    Write-Host "Consider running: Start-Process powershell -Verb RunAs -ArgumentList '-File ""$PSCommandPath""'" -ForegroundColor Yellow
    Write-Host ""
}

# 1. Check/Install Chocolatey
Write-Host "[1/6] Checking Chocolatey..." -ForegroundColor Yellow
if (Test-Command "choco") {
    $chocoVersion = (choco --version)
    Write-Host "  [OK] Chocolatey is installed: $chocoVersion" -ForegroundColor Green
}
else {
    Write-Host "  [MISSING] Chocolatey is not installed" -ForegroundColor Red
    Write-Host "  Installing Chocolatey..." -ForegroundColor Yellow
    try {
        Set-ExecutionPolicy Bypass -Scope Process -Force
        [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
        Invoke-Expression ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))
        
        # Refresh environment variables
        $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")
        
        if (Test-Command "choco") {
            Write-Host "  [OK] Chocolatey installed successfully" -ForegroundColor Green
        }
        else {
            Write-Host "  [FAIL] Chocolatey installation failed. Please install manually from https://chocolatey.org/install" -ForegroundColor Red
            $hasErrors = $true
        }
    }
    catch {
        Write-Host "  [FAIL] Error installing Chocolatey: $_" -ForegroundColor Red
        $hasErrors = $true
    }
}
Write-Host ""

# 2. Check/Install Go
Write-Host "[2/6] Checking Go (1.21+)..." -ForegroundColor Yellow
if (Test-Command "go") {
    $goVersion = (go version)
    Write-Host "  [OK] Go is installed: $goVersion" -ForegroundColor Green
}
else {
    Write-Host "  [MISSING] Go is not installed" -ForegroundColor Red
    Write-Host "  Installing Go via Chocolatey..." -ForegroundColor Yellow
    try {
        choco install golang -y
        
        # Refresh environment variables
        $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")
        
        if (Test-Command "go") {
            Write-Host "  [OK] Go installed successfully" -ForegroundColor Green
        }
        else {
            Write-Host "  [FAIL] Go installation failed. Please install manually from https://golang.org/dl/" -ForegroundColor Red
            $hasErrors = $true
        }
    }
    catch {
        Write-Host "  [FAIL] Error installing Go: $_" -ForegroundColor Red
        $hasErrors = $true
    }
}
Write-Host ""

# 3. Check/Install Make
Write-Host "[3/6] Checking Make..." -ForegroundColor Yellow
if (Test-Command "make") {
    $makeVersion = (make --version | Select-Object -First 1)
    Write-Host "  [OK] Make is installed: $makeVersion" -ForegroundColor Green
}
else {
    Write-Host "  [MISSING] Make is not installed" -ForegroundColor Red
    Write-Host "  Installing Make via Chocolatey..." -ForegroundColor Yellow
    try {
        choco install make -y
        
        # Refresh environment variables
        $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")
        
        if (Test-Command "make") {
            Write-Host "  [OK] Make installed successfully" -ForegroundColor Green
        }
        else {
            Write-Host "  [FAIL] Make installation failed. Consider installing manually or using build commands directly." -ForegroundColor Red
            Write-Host "    Note: Make is optional - you can run build commands manually." -ForegroundColor Yellow
        }
    }
    catch {
        Write-Host "  [FAIL] Error installing Make: $_" -ForegroundColor Red
        Write-Host "    Note: Make is optional - you can run build commands manually." -ForegroundColor Yellow
    }
}
Write-Host ""

# 4. Check/Install Docker Desktop
Write-Host "[4/6] Checking Docker..." -ForegroundColor Yellow
if (Test-Command "docker") {
    $dockerVersion = (docker --version)
    Write-Host "  [OK] Docker is installed: $dockerVersion" -ForegroundColor Green
}
else {
    Write-Host "  [MISSING] Docker is not installed" -ForegroundColor Red
    Write-Host "  Installing Docker Desktop via Chocolatey..." -ForegroundColor Yellow
    try {
        choco install docker-desktop -y
        
        Write-Host "  [WARNING] Docker Desktop installed. Please:" -ForegroundColor Yellow
        Write-Host "    1. Restart your computer" -ForegroundColor Yellow
        Write-Host "    2. Launch Docker Desktop" -ForegroundColor Yellow
        Write-Host "    3. Complete the Docker Desktop setup wizard" -ForegroundColor Yellow
    }
    catch {
        Write-Host "  [FAIL] Error installing Docker: $_" -ForegroundColor Red
        Write-Host "    Please install manually from https://www.docker.com/products/docker-desktop" -ForegroundColor Red
        $hasErrors = $true
    }
}
Write-Host ""

# 5. Check/Install OpenSSL
Write-Host "[5/6] Checking OpenSSL..." -ForegroundColor Yellow
if (Test-Command "openssl") {
    $opensslVersion = (openssl version)
    Write-Host "  [OK] OpenSSL is installed: $opensslVersion" -ForegroundColor Green
}
else {
    Write-Host "  [MISSING] OpenSSL is not installed" -ForegroundColor Red
    Write-Host "  Installing OpenSSL Light via Chocolatey..." -ForegroundColor Yellow
    try {
        choco install openssl.light -y
        
        # Refresh environment variables
        $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")
        
        if (Test-Command "openssl") {
            Write-Host "  [OK] OpenSSL installed successfully" -ForegroundColor Green
        }
        else {
            Write-Host "  [FAIL] OpenSSL installation failed. Please install manually." -ForegroundColor Red
            $hasErrors = $true
        }
    }
    catch {
        Write-Host "  [FAIL] Error installing OpenSSL: $_" -ForegroundColor Red
        $hasErrors = $true
    }
}
Write-Host ""

# 6. Check/Install Node.js and npm packages
Write-Host "[6/6] Checking Node.js (v18+), npm, and npm packages..." -ForegroundColor Yellow
$nodeInstalled = $false
$npmInstalled = $false
if (Test-Command "node") {
    $nodeVersion = (node --version)
    Write-Host "  [OK] Node.js is installed: $nodeVersion" -ForegroundColor Green
    $nodeInstalled = $true
} else {
    Write-Host "  [MISSING] Node.js is not installed" -ForegroundColor Red
}
if (Test-Command "npm") {
    $npmVersion = (npm --version)
    Write-Host "  [OK] npm is installed: $npmVersion" -ForegroundColor Green
    $npmInstalled = $true
} else {
    Write-Host "  [MISSING] npm is not installed" -ForegroundColor Red
}

if (-not $nodeInstalled -or -not $npmInstalled) {
    Write-Host "  Installing Node.js and npm via Chocolatey..." -ForegroundColor Yellow
    try {
        choco install nodejs -y
        # Refresh environment variables
        $env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path", "User")
        if (Test-Command "node") {
            $nodeVersion = (node --version)
            Write-Host "  [OK] Node.js installed successfully: $nodeVersion" -ForegroundColor Green
            $nodeInstalled = $true
        }
        if (Test-Command "npm") {
            $npmVersion = (npm --version)
            Write-Host "  [OK] npm installed successfully: $npmVersion" -ForegroundColor Green
            $npmInstalled = $true
        }
    } catch {
        Write-Host "  [FAIL] Error installing Node.js/npm: $_" -ForegroundColor Red
        $hasErrors = $true
    }
}

if ($nodeInstalled -and $npmInstalled) {
    # Check npm packages
    Write-Host "  Checking npm packages (ws, https-proxy-agent)..." -ForegroundColor Yellow
    $wsInstalled = $false
    $proxyInstalled = $false
    try {
        $npmList = npm list -g --depth=0 2>$null
        $wsInstalled = $npmList -match "ws@"
        $proxyInstalled = $npmList -match "https-proxy-agent@"
    } catch {
        # Ignore errors, will check individually
    }
    if (-not $wsInstalled -or -not $proxyInstalled) {
        Write-Host "  Installing required npm packages..." -ForegroundColor Yellow
        # First try with default SSL settings
        $installResult = npm install -g ws https-proxy-agent 2>&1
        # Check if installation failed due to SSL certificate issues
        if ($LASTEXITCODE -ne 0 -and $installResult -match "SELF_SIGNED_CERT_IN_CHAIN|CERT_|SSL_") {
            Write-Host "  [WARNING] SSL certificate error detected (likely corporate proxy)" -ForegroundColor Yellow
            Write-Host "  Retrying with SSL verification disabled..." -ForegroundColor Yellow
            try {
                npm install -g ws https-proxy-agent --strict-ssl=false
                if ($LASTEXITCODE -eq 0) {
                    Write-Host "  [OK] npm packages installed successfully (with SSL disabled)" -ForegroundColor Green
                } else {
                    Write-Host "  [FAIL] Error installing npm packages" -ForegroundColor Red
                    $hasErrors = $true
                }
            } catch {
                Write-Host "  [FAIL] Error installing npm packages: $_" -ForegroundColor Red
                $hasErrors = $true
            }
        } elseif ($LASTEXITCODE -ne 0) {
            Write-Host "  [FAIL] Error installing npm packages" -ForegroundColor Red
            $hasErrors = $true
        } else {
            Write-Host "  [OK] npm packages installed successfully" -ForegroundColor Green
        }
    } else {
        Write-Host "  [OK] Required npm packages are installed" -ForegroundColor Green
    }
}
Write-Host ""

# Summary
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Setup Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

if ($hasErrors) {
    Write-Host "[WARNING] Setup completed with errors. Please review the output above." -ForegroundColor Yellow
    Write-Host "  Some prerequisites may need to be installed manually." -ForegroundColor Yellow
    exit 1
}
else {
    Write-Host "[SUCCESS] All prerequisites are installed!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Next steps:" -ForegroundColor Cyan
    Write-Host "  1. Close and reopen your terminal to refresh environment variables" -ForegroundColor White
    Write-Host "  2. Generate certificates: cd scripts; .\generate-certs.ps1" -ForegroundColor White
    Write-Host "  3. Build the project: make build-win" -ForegroundColor White
    Write-Host "  4. Run tests: .\scripts\test-local.ps1" -ForegroundColor White
    exit 0
}
