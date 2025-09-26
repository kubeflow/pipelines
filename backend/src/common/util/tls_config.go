package util

import (
	"crypto/tls"
	"crypto/x509"
	"os"
)

// GetTLSConfig returns TLS config set with system CA certs as well as custom CA stored at input caCertPath if provided.
func GetTLSConfig(caCertPath string) (*tls.Config, error) {
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	if caCertPath != "" {
		caCert, err := os.ReadFile(caCertPath)
		if err != nil {
			return nil, err
		}
		if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
			return nil, err
		}
	}
	return &tls.Config{
		RootCAs: caCertPool,
	}, nil
}
