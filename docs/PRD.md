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
- [ ] **FR-001**: Accept and manage secure connections from tunnel agents
- [ ] **FR-002**: Route HTTP requests to target destinations
- [ ] **FR-003**: Return responses through the established tunnel
- [ ] **FR-004**: Support single agent connection with multiple concurrent requests
- [ ] **FR-005**: Implement authentication and authorization mechanisms
- [ ] **FR-006**: Provide connection monitoring and health checks

### 4.2 Tunnel Agent Requirements
- [ ] **FR-007**: Establish secure connection to tunnel server
- [ ] **FR-008**: Intercept local HTTP traffic
- [ ] **FR-009**: Forward requests through tunnel to server
- [ ] **FR-010**: Return responses to local applications
- [ ] **FR-011**: Provide configuration interface
- [ ] **FR-012**: Support automatic reconnection on connection loss
- [ ] **FR-022**: Handle server IP configuration with CLI override and persistent update capability

### 4.3 Security Requirements
- [ ] **FR-013**: Implement end-to-end encryption for tunnel traffic
- [ ] **FR-014**: Support secure authentication to prevent unauthorized server access
- [ ] **FR-015**: Implement mutual TLS (mTLS) authentication between agent and server
- [ ] **FR-016**: Use private Certificate Authority (CA) for certificate management
- [ ] **FR-017**: Single agent must have unique client certificate signed by private CA
- [ ] **FR-018**: Server must validate client certificate against private CA
- [ ] **FR-019**: Implement recommended security measures for public server deployment
- [ ] **FR-020**: Minimal logging for debugging purposes only (startup, connections, endpoint routing)
- [ ] **FR-021**: Support all protocols required for general web browsing
- [ ] **FR-023**: Test and analyze detection by endpoint security tools (CrowdStrike, Carbon Black, etc.)
- [ ] **FR-024**: Implement mitigations to avoid triggering EDR/security monitoring alerts

### 4.4 Testing Requirements
- [ ] **FR-025**: Develop integration tests for HTTP tunneling functionality
- [ ] **FR-026**: Develop integration tests for HTTPS CONNECT tunneling
- [ ] **FR-027**: Implement end-to-end test scenarios with real servers
- [ ] **FR-028**: Create automated test suite for CI/CD pipeline
- [ ] **FR-029**: Test connection recovery and error handling scenarios
- [ ] **FR-030**: Validate certificate validation and mTLS authentication flows

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
- [ ] **NFR-001**: Support multiple concurrent connections for single personal agent (minimum 10-20 parallel requests)
- [ ] **NFR-002**: Optimize for general web browsing performance with parallel resource loading
- [ ] **NFR-003**: Support all protocols required for web browsing (HTTP, HTTPS, WebSocket, etc.)
- [ ] **NFR-004**: Server startup time under 30 seconds for on-demand usage

### 5.3 Security Requirements
- [ ] **NFR-005**: All tunnel traffic encrypted with TLS 1.3 or higher
- [ ] **NFR-006**: Mutual TLS (mTLS) authentication with client certificates
- [ ] **NFR-007**: Private Certificate Authority (CA) for certificate management
- [ ] **NFR-008**: Certificate-based identity verification for single personal agent
- [ ] **NFR-009**: Minimal logging for debugging purposes (startup, connections, endpoint routing - no sensitive data)
- [ ] **NFR-010**: Standard security measures recommended for public deployment
- [ ] **NFR-011**: Server accessible only via mTLS authenticated agent connections

### 5.4 Scalability Requirements
- [ ] **NFR-012**: Support for on-demand server startup and shutdown
- [ ] **NFR-013**: Pay-as-you-use resource optimization
- [ ] **NFR-014**: Fully containerized deployment ensuring complete cloud provider portability
- [ ] **NFR-015**: Certificate management for single personal agent deployment
- [ ] **NFR-016**: Handle dynamic IP address changes for on-demand server deployments
- [ ] **NFR-017**: Connection pooling and efficient resource management for concurrent requests
- [ ] **NFR-018**: Server IP configuration with CLI override and persistent storage capabilities

---

## 6. User Experience Requirements

### 6.1 Setup & Configuration
- [ ] **UX-001**: CLI-based installation and setup using Docker
- [ ] **UX-002**: Configuration through command-line parameters and config files (including server IP with CLI override)
- [ ] **UX-003**: Terminal-based services (no GUI required)
- [ ] **UX-004**: Developer-level technical expertise assumed
- [ ] **UX-005**: Simple start/stop commands for server and agent

### 6.2 Monitoring & Feedback
- [ ] **UX-006**: Certificate-based setup with generated client certificates
- [ ] **UX-007**: Simple certificate installation process for agents
- [ ] **UX-008**: Terminal-based status indicators
- [ ] **UX-009**: Minimal logging output for debugging (startup, connections, endpoint routing)
- [ ] **UX-010**: CLI-based health checks and diagnostics
- [ ] **UX-011**: No GUI components required

---

## 7. Implementation Phases

### 7.1 Phase 1: Core Infrastructure (Weeks 1-4)
- Implement basic tunnel server in Go
- Implement basic tunnel agent in Go
- Establish secure communication protocol
- Create Docker containers for both components
- **Phase 1.5: Security & Testing (Before Phase 2)**
  - Test and evaluate detection by security tools (CrowdStrike, Carbon Black, etc.)
  - Implement mitigations to prevent triggering EDR/security monitoring tools
  - Develop comprehensive integration tests for end-to-end functionality
  - Validate HTTP and HTTPS tunneling with real-world scenarios

### 7.2 Phase 2: Cloud Deployment (Weeks 5-6)
- Deploy server to chosen cloud provider
- Implement scaling and monitoring
- Set up CI/CD pipeline for deployments

### 7.3 Phase 3: Enhanced Features (Weeks 7-8)
- Implement mTLS authentication and certificate management
- Set up private Certificate Authority (CA) infrastructure
- Implement certificate generation and distribution system
- Add advanced configuration options
- Add monitoring and logging capabilities

### 7.4 Phase 4: Testing & Documentation (Weeks 9-10)
- Comprehensive testing (unit, integration, performance)
- Complete user documentation
- Security audit and vulnerability assessment

---

## 8. Technical Specifications

### 8.1 Communication Protocol
**TBD** - Secure protocol supporting all web browsing requirements (HTTP, HTTPS, WebSocket, etc.)

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