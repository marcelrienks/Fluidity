# Product Requirements Document (PRD)
# Fluidity - HTTP Tunnel Solution

---

## 1. Executive Summary

### 1.1 Project Vision
Fluidity aims to provide a robust, secure HTTP tunneling solution that enables users to bypass restrictive corporate firewalls while maintaining security and compliance standards.

### 1.2 Problem Statement
Many organizations implement restrictive firewall policies that block access to legitimate websites and services, impacting productivity and limiting access to necessary resources for work and personal development.

### 1.3 Solution Overview
A dual-component system consisting of:
- **Cloud-hosted Tunnel Server**: Go-based server deployed to a cloud provider
- **Local Tunnel Agent**: Containerized Go application running on Docker Desktop

---

## 2. Product Goals & Success Metrics

### 2.1 Primary Goals
- Enable secure HTTP traffic tunneling through restrictive firewalls
- Provide easy setup and configuration for end users
- Maintain high performance and reliability
- Ensure security and privacy of tunneled traffic

### 2.2 Success Metrics
- [ ] **Performance**: Successful bypass of corporate firewall restrictions (e.g., YouTube access)
- [ ] **Reliability**: Server can be started/stopped on-demand without issues
- [ ] **Security**: No unauthorized access to tunnel server
- [ ] **Usability**: CLI setup and operation straightforward for IT professionals

---

## 3. User Stories & Use Cases

### 3.1 Primary User Personas
**IT Professionals**: Technical users who need to bypass corporate firewall restrictions to access useful sites and resources required for their work.

### 3.2 Core Use Cases
1. **Site Access**: Bypassing corporate firewall blocking of useful sites (e.g., YouTube, educational content)
2. **Development Resources**: Accessing development tools, documentation, and technical resources
3. **Research & Learning**: Accessing educational content and technical resources blocked by corporate policies
4. **General Web Browsing**: Enabling full web browsing capabilities through corporate firewalls

---

## 4. Functional Requirements

### 4.1 Tunnel Server Requirements
- [x] **FR-001**: Accept and manage secure connections from tunnel agents (Oct 2025)
- [x] **FR-002**: Route HTTP requests to target destinations (Oct 2025)
- [x] **FR-003**: Return responses through the established tunnel (Oct 2025)
- [x] **FR-004**: Support single agent connection with multiple concurrent requests (Oct 2025)
- [x] **FR-005**: Implement authentication and authorization mechanisms (Oct 2025 - mTLS)
- [ ] **FR-006**: Provide connection monitoring and health checks
- [ ] **FR-033**: Emit CloudWatch metrics for active connections and last activity
- [ ] **FR-034**: Support configurable metrics emission interval

### 4.2 Tunnel Agent Requirements
- [x] **FR-007**: Establish secure connection to tunnel server
- [x] **FR-008**: Intercept local HTTP traffic (local proxy on port 8080)
- [x] **FR-009**: Forward requests through tunnel to server
- [x] **FR-010**: Return responses to local applications
- [x] **FR-011**: Provide configuration interface (CLI, config files, env vars)
- [x] **FR-012**: Support automatic reconnection on connection loss
- [x] **FR-022**: Handle server IP configuration with CLI override and persistent update capability
- [ ] **FR-035**: Call Wake Lambda on startup via API Gateway
- [ ] **FR-036**: Retry connection for configurable duration after wake
- [ ] **FR-037**: Call Kill Lambda on shutdown via API Gateway
- [ ] **FR-038**: Support API Gateway authentication (API key)

### 4.3 Lambda Control Plane Requirements
- [ ] **FR-039**: Wake Lambda checks ECS service state and sets desired count to 1 if needed
- [ ] **FR-040**: Sleep Lambda queries CloudWatch metrics and scales down if idle
- [ ] **FR-041**: Kill Lambda immediately sets desired count to 0 without validation
- [ ] **FR-042**: API Gateway provides HTTPS endpoints for Wake and Kill Lambdas
- [ ] **FR-043**: EventBridge triggers Sleep Lambda on configurable schedule (default every 5 minutes)
- [ ] **FR-044**: EventBridge triggers Kill Lambda on daily schedule (configurable time, default 11 PM UTC)
- [ ] **FR-045**: Wake Lambda returns service status (waking/already_running/starting)
- [ ] **FR-046**: Sleep Lambda implements configurable idle threshold (default 15 minutes)
- [ ] **FR-047**: All Lambdas have proper IAM permissions (least-privilege)
- [ ] **FR-048**: API Gateway supports API key authentication
- [ ] **FR-049**: CloudFormation template deploys complete Lambda control plane infrastructure

