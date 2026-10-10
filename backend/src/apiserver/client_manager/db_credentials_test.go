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

package clientmanager

import (
	"testing"

	mysqlStd "github.com/go-sql-driver/mysql"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	commonsql "github.com/kubeflow/pipelines/backend/src/apiserver/common/sql"
	"github.com/kubeflow/pipelines/backend/src/common/dbcreds"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setConfig sets viper keys for one test and restores them afterwards.
func setConfig(t *testing.T, values map[string]interface{}) {
	t.Helper()
	for key, value := range values {
		previous := viper.Get(key)
		t.Cleanup(func() { viper.Set(key, previous) })
		viper.Set(key, value)
	}
}

// The adapter is the only part that is binary-specific; everything it produces
// is exercised by the dbcreds settings tests.
func TestDBSettingsFromConfig(t *testing.T) {
	assert.False(t, dbSettings().Enabled, "an installation that has not opted in must keep the existing path")

	setConfig(t, map[string]interface{}{
		common.DBCredentialProviderEnabled:  true,
		common.DBCredentialProvider:         "aws-iam",
		common.DBTLSCAPath:                  "/etc/db-tls/ca.pem",
		common.DBCredentialProviderSettings: `{"region":"us-east-1"}`,
	})

	settings := dbSettings()
	assert.True(t, settings.Enabled)
	assert.Equal(t, "aws-iam", settings.ProviderName)
	assert.Equal(t, "/etc/db-tls/ca.pem", settings.CABundlePath)
	assert.Equal(t, `{"region":"us-east-1"}`, settings.ProviderSettings)
}

func TestMySQLTargetFromConfig(t *testing.T) {
	setConfig(t, map[string]interface{}{
		mysqlServiceHost:       "aurora.example.com",
		mysqlServicePort:       "3307",
		mysqlUser:              "kfp",
		mysqlGroupConcatMaxLen: "4194304",
	})

	target := mysqlTargetFromConfig(dbSettings())
	assert.Equal(t, dbcreds.DriverMySQL, target.Driver)
	assert.Equal(t, "aurora.example.com", target.Host)
	assert.Equal(t, "3307", target.Port)
	assert.Equal(t, "kfp", target.User)
	assert.Equal(t, "4194304", target.Params["group_concat_max_len"])
}

func TestMySQLTargetFromConfigExtraParamsWin(t *testing.T) {
	setConfig(t, map[string]interface{}{
		mysqlExtraParams: map[string]string{"group_concat_max_len": "512", "readTimeout": "30s"},
	})

	target := mysqlTargetFromConfig(dbSettings())
	assert.Equal(t, "512", target.Params["group_concat_max_len"],
		"ExtraParams must override the defaults, as it does today")
	assert.Equal(t, "30s", target.Params["readTimeout"])
}

// TestMySQLTargetMatchesLegacyConfig pins the connection an existing
// installation gets. The pre-provider code built the configuration below and
// handed its DSN to sql.Open; any divergence here changes the default path.
func TestMySQLTargetMatchesLegacyConfig(t *testing.T) {
	const dbName = "mlpipeline"

	target := mysqlTargetFromConfig(dbSettings())
	target.DBName = dbName
	target.Params["clientFoundRows"] = "true"

	got, err := dbcreds.MySQLConfig(target)
	require.NoError(t, err)

	legacy := commonsql.CreateMySQLConfig("root", "", "mysql", "3306", dbName, "1024", map[string]string{})
	legacy.ClientFoundRows = true
	want, err := mysqlStd.ParseDSN(legacy.FormatDSN())
	require.NoError(t, err)

	assert.Equal(t, want.FormatDSN(), got.FormatDSN())

	// The credential is applied to the configuration by the provider rather
	// than encoded into the DSN, so it cannot leak through a logged DSN and is
	// not mangled by DSN escaping.
	assert.Empty(t, got.Passwd)
}

func TestPostgresTargetFromConfig(t *testing.T) {
	setConfig(t, map[string]interface{}{
		postgresHost: "aurora-pg.example.com",
		postgresPort: 5433,
		postgresUser: "kfp",
	})

	target := postgresTargetFromConfig(dbSettings())
	assert.Equal(t, dbcreds.DriverPostgreSQL, target.Driver)
	assert.Equal(t, "aurora-pg.example.com", target.Host)
	assert.Equal(t, "5433", target.Port)
	assert.Equal(t, "kfp", target.User)
	assert.Nil(t, target.TLS, "no CA bundle must leave the connection unencrypted, as it is today")
}

// TestPostgresTargetMatchesLegacyDefaults pins the connection an existing
// PostgreSQL installation gets. The pre-provider path refuses to build a
// connection whose sslmode was never stated, and the provider path inherits
// both the requirement and the operator's answer to it.
func TestPostgresTargetMatchesLegacyDefaults(t *testing.T) {
	_, err := dbcreds.PostgreSQLConfig(postgresTargetFromConfig(dbSettings()))
	require.Error(t, err, "an unstated sslmode must be refused, not silently disabled")
	assert.Contains(t, err.Error(), "sslmode must be explicitly set")

	setConfig(t, map[string]interface{}{
		postgresExtraParams: map[string]string{"sslmode": "disable"},
	})

	config, err := dbcreds.PostgreSQLConfig(postgresTargetFromConfig(dbSettings()))
	require.NoError(t, err)
	assert.Nil(t, config.TLSConfig, "sslmode=disable is what the legacy connection string produces")
	assert.Empty(t, config.Password, "the credential is applied by the provider, not encoded into the connection string")
}
