// Copyright 2023 The Kubeflow Authors
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

package integration

import (
	"testing"
	"time"

	cm "github.com/kubeflow/pipelines/backend/src/apiserver/client_manager"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/dbcreds"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type DBTestSuite struct {
	suite.Suite
}

// Skip if it's not integration test running.
func (s *DBTestSuite) SetupTest() {
	if !*runIntegrationTests {
		s.T().SkipNow()
		return
	}
}

// Test MySQL initializes correctly
func (s *DBTestSuite) TestInitDBClient_MySQL() {
	if *runPostgreSQLTests {
		s.T().SkipNow()
		return
	}
	t := s.T()
	viper.Set("DBDriverName", "mysql")
	viper.Set("DBConfig.MySQLConfig.DBName", "mlpipeline")
	// The default port-forwarding IP address that test uses is different compared to production
	viper.Set("DBConfig.MySQLConfig.Host", "localhost")
	duration, _ := time.ParseDuration("1m")
	db, dialect, _ := cm.InitDBClient(duration)
	assert.NotNil(t, db)
	assert.Equal(t, "mysql", dialect.Name())
}

// Test PostgreSQL initializes correctly
func (s *DBTestSuite) TestInitDBClient_PostgreSQL() {
	if !*runPostgreSQLTests {
		s.T().SkipNow()
		return
	}
	t := s.T()
	viper.Set("DBDriverName", "pgx")
	viper.Set("DBConfig.PostgreSQLConfig.DBName", "mlpipeline")
	// Using localhost to match MySQL behavior and port-forward default.
	viper.Set("DBConfig.PostgreSQLConfig.Host", "localhost")
	viper.Set("DBConfig.PostgreSQLConfig.User", "user")
	viper.Set("DBConfig.PostgreSQLConfig.Password", "password")
	// sslmode must be set explicitly (secure-by-default); the local PostgreSQL
	// used by integration tests does not use TLS, so opt into "disable".
	viper.Set("DBConfig.PostgreSQLConfig.ExtraParams", map[string]string{"sslmode": "disable"})
	duration, _ := time.ParseDuration("1m")
	db, dialect, _ := cm.InitDBClient(duration)
	assert.NotNil(t, db)
	assert.Equal(t, "pgx", dialect.Name())
}

// Test the credential provider path initializes correctly against a real
// database. The static provider carries the configured password, so this
// exercises everything the opt-in path does -- registry lookup, provider
// construction, connector building, and the schema migration that follows --
// without needing a cloud identity.
func (s *DBTestSuite) TestInitDBClient_MySQL_CredentialProvider() {
	if *runPostgreSQLTests {
		s.T().SkipNow()
		return
	}
	t := s.T()
	viper.Set("DBDriverName", "mysql")
	viper.Set("DBConfig.MySQLConfig.DBName", "mlpipeline")
	viper.Set("DBConfig.MySQLConfig.Host", "localhost")
	// Set explicitly rather than inheriting whatever a sibling test left in
	// viper: the suite only happens to run them in an order that works.
	viper.Set("DBConfig.MySQLConfig.User", "root")
	viper.Set("DBConfig.MySQLConfig.Password", "")
	viper.Set(common.DBCredentialProviderEnabled, true)
	viper.Set(common.DBCredentialProvider, dbcreds.StaticProviderName)
	defer func() {
		viper.Set(common.DBCredentialProviderEnabled, false)
		viper.Set(common.DBCredentialProvider, "")
	}()

	duration, _ := time.ParseDuration("1m")
	db, dialect, _ := cm.InitDBClient(duration)
	assert.NotNil(t, db)
	assert.Equal(t, "mysql", dialect.Name())
}

// Test the credential provider path for PostgreSQL.
func (s *DBTestSuite) TestInitDBClient_PostgreSQL_CredentialProvider() {
	if !*runPostgreSQLTests {
		s.T().SkipNow()
		return
	}
	t := s.T()
	viper.Set("DBDriverName", "pgx")
	viper.Set("DBConfig.PostgreSQLConfig.DBName", "mlpipeline")
	// localhost, matching the pre-provider case above: the integration
	// workflow port-forwards without --address, which binds loopback only.
	viper.Set("DBConfig.PostgreSQLConfig.Host", "localhost")
	viper.Set("DBConfig.PostgreSQLConfig.User", "user")
	viper.Set("DBConfig.PostgreSQLConfig.Password", "password")
	// sslmode must be set explicitly (secure-by-default); the local PostgreSQL
	// used by integration tests does not use TLS, so opt into "disable".
	viper.Set("DBConfig.PostgreSQLConfig.ExtraParams", map[string]string{"sslmode": "disable"})
	viper.Set(common.DBCredentialProviderEnabled, true)
	viper.Set(common.DBCredentialProvider, dbcreds.StaticProviderName)
	defer func() {
		viper.Set(common.DBCredentialProviderEnabled, false)
		viper.Set(common.DBCredentialProvider, "")
	}()

	duration, _ := time.ParseDuration("1m")
	db, dialect, _ := cm.InitDBClient(duration)
	assert.NotNil(t, db)
	assert.Equal(t, "pgx", dialect.Name())
}

func TestDB(t *testing.T) {
	suite.Run(t, new(DBTestSuite))
}
