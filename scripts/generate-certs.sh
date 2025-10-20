#!/bin/bash

# Certificate generation script for development
# This script generates a self-signed CA and client/server certificates for mTLS

set -e

CERTS_DIR="./certs"
DAYS=730  # 2 years

# Create certs directory if it doesn't exist
mkdir -p "$CERTS_DIR"

echo "Generating development certificates..."

# Generate CA private key
echo "1. Generating CA private key..."
openssl genrsa -out "$CERTS_DIR/ca.key" 4096

# Generate CA certificate
echo "2. Generating CA certificate..."
openssl req -new -x509 -days $DAYS -key "$CERTS_DIR/ca.key" -out "$CERTS_DIR/ca.crt" -subj "/C=US/ST=CA/L=SF/O=Fluidity/OU=Dev/CN=Fluidity-CA"

# Generate server private key
echo "3. Generating server private key..."
openssl genrsa -out "$CERTS_DIR/server.key" 4096

# Generate server certificate signing request
echo "4. Generating server CSR..."
openssl req -new -key "$CERTS_DIR/server.key" -out "$CERTS_DIR/server.csr" -subj "/C=US/ST=CA/L=SF/O=Fluidity/OU=Server/CN=fluidity-server"

# Create server certificate extensions file
cat > "$CERTS_DIR/server.ext" << EOF
[v3_req]
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = fluidity-server
DNS.2 = localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF

# Generate server certificate signed by CA
echo "5. Generating server certificate..."
openssl x509 -req -in "$CERTS_DIR/server.csr" -CA "$CERTS_DIR/ca.crt" -CAkey "$CERTS_DIR/ca.key" -CAcreateserial -out "$CERTS_DIR/server.crt" -days $DAYS -extensions v3_req -extfile "$CERTS_DIR/server.ext"

# Generate client private key
echo "6. Generating client private key..."
openssl genrsa -out "$CERTS_DIR/client.key" 4096

# Generate client certificate signing request
echo "7. Generating client CSR..."
openssl req -new -key "$CERTS_DIR/client.key" -out "$CERTS_DIR/client.csr" -subj "/C=US/ST=CA/L=SF/O=Fluidity/OU=Client/CN=fluidity-client"

# Generate client certificate signed by CA
echo "8. Generating client certificate..."
openssl x509 -req -in "$CERTS_DIR/client.csr" -CA "$CERTS_DIR/ca.crt" -CAkey "$CERTS_DIR/ca.key" -CAcreateserial -out "$CERTS_DIR/client.crt" -days $DAYS

# Clean up temporary files
rm "$CERTS_DIR/server.csr" "$CERTS_DIR/server.ext" "$CERTS_DIR/client.csr"

# Set appropriate permissions
chmod 600 "$CERTS_DIR"/*.key
chmod 644 "$CERTS_DIR"/*.crt

echo "Certificates generated successfully!"
echo ""
echo "Files created:"
echo "  $CERTS_DIR/ca.crt       - CA certificate"
echo "  $CERTS_DIR/ca.key       - CA private key"
echo "  $CERTS_DIR/server.crt   - Server certificate"
echo "  $CERTS_DIR/server.key   - Server private key"
echo "  $CERTS_DIR/client.crt   - Client certificate"
echo "  $CERTS_DIR/client.key   - Client private key"
echo ""
echo "Note: These certificates are for development only!"
echo "For production, use properly signed certificates from a trusted CA."