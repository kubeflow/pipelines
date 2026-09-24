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

// A CA bundle says the connection must be encrypted. An sslmode in the extra
// params is merged after it, so without this check it would silently win and
// the connection would go out in plaintext -- carrying, for a provider whose
// credential is a token, a bearer credential in the startup packet. MySQL
// refuses the same conflict; so must this.
func TestPostgreSQLConfigRejectsTLSDowngrade(t *testing.T) {
	for _, mode := range []string{"disable", "allow", "prefer"} {
		t.Run(mode, func(t *testing.T) {
			target := postgresTarget()
			target.TLS = &dbcreds.TLSOptions{CABundlePath: writeCABundle(t)}
			target.Params = map[string]string{"sslmode": mode}

			_, err := dbcreds.PostgreSQLConfig(target)
			require.Error(t, err, "a CA bundle must not be silently overridden")
			assert.Contains(t, err.Error(), "permits an unencrypted connection")
		})
	}
}

// The modes that always encrypt stay available, so an operator who wants
// verify-ca or require alongside a bundle is not blocked.
//
// "Encrypted" is not the whole assertion. An unverified TLS config is still
// non-nil, so each mode is checked for a certificate it will actually verify
// against: pgx leaves InsecureSkipVerify set for require and verify-ca and
// supplies a VerifyPeerCertificate that checks the chain without the hostname,
// while verify-full verifies in the usual way.
func TestPostgreSQLConfigAllowsEncryptingModesWithABundle(t *testing.T) {
	for _, mode := range []string{"require", "verify-ca", "verify-full"} {
		t.Run(mode, func(t *testing.T) {
			target := postgresTarget()
			target.TLS = &dbcreds.TLSOptions{CABundlePath: writeCABundle(t)}
			target.Params = map[string]string{"sslmode": mode}

			config, err := dbcreds.PostgreSQLConfig(target)
			require.NoError(t, err)
			require.NotNil(t, config.TLSConfig, "%s must encrypt", mode)

			assert.NotNil(t, config.TLSConfig.RootCAs, "%s must verify against the configured bundle", mode)
			assert.True(t, !config.TLSConfig.InsecureSkipVerify || config.TLSConfig.VerifyPeerCertificate != nil,
				"%s must not skip verification without a replacement check", mode)
		})
	}
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
			target.Params = map[string]string{"sslmode": "require", test.param: test.value}

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
