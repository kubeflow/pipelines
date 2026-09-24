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
	"database/sql/driver"
	"fmt"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/kubeflow/pipelines/backend/src/common/dbcreds"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The switch reaches the process as a string from a ConfigMap, so it can be
// empty or an unexpanded reference when the key is absent. Neither may be fatal.
func TestParseEnabled(t *testing.T) {
	tests := []struct {
		raw  string
		want bool
	}{
		{raw: "", want: false},
		{raw: "$(DB_CREDENTIAL_PROVIDER_ENABLED)", want: false},
		{raw: "  ", want: false},
		{raw: "nonsense", want: false},
		{raw: "true", want: true},
		{raw: " true ", want: true},
		{raw: "false", want: false},
	}

	for _, test := range tests {
		t.Run(test.raw, func(t *testing.T) {
			assert.Equal(t, test.want, dbcreds.ParseEnabled(test.raw))
		})
	}
}

func TestSettingsValidate(t *testing.T) {
	assert.NoError(t, dbcreds.Settings{}.Validate(), "an installation that has not opted in is always valid")
	assert.NoError(t, dbcreds.Settings{Enabled: true, ProviderName: dbcreds.StaticProviderName}.Validate())

	err := dbcreds.Settings{Enabled: true}.Validate()
	require.Error(t, err, "enabling a provider without naming one is a configuration error")
	assert.Contains(t, err.Error(), dbcreds.StaticProviderName, "the error must list what is available")
}

func TestSettingsDescribe(t *testing.T) {
	off := dbcreds.Settings{}.Describe(dbcreds.DriverMySQL)
	assert.Contains(t, off, "credentials=configured password")
	assert.Contains(t, off, "TLS=disabled")

	on := dbcreds.Settings{Enabled: true, ProviderName: "aws-iam", CABundlePath: "/etc/db-tls/ca.pem"}.Describe(dbcreds.DriverMySQL)
	assert.Contains(t, on, `credentials="aws-iam" provider`)
	assert.Contains(t, on, "verified against /etc/db-tls/ca.pem")
}

// recordingProvider captures every Target a connector was built for, which is
// how these tests observe the sequence without a database.
type recordingProvider struct {
	targets []dbcreds.Target
}

func (*recordingProvider) Name() string { return "recording" }

func (r *recordingProvider) Connector(_ context.Context, t dbcreds.Target) (driver.Connector, error) {
	r.targets = append(r.targets, t)
	switch t.Driver {
	case dbcreds.DriverMySQL:
		config, err := dbcreds.MySQLConfig(t)
		if err != nil {
			return nil, err
		}
		return dbcreds.MySQLConnector(config)
	default:
		config, err := dbcreds.PostgreSQLConfig(t)
		if err != nil {
			return nil, err
		}
		return dbcreds.PostgreSQLConnector(config), nil
	}
}

func TestSettingsWarnings(t *testing.T) {
	// A CA bundle with the switch off is inert, and silence would hide that.
	assert.NotEmpty(t, dbcreds.Settings{CABundlePath: "/etc/db-tls/ca.pem"}.IgnoredTLSWarning())
	assert.Empty(t, dbcreds.Settings{Enabled: true, CABundlePath: "/etc/db-tls/ca.pem"}.IgnoredTLSWarning())

	// A provider named with the switch off, which falls back to the password
	// the operator was moving away from. The name is quoted into the message,
	// so that the log says which provider was meant to be in force.
	assert.Contains(t, dbcreds.Settings{ProviderName: "aws-iam"}.IgnoredProviderWarning(), `"aws-iam"`)
	assert.Empty(t, dbcreds.Settings{Enabled: true, ProviderName: "aws-iam"}.IgnoredProviderWarning())
	assert.Empty(t, dbcreds.Settings{}.IgnoredProviderWarning(),
		"an installation that has not opted in must stay silent")

	// A password a provider will not use. It is an argument rather than a
	// field, so the same settings answer differently for different passwords.
	provider := dbcreds.Settings{Enabled: true, ProviderName: "aws-iam"}
	assert.NotEmpty(t, provider.IgnoredPasswordWarning("hunter2"))
	assert.Empty(t, provider.IgnoredPasswordWarning(""), "there is no password to remove")

	static := dbcreds.Settings{Enabled: true, ProviderName: dbcreds.StaticProviderName}
	assert.Empty(t, static.IgnoredPasswordWarning("hunter2"), "the static provider does use the password")

	assert.Empty(t, dbcreds.Settings{}.IgnoredPasswordWarning("hunter2"),
		"the pre-provider path uses the password, so there is nothing to warn about")
}

