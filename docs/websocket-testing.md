# WebSocket Testing Guide

This document explains how to test WebSocket functionality in Fluidity.

## Overview

Fluidity supports full WebSocket tunneling, including:
- WebSocket upgrade detection and handling
- Bidirectional message relay (client ↔ agent ↔ server ↔ target)
- Support for all WebSocket message types (text, binary, ping, pong, close)
- Proper connection lifecycle management

## Automatic Testing

The test scripts (`test-local.ps1`, `test-local.sh`, `test-docker.ps1`, `test-docker.sh`) automatically include WebSocket testing as an optional step.

### Prerequisites for WebSocket Tests

WebSocket tests require Node.js and two npm packages:

1. **Install Node.js** (if not already installed):
   - Download from: https://nodejs.org/
   - Verify: `node --version`

2. **Install required npm packages globally**:
   ```bash
   npm install -g ws https-proxy-agent
   ```

### Running Tests

**Local binaries:**
```powershell
# Windows
.\scripts\test-local.ps1

# Linux/macOS
./scripts/test-local.sh
```

**Docker containers:**
```powershell
# Windows
.\scripts\test-docker.ps1

# Linux/macOS
./scripts/test-docker.sh
```

### Test Behavior

- If Node.js is not installed, WebSocket tests are **skipped** (not failed)
- If npm packages are missing, WebSocket tests are **skipped** with installation instructions
- Core HTTP/HTTPS tests always run regardless of WebSocket test status
- WebSocket test uses `wss://echo.websocket.org/` as the test endpoint

## Manual WebSocket Testing

You can manually test WebSocket functionality using various methods:

### Method 1: Browser-based Test

1. Start Fluidity agent and server
2. Configure your browser to use `http://localhost:8080` as proxy
3. Visit a WebSocket test page (e.g., https://www.websocket.org/echo.html)
4. Connect to `wss://echo.websocket.org/`
5. Send messages and verify echoes are received

### Method 2: Node.js Script

Create a test script `ws-test.js`:

```javascript
const WebSocket = require('ws');
const HttpsProxyAgent = require('https-proxy-agent');

const proxyUrl = 'http://127.0.0.1:8080';
const wsUrl = 'wss://echo.websocket.org/';

const agent = new HttpsProxyAgent(proxyUrl);
const ws = new WebSocket(wsUrl, { agent });

ws.on('open', function() {
    console.log('WebSocket connected');
    ws.send('Hello WebSocket!');
});

ws.on('message', function(data) {
    console.log('Received:', data.toString());
    ws.close();
});

ws.on('error', function(err) {
    console.error('Error:', err.message);
});

ws.on('close', function() {
    console.log('WebSocket closed');
});
```

Run: `node ws-test.js`

### Method 3: Python Script

Using `websocket-client`:

```bash
pip install websocket-client
```

Create `ws-test.py`:

```python
import websocket

ws = websocket.WebSocket(http_proxy_host="127.0.0.1", http_proxy_port=8080)
ws.connect("wss://echo.websocket.org/")
print("WebSocket connected")

ws.send("Hello WebSocket!")
result = ws.recv()
print(f"Received: {result}")

ws.close()
print("WebSocket closed")
```

Run: `python ws-test.py`

### Method 4: Command-line Tools

Using `wscat` (Node.js):

```bash
npm install -g wscat
HTTP_PROXY=http://127.0.0.1:8080 wscat -c wss://echo.websocket.org/
```

## Troubleshooting

### WebSocket test times out

- Verify agent and server are running and connected
- Check firewall rules allow WebSocket traffic
- Ensure `echo.websocket.org` is accessible from your network
- Try with `--keep-containers` flag to inspect logs

### Connection refused

- Verify agent proxy is listening on correct port (default: 8080)
- Check agent successfully connected to server (check logs)

### Test skipped with "requires Node.js"

- Install Node.js from https://nodejs.org/
- Ensure `node` is in your PATH
- Re-run the test

### Test skipped with "requires ws and https-proxy-agent"

- Run: `npm install -g ws https-proxy-agent`
- Re-run the test

## WebSocket Protocol Support

Fluidity supports the full WebSocket protocol:

- **Message Types**: Text (1), Binary (2), Close (8), Ping (9), Pong (10)
- **Upgrade Mechanism**: Proper HTTP Upgrade header detection
- **Bidirectional**: Full duplex communication
- **Connection Management**: Graceful close with status codes
- **Error Handling**: Connection errors propagated correctly

## Test Coverage

Current WebSocket test coverage:

- ✅ WebSocket upgrade detection (Upgrade: websocket header)
- ✅ Secure WebSocket connections (wss://)
- ✅ Bidirectional message relay
- ✅ Echo service testing
- ⚠️ Manual testing required for:
  - Large message payloads
  - Long-lived connections
  - Ping/pong keep-alive
  - Connection close handling with various codes

## Next Steps

For comprehensive WebSocket testing:

1. Add Go-based WebSocket integration tests
2. Test various message sizes (small, medium, large)
3. Test long-lived connections with periodic messages
4. Test connection recovery scenarios
5. Add WebSocket-specific performance benchmarks
