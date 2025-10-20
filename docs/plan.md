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


## Phase 2: Cloud Deployment

### Deployment & Scaling
- [ ] Deploy Tunnel Server to chosen cloud provider (AWS, Azure, GCP, etc.)
- [ ] Implement container orchestration (Docker Compose, Kubernetes, or cloud container services)
- [ ] Set up CI/CD pipeline for automated builds and deployments
- [ ] Implement scaling and monitoring for cloud deployment

### Configuration & Networking
- [ ] Handle dynamic IP address changes for cloud deployments
- [ ] Secure cloud networking and firewall rules

---

## Phase 3: Enhanced Features

### Security & Certificates
- [ ] Implement advanced mTLS authentication and certificate management
- [ ] Set up private Certificate Authority (CA) infrastructure
- [ ] Implement certificate generation and distribution system
- [ ] Plan for certificate renewal and rotation

### Configuration & Usability
- [ ] Add advanced configuration options (CLI, environment variables, config files)
- [ ] Improve persistent configuration storage and update mechanisms

### Monitoring & Logging
- [ ] Add monitoring endpoints and health checks
- [ ] Enhance logging capabilities (structured, contextual, privacy-focused)

---

## Phase 4: Testing & Documentation

### Testing
- [ ] Comprehensive unit, integration, and performance tests
- [ ] End-to-end test scenarios with real servers
- [ ] Security audit and vulnerability assessment

### Documentation
- [ ] Complete user documentation and guides
- [ ] Finalize troubleshooting and FAQ sections
- [ ] Document cloud deployment and scaling procedures

---

---

## References
- [Architecture Design](architecture.md)
- [Product Requirements Document (PRD)](PRD.md)

For future phases and a full requirements list, see the PRD and architecture documents.
