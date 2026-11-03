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
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/kubeflow/pipelines/backend/src/common/dbcreds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mysqlTarget() dbcreds.Target {
	return dbcreds.Target{
		Driver: dbcreds.DriverMySQL,
		Host:   "mysql",
		Port:   "3306",
		User:   "root",
		DBName: "mlpipeline",
		Params: map[string]string{"group_concat_max_len": "1024"},
	}
}

func TestMySQLConfig(t *testing.T) {
	config, err := dbcreds.MySQLConfig(mysqlTarget())
	require.NoError(t, err)

	assert.Equal(t, "root", config.User)
	assert.Equal(t, "tcp", config.Net)
	assert.Equal(t, "mysql:3306", config.Addr)
	assert.Equal(t, "mlpipeline", config.DBName)
	assert.True(t, config.AllowNativePasswords)

	// The credential is the provider's job, never part of the configuration
	// this function returns.
	assert.Empty(t, config.Passwd)

	// Recognized parameters must land on their typed fields. Left in Params
	// they would be issued as SET statements at connect time.
	assert.True(t, config.ParseTime)
	assert.Equal(t, time.Local, config.Loc)
	assert.NotContains(t, config.Params, "parseTime")
	assert.NotContains(t, config.Params, "loc")
	assert.NotContains(t, config.Params, "charset")

	// Unrecognized parameters stay in Params, which is how they reach the
	// server as session variables.
	assert.Equal(t, "1024", config.Params["group_concat_max_len"])
}

func TestMySQLConfigParamsOverrideDefaults(t *testing.T) {
	target := mysqlTarget()
	target.Params = map[string]string{"parseTime": "false", "readTimeout": "30s"}

	config, err := dbcreds.MySQLConfig(target)
	require.NoError(t, err)

	assert.False(t, config.ParseTime, "Target.Params must override the built-in defaults")
	assert.Equal(t, 30*time.Second, config.ReadTimeout)
}

func TestMySQLConfigJoinsIPv6Host(t *testing.T) {
	target := mysqlTarget()
	target.Host = "::1"

	config, err := dbcreds.MySQLConfig(target)
	require.NoError(t, err)
	assert.Equal(t, "[::1]:3306", config.Addr)
}

func TestMySQLConfigRejectsInvalidTargets(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*dbcreds.Target)
		wantErr string
	}{
		{
			name:    "wrong driver",
			mutate:  func(target *dbcreds.Target) { target.Driver = dbcreds.DriverPostgreSQL },
			wantErr: `MySQLConfig called with driver "pgx"`,
		},
		{
			name:    "empty host",
			mutate:  func(target *dbcreds.Target) { target.Host = "" },
			wantErr: "database host is empty",
		},
		{
			name:    "unparseable parameter",
			mutate:  func(target *dbcreds.Target) { target.Params = map[string]string{"readTimeout": "not-a-duration"} },
			wantErr: "invalid MySQL connection parameters",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			target := mysqlTarget()
			test.mutate(&target)

			_, err := dbcreds.MySQLConfig(target)
			require.Error(t, err)
			assert.Contains(t, err.Error(), test.wantErr)
		})
	}
}

// TestMySQLConfigMatchesLegacyDSN pins the behavior an existing installation
// gets today. The pre-provider code built the configuration below, formatted it
// as a DSN, and handed that to sql.Open, which parses it back. Any divergence
// here is a change to the default connection path.
func TestMySQLConfigMatchesLegacyDSN(t *testing.T) {
	target := mysqlTarget()

	legacyParams := map[string]string{
		"charset":              "utf8",
		"parseTime":            "True",
		"loc":                  "Local",
		"group_concat_max_len": "1024",
	}
	legacy := &mysql.Config{
		User:                 target.User,
		Passwd:               "",
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%s", target.Host, target.Port),
		Params:               legacyParams,
		DBName:               target.DBName,
		AllowNativePasswords: true,
	}
	want, err := mysql.ParseDSN(legacy.FormatDSN())
	require.NoError(t, err)

	got, err := dbcreds.MySQLConfig(target)
	require.NoError(t, err)

	// MaxAllowedPacket is the field most likely to drift: the legacy literal
	// leaves it at zero, which is not the driver's NewConfig default.
	assert.Equal(t, want.MaxAllowedPacket, got.MaxAllowedPacket)
	assert.Equal(t, want.FormatDSN(), got.FormatDSN())
}

func TestMySQLConnectorAppliesOptions(t *testing.T) {
	config, err := dbcreds.MySQLConfig(mysqlTarget())
	require.NoError(t, err)

	connector, err := dbcreds.MySQLConnector(config, mysql.BeforeConnect(
		func(ctx context.Context, c *mysql.Config) error { return nil }))
	require.NoError(t, err)
	assert.NotNil(t, connector)
}

