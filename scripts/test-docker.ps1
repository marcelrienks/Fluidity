# End-to-End Docker Test Script
param(
    [switch]$SkipBuild,
    [switch]$KeepContainers,
    [string]$TestUrl = "http://httpbin.org/get"
)

$ErrorActionPreference = "Stop"

# Configuration
$scriptRoot = if ($PSScriptRoot) { $PSScriptRoot } else { (Get-Location).Path }
$projectRoot = Split-Path -Parent $scriptRoot
$serverImage = "fluidity-server:test"
$agentImage = "fluidity-agent:test"
$networkName = "fluidity-test-net"
$serverContainer = "fluidity-server"  # Must match agent.docker.yaml server_ip
$agentContainer = "fluidity-agent"
$proxyPort = 8081

Write-Host "`n=== Fluidity End-to-End Docker Test ===" -ForegroundColor Magenta

try {
    # Step 1: Build
    if (-not $SkipBuild) {
        Write-Host "`n[Step 1] Building Linux binaries" -ForegroundColor Yellow
        Set-Location $projectRoot
        
        $env:GOOS = "linux"
        $env:GOARCH = "amd64"
        $env:CGO_ENABLED = "0"
        
        Write-Host "  Building server..." -ForegroundColor Cyan
        go build -o build/fluidity-server ./cmd/server
        if ($LASTEXITCODE -ne 0) { throw "Server build failed" }
        
        Write-Host "  Building agent..." -ForegroundColor Cyan
        go build -o build/fluidity-agent ./cmd/agent
        if ($LASTEXITCODE -ne 0) { throw "Agent build failed" }
        
        Write-Host "  [OK] Binaries built" -ForegroundColor Green
    } else {
        Write-Host "`n[Step 1] Skipping build" -ForegroundColor Yellow
    }

    # Step 2: Docker images
    Write-Host "`n[Step 2] Building Docker images" -ForegroundColor Yellow
    docker build -q -t $serverImage -f deployments/server/Dockerfile.local .
    docker build -q -t $agentImage -f deployments/agent/Dockerfile.local .
    Write-Host "  [OK] Images built" -ForegroundColor Green

    # Step 3: Network
    Write-Host "`n[Step 3] Setting up network" -ForegroundColor Yellow
    $netExists = docker network ls --filter "name=^${networkName}$" --format "{{.Name}}" | Select-Object -First 1
    if (-not $netExists) {
        docker network create $networkName 2>&1 | Out-Null
    }
    Write-Host "  [OK] Network ready" -ForegroundColor Green

    # Step 4: Cleanup old
    Write-Host "`n[Step 4] Cleaning old containers" -ForegroundColor Yellow
    docker ps -aq --filter "name=$serverContainer" | ForEach-Object { docker rm -f $_ } 2>&1 | Out-Null
    docker ps -aq --filter "name=$agentContainer" | ForEach-Object { docker rm -f $_ } 2>&1 | Out-Null
    Write-Host "  [OK] Cleaned" -ForegroundColor Green

    # Step 5: Start server
    Write-Host "`n[Step 5] Starting server" -ForegroundColor Yellow
    docker run -d `
        --name $serverContainer `
        --network $networkName `
        -v "${projectRoot}/configs:/root/configs:ro" `
        -v "${projectRoot}/certs:/root/certs:ro" `
        $serverImage `
        --config configs/server.docker.yaml 2>&1 | Out-Null
    
    if ($LASTEXITCODE -ne 0) { throw "Failed to start server" }
    Write-Host "  [OK] Server started" -ForegroundColor Green

    # Step 6: Start agent
    Write-Host "`n[Step 6] Starting agent" -ForegroundColor Yellow
    docker run -d `
        --name $agentContainer `
        --network $networkName `
        -p "${proxyPort}:8080" `
        -v "${projectRoot}/configs:/root/configs:ro" `
        -v "${projectRoot}/certs:/root/certs:ro" `
        $agentImage `
        --config configs/agent.docker.yaml 2>&1 | Out-Null
    
    if ($LASTEXITCODE -ne 0) { throw "Failed to start agent" }
    Write-Host "  [OK] Agent started" -ForegroundColor Green

    # Step 7: Wait
    Write-Host "`n[Step 7] Waiting for initialization..." -ForegroundColor Yellow
    Start-Sleep -Seconds 3
    Write-Host "  [OK] Ready" -ForegroundColor Green

    # Step 8: Test
    Write-Host "`n[Step 8] Testing HTTP tunnel" -ForegroundColor Yellow
    Write-Host "  URL: $TestUrl" -ForegroundColor Cyan
    
    $response = curl.exe -x "http://127.0.0.1:$proxyPort" -s -m 10 -w "`nHTTP_CODE:%{http_code}`n" "$TestUrl"
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

    # Step 9: Additional tests
    Write-Host "`n[Step 9] Additional tests" -ForegroundColor Yellow
    
    # Track failures for final verdict
    $testFailures = @()
    
    # Test HTTPS via CONNECT
    $ghResponse = curl.exe -x "http://127.0.0.1:$proxyPort" -s -m 10 -w "`nHTTP_CODE:%{http_code}`n" "https://api.github.com" 2>&1
    $ghCodeMatch = $ghResponse | Select-String "HTTP_CODE:(\d+)"
    if ($ghCodeMatch) {
        $ghCode = $ghCodeMatch.Matches[0].Groups[1].Value
        if ($ghCode -eq "200") {
            Write-Host "  GitHub API (HTTPS): HTTP $ghCode" -ForegroundColor Green
        } else {
            Write-Host "  GitHub API (HTTPS): HTTP $ghCode" -ForegroundColor Red
            $testFailures += "GitHub API returned HTTP $ghCode"
        }
    } else {
        Write-Host "  GitHub API (HTTPS): Connection failed" -ForegroundColor Red
        $testFailures += "GitHub API connection failed"
    }
    
    # Test HTTP
    $exResponse = curl.exe -x "http://127.0.0.1:$proxyPort" -s -m 10 -w "`nHTTP_CODE:%{http_code}`n" "http://example.com" 2>&1
    $exCodeMatch = $exResponse | Select-String "HTTP_CODE:(\d+)"
    if ($exCodeMatch) {
        $exCode = $exCodeMatch.Matches[0].Groups[1].Value
        if ($exCode -eq "200") {
            Write-Host "  example.com (HTTP): HTTP $exCode" -ForegroundColor Green
        } else {
            Write-Host "  example.com (HTTP): HTTP $exCode" -ForegroundColor Red
            $testFailures += "example.com returned HTTP $exCode"
        }
    } else {
        Write-Host "  example.com (HTTP): Connection failed" -ForegroundColor Red
        $testFailures += "example.com connection failed"
    }
    
    # Fail if any additional tests failed
    if ($testFailures.Count -gt 0) {
        throw "Additional tests failed: $($testFailures -join ', ')"
    }

    # Success
    Write-Host "`n========================================" -ForegroundColor Green
    Write-Host "  ALL TESTS PASSED!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    
} catch {
    Write-Host "`n========================================" -ForegroundColor Red
    Write-Host "  TEST FAILED: $_" -ForegroundColor Red
    Write-Host "========================================" -ForegroundColor Red
    
    Write-Host "`nServer logs:" -ForegroundColor Yellow
    docker logs $serverContainer --tail 10 2>&1
    
    Write-Host "`nAgent logs:" -ForegroundColor Yellow
    docker logs $agentContainer --tail 10 2>&1
    
    exit 1
} finally {
    if (-not $KeepContainers) {
        Write-Host "`nCleaning up test containers..." -ForegroundColor Yellow
        docker rm -f $serverContainer 2>&1 | Out-Null
        docker rm -f $agentContainer 2>&1 | Out-Null
        Write-Host "  [OK] Cleanup complete" -ForegroundColor Cyan
    } else {
        Write-Host "`nContainers kept for manual testing:" -ForegroundColor Cyan
        Write-Host "  docker logs -f $serverContainer"
        Write-Host "  docker logs -f $agentContainer"
        Write-Host "  curl.exe -x http://127.0.0.1:$proxyPort http://httpbin.org/get"
    }
}
