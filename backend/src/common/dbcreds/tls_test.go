// Copyright 2026 The Kubeflow Authors
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

package dbcreds_test

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

	"github.com/kubeflow/pipelines/backend/src/common/dbcreds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeCABundle writes a self-signed certificate and returns its path.
func writeCABundle(t *testing.T) string {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "kfp-test-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "ca.pem")
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))
	return path
}

func TestTLSConfigDisabledWhenUnset(t *testing.T) {
	config, err := dbcreds.TLSConfig(nil, "mysql")
	require.NoError(t, err)
	assert.Nil(t, config, "a nil TLSOptions must leave the connection unencrypted, as it is today")
}

// The bundle replaces the system roots rather than joining them, so a server
// certificate signed by a publicly trusted CA that is not in the bundle is not
// accepted. Appending would make DB_TLS_CA_PATH mean "this CA or any public
// one", which is not what the setting reads as.
func TestTLSConfigPinsToTheBundleOnly(t *testing.T) {
	path := writeCABundle(t)
	config, err := dbcreds.TLSConfig(&dbcreds.TLSOptions{CABundlePath: path}, "db.example.com")
	require.NoError(t, err)
	require.NotNil(t, config.RootCAs)

	bundle, err := os.ReadFile(path)
	require.NoError(t, err)
	onlyTheBundle := x509.NewCertPool()
	require.True(t, onlyTheBundle.AppendCertsFromPEM(bundle))
	assert.True(t, config.RootCAs.Equal(onlyTheBundle),
		"the pool must contain exactly the configured bundle, not the system roots plus it")

	if system, err := x509.SystemCertPool(); err == nil {
		require.True(t, system.AppendCertsFromPEM(bundle))
		assert.False(t, config.RootCAs.Equal(system),
			"appending to the system roots would make DB_TLS_CA_PATH mean \"this CA or any public one\"")
	}
}

func TestTLSConfigLoadsCABundle(t *testing.T) {
	config, err := dbcreds.TLSConfig(&dbcreds.TLSOptions{CABundlePath: writeCABundle(t)}, "aurora.example.com")
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "aurora.example.com", config.ServerName)
	assert.NotNil(t, config.RootCAs)
}

func TestTLSConfigWithoutCABundleUsesSystemRoots(t *testing.T) {
	config, err := dbcreds.TLSConfig(&dbcreds.TLSOptions{}, "aurora.example.com")
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "aurora.example.com", config.ServerName)
	assert.Nil(t, config.RootCAs, "an empty CA path must fall back to the system roots")
}

func TestTLSConfigRejectsBadCABundle(t *testing.T) {
	unreadable := filepath.Join(t.TempDir(), "missing.pem")

	notPEM := filepath.Join(t.TempDir(), "garbage.pem")
	require.NoError(t, os.WriteFile(notPEM, []byte("this is not a certificate"), 0o600))

	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{
			name:    "missing file",
			path:    unreadable,
			wantErr: "failed to read DB_TLS_CA_PATH",
		},
		{
			name:    "not a certificate",
			path:    notPEM,
			wantErr: "did not contain valid PEM certificates",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := dbcreds.TLSConfig(&dbcreds.TLSOptions{CABundlePath: test.path}, "mysql")
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.wantErr)
			assert.Contains(t, err.Error(), test.path, "the error must name the path an operator has to fix")
		})
	}
}
