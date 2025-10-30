package secretsmanager

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/sirupsen/logrus"
)

// CertificateSecret represents the structure of the certificate secret in AWS Secrets Manager
type CertificateSecret struct {
	CertPEM string `json:"cert_pem"` // Base64-encoded certificate PEM
	KeyPEM  string `json:"key_pem"`  // Base64-encoded key PEM
	CaPEM   string `json:"ca_pem"`   // Base64-encoded CA certificate PEM
}

// LoadTLSConfigFromSecrets loads TLS configuration from AWS Secrets Manager
// Returns client or server TLS config depending on the isServer parameter
func LoadTLSConfigFromSecrets(ctx context.Context, secretName string, isServer bool) (*tls.Config, error) {
	logrus.WithFields(logrus.Fields{
		"secret_name": secretName,
		"is_server":   isServer,
	}).Info("Loading TLS configuration from AWS Secrets Manager")

	// Create AWS SDK config
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Secrets Manager client
	client := secretsmanager.NewFromConfig(cfg)

	// Retrieve secret
	result, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve secret from AWS Secrets Manager: %w", err)
	}

	// Parse the secret JSON
	var secret CertificateSecret
	if err := json.Unmarshal([]byte(*result.SecretString), &secret); err != nil {
		return nil, fmt.Errorf("failed to parse certificate secret JSON: %w", err)
	}

	// Decode base64-encoded certificate data
	certPEM, err := base64.StdEncoding.DecodeString(secret.CertPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to decode certificate PEM: %w", err)
	}

	keyPEM, err := base64.StdEncoding.DecodeString(secret.KeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key PEM: %w", err)
	}

	caPEM, err := base64.StdEncoding.DecodeString(secret.CaPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to decode CA PEM: %w", err)
	}

	// Load certificate and key
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate key pair: %w", err)
	}

	// Log certificate details
	if len(cert.Certificate) > 0 {
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		if err == nil {
			logrus.WithFields(logrus.Fields{
				"subject":    x509Cert.Subject.CommonName,
				"issuer":     x509Cert.Issuer.CommonName,
				"not_before": x509Cert.NotBefore,
				"not_after":  x509Cert.NotAfter,
			}).Info("Loaded certificate from Secrets Manager")
		}
	}

	// Load CA certificate
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	// Create TLS config
	if isServer {
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    caCertPool,
			MinVersion:   tls.VersionTLS13,
		}

		logrus.WithFields(logrus.Fields{
			"num_certificates": len(tlsConfig.Certificates),
			"client_auth":      "RequireAndVerifyClientCert",
			"has_client_cas":   tlsConfig.ClientCAs != nil,
			"min_version":      "TLS 1.3",
		}).Info("Created server TLS config from Secrets Manager")

		return tlsConfig, nil
	}

	// Client TLS config
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS13,
		ServerName:   "", // Will be set dynamically
	}

	logrus.WithFields(logrus.Fields{
		"num_certificates": len(tlsConfig.Certificates),
		"has_root_cas":     tlsConfig.RootCAs != nil,
		"min_version":      "TLS 1.3",
	}).Info("Created client TLS config from Secrets Manager")

	return tlsConfig, nil
}

// SaveCertificatesToSecrets saves TLS certificates to AWS Secrets Manager
// This is a utility function to initially store certificates in the secret
func SaveCertificatesToSecrets(ctx context.Context, secretName string, certFile, keyFile, caFile string) error {
	logrus.WithFields(logrus.Fields{
		"secret_name": secretName,
		"cert_file":   certFile,
		"key_file":    keyFile,
		"ca_file":     caFile,
	}).Info("Saving certificates to AWS Secrets Manager")

	// Read certificate files
	certData, err := os.ReadFile(certFile)
	if err != nil {
		return fmt.Errorf("failed to read certificate file: %w", err)
	}

	keyData, err := os.ReadFile(keyFile)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	caData, err := os.ReadFile(caFile)
	if err != nil {
		return fmt.Errorf("failed to read CA file: %w", err)
	}

	// Create certificate secret
	secret := CertificateSecret{
		CertPEM: base64.StdEncoding.EncodeToString(certData),
		KeyPEM:  base64.StdEncoding.EncodeToString(keyData),
		CaPEM:   base64.StdEncoding.EncodeToString(caData),
	}

	secretJSON, err := json.Marshal(secret)
	if err != nil {
		return fmt.Errorf("failed to marshal certificate secret: %w", err)
	}

	// Create AWS SDK config
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Secrets Manager client
	client := secretsmanager.NewFromConfig(cfg)

	// Try to create the secret first
	_, err = client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         aws.String(secretName),
		SecretString: aws.String(string(secretJSON)),
	})

	// If secret already exists, update it
	if err != nil {
		_, updateErr := client.UpdateSecret(ctx, &secretsmanager.UpdateSecretInput{
			SecretId:     aws.String(secretName),
			SecretString: aws.String(string(secretJSON)),
		})
		if updateErr != nil {
			return fmt.Errorf("failed to update secret: %w (original create error: %w)", updateErr, err)
		}
		logrus.WithFields(logrus.Fields{
			"secret_name": secretName,
		}).Info("Updated existing certificate secret in AWS Secrets Manager")
		return nil
	}

	logrus.WithFields(logrus.Fields{
		"secret_name": secretName,
	}).Info("Created new certificate secret in AWS Secrets Manager")

	return nil
}

// LoadTLSConfigFromSecretsOrFallback attempts to load from Secrets Manager, falls back to local files
func LoadTLSConfigFromSecretsOrFallback(ctx context.Context, secretName string, certFile, keyFile, caFile string, isServer bool, fallbackFn func() (*tls.Config, error)) (*tls.Config, error) {
	// Try to load from Secrets Manager
	tlsConfig, err := LoadTLSConfigFromSecrets(ctx, secretName, isServer)
	if err == nil {
		logrus.Info("Successfully loaded TLS configuration from AWS Secrets Manager")
		return tlsConfig, nil
	}

	logrus.WithFields(logrus.Fields{
		"error": err.Error(),
	}).Warn("Failed to load TLS configuration from AWS Secrets Manager, falling back to local files")

	// Fall back to local file-based loading
	return fallbackFn()
}
