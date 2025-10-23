# Local Binary Test Script
# Tests native binaries running directly on the host (no Docker)

param(
    [switch]$SkipBuild,
    [string]$TestUrl = "http://httpbin.org/get",
    [int]$ProxyPort = 8080,
    [int]$ServerPort = 8443
)

$ErrorActionPreference = "Stop"

# Configuration
$scriptRoot = if ($PSScriptRoot) { $PSScriptRoot } else { (Get-Location).Path }
$projectRoot = Split-Path -Parent $scriptRoot
$serverConfig = Join-Path $projectRoot "configs\server.local.yaml"
$agentConfig = Join-Path $projectRoot "configs\agent.local.yaml"
$serverBinary = Join-Path $projectRoot "build\fluidity-server.exe"
$agentBinary = Join-Path $projectRoot "build\fluidity-agent.exe"

# Process tracking
$serverProcess = $null
$agentProcess = $null

Write-Host "`n=== Fluidity Local Binary Test ===" -ForegroundColor Magenta

function cleanup {
    Write-Host "`nCleaning up processes..." -ForegroundColor Yellow
    
    if ($serverProcess -and -not $serverProcess.HasExited) {
        Write-Host "  Stopping server..." -ForegroundColor Cyan
        Stop-Process -Id $serverProcess.Id -Force -ErrorAction SilentlyContinue
    }
    
    if ($agentProcess -and -not $agentProcess.HasExited) {
        Write-Host "  Stopping agent..." -ForegroundColor Cyan
        Stop-Process -Id $agentProcess.Id -Force -ErrorAction SilentlyContinue
    }
    
    Write-Host "  [OK] Cleanup complete" -ForegroundColor Cyan
}

