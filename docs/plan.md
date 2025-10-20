# Fluidity Project Plan

This document outlines all outstanding work required for Phase 1, organized by phases and actionable steps. It replaces `PHASE1.md`.

---

## Phase 1: Core Infrastructure (Outstanding Work)

### Protocol Support
- [ ] Implement HTTPS/CONNECT tunneling
- [ ] Implement WebSocket support

### Security
- [ ] Automate development mTLS certificate generation and integration
- [ ] Replace insecure TLS verification (remove InsecureSkipVerify)

### Error Handling & Reliability
- [ ] Improve error handling and recovery throughout agent and server
- [ ] Implement circuit breaker pattern for external requests
- [ ] Enhance retry logic for connection attempts and request forwarding

### Testing
- [ ] Develop comprehensive integration tests for HTTP tunneling
- [ ] Prepare initial tests for HTTPS/CONNECT tunneling (once implemented)
- [ ] Test and analyze detection by endpoint security tools (EDR/AV)
- [ ] Implement mitigations to avoid triggering security monitoring alerts

### Documentation
- [ ] Finalize user guides and deployment instructions
- [ ] Expand troubleshooting and FAQ sections

### Cloud Deployment
- [ ] Deploy Tunnel Server to a cloud provider (initial deployment)

### Performance & Monitoring
- [ ] Optimize connection pooling and concurrent request handling
- [ ] Add health checks and monitoring endpoints

---

## Phase 1.5: Security & Integration Testing (Pre-Phase 2)
- [ ] Security tool/EDR testing and mitigation
- [ ] Comprehensive integration test suite
- [ ] Performance and stress testing

---

## References
- [Architecture Design](architecture.md)
- [Product Requirements Document (PRD)](PRD.md)

For future phases and a full requirements list, see the PRD and architecture documents.
