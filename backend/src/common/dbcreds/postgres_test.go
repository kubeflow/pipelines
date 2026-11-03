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
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/common/dbcreds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func postgresTarget() dbcreds.Target {
	return dbcreds.Target{
		Driver: dbcreds.DriverPostgreSQL,
		Host:   "postgresql",
		Port:   "5432",
		User:   "root",
		DBName: "mlpipeline",
		// sslmode has no default; each target states its choice.
		Params: map[string]string{"sslmode": "disable"},
	}
}

func TestPostgreSQLConfig(t *testing.T) {
	config, err := dbcreds.PostgreSQLConfig(postgresTarget())
	require.NoError(t, err)

	assert.Equal(t, "postgresql", config.Host)
	assert.Equal(t, uint16(5432), config.Port)
	assert.Equal(t, "root", config.User)
	assert.Equal(t, "mlpipeline", config.Database)

	// The credential is the provider's job, never part of the configuration
	// this function returns.
	assert.Empty(t, config.Password)
}

// TestPostgreSQLConfigRequiresAnExplicitSSLMode pins that neither a CA bundle
// nor an sslmode is a refusal rather than a silent plaintext connection, which
// is the same rule the pre-provider connection enforces.
func TestPostgreSQLConfigRequiresAnExplicitSSLMode(t *testing.T) {
	target := postgresTarget()
	target.Params = nil

	_, err := dbcreds.PostgreSQLConfig(target)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sslmode must be explicitly set")
}

// An operator who chooses plaintext explicitly still gets it.
func TestPostgreSQLConfigHonoursExplicitDisable(t *testing.T) {
	target := postgresTarget()
	target.Params = map[string]string{"sslmode": "disable"}

	config, err := dbcreds.PostgreSQLConfig(target)
	require.NoError(t, err)
	assert.Nil(t, config.TLSConfig, "sslmode=disable must leave the connection unencrypted")
}

func TestPostgreSQLConfigTLSModes(t *testing.T) {
	tests := []struct {
		name    string
		tls     *dbcreds.TLSOptions
		wantTLS bool
	}{
		{name: "explicit disable stays unencrypted", tls: nil, wantTLS: false},
		{name: "options verify the server", tls: &dbcreds.TLSOptions{}, wantTLS: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			target := postgresTarget()
			target.TLS = test.tls
			if test.tls != nil {
				// A CA bundle is itself the choice, so no sslmode is set.
				target.Params = nil
			}

			config, err := dbcreds.PostgreSQLConfig(target)
			require.NoError(t, err)

			if !test.wantTLS {
				assert.Nil(t, config.TLSConfig)
				return
			}
			require.NotNil(t, config.TLSConfig)
			assert.False(t, config.TLSConfig.InsecureSkipVerify, "the server must always be verified")
		})
	}
}

func TestPostgreSQLConfigParams(t *testing.T) {
	target := postgresTarget()
	target.Params = map[string]string{
		"connect_timeout":  "7",
		"application_name": "kfp",
		"sslmode":          "require",
	}

	config, err := dbcreds.PostgreSQLConfig(target)
	require.NoError(t, err)

	// Recognized settings land on their typed field, unrecognized ones become
	// server runtime parameters.
	assert.Equal(t, 7*time.Second, config.ConnectTimeout)
	assert.Equal(t, "kfp", config.RuntimeParams["application_name"])

	// Params override the defaults this package sets, including sslmode -- but
	// only in a direction that keeps the connection encrypted; see
	// TestPostgreSQLConfigRejectsTLSDowngrade.
	require.NotNil(t, config.TLSConfig, "sslmode=require must enable TLS")
}

// A CA bundle says the connection must be verified against it, so every other
// sslmode is refused -- not only the ones that permit plaintext. require and
// verify-ca do encrypt, but pgx checks the certificate chain without the
// hostname under both, and for RDS one global bundle signs every instance in
// every account: a chain-only check accepts any RDS endpoint an attacker
// controls, which for a token presented as a password means handing it over.
func TestPostgreSQLConfigRequiresVerifyFullWithABundle(t *testing.T) {
	for _, mode := range []string{"disable", "allow", "prefer", "require", "verify-ca"} {
		t.Run(mode, func(t *testing.T) {
			target := postgresTarget()
			target.TLS = &dbcreds.TLSOptions{CABundlePath: writeCABundle(t)}
			target.Params = map[string]string{"sslmode": mode}

			_, err := dbcreds.PostgreSQLConfig(target)
			require.Error(t, err, "a CA bundle must not be silently weakened")
			assert.Contains(t, err.Error(), "does not verify that the certificate belongs to")
		})
	}
}