func TestSettingsTLSOptions(t *testing.T) {
	assert.Nil(t, dbcreds.Settings{}.TLSOptions(), "no CA bundle must leave the connection unencrypted")

	options := dbcreds.Settings{CABundlePath: "/etc/db-tls/ca.pem"}.TLSOptions()
	require.NotNil(t, options)
	assert.Equal(t, "/etc/db-tls/ca.pem", options.CABundlePath)
}

func TestSettingsNewProvider(t *testing.T) {
	provider, err := dbcreds.Settings{
		Enabled:      true,
		ProviderName: dbcreds.StaticProviderName,
	}.NewProvider("hunter2")
	require.NoError(t, err)
	assert.Equal(t, dbcreds.StaticProviderName, provider.Name())

	_, err = dbcreds.Settings{
		Enabled:          true,
		ProviderName:     dbcreds.StaticProviderName,
		ProviderSettings: "not json",
	}.NewProvider("hunter2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a JSON object")
}

func TestEnsureDatabaseCreates(t *testing.T) {
	warning, err := dbcreds.EnsureDatabase("mlpipeline", time.Second,
		func() error { return nil },
		func(err error) error { return err },
		func() error { return fmt.Errorf("probe must not run") })
	require.NoError(t, err)
	assert.Empty(t, warning, "creating the database is the strong outcome and needs no warning")
}

func TestEnsureDatabaseAcceptsExistingWhenRefused(t *testing.T) {
	warning, err := dbcreds.EnsureDatabase("mlpipeline", time.Second,
		func() error { return &pgconn.PgError{Code: "42501", Message: "permission denied to create database"} },
		func(err error) error { return err },
		func() error { return nil })
	require.NoError(t, err)
	assert.Contains(t, warning, "was not created")
	assert.Contains(t, warning, "already exists and is reachable")
}

func TestEnsureDatabaseFailsWhenUnreachable(t *testing.T) {
	_, err := dbcreds.EnsureDatabase("mlpipeline", time.Second,
		func() error { return &pgconn.PgError{Code: "42501", Message: "permission denied to create database"} },
		func(err error) error { return err },
		func() error { return fmt.Errorf(`database "mlpipeline" does not exist`) })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not create database")
	assert.Contains(t, err.Error(), "could not reach it")
}

// Reachability only answers a refusal for lack of privilege. Any other failure
// must not be excused by the database happening to exist.
func TestEnsureDatabaseDoesNotMaskOtherFailures(t *testing.T) {
	probed := false
	_, err := dbcreds.EnsureDatabase("mlpipeline", 200*time.Millisecond,
		func() error { return fmt.Errorf("ERROR: syntax error at or near \"DATABASE\"") },
		func(err error) error { return err },
		func() error { probed = true; return nil })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "syntax error")
	assert.False(t, probed, "a reachable database must not excuse an unrelated failure")
}

// tlsRequiringFactory stands in for a provider whose credential is unsafe to
// send over an unverified connection.
type tlsRequiringFactory struct{ fakeFactory }

func (tlsRequiringFactory) RequiresTLS() bool { return true }

// A provider that requires a verified connection is rejected from configuration
// alone, before anything reaches the cloud. An operator who has configured
// neither the CA bundle nor the provider's own settings is told about the CA
// bundle immediately, rather than fixing one problem and meeting the other.
func TestSettingsValidateHonoursProviderTLSRequirement(t *testing.T) {
	restoreRegistry(t)
	dbcreds.RegisterFactory(tlsRequiringFactory{fakeFactory{name: "needs-tls"}})
	dbcreds.RegisterFactory(fakeFactory{name: "no-tls-needed"})

	err := dbcreds.Settings{Enabled: true, ProviderName: "needs-tls"}.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires a verified connection")
	assert.Contains(t, err.Error(), "DB_TLS_CA_PATH", "the error must name the setting to add")

	assert.NoError(t, dbcreds.Settings{Enabled: true, ProviderName: "needs-tls", CABundlePath: "/etc/db-tls/ca.pem"}.Validate())

	// A provider that does not need TLS answers false, which it must do
	// explicitly -- RequiresTLS is part of the interface precisely so that the
	// answer is never inferred from silence. The requirement stays the
	// provider's own rather than a rule in shared code.
	assert.NoError(t, dbcreds.Settings{Enabled: true, ProviderName: "no-tls-needed"}.Validate())

	// An unregistered name is reported by provider construction, not here.
	assert.NoError(t, dbcreds.Settings{Enabled: true, ProviderName: "unknown"}.Validate())
}