### 4.4 Security Requirements
- [x] **FR-013**: Implement end-to-end encryption for tunnel traffic (Oct 2025 - TLS 1.3)
- [x] **FR-014**: Support secure authentication to prevent unauthorized server access (Oct 2025 - mTLS)
- [x] **FR-015**: Implement mutual TLS (mTLS) authentication between agent and server (Oct 2025)
- [x] **FR-016**: Use private Certificate Authority (CA) for certificate management (Oct 2025)
- [x] **FR-017**: Single agent must have unique client certificate signed by private CA (Oct 2025)
- [x] **FR-018**: Server must validate client certificate against private CA (Oct 2025)
- [x] **FR-019**: Implement recommended security measures for public server deployment (Oct 2025)
- [x] **FR-020**: Minimal logging for debugging purposes only (startup, connections, endpoint routing) (Oct 2025)
- [x] **FR-021**: Support all protocols required for general web browsing (Oct 2025 - HTTP/HTTPS/WebSocket)

### 4.5 Testing Requirements
- [x] **FR-025**: Develop integration tests for HTTP tunneling functionality (Oct 2025 - test-docker.ps1/.sh)
- [x] **FR-026**: Develop integration tests for HTTPS CONNECT tunneling (Oct 2025 - test-docker.ps1/.sh)
- [x] **FR-027**: Implement end-to-end test scenarios with real servers (Oct 2025 - httpbin.org, github.com, example.com)
- [x] **FR-028**: Create automated test suite for CI/CD pipeline (Oct 2025 - test-local.ps1/.sh, test-docker.ps1/.sh)
- [x] **FR-029**: Test connection recovery and error handling scenarios (Oct 2025 - basic reconnection testing)
- [x] **FR-030**: Validate certificate validation and mTLS authentication flows (Oct 2025 - validated in tests)
- [ ] **FR-031**: Expand test coverage for edge cases and error scenarios (planned)
- [ ] **FR-032**: Implement performance and load testing (planned)

---

## 5. Technical Requirements

### 5.1 Architecture Requirements
- **Language**: Go (for both server and agent)
- **Containerization**: Docker for both components (portable between cloud providers)
- **Cloud Deployment**: Cloud-agnostic containerized deployment (AWS, Azure, or other providers)
- **Domain Setup**: Public IP address provided by cloud platform (no custom domain for security reasons)
- **Local Runtime**: Docker Desktop
- **Billing Model**: Pay-as-you-use cloud resources (start/stop as needed)

### 5.2 Performance Requirements
- [x] **NFR-001**: Support multiple concurrent connections for single personal agent (minimum 10-20 parallel requests) (Oct 2025)
- [x] **NFR-002**: Optimize for general web browsing performance with parallel resource loading (Oct 2025)
- [x] **NFR-003**: Support all protocols required for web browsing (HTTP, HTTPS, WebSocket) (Oct 2025)
- [x] **NFR-004**: Server startup time under 30 seconds for on-demand usage (Oct 2025 - typically <5 seconds)

### 5.3 Security Requirements
- [x] **NFR-005**: All tunnel traffic encrypted with TLS 1.3 or higher (Oct 2025)
- [x] **NFR-006**: Mutual TLS (mTLS) authentication with client certificates (Oct 2025)
- [x] **NFR-007**: Private Certificate Authority (CA) for certificate management (Oct 2025)
- [x] **NFR-008**: Certificate-based identity verification for single personal agent (Oct 2025)
- [x] **NFR-009**: Minimal logging for debugging purposes (startup, connections, endpoint routing - no sensitive data) (Oct 2025)
- [x] **NFR-010**: Standard security measures recommended for public deployment (Oct 2025)
- [x] **NFR-011**: Server accessible only via mTLS authenticated agent connections (Oct 2025)

### 5.4 Scalability Requirements
- [x] **NFR-012**: Support for on-demand server startup and shutdown (Oct 2025 - Docker-based)
- [ ] **NFR-013**: Pay-as-you-use resource optimization (planned for cloud deployment)
- [x] **NFR-014**: Fully containerized deployment ensuring complete cloud provider portability (Oct 2025)
- [x] **NFR-015**: Certificate management for single personal agent deployment (Oct 2025)
- [x] **NFR-016**: Handle dynamic IP address changes for on-demand server deployments (Oct 2025)
- [ ] **NFR-017**: Connection pooling and efficient resource management for concurrent requests (partially complete - needs optimization)
- [x] **NFR-018**: Server IP configuration with CLI override and persistent storage capabilities (Oct 2025)

---

## 6. User Experience Requirements

### 6.1 Setup & Configuration
- [x] **UX-001**: CLI-based installation and setup using Docker (Oct 2025)
- [x] **UX-002**: Configuration through command-line parameters and config files (including server IP with CLI override) (Oct 2025)
- [x] **UX-003**: Terminal-based services (no GUI required) (Oct 2025)
- [x] **UX-004**: Developer-level technical expertise assumed (Oct 2025)
- [x] **UX-005**: Simple start/stop commands for server and agent (Oct 2025)

