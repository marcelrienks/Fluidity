# Certificate Management

## Quick Start

Generate certificates locally:
```bash
./scripts/manage-certs.sh
```

Generate and save to AWS Secrets Manager:
```bash
./scripts/manage-certs.sh --save-to-secrets
```

PowerShell equivalent:
```powershell
.\scripts\manage-certs.ps1 -Command generate-and-save -SecretsManager
```

## Scripts

- `scripts/manage-certs.sh` - Unified Bash script (generate/save/both)
- `scripts/manage-certs.ps1` - Unified PowerShell script (generate/save/both)
- Legacy scripts still available: `generate-certs.*`, `save-certs-to-secrets.*`

## Options

### Bash
```
--save-to-secrets      Save to AWS Secrets Manager
--secret-name NAME     Secret name (default: fluidity/certificates)
--certs-dir DIR        Certificate directory (default: ./certs)
--help                 Show help
```

### PowerShell
```
-Command               "generate" | "save" | "generate-and-save"
-SecretsManager        Enable Secrets Manager saving
-SecretName NAME       Secret name (default: fluidity/certificates)
-CertsDir DIR          Certificate directory (default: .\certs)
-Days NUM              Validity days (default: 730)
```

## Configuration

After generating/saving certificates, enable in Fluidity config:

**configs/server.yaml** and **configs/agent.yaml**:
```yaml
use_secrets_manager: true
secrets_manager_name: "fluidity/certificates"
```

## Certificate Output

**Local certificates (./certs/):**
- `ca.crt`, `ca.key` - CA certificate and key
- `server.crt`, `server.key` - Server certificate and key
- `client.crt`, `client.key` - Client certificate and key

**AWS Secrets Manager:**
- `cert_pem` - Base64-encoded server certificate
- `key_pem` - Base64-encoded server key
- `ca_pem` - Base64-encoded CA certificate

## Certificates Details

- **CA**: Self-signed, 730 days, subject `/C=US/ST=CA/L=SF/O=Fluidity/OU=Dev/CN=Fluidity-CA`
- **Server**: mTLS serverAuth, SANs: `fluidity-server`, `localhost`, `127.0.0.1`, `::1`, `host.docker.internal`
- **Client**: mTLS clientAuth
- **Development only** - For production, use certificates from trusted CA

## Prerequisites

**Required:**
- OpenSSL
  - Linux: `apt-get install openssl`
  - macOS: `brew install openssl`
  - Windows: https://slproweb.com/products/Win32OpenSSL.html

**Optional (for AWS):**
- AWS CLI: https://aws.amazon.com/cli/
- AWS credentials configured (environment variables, `~/.aws/credentials`, or IAM role)

## AWS IAM Permissions

**For running Fluidity (read-only):**
```json
{
  "Effect": "Allow",
  "Action": ["secretsmanager:GetSecretValue"],
  "Resource": "arn:aws:secretsmanager:*:*:secret:fluidity/certificates*"
}
```

**For certificate management (create/update):**
```json
{
  "Effect": "Allow",
  "Action": ["secretsmanager:CreateSecret", "secretsmanager:UpdateSecret", "secretsmanager:DescribeSecret"],
  "Resource": "arn:aws:secretsmanager:*:*:secret:fluidity/certificates*"
}
```

## Common Tasks

**Local development:**
```bash
./scripts/manage-certs.sh
# Creates ./certs/ with 6 certificate files
```

**AWS deployment (Fargate/Lambda):**
```bash
./scripts/manage-certs.sh --save-to-secrets --secret-name "fluidity/certificates"
# Generates certs and saves to AWS Secrets Manager
```

**Custom secret name:**
```bash
./scripts/manage-certs.sh --save-to-secrets --secret-name "my-org/my-app/certs"
```

**Certificate rotation:**
```bash
./scripts/manage-certs.sh --save-to-secrets
# Generates new certs and updates AWS secret
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| OpenSSL not found | Install from https://slproweb.com/products/Win32OpenSSL.html |
| AWS CLI not found | Install from https://aws.amazon.com/cli/ |
| Unable to locate credentials | Run `aws configure` or set `AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY` |
| Secret already exists | Scripts auto-update (no error) |
| Certs already exist | Delete with `rm ./certs/*` or `Remove-Item .\certs\*` and re-run |

## Security Notes

- ⚠️ **Development only** - Self-signed certificates
- Store `*.key` files securely, never commit to version control
- Private keys are base64-encoded in Secrets Manager (not encrypted at rest by script)
- Use AWS KMS encryption for Secrets Manager in production
- Regenerate certificates regularly for production deployments

## Examples

**Generate in custom directory:**
```bash
./scripts/manage-certs.sh --certs-dir /opt/fluidity/certs
```

**1-year valid certificates:**
```powershell
.\scripts\manage-certs.ps1 -Days 365 -SecretsManager
```

**Generate only (existing secrets):**
```bash
./scripts/manage-certs.sh                    # Bash - just generate
.\scripts\manage-certs.ps1 -Command save -SecretsManager  # PS - just save
```

## Verification

After running `./scripts/manage-certs.sh`:

```bash
# Verify local certificates
ls -la ./certs/

# Verify AWS Secrets Manager
aws secretsmanager get-secret-value --secret-id fluidity/certificates
```

## Related Documentation

- Architecture: `docs/architecture.md`
- Deployment guide: `docs/deployment.md`
- Docker deployment: `docs/docker.md`
- AWS Fargate: `docs/fargate.md`
- Infrastructure as Code: `docs/infrastructure.md`
