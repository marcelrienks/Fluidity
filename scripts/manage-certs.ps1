# Fluidity Certificate Management Script (PowerShell)
# Generates development certificates and optionally saves them to AWS Secrets Manager
# Usage: .\scripts\manage-certs.ps1 [[-Command] <string>] [[-SecretsManager] <switch>] [[-SecretName] <string>] [[-CertsDir] <string>]

param(
    [string]$Command = "generate-and-save",
    [switch]$SecretsManager,
    [string]$SecretName = "fluidity/certificates",
    [string]$CertsDir = ".\certs",
    [int]$Days = 730
)

# ============================================================================
# UTILITY FUNCTIONS
# ============================================================================

function Show-Usage {
    @"
Fluidity Certificate Management Script

Usage: .\scripts\manage-certs.ps1 [OPTIONS]

Options:
  -Command                "generate" | "save" | "generate-and-save" (default)
  -SecretName NAME       AWS Secrets Manager secret name (default: fluidity/certificates)
  -CertsDir DIR          Directory for certificates (default: .\certs)
  -SecretsManager        Enable saving to AWS Secrets Manager
  -Days NUM              Certificate validity days (default: 730)

Examples:
  # Generate certificates only
  .\scripts\manage-certs.ps1 -Command generate

  # Generate and save to Secrets Manager
  .\scripts\manage-certs.ps1 -Command generate-and-save -SecretsManager

  # Just save existing certificates
  .\scripts\manage-certs.ps1 -Command save -SecretsManager

  # Custom secret name
  .\scripts\manage-certs.ps1 -SecretsManager -SecretName "my-app/certs"

"@
}

function Write-Success {
    param([string]$Message)
    Write-Host "✓ $Message" -ForegroundColor Green
}

function Write-Error-Custom {
    param([string]$Message)
    Write-Host "✗ $Message" -ForegroundColor Red
}

function Write-Info {
    param([string]$Message)
    Write-Host "  $Message" -ForegroundColor Cyan
}

function Write-Section {
    param([string]$Title)
    Write-Host ""
    Write-Host "═══════════════════════════════════════════════════════════" -ForegroundColor Yellow
    Write-Host "  $Title" -ForegroundColor Yellow
    Write-Host "═══════════════════════════════════════════════════════════" -ForegroundColor Yellow
    Write-Host ""
}

# ============================================================================
# CERTIFICATE GENERATION FUNCTIONS
# ============================================================================

