# Deployment Guide

Last updated: October 27, 2025

This guide covers all supported deployment options for the Fluidity Tunnel Server and Agent: local development, Docker, and AWS Fargate (manual and CloudFormation). Use this as your single reference to get up and running in each environment.

---

## Prerequisites

- Go 1.21+ (for local builds)
- Docker Desktop (for container runs)
- Make (for platform-specific Makefiles)
- OpenSSL (for certificate generation)
- Node.js 18+ (for WebSocket tests)
- AWS account + AWS CLI v2 (for cloud deployment)

Certs: Generate client/server certs with scripts in `scripts/` (see Quick Start below).

---

## Option A — Local (Binaries)

Best for: development and quick iteration.

1) Generate certificates

```powershell
# Windows
./scripts/generate-certs.ps1
```

```bash
# macOS/Linux
chmod +x scripts/generate-certs.sh
./scripts/generate-certs.sh
```

2) Run server and agent in separate terminals

```powershell
# Windows
make -f Makefile.win run-server-local
make -f Makefile.win run-agent-local
```

```bash
# macOS
make -f Makefile.macos run-server-local
make -f Makefile.macos run-agent-local

# Linux
make -f Makefile.linux run-server-local
make -f Makefile.linux run-agent-local
```

3) Configure your browser to use HTTP/HTTPS proxy at `127.0.0.1:8080`

4) Test with curl

```powershell
curl.exe -x http://127.0.0.1:8080 https://example.com -I
```

---

## Option B — Docker (Local Containers)

Best for: verifying container builds locally.

1) Build images

```powershell
# Windows
make -f Makefile.win docker-build-server
make -f Makefile.win docker-build-agent
```

```bash
# macOS/Linux
make -f Makefile.macos docker-build-server
make -f Makefile.macos docker-build-agent
# or
make -f Makefile.linux docker-build-server
make -f Makefile.linux docker-build-agent
```

2) Run containers (Windows PowerShell)

```powershell
# Server
docker run --rm `
  -v ${PWD}\certs:/root/certs:ro `
  -v ${PWD}\configs\server.local.yaml:/root/config/server.yaml:ro `
  -p 8443:8443 `
  fluidity-server `
  ./fluidity-server --config ./config/server.yaml

# Agent
docker run --rm `
  -v ${PWD}\certs:/root/certs:ro `
  -v ${PWD}\configs\agent.local.yaml:/root/config/agent.yaml:ro `
  -p 8080:8080 `
  fluidity-agent `
  ./fluidity-agent --config ./config/agent.yaml
```

3) Run containers (macOS/Linux)

```bash
# Server
docker run --rm \
  -v "$(pwd)/certs:/root/certs:ro" \
  -v "$(pwd)/configs/server.local.yaml:/root/config/server.yaml:ro" \
  -p 8443:8443 \
  fluidity-server \
  ./fluidity-server --config ./config/server.yaml

# Agent
docker run --rm \
  -v "$(pwd)/certs:/root/certs:ro" \
  -v "$(pwd)/configs/agent.local.yaml:/root/config/agent.yaml:ro" \
  -p 8080:8080 \
  fluidity-agent \
  ./fluidity-agent --config ./config/agent.yaml
```

---

## Option C — AWS Fargate (ECS) — Manual

Best for: on-demand personal cloud use with minimal management.

Summary of steps (full details in `docs/fargate.md`):

1) Build, tag, and push the server image to ECR
2) Create CloudWatch Log Group, Security Group, ECS Cluster
3) Register a Fargate Task Definition (CPU=256, Memory=512, port 8443, awslogs)
4) Create an ECS Service with `desiredCount=0`, public subnets, `assignPublicIp=ENABLED`
5) Start on demand with `desiredCount=1`, wait ~60 seconds for cold start
6) Fetch the task public IP and run the Agent with `--server-ip <IP>`
7) Stop when done with `desiredCount=0`

Handy scripts (PowerShell and Bash) to start/stop and print the public IP are included in `docs/fargate.md`.

---

## Option D — AWS Fargate (ECS) — CloudFormation

Best for: repeatable, parameterized provisioning.

Use the template at `deployments/cloudformation/fargate.yaml`. It creates:
- ECS Cluster (Fargate), IAM execution role
- CloudWatch Log Group
- Security Group allowing inbound TCP on 8443 (or your port)
- ECS Task Definition + Service (desired count default 0)

1) Prepare `params.json` (example in `docs/fargate.md`):
- `ContainerImage` (ECR URI)
- `VpcId`, `PublicSubnets`
- Optional tuning: `DesiredCount`, `AllowedIngressCidr`, `Cpu`, `Memory`, `ContainerPort`

2) Deploy/Update

```powershell
$stackName = "fluidity-fargate"
aws cloudformation deploy `
  --template-file deployments/cloudformation/fargate.yaml `
  --stack-name $stackName `
  --parameter-overrides (Get-Content deployments/cloudformation/params.json | Out-String) `
  --capabilities CAPABILITY_NAMED_IAM
```

3) Start/Stop
- Update `DesiredCount` to 1 (start) or 0 (stop) via `aws cloudformation update-stack` (commands in `docs/fargate.md`)
- Or use `aws ecs update-service --desired-count ...`

4) Fetch Public IP
- Use the start script in `docs/fargate.md` to get the task’s public IP and pass it to the Agent

---

## Security Tips

- Restrict Security Group ingress on 8443 to your current public IP (use `/32` CIDR)
- Keep mTLS enabled end-to-end; protect private keys and CA material
- Set CloudWatch Logs retention (e.g., 7–30 days)
- Prefer AWS Secrets Manager/SSM Parameter Store if you later externalize certs/configs

---

## Troubleshooting

- Fargate task stuck in `PENDING`: check subnets are public, `assignPublicIp=ENABLED`, sufficient Fargate quota
- No logs: verify `awslogs` configuration and log group exists
- Connection refused: confirm SG allows port 8443 and container is listening
- TLS errors: confirm CA, server cert, key paths, and that Agent uses the correct `--server-ip`

---

## Costs (estimates)

- Fargate (0.25 vCPU, 0.5GB): ~$0.012/hour (≈ $0.50/month for 2h/day, ≈ $3/month for 8h/day)
- 24/7 Fargate: ≈ $9/month
- Additional minor charges: CloudWatch logs, ECR storage, data transfer

---

## References

- `docs/fargate.md` — Detailed Fargate and CloudFormation instructions
- `deployments/cloudformation/fargate.yaml` — CloudFormation template for ECS on Fargate
- `scripts/` — Certificate generation and test scripts
- `configs/` — Example configuration files
