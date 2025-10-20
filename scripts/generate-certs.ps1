# Certificate generation script for development (PowerShell version)
# This script generates a self-signed CA and client/server certificates for mTLS

param(
    [string]$CertsDir = ".\certs",
    [int]$Days = 730
)

# Create certs directory if it doesn't exist
if (!(Test-Path $CertsDir)) {
    New-Item -ItemType Directory -Path $CertsDir -Force | Out-Null
}

Write-Host "Generating development certificates..." -ForegroundColor Green

try {
    # Check if OpenSSL is available
    $null = Get-Command openssl -ErrorAction Stop
    
    Write-Host "1. Generating CA private key..." -ForegroundColor Yellow
    & openssl genrsa -out "$CertsDir\ca.key" 4096
    
    Write-Host "2. Generating CA certificate..." -ForegroundColor Yellow
    & openssl req -new -x509 -days $Days -key "$CertsDir\ca.key" -out "$CertsDir\ca.crt" -subj "/C=US/ST=CA/L=SF/O=Fluidity/OU=Dev/CN=Fluidity-CA"
    
    Write-Host "3. Generating server private key..." -ForegroundColor Yellow
    & openssl genrsa -out "$CertsDir\server.key" 4096
    
    Write-Host "4. Generating server CSR..." -ForegroundColor Yellow
    & openssl req -new -key "$CertsDir\server.key" -out "$CertsDir\server.csr" -subj "/C=US/ST=CA/L=SF/O=Fluidity/OU=Server/CN=fluidity-server"
    
    # Create server certificate extensions file
    Write-Host "5. Creating server extensions file..." -ForegroundColor Yellow
    @"
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = fluidity-server
DNS.2 = localhost
IP.1 = 127.0.0.1
IP.2 = ::1
"@ | Out-File -FilePath "$CertsDir\server.ext" -Encoding ascii
    
    Write-Host "6. Generating server certificate..." -ForegroundColor Yellow
    & openssl x509 -req -in "$CertsDir\server.csr" -CA "$CertsDir\ca.crt" -CAkey "$CertsDir\ca.key" -CAcreateserial -out "$CertsDir\server.crt" -days $Days -extfile "$CertsDir\server.ext"
    
    Write-Host "7. Generating client private key..." -ForegroundColor Yellow
    & openssl genrsa -out "$CertsDir\client.key" 4096
    
    Write-Host "8. Generating client CSR..." -ForegroundColor Yellow
    & openssl req -new -key "$CertsDir\client.key" -out "$CertsDir\client.csr" -subj "/C=US/ST=CA/L=SF/O=Fluidity/OU=Client/CN=fluidity-client"
    
    # Create client certificate extensions file
    Write-Host "9. Creating client extensions file..." -ForegroundColor Yellow
    @"
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
extendedKeyUsage = clientAuth
"@ | Out-File -FilePath "$CertsDir\client.ext" -Encoding ascii
    
    Write-Host "10. Generating client certificate..." -ForegroundColor Yellow
    & openssl x509 -req -in "$CertsDir\client.csr" -CA "$CertsDir\ca.crt" -CAkey "$CertsDir\ca.key" -CAcreateserial -out "$CertsDir\client.crt" -days $Days -extfile "$CertsDir\client.ext"
    
    # Clean up temporary files
    Remove-Item "$CertsDir\server.csr", "$CertsDir\server.ext", "$CertsDir\client.csr", "$CertsDir\client.ext" -ErrorAction SilentlyContinue
    
    Write-Host "Certificates generated successfully!" -ForegroundColor Green
    Write-Host ""
    Write-Host "Files created:" -ForegroundColor Cyan
    Write-Host "  $CertsDir\ca.crt       - CA certificate"
    Write-Host "  $CertsDir\ca.key       - CA private key"
    Write-Host "  $CertsDir\server.crt   - Server certificate"
    Write-Host "  $CertsDir\server.key   - Server private key"
    Write-Host "  $CertsDir\client.crt   - Client certificate"
    Write-Host "  $CertsDir\client.key   - Client private key"
    Write-Host ""
    Write-Host "Note: These certificates are for development only!" -ForegroundColor Red
    Write-Host "For production, use properly signed certificates from a trusted CA." -ForegroundColor Red
    
} catch {
    if ($_.Exception.Message -like "*openssl*") {
        Write-Host "Error: OpenSSL not found!" -ForegroundColor Red
        Write-Host "Please install OpenSSL and ensure it's in your PATH." -ForegroundColor Yellow
        Write-Host "You can install OpenSSL from: https://slproweb.com/products/Win32OpenSSL.html" -ForegroundColor Yellow
        Write-Host "Or install via Chocolatey: choco install openssl" -ForegroundColor Yellow
    } else {
        Write-Host "Error generating certificates: $($_.Exception.Message)" -ForegroundColor Red
    }
    exit 1
}