### 6.2 Monitoring & Feedback
- [x] **UX-006**: Certificate-based setup with generated client certificates (Oct 2025)
- [x] **UX-007**: Simple certificate installation process for agents (Oct 2025)
- [x] **UX-008**: Terminal-based status indicators (Oct 2025)
- [x] **UX-009**: Minimal logging output for debugging (startup, connections, endpoint routing) (Oct 2025)
- [ ] **UX-010**: CLI-based health checks and diagnostics (planned)
- [x] **UX-011**: No GUI components required (Oct 2025)

---

## 7. Implementation Phases

### 7.1 Phase 1: Core Infrastructure ✅ COMPLETE
- ✅ Implement basic tunnel server in Go
- ✅ Implement basic tunnel agent in Go
- ✅ Establish secure communication protocol
- ✅ Create Docker containers for both components
- ✅ Implement HTTP and HTTPS CONNECT tunneling
- ✅ Implement mTLS authentication
- ✅ Implement WebSocket tunneling support
- ✅ Create automated testing scripts (Docker and local)
- ✅ Comprehensive documentation (README, Architecture, PRD)

### 7.2 Phase 2: Cloud Deployment 🚧 NOT STARTED
- [ ] Deploy server to chosen cloud provider (AWS/Azure/GCP)
- [ ] Configure cloud networking and firewall rules
- [ ] Implement monitoring and alerting
- [ ] Set up CI/CD pipeline for automated deployments
- [ ] Document cloud deployment procedures

### 7.2.5 Phase 2.5: Lambda Control Plane 🚧 NOT STARTED
- [ ] Implement Wake Lambda (Python) with ECS API integration
- [ ] Implement Sleep Lambda with CloudWatch metrics query
- [ ] Implement Kill Lambda with immediate shutdown
- [ ] Deploy API Gateway with Wake and Kill endpoints
- [ ] Configure EventBridge schedulers (periodic Sleep, daily Kill)
- [ ] Add agent lifecycle integration (wake on startup, kill on shutdown)
- [ ] Add server CloudWatch metrics emission
- [ ] Create CloudFormation template for Lambda infrastructure
- [ ] End-to-end testing with full lifecycle

### 7.3 Phase 3: Enhanced Features 🚧 PARTIALLY COMPLETE
- [x] WebSocket protocol support (Oct 2025)
- [ ] Advanced certificate management (rotation, monitoring, alerts)
- [ ] Enhanced configuration options (hot-reload, validation)
- [ ] Connection pooling optimization
- [ ] Circuit breaker pattern implementation
- [ ] Improved error handling and recovery

### 7.4 Phase 4: Security Hardening 🚧 NOT STARTED
- [ ] Security audit and vulnerability assessment
- [ ] Penetration testing
- [ ] Document security best practices

### 7.5 Phase 5: Testing & Documentation 🚧 PARTIALLY COMPLETE
- [x] Develop comprehensive integration tests for end-to-end functionality (Oct 2025)
- [x] Create automated test suite for CI/CD pipeline (Oct 2025)
- [x] Complete user documentation (Oct 2025)
- [ ] Expand test coverage for edge cases
- [ ] Performance and load testing
- [ ] API and protocol documentation
- [ ] Troubleshooting and FAQ expansion

---

## 8. Technical Specifications

### 8.1 Communication Protocol
**Implemented** - JSON-based envelope protocol over TLS 1.3 supporting:
- **HTTP Requests/Responses**: Standard HTTP method tunneling
- **CONNECT Method**: HTTPS tunneling with bidirectional data streams
- **WebSocket Protocol**: Full bidirectional WebSocket support with message framing

Protocol Details:
- Envelope-based message framing with type discrimination
- Request/Response correlation via unique IDs
- Support for concurrent multiplexed connections
- Efficient binary data transfer for WebSocket frames

### 8.2 Authentication Method
**Mutual TLS (mTLS)** - Certificate-based mutual authentication ensuring only agent and server can communicate with each other. Each agent will have a unique client certificate signed by a private Certificate Authority (CA), and the server will have its own certificate signed by the same CA.

### 8.3 Cloud Provider Details
**Cloud-Agnostic Deployment** - Fully containerized architecture enabling deployment to any cloud provider (AWS, Azure, GCP, or others). Server accessible via cloud-provided public IP address regardless of provider choice.

### 8.4 Logging Specification
**Minimal Debugging Logs** - Limited logging to essential debugging information only:

#### Server Logging:
- **Startup Events**: Server initialization, port binding, certificate loading
- **Connection Events**: Agent connections/disconnections (certificate CN only, no IPs)
- **Endpoint Routing**: Target endpoints being accessed (domain only, no full URLs or query parameters)
- **Error Events**: Connection failures, certificate validation errors, routing failures

