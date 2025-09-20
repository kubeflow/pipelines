package util

import (
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/golang/glog"
)

// GetTLSConfig returns TLS config set with system CA certs as well as custom CA stored at input caCertPath.
func GetTLSConfig(caCertPath string) *tls.Config {
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		glog.Fatalf("Failed to load system cert pool: %v", err)
	}
	if caCertPath != "" {
		caCert, err := os.ReadFile(caCertPath)
		if err != nil {
			glog.Fatalf("Failed to read CA cert from %s", caCertPath)
		}
		if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
			glog.Fatalf("Failed to append CA cert from %s", caCertPath)
		}
	}
	return &tls.Config{
		RootCAs: caCertPool,
	}
}
