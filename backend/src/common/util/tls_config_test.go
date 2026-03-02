// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetTLSConfigEmptyPath(t *testing.T) {
	config, err := GetTLSConfig("")
	assert.Nil(t, config)
	assert.NoError(t, err)
}

func TestGetTLSConfigNonexistentPath(t *testing.T) {
	nonexistentPath := filepath.Join(t.TempDir(), "nonexistent-cert.pem")
	config, err := GetTLSConfig(nonexistentPath)
	assert.Nil(t, config)
	assert.Error(t, err)
}

func TestGetTLSConfigValidCert(t *testing.T) {
	// Generate a self-signed certificate
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"Test"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		IsCA:         true,
		KeyUsage:     x509.KeyUsageCertSign,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	assert.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	certFile := filepath.Join(t.TempDir(), "valid-cert.pem")
	err = os.WriteFile(certFile, certPEM, 0644)
	assert.NoError(t, err)

	config, err := GetTLSConfig(certFile)
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.NotNil(t, config.RootCAs)
}

func TestGetTLSConfigInvalidPEM(t *testing.T) {
	invalidPEMFile := filepath.Join(t.TempDir(), "invalid-cert.pem")
	err := os.WriteFile(invalidPEMFile, []byte("this is not valid PEM content"), 0644)
	assert.NoError(t, err)

	config, err := GetTLSConfig(invalidPEMFile)
	assert.Nil(t, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse certificate")
}
