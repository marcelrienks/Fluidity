# Fluidity

**Allowing http traffic to flow freely**

Provides a way for enabling HTTP traffic, tunnelling, and routing.
Or in layman's terms 'bypass corporate firewall blocking useful sites'

![Status](https://img.shields.io/badge/status-planning-blue)
![License](https://img.shields.io/badge/license-custom-lightgrey)

## Project Overview

Fluidity is currently in the **planning phase** and aims to create a robust HTTP tunnel solution consisting of two main components:

- **Tunnel Server**: A Go-based server application deployed to a cloud service provider
- **Tunnel Agent**: A Go-based client agent running locally

This architecture enables HTTP traffic to bypass restrictive corporate firewalls by establishing a secure tunnel between the local agent and the cloud-hosted server.

## Intended Architecture

```
   [Local Network]          [Internet]          [Cloud Provider]
┌───────────────────┐      ┌──────────┐        ┌──────────────────┐
│  Docker Desktop   │      │          │        │  Tunnel Server   │
│  ┌─────────────┐  │      │ Firewall │        │   (Go Binary)    │
│  │Tunnel Agent ├──┼──────┤ Bypass   ├────────┤   Containerized  │
│  │ (Go Binary) │  │      │          │        │                  │
│  │Containerized│  │      │          │        │                  │
│  └─────────────┘  │      └──────────┘        └──────────────────┘
└───────────────────┘
```

## Key Features (Planned)

- **Go-based Implementation**: Both server and agent written in Go for performance and cross-platform compatibility
- **Containerized Deployment**: Docker containers for easy deployment and management
- **Cloud-hosted Server**: Deployed to major cloud service providers for reliability and global access
- **Local Agent**: Runs within Docker Desktop for easy local setup and management
- **HTTP Traffic Tunneling**: Secure routing of HTTP requests through the tunnel
- **Firewall Bypass**: Designed to work around restrictive corporate network policies

## Technology Stack

- **Language**: Go
- **Containerization**: Docker
- **Deployment**: Cloud service provider (TBD)
- **Local Runtime**: Docker Desktop

## Current Status

This project is currently in the **planning and design phase**. Implementation has not yet begun.

### Roadmap

1. **Architecture Design** - Define detailed system architecture and communication protocols
2. **Server Development** - Implement the Go-based tunnel server
3. **Agent Development** - Implement the Go-based tunnel agent
4. **Containerization** - Create Docker images for both components
5. **Cloud Deployment** - Deploy server to chosen cloud provider
6. **Testing & Validation** - Comprehensive testing of tunnel functionality
7. **Documentation** - Complete user guides and deployment instructions

## Disclaimer

⚠️ **Important**: This tool is intended for legitimate use cases such as accessing necessary resources for work or personal use. Users are responsible for ensuring compliance with their organization's network policies and local laws. The developers are not responsible for any misuse of this software.
