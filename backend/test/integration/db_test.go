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

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	_ "github.com/jackc/pgx/v5/stdlib"
	cm "github.com/kubeflow/pipelines/backend/src/apiserver/client_manager"
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
	if *localTest {
		viper.Set("DBConfig.MySQLConfig.Host", "localhost")
	}
	duration, _ := time.ParseDuration("1m")
	db := cm.InitDBClient(duration)
	assert.NotNil(t, db)
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
	// The default port-forwarding IP address that test uses is different compared to production
	viper.Set("DBConfig.PostgreSQLConfig.Host", "127.0.0.3")
	viper.Set("DBConfig.PostgreSQLConfig.User", "user")
	viper.Set("DBConfig.PostgreSQLConfig.Password", "password")
	duration, _ := time.ParseDuration("1m")
	db := cm.InitDBClient(duration)
	assert.NotNil(t, db)
}

func TestDB(t *testing.T) {
	suite.Run(t, new(DBTestSuite))
}
