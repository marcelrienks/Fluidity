# Certificate Management

## Quick Start

**Local development:**
```bash
./scripts/manage-certs.sh              # Linux/macOS
.\scripts\manage-certs.ps1             # Windows
```

**AWS deployment:**
```bash
./scripts/manage-certs.sh --save-to-secrets
.\scripts\manage-certs.ps1 -Command generate-and-save -SecretsManager
```

## Script Options

### Bash (`manage-certs.sh`)
```
--save-to-secrets      Save to AWS Secrets Manager
--secret-name NAME     Secret name (default: fluidity/certificates)
--certs-dir DIR        Certificate directory (default: ./certs)
```

### PowerShell (`manage-certs.ps1`)
```
-Command               "generate" | "save" | "generate-and-save"
-SecretsManager        Enable Secrets Manager saving
-SecretName NAME       Secret name (default: fluidity/certificates)
-CertsDir DIR          Certificate directory (default: .\certs)
-Days NUM              Validity days (default: 730)
```

## Output

**Local files (./certs/):**
- `ca.crt`, `ca.key` - CA certificate and key
- `server.crt`, `server.key` - Server certificate and key
- `client.crt`, `client.key` - Client certificate and key

**AWS Secrets Manager:**
- `cert_pem` - Base64 server certificate
- `key_pem` - Base64 server key
- `ca_pem` - Base64 CA certificate

## Configuration

Enable Secrets Manager in config files:

```yaml
# configs/server.yaml and configs/agent.yaml
use_secrets_manager: true
secrets_manager_name: "fluidity/certificates"
```

## Prerequisites

**Required:**
- OpenSSL ([Windows download](https://slproweb.com/products/Win32OpenSSL.html))

**For AWS:**
- AWS CLI
- Configured AWS credentials

## IAM Permissions

**For running Fluidity:**
```json
{
  "Effect": "Allow",
  "Action": ["secretsmanager:GetSecretValue"],
  "Resource": "arn:aws:secretsmanager:*:*:secret:fluidity/certificates*"
}
```

**For certificate management:**
```json
{
  "Effect": "Allow",
  "Action": ["secretsmanager:CreateSecret", "secretsmanager:UpdateSecret"],
  "Resource": "arn:aws:secretsmanager:*:*:secret:fluidity/certificates*"
}
```

## Common Tasks

**Certificate rotation:**
```bash
./scripts/manage-certs.sh --save-to-secrets
```

**Custom secret name:**
```bash
./scripts/manage-certs.sh --save-to-secrets --secret-name "my-org/fluidity/certs"
```

**Custom directory:**
```bash
./scripts/manage-certs.sh --certs-dir /opt/fluidity/certs
```

**Verify certificates:**
```bash
ls -la ./certs/
aws secretsmanager get-secret-value --secret-id fluidity/certificates
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| OpenSSL not found | Install from link above |
| AWS CLI not found | Install from https://aws.amazon.com/cli/ |
| Unable to locate credentials | Run `aws configure` |
| Certs already exist | Delete with `rm ./certs/*` and re-run |

## Security

- ⚠️ Self-signed certificates for **development only**
- Never commit `*.key` files to version control
- Use AWS KMS encryption for Secrets Manager in production
- Rotate certificates regularly (default: 2 years)
- For production, use certificates from trusted CA

## Related Documentation

- [Deployment Guide](deployment.md) - Using certificates in deployments
- [Docker Guide](docker.md) - Baking certificates into images
- [Infrastructure Guide](infrastructure.md) - AWS Secrets Manager integration