func TestMySQLConnectorRejectsFailedOption(t *testing.T) {
	config, err := dbcreds.MySQLConfig(mysqlTarget())
	require.NoError(t, err)

	failing := func(*mysql.Config) error { return fmt.Errorf("boom") }
	_, err = dbcreds.MySQLConnector(config, failing)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apply MySQL driver options")
}

// TestMySQLConnectorRunsHookPerConnection pins the mechanism the credential
// providers depend on: the driver invokes the hook for every new connection,
// with its own copy of the configuration, before it dials. The dial then fails
// because nothing is listening, which is fine -- the hook has already run.
func TestMySQLConnectorRunsHookPerConnection(t *testing.T) {
	target := mysqlTarget()
	// Port 1 is reserved and refuses connections quickly.
	target.Host, target.Port = "127.0.0.1", "1"

	config, err := dbcreds.MySQLConfig(target)
	require.NoError(t, err)

	var calls int
	var sawAddr, sawUser string
	connector, err := dbcreds.MySQLConnector(config, mysql.BeforeConnect(
		func(ctx context.Context, c *mysql.Config) error {
			calls++
			sawAddr, sawUser = c.Addr, c.User
			c.Passwd = fmt.Sprintf("token-%d", calls)
			return nil
		}))
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		conn, connectErr := connector.Connect(context.Background())
		require.Error(t, connectErr, "nothing is listening on the target port")
		assert.Nil(t, conn)
	}

	assert.Equal(t, 3, calls, "the credential must be refreshed for every new connection")
	assert.Equal(t, "127.0.0.1:1", sawAddr)
	assert.Equal(t, "root", sawUser)

	// The hook mutates a copy, so the password it set never accumulates on the
	// configuration the connector was built from.
	assert.Empty(t, config.Passwd)
}

// TestMySQLConnectorPropagatesHookFailure ensures a credential failure surfaces
// as a connection error rather than an attempt to connect without one.
func TestMySQLConnectorPropagatesHookFailure(t *testing.T) {
	config, err := dbcreds.MySQLConfig(mysqlTarget())
	require.NoError(t, err)

	connector, err := dbcreds.MySQLConnector(config, mysql.BeforeConnect(
		func(context.Context, *mysql.Config) error { return fmt.Errorf("token expired") }))
	require.NoError(t, err)

	_, err = connector.Connect(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token expired")
}

// TestMySQLConfigAppliesTLS pins where transport security is resolved. It is
// applied by the shared builder rather than by each provider, so a provider
// cannot omit it and connect unencrypted while the operator believes TLS is on.
func TestMySQLConfigAppliesTLS(t *testing.T) {
	target := mysqlTarget()
	target.TLS = &dbcreds.TLSOptions{CABundlePath: writeCABundle(t)}

	config, err := dbcreds.MySQLConfig(target)
	require.NoError(t, err)
	require.NotNil(t, config.TLS, "MySQLConfig must resolve Target.TLS itself")
	assert.Equal(t, target.Host, config.TLS.ServerName)
	assert.NotNil(t, config.TLS.RootCAs)
	assert.False(t, config.TLS.InsecureSkipVerify, "the server must always be verified")
}

func TestMySQLConfigWithoutTLSStaysPlaintext(t *testing.T) {
	config, err := dbcreds.MySQLConfig(mysqlTarget())
	require.NoError(t, err)
	assert.Nil(t, config.TLS, "no TLS options must leave the connection unencrypted, as it is today")
}

// TestMySQLConfigRejectsTLSDowngrade covers a credential-disclosure path. The
// driver's tls=preferred also sets AllowFallbackToPlaintext, which lets it drop
// TLS silently when the server does not advertise it. Assigning the verified
// configuration without clearing that permission would leave a provider's TLS
// requirement defeatable by an operator-supplied connection parameter, with the
// credential still sent -- in cleartext, for providers using that plugin.
func TestMySQLConfigRejectsTLSDowngrade(t *testing.T) {
	for _, value := range []string{"preferred", "skip-verify", "true", "false"} {
		t.Run(value, func(t *testing.T) {
			target := mysqlTarget()
			target.TLS = &dbcreds.TLSOptions{CABundlePath: writeCABundle(t)}
			target.Params["tls"] = value

			_, err := dbcreds.MySQLConfig(target)
			require.Error(t, err, "a connection parameter must not be able to weaken configured TLS")
			assert.Contains(t, err.Error(), "DB_TLS_CA_PATH")
		})
	}
}

// Without a CA bundle the package does not manage TLS, so connection parameters
// keep working as they did before.
func TestMySQLConfigLeavesParamsAloneWithoutTLS(t *testing.T) {
	target := mysqlTarget()
	target.Params["tls"] = "preferred"

	config, err := dbcreds.MySQLConfig(target)
	require.NoError(t, err)
	assert.True(t, config.AllowFallbackToPlaintext)
}