// verify-full is what a bundle selects, and it must keep working.
func TestPostgreSQLConfigAcceptsVerifyFullWithABundle(t *testing.T) {
	target := postgresTarget()
	target.TLS = &dbcreds.TLSOptions{CABundlePath: writeCABundle(t)}
	target.Params = map[string]string{"sslmode": "verify-full"}

	config, err := dbcreds.PostgreSQLConfig(target)
	require.NoError(t, err)
	require.NotNil(t, config.TLSConfig)
	assert.False(t, config.TLSConfig.InsecureSkipVerify, "verify-full must check the hostname")
	assert.Equal(t, target.Host, config.TLSConfig.ServerName)
}

// sslmode is not the only way to disarm verification. sslrootcert reaches the
// same driver settings map, so an empty one drops the certificate pool and
// leaves pgx with InsecureSkipVerify and no VerifyPeerCertificate -- encrypted,
// wholly unverified, and still logged as verified against the bundle. "system"
// is the quieter variant, swapping the pinned CA for the public web PKI. The
// whole ssl* namespace is refused rather than these two keys.
func TestPostgreSQLConfigRejectsTLSParamsAlongsideABundle(t *testing.T) {
	tests := []struct {
		name  string
		param string
		value string
	}{
		{name: "empty sslrootcert drops verification", param: "sslrootcert", value: ""},
		{name: "system sslrootcert unpins the CA", param: "sslrootcert", value: "system"},
		{name: "a replacement bundle", param: "sslrootcert", value: "/tmp/other-ca.pem"},
		{name: "client certificate", param: "sslcert", value: "/tmp/client.pem"},
		{name: "sni", param: "sslsni", value: "0"},
		{name: "negotiation", param: "sslnegotiation", value: "direct"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			target := postgresTarget()
			target.TLS = &dbcreds.TLSOptions{CABundlePath: writeCABundle(t)}
			target.Params = map[string]string{"sslmode": "verify-full", test.param: test.value}

			_, err := dbcreds.PostgreSQLConfig(target)
			require.Error(t, err, "the bundle must not be silently reconfigured")
			assert.Contains(t, err.Error(), test.param)
		})
	}
}

// Without a bundle the operator owns transport security outright, so the same
// parameters are theirs to set.
func TestPostgreSQLConfigAllowsTLSParamsWithoutABundle(t *testing.T) {
	target := postgresTarget()
	target.Params = map[string]string{"sslmode": "verify-full", "sslrootcert": "system"}

	_, err := dbcreds.PostgreSQLConfig(target)
	require.NoError(t, err)
}

func TestPostgreSQLConfigRejectsInvalidTargets(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*dbcreds.Target)
		wantErr string
	}{
		{
			name:    "wrong driver",
			mutate:  func(target *dbcreds.Target) { target.Driver = dbcreds.DriverMySQL },
			wantErr: `PostgreSQLConfig called with driver "mysql"`,
		},
		{
			name:    "empty host",
			mutate:  func(target *dbcreds.Target) { target.Host = "" },
			wantErr: "database host is empty",
		},
		{
			name: "unparseable parameter",
			mutate: func(target *dbcreds.Target) {
				target.Params = map[string]string{"sslmode": "disable", "port": "not-a-port"}
			},
			wantErr: "invalid PostgreSQL connection parameters",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			target := postgresTarget()
			test.mutate(&target)

			_, err := dbcreds.PostgreSQLConfig(target)
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.wantErr)
		})
	}
}

func TestPostgreSQLConnectorIsUsable(t *testing.T) {
	config, err := dbcreds.PostgreSQLConfig(postgresTarget())
	require.NoError(t, err)
	assert.NotNil(t, dbcreds.PostgreSQLConnector(config))
}