try {
    # Step 1: Build
    if (-not $SkipBuild) {
        Write-Host "`n[Step 1] Building Windows binaries" -ForegroundColor Yellow
        Set-Location $projectRoot
        
        Write-Host "  Building server..." -ForegroundColor Cyan
        go build -o build/fluidity-server.exe ./cmd/server
        if ($LASTEXITCODE -ne 0) { throw "Server build failed" }
        
        Write-Host "  Building agent..." -ForegroundColor Cyan
        go build -o build/fluidity-agent.exe ./cmd/agent
        if ($LASTEXITCODE -ne 0) { throw "Agent build failed" }
        
        Write-Host "  [OK] Binaries built" -ForegroundColor Green
    } else {
        Write-Host "`n[Step 1] Skipping build" -ForegroundColor Yellow
    }

    # Step 2: Verify files exist
    Write-Host "`n[Step 2] Verifying files" -ForegroundColor Yellow
    
    if (-not (Test-Path $serverBinary)) {
        throw "Server binary not found: $serverBinary"
    }
    if (-not (Test-Path $agentBinary)) {
        throw "Agent binary not found: $agentBinary"
    }
    if (-not (Test-Path $serverConfig)) {
        throw "Server config not found: $serverConfig"
    }
    if (-not (Test-Path $agentConfig)) {
        throw "Agent config not found: $agentConfig"
    }
    
    Write-Host "  [OK] All files verified" -ForegroundColor Green

    # Step 3: Check ports are available
    Write-Host "`n[Step 3] Checking ports" -ForegroundColor Yellow
    
    $serverPortInUse = Get-NetTCPConnection -LocalPort $ServerPort -ErrorAction SilentlyContinue
    if ($serverPortInUse) {
        throw "Port $ServerPort already in use"
    }
    
    $proxyPortInUse = Get-NetTCPConnection -LocalPort $ProxyPort -ErrorAction SilentlyContinue
    if ($proxyPortInUse) {
        throw "Port $ProxyPort already in use"
    }
    
    Write-Host "  [OK] Ports available" -ForegroundColor Green

    # Step 4: Start server
    Write-Host "`n[Step 4] Starting server" -ForegroundColor Yellow
    Write-Host "  Command: $serverBinary --config $serverConfig" -ForegroundColor Cyan
    
    $serverProcess = Start-Process -FilePath $serverBinary `
        -ArgumentList "--config", $serverConfig `
        -PassThru `
        -NoNewWindow `
        -RedirectStandardOutput (Join-Path $projectRoot "logs\test-server.out") `
        -RedirectStandardError (Join-Path $projectRoot "logs\test-server.err")
    
    if (-not $serverProcess) {
        throw "Failed to start server process"
    }
    
    Write-Host "  [OK] Server started (PID: $($serverProcess.Id))" -ForegroundColor Green

    # Step 5: Start agent
    Write-Host "`n[Step 5] Starting agent" -ForegroundColor Yellow
    Write-Host "  Command: $agentBinary --config $agentConfig" -ForegroundColor Cyan
    
    Start-Sleep -Seconds 2  # Give server time to start
    
    $agentProcess = Start-Process -FilePath $agentBinary `
        -ArgumentList "--config", $agentConfig `
        -PassThru `
        -NoNewWindow `
        -RedirectStandardOutput (Join-Path $projectRoot "logs\test-agent.out") `
        -RedirectStandardError (Join-Path $projectRoot "logs\test-agent.err")
    
    if (-not $agentProcess) {
        throw "Failed to start agent process"
    }
    
    Write-Host "  [OK] Agent started (PID: $($agentProcess.Id))" -ForegroundColor Green

    # Step 6: Wait for initialization
    Write-Host "`n[Step 6] Waiting for initialization..." -ForegroundColor Yellow
    Start-Sleep -Seconds 3
    
    # Check processes are still running
    if ($serverProcess.HasExited) {
        throw "Server process exited unexpectedly (exit code: $($serverProcess.ExitCode))"
    }
    if ($agentProcess.HasExited) {
        throw "Agent process exited unexpectedly (exit code: $($agentProcess.ExitCode))"
    }
    
    Write-Host "  [OK] Processes running" -ForegroundColor Green

    # Step 7: Test HTTP tunnel
    Write-Host "`n[Step 7] Testing HTTP tunnel" -ForegroundColor Yellow
    Write-Host "  URL: $TestUrl" -ForegroundColor Cyan
    
    $response = curl.exe -x "http://127.0.0.1:$ProxyPort" -s -w "`nHTTP_CODE:%{http_code}`n" "$TestUrl"
    $httpCodeMatch = $response | Select-String "HTTP_CODE:(\d+)"
    
    if ($httpCodeMatch) {
        $httpCode = $httpCodeMatch.Matches[0].Groups[1].Value
        if ($httpCode -eq "200") {
            Write-Host "  [OK] Test passed (HTTP $httpCode)" -ForegroundColor Green
        } else {
            throw "Test failed (HTTP $httpCode)"
        }
    } else {
        Write-Host "  Response:" -ForegroundColor Red
        $response | ForEach-Object { Write-Host "    $_" }
        throw "Could not extract HTTP code from response"
    }

    # Step 8: Additional tests
    Write-Host "`n[Step 8] Additional tests" -ForegroundColor Yellow
    
    $ghResponse = curl.exe -x "http://127.0.0.1:$ProxyPort" -s -w "`nHTTP_CODE:%{http_code}`n" "https://api.github.com" 2>&1
    $ghCodeMatch = $ghResponse | Select-String "HTTP_CODE:(\d+)"
    if ($ghCodeMatch) {
        $ghCode = $ghCodeMatch.Matches[0].Groups[1].Value
        if ($ghCode -eq "200") {
            Write-Host "  GitHub API: HTTP $ghCode" -ForegroundColor Green
        } else {
            Write-Host "  GitHub API: HTTP $ghCode" -ForegroundColor Yellow
        }
    }
    
    $exResponse = curl.exe -x "http://127.0.0.1:$ProxyPort" -s -w "`nHTTP_CODE:%{http_code}`n" "http://example.com" 2>&1
    $exCodeMatch = $exResponse | Select-String "HTTP_CODE:(\d+)"
    if ($exCodeMatch) {
        $exCode = $exCodeMatch.Matches[0].Groups[1].Value
        if ($exCode -eq "200") {
            Write-Host "  example.com: HTTP $exCode" -ForegroundColor Green
        } else {
            Write-Host "  example.com: HTTP $exCode" -ForegroundColor Yellow
        }
    }

    # Success
    Write-Host "`n========================================" -ForegroundColor Green
    Write-Host "  ALL TESTS PASSED!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    
    Write-Host "`nLog files:" -ForegroundColor Cyan
    Write-Host "  Server: logs\test-server.out" -ForegroundColor White
    Write-Host "  Agent:  logs\test-agent.out" -ForegroundColor White
    
} catch {
    Write-Host "`n========================================" -ForegroundColor Red
    Write-Host "  TEST FAILED: $_" -ForegroundColor Red
    Write-Host "========================================" -ForegroundColor Red
    
    if (Test-Path (Join-Path $projectRoot "logs\test-server.err")) {
        Write-Host "`nServer stderr:" -ForegroundColor Yellow
        Get-Content (Join-Path $projectRoot "logs\test-server.err") -Tail 10 -ErrorAction SilentlyContinue
    }
    
    if (Test-Path (Join-Path $projectRoot "logs\test-agent.err")) {
        Write-Host "`nAgent stderr:" -ForegroundColor Yellow
        Get-Content (Join-Path $projectRoot "logs\test-agent.err") -Tail 10 -ErrorAction SilentlyContinue
    }
    
    exit 1
} finally {
    cleanup
}