// A privilege refusal is matched by code, not by message. MySQL says "access
// denied" for both 1044 (this user may not touch this database) and 1045 (this
// user did not authenticate), and only the first is answered by the database
// turning out to exist. Treating a rejected password or an invalid token as a
// privilege problem would start the server on a credential the database refused
// and tell the operator to grant permissions they already have.
func TestEnsureDatabaseDoesNotExcuseAuthenticationFailures(t *testing.T) {
	probed := false
	_, err := dbcreds.EnsureDatabase("mlpipeline", 200*time.Millisecond,
		func() error {
			return &mysql.MySQLError{Number: 1045, Message: "Access denied for user 'kfp'@'10.0.0.1' (using password: YES)"}
		},
		func(err error) error { return err },
		func() error { probed = true; return nil })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Access denied")
	assert.NotContains(t, err.Error(), "grant the user permission",
		"an authentication failure must not be reported as a privilege problem")
	assert.False(t, probed, "a rejected credential must not be excused by a reachable database")
}

// The privilege refusal itself still works, on both engines.
func TestEnsureDatabaseAcceptsPrivilegeRefusalOnBothEngines(t *testing.T) {
	for name, refusal := range map[string]error{
		"mysql":      &mysql.MySQLError{Number: 1044, Message: "Access denied for user 'kfp'@'%' to database 'mlpipeline'"},
		"postgresql": &pgconn.PgError{Code: "42501", Message: "permission denied to create database"},
	} {
		t.Run(name, func(t *testing.T) {
			warning, err := dbcreds.EnsureDatabase("mlpipeline", time.Second,
				func() error { return refusal },
				func(err error) error { return err },
				func() error { return nil })
			require.NoError(t, err)
			assert.Contains(t, warning, "already exists and is reachable")
		})
	}
}

// Bootstrap is the sequence both binaries used to carry a copy of. These pin
// the parts that differed between those copies, which is what made them worth
// sharing: which database the bootstrap connection names, and where
// clientFoundRows is applied.
func TestBootstrapNamesTheRightBootstrapDatabase(t *testing.T) {
	tests := []struct {
		driver string
		want   string
	}{
		{driver: dbcreds.DriverMySQL, want: ""},
		{driver: dbcreds.DriverPostgreSQL, want: "postgres"},
	}

	for _, test := range tests {
		t.Run(test.driver, func(t *testing.T) {
			recorder := &recordingProvider{}
			target := dbcreds.Target{
				Driver: test.driver,
				Host:   "db.example.com",
				Port:   "5432",
				User:   "kfp",
				Params: map[string]string{"sslmode": "disable"},
			}

			_, _, err := dbcreds.Bootstrap(context.Background(), recorder, target, dbcreds.BootstrapOptions{
				DBName:          "mlpipeline",
				QuoteIdentifier: func(s string) string { return s },
				Tolerate:        func(error) error { return nil },
				Timeout:         time.Second,
			})
			require.NoError(t, err)

			require.NotEmpty(t, recorder.targets)
			assert.Equal(t, test.want, recorder.targets[0].DBName,
				"the bootstrap connection must not assume the database exists")
			last := recorder.targets[len(recorder.targets)-1]
			assert.Equal(t, "mlpipeline", last.DBName, "the application connection names the database")
		})
	}
}

// clientFoundRows is MySQL-only, and Bootstrap must not write it into the
// caller's map: the same Target is used for the bootstrap connection first.
func TestBootstrapAppliesClientFoundRowsWithoutMutatingTheCaller(t *testing.T) {
	recorder := &recordingProvider{}
	params := map[string]string{"group_concat_max_len": "1024"}
	target := dbcreds.Target{
		Driver: dbcreds.DriverMySQL,
		Host:   "db.example.com",
		Port:   "3306",
		User:   "kfp",
		Params: params,
	}

	_, _, err := dbcreds.Bootstrap(context.Background(), recorder, target, dbcreds.BootstrapOptions{
		DBName:          "cachedb",
		QuoteIdentifier: func(s string) string { return s },
		Tolerate:        func(error) error { return nil },
		Timeout:         time.Second,
	})
	require.NoError(t, err)

	assert.NotContains(t, params, "clientFoundRows", "the caller's map must not be mutated")
	last := recorder.targets[len(recorder.targets)-1]
	assert.Equal(t, "true", last.Params["clientFoundRows"])
	assert.Equal(t, "1024", last.Params["group_concat_max_len"], "the caller's parameters survive")
}

func TestBootstrapRejectsAnUnsupportedDriver(t *testing.T) {
	_, _, err := dbcreds.Bootstrap(context.Background(), &recordingProvider{}, dbcreds.Target{Driver: "oracle"},
		dbcreds.BootstrapOptions{DBName: "x", QuoteIdentifier: func(s string) string { return s }, Tolerate: func(error) error { return nil }, Timeout: time.Second})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support driver")
}