#### Agent Logging:
- **Startup Events**: Agent initialization, server connection attempts, certificate loading
- **Connection Events**: Server connection status, reconnection attempts
- **Traffic Routing**: Local traffic interception and forwarding (destination domain only)
- **Error Events**: Connection failures, certificate issues, traffic routing errors

#### Security Considerations:
- **No sensitive data**: No user credentials, full URLs, query parameters, or POST data
- **No IP logging**: Client IP addresses not logged for privacy
- **Timestamp only**: Simple timestamp format for debugging correlation
- **Configurable levels**: ERROR, WARN, INFO, DEBUG levels

### 8.5 Certificate Management Strategy
**mTLS Certificate Lifecycle Management** - Simplified for single-user personal deployment:

### 8.6 Dynamic IP Configuration Strategy
**Server IP Configuration and Update Handling** - Handling cloud provider IP changes:

#### Initial Configuration:
- **Deployment-time Configuration**: Server IP address provided during agent deployment
- **Container Variables**: Server IP stored using containerized architecture best practices (environment variables)
- **Persistent Storage**: Configuration stored in Docker volumes or bind mounts
- **Default Behavior**: Agent uses stored/configured IP address on startup

#### CLI Override Mechanism:
- **Startup Parameter**: Agent accepts `--server-ip` or similar CLI parameter at startup
- **Priority Logic**: CLI-provided IP takes precedence over stored configuration
- **Connection Testing**: Agent tests connectivity to new IP before updating configuration
- **Fallback**: If CLI IP fails, agent attempts connection with stored IP

#### Configuration Update Process:
1. **Startup**: Agent attempts connection with stored IP (default behavior)
2. **CLI Override**: If `--server-ip` provided, use new IP address
3. **Connection Success**: If new IP connects successfully, update stored configuration
4. **Persistent Update**: New IP overwrites previously configured IP in persistent storage
5. **Error Logging**: Connection failures logged with clear error messages indicating IP connectivity issues

#### Configuration Storage:
- **Environment Variables**: `FLUIDITY_SERVER_IP` for default configuration
- **Config Files**: JSON/YAML configuration file with server endpoint
- **Docker Volumes**: Persistent storage for configuration updates
- **Container Best Practices**: Follow 12-factor app configuration principles

#### Error Handling:
- **Connection Failure**: Clear error message indicating server IP connectivity issues
- **DNS Resolution**: Handle both IP addresses and hostnames
- **Timeout Handling**: Configurable connection timeout for IP testing
- **Retry Logic**: Limited retry attempts before falling back or failing

#### Certificate Authority (CA) Setup:
- **Private Root CA**: Self-signed root certificate for personal Fluidity deployment
- **Single-purpose CA**: CA used only for this tunnel system
- **CA Key Security**: Root CA private key stored securely with personal files
- **Certificate Validity**: 2-year validity for both client and server certificates (personal use)

#### Certificate Generation Process:
- **Server Certificate**: Generated once during initial server setup
- **Single Client Certificate**: One certificate for personal agent use
- **Simple Identity**: Client certificate with basic personal identifier in CN
- **Certificate Storage**: Certificates embedded in Docker images or config files

#### Simplified Distribution:
- **Pre-generated Certificates**: Client certificate created during initial setup
- **Local Storage**: Certificates stored as local files or in Docker images
- **No Dynamic Generation**: Fixed certificates for personal use (no on-demand generation needed)

#### Certificate Lifecycle (Simplified):
- **Manual Monitoring**: Personal monitoring of certificate expiration (2-year cycle)
- **Manual Renewal**: Manual certificate regeneration before expiration
- **No Revocation Needed**: Single user, compromised certificates handled by regeneration
- **Local Backup**: Certificates backed up with personal configuration files

#### Security Considerations (Personal Use):
- **Key Protection**: Private keys stored securely in personal environment
- **Simple Validation**: Server validates single client certificate
- **Personal Access Control**: Only personal systems access certificates
- **Basic Audit**: Simple logging for personal debugging

---

## 9. Security & Compliance

### 9.1 Security Measures
- End-to-end encryption of all tunnel traffic using TLS 1.3
- Mutual TLS (mTLS) authentication with private Certificate Authority
- Unique client certificates for each agent instance
- Certificate-based identity verification and access control
- Regular security updates and vulnerability patches
- Secure certificate storage and management
- Network isolation and access controls

### 9.2 Compliance Considerations
- Users responsible for compliance with organizational policies
- Clear disclaimer about proper usage
- No logging of sensitive user data
- Optional audit logging for enterprise use

---

## 12. Appendices

### 12.1 Related Documents
- [README.md](../README.md) - Project overview and architecture
- [Architecture Design](./architecture.md) - Detailed technical architecture (TBD)
- [API Specification](./api.md) - Server-agent communication protocol (TBD)