function Generate-Certificates {
    Write-Section "Fluidity Certificate Generation"
    
    # Create certs directory if it doesn't exist
    if (!(Test-Path $CertsDir)) {
        New-Item -ItemType Directory -Path $CertsDir -Force | Out-Null
    }
    
    Write-Info "Certificates directory: $CertsDir"
    Write-Info "Validity period: $Days days"
    Write-Host ""
    
    try {
        # Generate CA private key
        Write-Host "1. Generating CA private key..." -ForegroundColor Yellow
        & openssl genrsa -out "$CertsDir\ca.key" 4096 2>&1 | Out-Null
        
        # Generate CA certificate
        Write-Host "2. Generating CA certificate..." -ForegroundColor Yellow
        & openssl req -new -x509 -days $Days -key "$CertsDir\ca.key" -out "$CertsDir\ca.crt" `
            -subj "/C=US/ST=CA/L=SF/O=Fluidity/OU=Dev/CN=Fluidity-CA" 2>&1 | Out-Null
        
        # Generate server private key
        Write-Host "3. Generating server private key..." -ForegroundColor Yellow
        & openssl genrsa -out "$CertsDir\server.key" 4096 2>&1 | Out-Null
        
        # Generate server certificate signing request
        Write-Host "4. Generating server CSR..." -ForegroundColor Yellow
        & openssl req -new -key "$CertsDir\server.key" -out "$CertsDir\server.csr" `
            -subj "/C=US/ST=CA/L=SF/O=Fluidity/OU=Server/CN=fluidity-server" 2>&1 | Out-Null
        
        # Create server certificate extensions file
        Write-Host "5. Creating server extensions file..." -ForegroundColor Yellow
        $ServerExtContent = @"
[v3_req]
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = fluidity-server
DNS.2 = localhost
DNS.3 = host.docker.internal
IP.1 = 127.0.0.1
IP.2 = ::1
"@
        $ServerExtContent | Out-File -FilePath "$CertsDir\server.ext" -Encoding ascii -NoNewline
        
        # Generate server certificate signed by CA
        Write-Host "6. Generating server certificate..." -ForegroundColor Yellow
        & openssl x509 -req -in "$CertsDir\server.csr" -CA "$CertsDir\ca.crt" -CAkey "$CertsDir\ca.key" `
            -CAcreateserial -out "$CertsDir\server.crt" -days $Days `
            -extensions v3_req -extfile "$CertsDir\server.ext" 2>&1 | Out-Null
        
        # Generate client private key
        Write-Host "7. Generating client private key..." -ForegroundColor Yellow
        & openssl genrsa -out "$CertsDir\client.key" 4096 2>&1 | Out-Null
        
        # Generate client certificate signing request
        Write-Host "8. Generating client CSR..." -ForegroundColor Yellow
        & openssl req -new -key "$CertsDir\client.key" -out "$CertsDir\client.csr" `
            -subj "/C=US/ST=CA/L=SF/O=Fluidity/OU=Client/CN=fluidity-client" 2>&1 | Out-Null
        
        # Create client certificate extensions file
        Write-Host "9. Creating client extensions file..." -ForegroundColor Yellow
        $ClientExtContent = @"
[v3_req]
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
extendedKeyUsage = clientAuth
"@
        $ClientExtContent | Out-File -FilePath "$CertsDir\client.ext" -Encoding ascii -NoNewline
        
        # Generate client certificate signed by CA
        Write-Host "10. Generating client certificate..." -ForegroundColor Yellow
        & openssl x509 -req -in "$CertsDir\client.csr" -CA "$CertsDir\ca.crt" -CAkey "$CertsDir\ca.key" `
            -CAcreateserial -out "$CertsDir\client.crt" -days $Days `
            -extensions v3_req -extfile "$CertsDir\client.ext" 2>&1 | Out-Null
        
        # Clean up temporary files
        Remove-Item "$CertsDir\server.csr", "$CertsDir\server.ext", "$CertsDir\client.csr", "$CertsDir\client.ext" -ErrorAction SilentlyContinue
        
        Write-Host ""
        Write-Success "Certificates generated successfully!"
        Write-Host ""
        Write-Host "Files created:" -ForegroundColor Cyan
        Write-Info "$CertsDir\ca.crt       - CA certificate"
        Write-Info "$CertsDir\ca.key       - CA private key"
        Write-Info "$CertsDir\server.crt   - Server certificate"
        Write-Info "$CertsDir\server.key   - Server private key"
        Write-Info "$CertsDir\client.crt   - Client certificate"
        Write-Info "$CertsDir\client.key   - Client private key"
        Write-Host ""
        Write-Host "⚠️  WARNING: These certificates are for development only!" -ForegroundColor Red
        Write-Host "For production, use properly signed certificates from a trusted CA." -ForegroundColor Red
        Write-Host ""
        
        return $true
    } catch {
        if ($_.Exception.Message -like "*openssl*" -or $_.Exception.Message -like "*not found*") {
            Write-Error-Custom "OpenSSL not found!"
            Write-Host "Please install OpenSSL and ensure it's in your PATH." -ForegroundColor Yellow
            Write-Host "You can install OpenSSL from: https://slproweb.com/products/Win32OpenSSL.html" -ForegroundColor Yellow
            Write-Host "Or install via Chocolatey: choco install openssl" -ForegroundColor Yellow
        } else {
            Write-Error-Custom "Error generating certificates: $($_.Exception.Message)"
        }
        return $false
    }
}

# ============================================================================
# SECRETS MANAGER FUNCTIONS
# ============================================================================

function Save-To-SecretsManager {
    Write-Section "Saving Certificates to AWS Secrets Manager"
    
    Write-Info "Secret name: $SecretName"
    Write-Info "Certificate file: $CertsDir\server.crt"
    Write-Info "Key file: $CertsDir\server.key"
    Write-Info "CA file: $CertsDir\ca.crt"
    Write-Host ""
    
    # Check files exist
    if (!(Test-Path "$CertsDir\server.crt")) {
        Write-Error-Custom "Certificate file not found: $CertsDir\server.crt"
        return $false
    }
    
    if (!(Test-Path "$CertsDir\server.key")) {
        Write-Error-Custom "Key file not found: $CertsDir\server.key"
        return $false
    }
    
    if (!(Test-Path "$CertsDir\ca.crt")) {
        Write-Error-Custom "CA file not found: $CertsDir\ca.crt"
        return $false
    }
    
    try {
        # Read and base64 encode the certificates
        Write-Host "Encoding certificates..." -ForegroundColor Yellow
        $CertContent = [System.IO.File]::ReadAllBytes("$CertsDir\server.crt")
        $KeyContent = [System.IO.File]::ReadAllBytes("$CertsDir\server.key")
        $CaContent = [System.IO.File]::ReadAllBytes("$CertsDir\ca.crt")
        
        $CertPem = [Convert]::ToBase64String($CertContent)
        $KeyPem = [Convert]::ToBase64String($KeyContent)
        $CaPem = [Convert]::ToBase64String($CaContent)
        
        # Create the JSON secret
        $SecretObject = @{
            cert_pem = $CertPem
            key_pem = $KeyPem
            ca_pem = $CaPem
        } | ConvertTo-Json -Compress
        
        # Try to create/update the secret
        Write-Host "Contacting AWS Secrets Manager..." -ForegroundColor Yellow
        
        $existing = aws secretsmanager describe-secret --secret-id $SecretName 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Host "Secret exists, updating..." -ForegroundColor Yellow
            aws secretsmanager update-secret `
                --secret-id $SecretName `
                --secret-string $SecretObject | Out-Null
            Write-Success "Secret updated successfully!"
        } else {
            Write-Host "Secret does not exist, creating..." -ForegroundColor Yellow
            aws secretsmanager create-secret `
                --name $SecretName `
                --secret-string $SecretObject | Out-Null
            Write-Success "Secret created successfully!"
        }
        
        Write-Host ""
        return $true
    } catch {
        Write-Error-Custom "Error saving to Secrets Manager: $($_.Exception.Message)"
        return $false
    }
}

# ============================================================================
# MAIN
# ============================================================================

# Check if OpenSSL is available
try {
    $null = Get-Command openssl -ErrorAction Stop
} catch {
    Write-Error-Custom "OpenSSL not found!"
    Write-Host "Please install OpenSSL and ensure it's in your PATH." -ForegroundColor Yellow
    Write-Host "You can install OpenSSL from: https://slproweb.com/products/Win32OpenSSL.html" -ForegroundColor Yellow
    Write-Host "Or install via Chocolatey: choco install openssl" -ForegroundColor Yellow
    exit 1
}

# Check if AWS CLI is available (if using Secrets Manager)
if ($SecretsManager) {
    try {
        $null = Get-Command aws -ErrorAction Stop
    } catch {
        Write-Error-Custom "AWS CLI not found!"
        Write-Host "Please install the AWS CLI and ensure it's in your PATH." -ForegroundColor Yellow
        Write-Host "Visit: https://aws.amazon.com/cli/" -ForegroundColor Yellow
        exit 1
    }
}

# Process commands
switch ($Command.ToLower()) {
    "generate" {
        if (-not (Generate-Certificates)) {
            exit 1
        }
    }
    "save" {
        if (-not $SecretsManager) {
            Write-Error-Custom "Secrets Manager not enabled. Use -SecretsManager flag."
            exit 1
        }
        if (-not (Save-To-SecretsManager)) {
            exit 1
        }
    }
    "generate-and-save" {
        if (-not (Generate-Certificates)) {
            exit 1
        }
        if ($SecretsManager) {
            if (-not (Save-To-SecretsManager)) {
                exit 1
            }
        }
    }
    default {
        Write-Error-Custom "Unknown command: $Command"
        Show-Usage
        exit 1
    }
}

# Show configuration instructions if saving to secrets
if ($SecretsManager -and ($Command -eq "save" -or $Command -eq "generate-and-save")) {
    Write-Host "Configuration for Fluidity:" -ForegroundColor Cyan
    Write-Info "use_secrets_manager: true"
    Write-Info "secrets_manager_name: $SecretName"
    Write-Host ""
}

Write-Section "Certificate management complete!"
