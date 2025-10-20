# Stop packet capture
Write-Host "Stopping packet capture..." -ForegroundColor Yellow
pktmon stop

Write-Host "`nConverting capture to text format..." -ForegroundColor Yellow
pktmon format .\logs\tls-capture.etl -o .\logs\tls-capture.txt -f text

Write-Host "`nConverting capture to pcapng format for Wireshark..." -ForegroundColor Yellow
pktmon format .\logs\tls-capture.etl -o .\logs\tls-capture.pcapng -f pcapng

Write-Host "`nCapture files created:" -ForegroundColor Green
Write-Host "  - .\logs\tls-capture.txt (text format)"
Write-Host "  - .\logs\tls-capture.pcapng (Wireshark format)"

Write-Host "`nShowing packets on port 8443..." -ForegroundColor Yellow
Get-Content .\logs\tls-capture.txt | Select-String -Pattern "8443" -Context 2,2 | Select-Object -First 50

Write-Host "`nPress any key to exit..."
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")
